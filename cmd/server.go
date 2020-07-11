package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"sync"

	"github.com/pleimer/ci-server-go/pkg/server"
)

var configPath string

func init() {
	flag.Usage = func() {
		flag.PrintDefaults()
	}

	flag.StringVar(&configPath, "config", "/etc/ci-server-go.conf.yaml", "path to config file")
	flag.Parse()
}

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

	err := server.Init(configPath)
	if err != nil {
		fmt.Println(err)
		return
	}
	defer server.Close()

	server.Run(ctx, &wg)
	wg.Wait()
}
