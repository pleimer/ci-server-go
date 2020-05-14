package server

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/infrawatch/ci-server-go/pkg/config"
	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/job"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

// Run server main function
func Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log, err := logging.NewLogger(logging.INFO, "console")
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Timestamp = true

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	defer log.Destroy()

	config, err := config.NewConfig()
	if err != nil {
		log.Error(err.Error())
		return
	}

	evChan := make(chan ghclient.Event)
	errChan := make(chan error)

	gh := ghclient.NewClient(evChan, errChan)
	err = gh.Api.Authenticate(strings.NewReader(config.Oauth))
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("successfully authenticated with oauth token")

	wg.Add(1)
	server := gh.Listen(wg, config.Proxy)
	log.Info(fmt.Sprintf("listening on %s for webhooks", config.Proxy))
	if err := server.Shutdown(ctx); err != nil {
		log.Error(fmt.Sprintf("error shutting down server gracefully: %s", err))
	}

	jobChan := make(chan job.Job)

	jobManager := NewJobManager(4, log)
	wg.Add(1)
	go jobManager.Run(ctx, wg, jobChan)

	select {
	case ev := <-evChan:
		j, err := job.Factory(ev, &gh, log)
		if err != nil {
			log.Error(err.Error())
			break
		}
		jobChan <- j

	case err := <-errChan:
		log.Error(fmt.Sprintf("%v", err))
		cancel()
	case <-ctx.Done():
		log.Info("server cleaning up resources")
	}
}
