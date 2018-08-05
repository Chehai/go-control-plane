package envoy

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync/atomic"

	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"

	"github.com/gogo/protobuf/proto"
	"github.com/gogo/protobuf/types"
	"google.golang.org/grpc"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
)

const (
	typePrefix   = "type.googleapis.com/envoy.api.v2."
	EndpointType = typePrefix + "ClusterLoadAssignment"
	ClusterType  = typePrefix + "Cluster"
	RouteType    = typePrefix + "RouteConfiguration"
	ListenerType = typePrefix + "Listener"
	AnyType      = ""
)

type grpcStream interface {
	Send(*v2.DiscoveryResponse) error
	Recv() (*v2.DiscoveryRequest, error)
}

type resource interface {
	proto.Message
	Equal(interface{}) bool
}

const grpcMaxConcurrentStreams = 1000000

type grpcService interface {
	v2.EndpointDiscoveryServiceServer
	v2.ClusterDiscoveryServiceServer
	v2.RouteDiscoveryServiceServer
	v2.ListenerDiscoveryServiceServer
	discovery.AggregatedDiscoveryServiceServer

	// Fetch is the universal fetch method.
	Fetch(context.Context, *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error)
}

func (r *Router) startGrpcServer(ctx context.Context) error {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(grpcMaxConcurrentStreams))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", r.port))
	if err != nil {
		return err
	}

	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, r)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, r)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, r)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, r)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, r)

	log.WithFields(log.Fields{"port": r.port}).Infof("Envoy Management Server listening on port %v", r.port)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Error(err)
		}
	}()

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}()

	return nil
}

func pushResponse(stream *pushStream, resp *v2.DiscoveryResponse) error {
	stream.Lock()
	defer stream.Unlock()
	if err := stream.Send(resp); err != nil {
		log.Error(err)
		return err
	}
	return nil
}

// golang's %s is greedy, so we have to put number first and then string
func makeNonce(pushID string, i int) string {
	return fmt.Sprintf("%d-%s", i, pushID)
}

func readNonce(nonce string) (string, int) {
	var pushID string
	var i int
	fmt.Sscanf(nonce, "%d-%s", &i, &pushID)
	return pushID, i
}

func (r *Router) makeVersionInfo() string {
	v := atomic.AddUint64(&r.VersionCounter, 1)
	return fmt.Sprintf("%d", v)
}

func makeResponse(res []resource, typeUrl string, versionInfo string, nonce string) (*v2.DiscoveryResponse, error) {
	resources := make([]types.Any, 0, len(res))
	for _, r := range res {
		data, err := proto.Marshal(r)
		if err != nil {
			return nil, err
		}
		resources = append(resources, types.Any{TypeUrl: typeUrl, Value: data})
	}
	return &v2.DiscoveryResponse{
		VersionInfo: versionInfo,
		Resources:   resources,
		TypeUrl:     typeUrl,
		Nonce:       nonce,
	}, nil
}

func pushResourcesToStream(s *pushStream, resources []resource, typeUrl string, versionInfo string, nonce string) error {
	resp, err := makeResponse(resources, typeUrl, versionInfo, nonce)
	if err != nil {
		log.Errorf("Pushing resources %s %s %s failed: %v", typeUrl, versionInfo, nonce, err)
		return err
	}
	go pushResponse(ps, resp)
	return nil
}

func (r *Router) bootstrapResources(s grpcStream, typeUrl string) error {
	ps := r.PushStreams.find(s)
	if ps == nil {
		return fmt.Errorf("Pushing bootstrap resources failed: cannot find push stream for %v", s)
	}
	ctx := context.TODO()
	var resources []resources
	var err error
	switch typeUrl {
	case EndpointType:
		resources, err = r.makeEndpointResources(ctx)
	case ClusterType:
		resources, err = r.makeClusterResources(ctx)
	case RouteType:
		resources, err = r.makeRouteResources(ctx)
	case ListenerType:
		resources, err = r.makeListenerResources(ctx)
	}
	if err != nil {
		log.Errorf("Bootstrapping resources %s failed: %v", typeUrl, err)
		return err
	}
	err = pushResourcesToStream(ps, resources, typeUrl, r.makeVersionInfo(), "0-0")
	if err != nil {
		log.Errorf("Bootstrapping resources %s failed: %v", typeUrl, err)
		return err
	}
	return nil
}

