package stores

import (
	"context"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/compass/stores/mysql"
)

type Store interface {
	Init(ctx context.Context, confFile string) error
	UpsertCluster(ctx context.Context, cluster *common.Cluster) error
	UpsertRoute(ctx context.Context, route *common.Route) error
	DeleteCluster(ctx context.Context, clusterName string) error
	DeleteRoute(ctx context.Context, vhost string) error
	GetRoute(ctx context.Context, vhost string) (*common.Route, error)
	GetRoutes(ctx context.Context) ([]*common.Route, error)
	GetClusters(ctx context.Context) ([]*common.Cluster, error)
}

func NewMysqlStore() Store {
	return Store(&mysql.Store{})
}
