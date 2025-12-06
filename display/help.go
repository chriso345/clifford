package display

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/chriso345/clifford/internal/common"
)

func BuildHelp(target any, long bool) (string, error) {
	_ = long // Unused parameter, kept for compatibility
	if !common.IsStructPtr(target) {
		return "", fmt.Errorf("invalid type: must pass pointer to struct")
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
		return "", fmt.Errorf("struct must embed `Clifford` with `name` tag")
	}

	var builder strings.Builder
	builder.WriteString(ansiHelp("Usage:", ansiBold, ansiUnderline) + " ")
	builder.WriteString(ansiHelp(name, ansiBold))

	// Collect required args
	requiredArgs := getRequiredArgs(target)
	for _, arg := range requiredArgs {
		builder.WriteString(fmt.Sprintf(" [%s]", strings.ToUpper(arg)))
	}

	if hasOptions(target) {
		builder.WriteString(" [OPTIONS]")
	}
	builder.WriteString("\n")

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
	var lines []string
	maxLen := 0

	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Kind() != reflect.Struct {
			continue
		}
		// detect subcommand via tag or embedded marker
		explicit := field.Tag.Get("subcmd")
		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if explicit == "" && tags["subcmd"] != "true" {
			continue
		}
		name := explicit
		if name == "" {
			name = strings.ToLower(field.Name)
		}
		desc := tags["desc"]
		line := fmt.Sprintf("  %s||%s", name, desc)
		if len(line) > maxLen {
			maxLen = len(line)
		}
		lines = append(lines, line)
	}

	var builder strings.Builder
	for _, line := range lines {
		parts := strings.SplitN(line, "||", 2)
		padding := strings.Repeat(" ", maxLen-len(parts[0]))
		builder.WriteString(fmt.Sprintf("%s%s  %s\n", parts[0], padding, parts[1]))
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

		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if tags["short"] != "" || tags["long"] != "" {
			continue
		}

		argName := field.Name
		desc := tags["desc"]
		line := fmt.Sprintf("  [%s]", strings.ToUpper(argName))
		if len(line) > maxLen {
			maxLen = len(line)
		}
		lines = append(lines, fmt.Sprintf("%s||%s", line, desc))
	}

	// Format with aligned colons
	var builder strings.Builder
	for _, line := range lines {
		parts := strings.SplitN(line, "||", 2)
		padding := strings.Repeat(" ", maxLen-len(parts[0])+1)
		builder.WriteString(fmt.Sprintf("%s%s %s\n", parts[0], padding, parts[1]))
	}
	return builder.String()
}

// optionsHelp generates help text for options in the target struct.
func optionsHelp(target any) string {
	t := common.GetStructType(target)

	var lines []string
	maxLen := 0

	for i := range t.NumField() {
		field := t.Field(i)
		if field.Type.Name() == "Clifford" {
			if field.Tag.Get("version") != "" {
				curr := "  --version||Show version information"
				lines = append(lines, curr)
				if 11 > maxLen {
					maxLen = 11
				}
			}
			if field.Tag.Get("help") != "" {
				curr := "  --help||Show this help message"
				lines = append(lines, curr)
				if 8 > maxLen {
					maxLen = 8
				}
			}
			continue
		}

		if field.Type.Name() == "Version" {
			curr := "  --version||Show version information"
			lines = append(lines, curr)
			if 11 > maxLen {
				maxLen = 11
			}
			continue
		}

		if field.Type.Name() == "Help" {
			curr := "  --help||Show this help message"
			lines = append(lines, curr)
			if 8 > maxLen {
				maxLen = 8
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
		typeHint := fmt.Sprintf("[%s]", strings.ToUpper(field.Name))

		var flag string
		switch {
		case short != "" && long != "":
			flag = fmt.Sprintf("  -%s, --%s %s", short, long, typeHint)
		case short != "":
			flag = fmt.Sprintf("  -%s %s", short, typeHint)
		case long != "":
			flag = fmt.Sprintf("  --%s %s", long, typeHint)
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
		if field.Type.Name() == "Help" || field.Tag.Get("help") != "" {
			return true
		}

		tags := common.GetTagsFromEmbedded(field.Type, field.Name)
		if tags["short"] != "" || tags["long"] != "" {
			return true
		}
	}
	return false
}
