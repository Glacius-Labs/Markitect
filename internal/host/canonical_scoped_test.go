package host

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/projectionengine"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func scopedTestGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	c := exec.Command("git", append([]string{"-c", "safe.directory=" + filepath.ToSlash(root), "-C", root}, args...)...)
	c.Env = append(source.CleanGitEnv(), "GIT_AUTHOR_DATE=2000-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2000-01-01T00:00:00Z")
	out, err := c.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}
func scopedTestWrite(t *testing.T, root, name, text string) {
	t.Helper()
	p := filepath.Join(root, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(text), 0644); err != nil {
		t.Fatal(err)
	}
}
func scopedWriteFixture(t *testing.T) (string, canonicalScopedWriteCapture) {
	t.Helper()
	root := t.TempDir()
	scopedTestGit(t, root, "init", "--template=", "--object-format=sha1", "-b", "codex/scoped-test")
	scopedTestGit(t, root, "config", "user.name", "Scoped test")
	scopedTestGit(t, root, "config", "user.email", "scoped@example.invalid")
	scopedTestGit(t, root, "config", "core.autocrlf", "false")
	scopedTestWrite(t, root, ".gitattributes", "* -text\n")
	scopedTestWrite(t, root, "canonical.yaml", "fixed canonical input\n")
	scopedTestWrite(t, root, "out/current.txt", "old\n")
	scopedTestWrite(t, root, "unrelated/Billing.txt", "opaque unrelated bytes\n")
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "fixed input")
	revision := scopedTestGit(t, root, "rev-parse", "HEAD")
	observed, err := source.ObserveSelectedWorking(root, []string{"canonical.yaml", "out/current.txt"})
	if err != nil {
		t.Fatal(err)
	}
	inventory, err := source.InventoryWorkingRoots(root, []string{"out"})
	if err != nil {
		t.Fatal(err)
	}
	return root, canonicalScopedWriteCapture{Revision: revision, CanonicalPaths: []string{"canonical.yaml"}, Observed: observed.Snapshot, Inventory: inventory}
}

func TestCanonicalScopedWriterBindsOnlyExplicitBytesAndInventory(t *testing.T) {
	root, capture := scopedWriteFixture(t)
	// Changes to excluded unrelated bytes are not inferred semantic inputs.
	scopedTestWrite(t, root, "unrelated/Billing.txt", "unrelated changed\n")
	paths, err := writeCanonicalScopedOutputs(root, capture, map[string][]byte{"out/current.txt": []byte("new\n"), "out/new.txt": []byte("created\n")})
	if err != nil || len(paths) != 2 {
		t.Fatalf("scoped apply: paths=%v err=%v", paths, err)
	}
	if head := scopedTestGit(t, root, "rev-parse", "HEAD"); head != capture.Revision {
		t.Fatal("apply changed HEAD")
	}
	if data, _ := os.ReadFile(filepath.Join(root, "canonical.yaml")); !bytes.Equal(data, capture.Observed.Files["canonical.yaml"]) {
		t.Fatal("canonical input changed")
	}
}

func TestCanonicalScopedWriterRejectsStaleScopeBeforeMutation(t *testing.T) {
	for _, mutation := range []string{"canonical", "target", "inventory", "head", "protected", "outside"} {
		t.Run(mutation, func(t *testing.T) {
			root, capture := scopedWriteFixture(t)
			output := map[string][]byte{"out/current.txt": []byte("candidate\n")}
			switch mutation {
			case "canonical":
				scopedTestWrite(t, root, "canonical.yaml", "changed")
			case "target":
				scopedTestWrite(t, root, "out/current.txt", "changed")
			case "inventory":
				scopedTestWrite(t, root, "out/unknown.txt", "unknown")
			case "head":
				scopedTestGit(t, root, "commit", "--allow-empty", "-m", "head changed")
			case "protected":
				output["canonical.yaml"] = []byte("illegal")
			case "outside":
				output["outside.txt"] = []byte("illegal")
			}
			old, _ := os.ReadFile(filepath.Join(root, "out/current.txt"))
			written, err := writeCanonicalScopedOutputs(root, capture, output)
			if err == nil || len(written) != 0 {
				t.Fatalf("stale %s accepted: %v %v", mutation, written, err)
			}
			current, _ := os.ReadFile(filepath.Join(root, "out/current.txt"))
			if !bytes.Equal(current, old) {
				t.Fatal("refusal modified target")
			}
		})
	}
}

