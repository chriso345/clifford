package core

import (
	"fmt"
	"os"
	"path/filepath"
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

		// Skip meta fields like Clifford, Version, Help and inline Desc or other non-value structs
		if field.Type.Name() == "Clifford" || field.Type.Name() == "Version" || field.Type.Name() == "Help" {
			continue
		}
		if field.Type.Kind() != reflect.Struct {
			// Skip anonymous embedded non-struct markers (like Subcommand)
			if field.Anonymous {
				continue
			}
			// Handle inline primitive fields (e.g. MaxItems int `short:"n" long:"max-items"`)
			tags := make(map[string]string)
			for _, key := range []string{"default", "desc", "required", "short", "long"} {
				if val := field.Tag.Get(key); val != "" {
					tags[key] = val
				}
			}

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

			// Set the value directly on the field
			if found {
				valField := v.Field(i)
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

			continue
		}

		// Skip marker-only embedded structs that don't have a Value field
		if _, ok := field.Type.FieldByName("Value"); !ok {
			// If the field is marked Required at the struct level (e.g., embedded Required), report missing
			if common.GetTagsFromEmbedded(field.Type, field.Name)["required"] == "true" {
				return errors.NewMissingArg(field.Name)
			}
			continue
		}

		subVal := v.Field(i)
		subType := field.Type
		tags := common.GetTagsFromEmbedded(subType, field.Name)
		// Skip subcommand containers; they are dispatched separately
		if tags["subcmd"] == "true" {
			continue
		}

		// First, check if the sub-struct itself has a short/long tags (i.e., acts as a flag container)
		var value string
		found := false

		longFlag := "--" + tags["long"]
		shortFlag := "-" + tags["short"]

		// Check long flag on container
		if tags["long"] != "" {
			if val, ok := argMap[longFlag]; ok {
				value = val
				found = true
			}
		}
		// Check short flag on container
		if !found && tags["short"] != "" {
			if val, ok := argMap[shortFlag]; ok {
				value = val
				found = true
			}
		}
		// Handle boolean flags (without values) on container
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

		// Handle positional arguments for the container (no short or long tag)
		if !found && tags["short"] == "" && tags["long"] == "" {
			if positionalIndex < len(positionals) {
				value = positionals[positionalIndex]
				positionalIndex++
				found = true
			}
		}

		// If not found, use any declared default value on container.
		if !found {
			if d, ok := tags["default"]; ok && d != "" {
				value = d
				found = true
			}
		}

		// Required check for container
		if !found && tags["required"] == "true" {
			return errors.NewMissingArg(field.Name)
		}

		// If a value was found for the container, set it to its Value field
		if found {
			valField := subVal.FieldByName("Value")
			if valField.IsValid() && valField.CanSet() {
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

		// Next, handle inline primitive fields declared inside the sub-struct (e.g., MaxItems int `short:"n" long:"max-items"`)
		for j := 0; j < subType.NumField(); j++ {
			inner := subType.Field(j)
			// skip anonymous embedded markers
			if inner.Anonymous {
				continue
			}
			// Skip inner Value field (handled by container) and only consider non-struct primitive fields
			if inner.Name == "Value" {
				continue
			}
			if inner.Type.Kind() == reflect.Struct {
				continue
			}

			// Collect tags from struct tags on the inner field
			tags2 := make(map[string]string)
			for _, key := range []string{"default", "desc", "required", "short", "long"} {
				if val := inner.Tag.Get(key); val != "" {
					tags2[key] = val
				}
			}
			// allow positional inner fields (no short/long) as well
			// proceed even if short/long empty

			lFlag := "--" + tags2["long"]
			sFlag := "-" + tags2["short"]
			var iv string
			foundInner := false
			if tags2["long"] != "" {
				if val, ok := argMap[lFlag]; ok {
					iv = val
					foundInner = true
				}
			}
			if !foundInner && tags2["short"] != "" {
				if val, ok := argMap[sFlag]; ok {
					iv = val
					foundInner = true
				}
			}
			if !foundInner && tags2["long"] != "" {
				if _, ok := argIndex[lFlag]; ok {
					iv = "true"
					foundInner = true
				}
			}
			if !foundInner && tags2["short"] != "" {
				if _, ok := argIndex[sFlag]; ok {
					iv = "true"
					foundInner = true
				}
			}
			if !foundInner && tags2["short"] == "" && tags2["long"] == "" {
				// positional inner field
				if positionalIndex < len(positionals) {
					iv = positionals[positionalIndex]
					positionalIndex++
					foundInner = true
				}
			}
			if !foundInner {
				if d, ok := tags2["default"]; ok && d != "" {
					iv = d
					foundInner = true
				}
			}
			if !foundInner && tags2["required"] == "true" {
				return errors.NewMissingArg(inner.Name)
			}
			if foundInner {
				// set value on subVal's field
				f := subVal.FieldByName(inner.Name)
				if !f.IsValid() || !f.CanSet() {
					continue
				}
				switch f.Kind() {
				case reflect.String:
					f.SetString(iv)
				case reflect.Int:
					if intVal, err := strconv.Atoi(iv); err == nil {
						f.SetInt(int64(intVal))
					}
				case reflect.Float64:
					if floatVal, err := strconv.ParseFloat(iv, 64); err == nil {
						f.SetFloat(floatVal)
					}
				case reflect.Bool:
					if boolVal, err := strconv.ParseBool(iv); err == nil {
						f.SetBool(boolVal)
					}
				default:
					return errors.NewUnsupportedField(inner.Name, f.Kind().String())
				}
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
				// Always exit after printing help
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
						// Always exit after printing help
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
				// Support positional form: app <subcmd> help
				if len(subArgs) > 0 && subArgs[0] == "help" {
					helper, err := display.BuildHelpWithParent(target, name, subPtr, false)
					if err != nil {
						return err
					}
					// Adjust the Usage line to omit the subcommand name
					parts := strings.SplitN(helper, "\n", 2)
					if len(parts) > 0 {
						usage := parts[0]
						parentName := ""
						if common.IsStructPtr(target) {
							pt := common.GetStructType(target)
							for i := range pt.NumField() {
								f := pt.Field(i)
								if f.Type.Name() == "Clifford" {
									parentName = f.Tag.Get("name")
									break
								}
							}
						}
						if parentName == "" {
							parentName = filepath.Base(os.Args[0])
						}
						usage = strings.Replace(usage, parentName+" "+name, parentName, 1)
						if len(parts) == 1 {
							helper = usage
						} else {
							helper = usage + "\n" + parts[1]
						}
					}
					// Build Arguments section from subcommand's positional fields
					subT := common.GetStructType(subPtr)
					var entries []struct {
						name, desc string
						req        bool
					}
					for k := 0; k < subT.NumField(); k++ {
						f := subT.Field(k)
						if f.Type.Kind() != reflect.Struct {
							continue
						}
						// only include value-carrying fields without short/long tags
						if _, ok := f.Type.FieldByName("Value"); !ok {
							continue
						}
						sTags := common.GetTagsFromEmbedded(f.Type, f.Name)
						if sTags["short"] != "" || sTags["long"] != "" {
							continue
						}
						nameUp := strings.ToUpper(f.Name)
						desc := sTags["desc"]
						req := sTags["required"] == "true"
						entries = append(entries, struct {
							name, desc string
							req        bool
						}{nameUp, desc, req})
					}
					if len(entries) > 0 {
						maxLen := 0
						for _, e := range entries {
							if len(e.name) > maxLen {
								maxLen = len(e.name)
							}
						}
						pad := maxLen
						if pad < 1 {
							pad = 1
						}
						if pad > 12 {
							pad = 12
						}
						var b strings.Builder
						for _, e := range entries {
							left := e.name
							// ensure at least 4 spaces between name and desc
							minGap := 4
							paddingCount := pad - len(left) + minGap
							if paddingCount < minGap {
								paddingCount = minGap
							}
							padding := strings.Repeat(" ", paddingCount)
							b.WriteString(fmt.Sprintf("  %s%s %s\n", left, padding, e.desc))
						}
						helper = helper + "\nArguments:\n" + b.String() + "\n"
					}
					fmt.Println(helper)
					// Always exit after printing help
					osExit(0)
				}
				for _, a := range subArgs {
					if a == "-h" || a == "--help" {
						// Check if the subcommand struct explicitly enables help as a flag
						subTags := common.GetTagsFromEmbedded(subType, field.Name)
						if ht := subTags["help"]; ht == "flag" || ht == "both" {
							helper, err := display.BuildHelpWithParent(target, name, subPtr, a == "--help")
							if err != nil {
								return err
							}
							fmt.Println(helper)
							osExit(0)
						}
						// Otherwise, consult root Help embedding: only allow flag-style help if root help mode is not "subcmd".
						if common.MetaArgEnabled("Help", target) {
							helpMode := "flag"
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
							if helpMode != "subcmd" {
								helper, err := display.BuildHelpWithParent(target, name, subPtr, a == "--help")
								if err != nil {
									return err
								}
								fmt.Println(helper)
								osExit(0)
							}
						}
						// If we get here, help isn't enabled in this context; treat as unknown flag
						return errors.NewParseError("unknown flag: " + a)
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
