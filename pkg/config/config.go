package config

import (
	"fmt"
	"os"
)

type Config struct {
	User          string
	Oauth         string
	Proxy         string
	ListenAddress string
}

func NewConfig() (*Config, error) {
	c := &Config{
		User:  os.Getenv("GITHUB_USER"),
		Oauth: os.Getenv("OAUTH"),
		Proxy: os.Getenv("https_proxy"),
	}

	if c.User == "" {
		return nil, fmt.Errorf("organization not specified by GITHUB_USER environmental variable")
	}

	if c.Oauth == "" {
		return nil, fmt.Errorf("oauth token not specified by OAUTH environemental variable")
	}

	c.ListenAddress = ":3000"
	if c.Proxy != "" {
		c.ListenAddress = c.Proxy
	}

	return c, nil
}
