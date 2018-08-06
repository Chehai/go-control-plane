package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	log "github.com/sirupsen/logrus"

	"github.com/envoyproxy/go-control-plane/pkg/compass/admin"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers"
	"github.com/envoyproxy/go-control-plane/pkg/compass/stores"
)

func init() {
	log.SetOutput(os.Stdout)
	log.SetLevel(log.DebugLevel)
}

func main() {
	log.Info("Compass started")
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

	log.Info("Init store")
	s := stores.NewMysqlStore()
	err := s.Init(ctx, "todo")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Init router")
	r := routers.NewEnvoyRouter()
	err = r.Init(ctx, s, "todo")
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Init admin")
	err = admin.Init(ctx, "todo", r, s)
	if err != nil {
		log.Fatal(err)
	}

	log.Info("Compass init done")

	<-ctx.Done()
	log.Info("Compass stopped")
}
