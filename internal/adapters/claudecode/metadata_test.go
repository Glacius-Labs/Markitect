package claudecode

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runtimeconfig"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestImageMetadataBindsTheSelectedImageAndWholeRuntimeRecord(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	input := profileInput(t, claudeProfile(engine), nil)
	fake := &fakeImages{data: []byte(fixtureMetadata)}
	var config EngineConfig
	port := NewMetadataPreparer(func(c EngineConfig) (ImageReader, error) { config = c; return fake, nil })
	out, err := port.PrepareAgentProfile(context.Background(), input)
	if err != nil || out.Status != model.Passed || config.Executable != engine {
		t.Fatalf("status=%s err=%v config=%+v diagnostics=%+v", out.Status, err, config, out.Diagnostics)
	}
	image := out.AgentImage
	want := runtimeconfig.AgentImageMetadataFile{Path: runtimeMetadataPath, ContentID: snapshot.ContentID(sha256.Sum256([]byte(fixtureMetadata))), Size: int64(len(fixtureMetadata))}
	if out.Contract != (runtimeconfig.ProfileContract{Name: metadataContract, Version: 1}) || image.Provider != runtimeconfig.AgentImageClaudeCode || image.ImageID != fixtureImage || image.Metadata != want || image.CLIVersion != supportedClaudeVersion || image.AgentdVersion != "development" || image.RecordedCodexSchemaDigest != "" {
		t.Fatalf("observation %+v %+v", out.Contract, image)
	}
	if !reflect.DeepEqual(fake.calls, []string{"inspect:" + fixtureImage, "read:" + fixtureImage + ":" + runtimeMetadataPath, "inspect:" + fixtureImage}) {
		t.Fatal("acquisition sequence", fake.calls)
	}
}

func TestRuntimeRecordRequiresTheClaudeRecordAndPinnedVersion(t *testing.T) {
	input := profileInput(t, claudeProfile(filepath.Join(t.TempDir(), "engine")), nil)
	for _, tc := range []struct{ name, data string }{
		{"Codex-only image", "codex=codex-cli 0.162.0\ncodexSchema=sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\nagentd=development\n"},
		{"other CLI version", strings.Replace(fixtureMetadata, "2.1.308", "2.1.309", 1)},
		{"different agentd", strings.Replace(fixtureMetadata, "agentd=development", "agentd=another-build", 1)},
		{"duplicate", fixtureMetadata + "claude=2.1.308 (Claude Code)\n"},
		{"unknown", fixtureMetadata + "ungranted=do-not-execute\n"},
		{"shell syntax", strings.Replace(fixtureMetadata, "agentd=development", "export agentd=development", 1)},
		{"blank value", strings.Replace(fixtureMetadata, "agentd=development", "agentd=", 1)},
		{"CRLF", strings.ReplaceAll(fixtureMetadata, "\n", "\r\n")},
		{"empty", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := imagesPreparer(&fakeImages{data: []byte(tc.data)}).PrepareAgentProfile(context.Background(), input)
			if err != nil || out.Status != model.Failed || out.AgentImage == nil || out.AgentImage.Metadata.Size != int64(len(tc.data)) {
				t.Fatalf("accepted unsupported metadata or lost its bytes: %v %+v", err, out)
			}
		})
	}
	// Claude-only records suffice for this adapter.
	only := "agentd=development\nclaude=2.1.308 (Claude Code)\n"
	if out, err := imagesPreparer(&fakeImages{data: []byte(only)}).PrepareAgentProfile(context.Background(), input); err != nil || out.Status != model.Passed {
		t.Fatalf("Claude-only metadata refused: %v %+v", err, out.Diagnostics)
	}
}

func TestMalformedProfilesNeverConstructAnImageReader(t *testing.T) {
	engine := filepath.Join(t.TempDir(), "engine")
	base := claudeProfile(engine)
	for name, inline := range map[string]string{
		"version 3":         strings.Replace(base, "profileVersion: 1", "profileVersion: 3", 1),
		"Codex schema":      base + "recordedCodexSchemaDigest: sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa\n",
		"tag image":         strings.Replace(base, fixtureImage, "markitect-runtime:latest", 1),
		"relative engine":   strings.Replace(base, fmt.Sprintf("%q", engine), `"engine"`, 1),
		"wire limit":        strings.Replace(base, "maxWireBytes: 4194304", "maxWireBytes: 9000000", 1),
		"Codex effort":      strings.Replace(base, "effort: high", "effort: minimal", 1),
		"option-like model": strings.Replace(base, "model: sonnet", "model: --dangerous", 1),
		"host network":      strings.Replace(base, "network: ''", "network: host", 1),
		"missing session":   base[:strings.Index(base, "session:")],
	} {
		t.Run(name, func(t *testing.T) {
			constructed := false
			port := NewMetadataPreparer(func(EngineConfig) (ImageReader, error) { constructed = true; return &fakeImages{}, nil })
			out, err := port.PrepareAgentProfile(context.Background(), profileInput(t, inline, nil))
			if err != nil || out.Status != model.Failed || constructed || out.AgentImage != nil {
				t.Fatalf("status=%s constructed=%v err=%v", out.Status, constructed, err)
			}
		})
	}
}

