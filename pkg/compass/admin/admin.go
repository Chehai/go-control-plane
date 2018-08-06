package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/pkg/compass/common"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers"
	"github.com/envoyproxy/go-control-plane/pkg/compass/stores"
)

type admin struct {
	router routers.Router
	store  stores.Store
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
	r := mux.NewRouter()
	r.HandleFunc("/upsert_cluster", httpHandleFunc(ctx, a, upsertCluster))
	r.HandleFunc("/upsert_route", httpHandleFunc(ctx, a, upsertRoute))
	r.HandleFunc("/delete_cluster", httpHandleFunc(ctx, a, deleteCluster))
	r.HandleFunc("/delete_route", httpHandleFunc(ctx, a, deleteRoute))
	r.HandleFunc("/get_route", httpHandleFunc(ctx, a, getRoute))
	go func() {
		log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), r))
	}()
	return nil
}

type handleFuncType func(context.Context, admin, http.ResponseWriter, *http.Request)

func httpHandleFunc(ctx context.Context, a admin, f handleFuncType) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(ctx, 1*time.Minute)
		defer cancel()
		f(ctx, a, w, r)
	}
}

func upsertCluster(ctx context.Context, a admin, w http.ResponseWriter, r *http.Request) {
	cluster, err := parseCluster(r)
	if err != nil {
		http.Error(w, "Parsing cluster failed", http.StatusUnsupportedMediaType)
		return
	}

	err = a.store.UpsertCluster(ctx, cluster)
	if err != nil {
		log.Errorf("Upserting cluster to store failed: %v", err)
		http.Error(w, "Upserting cluster to store failed", http.StatusUnprocessableEntity)
		return
	}

	err = a.router.UpsertCluster(ctx, cluster)
	if err != nil {
		log.Errorf("Upserting cluster to router failed: %v", err)
		http.Error(w, "Upserting cluster to router failed", http.StatusUnprocessableEntity)
		return
	}

	fmt.Fprintf(w, "Upserting clustser succeeded")
}

func upsertRoute(ctx context.Context, a admin, w http.ResponseWriter, r *http.Request) {
	route, err := parseRoute(r)
	if err != nil {
		http.Error(w, "Parsing route failed", http.StatusUnsupportedMediaType)
		return
	}

	err = a.store.UpsertRoute(ctx, route)
	if err != nil {
		log.Errorf("Upserting route to store failed: %v", err)
		http.Error(w, "Upserting route to store failed", http.StatusUnprocessableEntity)
		return
	}

	err = a.router.UpsertRoute(ctx, route)
	if err != nil {
		log.Errorf("Upserting cluster to router failed: %v", err)
		http.Error(w, "Upserting cluster to router failed", http.StatusUnprocessableEntity)
		return
	}

	fmt.Fprintf(w, "Upserting route succeeded")
}

func deleteRoute(ctx context.Context, a admin, w http.ResponseWriter, r *http.Request) {
	route, err := parseRoute(r)
	if err != nil {
		http.Error(w, "Parsing route failed", http.StatusUnsupportedMediaType)
		return
	}
	vhost := route.Vhost

	err = a.store.DeleteRoute(ctx, vhost)
	if err != nil {
		log.Errorf("Deleting route from store failed: %v", err)
		http.Error(w, "Deleting route from store failed", http.StatusUnprocessableEntity)
		return
	}

	err = a.router.DeleteRoute(ctx, vhost)
	if err != nil {
		log.Errorf("Deleting route from router failed: %v", err)
		http.Error(w, "Deleting route from router failed", http.StatusUnprocessableEntity)
		return
	}

	fmt.Fprintf(w, "Deleting route succeeded")
}

func deleteCluster(ctx context.Context, a admin, w http.ResponseWriter, r *http.Request) {
	cluster, err := parseCluster(r)
	if err != nil {
		http.Error(w, "Decoding cluster json failed", http.StatusUnsupportedMediaType)
		return
	}
	clusterName := cluster.Name

	err = a.store.DeleteCluster(ctx, clusterName)
	if err != nil {
		log.Errorf("Deleting cluster from store failed: %v", err)
		http.Error(w, "Deleting cluster from store failed", http.StatusUnprocessableEntity)
		return
	}

	err = a.router.DeleteCluster(ctx, clusterName)
	if err != nil {
		log.Errorf("Deleting cluster from router failed: %v", err)
		http.Error(w, "Deleting cluster from router failed", http.StatusUnprocessableEntity)
		return
	}

	fmt.Fprintf(w, "Deleting cluster succeeded")
}

func getRoute(ctx context.Context, a admin, w http.ResponseWriter, r *http.Request) {
	rte, err := parseRoute(r)
	if err != nil {
		http.Error(w, "Parsing route failed", http.StatusUnsupportedMediaType)
	}
	vhost := rte.Vhost

	route, err := a.store.GetRoute(ctx, vhost)
	if err != nil {
		log.Errorf("Getting route from store failed: %v", err)
		http.Error(w, "Getting route from store failed", http.StatusUnprocessableEntity)
		return
	}

	fmt.Fprintf(w, "%s", route.Cluster)
}

func parseCluster(r *http.Request) (*common.Cluster, error) {
	decoder := json.NewDecoder(r.Body)
	var cluster common.Cluster
	err := decoder.Decode(&cluster)
	if err != nil {
		log.Errorf("Decoding cluster json failed: %v", err)
		return nil, err
	}
	return &cluster, nil
}

func parseRoute(r *http.Request) (*common.Route, error) {
	decoder := json.NewDecoder(r.Body)
	var route common.Route
	err := decoder.Decode(&route)
	if err != nil {
		log.Errorf("Decoding route json failed: %v", err)
		return nil, err
	}
	return &route, nil
}
