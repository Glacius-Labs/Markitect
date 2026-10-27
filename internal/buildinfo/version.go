// Package buildinfo owns the version shared by product and release tools.
package buildinfo

import "runtime/debug"

var version = "0.14.1"

// injectedRevision is the source revision a build without VCS information
// names through the linker, as the runner image does (-X ...buildinfo.injectedRevision=SHA).
// It is the builder's assertion; an embedded VCS revision always takes precedence.
var injectedRevision string

// Version returns the source version or the immutable build's injected version.
func Version() string { return version }

// Build identifies this binary: the version plus the VCS revision the Go
// toolchain embedded, marked "-modified" for a changed tree. Without an
// embedded revision it uses the injected revision, else the version alone.
func Build() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return build(version, nil, injectedRevision)
	}
	return build(version, info.Settings, injectedRevision)
}

func build(version string, settings []debug.BuildSetting, injected string) string {
	var revision, modified string
	for _, setting := range settings {
		switch setting.Key {
		case "vcs.revision":
			revision = setting.Value
		case "vcs.modified":
			if setting.Value == "true" {
				modified = "-modified"
			}
		}
	}
	if revision == "" {
		if injected == "" {
			return version
		}
		return version + "+" + injected
	}
	return version + "+" + revision + modified
}
