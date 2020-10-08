package config

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/go-playground/validator.v9"
	"gopkg.in/yaml.v2"
)

//Config holds foundational configurations for the ci engine.
type Config struct {
	Github struct {
		User  string `yaml:"user" validate:"required"`
		Oauth string `yaml:"oauth" validate:"required"`
	} `yaml:"github" validate:"required"`

	Listener struct {
		Address string `yaml:"address" validate:"required"`
	} `yaml:"listener" validate:"required"`

	Logger struct {
		Level  string `yaml:"level" validate:"required"`
		Target string `yaml:"target" validate:"required"`
	} `yaml:"logger" validate:"required"`

	Runner struct {
		NumWorkers      int      `yaml:"numWorkers" validate:"required"`
		AuthorizedUsers []string `yaml:"authorizedUsers" validate:"required"`
	} `yaml:"runner" validate:"required"`
}

// New generate config object with defaults
func New() *Config {
	return &Config{
		Listener: struct {
			Address string `yaml:"address" validate:"required"`
		}{
			Address: ":3000",
		},
		Logger: struct {
			Level  string `yaml:"level" validate:"required"`
			Target string `yaml:"target" validate:"required"`
		}{
			Level:  "INFO",
			Target: "console",
		},
		Runner: struct {
			NumWorkers      int      `yaml:"numWorkers" validate:"required"`
			AuthorizedUsers []string `yaml:"authorizedUsers" validate:"required"`
		}{
			NumWorkers: 4,
		},
	}
}

//Parse parse yaml from reader
func (c *Config) Parse(r io.Reader) error {
	validate := validator.New()

	configBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "while reading config file")
	}

	err = yaml.Unmarshal(configBytes, c)
	if err != nil {
		return errors.Wrap(err, "while parsing yaml")
	}

	err = validate.Struct(c)
	if err != nil {
		if e, ok := err.(validator.ValidationErrors); ok {
			missingFields := []string{}
			for _, fe := range e {
				missingFields = append(missingFields, setCamelCase(fe.Namespace()))
			}
			return fmt.Errorf("missing fields in config: (%s)", strings.Join(missingFields, " , "))
		}
		return errors.Wrap(err, "error while validating configuration")
	}

	return nil
}

func setCamelCase(field string) string {
	items := strings.Split(field, ".")
	ret := []string{}
	for _, item := range items {
		camel := []byte(item)
		l := bytes.ToLower([]byte{camel[0]})
		camel[0] = l[0]
		ret = append(ret, string(camel))
	}
	return strings.Join(ret, ".")
}
