package display

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/chriso345/clifford/errors"
	"github.com/chriso345/clifford/internal/common"
)

const maxPad = 16 // maximum padding width to avoid excessive indentation

func BuildHelp(target any, long bool) (string, error) {
	_ = long // Unused parameter, kept for compatibility
	if !common.IsStructPtr(target) {
		return "", errors.NewParseError("invalid type: must pass pointer to struct")
	}

	t := common.GetStructType(target)

	// Find struct tag with `name`
	name := ""
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Tag.Get("name") != "" {
			name = field.Tag.Get("name")
			break
		}
	}
	if name == "" {
		// Fall back to the running program name if no explicit `name` tag is present.
		name = filepath.Base(os.Args[0])
	}

	var builder strings.Builder
	builder.WriteString(ansiHelp("Usage:", ansiBold, ansiUnderline) + " ")
	builder.WriteString(ansiHelp(name, ansiBold))

	// Collect required args
	requiredArgs := getRequiredArgs(target)
	for _, arg := range requiredArgs {
		// Required positional arguments are shown as angle-bracketed names.
		builder.WriteString(fmt.Sprintf(" <%s>", strings.ToUpper(arg)))
	}

	if hasOptions(target) {
		builder.WriteString(" [OPTIONS]")
	}
	builder.WriteString("\n")

	// Description (if provided) should appear beneath Usage and above the rest of the help.
	// Only include a top-level description when it is provided on the Clifford embedding.
	if d := topLevelDescription(target); d != "" {
		builder.WriteString("\n" + d + "\n")
	}

	// List subcommands if any
	if subcommandsHelp := buildSubcommandsHelp(target); subcommandsHelp != "" {
		builder.WriteString("\n" + ansiHelp("Subcommands:", ansiBold, ansiUnderline) + "\n")
		builder.WriteString(subcommandsHelp)
	}

	if len(requiredArgs) > 0 {
		builder.WriteString("\n" + ansiHelp("Arguments:", ansiBold, ansiUnderline) + "\n")
		builder.WriteString(argsHelp(target))
	}

	if hasOptions(target) {
		builder.WriteString("\n" + ansiHelp("Options:", ansiBold, ansiUnderline) + "\n")
		builder.WriteString(optionsHelp(target))
	}

	return builder.String(), nil
}

// buildSubcommandsHelp returns formatted subcommands lines for the target struct.
func buildSubcommandsHelp(target any) string {
	t := common.GetStructType(target)
	var entries []struct{ name, desc string }
	maxName := 0
	const maxPad = 16 // maximum padding width to avoid excessive indentation

	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Struct {
			continue
		}
		// detect subcommand via embedded marker
		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if tags["subcmd"] != "true" {
			continue
		}
		name := tags["name"]
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		desc := tags["desc"]
		// If the subcommand has an embedded Help with tag "subcmd" or "both",
		// mention that help is available as a subcommand under this entry.
		if tagsHelp := tags["help"]; tagsHelp == "subcmd" || tagsHelp == "both" {
			desc = strings.TrimSpace(desc + " (use '" + name + " help' for more details)")
		}
		entries = append(entries, struct{ name, desc string }{name, desc})
		if len(name) > maxName {
			maxName = len(name)
		}
	}
	// Also include a top-level help subcommand if the root exposes help via subcmd/both
	pt := common.GetStructType(target)
	for i := range pt.NumField() {
		f := pt.Field(i)
		if f.Type.Name() == "Help" {
			helpTag := f.Tag.Get("help")
			if helpTag == "" {
				helpTag = f.Tag.Get("type")
			}
			if helpTag == "subcmd" || helpTag == "both" {
				entries = append(entries, struct{ name, desc string }{"help", "Show help for a specific command"})
				if len("help") > maxName {
					maxName = len("help")
				}
			}
			break
		}
	}

	var builder strings.Builder
	pad := min(maxName, maxPad)
	for _, e := range entries {
		builder.WriteString(fmt.Sprintf("  %-*s %s\n", pad, e.name, e.desc))
	}
	return builder.String()
}

// === HELPERS ===

