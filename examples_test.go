package clifford_test

import (
	"fmt"
	"os"
	"regexp"

	stderrors "errors"

	"github.com/chriso345/clifford"
	"github.com/chriso345/clifford/errors"
)

func Example_readme() {
	// Simulate command line arguments
	os.Args = []string{"mytool", "--name", "Alice", "-a", "30"}

	target := struct {
		clifford.Clifford `name:"mytool"`   // Set the name of the CLI tool
		clifford.Version  `version:"1.2.3"` // Enable automatic version flag
		clifford.Help                       // Enable automatic help flags

		Name struct {
			Value             string
			clifford.Clifford `short:"n" long:"name" desc:"User name"`
		}
		Age struct {
			Value             string
			clifford.ShortTag // auto generates -a
			clifford.LongTag  // auto generates --age
			clifford.Desc     `desc:"Age of the user"`
		}
	}{}

	err := clifford.Parse(&target)
	if err != nil {
		panic(err)
	}

	fmt.Printf("Name: %s\n", target.Name.Value)
	fmt.Printf("Age: %s\n", target.Age.Value)
	// Output: Name: Alice
	// Age: 30
}

func Example_simple_cli() {
	// Simulate command line arguments
	os.Args = []string{"example_cli", "bob"}

	target := struct {
		clifford.Clifford `name:"example_cli"` // This is the name of the cli command
		clifford.Version  `version:"1.0.0"`
		clifford.Help

		Name struct {
			Value string
		}
	}{}

	err := clifford.Parse(&target)
	if err != nil {
		panic(err)
	}

	if target.Name.Value != "" {
		fmt.Println("Hello, " + target.Name.Value + "!")
	} else {
		fmt.Println("Hello, World!")
	}
	// Output: Hello, bob!
}

func Example_flag() {
	target := struct {
		clifford.Clifford `name:"sampleapp"`

		Flag struct {
			Value             bool
			clifford.Clifford `short:"f" long:"flag" desc:"A sample boolean flag"`
		}
	}{}

	// Simulate command line arguments
	os.Args = []string{"sampleapp", "-f"}

	err := clifford.Parse(&target)
	if err != nil {
		panic(err)
	}
	fmt.Println("Flag value:", target.Flag.Value)
	// Output: Flag value: true
}

func Example_subcommand() {
	// Demonstrate a single-level subcommand parse
	os.Args = []string{"app", "serve", "--port", "8080"}

	target := struct {
		clifford.Clifford `name:"app"`
		clifford.Help

		Serve struct {
			clifford.Subcommand `name:"serve" desc:"Start the server"`
			Port                struct {
				Value             int
				clifford.Clifford `long:"port" desc:"Port to run the server on"`
			}
		}

		Status struct {
			clifford.Subcommand `name:"status" desc:"Show server status"`
		}
	}{}

	err := clifford.Parse(&target)
	if err != nil {
		panic(err)
	}

	fmt.Println("Serve port:", target.Serve.Port.Value)
	// Output: Serve port: 8080
}

func Example_list_subcommands() {
	// Build help that lists multiple subcommands and print a marker line for the example.
	target := struct {
		clifford.Clifford `name:"app" desc:"Application with multiple subcommands"`
		clifford.Help

		Serve struct {
			clifford.Subcommand `name:"serve" desc:"Start the server"`
		}
		Status struct {
			clifford.Subcommand `name:"status" desc:"Show server status"`
		}
	}{}

	if _, err := clifford.BuildHelp(&target, false); err != nil {
		panic(err)
	}
	fmt.Println("Subcommands:")
	// Output: Subcommands:
}

func Example_disable_shorthand() {
	// Demonstrate disabling -h/-v short forms on the top-level command
	target := struct {
		clifford.Clifford `name:"no-shorts" version:"0.1.0" help_short:"false" version_short:"false"`
		clifford.Help
		clifford.Version
	}{}

	if _, err := clifford.BuildHelp(&target, false); err != nil {
		panic(err)
	}
	fmt.Println("version")
	// Output: version
}

func Example_nested_subcommands() {
	// Demonstrate nested subcommands: app remote add --name foo
	os.Args = []string{"app", "remote", "add", "--name", "origin"}
	target := struct {
		clifford.Clifford `name:"app"`
		clifford.Help

		Remote struct {
			clifford.Subcommand `name:"remote" desc:"Remote operations"`
			Add                 struct {
				clifford.Subcommand `name:"add" desc:"Add a remote"`
				Name                struct {
					Value             string
					clifford.Clifford `long:"name" desc:"Name of the remote"`
				}
			}
		}
	}{}

	err := clifford.Parse(&target)
	if err != nil {
		panic(err)
	}
	fmt.Println("Remote add name:", target.Remote.Add.Name.Value)
	// Output: Remote add name: origin
}

