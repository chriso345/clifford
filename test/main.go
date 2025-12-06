package main

import (
	"fmt"
	"os"

	"github.com/chriso345/clifford"
)

type CLIArgs struct {
	clifford.Clifford `name:"app"`
	clifford.Help
	clifford.Version `version:"0.1.0"`
	clifford.Desc    `desc:"An example application demonstrating clifford features"`

	Serve struct {
		clifford.Subcommand `name:"server"`
		clifford.Help
		clifford.Desc `desc:"Start the server"`

		Port struct {
			Value             int `default:"8080"`
			clifford.Clifford `long:"port"`
			clifford.Desc     `desc:"Port to run the server on"`
		}

		Verbose struct {
			Value             bool
			clifford.Clifford `short:"v" long:"verbose" desc:"Enable verbose output"`
		}
	}
}

func main() {
	args := &CLIArgs{}

	if err := clifford.Parse(args); err != nil {
		fmt.Fprintln(os.Stderr, "Error parsing arguments:", err)
		os.Exit(1)
	}

	fmt.Printf("Parsed Arguments: %+v\n", args)
}
