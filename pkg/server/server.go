package server

import (
	"fmt"

	"github.com/infrawatch/ci-server-go/pkg/config"
	"github.com/infrawatch/ci-server-go/pkg/ghclient"
	"github.com/infrawatch/ci-server-go/pkg/logging"
)

func Run() error {
	evChan := make(chan ghclient.Event, 10)
	errChan := make(chan error, 10)

	log, err := logging.NewLogger(logging.ERROR, "console")
	if err != nil {
		return err
	}

	gh := ghclient.NewClient(evChan, errChan)

	config, err := config.NewConfig()
	if err != nil {
		return err
	}

	go gh.Listen(config.Proxy)

	select {
	case ev := <-evChan:
		log.Info(ev.String())
	case err := <-errChan:
		log.Error(fmt.Sprintf("%v", err))
	default:
		t.Errorf("Did not receive event or error")
	}
}
