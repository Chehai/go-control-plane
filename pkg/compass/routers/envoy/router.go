package envoy

import (
	"context"
	"io/ioutil"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
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
}

func (r *Router) Init(ctx context.Context, confFile string) error {
	// err := r.readConfFile(confFile)
	// if err != nil {
	// 	return err
	// }
	r.port = 18088
	r.initPushStreams()
	r.initPushCallbacks()
	return r.startGrpcServer(ctx)
}

func (r *Router) UpsertCluster(ctx context.Context, cluster *common.Cluster) error {
	log.Debug("Router.UpsertCluster: before makeClusterResource")
	clusterResource := makeClusterResource(cluster)
	if err := r.pushResource(ctx, clusterResource, ClusterType); err != nil {
		return err
	}
	endpointResource := makeEndpointResource(cluster)
	log.Debug("Router.upsertCluster: before push Enpdoint resources")
	if err := r.pushResource(ctx, endpointResource, EndpointType); err != nil {
		log.Debugf("Router.UpsertCluster: pushResource Endpoint err %v", err)
		return err
	}
	return nil
}

func (r *Router) UpsertRoute(ctx context.Context) error {
	routeResource := makeRouteResource()
	if err := r.pushResource(ctx, routeResource, RouteType); err != nil {
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
