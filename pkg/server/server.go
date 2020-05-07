package server

import (
	"fmt"
	"strings"

	"github.com/infrawatch/ci-server-go/pkg/config"
	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/job"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

// Run server main function
func Run() {
	log, err := logging.NewLogger(logging.INFO, "console")
	if err != nil {
		log.Error(err.Error())
		return
	}
	defer log.Destroy()

	config, err := config.NewConfig()
	if err != nil {
		log.Error(err.Error())
		return
	}

	evChan := make(chan ghclient.Event, 4)
	errChan := make(chan error)

	gh := ghclient.NewClient(evChan, errChan)
	err = gh.Api.Authenticate(strings.NewReader(config.Oauth))
	if err != nil {
		log.Error(err.Error())
		return
	}

	log.Info(fmt.Sprintf("Listening on %s for webhooks", config.Proxy))
	go gh.Listen(config.Proxy)

	select {
	case ev := <-evChan:
		// run job based on event type
		j, err := job.Factory(ev, &gh)
		if err != nil {
			log.Error(err.Error())
			break
		}
		j.SetLogger(log)
		go j.Run()
	case err := <-errChan:
		log.Error(fmt.Sprintf("%v", err))
	}
}
