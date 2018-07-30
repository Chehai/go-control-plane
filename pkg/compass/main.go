package main
import (
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"context"
	//"github.com/Chehai/go-control-plane/pkg/compass/routers/router"
	//"github.com/Chehai/go-control-plane/pkg/compass/admin"
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

	r := router.NewEnvoyRouter()
	r.Init(ctx, "todo")
	admin.Init(ctx, "todo", r)

	<-ctx.Done()
	fmt.Println("Compass stopped")
}
