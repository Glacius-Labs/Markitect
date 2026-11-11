// Package buildinfo owns the version of this build and the upstream license
// notices that `markitect version` and `markitect licenses` print.
package buildinfo

import (
	"regexp"
	"runtime/debug"
)

// version is the source version. A release commit sets the version it releases
// and the plugin's version with it; the commit after it sets the next development
// version, such as 0.16.0-dev (docs/releasing.md). The
// retired versions 0.1.0 to 0.14.1 are never reused.
var version = "0.27.0-dev"

// injectedRevision is the source revision a build without VCS information
// names through the linker, as the release workflow does (-X ...buildinfo.injectedRevision=SHA).
// It is the builder's assertion; a revision the Go toolchain recorded always takes precedence.
var injectedRevision string

// Version returns the source version or the immutable build's injected version.
func Version() string { return version }

// Build identifies this binary: the version plus the source revision. The
// VCS revision the Go toolchain embedded comes first, marked "-modified" for
// a changed tree; then the revision in the module's pseudo-version, which
// `go install module@commit` records; then the injected revision; without
// any of them, the version alone.
func Build() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return build(version, "", nil, injectedRevision)
	}
	return build(version, info.Main.Version, info.Settings, injectedRevision)
}

// pseudoVersion matches a Go pseudo-version such as
// v0.0.0-20261029221032-903467c2d394 and captures its revision.
var pseudoVersion = regexp.MustCompile(`^v[0-9]+\.[0-9]+\.[0-9]+-(?:[0-9A-Za-z.-]*\.)?[0-9]{14}-([0-9a-f]{12})(?:\+[0-9A-Za-z.-]+)?$`)

func build(version, module string, settings []debug.BuildSetting, injected string) string {
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
		modified = "" // a modified tree without its revision names nothing
		if m := pseudoVersion.FindStringSubmatch(module); m != nil {
			revision = m[1]
		} else {
			revision = injected
		}
	}
	if revision == "" {
		return version
	}
	return version + "+" + revision + modified
}
