package errors

import "fmt"

// ParseError represents a generic parsing error produced by the CLI parser.
// It is intended for user-facing messages.
type ParseError struct{ Msg string }

func (e ParseError) Error() string { return e.Msg }

// MissingArgError indicates a required positional or flag was not provided.
type MissingArgError struct{ Field string }

func (e MissingArgError) Error() string {
	return fmt.Sprintf("missing required argument: %s", e.Field)
}

// UnknownSubcommandError indicates the user invoked a subcommand that does not exist.
// Suggestion, if present, is a close match the user may have intended.
type UnknownSubcommandError struct{ Name, Suggestion string }

func (e UnknownSubcommandError) Error() string {
	if e.Suggestion != "" {
		return fmt.Sprintf("unknown subcommand: %s (did you mean %q?)", e.Name, e.Suggestion)
	}
	return fmt.Sprintf("unknown subcommand: %s", e.Name)
}

// UnsupportedFieldTypeError indicates the CLI contains an unsupported field type.
type UnsupportedFieldTypeError struct{ Field, Type string }

func (e UnsupportedFieldTypeError) Error() string {
	return fmt.Sprintf("unsupported type for field %s: %s", e.Field, e.Type)
}

// Helper constructors
func NewParseError(msg string) error   { return ParseError{Msg: msg} }
func NewMissingArg(field string) error { return MissingArgError{Field: field} }
func NewUnknownSubcommand(name, suggestion string) error {
	return UnknownSubcommandError{Name: name, Suggestion: suggestion}
}
func NewUnsupportedField(field, typ string) error {
	return UnsupportedFieldTypeError{Field: field, Type: typ}
}
