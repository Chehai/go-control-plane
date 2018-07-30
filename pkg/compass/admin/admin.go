package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
)

func Init(ctx context.Context, confFile string) error {
	err := readConfFile(confFile)
	if err != nil {
		return err
	}

	err = startServer(ctx, 18080)
	if err != nil {
		return err
	}

	return nil
}

func readConfFile(confFile string) error {
	return nil
}

func startServer(ctx context.Context, port uint) error {
	http.HandleFunc("/upsert_vhost/", upsertVhost)
	// http.HandleFunc("/delete_vhost/", deleteVhost)
	// http.HandleFunc("/upsert_cluster/", upsertCluster)
	// http.HandleFunc("/delete_cluster/", deleteCluster)

	go log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
	return nil
}


func upsertVhost(w http.ResponseWriter, r *http.Request) {
	// validate vhost
	// save vhost to db
	fmt.Fprintf(w, "Welcome to my website!")

}

func deleteVhost(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to my website!")
}

type endpoint struct {
	host string
	port uint16
}

func (r *Router) UpdateCluster(ctx contexts.Context, cl []endpoint) (chan string, error) {
	return a, b
}

var router Router

func upsertCluster(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)

	defer cancel()

	cluster := "test"
	// validate cluster
	// save cluster to db

	if err := router.UpdateCluster(ctx, cluster); err != nil {
		fmt.Fprintf(w, "error")
		return
	}
	fmt.Fprintf(w, "success")
}

func deleteCluster(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Welcome to my website!")
}

func httpHandlFunc(upsertCluster, router) {
	return func (w, r) {
		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
		defer cancel()
		handleRequest(ctx, w, r, router)
	}
}
