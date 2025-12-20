package core

import (
	"os"
	"strings"
	"testing"
)

// Test that flag-style help (--help) exits when Help is present as default (flag)
func TestFlagHelpExits(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"app", "--help"}

	target := struct {
		Clifford `name:"app"`
		Help
	}{}

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

// Test that when Help is type:"subcmd" the flag --help does not exit (i.e. is unknown)
func TestFlagHelpNotAllowedForSubcmd(t *testing.T) {
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"app", "serve", "--help"}

	target := struct {
		Clifford `name:"app"`
		Help     `type:"subcmd"`

		Serve struct {
			Subcommand `name:"serve"`
		}
	}{}

	err := Parse(&target)
	if err == nil {
		t.Fatalf("expected parse error when using --help but help is subcmd only")
	}
}

// Test that type:"both" allows both positional "help" and flag "--help" to exit
func TestBothHelpModesExit(t *testing.T) {
	// flag form
	oldArgs := os.Args
	defer func() { os.Args = oldArgs }()
	os.Args = []string{"app", "--help"}

	target := struct {
		Clifford `name:"app"`
		Help     `type:"both"`
	}{}

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
			t.Fatalf("expected os.Exit panic for flag form")
		}
		buf := make([]byte, 4096)
		n, _ := r.Read(buf)
		out := string(buf[:n])
		if !exited {
			t.Fatalf("expected osExit to be called for flag form")
		}
		if !strings.Contains(out, "Usage:") {
			t.Fatalf("help output missing; got: %q", out)
		}
	}()

	_ = Parse(&target)

	// positional form
	oldArgs2 := os.Args
	defer func() { os.Args = oldArgs2 }()
	os.Args = []string{"app", "help"}

	oldExit = osExit
	defer func() { osExit = oldExit }()
	exited = false
	osExit = func(code int) { exited = true; panic("os.Exit") }

	// capture stdout
	r2, w2, _ := os.Pipe()
	oldOut2 := os.Stdout
	os.Stdout = w2
	defer func() { if err := w2.Close(); err != nil { t.Fatalf("close pipe: %v", err) }; os.Stdout = oldOut2 }()

	defer func() {
		os.Stdout = oldOut2
		if rec := recover(); rec == nil {
			t.Fatalf("expected os.Exit panic for positional form")
		}
		buf := make([]byte, 4096)
		n, _ := r2.Read(buf)
		out := string(buf[:n])
		if !exited {
			t.Fatalf("expected osExit to be called for positional form")
		}
		if !strings.Contains(out, "Usage:") {
			t.Fatalf("help output missing; got: %q", out)
		}
	}()

	_ = Parse(&target)
}
