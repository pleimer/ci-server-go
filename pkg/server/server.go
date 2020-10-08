package server

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/pleimer/ci-server-go/pkg/config"
	"github.com/pleimer/ci-server-go/pkg/ghclient"
	"github.com/pleimer/ci-server-go/pkg/job"
	"github.com/pleimer/ci-server-go/pkg/logging"
)

var (
	serverConfig *config.Config
	logger       *logging.Logger
	github       *ghclient.Client
	jobManager   *JobManager
	eventChan    chan ghclient.Event
	jobChan      chan job.Job
)

// Init initialize server resources
func Init(configPath string) error {
	serverConfig = config.New()

	file, err := os.Open(configPath)
	if err != nil {
		return errors.Wrap(err, "failed opening configuration file")
	}
	defer file.Close()

	err = serverConfig.Parse(file)
	if err != nil {
		return errors.Wrap(err, "failed parsing configuration file")
	}

	logger, err = logging.NewLogger(logging.FromString(serverConfig.Logger.Level), serverConfig.Logger.Target)
	if err != nil {
		return fmt.Errorf("error creating logger: %s", err)
	}
	logger.Timestamp = true
	logger.Info(fmt.Sprintf("initialized logger to level %s", serverConfig.Logger.Level))

	eventChan = make(chan ghclient.Event)
	github = ghclient.NewClient(eventChan, serverConfig.Github.User)
	err = github.Api.Authenticate(strings.NewReader(serverConfig.Github.Oauth))
	if err != nil {
		logger.Metadata(map[string]interface{}{"module": "server", "error": err})
		logger.Error("failed to authenticate github with oauth")
		return err
	}
	logger.Metadata(map[string]interface{}{"module": "server"})
	logger.Info("successfully authenticated github with oauth token")

	jobChan = make(chan job.Job)
	jobManager = NewJobManager(serverConfig.Runner.NumWorkers, logger)

	return nil
}

// Run server main function
func Run(ctx context.Context, wg *sync.WaitGroup) {
	defer wg.Done()

	if err := checkResources(); err != nil {
		fmt.Println(err)
		return
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	wg.Add(1)
	server := github.Listen(wg, serverConfig.Listener.Address, logger)
	logger.Info(fmt.Sprintf("listening on %s for webhooks", serverConfig.Listener.Address))

	wg.Add(1)
	go jobManager.Run(ctx, wg, jobChan, serverConfig.Runner.AuthorizedUsers)

	for {
		select {
		case ev := <-eventChan:
			j, err := job.Factory(ev, github, logger)
			if err != nil {
				logger.Metadata(map[string]interface{}{"process": "server", "error": err})
				logger.Error("failed creating job from event")
				break
			}
			jobChan <- j
		case <-ctx.Done():
			if err := server.Shutdown(ctx); err != nil {
				logger.Metadata(map[string]interface{}{"process": "server", "error": err})
				logger.Error("failed shutting down server gracefully")
			}
			return
		}
	}
}

//Close cleanup server resources
func Close() {
	if err := checkResources(); err != nil {
		return
	}

	logger.Metadata(map[string]interface{}{"process": "server"})
	logger.Info("cleaning up server resources")
	err := logger.Destroy()
	close(eventChan)
	close(jobChan)

	if err != nil {
		fmt.Printf("there was an error while closing the logger: %s", err)
		return
	}
	fmt.Println("server exited cleanly")
}

func checkResources() error {
	if serverConfig == nil ||
		logger == nil ||
		github == nil ||
		jobManager == nil ||
		eventChan == nil ||
		jobChan == nil {

		return fmt.Errorf("error: not all resources were initialized - did you run server.Init()?")
	}
	return nil
}
