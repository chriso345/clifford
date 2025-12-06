package core

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/chriso345/clifford/display"
	"github.com/chriso345/clifford/internal/common"
)

var osExit = os.Exit // Mockable for testing

// buildArgMaps processes the provided args and returns maps for flags and positionals.
func buildArgMaps(args []string) (map[string]string, map[string]int, []string, []int) {
	argMap := map[string]string{}
	argIndex := map[string]int{}
	used := map[int]bool{}

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if strings.HasPrefix(arg, "--") || strings.HasPrefix(arg, "-") {
			argIndex[arg] = i
			used[i] = true
			if i+1 < len(args) && !strings.HasPrefix(args[i+1], "-") {
				argMap[arg] = args[i+1]
				used[i+1] = true
				i++ // skip the value
			}
		}
	}

	var positionals []string
	var positionalIdxs []int
	for i, arg := range args {
		if !used[i] {
			positionals = append(positionals, arg)
			positionalIdxs = append(positionalIdxs, i)
		}
	}
	return argMap, argIndex, positionals, positionalIdxs
}

// parseFields parses flags/positionals into the provided target using only the given args.
// This function does not perform subcommand dispatching.
func parseFields(target any, args []string) error {
	if !common.IsStructPtr(target) {
		return fmt.Errorf("invalid type: must pass pointer to struct")
	}

	argMap, argIndex, positionals, _ := buildArgMaps(args)

	// Handle --help
	if common.MetaArgEnabled("Help", target) {
		if _, ok := argIndex["-h"]; ok {
			help, err := display.BuildHelp(target, false)
			if err != nil {
				return err
			}
			fmt.Println(help)
			osExit(0)
		}
		if _, ok := argIndex["--help"]; ok {
			help, err := display.BuildHelp(target, true)
			if err != nil {
				return err
			}
			fmt.Println(help)
			osExit(0)
		}
	}

	// Handle --version
	if common.MetaArgEnabled("Version", target) {
		if _, ok := argIndex["--version"]; ok {
			version, err := display.BuildVersion(target)
			if err != nil {
				return err
			}
			fmt.Println(version)
			osExit(0)
		}
	}

	v := reflect.ValueOf(target).Elem()
	t := v.Type()
	positionalIndex := 0

	for i := range t.NumField() {
		field := t.Field(i)

		// Skip meta fields like Clifford, Version, Help
		if field.Type.Name() == "Clifford" || field.Type.Name() == "Version" || field.Type.Name() == "Help" {
			continue
		}
		if field.Type.Kind() != reflect.Struct {
			continue
		}

		subVal := v.Field(i)
		subType := field.Type
		tags := common.GetTagsFromEmbedded(subType, field.Name)

		var value string
		found := false

		longFlag := "--" + tags["long"]
		shortFlag := "-" + tags["short"]

		// Check long flag
		if tags["long"] != "" {
			if val, ok := argMap[longFlag]; ok {
				value = val
				found = true
			}
		}
		// Check short flag
		if !found && tags["short"] != "" {
			if val, ok := argMap[shortFlag]; ok {
				value = val
				found = true
			}
		}
		// Handle boolean flags (without values)
		if !found && tags["long"] != "" {
			if _, ok := argIndex[longFlag]; ok {
				value = "true"
				found = true
			}
		}
		if !found && tags["short"] != "" {
			if _, ok := argIndex[shortFlag]; ok {
				value = "true"
				found = true
			}
		}

		// Handle positional arguments (no short or long tag)
		if !found && tags["short"] == "" && tags["long"] == "" {
			if positionalIndex < len(positionals) {
				value = positionals[positionalIndex]
				positionalIndex++
				found = true
			}
		}

		// Required check
		if !found && tags["required"] == "true" {
			return fmt.Errorf("missing required argument: %s", field.Name)
		}

		// Set the value to the `Value` field
		if found {
			valField := subVal.FieldByName("Value")
			if !valField.IsValid() || !valField.CanSet() {
				continue
			}

			switch valField.Kind() {
			case reflect.String:
				valField.SetString(value)
			case reflect.Int:
				if intVal, err := strconv.Atoi(value); err == nil {
					valField.SetInt(int64(intVal))
				}
			case reflect.Float64:
				if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
					valField.SetFloat(floatVal)
				}
			case reflect.Bool:
				if boolVal, err := strconv.ParseBool(value); err == nil {
					valField.SetBool(boolVal)
				}
			default:
				return fmt.Errorf("unsupported type for field %s: %s", field.Name, valField.Kind())
			}
		}
	}

	return nil
}

// parseWithArgs is the recursive parser that supports subcommand dispatch.
func parseWithArgs(target any, args []string) error {
	if !common.IsStructPtr(target) {
		return fmt.Errorf("invalid type: must pass pointer to struct")
	}

	// Normalize args: drop everything before "--"
	if i := common.ArgsIndexOf(args, "--"); i >= 0 {
		args = args[i+1:]
	}

	// Build maps for full args to discover subcommands
	_, _, positionals, positionalIdxs := buildArgMaps(args)

	// If there's a potential subcommand (first positional), attempt to match it.
	if len(positionals) > 0 {
		first := positionals[0]
		v := reflect.ValueOf(target).Elem()
		t := v.Type()
		for i := range t.NumField() {
			field := t.Field(i)
			if field.Type.Kind() != reflect.Struct {
				continue
			}
			// Check for explicit subcmd tag on the parent field
			explicit := field.Tag.Get("subcmd")
			// Check for embedded Subcommand marker
			tags := common.GetTagsFromEmbedded(field.Type, field.Name)
			if explicit == "" && tags["subcmd"] != "true" {
				continue
			}
			name := explicit
			if name == "" {
				name = strings.ToLower(field.Name)
			}
			if name == first {
				// Parse root fields with only args before the subcommand token
				posIdx := positionalIdxs[0]
				rootArgs := args[:posIdx]
				if err := parseFields(target, rootArgs); err != nil {
					return err
				}
				// Descend into subcommand
				subPtr := v.Field(i).Addr().Interface()
				subArgs := args[posIdx+1:]
				return parseWithArgs(subPtr, subArgs)
			}
		}
	}

	// No subcommand matched: parse all fields for this target
	return parseFields(target, args)
}

func Parse(target any) error {
	return parseWithArgs(target, os.Args[1:])
}
