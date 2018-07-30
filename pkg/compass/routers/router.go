package router

import(
	"github.com/Chehai/go-control-plane/pkg/compass/routers/envoy"
	"github.com/Chehai/go-control-plane/pkg/compass/routers/envoy"

)

type Router interface {
	Init(ctx context.Context, confFile string) error
	UpsertCluster(ctx context.Context, cluster *Cluster) error {
}

func NewEnvoyRouter() Router {
	return envoy.Router{}
}
