package admin

import (
	"context"
	"fmt"
	"net/http"
	"time"

	log "github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers"
)

func Init(ctx context.Context, confFile string, r routers.Router) error {
	err := readConfFile(confFile)
	if err != nil {
		return err
	}

	err = startServer(ctx, 18080, r)
	if err != nil {
		return err
	}

	return nil
}

func readConfFile(confFile string) error {
	return nil
}

func startServer(ctx context.Context, port uint, rt routers.Router) error {
	http.HandleFunc("/upsert_cluster", httpHandleFunc(ctx, rt, upsertCluster))
	http.HandleFunc("/upsert_route", httpHandleFunc(ctx, rt, upsertRoute))
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	}()
	return nil
}

type handleFuncType func(context.Context, routers.Router, http.ResponseWriter, *http.Request)
type httpHandleFuncType func(http.ResponseWriter, *http.Request)

func httpHandleFunc(ctx context.Context, rt routers.Router, f handleFuncType) httpHandleFuncType {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		f(ctx, rt, w, r)
	}
}

func upsertCluster(ctx context.Context, rt routers.Router, w http.ResponseWriter, r *http.Request) {
	// validate cluster
	// save cluster to db

	cluster := common.Cluster{
		Name: "cluster-apm",
		Endpoints: []common.Endpoint{
			common.Endpoint{
				Host: "162.216.20.141",
				Port: 80,
			},
		},
	}

	if err := rt.UpsertCluster(ctx, &cluster); err != nil {
		fmt.Fprintf(w, "error")
		return
	}
	fmt.Fprintf(w, "success")
}

func upsertRoute(ctx context.Context, rt routers.Router, w http.ResponseWriter, r *http.Request) {
	if err := rt.UpsertRoute(ctx); err != nil {
		fmt.Fprintf(w, "error")
		return
	}
	fmt.Fprintf(w, "success")
}
