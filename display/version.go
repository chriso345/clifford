package display

import (
	"fmt"
	"runtime/debug"

	"github.com/chriso345/clifford/errors"
	"github.com/chriso345/clifford/internal/common"
)

func BuildVersion(target any) (string, error) {
	if !common.IsStructPtr(target) {
		return "", errors.NewParseError("invalid type: must pass pointer to struct")
	}

	t := common.GetStructType(target)

	var name string
	var versionFromClifford string
	var versionFromVersionField string

	for i := range t.NumField() {
		field := t.Field(i)

		if field.Type.Name() == "Clifford" {
			if tag := field.Tag.Get("name"); tag != "" {
				name = tag
			}
			if tag := field.Tag.Get("version"); tag != "" {
				versionFromClifford = tag
			}
			continue
		}

		if field.Name == "Version" {
			if tag := field.Tag.Get("version"); tag != "" {
				versionFromVersionField = tag
			}
		}
	}

	if versionFromClifford != "" && versionFromVersionField != "" {
		return "", errors.NewParseError("conflicting version tags: both Clifford and Version field specify a version")
	}

	version := versionFromVersionField
	if version == "" {
		version = versionFromClifford
	}

	if name != "" {
		name = name + " "
	}

	if version == "" {
		infered, err := inferVersion()
		if err != nil {
			return "No version specified", nil
		}
		version = infered
	}

	return fmt.Sprintf("%sv%s", name, version), nil
}

// inferVersion attempts to infer the user's module version from build info.
func inferVersion() (string, error) {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "", errors.NewParseError("unable to read build info")
	}

	if info.Main.Version != "" && info.Main.Version != "(devel)" {
		return info.Main.Version, nil
	}

	return "", errors.NewParseError("no version info found in build metadata")
}
