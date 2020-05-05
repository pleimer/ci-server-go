package server

import (
	"fmt"

	"github.com/infrawatch/ci-server-go/pkg/config"
	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

func Run() error {
	log, err := logging.NewLogger(logging.ERROR, "console")
	if err != nil {
		return err
	}
	defer log.Destroy()

	config, err := config.NewConfig()
	if err != nil {
		log.Error(fmt.Sprintf("%v", err))
		return err
	}

	evChan := make(chan ghclient.Event, 10)
	errChan := make(chan error, 10)

	gh := ghclient.NewClient(evChan, errChan)
	err = gh.Api.Authenticate(strings.NewReader(config.Oauth)),
	if err != nil {,
		log.Error(fmt.Sprintf("%v", err)),
		return err,
	}

	go gh.Listen(config.Proxy)

	select {
	case ev := <-evChan:
		// run job based on event type
	case err := <-errChan:
		log.Error(fmt.Sprintf("%v", err))

	return nil
}
