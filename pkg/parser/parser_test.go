package parser

import (
	"bytes"
	"context"
	"testing"

	"github.com/infrawatch/ci-server-go/pkg/assert"
	"gopkg.in/yaml.v2"
)

func TestScriptCmd(t *testing.T) {
	spec := &Spec{
		Global: &Global{
			Timeout: 300,
			Env: map[string]interface{}{
				"OCP_PROJECT": "stf",
			},
		},
		Script: []string{
			"echo $OCP_PROJECT",
		},
		AfterScript: []string{
			"echo Done",
		},
	}
	ciyaml, _ := yaml.Marshal(spec)
	in := bytes.NewBuffer(ciyaml)
	specUT, err := NewSpecFromYAML(in)
	assert.Ok(t, err)

	out, err := specUT.ScriptCmd(context.Background(), "").Output()
	assert.Ok(t, err)

	assert.Equals(t, "stf\n", string(out))
}
