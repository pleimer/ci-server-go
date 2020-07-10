package config

import (
	"fmt"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type Section struct {
	Options map[string]interface{}
}

//Config ..
type Config struct {
	sections map[string]*Section
	metadata map[string][]Parameter
}

func (c *Config) GetUser() string {
	return c.sections["github"].Options["user"].(string)
}

func (c *Config) GetOauth() string {
	return c.sections["github"].Options["oauth"].(string)
}

func (c *Config) GetAddress() string {
	return c.sections["listener"].Options["address"].(string)
}

func (c *Config) GetNumWorkers() int {
	return c.sections["runner"].Options["numWorkers"].(int)
}

func (c *Config) GetAuthorizedUsers() []string {
	return c.sections["runner"].Options["authorizedUsers"].([]string)
}

/*******************************************************************/
type validator func(interface{}) error

func intValidatorFactory() validator {
	return func(input interface{}) error {
		if _, ok := input.(int); !ok {
			return fmt.Errorf("expected type 'int', got '%T'", input)
		}
		return nil
	}
}

func stringValidatorFactory() validator {
	return func(input interface{}) error {
		if _, ok := input.(string); !ok {
			return fmt.Errorf("expected type 'string', got '%T'", input)
		}
		return nil
	}
}

func stringSliceValidatorFactory() validator {
	return func(input interface{}) error {
		if _, ok := input.([]interface{}); !ok {
			return fmt.Errorf("expected list, got '%T'", input)
		}
		for _, item := range input.([]interface{}) {
			if _, ok := item.(string); !ok {
				return fmt.Errorf("expected entry should be of type 'string', got '%T'", item)
			}
		}
		return nil
	}
}

/*******************************************************************/

// Parameter ..
type Parameter struct {
	Name       string
	Default    interface{}
	Required   bool
	Validators []validator
}

func (p *Parameter) validate(value interface{}, sectionName string) error {
	if value == nil && p.Required {
		return fmt.Errorf("'%s.%s' parameter required but not specified", sectionName, p.Name)
	}
	if value == nil && !p.Required {
		return nil
	}

	for _, validator := range p.Validators {
		err := validator(value)
		if err != nil {
			return errors.Wrapf(err, "invalid value '%v'", value)
		}
	}
	return nil
}

func getConfigMetadata() map[string][]Parameter {
	return map[string][]Parameter{
		"github": {
			{"user", "", true, []validator{stringValidatorFactory()}},
			{"oauth", "", true, []validator{stringValidatorFactory()}},
		},
		"listener": {
			{"address", ":3000", false, []validator{stringValidatorFactory()}},
		},
		"runner": {
			{"numWorkers", 4, false, []validator{intValidatorFactory()}},
			{"authorizedUsers", nil, false, []validator{stringSliceValidatorFactory()}},
		},
	}
}

func NewConfig() *Config {
	config := &Config{
		metadata: getConfigMetadata(),
		sections: map[string]*Section{},
	}
	return config
}

func (c *Config) Parse(r io.Reader) error {
	c.sections = map[string]*Section{}

	configMap := make(map[string]map[string]interface{})
	configBytes, err := ioutil.ReadAll(r)
	if err != nil {
		return errors.Wrap(err, "while reading config file")
	}
	err = yaml.Unmarshal(configBytes, configMap)
	if err != nil {
		return errors.Wrap(err, "while parsing yaml")
	}

	for sectionName, parameters := range c.metadata {
		c.sections[sectionName] = &Section{}
		c.sections[sectionName].Options = map[string]interface{}{}
		for _, param := range parameters {
			c.sections[sectionName].Options[param.Name] = param.Default

			option := configMap[sectionName][param.Name]
			err := param.validate(option, sectionName)
			if err != nil {
				return errors.Wrapf(err, "failed to validate parameter '%s'", param.Name)
			}
			if option != nil {
				c.sections[sectionName].Options[param.Name] = option
			}
		}
	}

	return nil
}
