package core

import (
	stderrs "errors"
	"os"
	"testing"

	clierr "github.com/chriso345/clifford/errors"
	"github.com/chriso345/gore/assert"
)

func TestParse_ShortAndLongFlags(t *testing.T) {
	// Save original args and restore later
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "--name", "Alice", "-a", "30"}

	cli := struct {
		Clifford `name:"mytool"`

		Name struct {
			Value    string
			Clifford `short:"n" long:"name" desc:"User name"`
		}

		Age struct {
			Value string
			ShortTag
			LongTag
			Desc `desc:"Age of user"`
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.Equal(t, "Alice", cli.Name.Value)
	assert.Equal(t, "30", cli.Age.Value)
}

func TestParse_PositionalArgs(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "Alice", "30"}

	cli := struct {
		Clifford `name:"mytool"`

		Name struct {
			Value string
			Required
		}
		Age struct {
			Value string
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.Equal(t, "Alice", cli.Name.Value)
	assert.Equal(t, "30", cli.Age.Value)
}

func TestParse_DebugPositionalArgs(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"junk", "junk", "--", "Alice", "30"}

	cli := struct {
		Clifford `name:"mytool"`

		Name struct {
			Value string
			Required
		}
		Age struct {
			Value string
		}
	}{}

	err := Parse(&cli)
	assert.Nil(t, err)
	assert.Equal(t, cli.Name.Value, "Alice")
	assert.Equal(t, cli.Age.Value, "30")
}

func TestParse_MissingRequired(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "--age", "30"}

	cli := struct {
		Clifford `name:"mytool"`

		Name struct {
			Value string
			Required
		}
		Age struct {
			Value string
			LongTag
		}
	}{}

	err := Parse(&cli)
	assert.NotNil(t, err)
	var me clierr.MissingArgError
	ok := stderrs.As(err, &me)
	assert.True(t, ok)
	assert.Equal(t, me.Field, "Name")
}

func TestParse_UnsupportedFieldType(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "--opt", "v"}

	target := struct {
		Clifford `name:"myapp"`
		Opt      struct {
			Value    []string
			Clifford `long:"opt"`
		}
	}{}

	err := Parse(&target)
	assert.NotNil(t, err)
	var ue clierr.UnsupportedFieldTypeError
	ok := stderrs.As(err, &ue)
	assert.True(t, ok)
	assert.Equal(t, ue.Field, "Opt")
	assert.StringContains(t, err.Error(), "slice")
}

func TestParse_InvalidTarget(t *testing.T) {
	// Passing a non-struct pointer/value should return a ParseError
	err := Parse(123)
	assert.NotNil(t, err)
	var pe clierr.ParseError
	ok := stderrs.As(err, &pe)
	assert.True(t, ok)
}

func TestParse_HelpFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "--help"}

	cli := struct {
		Clifford `name:"mytool"`
		Help
		Name struct {
			Value string
			Required
		}
	}{}

	// Temporarily override os.Exit
	calledExit := false
	osExit = func(code int) {
		calledExit = true
		panic("os.Exit called")
	}
	defer func() { osExit = os.Exit }()

	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, true, calledExit)
		}
	}()

	_ = Parse(&cli)
	t.Errorf("should have exited before this line")
}

func TestParse_VersionFlag(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"cmd", "--version"}

	cli := struct {
		Clifford `name:"mytool"`
		Version  `version:"1.2.3"`
	}{}

	calledExit := false
	osExit = func(code int) {
		calledExit = true
		panic("os.Exit called")
	}
	defer func() { osExit = os.Exit }()

	defer func() {
		if r := recover(); r != nil {
			assert.Equal(t, true, calledExit)
		}
	}()

	_ = Parse(&cli)
	t.Errorf("should have exited before this line")
}

func TestParse_UnknownSubcommand(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()

	os.Args = []string{"app", "srve"} // typo for 'serve'

	target := struct {
		Clifford `name:"app"`
		Serve    struct {
			Subcommand
		}
	}{}

	err := Parse(&target)
	assert.NotNil(t, err)
	var ue clierr.UnknownSubcommandError
	ok := stderrs.As(err, &ue)
	assert.True(t, ok)
	// suggestion should be present when typo is close
	assert.StringContains(t, err.Error(), "did you mean")
}
