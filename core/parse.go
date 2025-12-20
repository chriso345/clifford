package core

import (
	"fmt"
	"os"
	"reflect"
	"strconv"
	"strings"

	"github.com/chriso345/clifford/display"
	"github.com/chriso345/clifford/errors"
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
		return errors.NewParseError("invalid type: must pass pointer to struct")
	}

	argMap, argIndex, positionals, _ := buildArgMaps(args)

	// Determine root help exposure mode (flag/subcmd/both). Default is flag.
	helpMode := "flag"
	if common.IsStructPtr(target) {
		pt := common.GetStructType(target)
		for i := range pt.NumField() {
			f := pt.Field(i)
			if f.Type.Name() == "Help" {
				if val := f.Tag.Get("help"); val != "" {
					helpMode = val
				} else if val := f.Tag.Get("type"); val != "" {
					helpMode = val
				}
				break
			}
		}
	}
	// Handle --help only when helpMode allows flag-based help
	if helpMode != "subcmd" && common.MetaArgEnabled("Help", target) {
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

		// If not found, use any declared default value.
		if !found {
			if d, ok := tags["default"]; ok && d != "" {
				value = d
				found = true
			}
		}

		// Required check
		if !found && tags["required"] == "true" {
			return errors.NewMissingArg(field.Name)
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
				return errors.NewUnsupportedField(field.Name, valField.Kind().String())
			}
		}
	}

	return nil
}

// parseWithArgs is the recursive parser that supports subcommand dispatch.
func parseWithArgs(target any, args []string) error {
	if !common.IsStructPtr(target) {
		return errors.NewParseError("invalid type: must pass pointer to struct")
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
		// Support invocation form: app help [subcommand]
		if first == "help" {
			if len(positionals) == 1 {
				helper, err := display.BuildHelp(target, false)
				if err != nil {
					return err
				}
				fmt.Println(helper)
				osExit(0)
			}
			second := positionals[1]
			// collect subcommand names for suggestion
			var subNames []string
			for i := range t.NumField() {
				field := t.Field(i)
				if field.Type.Kind() != reflect.Struct {
					continue
				}
				tags := common.GetTagsFromEmbedded(field.Type, field.Name)
				if tags["subcmd"] != "true" {
					continue
				}
				name := tags["name"]
				if name == "" {
					name = strings.ToLower(field.Name)
				}
				subNames = append(subNames, name)
				if name == second {
					// Only allow help via subcommand when the subcommand advertises help as subcmd or both
					if ht := tags["help"]; ht == "subcmd" || ht == "both" {
						subPtr := v.Field(i).Addr().Interface()
						helper, err := display.BuildHelpWithParent(target, name, subPtr, false)
						if err != nil {
							return err
						}
						fmt.Println(helper)
						osExit(0)
					}
				}
			}
			// No matching subcommand found: return informative error
			if len(subNames) > 0 {
				suggestion := closestMatch(second, subNames)
				return errors.NewUnknownSubcommand(second, suggestion)
			}
		}
		var subNames []string
		for i := range t.NumField() {
			field := t.Field(i)
			if field.Type.Kind() != reflect.Struct {
				continue
			}
			// Check for embedded Subcommand marker
			tags := common.GetTagsFromEmbedded(field.Type, field.Name)
			if tags["subcmd"] != "true" {
				continue
			}
			name := tags["name"]
			if name == "" {
				name = strings.ToLower(field.Name)
			}
			subNames = append(subNames, name)
			if name == first {
				// Parse root fields with only args before the subcommand token
				posIdx := positionalIdxs[0]
				rootArgs := args[:posIdx]
				if err := parseFields(target, rootArgs); err != nil {
					return err
				}
				// Mark the embedded Subcommand boolean field as used (true) so callers can inspect the parsed struct.
				subVal := v.Field(i)
				subType := subVal.Type()
				for j := 0; j < subType.NumField(); j++ {
					nf := subType.Field(j)
					if nf.Anonymous && nf.Type.Name() == "Subcommand" {
						f := subVal.Field(j)
						if f.IsValid() && f.CanSet() && f.Kind() == reflect.Bool {
							f.SetBool(true)
						}
						break
					}
				}
				// If the subcommand help/version is being requested, build help that shows parent + subcommand.
				subPtr := v.Field(i).Addr().Interface()
				subArgs := args[posIdx+1:]
				for _, a := range subArgs {
					if a == "-h" || a == "--help" {
						helper, err := display.BuildHelpWithParent(target, name, subPtr, a == "--help")
						if err != nil {
							return err
						}
						fmt.Println(helper)
						osExit(0)
					}
				}
				return parseWithArgs(subPtr, subArgs)
			}
		}
		// If we had positionals and potential subcommands but no match, return an informative error
		if len(subNames) > 0 {
			suggestion := closestMatch(first, subNames)
			return errors.NewUnknownSubcommand(first, suggestion)
		}
	}

	// No subcommand matched: parse all fields for this target
	return parseFields(target, args)
}

// closestMatch returns the candidate with the smallest edit distance to target, or
// empty string if none are within a reasonable threshold.
func closestMatch(target string, candidates []string) string {
	if target == "" || len(candidates) == 0 {
		return ""
	}
	low := strings.ToLower(target)
	// Prefer prefix matches (case-insensitive)
	for _, c := range candidates {
		if strings.HasPrefix(strings.ToLower(c), low) {
			return c
		}
	}

	best := ""
	bestDist := -1
	for _, c := range candidates {
		lc := strings.ToLower(c)
		// Quick length check to avoid large distances
		if abs(len(lc)-len(low)) > 3 {
			continue
		}
		// Treat single transposition as distance 1
		if isTransposition(low, lc) {
			return c
		}
		d := levenshtein(low, lc)
		if bestDist == -1 || d < bestDist {
			bestDist = d
			best = c
		}
	}
	// Only suggest if distance is small (adaptive threshold)
	if bestDist >= 0 && bestDist <= max(2, len(low)/3) {
		return best
	}
	return ""
}

// isTransposition checks for one-character transposition (Damerau case)
func isTransposition(a, b string) bool {
	if len(a) != len(b) || len(a) < 2 {
		return false
	}
	var diff []int
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			diff = append(diff, i)
			if len(diff) > 2 {
				return false
			}
		}
	}
	if len(diff) != 2 {
		return false
	}
	return a[diff[0]] == b[diff[1]] && a[diff[1]] == b[diff[0]]
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func abs(a int) int {
	if a < 0 {
		return -a
	}
	return a
}

// levenshtein computes the Levenshtein edit distance between a and b.
func levenshtein(a, b string) int {
	if a == b {
		return 0
	}
	la := len(a)
	lb := len(b)
	if la == 0 {
		return lb
	}
	if lb == 0 {
		return la
	}
	// Initialize distance matrix with two rows to save memory
	prev := make([]int, lb+1)
	curr := make([]int, lb+1)
	for j := 0; j <= lb; j++ {
		prev[j] = j
	}
	for i := 1; i <= la; i++ {
		curr[0] = i
		ai := a[i-1]
		for j := 1; j <= lb; j++ {
			cost := 0
			if ai != b[j-1] {
				cost = 1
			}
			del := prev[j] + 1
			ins := curr[j-1] + 1
			sub := prev[j-1] + cost
			min := del
			if ins < min {
				min = ins
			}
			if sub < min {
				min = sub
			}
			curr[j] = min
		}
		copy(prev, curr)
	}
	return prev[lb]
}

func Parse(target any) error {
	return parseWithArgs(target, os.Args[1:])
}
