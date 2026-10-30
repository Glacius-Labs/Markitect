package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestBuildAddsTheSourceRevision(t *testing.T) {
	const pseudo = "v0.0.0-20261029221032-903467c2d394"
	for _, tc := range []struct {
		name     string
		module   string
		settings []debug.BuildSetting
		injected string
		want     string
	}{
		{"no revision", "", nil, "", "1.2.3"},
		{"clean", "", []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}, {Key: "vcs.modified", Value: "false"}}, "", "1.2.3+abc"},
		{"modified", "", []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}, {Key: "vcs.revision", Value: "abc"}}, "", "1.2.3+abc-modified"},
		{"modified without revision", "", []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}, "", "1.2.3"},
		{"injected without embedded revision", "", nil, "def", "1.2.3+def"},
		{"embedded revision wins over injected", "", []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}}, "def", "1.2.3+abc"},
		{"go install of a commit", pseudo, nil, "", "1.2.3+903467c2d394"},
		{"pseudo-version after a tag", "v1.2.4-0.20261029221032-903467c2d394", nil, "", "1.2.3+903467c2d394"},
		{"pseudo-version after a pre-release", "v1.2.3-pre.0.20261029221032-903467c2d394", nil, "", "1.2.3+903467c2d394"},
		{"pseudo-version with build metadata", pseudo + "+dirty", nil, "", "1.2.3+903467c2d394"},
		{"embedded revision wins over the pseudo-version", pseudo, []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}}, "", "1.2.3+abc"},
		{"pseudo-version wins over injected", pseudo, nil, "def", "1.2.3+903467c2d394"},
		{"pseudo-version ignores modified without revision", pseudo, []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}, "", "1.2.3+903467c2d394"},
		{"tagged module version adds nothing", "v1.2.3", nil, "", "1.2.3"},
		{"devel module version adds nothing", "(devel)", nil, "", "1.2.3"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := build("1.2.3", tc.module, tc.settings, tc.injected); got != tc.want {
				t.Fatalf("build() = %q, want %q", got, tc.want)
			}
		})
	}
}

// The 18 versions retired on 17 October 2026 (docs/validation/retired-releases-20261017.json).
var retired = []string{
	"0.1.0", "0.2.0", "0.3.0", "0.3.1", "0.4.0", "0.4.1", "0.5.0", "0.6.0", "0.7.0",
	"0.8.0", "0.8.1", "0.9.0", "0.9.1", "0.10.0", "0.11.0", "0.12.0", "0.13.0", "0.14.1",
}

func TestSourceVersionReusesNoRetiredRelease(t *testing.T) {
	for _, v := range retired {
		if Version() == v {
			t.Fatalf("Version() = %q, a retired release version that is never reused", v)
		}
	}
}
