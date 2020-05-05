package parser

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type Global struct {
	Timeout int                    `yaml:"timeout"`
	Env     map[string]interface{} `yaml:"env"`
}
type Spec struct {
	Global      *Global  `yaml:"global"`
	Script      []string `yaml:"script"`
	AfterScript []string `yaml:"after_script"`
}

func (s *Spec) ScriptCmd() *exec.Cmd {
	return s.genEnv(s.Script)
}

func (s *Spec) AfterScriptCmd() *exec.Cmd {
	return s.genEnv(s.AfterScript)
}

func (s *Spec) genEnv(comList []string) *exec.Cmd {
	cmdString := strings.Join(s.Script, ";")
	cmd := exec.Command("bash", "-c", cmdString)
	var newEnv []string
	for key, val := range s.Global.Env {
		switch v := val.(type) {
		case int:
			newEnv = append(newEnv, key+"="+strconv.Itoa(v))
		case string:
			newEnv = append(newEnv, key+"="+v)

		}
	}
	cmd.Env = append(os.Environ(), newEnv...)
	return cmd
}

func NewSpecFromYAML(yamlSpec io.Reader) (*Spec, error) {
	var spec Spec
	res, err := ioutil.ReadAll(yamlSpec)
	if err != nil {
		return nil, fmt.Errorf("when unmarshalling yaml spec: %s", err)
	}
	err = yaml.Unmarshal(res, &spec)
	if err != nil {
		return nil, fmt.Errorf("when unmarshalling yaml spec: %s", err)
	}
	return &spec, nil
}