func TestCanonicalAgentContextExplainsBoundedReferenceInclusion(t *testing.T) {
	fixed, _ := reconciliationFixture(t)
	requests, err := projectionRequestIndex(fixed)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range requests {
		context, err := BuildCanonicalAgentContext(fixed.Model, request, 1)
		if err != nil {
			t.Fatal(err)
		}
		if context.RequestDigest != request.RequestDigest || len(context.ScopeIDs) != len(request.Definitions) {
			t.Fatal("context lost exact selected scope")
		}
		if len(context.Inclusions) != len(context.Definitions)+len(context.RelatedDefinitions) {
			t.Fatal("missing inclusion provenance")
		}
		for _, inclusion := range context.Inclusions {
			if inclusion.Reason == "declared-outgoing-reference-context-only" && inclusion.Via == nil {
				t.Fatal("related inclusion has no edge")
			}
		}
		if _, err := BuildCanonicalAgentContext(fixed.Model, request, 3); err == nil {
			t.Fatal("unbounded context accepted")
		}
		request.ModelDigest = "sha256:" + strings.Repeat("0", 64)
		if _, err := BuildCanonicalAgentContext(fixed.Model, request, 0); err == nil {
			t.Fatal("stale request accepted")
		}
	}
}

func TestCanonicalScopedWriterRequiresProvisionalEvidence(t *testing.T) {
	root, capture := scopedWriteFixture(t)
	capture.Observed = &snapshot.Snapshot{ID: capture.Revision, Files: capture.Observed.Files, Modes: capture.Observed.Modes}
	if _, err := writeCanonicalScopedOutputs(root, capture, map[string][]byte{}); err == nil {
		t.Fatal("immutable observation accepted for write")
	}
}

func scopedCanonicalFixture(t *testing.T) (string, string, *CanonicalSource) {
	t.Helper()
	repo, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	scopedTestGit(t, root, "init", "--template=", "--object-format=sha1", "-b", "codex/scoped-canonical")
	scopedTestGit(t, root, "config", "user.name", "Scoped canonical test")
	scopedTestGit(t, root, "config", "user.email", "scoped@example.invalid")
	scopedTestGit(t, root, "config", "core.autocrlf", "false")
	copyScopedPublicFixture(t, repo, "examples/canonical-projection", root)
	scopedTestWrite(t, root, ".gitattributes", "* -text\n")
	addScopedMarkdownPolicies(t, root)
	scopedTestWrite(t, root, "unrelated/Billing/note.txt", "unique content outside source and declared targets\n")
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "fixed canonical source")
	revision := scopedTestGit(t, root, "rev-parse", "HEAD")
	fixed, err := LoadSelectedCanonicalSource(root, revision, "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatal(err)
	}
	return root, revision, fixed
}

