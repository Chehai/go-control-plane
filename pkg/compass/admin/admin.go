package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

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
	http.HandleFunc("/upsert_vhost/", httpHandleFunc(ctx, rt, upsertCluster))
	go log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
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
		Name: "test-cluster",
		Endpoints: []common.Endpoint{
			common.Endpoint{
				Host: "www.google.com",
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
