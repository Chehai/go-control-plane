package envoy

import (
	"context"
	"fmt"
	"net"
	"sync"
 	"sync/atomic"
	"io/ioutil"

	"github.com/ghodss/yaml"
	"github.com/rs/xid"
	log "github.com/sirupsen/logrus"

	"google.golang.org/grpc"
	"github.com/gogo/protobuf/types"
	"github.com/gogo/protobuf/proto"

	"github.com/envoyprox/go-control-plane/envoy/api/v2/core"
	"github.com/envoyprox/go-control-plane/envoy/api/v2"
	"github.com/envoyprox/go-control-plane/envoy/api/v2/endpoint"
	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
)

// Resource types in xDS v2.
const (
	typePrefix   = "type.googleapis.com/envoy.api.v2."
	EndpointType = typePrefix + "ClusterLoadAssignment"
	ClusterType  = typePrefix + "Cluster"
	RouteType    = typePrefix + "RouteConfiguration"
	ListenerType = typePrefix + "Listener"

	// AnyType is used only by ADS
	AnyType = ""
)

type Endpoint {
	Host string
	port uint
}

type Cluster struct {
	Name string
	Endpoints []Endpoint
}

type pushCallback struct {
	context.Context
	Channel chan error
}

type grpcStream interface {
	Send(*v2.DiscoveryResponse) error
	Recv() (*v2.DiscoveryRequest, error)
}

type pushStream struct {
	grpcStream
	sync.Mutex
}

type pushStreams struct {
	Streams []pushStream
	sync.RWMutex
}

// EnvoyRouter type
type EnvoyRouter struct {
	PushStreams map[string]pushStreams
	VersionCounter uint64
	PushCallbacks map[string]map[string]pushCallback
	port uint
}

type Resource interface {
	proto.Message
	Equal(interface{}) bool
}

type envoyRouterConfig struct {
	GrpcPort int 'json:"gprcPort"'
}

func (r *EnvoyRouter) initPushStreams() {
	r.PushStreams = map[string]pushStreams{
		EndpointType: pushStreams{Streams: make([]pushStream)},
		ClusterType: pushStreams{Streams: make([]pushStream)},
		RouteType: pushStreams{Streams: make([]pushStream)},
		ListenerType: pushStreams{Streams: make([]pushStream)},
	}
}


func (r *EnvoyRouter) readConfFile(confFile string) error {
	content, err := ioutil.ReadFile(confFile)
	if err != nil {
		return err
	}

	var config envoyRouterConfig
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return err
	}

	r.port = config.GrpcPort

	return nil
}

const grpcMaxConcurrentStreams = 1000000

func (r *EnvoyRouter) startGrpcServer(ctx context.Context) error {
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

	log.WithFields(log.Fields{"port": r.port}).Infof("Envoy gRPC Server listening on port %v", r.port)
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			log.Error(err)
		}
	}()

	go func() {
		<-ctx.Done()
		grpcServer.GracefulStop()
	}

	return nil
}

func (r *EnvoyRouter) Init(ctx context.Context, confFile string) {
	err := r.readConfFile(confFile)
	if err != nil {
		return err
	}
	r.initPushStreams()
	return r.startGrpcServer(ctx)
}

func (r *EnvoyRouter) UpsertCluster(ctx context.Context, cluster *Cluster) error {
	endpointResource, _ := makeEndpointResource(cluster)
	if err := r.pushResource(ctx, endpointResource, EndpointType); err != nil {
		return err
	}
	return nil
}

func makeEndpointResource(cluster *Cluster) (*v2.ClusterLoadAssignment, error) {
	eps := make([]endpoint.LbEndpoint)
	for ep := range cluster.Endpoints {
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
			LbEndpoints: eps
		}},
	}, nil
}

func pushResponse(stream pushStream, resp *v2.DiscoveryResponse) {
	stream.Lock()
	if err := stream.Send(r); err != nil {
		log.Error(err)
	}
	stream.Unlock()
}

func makeNonce(pushId *string, i int) *string {
	return &fmt.Sprintf("%s-%d", *pushId, i)
}

func (r *EnvoyRouter) makeVersionInfo() *string {
	v := atomic.AddUint64(&r.VersionCounter, 1)
	return &fmt.Sprintf("%d", v)
}

func (r *EnvoyRouter) createPushCallback(ctx context.Context, pushId string, i int) pushCallback {
	cb := pushCallback{Context: ctx, Channel: make(chan error)}
	is := fmt.Sprintf("%d", i)
	if _, ok := r.PushCallbacks[pushId]; !ok {
		m := make(map[string]pushCallback)
		m[is] = cb
		r.PushCallbacks[pushId] = m
		return cb
	}
	r.PushCallbacks[pushId][is] = cb
	return cb
}

func (r *EnvoyRouter) getPushCallback(pushId string, is string) pushCallBack{
	if _, ok := r.PushCallbacks[pushId]; !ok {
		return nil
	}
	return r.PushCallbacks[pushId][is]
}

func (r *EnvoyRouter) deletePushCallback(pushId string) {
	delete(r.PushCallbacks, pushId)
}

func makeResponse(resource Resource, typeUrl string, versionInfo string, nonce string) (*v2.DiscoveryResponse, error) {
	data, err := proto.Marshal(resource)
	if err != nil {
		return nil, err
	}
	return &v2.DiscoveryResponse{
		VersionInfo: versionInfo,
		Resources:   []types.Any{types.Any{TypeUrl: typeUrl, Value: data}},
		TypeUrl:     typeUrl,
		Nonce: nonce,
	}, nil
}

