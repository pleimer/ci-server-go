package server

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/pleimer/ci-server-go/pkg/config"
	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/job"
	"github.com/pleimer/ci-server-go/pkg/logging"
)

// Run server main function
func Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	log, err := logging.NewLogger(logging.DEBUG, "console")
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

	gh := ghclient.NewClient(evChan)
	err = gh.Api.Authenticate(strings.NewReader(config.Oauth))
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("successfully authenticated with oauth token")

	wg.Add(1)
	server := gh.Listen(wg, config.ListenAddress, log)
	log.Info(fmt.Sprintf("listening on %s for webhooks", config.ListenAddress))

	jobChan := make(chan job.Job)

	jobManager := NewJobManager(4, log)
	wg.Add(1)
	go jobManager.Run(ctx, wg, jobChan)

	for {
		select {
		case ev := <-evChan:
			j, err := job.Factory(ev, &gh, log, config.Organization)
			if err != nil {
				log.Metadata(map[string]interface{}{"process": "server", "error": err})
				log.Error("failed creating job from event")
				break
			}
			jobChan <- j
		case <-ctx.Done():
			if err := server.Shutdown(ctx); err != nil {
				log.Metadata(map[string]interface{}{"process": "server", "error": err})
				log.Error("failed shutting down server gracefully")
			}
			return
		}
	}
}