func TestAcquisitionFailuresAndImageChangesRetainOnlyObservedEvidence(t *testing.T) {
	input := profileInput(t, claudeProfile(filepath.Join(t.TempDir(), "engine")), nil)
	failure := errors.New("controlled engine failure")
	out, err := imagesPreparer(&fakeImages{inspectErr: failure}).PrepareAgentProfile(context.Background(), input)
	if !errors.Is(err, failure) || out.Status != model.NotCompleted || out.AgentImage != nil {
		t.Fatalf("inspect failure: %v %+v", err, out)
	}
	out, err = imagesPreparer(&fakeImages{data: []byte(fixtureMetadata), ids: []string{fixtureImage, "sha256:" + strings.Repeat("c", 64)}}).PrepareAgentProfile(context.Background(), input)
	if err != nil || out.Status != model.Failed {
		t.Fatalf("image change across acquisition admitted: %v %+v", err, out)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if out, err := imagesPreparer(&fakeImages{}).PrepareAgentProfile(ctx, input); err == nil || out.Status != model.NotCompleted {
		t.Fatalf("cancelled preparation: %v %+v", err, out)
	}
	if out, err := NewMetadataPreparer(nil).PrepareAgentProfile(context.Background(), input); err == nil || out.Status != model.NotCompleted {
		t.Fatal("a missing image-reader factory must report incomplete acquisition", err)
	}
}

func TestProfileProjectsOnlyCompleteClaudeEvidence(t *testing.T) {
	inline := claudeProfile(filepath.Join(t.TempDir(), "engine"))
	location := project.SourceLocation{Boundary: "captured", Path: testPath(t, "instruction.md")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("Render this separately.\n"))
	if err != nil {
		t.Fatal(err)
	}
	inline = strings.Replace(inline, "instructions: []", "instructions: [{boundary: captured, path: instruction.md}]", 1)
	profiles := consumeProfile(t, inline, []project.SourceFile{{Location: location, File: entry}}, imagesPreparer(&fakeImages{data: []byte(fixtureMetadata)})).Profiles
	if len(profiles) != 1 {
		t.Fatal(profiles)
	}
	profile, err := ProfileFromPrepared(profiles[0])
	want := Profile{ImageID: fixtureImage, CredentialRef: "exact-runtime-credential", MaxWireBytes: 4 << 20, Session: SessionProfile{Version: 1, Model: "sonnet", Effort: "high", BaseInstructions: "Literal ${NO_EXPANSION}\n", Store: "explicit-store", CPUs: 1.5, MemoryBytes: 2147483648, Pids: 192, TmpBytes: 2097152}}
	if err != nil || !reflect.DeepEqual(profile, want) {
		t.Fatalf("projection: %v %+v", err, profile)
	}
	// Incomplete or foreign evidence is refused.
	incomplete := preparedProfile(t, claudeProfile(filepath.Join(t.TempDir(), "engine")), imagesPreparer(&fakeImages{data: []byte("agentd=development\n")}))
	if _, err := ProfileFromPrepared(incomplete); err == nil {
		t.Fatal("failed preparation projected")
	}
	foreign := consumeProfileSection(t, "profileVersion: 1\n", nil, captureProfile(func(_ context.Context, p runtimeconfig.ProfileInput) (runtimeconfig.ProfileObservation, error) {
		return runtimeconfig.ProfileObservation{InputBinding: p.Binding(), Status: model.NotCompleted}, nil
	}), "codexAppServer").Profiles
	if len(foreign) != 1 {
		t.Fatal(foreign)
	}
	if _, err := ProfileFromPrepared(foreign[0]); err == nil {
		t.Fatal("a Codex section projected as Claude Code")
	}
}
