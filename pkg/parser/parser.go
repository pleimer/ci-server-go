package parser

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"gopkg.in/yaml.v2"
)

type ParserError struct {
	msg string
	err error
}

func (pe *ParserError) Error() string {
	return fmt.Sprintf("parser: %s: %s", pe.msg, pe.err)
}

type Global struct {
	Timeout int                    `yaml:"timeout"`
	Env     map[string]interface{} `yaml:"env"`
}
type Spec struct {
	Global      *Global  `yaml:"global"`
	Script      []string `yaml:"script"`
	AfterScript []string `yaml:"after_script"`

	metaVars map[string]string
}

func (s *Spec) SetMetaVar(key, val string) {
	s.metaVars[key] = val
}

func (s *Spec) ScriptCmd(ctx context.Context, basePath string) *exec.Cmd {
	cmd := s.genEnv(ctx, s.Script)
	cmd.Dir = basePath
	return cmd
}

func (s *Spec) AfterScriptCmd(ctx context.Context, basePath string) *exec.Cmd {
	cmd := s.genEnv(ctx, s.AfterScript)
	cmd.Dir = basePath
	return cmd
}

func (s *Spec) genEnv(ctx context.Context, comList []string) *exec.Cmd {
	cmdString := strings.Join(comList, ";")
	cmd := exec.CommandContext(ctx, "bash", "-ce", cmdString)
	var newEnv []string
	for key, val := range s.Global.Env {
		switch v := val.(type) {
		case int:
			newEnv = append(newEnv, key+"="+strconv.Itoa(v))
		case string:
			if meta, ok := s.metaVars[v]; ok {
				newEnv = append(newEnv, key+"="+meta)
			} else {
				newEnv = append(newEnv, key+"="+v)
			}
		}
	}

	cmd.Env = append(os.Environ(), newEnv...)
	return cmd
}

func NewSpecFromYAML(yamlSpec io.Reader) (*Spec, error) {
	var spec Spec
	res, err := ioutil.ReadAll(yamlSpec)
	if err != nil {
		return nil, &ParserError{msg: "failed unmarshalling yaml spec", err: err}
	}
	err = yaml.Unmarshal(res, &spec)
	if err != nil {
		return nil, &ParserError{msg: "failed unmarshalling yaml spec", err: err}
	}

	if spec.Global == nil {
		spec.Global = &Global{}
	}

	if spec.Global.Timeout == 0 {
		spec.Global.Timeout = 300
	}
	spec.metaVars = make(map[string]string)
	return &spec, nil
}
