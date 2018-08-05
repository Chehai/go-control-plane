package envoy

import (
	"context"
	"io/ioutil"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/compass/stores"
	log "github.com/sirupsen/logrus"

	"github.com/ghodss/yaml"
)

type routerConfig struct {
	GrpcPort uint `json:"gprcPort"`
}

type Router struct {
	VersionCounter uint64
	port           uint
	PushStreams    pushStreams
	PushCallbacks  pushCallbacks
	store 				 stores.Store
}

func (r *Router) Init(ctx context.Context, store stores.Store, confFile string) error {
	// err := r.readConfFile(confFile)
	// if err != nil {
	// 	return err
	// }
	r.port = 18088
	r.store = store
	r.initPushStreams()
	r.initPushCallbacks()
	return r.startGrpcServer(ctx)
}

func (r *Router) UpsertCluster(ctx context.Context, cluster *common.Cluster) error {
	clusterResource := makeClusterResource(cluster)
	if err := r.pushResource(ctx, clusterResource, ClusterType); err != nil {
		log.Errorf("Upserting cluster failed: %v", err)
		return err
	}
	endpointResource := makeEndpointResource(cluster)
	if err := r.pushResource(ctx, endpointResource, EndpointType); err != nil {
		log.Errorf("Upserting cluster failed: %v", err)
		return err
	}
	return nil
}

func (r *Router) UpsertRoute(ctx context.Context, _ *common.Route) error {
	routes, err := r.store.GetRoutes()
	if err != nil {
		log.Errorf("Upserting route failed: %v", err)
		return err
	}
	routeResource := makeRouteResource(routes)
	err = r.pushResource(ctx, routeResource, RouteType)
	if err != nil {
		log.Errorf("Upserting route failed: %v", err)
		return err
	}
	return nil
}

func (r *Router) initPushStreams() {
	r.PushStreams.init([]string{
		EndpointType,
		ClusterType,
		RouteType, ListenerType,
	})
}

func (r *Router) initPushCallbacks() {
	r.PushCallbacks.init()
}

func (r *Router) readConfFile(confFile string) error {
	content, err := ioutil.ReadFile(confFile)
	if err != nil {
		return err
	}

	var config routerConfig
	err = yaml.Unmarshal(content, &config)
	if err != nil {
		return err
	}

	r.port = config.GrpcPort

	return nil
}
