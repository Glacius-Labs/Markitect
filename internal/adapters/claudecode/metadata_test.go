package claudecode

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const fixtureSchema = "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
const fixtureImage = "sha256:bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
const fixtureMetadata = "codex=codex-cli 0.162.0\nclaude=2.1.301 (Claude Code)\ncodexSchema=" + fixtureSchema + "\nagentd=development\n"

func imageProfile(engine string) string {
	return fmt.Sprintf("profileVersion: 1\nengineExecutable: %q\nimageID: %s\nagentdVersion: development\ninstructions: []\n", engine, fixtureImage) + executionSettings
}

type fakeImages struct {
	data                []byte
	calls               []string
	ids                 []string
	inspectErr, readErr error
	onRead              func()
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
	if f.onRead != nil {
		f.onRead()
	}
	return f.data, f.readErr
}
func imagesPreparer(f *fakeImages) *MetadataPreparer {
	return NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { return f, nil })
}

func TestImageMetadataUsesSelectedConfigurationAndWholeActualBytes(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	input := profileInput(t, imageProfile(engine), nil)
	fake := &fakeImages{data: []byte(fixtureMetadata)}
	var config EngineConfig
	port := NewMetadataPreparer(func(c EngineConfig) (ImageReader, error) { config = c; return fake, nil })
	out, err := port.PrepareAgentProfile(context.Background(), input)
	if err != nil || out.Status != model.Passed {
		t.Fatalf("%v %+v", err, out)
	}
	want := &apply.AgentImageObservation{Provider: apply.AgentImageClaudeCode, ImageID: fixtureImage,
		Metadata:   apply.AgentImageMetadataFile{Path: runtimeMetadataPath, ContentID: snapshot.ContentID(sha256.Sum256([]byte(fixtureMetadata))), Size: int64(len(fixtureMetadata))},
		CLIVersion: supportedClaudeVersion, AgentdVersion: "development"}
	if config.Executable != engine || !reflect.DeepEqual(out.AgentImage, want) || out.InputBinding != input.Binding() || out.Contract != (apply.ProfileContract{Name: "markitect.claude-code.image-metadata", Version: 1}) || len(out.Files) != 0 || len(out.Inventories) != 0 {
		t.Fatalf("configuration/binding/evidence: %+v %+v", config, out)
	}
	wantCalls := []string{"inspect:" + fixtureImage, "read:" + fixtureImage + ":" + runtimeMetadataPath, "inspect:" + fixtureImage}
	if !reflect.DeepEqual(fake.calls, wantCalls) {
		t.Fatal(fake.calls)
	}
}

func TestApplyRetainsImageEvidenceAndInvalidatesChangedWholeMetadata(t *testing.T) {
	inline := imageProfile(filepath.Join(t.TempDir(), "engine"))
	boundary := filepath.Join(t.TempDir(), "same-runtime")
	identity := func(data string) string {
		prepared := consumeProfileAt(t, inline, nil, imagesPreparer(&fakeImages{data: []byte(data)}), boundary)
		profiles := prepared.ProfileObservations()
		if len(profiles) != 1 || profiles[0].Observation().Status != model.Passed {
			t.Fatalf("%+v", profiles)
		}
		id, ok := profiles[0].Identity()
		if !ok {
			t.Fatal("actual Apply validator did not accept complete image evidence")
		}
		return id
	}
	first := identity(fixtureMetadata)
	if first != identity(fixtureMetadata) {
		t.Fatal("identical source and metadata did not retain identity")
	}
	if first == identity(strings.Replace(fixtureMetadata, "0.162.0", "0.162.1", 1)) {
		t.Fatal("other actual metadata bytes were omitted from identity")
	}
}

