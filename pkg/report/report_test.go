package report

import (
	"strings"
	"testing"

	"github.com/pleimer/ci-server-go/pkg/assert"
)

func TestReport(t *testing.T) {
	exp := "## Title1\n```\ncodeblock\n```\n## Title2\nbodytext\n"
	t.Run("2 section build", func(t *testing.T) {
		sb := &strings.Builder{}
		rep := NewWriter(sb)
		rep.AddTitle("Title1")
		rep.OpenBlock()
		rep.Write("codeblock")
		rep.CloseBlock()
		rep.AddTitle("Title2")
		rep.Write("bodytext")
		rep.Flush()
		assert.Ok(t, rep.Err())
		assert.Equals(t, exp, sb.String())
	})

	t.Run("close block before open", func(t *testing.T) {
		sb := &strings.Builder{}
		rep := NewWriter(sb)
		rep.CloseBlock()
		assert.Equals(t, rep.Err(), ErrBlockNotOpen)
	})

	t.Run("write title in block", func(t *testing.T) {
		sb := &strings.Builder{}
		rep := NewWriter(sb)
		rep.OpenBlock()
		rep.AddTitle("title")
		assert.Equals(t, rep.Err(), ErrTitleInBlock)
	})
}
