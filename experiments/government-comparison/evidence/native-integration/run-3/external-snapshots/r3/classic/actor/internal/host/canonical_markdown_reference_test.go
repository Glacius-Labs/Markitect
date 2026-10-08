package host

import (
	"bytes"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func TestCanonicalMarkdownModuleReplacementPreservesMeaningAndUsesOwnLifecycle(t *testing.T) {
	root, _ := canonicalWorkflowFixtureRepository(t)
	_, filename, _, _ := runtime.Caller(0)
	checkout := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	if err := copyCanonicalFixtureTree(filepath.Join(checkout, "examples", "module-replacement"), filepath.Join(root, "examples", "module-replacement")); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "independent representation package")
	revision := scopedTestGit(t, root, "rev-parse", "HEAD")
	projection := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "example", Name: "workflow-markdown"}
	var originalDigest, originalOutput string
	for _, variant := range []string{"original", "replacement"} {
		fixed, err := LoadSelectedCanonicalSource(root, revision, "examples/module-replacement/"+variant+".yaml", true)
		if err != nil || len(fixed.Diagnostics) != 0 {
			t.Fatalf("%s: %v diagnostics %v", variant, err, fixed)
		}
		observed := &snapshot.Snapshot{ID: "working", Provisional: true, Files: cloneByteMap(fixed.Snapshot.Files), Modes: fixed.Snapshot.Modes}
		prepared, err := PrepareCanonicalProjection(fixed, observed, projection, "replacement-test/1", sha256Prefix(sha256Hex([]byte("replacement-test"))), nil)
		if err != nil || len(prepared.Escalations) != 0 || prepared.Plan == nil || len(prepared.Outputs) != 1 {
			t.Fatalf("%s preparation: %v / %#v", variant, err, prepared)
		}
		var outputPath string
		var content []byte
		for name, data := range prepared.Outputs {
			outputPath, content = name, data
			if prepared.OutputModes[name] != snapshot.RegularMode {
				t.Fatal("reference representation lost its regular-file mode")
			}
		}
		for _, definition := range prepared.Request.Definitions {
			if !bytes.Contains(content, []byte(definition.Metadata.Name)) {
				t.Fatalf("%s omitted %s", variant, definition.Identity().Key())
			}
		}
		if variant == "original" {
			originalDigest, originalOutput = fixed.Model.Digest, string(content)
			continue
		}
		if fixed.Model.Digest != originalDigest || string(content) == originalOutput || !strings.HasSuffix(outputPath, "/canonical-projection.md") {
			t.Fatal("replacement changed canonical meaning or reused the original representation")
		}
		proposal, err := proposeMarkdownReferenceProjection(prepared.Request, nil, observed, nil, false)
		if err != nil || proposal.Decision != "work" || !bytes.Equal(proposal.Outputs[outputPath], content) {
			t.Fatalf("initial Module proposal: %v / %#v", err, proposal)
		}
		inventory := []source.WorkingFileMetadata{{Path: outputPath, Mode: snapshot.RegularMode}}
		observed.Files[outputPath] = content
		proposal, err = proposeMarkdownReferenceProjection(prepared.Request, inventory, observed, nil, false)
		if err != nil || proposal.Decision != "escalate" {
			t.Fatal("existing unowned output was not refused")
		}
		prior := &records.ProjectionRecord{State: records.StateMaterializedUnverified, RequestDigest: prepared.Request.RequestDigest, Artifacts: []records.Artifact{{Path: outputPath, Mode: snapshot.RegularMode, Digest: sha256Prefix(sha256Hex(content))}}}
		proposal, err = proposeMarkdownReferenceProjection(prepared.Request, inventory, observed, prior, false)
		if err != nil || proposal.Decision != "no-op" || !proposal.EvidenceRefreshRequired || len(proposal.Outputs) != 0 {
			t.Fatalf("retained representation must require fresh evidence: %v / %#v", err, proposal)
		}
		observed.Files[outputPath] = []byte("drift")
		proposal, err = proposeMarkdownReferenceProjection(prepared.Request, inventory, observed, prior, false)
		if err != nil || proposal.Decision != "work" || !bytes.Equal(proposal.Outputs[outputPath], content) {
			t.Fatalf("Module drift repair: %v / %#v", err, proposal)
		}
		delete(observed.Files, outputPath)
		if _, err := prepareMarkdownReferenceInput(prepared.Request, inventory, observed, prior, false); err == nil {
			t.Fatal("inventory without observed bytes was trusted")
		}
	}
}
