package envoy

import (
	"time"
	"context"
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

func makeRouteResource(routes []*common.Route) *v2.RouteConfiguration {
	vhm := make(map[string]route.VirtualHost)
	for _, r := range routes {
		if c, ok := vhm[r.Cluster]; !ok {
			vhm[r.Cluster] = route.VirtualHost{
				Name: r.Cluster,
				Domains: []string{r.Vhost},
				Routes: []route.Route{{
					Action: &route.Route_Route{
						Route: &route.RouteAction{
							ClusterSpecifier: &route.RouteAction_Cluster{
								Cluster: r.Cluster,
							},
						},
					},
				}},
			}
		} else {
			c.Domains = append(c.Domains, r.Vhost)
		}
	}

	vhs := make([]route.VirtualHost, 0, len(vhm))
	for _, v := vhm {
		vhs = append(vhs, v)
	}

	return &v2.RouteConfiguration{
		Name: routeConfigName,
		VirtualHosts: vhs,
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

func (r *Router) makeEndpointBootstrapResources(ctx context.Context) []*v2.ClusterLoadAssignment {
	clusters, err := r.store.GetClusters(ctx)
	if err != nil {
		return nil
	}
	ret := make([]*v2.ClusterLoadAssignment, 0, len(clusters))
	for _, c := clusters range {
		ret = append(ret, makeEndpointResource(c))
	}
	return ret
}

func (r *Router) makeRouteBootstrapResources(ctx context.Context) []*v2.RouteConfiguration {
	routes, err := r.store.GetRoutes(ctx)
	if err != nil {
		return nil
	}
	routeResource := makeRouteResource(routes)
	return []*v2.RouteConfiguration{
		routeResource,
	}
}

func (r *Router) makeClusterBootstrapResources(ctx context.Context) []*v2.Cluster {
	clusters, err := r.store.GetClusters(ctx)
	if err != nil {
		return nil
	}
	ret := make([]*v2.Cluster, 0, len(clusters))
	for _, c := clusters range {
		ret = append(ret, makeClusterResource(c))
	}
	return ret
}

func (r *Router) makeListenerBootstrapResources(_ context.Context) []*v2.Listener {
	return []*v2.Listener{
		makeListenerResource(),
	}
}