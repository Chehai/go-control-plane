package routers

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers/envoy"
	"github.com/envoyproxy/go-control-plane/pkg/compass/stores"
)

type Router interface {
	Init(ctx context.Context, store stores.Store, confFile string) error
	UpsertCluster(ctx context.Context, cluster *common.Cluster) error
	UpsertRoute(ctx context.Context, route *common.Route) error
}

func NewEnvoyRouter() Router {
	return Router(&envoy.Router{})
}
