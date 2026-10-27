package claudecode

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const fixtureImage = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"

// fixtureMetadata is the shared image's runtime.env: both CLIs and agentd.
const fixtureMetadata = "codex=codex-cli 0.162.0\ncodexSchema=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nagentd=development\nclaude=2.1.308 (Claude Code)\n"

func claudeProfile(engine string) string {
	return fmt.Sprintf(`profileVersion: 1
engineExecutable: %q
imageID: %s
agentdVersion: development
instructions: []
credentialRef: exact-runtime-credential
maxWireBytes: 4194304
session:
  version: 1
  model: sonnet
  effort: high
  baseInstructions: "Literal ${NO_EXPANSION}\n"
  developerInstructions: ''
  store: explicit-store
  cpus: 1.5
  memoryBytes: 2147483648
  pids: 192
  tmpBytes: 2097152
  network: ''
`, engine, fixtureImage)
}

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

// Obtain opaque inputs from the actual closed static Runtime consumer.
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
	return consumeProfileSection(t, inline, instructions, port, "claudeCode")
}

func consumeProfileSection(t *testing.T, inline string, instructions []project.SourceFile, port runtimeconfig.AgentProfilePreparer, section string) runtimeconfig.Result {
	t.Helper()
	document := "apiVersion: markitect.runtime/v1\nagentProfiles:\n  library-agent:\n    " + section + ":\n"
	for _, line := range strings.Split(strings.TrimSuffix(inline, "\n"), "\n") {
		document += "      " + line + "\n"
	}
	document += "checkProfiles: {}\n"
	location := project.SourceLocation{Boundary: filepath.Join(t.TempDir(), "runtime"), Path: testPath(t, "profiles.yaml")}
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

type fakeImages struct {
	data                []byte
	calls               []string
	ids                 []string
	inspectErr, readErr error
	env                 []string
	etc                 []string
	envErr, etcErr      error
}

func (f *fakeImages) ImageID(_ context.Context, ref string) (string, error) {
	f.calls = append(f.calls, "inspect:"+ref)
	if len(f.ids) > 0 {
		id := f.ids[0]
		f.ids = f.ids[1:]
		return id, f.inspectErr
	}
	return fixtureImage, f.inspectErr
}
func (f *fakeImages) ReadImageFile(_ context.Context, id, path string) ([]byte, error) {
	f.calls = append(f.calls, "read:"+id+":"+path)
	return f.data, f.readErr
}
func (f *fakeImages) ImageEnvironment(_ context.Context, id string) ([]string, error) {
	f.calls = append(f.calls, "env:"+id)
	if f.env == nil {
		return []string{"PATH=/usr/local/bin:/usr/bin", "HOME=/home/agent", "NODE_VERSION=22"}, f.envErr
	}
	return f.env, f.envErr
}
func (f *fakeImages) ImageDirectoryNames(_ context.Context, id, dir string, _ int64) ([]string, error) {
	f.calls = append(f.calls, "list:"+id+":"+dir)
	if f.etc == nil {
		return []string{"etc", "etc/hosts", "etc/ssl", "etc/ssl/certs"}, f.etcErr
	}
	return f.etc, f.etcErr
}
func imagesPreparer(f *fakeImages) *MetadataPreparer {
	return NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { return f, nil })
}

func preparedProfile(t *testing.T, inline string, port runtimeconfig.AgentProfilePreparer) runtimeconfig.PreparedProfile {
	t.Helper()
	profiles := consumeProfile(t, inline, nil, port).Profiles
	if len(profiles) != 1 {
		t.Fatalf("expected one actual Runtime profile, got %d", len(profiles))
	}
	return profiles[0]
}
