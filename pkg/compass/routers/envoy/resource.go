package envoy

import (
	"context"
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
	xdsCluster            = "cluster-xds"
	routeConfigName       = "route-apm"
	listenerName          = "listener-apm"
	listenerAddress       = "0.0.0.0"
	listenerPort          = 10000
	dnsRefreshRate        = 1 * time.Minute
	clusterConnectTimeout = 30 * time.Second
)

func (r *Router) makeListenerResources(_ context.Context) ([]resource, error) {
	listener, err := makeListenerResource()
	if err != nil {
		return nil, err
	}
	return []resource{
		resource(listener),
	}, nil
}

func (r *Router) makeClusterResources(ctx context.Context) ([]resource, error) {
	clusters, err := r.store.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]resource, 0, len(clusters))
	for _, c := range clusters {
		res, err := makeClusterResource(c)
		if err != nil {
			return nil, err
		}
		ret = append(ret, resource(res))
	}
	return ret, nil
}

func (r *Router) makeEndpointResources(ctx context.Context) ([]resource, error) {
	clusters, err := r.store.GetClusters(ctx)
	if err != nil {
		return nil, err
	}
	ret := make([]resource, 0, len(clusters))
	for _, c := range clusters {
		res, err := makeEndpointResource(c)
		if err != nil {
			return nil, err
		}
		ret = append(ret, resource(res))
	}
	return ret, nil
}

func (r *Router) makeRouteResources(ctx context.Context) ([]resource, error) {
	routes, err := r.store.GetRoutes(ctx)
	if err != nil {
		return nil, err
	}
	routeResource, err := makeRouteResource(routes)
	if err != nil {
		return nil, err
	}
	return []resource{
		resource(routeResource),
	}, nil
}

func makeListenerResource() (*v2.Listener, error) {
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
		return nil, err
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
	}, nil
}

func makeRouteResource(routes []*common.Route) (*v2.RouteConfiguration, error) {
	vhm := make(map[string]*route.VirtualHost)
	for _, r := range routes {
		pvh, ok := vhm[r.Cluster]
		if !ok {
			pvh = &route.VirtualHost{
				Name:    r.Cluster,
				Domains: []string{},
				Routes: []route.Route{{
					Match: route.RouteMatch{
						PathSpecifier: &route.RouteMatch_Prefix{
							Prefix: "/",
						},
					},
					Action: &route.Route_Route{
						Route: &route.RouteAction{
							ClusterSpecifier: &route.RouteAction_Cluster{
								Cluster: r.Cluster,
							},
						},
					},
				}},
			}
		}
		pvh.Domains = append(pvh.Domains, r.Vhost)
		vhm[r.Cluster] = pvh
	}

	vhs := make([]route.VirtualHost, 0, len(vhm))
	for _, v := range vhm {
		vhs = append(vhs, *v)
	}

	return &v2.RouteConfiguration{
		Name:         routeConfigName,
		VirtualHosts: vhs,
	}, nil
}

func makeEndpointResource(cluster *common.Cluster) (*v2.ClusterLoadAssignment, error) {
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
	}, nil
}

func makeClusterResource(cluster *common.Cluster) (*v2.Cluster, error) {
	endpoints := cluster.Endpoints
	hosts := make([]*core.Address, len(endpoints))
	for i, e := range endpoints {
		hosts[i] = &core.Address{
			Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Protocol: core.TCP,
					Address:  e.Host,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: e.Port,
					},
				},
			},
		}
	}
	// cannot take address of constants
	localDNSRefreshRate := dnsRefreshRate
	return &v2.Cluster{
		Name:            cluster.Name,
		ConnectTimeout:  clusterConnectTimeout,
		Type:            v2.Cluster_STRICT_DNS,
		LbPolicy:        v2.Cluster_ROUND_ROBIN,
		DnsLookupFamily: v2.Cluster_V4_ONLY,
		DnsRefreshRate:  &localDNSRefreshRate,
		Hosts:           hosts,
	}, nil
}