// Copy the checked-in public fixture bytes into a new repository. The selected
// loader then reads only that repository's immutable commit, never the
// enclosing checkout's Git metadata or HEAD.
func copyScopedPublicFixture(t *testing.T, sourceRoot, fixtureDirectory, targetRoot string) {
	t.Helper()
	sourceDirectory := filepath.Join(sourceRoot, filepath.FromSlash(fixtureDirectory))
	targetDirectory := filepath.Join(targetRoot, filepath.FromSlash(fixtureDirectory))
	err := filepath.WalkDir(sourceDirectory, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceDirectory, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetDirectory, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		if !entry.Type().IsRegular() {
			return fmt.Errorf("public fixture contains unsupported file type: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		data = bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
		data = bytes.ReplaceAll(data, []byte("\r"), []byte("\n"))
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestScopedCanonicalPlannerKeepsSkippedTargetsUnobserved(t *testing.T) {
	root, revision, fixed := scopedCanonicalFixture(t)
	observed := &snapshot.Snapshot{ID: "working-tree", Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	// These synthetic unverified owners test planner boundaries, not convergence.
	active := reconciliationRecords(t, fixed, observed)
	for _, record := range active {
		for _, artifact := range record.Artifacts {
			scopedTestWrite(t, root, artifact.Path, string(observed.Files[artifact.Path]))
		}
	}
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "supplied unverified target specimens")
	base := scopedTestGit(t, root, "rev-parse", "HEAD")
	name := "examples/canonical-projection/definitions/create-order.projection-policy.yaml"
	data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(name)))
	if err != nil {
		t.Fatal(err)
	}
	data = bytes.Replace(data, []byte("purpose:"), []byte("purpose: Explicit singleton extension;"), 1)
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(name)), data, 0644); err != nil {
		t.Fatal(err)
	}
	scopedTestGit(t, root, "add", name)
	scopedTestGit(t, root, "commit", "-m", "canonical policy delta")
	candidate := scopedTestGit(t, root, "rev-parse", "HEAD")
	plan, err := ProposeScopedCanonicalReconciliation(root, base, candidate, "examples/canonical-projection/canonical.yaml", active, false)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "planned" || len(plan.Proposals) != 1 || plan.Proposals[0].Task == nil || plan.Proposals[0].Decision != "work" {
		t.Fatalf("bounded proposal: %+v", plan)
	}
	if len(plan.UnobservedProjections) != 1 || len(plan.EvidenceRefreshRequired) != 1 {
		t.Fatalf("skipped target freshness not separate: %+v", plan)
	}
	for _, path := range plan.ObservedPaths {
		if strings.HasPrefix(path, "docs/represented/") || strings.HasPrefix(path, "unrelated/") {
			t.Fatalf("unrelated byte path acquired: %s", path)
		}
	}
	second, err := ProposeScopedCanonicalReconciliation(root, base, candidate, "examples/canonical-projection/canonical.yaml", active, false)
	if err != nil || second.Digest != plan.Digest {
		t.Fatalf("non-deterministic scoped plan: %v", err)
	}
	if revision == candidate {
		t.Fatal("test failed to create real canonical evolution")
	}
	// Metadata-only UNKNOWN remains visible even in an otherwise skipped root.
	scopedTestWrite(t, root, "docs/represented/unowned.md", "unknown content")
	unknown, err := ProposeScopedCanonicalReconciliation(root, base, candidate, "examples/canonical-projection/canonical.yaml", active, false)
	if err != nil || unknown.Status != "escalated" || len(unknown.UnknownArtifacts) != 1 {
		t.Fatalf("unknown metadata vanished: %+v %v", unknown, err)
	}
	for _, path := range unknown.ObservedPaths {
		if path == "docs/represented/unowned.md" {
			t.Fatal("metadata-only unrelated unknown was read as content")
		}
	}
}

func TestScopedCanonicalPlannerNewScopesAreModuleOwned(t *testing.T) {
	root, revision, _ := scopedCanonicalFixture(t)
	plan, err := ProposeScopedCanonicalReconciliation(root, revision, revision, "examples/canonical-projection/canonical.yaml", nil, false)
	if err != nil || plan.Status != "planned" || len(plan.Proposals) != 2 {
		t.Fatalf("new module proposals: %+v %v", plan, err)
	}
	for _, proposal := range plan.Proposals {
		if proposal.Decision != "work" {
			t.Fatal("new target did not propose work")
		}
		if proposal.Task != nil {
			if proposal.Task.TargetPrefix == "" || len(proposal.Task.AllowedExtensions) == 0 {
				t.Fatal("Executor task lacks target boundary")
			}
		} else if len(proposal.Outputs) != 1 {
			t.Fatal("deterministic Module did not own its exact output")
		}
	}
}

