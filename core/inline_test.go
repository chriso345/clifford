package core

import (
	"os"
	"testing"

	"github.com/chriso345/gore/assert"
)

func TestParse_InlinePrimitiveFlagLong(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "--max-items", "5"}

	cli := struct {
		Clifford `name:"myapp"`

		Menu struct {
			Value    string
			Clifford `long:"menu"`
			MaxItems int `short:"n" long:"max-items"`
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.Equal(t, cli.Menu.MaxItems, 5)
}

func TestParse_InlinePrimitiveFlagShort(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "-n", "7"}

	cli := struct {
		Clifford `name:"myapp"`

		Menu struct {
			Value    string
			Clifford `long:"menu"`
			MaxItems int `short:"n" long:"max-items"`
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.Equal(t, cli.Menu.MaxItems, 7)
}

func TestParse_InlinePrimitiveDefault(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd"}

	cli := struct {
		Clifford `name:"myapp"`

		Menu struct {
			Value    string
			Clifford `long:"menu"`
			MaxItems int `default:"3"` // no explicit flags
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.Equal(t, cli.Menu.MaxItems, 3)
}

func TestParse_InlineBoolFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"cmd", "--dry-run"}

	cli := struct {
		Clifford `name:"myapp"`

		Menu struct {
			Value    string
			Clifford `long:"menu"`
			DryRun   bool `long:"dry-run"`
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.Equal(t, cli.Menu.DryRun, true)
}

func TestParse_SubcommandWithInlineFlags(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"app", "menu", "--max-items", "9", "--dry-run"}

	cli := struct {
		Clifford `name:"app"`
		Help

		Menu struct {
			Subcommand
			Clifford `name:"menu" long:"menu" desc:"Start in menu mode"`
			MaxItems int  `short:"n" long:"max-items"`
			DryRun   bool `long:"dry-run"`
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.True(t, bool(cli.Menu.Subcommand))
	assert.Equal(t, cli.Menu.MaxItems, 9)
	assert.Equal(t, cli.Menu.DryRun, true)
}
