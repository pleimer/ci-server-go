package parser

import (
	"fmt"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type Spec struct {
	Global struct {
		Timeout int                    `yaml:"timeout"`
		Env     map[string]interface{} `yaml:"env"`
	}
	Script      []string `yaml:"script"`
	AfterScript []string `yaml:"after_script"`
}

func NewSpecFromYAML(yamlSpec []byte) (*Spec, error) {
	var spec *Spec
	err := yaml.Unmarshal(yamlSpec, spec)
	if err != nil {
		return nil, fmt.Errorf("when unmarshalling yaml spec: %s", err)
	}
	return spec, nil
}

func NewSpecFromFile(path string) (*Spec, error) {
	data, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("while reading file %s: %s", path, err)
	}

	return NewSpecFromYAML(data)
}