// The new temporary scope supplies its own explicit representation policies.
// Historical example/package bytes are never rewritten by this test.
func addScopedMarkdownPolicies(t *testing.T, root string) {
	t.Helper()
	config := "examples/canonical-projection/canonical.yaml"
	source, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(config)))
	if err != nil {
		t.Fatal(err)
	}
	projection := "examples/canonical-projection/definitions/commerce.markdown-projection.yaml"
	projectionBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(projection)))
	if err != nil {
		t.Fatal(err)
	}
	refs := "  policies:\n"
	addedPaths := ""
	for _, kind := range []string{"UseCase", "Handler", "EffectAxis"} {
		name := strings.ToLower(kind) + "-markdown"
		file := "examples/canonical-projection/definitions/" + name + ".yaml"
		addedPaths += "  - " + file + "\n"
		refs += "    - apiVersion: markitect.foundation/v1\n      kind: ProjectionPolicy\n      namespace: commerce\n      name: " + name + "\n"
		scopedTestWrite(t, root, file, "apiVersion: markitect.foundation/v1\nkind: ProjectionPolicy\nmetadata:\n  namespace: commerce\n  name: "+name+"\npurpose: Document the selected canonical meaning with explicit source provenance.\nspec:\n  sourceKind:\n    apiVersion: commerce.example.org/v1\n    kind: "+kind+"\n  targetTechnology: markdown\n  guidance: Preserve canonical purpose and declared relationships; generated prose has no canonical authority.\n")
	}
	source = bytes.Replace(source, []byte("modules:\n"), []byte(addedPaths+"modules:\n"), 1)
	projectionBytes = bytes.Replace(projectionBytes, []byte("  representation: markdown\n"), []byte(refs+"  representation: markdown\n"), 1)
	scopedTestWrite(t, root, config, string(source))
	scopedTestWrite(t, root, projection, string(projectionBytes))
}

func TestCanonicalScopedWriterRejectsRepositorySubstitution(t *testing.T) {
	root, capture := scopedWriteFixture(t)
	other, _ := scopedWriteFixture(t)
	// Equal deterministic commits/content are insufficient to identify a location.
	if scopedTestGit(t, other, "rev-parse", "HEAD") != capture.Revision {
		t.Fatal("fixture revision is not deterministic")
	}
	paths, err := writeCanonicalScopedOutputs(other, capture, map[string][]byte{"out/current.txt": []byte("wrong clone")})
	if err == nil || len(paths) != 0 {
		t.Fatalf("substitution accepted: %v %v", paths, err)
	}
	data, _ := os.ReadFile(filepath.Join(root, "out/current.txt"))
	if string(data) != "old\n" {
		t.Fatal("source changed")
	}
	data, _ = os.ReadFile(filepath.Join(other, "out/current.txt"))
	if string(data) != "old\n" {
		t.Fatal("other clone changed")
	}
}

func TestScopedCanonicalPlannerSchedulesMetadataVisibleDeletion(t *testing.T) {
	root, _, fixed := scopedCanonicalFixture(t)
	observed := &snapshot.Snapshot{ID: "working-tree", Provisional: true, Files: map[string][]byte{}, Modes: map[string]string{}}
	active := reconciliationRecords(t, fixed, observed)
	for _, r := range active {
		for _, a := range r.Artifacts {
			scopedTestWrite(t, root, a.Path, string(observed.Files[a.Path]))
		}
	}
	scopedTestGit(t, root, "add", ".")
	scopedTestGit(t, root, "commit", "-m", "targets")
	revision := scopedTestGit(t, root, "rev-parse", "HEAD")
	// A record is a supplied unverified owner only; bind this temporary source
	// revision so the deletion, rather than a canonical delta, schedules the work.
	current, err := LoadSelectedCanonicalSource(root, revision, "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range active {
		r.Revision = revision
		r.ModelDigest = current.Model.Digest
		active[i], err = records.NewProjectionRecord(r)
		if err != nil {
			t.Fatal(err)
		}
	}
	var deleted string
	for _, r := range active {
		if r.Module.Name == "markitect-dotnet" {
			deleted = r.Artifacts[0].Path
		}
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(deleted))); err != nil {
		t.Fatal(err)
	}
	plan, err := ProposeScopedCanonicalReconciliation(root, revision, revision, "examples/canonical-projection/canonical.yaml", active, false)
	if err != nil || len(plan.Proposals) != 1 || plan.Proposals[0].Decision != "work" || plan.Proposals[0].Reasons[0] != "projection-drift" {
		t.Fatalf("missing artifact suppressed: %+v %v", plan, err)
	}
}

