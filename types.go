package clifford

import "github.com/chriso345/clifford/core"

// Clifford is the primary metadata marker for CLI definitions.
//
// It can be embedded in the root struct to define metadata for the CLI tool itself,
// such as its name, version, or other global settings via struct tags.
//
// It can also be embedded into sub-structs to provide additional annotations
// such as `short`, `long`, `desc`, and `required`, either directly or via helper types.
//
// Usage:
//
// Root-level CLI tool definition:
//
//	cli := struct {
//	    Clifford `name:"mytool" version:"1.0.0"`
//	    ...
//	}{}
//
// Sub-struct flag definition using Clifford for metadata:
//
//	cli := struct {
//	    Clifford `name:"mytool"`
//
//	    Name struct {
//	        Value    string
//	        Clifford `short:"n" long:"name" desc:"User name"`
//	    }
//	}{}
type Clifford = core.Clifford

// Version is a marker type that indicates the CLI tool supports a `--version` flag.
//
// When included in the root struct, `Version` enables version display logic.
// If the `version` struct tag is set, Clifford will use it directly. If left empty,
// Clifford may attempt to auto-detect or fallback to a default (implementation dependent).
//
// Usage:
//
// // Automatic detection or programmatic assignment
//
//	cli := struct {
//	    Clifford `name:"mytool"`
//	    Version
//	}{}
//
// // Static version string via struct tag
//
//	cli := struct {
//	    Clifford `name:"mytool"`
//	    Version `version:"1.0.0"`
//	}{}
type Version = core.Version

// Help is a marker type that enables the automatic `--help` and `-h` flag handling.
//
// When embedded in the root struct, this allows the user to request usage help.
// If `-h` or `--help` is passed as the first CLI argument, Clifford will invoke
// BuildHelp automatically and exit gracefully.
//
// Usage:
//
//	cli := struct {
//	    Clifford `name:"mytool"`
//	    Help
//	    ...
//	}{}
type Help = core.Help

// Subcommand is a helper exported from core to mark fields as subcommands.
// Usage: embed clifford.Subcommand in a sub-struct to mark it as a subcommand.
type Subcommand = core.Subcommand

// ShortTag is a helper type used to automatically generate a short flag
// (e.g. `-n`) for a CLI option based on the parent struct field name.
//
// You can embed it in a sub-struct to indicate that a short flag should be
// derived from the first letter of the field name, unless explicitly overridden.
//
// Usage:
//
//	cli := struct {
//	    Name struct {
//	        Value    string
//	        ShortTag // Will auto-generate: -n
//	    }
//	}{}
type ShortTag = core.ShortTag

// LongTag is a helper type used to automatically generate a long flag
// (e.g. `--name`) for a CLI option based on the parent struct field name.
//
// Usage:
//
//	cli := struct {
//	    Name struct {
//	        Value   string
//	        LongTag // Will auto-generate: --name
//	    }
//	}{}
type LongTag = core.LongTag

// Required is a marker type that indicates the associated argument or flag is required.
//
// If a required argument or flag is not provided in the CLI input, the parser will return an error.
//
// Usage:
//
//	cli := struct {
//	    File struct {
//	        Value    string
//	        Required // Must be supplied on the command line
//	    }
//	}{}
type Required = core.Required

// Desc is a helper type that allows you to annotate a CLI option or argument with a description.
//
// This description will be included in the generated help output.
//
// Usage:
//
//	cli := struct {
//	    Name struct {
//	        Value string
//	        Desc  `desc:"Name of the user"`
//	    }
//	}{}
type Desc = core.Desc
