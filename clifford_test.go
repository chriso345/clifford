package clifford_test

import (
	"os"
	"testing"

	"github.com/chriso345/clifford"
	"github.com/chriso345/gore/assert"
	"github.com/chriso345/gore/vital"
)

func TestBuildVersion(t *testing.T) {
	target := struct {
		clifford.Clifford `name:"mycli" version:"2.3.4"`
	}{}

	version, err := clifford.BuildVersion(&target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := "mycli v2.3.4"
	assert.Equal(t, version, expected)
}

func TestBuildHelp_Basic(t *testing.T) {
	target := struct {
		clifford.Clifford `name:"testapp"`

		Foo struct {
			Value             string
			clifford.Clifford `short:"f" long:"foo" desc:"A foo flag"`
		}
	}{}

	help, err := clifford.BuildHelp(&target, false)
	vital.Nil(t, err)
	assert.StringContains(t, help, "testapp")
	assert.StringContains(t, help, "-f, --foo [FOO]")
}

func TestParse_PositionalAndFlags(t *testing.T) {
	// Simulate CLI args
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"mycmd", "input.txt", "--verbose", "true"}

	target := struct {
		clifford.Clifford `name:"mycmd"`

		Input struct {
			Value string
		}

		Verbose struct {
			Value             bool
			clifford.Clifford `long:"verbose"`
		}
	}{}

	err := clifford.Parse(&target)
	vital.Nil(t, err)
	assert.Equal(t, target.Input.Value, "input.txt")
	assert.True(t, target.Verbose.Value)
}

func TestParse_SubcommandDispatch(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"app", "serve", "--port", "9000"}

	target := struct {
		clifford.Clifford `name:"app"`
		clifford.Help

		Serve struct {
			clifford.Subcommand `name:"serve"`
			Port                struct {
				Value             int
				clifford.Clifford `long:"port"`
			}
		}

		Other struct {
			clifford.Subcommand `name:"other"`
			Flag                struct {
				Value             bool
				clifford.Clifford `short:"o"`
			}
		}
	}{}

	err := clifford.Parse(&target)
	vital.Nil(t, err)
	assert.Equal(t, target.Serve.Port.Value, 9000)
}
