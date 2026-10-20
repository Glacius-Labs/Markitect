package claudecode

import (
	"context"
	"math"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const executionSettings = `credentialRef: exact-runtime-credential
maxWireBytes: 1048576
session:
  version: 1
  model: explicitly-selected-model
  effort: high
  systemPrompt: "Literal ${NO_EXPANSION}\n"
  builtinTools: [Read, Write]
  store: explicit-store
  cpus: 1.5
  memoryBytes: 2147483648
  pids: 192
  tmpBytes: 2097152
  network: ''
`

func executionRuntimeProfile(engine string) string { return imageProfile(engine) }

func preparedExecution(t *testing.T, inline string, port apply.AgentProfilePreparer) apply.PreparedProfile {
	t.Helper()
	profiles := consumeProfile(t, inline, nil, port).ProfileObservations()
	if len(profiles) != 1 {
		t.Fatalf("expected one actual Apply profile, got %d", len(profiles))
	}
	return profiles[0]
}

func TestExecutionProfileProjectsOnlyCapturedLiteralSettings(t *testing.T) {
	inline := executionRuntimeProfile(filepath.Join(t.TempDir(), "engine"))
	location := apply.SourceLocation{Boundary: "captured", Path: testPath(t, "instruction.md")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("Render this separately, once.\n"))
	if err != nil {
		t.Fatal(err)
	}
	inline = strings.Replace(inline, "instructions: []", "instructions: [{boundary: captured, path: instruction.md}, {boundary: captured, path: instruction.md}]", 1)
	fake := &fakeImages{data: []byte(fixtureMetadata)}
	profiles := consumeProfile(t, inline, []apply.SourceFile{{Location: location, File: entry}}, imagesPreparer(fake)).ProfileObservations()
	if len(profiles) != 1 {
		t.Fatal(profiles)
	}
	prepared := profiles[0]
	if prepared.Observation().Contract.Version != 1 || prepared.Observation().Status != model.Passed || len(prepared.Observation().Instructions) != 2 {
		t.Fatal(prepared.Observation())
	}
	want := Profile{ImageID: fixtureImage, CredentialRef: "exact-runtime-credential", MaxWireBytes: 1 << 20, Session: SessionProfile{Version: 1, Model: "explicitly-selected-model", Effort: "high", SystemPrompt: "Literal ${NO_EXPANSION}\n", BuiltinTools: []string{"Read", "Write"}, Store: "explicit-store", CPUs: 1.5, MemoryBytes: 2147483648, Pids: 192, TmpBytes: 2097152, Network: ""}}
	calls := append([]string(nil), fake.calls...)
	got, err := ProfileFromPrepared(prepared)
	if err != nil || !reflect.DeepEqual(got, want) {
		t.Fatalf("projection: %v %+v", err, got)
	}
	// Projection has no runtime/resolver seam and never revisits the fake reader.
	if !reflect.DeepEqual(fake.calls, calls) {
		t.Fatal("projection repeated acquisition", fake.calls)
	}
	got.Session.BuiltinTools[0] = "changed"
	again, err := ProfileFromPrepared(prepared)
	if err != nil || !reflect.DeepEqual(again, want) {
		t.Fatal("returned values changed captured settings", err)
	}
}

func TestExecutionProfileSourceChangesInvalidateStaticBinding(t *testing.T) {
	inline := executionRuntimeProfile(filepath.Join(t.TempDir(), "engine"))
	boundary := filepath.Join(t.TempDir(), "same-runtime")
	prepare := func(text string, port apply.AgentProfilePreparer) apply.PreparedProfile {
		return consumeProfileAt(t, text, nil, port, boundary).ProfileObservations()[0]
	}
	first := prepare(inline, imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}))
	changed := strings.Replace(inline, "explicitly-selected-model", "another-selected-model", 1)
	second := prepare(changed, imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}))
	firstID, _ := first.Identity()
	secondID, ok := second.Identity()
	if !ok || first.Input().Location() != second.Input().Location() || firstID == secondID || first.Input().Binding() == second.Input().Binding() {
		t.Fatal("changed literal settings lost source binding")
	}
	cached := prepare(changed, captureProfile(func(context.Context, apply.ProfileInput) (apply.ProfileObservation, error) {
		return first.Observation(), nil
	}))
	if _, err := ProfileFromPrepared(cached); err == nil {
		t.Fatal("accepted another source request's metadata")
	}
}

