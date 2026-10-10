package projectadoption

import (
	"errors"
	"runtime/debug"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

// CurrentBindings derives the two adoption bindings from the active schema
// supplied by the Host and the running tool's build identity. Callers should
// pass the same schema used to validate the target project.
func CurrentBindings(activeSchema core.Schema) (schemaDigest, buildDigest string, err error) {
	if activeSchema.APIVersion == "" || activeSchema.Kinds == nil {
		return "", "", errors.New("active project schema is empty")
	}
	info, ok := debug.ReadBuildInfo()
	if !ok || info == nil || info.Main.Path == "" {
		return "", "", errors.New("running Markitect build identity is unavailable")
	}
	executableDigest, err := projectwork.ToolBuildDigest()
	if err != nil {
		return "", "", err
	}
	return digestValue(activeSchema), digestValue(struct {
		GoVersion        string
		Path             string
		Main             debug.Module
		Deps             []*debug.Module
		Settings         []debug.BuildSetting
		ExecutableDigest string
	}{info.GoVersion, info.Path, info.Main, info.Deps, info.Settings, executableDigest}), nil
}
