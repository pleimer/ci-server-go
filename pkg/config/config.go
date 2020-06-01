package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	User    string
	Oauth   string
	Address string
	Workers int
}

func NewConfig() (*Config, error) {
	var err error
	c := &Config{
		User:    os.Getenv("GITHUB_USER"),
		Oauth:   os.Getenv("OAUTH"),
		Address: os.Getenv("ADDRESS"),
	}

	if c.User == "" {
		return nil, fmt.Errorf("organization not specified by GITHUB_USER environmental variable")
	}

	if c.Oauth == "" {
		return nil, fmt.Errorf("oauth token not specified by OAUTH environemental variable")
	}

	if c.Address == "" {
		c.Address = ":3000"
	}

	numWorkers := os.Getenv("NUM_WORKERS")
	c.Workers = 4
	if numWorkers != "" {
		c.Workers, err = strconv.Atoi(numWorkers)
		if err != nil {
			return nil, fmt.Errorf("invalid data type in NUM_WORKERS env variable: %s", err)
		}
	}

	return c, nil
}
