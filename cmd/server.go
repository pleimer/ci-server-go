package main

import (
	"context"
	"os"
	"os/signal"
	"sync"

	"github.com/infrawatch/ci-server-go/pkg/server"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)
	defer func() {
		signal.Stop(c)
		cancel()
	}()

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		select {
		case <-c:
			cancel()
		case <-ctx.Done():
		}
	}()

	server.Run(ctx, &wg)
	wg.Wait()
	// Print log of successful exit right here
}