func TestExecutionProfileRefusesIncompleteOrForeignEvidence(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	if _, err := ProfileFromPrepared(apply.PreparedProfile{}); err == nil {
		t.Fatal("accepted missing prepared profile")
	}
	for _, scenario := range []string{"contract", "provider", "image", "CLI", "agentd", "schema", "instructions", "incomplete", "binding"} {
		t.Run(scenario, func(t *testing.T) {
			port := captureProfile(func(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
				o, err := imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)}).PrepareAgentProfile(ctx, input)
				switch scenario {
				case "contract":
					o.Contract.Version = 3
				case "provider":
					o.AgentImage.Provider = apply.AgentImageCodexAppServer
					o.AgentImage.RecordedCodexSchemaDigest = fixtureSchema
				case "image":
					o.AgentImage.ImageID = "sha256:" + strings.Repeat("c", 64)
				case "CLI":
					o.AgentImage.CLIVersion = "2.1.300 (Claude Code)"
				case "agentd":
					o.AgentImage.AgentdVersion = "another-build"
				case "schema":
					o.AgentImage.RecordedCodexSchemaDigest = "sha256:" + strings.Repeat("c", 64)
				case "instructions":
					o.Instructions = []apply.SourceLocation{{Boundary: "missing", Path: testPath(t, "missing.md")}}
				case "binding":
					o.InputBinding = apply.ProfileBinding{}
				case "incomplete":
					o.Status = model.NotCompleted
				}
				return o, err
			})
			prepared := preparedExecution(t, executionRuntimeProfile(engine), port)
			if got, err := ProfileFromPrepared(prepared); err == nil || !reflect.DeepEqual(got, Profile{}) {
				t.Fatalf("accepted %s evidence or leaked partial projection: %v", scenario, err)
			}
		})
	}
}

func TestExecutionProfileInvalidSettingsRefuseBeforeImageAcquisition(t *testing.T) {
	base := executionRuntimeProfile(filepath.Join(t.TempDir(), "engine"))
	for _, tc := range []struct{ name, from, to string }{
		{"missing credential", "credentialRef: exact-runtime-credential\n", ""},
		{"metadata only", executionSettings, ""},
		{"missing model", "  model: explicitly-selected-model\n", ""},
		{"missing tools", "  builtinTools: [Read, Write]\n", ""},
		{"duplicate tools", "builtinTools: [Read, Write]", "builtinTools: [Read, Read]"},
		{"unknown tool", "builtinTools: [Read, Write]", "builtinTools: [Task]"},
		{"unsupported runtime version", "profileVersion: 1", "profileVersion: 3"},
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

func TestProfileProjectsAnyDistinctBuiltinSubsetAndEffort(t *testing.T) {
	for _, tc := range []struct {
		tools  string
		effort string
		want   []string
	}{
		{"[Read, Write, Edit, Glob, Grep, Bash]", "low", []string{"Read", "Write", "Edit", "Glob", "Grep", "Bash"}},
		{"[Bash, Grep]", "medium", []string{"Bash", "Grep"}},
		{"[]", "high", []string{}},
	} {
		t.Run(tc.tools, func(t *testing.T) {
			inline := strings.Replace(imageProfile(filepath.Join(t.TempDir(), "engine")), "builtinTools: [Read, Write]", "builtinTools: "+tc.tools, 1)
			inline = strings.Replace(inline, "effort: high", "effort: "+tc.effort, 1)
			got, err := ProfileFromPrepared(preparedExecution(t, inline, imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)})))
			if err != nil || !reflect.DeepEqual(got.Session.BuiltinTools, tc.want) || got.Session.Effort != tc.effort {
				t.Fatalf("got %+v: %v", got.Session, err)
			}
		})
	}
}

func validProfile() Profile {
	return Profile{ImageID: fixtureImage, CredentialRef: "ref", MaxWireBytes: 1 << 20, Session: SessionProfile{Version: 1, Model: "m", Effort: "high",
		SystemPrompt: "Use the Runtime instruction.", BuiltinTools: []string{"Read", "Write", "Edit"}, Store: "store", CPUs: 1, MemoryBytes: 1 << 30, Pids: 128, TmpBytes: 1 << 20}}
}

