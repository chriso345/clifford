package clifford_test

import (
	"fmt"
	"os"
	"regexp"

	"github.com/chriso345/clifford"
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
			clifford.Subcommand
			Port struct {
				Value             int
				clifford.Clifford `long:"port"`
			}
		} `subcmd:"serve"`
	}{}

	err := clifford.Parse(&target)
	if err != nil {
		panic(err)
	}

	fmt.Println("Serve port:", target.Serve.Port.Value)
	// Output: Serve port: 8080
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
	// Options:
	//   -f, --foo [FOO]  A foo flag
}

func stripANSI(input string) string {
	re := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return re.ReplaceAllString(input, "")
}
