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
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
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

func makeEndpointResource(cluster *common.Cluster) (*v2.ClusterLoadAssignment, error) {
	eps := make([]endpoint.LbEndpoint, len(cluster.Endpoints))
	for _, ep := range cluster.Endpoints {
		eps = append(eps, endpoint.LbEndpoint{
			Endpoint: &endpoint.Endpoint{
				Address: &core.Address{
					Address: &core.Address_SocketAddress{
						SocketAddress: &core.SocketAddress{
							Protocol: core.TCP,
							Address:  ep.Host,
							PortSpecifier: &core.SocketAddress_PortValue{
								PortValue: ep.Port,
							},
						},
					},
				},
			},
		})
	}
	return &v2.ClusterLoadAssignment{
		ClusterName: cluster.Name,
		Endpoints: []endpoint.LocalityLbEndpoints{{
			LbEndpoints: eps,
		}},
	}, nil
}

func pushResponse(stream *pushStream, resp *v2.DiscoveryResponse) {
	stream.Lock()
	defer stream.Unlock()
	if err := stream.Send(resp); err != nil {
		log.Error(err)
	}
}

func makeNonce(pushId *string, i int) string {
	return fmt.Sprintf("%s-%d", *pushId, i)
}

func (r *Router) makeVersionInfo() string {
	v := atomic.AddUint64(&r.VersionCounter, 1)
	return fmt.Sprintf("%d", v)
}

func makeResponse(res resource, typeUrl string, versionInfo string, nonce string) (*v2.DiscoveryResponse, error) {
	data, err := proto.Marshal(res)
	if err != nil {
		return nil, err
	}
	return &v2.DiscoveryResponse{
		VersionInfo: versionInfo,
		Resources:   []types.Any{types.Any{TypeUrl: typeUrl, Value: data}},
		TypeUrl:     typeUrl,
		Nonce:       nonce,
	}, nil
}

func (r *Router) pushResource(ctx context.Context, res resource, typeUrl string) error {
	streams := r.PushStreams.get(typeUrl)
	if streams == nil {
		return fmt.Errorf("Cannot find streams for %s", typeUrl)
	}

	pushId := xid.New().String()
	defer r.PushCallbacks.delete(pushId)

	cbChs := make([]<-chan error, len(streams))
	for i, s := range streams {
		versionInfo := r.makeVersionInfo()
		nonce := makeNonce(&pushId, i)
		resp, err := makeResponse(res, typeUrl, versionInfo, nonce)
		if err != nil {
			return err
		}
		ch := make(chan error)
		r.PushCallbacks.create(pushId, fmt.Sprintf("%d", i), ctx, ch)
		cbChs = append(cbChs, ch)
		go pushResponse(s, resp)
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
	log.Infof("handleGrpcStream: %v %s", s, typeUrl)
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
		err = r.processRequest(req)
		if err != nil {
			return err
		}
	}
}

func (r *Router) processRequest(req *v2.DiscoveryRequest) error {
	versionInfo := req.GetVersionInfo()
	nonce := req.GetResponseNonce()
	if versionInfo == "" && nonce == "" {
		//r.pushResourcesToStream(stream, req.GetTypeUrl())
		return nil
	}

	var pushId, i string
	fmt.Sscan(nonce, "%s-%s", &pushId, &i)
	cb := r.PushCallbacks.get(pushId, i)
	if cb == nil {
		return nil
	}

	ch := cb.Channel
	defer close(ch)
	err := req.GetErrorDetail()
	if err == nil {
		close(ch)
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