func ExampleBuildVersion() {
	target := struct {
		clifford.Clifford `name:"mycli" version:"2.3.4"`
	}{}

	version, err := clifford.BuildVersion(&target)
	if err != nil {
		panic(err)
	}
	fmt.Println(version)
	// Output: mycli v2.3.4
}

func ExampleBuildHelp() {
	target := struct {
		clifford.Clifford `name:"testapp"`
		clifford.Desc     `desc:"A test application"`

		Foo struct {
			Value             string
			clifford.Clifford `short:"f" long:"foo" desc:"A foo flag"`
		}
	}{}

	help, err := clifford.BuildHelp(&target, false)
	if err != nil {
		panic(err)
	}
	fmt.Println(stripANSI(help))
	// Output: Usage: testapp [OPTIONS]
	//
	// A test application
	//
	// Options:
	//   -f, --foo [FOO]  A foo flag
}

// Example_defaults demonstrates default values are applied when flags are omitted.
func Example_defaults() {
	os.Args = []string{"app"}
	target := struct {
		clifford.Clifford `name:"app" desc:"App showing defaults"`

		Port struct {
			Value             int `default:"8080"`
			clifford.Clifford `long:"port" desc:"Port to run the server on"`
		}
	}{}

	err := clifford.Parse(&target)
	if err != nil {
		panic(err)
	}
	fmt.Println("Port:", target.Port.Value)
	// Output: Port: 8080
}

// Example_help_output demonstrates building help for a parent with multiple subcommands.
func Example_help_output() {
	// Parent help shows subcommands and their descriptions
	target := struct {
		clifford.Clifford `name:"app" desc:"App for help output"`
		clifford.Help

		Serve struct {
			clifford.Subcommand `name:"serve" desc:"Start the server"`
		}
		Status struct {
			clifford.Subcommand `name:"status" desc:"Show server status"`
		}
	}{}

	help, err := clifford.BuildHelp(&target, false)
	if err != nil {
		panic(err)
	}
	fmt.Println(stripANSI(help))
}

func stripANSI(input string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(input, "")
}

// Example_error_types demonstrates checking for specific error kinds with errors.Is and accessing details with errors.As.
func Example_error_types() {
	os.Args = []string{"app"}
	target := struct {
		clifford.Clifford `name:"app"`

		File struct {
			Value             string
			clifford.Required `required:"true"`
		}
	}{}

	err := clifford.Parse(&target)
	if err == nil {
		fmt.Println("no error")
		return
	}

	if stderrors.Is(err, errors.ErrMissingArg) {
		fmt.Println("missing argument detected")
	}

	var ma errors.MissingArgError
	if stderrors.As(err, &ma) {
		fmt.Println("missing field:", ma.Field)
	}

	// Output:
	// missing field: File
}

// Example_unknown_subcommand shows the parser returning a helpful suggestion for mistyped subcommands.
func Example_unknown_subcommand() {
	os.Args = []string{"app", "srve"}
	target := struct {
		clifford.Clifford `name:"app"`
		clifford.Help

		Serve struct {
			clifford.Subcommand `name:"serve" desc:"Start server"`
		}
	}{}

	err := clifford.Parse(&target)
	if err != nil {
		fmt.Println(err.Error())
	}
	// Output: unknown subcommand: srve (did you mean "serve"?)
}

// Example_help_as_subcommand demonstrates generating help for a subcommand while keeping parent name in usage.
func Example_help_as_subcommand() {
	parent := struct {
		clifford.Clifford `name:"app" desc:"Parent app"`
		clifford.Help

		Serve struct {
			clifford.Subcommand `name:"serve" desc:"Start server"`
			Port                struct {
				Value             int
				clifford.Clifford `long:"port" desc:"Port number"`
			}
		}
	}{}

	sub := parent.Serve
	help, err := clifford.BuildHelpWithParent(&parent, "serve", &sub, false)
	if err != nil {
		panic(err)
	}
	fmt.Println(stripANSI(help))
	// Output: Usage: app serve [OPTIONS]
	//
	// Options:
	//   --port [PORT]  Port number
}
