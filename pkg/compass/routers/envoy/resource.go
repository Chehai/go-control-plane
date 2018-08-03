package envoy

import (
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/util"
)

const (
	xdsCluster      = "cluster-xds"
	routeConfigName = "route-apm"
	listenerName    = "listener-apm"
	listenerAddress = "0.0.0.0"
	listenerPort    = 9909
)

func makeListenerResource() *v2.Listener {
	// data source configuration
	rdsSource := core.ConfigSource{}
	rdsSource.ConfigSourceSpecifier = &core.ConfigSource_ApiConfigSource{
		ApiConfigSource: &core.ApiConfigSource{
			ApiType: core.ApiConfigSource_GRPC,
			GrpcServices: []*core.GrpcService{{
				TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
					EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: xdsCluster},
				},
			}},
		},
	}

	// HTTP filter configuration
	manager := &hcm.HttpConnectionManager{
		CodecType:  hcm.AUTO,
		StatPrefix: "http",
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				ConfigSource:    rdsSource,
				RouteConfigName: routeConfigName,
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: util.Router,
		}},
	}
	pbst, err := util.MessageToStruct(manager)
	if err != nil {
		panic(err)
	}

	return &v2.Listener{
		Name: listenerName,
		Address: core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  listenerAddress,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: listenerPort,
					},
				},
			},
		},
		FilterChains: []listener.FilterChain{{
			Filters: []listener.Filter{{
				Name:   util.HTTPConnectionManager,
				Config: pbst,
			}},
		}},
	}
}

func makeListenerResources() []*v2.Listener {
	return []*v2.Listener{
		makeListenerResource(),
	}
}

func makeRouteResource() *v2.RouteConfiguration {
	return &v2.RouteConfiguration{
		Name: routeConfigName,
		VirtualHosts: []route.VirtualHost{{
			Name:    "slice1",
			Domains: []string{"*"},
			Routes: []route.Route{{
				Match: route.RouteMatch{
					PathSpecifier: &route.RouteMatch_Prefix{
						Prefix: "/",
					},
				},
				Action: &route.Route_Route{
					Route: &route.RouteAction{
						ClusterSpecifier: &route.RouteAction_Cluster{
							Cluster: "cluster-apm",
						},
					},
				},
			}},
		}},
	}
}

func makeRouteResources() []*v2.RouteConfiguration {
	return []*v2.RouteConfiguration{
		//makeRouteResource(),
	}
}

func makeEndpointResource(cluster *common.Cluster) *v2.ClusterLoadAssignment {
	eps := make([]endpoint.LbEndpoint, 0, len(cluster.Endpoints))
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
	}
}

func makeEndpointResources() []*v2.ClusterLoadAssignment {
	// read from db
	// cluster := common.Cluster{
	// 	Name: "cluster-apm",
	// 	Endpoints: []common.Endpoint{
	// 		common.Endpoint{
	// 			Host: "162.216.20.141",
	// 			Port: 80,
	// 		},
	// 	},
	// }

	return []*v2.ClusterLoadAssignment{
		//makeEndpointResource(&cluster),
	}
}

func makeClusterResource(cluster *common.Cluster) *v2.Cluster {
	return &v2.Cluster{
		Name:           cluster.Name,
		ConnectTimeout: 30 * time.Second,
		Type:           v2.Cluster_EDS,
		EdsClusterConfig: &v2.Cluster_EdsClusterConfig{
			EdsConfig: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_ApiConfigSource{
					ApiConfigSource: &core.ApiConfigSource{
						ApiType: core.ApiConfigSource_GRPC,
						GrpcServices: []*core.GrpcService{{
							TargetSpecifier: &core.GrpcService_EnvoyGrpc_{
								EnvoyGrpc: &core.GrpcService_EnvoyGrpc{ClusterName: xdsCluster},
							},
						}},
					},
				},
			},
		},
	}
}

func makeClusterResources() []*v2.Cluster {
	// read from db
	// cluster := common.Cluster{
	// 	Name: "cluster-apm",
	// 	Endpoints: []common.Endpoint{
	// 		common.Endpoint{
	// 			Host: "162.216.20.141",
	// 			Port: 80,
	// 		},
	// 	},
	// }

	return []*v2.Cluster{
		// makeClusterResource(&cluster),
	}
}