func TestProfileValidationIsStrict(t *testing.T) {
	if err := validProfile().validate(); err != nil {
		t.Fatal(err)
	}
	for name, mutate := range map[string]func(*Profile){
		"image tag":       func(p *Profile) { p.ImageID = "runtime:latest" },
		"credential":      func(p *Profile) { p.CredentialRef = "" },
		"wire zero":       func(p *Profile) { p.MaxWireBytes = 0 },
		"wire large":      func(p *Profile) { p.MaxWireBytes = maxLineBytes + 1 },
		"version":         func(p *Profile) { p.Session.Version = 0 },
		"model":           func(p *Profile) { p.Session.Model = " " },
		"model utf8":      func(p *Profile) { p.Session.Model = string([]byte{0xff}) },
		"effort":          func(p *Profile) { p.Session.Effort = "max" },
		"effort empty":    func(p *Profile) { p.Session.Effort = "" },
		"system prompt":   func(p *Profile) { p.Session.SystemPrompt = "\t" },
		"absent builtins": func(p *Profile) { p.Session.BuiltinTools = nil },
		"unknown builtin": func(p *Profile) { p.Session.BuiltinTools = []string{"WebFetch"} },
		"duplicate":       func(p *Profile) { p.Session.BuiltinTools = []string{"Bash", "Bash"} },
		"store":           func(p *Profile) { p.Session.Store = "" },
		"cpus":            func(p *Profile) { p.Session.CPUs = math.NaN() },
		"memory":          func(p *Profile) { p.Session.MemoryBytes = 0 },
		"pids":            func(p *Profile) { p.Session.Pids = 0 },
		"tmp":             func(p *Profile) { p.Session.TmpBytes = 0 },
		"host network":    func(p *Profile) { p.Session.Network = "host" },
		"network name":    func(p *Profile) { p.Session.Network = "-bad" },
	} {
		t.Run(name, func(t *testing.T) {
			p := validProfile()
			mutate(&p)
			if err := p.validate(); err == nil {
				t.Fatal("accepted")
			}
		})
	}
}

func TestClaudeSectionIsRequiredBeforeAcquisitionAndProjection(t *testing.T) {
	inline := imageProfile(filepath.Join(t.TempDir(), "engine"))
	calls := 0
	reader := NewMetadataPreparer(func(EngineConfig) (ImageReader, error) {
		calls++
		return &fakeImages{data: []byte(fixtureMetadata)}, nil
	})
	var foreign apply.ProfileInput
	result := consumeProfileSection(t, inline, nil, captureProfile(func(ctx context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		foreign = input
		return reader.PrepareAgentProfile(ctx, input)
	}), filepath.Join(t.TempDir(), "runtime"), "codexAppServer")
	if calls != 0 || len(result.ProfileObservations()) != 1 || result.ProfileObservations()[0].Observation().Status != model.Failed {
		t.Fatal("foreign section reached image reader", calls)
	}
	if foreign.Section() != "codexAppServer" {
		t.Fatal("fixture did not select foreign section")
	}
	// A foreign adapter may produce internally complete static evidence, but
	// cannot pass Claude's explicit section and contract checks during projection.
	result = consumeProfileSection(t, inline, nil, captureProfile(func(_ context.Context, input apply.ProfileInput) (apply.ProfileObservation, error) {
		return apply.ProfileObservation{InputBinding: input.Binding(), Status: model.Passed,
			Contract: apply.ProfileContract{Name: "markitect.claude-code.image-metadata", Version: 1},
			AgentImage: &apply.AgentImageObservation{Provider: apply.AgentImageCodexAppServer, ImageID: fixtureImage,
				Metadata:   apply.AgentImageMetadataFile{Path: runtimeMetadataPath, ContentID: snapshot.ContentID{1}, Size: 1},
				CLIVersion: supportedClaudeVersion, AgentdVersion: "development", RecordedCodexSchemaDigest: fixtureSchema}}, nil
	}), filepath.Join(t.TempDir(), "runtime"), "codexAppServer")
	prepared := result.ProfileObservations()[0]
	if _, ok := prepared.Identity(); !ok {
		t.Fatal("expected complete foreign evidence", prepared.Observation())
	}
	if _, err := ProfileFromPrepared(prepared); err == nil {
		t.Fatal("projected a foreign section")
	}
}
