package core

import (
	"os"
	"strings"
	"testing"
)

func TestPositionalSubcommandHelpExits(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"app", "run", "help"}

	target := struct {
		Clifford `name:"app"`
		Help     `type:"subcmd"`

		Run struct {
			Subcommand
			Desc `desc:"Run a specific file"`

			File struct {
				Value string
				Required
				Desc `desc:"File to run"`
			}
		}
	}{}

	// override osExit
	oldExit := osExit
	defer func() { osExit = oldExit }()
	exited := false
	osExit = func(code int) { exited = true; panic("os.Exit") }

	// capture stdout
	r, w, _ := os.Pipe()
	oldOut := os.Stdout
	os.Stdout = w
	defer func() { if err := w.Close(); err != nil { t.Fatalf("close pipe: %v", err) }; os.Stdout = oldOut }()

	defer func() {
		os.Stdout = oldOut
		if rec := recover(); rec == nil {
			t.Fatalf("expected os.Exit panic")
		}
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		out := string(buf[:n])
		if !exited {
			t.Fatalf("expected osExit to be called")
		}
		if !strings.Contains(out, "Usage:") {
			t.Fatalf("help output missing; got: %q", out)
		}
	}()

	_ = Parse(&target)
}
