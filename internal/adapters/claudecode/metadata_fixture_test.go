package claudecode

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func testPath(t *testing.T, s string) snapshot.Path {
	t.Helper()
	p, err := snapshot.NewPath(s)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

type captureProfile func(context.Context, runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error)

func (f captureProfile) PrepareAgentProfile(c context.Context, p runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
	return f(c, p)
}

// Obtain opaque inputs from the actual closed static Runtime consumer. No
// model strategies, organization roles or execution defaults participate.
func profileInput(t *testing.T, inline string, instructions []project.SourceFile) runtimeconfig.ProfileInput {
	t.Helper()
	var input runtimeconfig.ProfileInput
	consumeProfile(t, inline, instructions, captureProfile(func(_ context.Context, p runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
		input = p
		return runtimeconfig.ProfileObservation{InputBinding: p.Binding(), Status: model.NotCompleted}, nil
	}))
	if input.Inline() == nil {
		t.Fatal("actual Runtime consumer did not select fixture profile")
	}
	return input
}
func consumeProfile(t *testing.T, inline string, instructions []project.SourceFile, port runtimeconfig.AgentProfilePreparer) runtimeconfig.Result {
	t.Helper()
	return consumeProfileAt(t, inline, instructions, port, filepath.Join(t.TempDir(), "runtime"))
}
func consumeProfileAt(t *testing.T, inline string, instructions []project.SourceFile, port runtimeconfig.AgentProfilePreparer, boundary string) runtimeconfig.Result {
	t.Helper()
	return consumeProfileSection(t, inline, instructions, port, boundary, "claudeCode")
}
func consumeProfileSection(t *testing.T, inline string, instructions []project.SourceFile, port runtimeconfig.AgentProfilePreparer, boundary, section string) runtimeconfig.Result {
	t.Helper()
	document := "apiVersion: markitect.runtime/v1\nagentProfiles:\n  library-agent:\n    " + section + ":\n"
	for _, line := range strings.Split(strings.TrimSuffix(inline, "\n"), "\n") {
		document += "      " + line + "\n"
	}
	document += "checkProfiles: {}\n"
	location := project.SourceLocation{Boundary: boundary, Path: testPath(t, "profiles.yaml")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte(document))
	if err != nil {
		t.Fatal(err)
	}
	result, err := runtimeconfig.Prepare(context.Background(), project.RawInputs{RuntimeProfiles: []project.SourceFile{{Location: location, File: entry}}, Instructions: instructions}, port, nil)
	if err != nil {
		t.Fatalf("actual static Runtime preparation: %v", err)
	}
	return result
}
