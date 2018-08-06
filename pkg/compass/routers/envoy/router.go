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
	store          stores.Store
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

func (r *Router) UpsertCluster(ctx context.Context, _ *common.Cluster) error {
	// Push all clusters from store to envoys, because the cluster is already upserted in store.
	return r.pushClusters(ctx)
}

func (r *Router) DeleteCluster(ctx context.Context, clusterName string) error {
	// Push all clusters from store to envoys, because the cluster is already deleted in store.
	return r.pushClusters(ctx)
}

func (r *Router) UpsertRoute(ctx context.Context, _ *common.Route) error {
	// Push all routes from store to envoy, because the route is already upserted in store.
	return r.pushRoutes(ctx)
}

func (r *Router) DeleteRoute(ctx context.Context, _ string) error {
	// Push all routes from store to envoy, because the route is already deleted in store.
	return r.pushRoutes(ctx)
}

func (r *Router) pushRoutes(ctx context.Context) error {
	routeResources, err := r.makeRouteResources(ctx)
	if err != nil {
		log.Errorf("Pushing routes failed: %v", err)
		return err
	}
	err = r.pushResources(ctx, routeResources, RouteType)
	if err != nil {
		log.Errorf("Pushing routes failed: %v", err)
		return err
	}
	return nil
}

func (r *Router) pushClusters(ctx context.Context) error {
	clusterResources, err := r.makeClusterResources(ctx)
	if err != nil {
		log.Errorf("Pushing clusters failed: %v", err)
		return err
	}
	err = r.pushResources(ctx, clusterResources, ClusterType)
	if err != nil {
		log.Errorf("Pushing clusters failed: %v", err)
		return err
	}
	endpointResources, err := r.makeEndpointResources(ctx)
	err = r.pushResources(ctx, endpointResources, EndpointType)
	if err != nil {
		log.Errorf("Pushing endpoints failed: %v", err)
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
