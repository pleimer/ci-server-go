package server

import (
	"context"
	"fmt"
	"os"
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
		fmt.Printf("error creating logger: %s\n", err)
		return
	}
	log.Timestamp = true

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	defer log.Destroy()

	config := config.NewConfig()
	file, err := os.Open("ci-server-config.yaml")
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "server", "error": err})
		log.Error("failed reading config file")
		return
	}
	defer file.Close()

	err = config.Parse(file)
	if err != nil {
		log.Metadata(map[string]interface{}{"process": "server", "error": err})
		log.Error("failed parsing config")
		return
	}

	evChan := make(chan ghclient.Event)

	gh := ghclient.NewClient(evChan, config.GetUser())
	err = gh.Api.Authenticate(strings.NewReader(config.GetOauth()))
	if err != nil {
		log.Error(err.Error())
		return
	}
	log.Info("successfully authenticated with oauth token")

	wg.Add(1)
	server := gh.Listen(wg, config.GetAddress(), log)
	log.Info(fmt.Sprintf("listening on %s for webhooks", config.GetAddress()))

	jobChan := make(chan job.Job)

	jobManager := NewJobManager(config.GetNumWorkers(), log)
	wg.Add(1)
	go jobManager.Run(ctx, wg, jobChan, config.GetAuthorizedUsers())

	for {
		select {
		case ev := <-evChan:
			j, err := job.Factory(ev, &gh, log)
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