func TestCanonicalAgentContextRejectsAlteredCanonicalAuthority(t *testing.T) {
	fixed, _ := reconciliationFixture(t)
	requests, err := projectionRequestIndex(fixed)
	if err != nil {
		t.Fatal(err)
	}
	for _, original := range requests {
		if len(original.Policies) == 0 {
			continue
		}
		for _, mutation := range []string{"policy", "projection", "scope", "target", "digest"} {
			t.Run(mutation, func(t *testing.T) {
				data, _ := json.Marshal(original)
				var request canonical.ProjectionRequest
				if err := json.Unmarshal(data, &request); err != nil {
					t.Fatal(err)
				}
				switch mutation {
				case "policy":
					request.Policies[0].Spec["guidance"] = "Forged canonical authority"
				case "projection":
					request.Projection.Purpose = "Forged scope authority"
				case "scope":
					request.Definitions = request.Definitions[:1]
				case "target":
					request.TargetPrefix = "elsewhere"
				case "digest":
					request.RequestDigest = "sha256:" + strings.Repeat("0", 64)
				}
				if _, err := BuildCanonicalAgentContext(fixed.Model, request, 1); err == nil {
					t.Fatal("altered authority accepted")
				}
			})
		}
		return
	}
	t.Fatal("fixture has no policy")
}

func TestScopedCanonicalMarkdownMatchesDirectPreparation(t *testing.T) {
	root, revision, _ := scopedCanonicalFixture(t)
	plan, err := ProposeScopedCanonicalReconciliation(root, revision, revision, "examples/canonical-projection/canonical.yaml", nil, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range plan.Proposals {
		if p.Task != nil {
			continue
		}
		prepared, err := PrepareCanonicalProjection(plan.fixed, plan.observed, p.Request.Projection.Identity(), "test", sha256Prefix(sha256Hex([]byte("test tool"))), nil, plan.fixed.Config.Checks...)
		if err != nil || prepared.Plan == nil {
			t.Fatalf("direct prepare: %v", err)
		}
		if outputDigest(prepared.Outputs) != outputDigest(p.Outputs) {
			t.Fatal("Module proposal and direct Prepare render different Markdown")
		}
		for _, content := range p.Outputs {
			if !bytes.Contains(content, []byte("## Projection policies")) {
				t.Fatal("projection policies omitted")
			}
		}
		return
	}
	t.Fatal("Markdown proposal missing")
}

func TestCanonicalRecordBindsMaterializedModeRatherThanDriftedPreimage(t *testing.T) {
	fixed, observed := reconciliationFixture(t) // Supplied fixed identity tests pure record construction only.
	requests, err := projectionRequestIndex(fixed)
	if err != nil {
		t.Fatal(err)
	}
	for _, request := range requests {
		if request.Projector.Target != "markdown" {
			continue
		}
		name := "docs/represented/index.md"
		observed.Files[name] = []byte("unchanged bytes\n")
		observed.Modes[name] = snapshot.ExecutableMode
		prepared := PreparedCanonicalProjection{
			Request: request,
			Outputs: map[string][]byte{name: append([]byte(nil), observed.Files[name]...)},
			Plan:    &projectionengine.Plan{PlanDigest: strings.Repeat("d", 64)},
		}
		record, err := buildCanonicalProjectionRecord(prepared, observed, []string{name}, records.StateMaterializedUnverified)
		if err != nil {
			t.Fatal(err)
		}
		if len(record.Artifacts) != 1 || record.Artifacts[0].Mode != snapshot.RegularMode || record.Artifacts[0].Change != records.ChangeModified {
			t.Fatalf("record retained drifted preimage mode: %+v", record)
		}
		return
	}
	t.Fatal("fixture lacks Markdown Projection")
}
