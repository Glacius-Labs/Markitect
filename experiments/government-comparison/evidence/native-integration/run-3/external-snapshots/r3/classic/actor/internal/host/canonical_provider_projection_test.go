package host

import (
	"bytes"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func TestCanonicalWorkflowProviderProjectionsShareOnlyCanonicalInputs(t *testing.T) {
	root, revision := canonicalWorkflowFixtureRepository(t)
	fixed, err := LoadSelectedCanonicalSource(root, revision, "examples/canonical-workflow/canonical.yaml", true)
	if err != nil {
		t.Fatalf("canonical workflow: %v", err)
	}
	if len(fixed.Diagnostics) != 0 {
		t.Fatalf("canonical workflow diagnostics: %v", fixed.Diagnostics)
	}
	observed := &snapshot.Snapshot{ID: "working", Provisional: true, Files: cloneByteMap(fixed.Snapshot.Files), Modes: fixed.Snapshot.Modes}
	outputs := map[string][]byte{}
	for _, provider := range []string{"codex", "claude", "markdown", "githooks", "azurepipelines"} {
		id := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "example", Name: "workflow-" + provider}
		prepared, err := PrepareCanonicalProjection(fixed, observed, id, "provider-test/1", sha256Prefix(sha256Hex([]byte("provider-test"))), nil, fixed.Config.Checks...)
		if err != nil || prepared.Plan == nil || len(prepared.Escalations) != 0 {
			t.Fatalf("%s: %v %v", provider, err, prepared.Escalations)
		}
		if len(prepared.Request.Definitions) != 4 {
			t.Fatal("lost exact shared canonical scope")
		}
		for name, content := range prepared.Outputs {
			if provider == "codex" && filepath.Base(name) != "AGENTS.md" {
				t.Fatal("module did not own Codex filename")
			}
			if provider == "claude" && filepath.Base(name) != "CLAUDE.md" {
				t.Fatal("module did not own Claude filename")
			}
			wantMode := snapshot.RegularMode
			if provider == "githooks" {
				wantMode = snapshot.ExecutableMode
				if filepath.Base(name) != "pre-commit" {
					t.Fatal("Git Hooks did not own the hook filename")
				}
			}
			if provider == "azurepipelines" && filepath.Base(name) != "azure-pipelines.yml" {
				t.Fatal("Azure Pipelines did not own its pipeline filename")
			}
			if prepared.OutputModes[name] != wantMode {
				t.Fatalf("%s: mode %s, want %s", name, prepared.OutputModes[name], wantMode)
			}
			for _, d := range prepared.Request.Definitions {
				if !bytes.Contains(content, []byte(d.Metadata.Name)) {
					t.Fatalf("%s omitted canonical subject %s", provider, d.Metadata.Name)
				}
			}
			outputs[name] = content
		}
	}
	if len(outputs) != 5 {
		t.Fatalf("expected five independent Module-owned workflow artifacts, got %d", len(outputs))
	}
	// Neither target representation was supplied as input to either provider.
	for name := range observed.Files {
		if strings.HasSuffix(name, "/AGENTS.md") || strings.HasSuffix(name, "/CLAUDE.md") {
			t.Fatal("provider depended on sibling target bytes")
		}
	}
}

// canonicalWorkflowFixtureRepository creates the public example in an isolated
// repository so provider tests never read the checkout's outer .git state.
func canonicalWorkflowFixtureRepository(t *testing.T) (string, string) {
	t.Helper()
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate provider projection test source")
	}
	checkout := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	root := t.TempDir()
	for _, fixture := range []string{
		"examples/canonical-workflow",
		"examples/canonical-projection/modules/markdown",
	} {
		if err := copyCanonicalFixtureTree(filepath.Join(checkout, filepath.FromSlash(fixture)), filepath.Join(root, filepath.FromSlash(fixture))); err != nil {
			t.Fatalf("copy canonical workflow fixture %s: %v", fixture, err)
		}
	}
	if err := os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("* -text\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module example.invalid/canonical-workflow-fixture\n\ngo 1.22\n"), 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "init", "--template=", "--object-format=sha1", "-b", "codex/canonical-workflow-test")
	scopedTestGit(t, root, "config", "user.name", "Canonical workflow test")
	scopedTestGit(t, root, "config", "user.email", "canonical-workflow@example.invalid")
	scopedTestGit(t, root, "config", "core.autocrlf", "false")
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "fixed canonical workflow fixture")
	return root, scopedTestGit(t, root, "rev-parse", "HEAD")
}

