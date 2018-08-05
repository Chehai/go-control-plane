package admin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers"
	"github.com/envoyproxy/go-control-plane/pkg/compass/stores"

)

type admin struct {
	router routers.Router
	store stores.Store
}

func Init(ctx context.Context, confFile string, r routers.Router, s stores.Store) error {
	err := readConfFile(confFile)
	if err != nil {
		return err
	}

	err = startServer(ctx, 18080, admin{router: r, store: s})
	if err != nil {
		return err
	}

	return nil
}

func readConfFile(confFile string) error {
	return nil
}

func startServer(ctx context.Context, port uint, a admin) error {
	http.HandleFunc("/upsert_cluster", httpHandleFunc(ctx, a, upsertCluster))
	http.HandleFunc("/upsert_route", httpHandleFunc(ctx, a, upsertRoute))
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}()
	return nil
}

type handleFuncType func(context.Context, admin, http.ResponseWriter, *http.Request)
type httpHandleFuncType func(http.ResponseWriter, *http.Request)

func httpHandleFunc(ctx context.Context, a admin, f handleFuncType) httpHandleFuncType {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		f(ctx, a, w, r)
	}
}

func upsertCluster(ctx context.Context, a admin, w http.ResponseWriter, r *http.Request) {
	// validate cluster
	// save cluster to db
	// db.parse

	cluster := common.Cluster{
		Name: "cluster-apm",
		Endpoints: []common.Endpoint{
			common.Endpoint{
				Host: "162.216.20.141",
				Port: 80,
			},
		},
	}

	err := a.store.UpsertCluster(ctx, &cluster)
	if err != nil {
		fmt.Fprintf(w, "error")
		return
	}

	err = a.router.UpsertCluster(ctx, &cluster)
	if err != nil {
		fmt.Fprintf(w, "error")
		return
	}

	fmt.Fprintf(w, "success")
}

func upsertRoute(ctx context.Context, a admin, w http.ResponseWriter, r *http.Request) {
	route := common.Route{
		Vhost: "*"
		Cluster: "cluster-apm"
	}

	err := a.store.UpsertRoute(ctx, &route)
	if err != nil {
		fmt.Fprintf(w, "error")
		return
	}

	err = a.router.UpsertRoute(ctx, &route)
	if err != nil {
		fmt.Fprintf(w, "error")
		return
	}

	fmt.Fprintf(w, "success")
}
