package codexappserver

import (
	"context"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func executionRuntimeProfile(engine string) string {
	return strings.Replace(imageProfile(engine), "profileVersion: 2", "profileVersion: 3", 1) + `credentialRef: exact-runtime-credential
maxWireBytes: 1048576
session:
  version: 1
  model: explicitly-selected-model
  effort: high
  baseInstructions: "Literal ${NO_EXPANSION}\n"
  developerInstructions: ''
  store: explicit-store
  cpus: 1.5
  memoryBytes: 2147483648
  pids: 192
  tmpBytes: 2097152
  network: ''
`
}

func preparedExecution(t *testing.T, inline string, port runtimeconfig.AgentProfilePreparer) runtimeconfig.PreparedProfile {
	t.Helper()
	profiles := consumeProfile(t, inline, nil, port).Profiles
	if len(profiles) != 1 {
		t.Fatalf("expected one actual Runtime profile, got %d", len(profiles))
	}
	return profiles[0]
}

func TestExecutionProfileProjectsOnlyCapturedLiteralSettings(t *testing.T) {
	inline := executionRuntimeProfile(filepath.Join(t.TempDir(), "engine"))
	location := project.SourceLocation{Boundary: "captured", Path: testPath(t, "instruction.md")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("Render this separately, once.\n"))
	if err != nil {
		t.Fatal(err)
	}
	inline = strings.Replace(inline, "instructions: []", "instructions: [{boundary: captured, path: instruction.md}, {boundary: captured, path: instruction.md}]", 1)
	fake := &fakeImages{data: []byte(fixtureMetadata)}
	profiles := consumeProfile(t, inline, []project.SourceFile{{Location: location, File: entry}}, imagesPreparer(fake)).Profiles
	if len(profiles) != 1 {
		t.Fatal(profiles)
	}
	prepared := profiles[0]
	if prepared.Observation().Contract.Version != 3 || prepared.Observation().Status != model.Passed || len(prepared.Observation().Instructions) != 2 {
		t.Fatal(prepared.Observation())
	}
	want := SessionProfile{Version: 1, Model: "explicitly-selected-model", Effort: "high", BaseInstructions: "Literal ${NO_EXPANSION}\n", DeveloperInstructions: "", Store: "explicit-store", CPUs: 1.5, MemoryBytes: 2147483648, Pids: 192, TmpBytes: 2097152, Network: ""}
	calls := append([]string(nil), fake.calls...)
	profile, err := ProfileFromPrepared(prepared)
	if err != nil || !reflect.DeepEqual(profile, Profile{ImageID: fixtureImage, CredentialRef: "exact-runtime-credential", MaxWireBytes: 1 << 20, Session: want}) {
		t.Fatalf("projection: %v %+v", err, profile)
	}
	// Projection has no runtime/resolver seam and never revisits the fake reader.
	if !reflect.DeepEqual(fake.calls, calls) {
		t.Fatal("projection repeated acquisition", fake.calls)
	}
}

func TestExecutionProfileSourceChangesInvalidateStaticBinding(t *testing.T) {
	inline := executionRuntimeProfile(filepath.Join(t.TempDir(), "engine"))
	boundary := filepath.Join(t.TempDir(), "same-runtime")
	prepare := func(text string, port runtimeconfig.AgentProfilePreparer) runtimeconfig.PreparedProfile {
		return consumeProfileAt(t, text, nil, port, boundary).Profiles[0]
	}
	first := prepare(inline, imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}))
	changed := strings.Replace(inline, "effort: high", "effort: low", 1)
	second := prepare(changed, imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}))
	firstID, _ := first.Identity()
	secondID, ok := second.Identity()
	if !ok || first.Input().Location() != second.Input().Location() || firstID == secondID || first.Input().Binding() == second.Input().Binding() {
		t.Fatal("changed literal settings lost source binding")
	}
	cached := prepare(changed, captureProfile(func(context.Context, runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
		return first.Observation(), nil
	}))
	if _, err := ProfileFromPrepared(cached); err == nil {
		t.Fatal("accepted another source request's metadata")
	}
}

func TestExecutionProfileRefusesMetadataOnlyAndIncompleteOrForeignEvidence(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	v2 := preparedExecution(t, imageProfile(engine), imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}))
	if _, err := ProfileFromPrepared(v2); err == nil || !strings.Contains(err.Error(), "metadata-only") {
		t.Fatal("v2 invented execution settings", err)
	}
	if _, err := ProfileFromPrepared(runtimeconfig.PreparedProfile{}); err == nil {
		t.Fatal("accepted missing prepared profile")
	}
	for _, scenario := range []string{"contract", "provider", "image", "CLI", "agentd", "schema", "instructions", "incomplete"} {
		t.Run(scenario, func(t *testing.T) {
			port := captureProfile(func(ctx context.Context, input runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
				o, err := imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}).PrepareAgentProfile(ctx, input)
				switch scenario {
				case "contract":
					o.Contract.Version = 1
				case "provider":
					o.AgentImage.Provider = runtimeconfig.AgentImageClaudeCode
					o.AgentImage.RecordedCodexSchemaDigest = ""
				case "image":
					o.AgentImage.ImageID = "sha256:" + strings.Repeat("c", 64)
				case "CLI":
					o.AgentImage.CLIVersion = "codex-cli another-version"
				case "agentd":
					o.AgentImage.AgentdVersion = "another-build"
				case "schema":
					o.AgentImage.RecordedCodexSchemaDigest = "sha256:" + strings.Repeat("c", 64)
				case "instructions":
					o.Instructions = []project.SourceLocation{{Boundary: "missing", Path: testPath(t, "missing.md")}}
				case "incomplete":
					o.Status = model.NotCompleted
				}
				return o, err
			})
			prepared := preparedExecution(t, executionRuntimeProfile(engine), port)
			if profile, err := ProfileFromPrepared(prepared); err == nil || profile != (Profile{}) {
				t.Fatalf("accepted %s evidence or leaked partial projection: %v", scenario, err)
			}
		})
	}
}

func TestExecutionProfileInvalidSettingsRefuseBeforeImageAcquisition(t *testing.T) {
	base := executionRuntimeProfile(filepath.Join(t.TempDir(), "engine"))
	for _, tc := range []struct{ name, from, to string }{
		{"missing credential", "credentialRef: exact-runtime-credential\n", ""},
		{"missing model", "  model: explicitly-selected-model\n", ""},
		{"missing instruction policy", "  developerInstructions: ''\n", ""},
		{"invalid effort", "effort: high", "effort: max"},
		{"wire too large", "maxWireBytes: 1048576", "maxWireBytes: 1048577"},
		{"wire zero", "maxWireBytes: 1048576", "maxWireBytes: 0"},
		{"wrong numeric type", "pids: 192", "pids: '192'"},
		{"invalid resources", "cpus: 1.5", "cpus: .nan"},
		{"unknown session field", "session:\n", "session:\n  extra: forbidden\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			port := NewMetadataPreparer(func(EngineConfig) (ImageReader, error) {
				calls++
				return &fakeImages{data: []byte(fixtureMetadata)}, nil
			})
			prepared := preparedExecution(t, strings.Replace(base, tc.from, tc.to, 1), port)
			if prepared.Observation().Status != model.Failed || calls != 0 {
				t.Fatal("invalid execution settings reached image reader", calls, prepared.Observation())
			}
			if _, err := ProfileFromPrepared(prepared); err == nil {
				t.Fatal("projected invalid execution settings")
			}
		})
	}
}
