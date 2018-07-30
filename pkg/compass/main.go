package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/envoyproxy/go-control-plane/pkg/compass/admin"
	"github.com/envoyproxy/go-control-plane/pkg/compass/routers"
)

func main() {
	fmt.Println("Compass started")
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		cancel()
	}()

	r := routers.NewEnvoyRouter()
	r.Init(ctx, "todo")
	admin.Init(ctx, "todo", r)

	<-ctx.Done()
	fmt.Println("Compass stopped")
}
