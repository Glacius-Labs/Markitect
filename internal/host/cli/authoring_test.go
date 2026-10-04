package cli

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/authoring"
)

func TestAuthoringEmitsCompiledBuiltInContextWithoutRepository(t *testing.T) {
	code, output, stderr := invoke("authoring")
	if code != 0 {
		t.Fatalf("authoring exit=%d stderr=%s output=%s", code, stderr, output)
	}
	got := decodeYAML[app.Context](t, output)
	wantDigest, err := currentToolDigest()
	if err != nil {
		t.Fatal(err)
	}
	want, err := authoring.Context(version, wantDigest)
	if err != nil {
		t.Fatalf("compile built-in context: %v", err)
	}
	if got.Entry != "core/Skill/authoring" || got.Revision != "embedded" || got.Provisional || got.ToolDigest != wantDigest || got.Digest != want.Digest || got.SnapshotDigest != want.SnapshotDigest {
		t.Fatalf("unexpected built-in context identity: got %#v, want %#v", got, want)
	}
	found := false
	for _, input := range got.Inputs {
		if input.Resource != nil && input.Resource.Kind == "Skill" && input.Resource.Metadata.Name == "authoring" {
			found = true
		}
	}
	if !found {
		t.Fatalf("built-in context omitted its authoring Skill: %#v", got.Inputs)
	}
}

func TestAuthoringRejectsRepositoryAndOtherOptions(t *testing.T) {
	for _, args := range [][]string{
		{"authoring", "--repo", "."},
		{"authoring", "--revision", "HEAD"},
		{"authoring", "--kind", "Skill"},
	} {
		t.Run(strings.Join(args[1:], "_"), func(t *testing.T) {
			code, _, _ := invoke(args...)
			if code != 2 {
				t.Fatalf("invoke(%v) = %d, want usage exit 2", args, code)
			}
		})
	}
}