// argsHelp generates help text for positional arguments in the target struct.
func argsHelp(target any) string {
	t := common.GetStructType(target)

	var lines []string
	maxLen := 0

	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Name() == "Clifford" || field.Type.Name() == "Version" || field.Type.Name() == "Help" {
			continue
		}

		if field.Type.Kind() != reflect.Struct {
			continue
		}

		// Skip marker-only structs (no Value field) such as Desc
		if _, ok := field.Type.FieldByName("Value"); !ok {
			continue
		}

		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if tags["short"] != "" || tags["long"] != "" {
			continue
		}

		argName := field.Name
		desc := tags["desc"]

		// Show required positional arguments without square brackets
		if _, req := tags["required"]; req {
			line := fmt.Sprintf("  %s", strings.ToUpper(argName))
			if len(line) > maxLen {
				maxLen = len(line)
			}
			lines = append(lines, fmt.Sprintf("%s||%s", line, desc))
			continue
		}

		line := fmt.Sprintf("  [%s]", strings.ToUpper(argName))
		if len(line) > maxLen {
			maxLen = len(line)
		}
		lines = append(lines, fmt.Sprintf("%s||%s", line, desc))
	}

	// Format with aligned colons
	var builder strings.Builder
	pad := min(maxLen, maxPad)
	for _, line := range lines {
		parts := strings.SplitN(line, "||", 2)
		padding := strings.Repeat(" ", pad-len(parts[0])+1)
		builder.WriteString(fmt.Sprintf("%s%s %s\n", parts[0], padding, parts[1]))
	}
	return builder.String()
}

// topLevelDescription returns the description provided on the top-level Clifford embedding, if present.
func topLevelDescription(target any) string {
	t := common.GetStructType(target)
	var desc string
	for i := range t.NumField() {
		field := t.Field(i)
		// First, prefer a description on the Clifford embedding.
		if field.Type.Name() == "Clifford" {
			if d := field.Tag.Get("desc"); d != "" {
				return d
			}
		}
		// Otherwise, allow an anonymous top-level Desc embedding.
		if field.Type.Name() == "Desc" {
			if d := field.Tag.Get("desc"); d != "" {
				return d
			}
		}
	}
	return desc
}

