package admin

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"
	"github.com/rs/xid"

)

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

// Router is test
type Router struct {
}

func (r *Router) UpdateCluster(ctx contexts.Context, cl []endpoint) (chan string, error) {
	return a, b
}

var router Router

func upsertCluster(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Second)
	ctx := context.WithValue(ctx, "request-id", xid.New())

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

// RunAdminServer admin interface
func RunAdminServer() {
	http.HandleFunc("/upsert_vhost/", upsertVhost)
	http.HandleFunc("/delete_vhost/", deleteVhost)
	http.HandleFunc("/upsert_cluster/", upsertCluster)
	http.HandleFunc("/delete_cluster/", deleteCluster)

	log.Fatal(http.ListenAndServe(":18080", nil))
}
