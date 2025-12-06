package display_test

import (
	"strings"
	"testing"

	"github.com/chriso345/gore/assert"

	"github.com/chriso345/clifford"
)

func TestBuildHelp_ValidInput(t *testing.T) {
	target := struct {
		clifford.Clifford `name:"mytool"`

		Input struct {
			Value string
			clifford.Required
			clifford.Desc `desc:"The input file"`
		}

		Verbose struct {
			Value             bool
			clifford.Clifford `short:"v" long:"verbose" desc:"Enable verbose output"`
		}
	}{}

	help, err := clifford.BuildHelp(&target, false)
	assert.Nil(t, err)
	assert.StringContains(t, help, "Usage:")
	assert.StringContains(t, help, "[INPUT]")
	assert.StringContains(t, help, "-v, --verbose [VERBOSE]")
	assert.StringContains(t, help, "The input file")
	assert.StringContains(t, help, "Enable verbose output")
}

func TestBuildHelp_MissingNameTag(t *testing.T) {
	target := struct {
		clifford.Clifford

		Foo struct {
			Value string
			clifford.Required
		}
	}{}

	_, err := clifford.BuildHelp(&target, false)
	assert.Nil(t, err)
}

func TestBuildHelp_NoOptionsOrArgs(t *testing.T) {
	target := struct {
		clifford.Clifford `name:"emptytool"`
	}{}

	help, err := clifford.BuildHelp(&target, false)
	assert.Nil(t, err)
	assert.StringContains(t, help, "Usage: emptytool")
	assert.NotStringContains(t, help, "Arguments:")
	assert.NotStringContains(t, help, "Options:")
}

func TestBuildHelp_VersionAndHelp(t *testing.T) {
	target := struct {
		clifford.Clifford `name:"tool"`

		clifford.Version
		clifford.Help
	}{}

	help, err := clifford.BuildHelp(&target, false)
	assert.Nil(t, err)
	assert.StringContains(t, help, "--version")
	assert.StringContains(t, help, "--help")
}

func TestOptionsAlignment(t *testing.T) {
	target := struct {
		clifford.Clifford `name:"alignmenttool"`

		Config struct {
			Value             string
			clifford.Clifford `short:"c" long:"config" desc:"Path to config file"`
		}
		Debug struct {
			Value             bool
			clifford.Clifford `short:"d" long:"debug" desc:"Enable debug mode"`
		}
	}{}

	help, err := clifford.BuildHelp(&target, false)
	assert.Nil(t, err)

	lines := strings.Split(help, "\n")
	optionLines := filterLinesContaining(lines, "--config", "--debug")
	assert.Equal(t, len(optionLines), 2)
	pIndex := strings.Index(optionLines[0], "Path")
	eIndex := strings.Index(optionLines[1], "Enable")
	assert.True(t, pIndex == eIndex)
}

func filterLinesContaining(lines []string, terms ...string) []string {
	var out []string
	for _, line := range lines {
		for _, term := range terms {
			if strings.Contains(line, term) {
				out = append(out, line)
				break
			}
		}
	}
	return out
}