// optionsHelp generates help text for options in the target struct.
func optionsHelp(target any) string {
	t := common.GetStructType(target)

	var lines []string
	maxLen := 0

	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Name() == "Clifford" {
			// By default show short + long for version/help; allow disabling via `help_short` or `version_short` tags on the Clifford field.
			showVersionShort := true
			showHelpShort := true
			if val := field.Tag.Get("version_short"); val == "false" {
				showVersionShort = false
			}
			if val := field.Tag.Get("help_short"); val == "false" {
				showHelpShort = false
			}
			if field.Tag.Get("version") != "" {
				if showVersionShort {
					curr := "  -v, --version||Show version information"
					lines = append(lines, curr)
					left := strings.SplitN(curr, "||", 2)[0]
					if len(left) > maxLen {
						maxLen = len(left)
					}
				} else {
					curr := "  --version||Show version information"
					lines = append(lines, curr)
					left := strings.SplitN(curr, "||", 2)[0]
					if len(left) > maxLen {
						maxLen = len(left)
					}
				}
			}
			// Decide whether to show help as a flag based on Clifford tags and Help field annotations
			helpShown := false
			// determine if help flag already present
			helpAdded := false
			for _, l := range lines {
				if strings.Contains(l, "--help") {
					helpAdded = true
					break
				}
			}
			// look for explicit help tag on Clifford
			if field.Tag.Get("help") != "" {
				helpShown = true
			}
			// Also accept `type:"subcmd"|"flag"|"both"` on a top-level Help embedding
			if !helpShown {
				// try scanning parent type for Help embedding type tag
				pt := t
				for i := range pt.NumField() {
					f := pt.Field(i)
					if f.Type.Name() == "Help" {
						tag := f.Tag.Get("help")
						if tag == "" {
							tag = f.Tag.Get("type")
						}
						if tag == "" || tag == "flag" || tag == "both" {
							helpShown = true
						}
						break
					}
				}
			}
			if helpShown && !helpAdded {
				if showHelpShort {
					curr := "  -h, --help||Show this help message"
					lines = append(lines, curr)
					left := strings.SplitN(curr, "||", 2)[0]
					if len(left) > maxLen {
						maxLen = len(left)
					}
				} else {
					curr := "  --help||Show this help message"
					lines = append(lines, curr)
					left := strings.SplitN(curr, "||", 2)[0]
					if len(left) > maxLen {
						maxLen = len(left)
					}
				}
			}
			continue
		}

		if field.Type.Name() == "Version" {
			curr := "  -v, --version||Show version information"
			lines = append(lines, curr)
			left := strings.SplitN(curr, "||", 2)[0]
			if len(left) > maxLen {
				maxLen = len(left)
			}
			continue
		}

		if field.Type.Kind() != reflect.Struct {
			continue
		}

		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if tags["short"] == "" && tags["long"] == "" {
			continue
		}

		short := tags["short"]
		long := tags["long"]
		desc := tags["desc"]

		// Determine the underlying type of the Value field so we can omit type hints for booleans.
		valField, ok := field.Type.FieldByName("Value")
		isBool := ok && valField.Type.Kind() == reflect.Bool
		var typeHint string
		if !isBool {
			typeHint = fmt.Sprintf("[%s]", strings.ToUpper(field.Name))
		}

		var flag string
		switch {
		case short != "" && long != "":
			if typeHint != "" {
				flag = fmt.Sprintf("  -%s, --%s %s", short, long, typeHint)
			} else {
				flag = fmt.Sprintf("  -%s, --%s", short, long)
			}
		case short != "":
			if typeHint != "" {
				flag = fmt.Sprintf("  -%s %s", short, typeHint)
			} else {
				flag = fmt.Sprintf("  -%s", short)
			}
		case long != "":
			if typeHint != "" {
				flag = fmt.Sprintf("  --%s %s", long, typeHint)
			} else {
				flag = fmt.Sprintf("  --%s", long)
			}
		}

		// Append default value to description if present
		if d, ok := tags["default"]; ok && d != "" {
			if desc == "" {
				desc = fmt.Sprintf("(default: %s)", d)
			} else {
				desc = fmt.Sprintf("%s (default: %s)", desc, d)
			}
		}

		if len(flag) > maxLen {
			maxLen = len(flag)
		}
		lines = append(lines, fmt.Sprintf("%s||%s", flag, desc))
	}

	// Format with aligned colons
	var builder strings.Builder
	for _, line := range lines {
		parts := strings.SplitN(line, "||", 2)
		padding := strings.Repeat(" ", maxLen-len(parts[0]))
		builder.WriteString(fmt.Sprintf("%s%s  %s\n", parts[0], padding, parts[1]))
	}
	return builder.String()
}

// getRequiredArgs returns a list of required argument names from the target struct.
func getRequiredArgs(target any) []string {
	t := common.GetStructType(target)

	var args []string
	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Struct {
			continue
		}

		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if tags["short"] != "" || tags["long"] != "" {
			continue
		}

		if _, ok := tags["required"]; ok {
			args = append(args, field.Name)
		}
	}
	return args
}

// hasOptions checks if the target struct has any options defined with short or long flags.
func hasOptions(target any) bool {
	t := common.GetStructType(target)

	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Struct {
			continue
		}

		if field.Type.Name() == "Version" || field.Tag.Get("version") != "" {
			return true
		}
		if field.Type.Name() == "Help" {
			// Check top-level Help tagging: only counts as option if type is flag or both (or unspecified)
			if val := field.Tag.Get("help"); val != "" {
				if val == "flag" || val == "both" {
					return true
				}
				return false
			}
			if val := field.Tag.Get("type"); val != "" {
				if val == "flag" || val == "both" {
					return true
				}
				return false
			}
			// default behavior: show help flag
			return true
		}
		if field.Tag.Get("help") != "" || field.Tag.Get("type") == "flag" {
			return true
		}

		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if tags["short"] != "" || tags["long"] != "" {
			return true
		}
	}
	return false
}
