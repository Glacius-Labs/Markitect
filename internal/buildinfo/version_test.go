package buildinfo

import (
	"runtime/debug"
	"testing"
)

func TestBuildAddsTheEmbeddedRevision(t *testing.T) {
	for _, tc := range []struct {
		name     string
		settings []debug.BuildSetting
		injected string
		want     string
	}{
		{"no revision", nil, "", "1.2.3"},
		{"clean", []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}, {Key: "vcs.modified", Value: "false"}}, "", "1.2.3+abc"},
		{"modified", []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}, {Key: "vcs.revision", Value: "abc"}}, "", "1.2.3+abc-modified"},
		{"modified without revision", []debug.BuildSetting{{Key: "vcs.modified", Value: "true"}}, "", "1.2.3"},
		{"injected without embedded revision", nil, "def", "1.2.3+def"},
		{"embedded revision wins over injected", []debug.BuildSetting{{Key: "vcs.revision", Value: "abc"}}, "def", "1.2.3+abc"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := build("1.2.3", tc.settings, tc.injected); got != tc.want {
				t.Fatalf("build() = %q, want %q", got, tc.want)
			}
		})
	}
}
