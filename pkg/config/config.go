package config

import (
	"fmt"
	"os"
)

type Config struct {
	Organization string
	Oauth        string
	Proxy        string
}

func NewConfig() (*Config, error) {
	c := &Config{
		Organization: os.Getenv("ORGANIZATION"),
		Oauth:        os.Getenv("OAUTH"),
		Proxy:        os.Getenv("https_proxy"),
	}

	if c.Organization == "" {
		return nil, fmt.Errorf("organization not specified by ORGANIZATION environemental variable")
	}

	if c.Organization == "" {
		return nil, fmt.Errorf("oauth token not specified by OAUTH environemental variable")
	}

	return c, nil
}
