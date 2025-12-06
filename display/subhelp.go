package display

import (
	"fmt"
	"github.com/chriso345/clifford/internal/common"
	"strings"
)

// BuildHelpWithParent builds help for a subcommand while showing the parent application name
// and the subcommand name together (e.g. "app server [OPTIONS]").
func BuildHelpWithParent(parent any, subName string, subTarget any, long bool) (string, error) {
	if !common.IsStructPtr(subTarget) {
		return "", fmt.Errorf("invalid type: must pass pointer to struct")
	}

	// Determine parent name from parent's Clifford embedding if present
	parentName := ""
	if common.IsStructPtr(parent) {
		pt := common.GetStructType(parent)
		for i := range pt.NumField() {
			f := pt.Field(i)
			if f.Type.Name() == "Clifford" {
				parentName = f.Tag.Get("name")
				break
			}
		}
	}
	if parentName == "" {
		parentName = "<app>"
	}

	fullName := parentName + " " + subName

	var builder strings.Builder
	builder.WriteString(ansiHelp("Usage:", ansiBold, ansiUnderline) + " ")
	builder.WriteString(ansiHelp(fullName, ansiBold))

	// required args for subTarget
	requiredArgs := getRequiredArgs(subTarget)
	for _, arg := range requiredArgs {
		builder.WriteString(fmt.Sprintf(" <%s>", strings.ToUpper(arg)))
	}
	if hasOptions(subTarget) {
		builder.WriteString(" [OPTIONS]")
	}
	builder.WriteString("\n")

	// description from subTarget (Desc embedded)
	if d := topLevelDescription(subTarget); d != "" {
		builder.WriteString("\n" + d + "\n")
	}

	if hasOptions(subTarget) {
		builder.WriteString("\n" + ansiHelp("Options:", ansiBold, ansiUnderline) + "\n")
		// For subcommand help, show options from subTarget; decide whether to include -h/-v based on parent Clifford tags
		builder.WriteString(optionsHelp(subTarget))
	}

	return builder.String(), nil
}
