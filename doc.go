// Package clifford is a CLI argument parsing library for Go that uses reflection
// and struct tags to define command-line interfaces declaratively.
//
// It supports positional arguments, short and long flags, required arguments,
// and automatic generation of help and version output.
//
// The library is designed to be easy to use and integrate into Go CLI tools,
// providing a clean API for defining and parsing command-line parameters.
package clifford

//go:generate gomarkdoc ./ -o docs/clifford.md
