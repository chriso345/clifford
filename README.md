# clifford

> [!WARNING]
> `clifford` is still under early development. Expect breaking changes and incomplete features.

Clifford is a simple and lightweight library for building command-line interfaces in Go. It makes it easy to define flags, arguments, and commands without minimal boilerplate.

---

## Features

- Define positional arguments, short flags (`-f`), and long flags (`--flag`) via struct embedding and tags.
- Support for required and optional arguments.
- Automatic generation of help messages (`--help`) and version information (`--version`).
- Extensible through embedding and custom struct types.

---

## Installation

`clifford` is available on GitHub and can be installed using Go modules:

```bash
go get github.com/chriso345/clifford
```

## Usage

Define your CLI argument structure using embedded `clifford.Clifford` and marker types for flags and descriptions:

```go
package main

import (
	"fmt"
	"log"

	"github.com/chriso345/clifford"
)

func main() {
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
		log.Fatal(err)
	}

	fmt.Printf("Name: %s\n", target.Name.Value)
	fmt.Printf("Age: %s\n", target.Age.Value)
}
```
- Passing `-h` or `--help` will print an automatically generated help message and exit.
- Passing `--version` will print the version information and exit.

If a user mistypes a subcommand, clifford will return a helpful message with a suggested correction:

```text
$ app srve
unknown subcommand: srve (did you mean "serve"?)
```

This suggestion is based on fuzzy matching and common transposition errors.

Notes:
- Use the `default` tag on a field to provide a fallback value which will also be shown in help output.
- Help can be exposed as a dedicated subcommand (e.g. `app help server`) when the `Help` embedding is tagged appropriately; subcommands may advertise that they accept `help` as a subcommand or flag.
- For subcommands, running `app <subcmd> help` or `app <subcmd> -h` will show help scoped to that subcommand when enabled.

## Public API

The public API of `clifford` is still under development. The following types and functions are available:

- `clifford.Parse(target any) error`: Parses the command-line arguments and populates the target struct.
- `clifford.BuildHelp(target any) string`: Generates a help message for the CLI defined by `target`.
- `clifford.BuildVersion(target any) string`: Generates a version message for the CLI defined by `target`.
- `clifford.BuildHelpWithParent(parent any, subName string, subTarget any, long bool) (string, error)`: Helper to generate subcommand help that shows the parent application name alongside the subcommand.

### Public Marker Types

`clifford` provides several marker types to define flags and descriptions in your CLI:
- `clifford.Clifford`: Base type for all CLI definitions. Must be embedded in the target struct. `clifford.Clifford` can also be used as a marker for individual fields.
- `clifford.Help`: Enables automatic help message generation.
- `clifford.Version`: Enables automatic version message generation.
- `clifford.ShortTag`: Marks a field as having a short flag (e.g., `-f`). If no short flag is specified, it defaults to the first letter of the field name.
- `clifford.LongTag`: Marks a field as having a long flag (e.g., `--flag`). If no long flag is specified, it defaults to the field name in kebab-case.
- `clifford.Desc`: Provides a description for the field, which is used in the help message. Requires a `desc` tag with the description text.
- `clifford.Required`: Marks a field as required. If a required field is not provided, `clifford.Parse` will return an error.
- `clifford.Subcommand`: Marks a sub-struct as a subcommand; subcommands can have their own flags/positionals and may opt-in to show help as a subcommand.

---

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.
