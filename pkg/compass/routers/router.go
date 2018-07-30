package routers

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers/envoy"
)

type Router interface {
	Init(ctx context.Context, confFile string) error
	UpsertCluster(ctx context.Context, cluster *common.Cluster) error
}

func NewEnvoyRouter() Router {
	return &envoy.Router{}
}
