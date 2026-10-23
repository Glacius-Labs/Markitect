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
	"go.yaml.in/yaml/v3"
)

func TestRuntimeMetadataClosedFormatAndSupportedRecordedValues(t *testing.T) {
	input := profileInput(t, imageProfile(filepath.Join(t.TempDir(), "engine")), nil)
	for _, tc := range []struct{ name, data string }{
		{"old CLI", strings.Replace(fixtureMetadata, "2.1.301", "2.1.300", 1)},
		{"different agentd", strings.Replace(fixtureMetadata, "agentd=development", "agentd=another-build", 1)},
		{"missing", strings.Replace(fixtureMetadata, "claude=2.1.301 (Claude Code)\n", "", 1)},
		{"duplicate", fixtureMetadata + "codex=codex-cli 0.162.0\n"},
		{"unknown", fixtureMetadata + "ungranted=do-not-execute\n"},
		{"shell syntax", strings.Replace(fixtureMetadata, "agentd=development", "export agentd=development", 1)},
		{"blank value", strings.Replace(fixtureMetadata, "agentd=development", "agentd=", 1)},
		{"control", strings.Replace(fixtureMetadata, "development", "dev\x00other", 1)},
		{"CRLF", strings.ReplaceAll(fixtureMetadata, "\n", "\r\n")},
		{"extra blank record", fixtureMetadata + "\n"},
		{"empty", ""},
		{"invalid UTF8", strings.Replace(fixtureMetadata, "development", "\xff", 1)},
		{"unbounded literal", strings.Replace(fixtureMetadata, "development", strings.Repeat("v", 4097), 1)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out, err := imagesPreparer(&fakeImages{data: []byte(tc.data)}).PrepareAgentProfile(context.Background(), input)
			if err != nil || out.Status != model.Failed || out.AgentImage == nil || out.AgentImage.Metadata.Size != int64(len(tc.data)) || out.AgentImage.Metadata.ContentID == (snapshot.ContentID{}) {
				t.Fatalf("lost actual bytes or accepted unsupported metadata: %v %+v", err, out)
			}
		})
	}
	out, err := imagesPreparer(&fakeImages{data: make([]byte, maxMetadataBytes+1)}).PrepareAgentProfile(context.Background(), input)
	if err != nil || out.Status != model.NotCompleted || out.AgentImage == nil {
		t.Fatalf("metadata bound: %v %+v", err, out)
	}
}

func TestDecoderRefusesDuplicatesAliasesAndRelativeEngineBeforeAcquisition(t *testing.T) {
	base := imageProfile(filepath.Join(t.TempDir(), "engine"))
	for _, text := range []string{
		base + "imageID: " + fixtureImage + "\n",
		strings.Replace(base, strings.Split(base, "\n")[1], "engineExecutable: docker", 1),
		strings.Replace(base, "instructions: []", "instructions: [{boundary: captured, path: instruction.md, extra: true}]", 1),
		strings.Replace(base, "instructions: []", "instructions: [{boundary: captured, path: instruction.md, path: other.md}]", 1),
		strings.Replace(strings.Replace(base, "engineExecutable: ", "engineExecutable: &engine ", 1), "agentdVersion: development", "agentdVersion: *engine", 1),
	} {
		var document yaml.Node
		if err := yaml.Unmarshal([]byte(text), &document); err != nil {
			t.Fatal(err)
		}
		p := preparation{out: runtimeconfig.ProfileObservation{Status: model.Passed}}
		if _, ok := p.decode(document.Content[0]); ok || p.out.Status != model.Failed {
			t.Fatal("closed decoder accepted ambiguous input", text)
		}
	}
}

func TestInstructionsUseCapturedInputsOnlyAndRetainMissingRequest(t *testing.T) {
	path := testPath(t, "instruction.md")
	location := project.SourceLocation{Boundary: "explicit-captured-instruction", Path: path}
	entry, err := snapshot.NewEntry(path, snapshot.Regular, []byte("captured instructions only\n"))
	if err != nil {
		t.Fatal(err)
	}
	inline := strings.Replace(imageProfile(filepath.Join(t.TempDir(), "engine")), "instructions: []", "instructions: [{boundary: explicit-captured-instruction, path: instruction.md}]", 1)
	for _, captured := range []bool{false, true} {
		var instructions []project.SourceFile
		if captured {
			instructions = []project.SourceFile{{Location: location, File: entry}}
		}
		input := profileInput(t, inline, instructions)
		f := &fakeImages{data: []byte(fixtureMetadata)}
		out, err := imagesPreparer(f).PrepareAgentProfile(context.Background(), input)
		if err != nil {
			t.Fatal(err)
		}
		if captured {
			if out.Status != model.Passed || len(out.Instructions) != 1 || out.Instructions[0] != location || len(out.RequiredInstructions) != 0 {
				t.Fatalf("%+v", out)
			}
		} else if out.Status != model.NotCompleted || len(out.RequiredInstructions) != 1 || out.RequiredInstructions[0] != location || len(out.Instructions) != 0 {
			t.Fatalf("%+v", out)
		}
		if out.AgentImage == nil || len(f.calls) != 3 {
			t.Fatal("lost independent actual metadata observation", out)
		}
	}
}