func (r *EnvoyRouter) pushResource(ctx context.Context, resource Resource, typeUrl string ) error {
	streams := r.PushStreams[typeUrl]
	if streams == nil {
		return fmt.Errorf("Cannot find streams for %s", *typeUrl)
	}

	pushId := xid.New().String()
	defer r.deletePushCallback(pushId)

	cbChs := make([]<-chan error)
	streams.RLock()
	for i, s := range streams
	{
		versionInfo := r.makeVersionInfo()
		nonce := makeNonce(&pushId, i)
		resp, err := makeResponse(resource, typeUrl, versionInfo, nonce)
		if err != nil {
			streams.RUnlock()
			return err
		}
		cb := r.createPushCallback(ctx, pushId, i)
		cbChs = append(cbChs, cb.Channel)
		go pushResponse(s, resp)
	}
	streams.RUnlock()

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

func (r *EnvoyRouter) handleGrpcStream(stream grpcStream, typeUrl string) error {
	r.createPushStream(stream, typeUrl)
	defer r.deletePushStream(stream, typeUrl)
	return r.readRequest(stream)
}

func (r *EnvoyRouter) createPushStream(stream grpcStream, typeUrl string) error {
	pStrm := pushStream{grpcStream: stream}
	pStrms, ok := r.PushStreams[typeUrl]
	if !ok {
		return fmt.Errorf("Cannot find push streams for %s", typeUrl)
	}
	pStrms.Streams.WLock()
	pStrms.Streams = append(pStrms.Streams, pStrm)
	pStrms.Streams.WUnlock()
	return nil
}

func (r *EnvoyRouter) deletePushStream(stream grpcStream, typeUrl string) error {
	pStrms, ok := r.PushStreams[typeUrl]
	if !ok {
		return fmt.Errorf("Cannot find push streams for %s", typeUrl)
	}
	pStrms.Streams.WLock()
	defer pStrms.Streams.WUnlock()
	for i, pStrm := pStrms.Streams {
		if pStrm.Stream == stream {
			copy(pStrms.Streams[i:], pStrms.Streams[i+1:])
			l := len(pStrms.Streams)
			pStrms.Streams[l - 1] = nil
			pStrms.Streams = pStrms.Streams[:l-1]
			return nil
		}
	}
	return fmt.Errorf("Cannot find push stream %v for %s", stream, typeUrl)
}

func (r *EnvoyRouter) readRequest(stream grpcStream) error {
	defer stream.Close()
	for {
		req, err := stream.Recv()
		if err != nil {
			return err
		}
		err = r.processRequest(req)
		if err != nil {
			return err
		}
	}
}

func (r *EnvoyRouter) processRequest(req *v2.DiscoveryRequest) error {
	versionInfo := req.GetVersionInfo()
	nonce := req.GetResponseNonce()
	if versionInfo == "" && nonce == "" {
		r.pushResourcesToStream(stream, req.GetTypeUrl())
		return nil
	}

	var pushId, i string
	fmt.Sscan(nonce, "%s-%s", &pushId, &i)
	cb := r.getPushCallback(pushId, i)
	if cb == nil {
		return nil
	}

	channel := cb.Channel
	defer close(channel)
	err := req.GetErrorDetail().Err()

	select {
		case channel<- err:
		case <-cb.Done():
			return
	}
	return nil
}

func (r *EnvoyRouter) StreamAggregatedResources(stream discovery.AggregatedDiscoveryService_StreamAggregatedResourcesServer) error {
	return r.handleGrpcStream(stream, &r.adsResponseChannels)
}

func (r *EnvoyRouter) StreamEndpoints(stream v2.EndpointDiscoveryService_StreamEndpointsServer) error {
	return r.handleGrpcStream(stream, &r.endpointResponseChannels)
}

func (r *EnvoyRouter) StreamClusters(stream v2.ClusterDiscoveryService_StreamClustersServer) error {
	return r.handleGrpcStream(stream, &r.clusterResponseChannels)
}

func (r *EnvoyRouter) StreamRoutes(stream v2.RouteDiscoveryService_StreamRoutesServer) error {
	return r.handleGrpcStream(stream, &r.routeResponseChannels)
}

func (r *EnvoyRouter) StreamListeners(stream v2.ListenerDiscoveryService_StreamListenersServer) error {
	return r.handleGrpcStream(stream, &r.listenerResponseChannels)
}

func (r *EnvoyRouter) IncrementalAggregatedResources(_ discovery.AggregatedDiscoveryService_IncrementalAggregatedResourcesServer) error {
	return errors.New("not implemented")
}

func (r *EnvoyRouter) IncrementalClusters(_ v2.ClusterDiscoveryService_IncrementalClustersServer) error {
	return errors.New("not implemented")
}

func (r *EnvoyRouter) IncrementalRoutes(_ v2.RouteDiscoveryService_IncrementalRoutesServer) error {
	return errors.New("not implemented")
}

func (r *EnvoyRouter) Fetch(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *EnvoyRouter) FetchEndpoints(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *EnvoyRouter) FetchClusters(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *EnvoyRouter) FetchRoutes(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}

func (r *EnvoyRouter) FetchListeners(ctx context.Context, req *v2.DiscoveryRequest) (*v2.DiscoveryResponse, error) {
	return nil, errors.New("not implemented")
}