func (r *Router) pushResources(ctx context.Context, res []resource, typeUrl string) error {
	streams := r.PushStreams.get(typeUrl)
	if streams == nil {
		return fmt.Errorf("Cannot find streams for %s", typeUrl)
	}
	pushID := xid.New().String()
	defer r.PushCallbacks.delete(pushID)

	cbChs := make([]<-chan error, len(streams))
	for i, s := range streams {
		ch := make(chan error)
		r.PushCallbacks.create(pushID, i, ctx, ch)
		cbChs[i] = ch
		versionInfo := r.makeVersionInfo()
		nonce := makeNonce(pushID, i)
		err = pushResourcesToStream(s, resources, typeUrl, versionInfo, nonce)
		if err != nil {
			log.Errorf("Pushing resources %s failed: %v", typeUrl, err)
			return err
		}
	}
	cbCh := mergeChannels(cbChs...)
	select {
	case err := <-cbCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		return ctx.Err()
	}
	return nil
}

func (r *Router) handleGrpcStream(s grpcStream, typeUrl string) error {
	r.PushStreams.create(typeUrl, s)
	defer r.PushStreams.delete(typeUrl, s)
	return r.readRequest(s)
}

func (r *Router) readRequest(s grpcStream) error {
	for {
		req, err := s.Recv()
		if err != nil {
			return err
		}
		err = r.processRequest(req, s)
		if err != nil {
			return err
		}
	}
}

func (r *Router) processRequest(req *v2.DiscoveryRequest, s grpcStream) error {
	versionInfo := req.GetVersionInfo()
	nonce := req.GetResponseNonce()
	if versionInfo == "" && nonce == "" {
		go r.bootstrapResources(s, req.GetTypeUrl())
		return nil
	}

	pushID, i := readNonce(nonce)
	cb := r.PushCallbacks.get(pushID, i)
	log.Debugf("Router.processRequest: PushCallbacks get %s %s: %v", pushID, i, cb)
	if cb == nil {
		return nil
	}

	ch := cb.Channel
	defer close(ch)
	err := req.GetErrorDetail()
	log.Debugf("Router.processRequest: req error: %v", err)
	if err == nil {
		return nil
	}

	select {
	case ch <- errors.New(err.GoString()):
	case <-cb.Done():
		return cb.Err()
	}
	return nil
}

func (r *Router) StreamAggregatedResources(stream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	return r.handleGrpcStream(stream, AnyType)
}

func (r *Router) StreamEndpoints(stream v2.EndpointDiscoveryService_StreamEndpointsServer) error {
	log.Debug("Started to stream endpoints.")
	return r.handleGrpcStream(stream, EndpointType)
}

func (r *Router) StreamClusters(stream v2.ClusterDiscoveryService_StreamClustersServer) error {
	return r.handleGrpcStream(stream, ClusterType)
}

func (r *Router) StreamRoutes(stream v2.RouteDiscoveryService_StreamRoutesServer) error {
	return r.handleGrpcStream(stream, RouteType)
}

func (r *Router) StreamListeners(stream v2.ListenerDiscoveryService_StreamListenersServer) error {
	return r.handleGrpcStream(stream, ListenerType)
}

func (r *Router) IncrementalAggregatedResources(_ discovery.AggregatedDiscoveryService_IncrementalAggregatedResourcesServer) error {
	return errors.New("not implemented")
}

func (r *Router) IncrementalClusters(_ v2.ClusterDiscoveryService_IncrementalClustersServer) error {
	return errors.New("not implemented")
}

func (r *Router) IncrementalRoutes(_ v2.RouteDiscoveryService_IncrementalRoutesServer) error {
	return errors.New("not implemented")
}

func (r *Router) Fetch(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *Router) FetchEndpoints(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *Router) FetchClusters(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *Router) FetchRoutes(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *Router) FetchListeners(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}