func TestMalformedProfilesNeverConstructAnImageReader(t *testing.T) {
	base := imageProfile(filepath.Join(t.TempDir(), "engine"))
	for _, tc := range []struct{ name, from, to string }{
		{"unsupported version", "profileVersion: 1", "profileVersion: 3"},
		{"string version", "profileVersion: 1", "profileVersion: '1'"},
		{"image tag", fixtureImage, "runtime:latest"},
		{"uppercase image", fixtureImage, "sha256:" + strings.Repeat("B", 64)},
		{"missing field", "agentdVersion: development\n", ""},
		{"empty version", "agentdVersion: development", "agentdVersion: ''"},
		{"version control", "agentdVersion: development", `agentdVersion: "dev\nother"`},
		{"unknown field", "instructions: []", "instructions: []\nmodel: ungranted"},
		{"old executable", "instructions: []", "instructions: []\nexecutable: provider"},
		{"wrong list", "instructions: []", "instructions: {}"},
		{"bad instruction", "instructions: []", "instructions: [{boundary: captured, path: ../escape}]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := profileInput(t, strings.Replace(base, tc.from, tc.to, 1), nil)
			calls := 0
			out, err := NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { calls++; return nil, nil }).PrepareAgentProfile(context.Background(), input)
			if err != nil || out.Status != model.Failed || calls != 0 || out.AgentImage != nil {
				t.Fatalf("%v %+v calls=%d", err, out, calls)
			}
			for _, d := range out.Diagnostics {
				if d.Source.ContentID != input.Document().File.ContentID() || !strings.HasPrefix(d.Source.Field, input.Field()) {
					t.Fatal("diagnostic lost source binding", d)
				}
			}
		})
	}
}

func TestAcquisitionErrorsAndCancellationRetainOnlyObservedImageEvidence(t *testing.T) {
	input := profileInput(t, imageProfile(filepath.Join(t.TempDir(), "engine")), nil)
	boom := errors.New("controlled acquisition failure")
	for _, phase := range []string{"nil factory", "factory error", "nil reader", "inspect error", "read error", "cancel before factory", "cancel after read", "recheck error"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			fake := &fakeImages{data: []byte(fixtureMetadata)}
			calls := 0
			factory := ImageReaderFactory(func(EngineConfig) (ImageReader, error) { calls++; return fake, nil })
			switch phase {
			case "nil factory":
				factory = nil
			case "factory error":
				factory = func(EngineConfig) (ImageReader, error) { return nil, boom }
			case "nil reader":
				factory = func(EngineConfig) (ImageReader, error) { return nil, nil }
			case "inspect error":
				fake.inspectErr = boom
			case "read error":
				fake.readErr = boom
			case "cancel before factory":
				cancel()
			case "cancel after read":
				fake.onRead = cancel
			case "recheck error":
				fake.onRead = func() { fake.inspectErr = boom }
			}
			out, err := NewMetadataPreparer(factory).PrepareAgentProfile(ctx, input)
			if err == nil || out.Status != model.NotCompleted {
				t.Fatalf("%v %+v", err, out)
			}
			read := phase == "read error" || phase == "cancel after read" || phase == "recheck error"
			if (out.AgentImage != nil) != read {
				t.Fatal("unobserved image or lost observed image", out)
			}
			if read && (out.AgentImage.Metadata.ContentID != (snapshot.ContentID{})) != (phase != "read error") {
				t.Fatal("partial bytes were called complete or actual whole bytes lost", out)
			}
			if phase == "cancel before factory" && calls != 0 {
				t.Fatal("canceled request constructed reader")
			}
		})
	}
}

func TestImageMismatchNeverReadsAndChangeAcrossAcquisitionRefuses(t *testing.T) {
	input := profileInput(t, imageProfile(filepath.Join(t.TempDir(), "engine")), nil)
	for _, phase := range []string{"first", "recheck"} {
		t.Run(phase, func(t *testing.T) {
			other := "sha256:" + strings.Repeat("c", 64)
			f := &fakeImages{data: []byte(fixtureMetadata), ids: []string{other}}
			if phase == "recheck" {
				f.ids = []string{fixtureImage, other}
			}
			out, err := imagesPreparer(f).PrepareAgentProfile(context.Background(), input)
			if err != nil || out.Status != model.Failed || out.AgentImage == nil {
				t.Fatalf("%v %+v", err, out)
			}
			if phase == "first" && len(f.calls) != 1 {
				t.Fatal("read metadata from an unverified identity", f.calls)
			}
			if phase == "recheck" && out.AgentImage.Metadata.ContentID == (snapshot.ContentID{}) {
				t.Fatal("lost completed metadata observation")
			}
		})
	}
}

func TestConcreteFactoryRequiresExplicitEnginePath(t *testing.T) {
	for _, name := range []string{"", "docker", t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		if _, err := NewEngineImageReader(EngineConfig{Executable: name}); err == nil {
			t.Fatalf("accepted invalid engine %q", name)
		}
	}
	name := filepath.Join(t.TempDir(), "regular-not-executed")
	if err := os.WriteFile(name, []byte("not an executable"), 0600); err != nil {
		t.Fatal(err)
	}
	if reader, err := NewEngineImageReader(EngineConfig{Executable: name}); err != nil || reader == nil {
		t.Fatal("factory should bind without executing", err)
	}
}