func copyCanonicalFixtureTree(source, destination string) error {
	return filepath.WalkDir(source, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

// This separate adopter fixture repairs selected intent, not the original C4
// check or failed evidence. It deliberately reuses the existing public API owners.
func TestCanonicalDocumentationAlignedFixtureSelectsExistingAPIIntent(t *testing.T) {
	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate fixture source")
	}
	checkout := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
	root := t.TempDir()
	if err := copyCanonicalFixtureTree(filepath.Join(checkout, "examples", "operating-model"), filepath.Join(root, "examples", "operating-model")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitattributes"), []byte("* -text\n"), 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "init", "--template=", "--object-format=sha1", "-b", "codex/documentation-intent-test")
	scopedTestGit(t, root, "config", "user.name", "Canonical documentation test")
	scopedTestGit(t, root, "config", "user.email", "documentation@example.invalid")
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "fixed documentation intent fixtures")
	revision := scopedTestGit(t, root, "rev-parse", "HEAD")
	projection := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "operating-model", Name: "product-markdown"}
	var original *CanonicalSource
	for _, tc := range []struct {
		config   string
		subjects int
		hasAPI   bool
	}{
		{"examples/operating-model/canonical.yaml", 3, false},
		{"examples/operating-model/canonical-documentation-aligned.yaml", 5, true},
	} {
		fixed, err := LoadSelectedCanonicalSource(root, revision, tc.config, true)
		if err != nil || len(fixed.Diagnostics) != 0 {
			t.Fatalf("%s: %v %v", tc.config, err, fixed.Diagnostics)
		}
		observed := &snapshot.Snapshot{ID: "working", Provisional: true, Files: cloneByteMap(fixed.Snapshot.Files), Modes: fixed.Snapshot.Modes}
		prepared, err := PrepareCanonicalProjection(fixed, observed, projection, "fixture-test/1", sha256Prefix(sha256Hex([]byte("fixture-test"))), nil, fixed.Config.Checks...)
		if err != nil || len(prepared.Escalations) != 0 || len(prepared.Request.Definitions) != tc.subjects {
			t.Fatalf("%s: %v %v", tc.config, err, prepared.Escalations)
		}
		content, ok := prepared.Outputs["docs/represented/index.md"]
		if !ok {
			t.Fatal("Module-owned Markdown target missing")
		}
		for _, name := range []string{"TryCalculateTotalCents", "SumOutstandingCents"} {
			if bytes.Contains(content, []byte(name)) != tc.hasAPI {
				t.Fatalf("%s: unexpected API presence for %s", tc.config, name)
			}
		}
		if original == nil {
			original = fixed
			continue
		}
		if !equalCanonicalValue(original.Config.Checks, fixed.Config.Checks) || !equalCanonicalValue(original.Config.Modules, fixed.Config.Modules) {
			t.Fatal("alignment changed existing checks or exact module pins")
		}
		for _, name := range []string{"orders.order-rules.yaml", "billing.billing-query.yaml", "product.composition.yaml", "orders-dotnet.projection-policy.yaml", "billing-dotnet.projection-policy.yaml"} {
			path := "examples/operating-model/definitions/" + name
			if !bytes.Equal(original.Snapshot.Files[path], fixed.Snapshot.Files[path]) {
				t.Fatalf("alignment changed canonical owner %s", path)
			}
		}
	}
}

// Matching provider bytes prove no materialization work, not independent
// verification. The complete ownership records below intentionally have no
// VerificationResult, while the canonical source/model are unchanged.
func TestCanonicalAgentRulesNoOpStillRequiresIndependentVerification(t *testing.T) {
	root, revision := canonicalWorkflowFixtureRepository(t)
	configPath := "examples/canonical-workflow/canonical.yaml"
	fixed, err := LoadSelectedCanonicalSource(root, revision, configPath, true)
	if err != nil {
		t.Fatal(err)
	}
	observed := &snapshot.Snapshot{ID: "working", Provisional: true, Files: cloneByteMap(fixed.Snapshot.Files), Modes: fixed.Snapshot.Modes}
	active := []records.ProjectionRecord{}
	wanted := map[string]bool{}
	for _, provider := range []string{"codex", "claude"} {
		identity := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "example", Name: "workflow-" + provider}
		prepared, err := PrepareCanonicalProjection(fixed, observed, identity, "provider-test/1", sha256Prefix(sha256Hex([]byte("provider-test"))), nil, fixed.Config.Checks...)
		if err != nil {
			t.Fatalf("prepare %s: %v", provider, err)
		}
		if prepared.Plan == nil || len(prepared.Escalations) != 0 {
			t.Fatalf("prepare %s: %v", provider, prepared.Escalations)
		}
		actual := &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}
		written := []string{}
		for name, data := range prepared.Outputs {
			scopedTestWrite(t, root, name, string(data))
			actual.Files[name] = append([]byte(nil), data...)
			actual.Modes[name] = prepared.OutputModes[name]
			written = append(written, name)
		}
		record, err := buildCanonicalProjectionRecord(prepared, observed, actual, written, records.StateMaterializedUnverified)
		if err != nil {
			t.Fatal(err)
		}
		active = append(active, record)
		wanted[identity.Key()] = true
	}
	plan, err := ProposeScopedCanonicalReconciliation(root, revision, revision, configPath, active, true)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, proposal := range plan.Proposals {
		if !wanted[proposal.ProjectionID] {
			continue
		}
		seen[proposal.ProjectionID] = true
		if proposal.Decision != "no-op" {
			t.Errorf("%s matching provider bytes requested %s", proposal.ProjectionID, proposal.Decision)
		}
		if !proposal.EvidenceRefreshRequired {
			t.Errorf("%s omitted independent verification refresh", proposal.ProjectionID)
		}
		inRefreshSet := false
		for _, id := range plan.EvidenceRefreshRequired {
			inRefreshSet = inRefreshSet || id == proposal.ProjectionID
		}
		if !inRefreshSet {
			t.Errorf("%s missing from plan evidence refresh set", proposal.ProjectionID)
		}
	}
	if len(seen) != len(wanted) {
		t.Fatalf("provider proposals missing: %v", seen)
	}
}
