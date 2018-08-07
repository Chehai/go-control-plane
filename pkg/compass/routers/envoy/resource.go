package envoy

import (
	"context"
	"time"

	"github.com/envoyproxy/go-control-plane/envoy/api/v2"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	"github.com/envoyproxy/go-control-plane/envoy/api/v2/route"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
)

const (
	routeConfigName       = "route-apm"
	dnsRefreshRate        = 1 * time.Minute
	clusterConnectTimeout = 30 * time.Second
)

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
