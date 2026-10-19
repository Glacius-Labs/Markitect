// Package buildinfo owns the version shared by product and release tools.
package buildinfo

import "runtime/debug"

var version = "0.14.1"

// Version returns the source version or the immutable build's injected version.
func Version() string { return version }

// Build identifies this binary: the version plus the VCS revision the Go
// toolchain embedded, marked "-modified" for a changed tree. Without an
// embedded revision it is the version alone.
func Build() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return version
	}
	return build(version, info.Settings)
}

func build(version string, settings []debug.BuildSetting) string {
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
		return version
	}
	return version + "+" + revision + modified
}
