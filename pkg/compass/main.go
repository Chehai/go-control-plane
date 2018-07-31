package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/pkg/compass/admin"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers"
)

func main() {
	fmt.Println("Compass started")
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	defer func() {
		signal.Stop(c)
		cancel()
	}()
	go func() {
		<-c
		log.Info("Stopping Compass")
		cancel()
	}()

	r := routers.NewEnvoyRouter()
	log.Info("Init router")
	r.Init(ctx, "todo")
	log.Info("Init admin")
	admin.Init(ctx, "todo", r)

	log.Info("Compass init done")
	<-ctx.Done()
	fmt.Println("Compass stopped")
}
