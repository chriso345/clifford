package clifford

import (
	"github.com/chriso345/clifford/core"
	"github.com/chriso345/clifford/display"
)

// Parse parses command-line arguments into the provided target struct.
//
// The target must be a pointer to a struct where each field represents either
// a CLI argument or a group of options. Each sub-struct should contain a `Value`
// field to hold the parsed value, and may be annotated using `clifford` tags or
// helper types like `ShortTag`, `LongTag`, `Required`, and `Desc`.
//
// If the first argument passed to the CLI is `-h` or `--help`, Parse will
// automatically call BuildHelp and exit the program.
//
// Usage:
//
//	target := struct {
//		clifford.Clifford `name:"mytool"`
//
//		Name struct {
//			Value    string
//			clifford.Clifford `short:"n" long:"name" desc:"User name"`
//		}
//
//		Age struct {
//			Value    string
//			clifford.ShortTag // Auto-generates: -a
//			clifford.LongTag  // Auto-generates: --age
//			clifford.Desc     `desc:"Age of the user"`
//		}
//	}{}
//
//	err := clifford.Parse(&target)
//	if err != nil {
//		log.Fatal(err)
//	}
var Parse = core.Parse

// BuildHelp generates and returns a formatted help message for a CLI tool
// defined by the given struct pointer.
// BuildHelp also takes in a boolean `long` parameter that, if set to true,
// will print a more detailed help message with the long_about description
//
// The `target` must be a pointer to a struct that embeds a `Clifford` field
// with a `name` tag. This tag specifies the CLI tool's name and is displayed
// in the usage header.
//
// The function inspects the struct to determine CLI arguments and options,
// including those marked as required. It outputs a help string that includes:
//   - The usage line with the command name and expected arguments
//   - A section for required arguments (based on `Required` tags)
//   - A section for optional flags (based on `short` or `long` tags)
//
// If no `name` tag is found on any embedded `Clifford` field, the function
// returns an error.
//
// Example:
//
//	target := struct {
//		clifford.Clifford `name:"mytool"`
//
//		Filename struct {
//			Value    string
//			clifford.Required
//			clifford.Desc `desc:"Input file path"`
//		}
//
//		Verbose struct {
//			Value    bool
//			clifford.Clifford `short:"v" long:"verbose" desc:"Enable verbose output"`
//		}
//	}{}
//
//	helpText, err := clifford.BuildHelp(&target, false)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(helpText)
var BuildHelp = display.BuildHelp

// BuildVersion returns a formatted version string for the CLI tool defined
// by the provided struct pointer.
//
// The `target` must be a pointer to a struct that embeds a `Clifford` field
// with a `version` tag. This tag specifies the version number of the CLI tool.
// If no version tag is found, the function returns an error.
//
// This function is automatically invoked by `Parse` if the CLI arguments
// include `--version`.
//
// Example:
//
//	target := struct {
//		clifford.Clifford `name:"mytool" version:"1.2.3"`
//	}{}
//
//	version, err := clifford.BuildVersion(&target)
//	if err != nil {
//		log.Fatal(err)
//	}
//	fmt.Println(version) // Output: mytool v1.2.3
var BuildVersion = display.BuildVersion

// BuildHelpWithParent exposes the subcommand-aware help builder for callers/tests.
func BuildHelpWithParent(parent any, subName string, subTarget any, long bool) (string, error) {
	return display.BuildHelpWithParent(parent, subName, subTarget, long)
}
