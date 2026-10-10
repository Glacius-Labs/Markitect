package projectapp

import (
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

func TestResolutionChoicesRequireEveryHumanDecision(t *testing.T) {
	complete := ResolutionChoices{Actor: "user", AuthorityClaim: "Owner", DecisionReference: "d-1", Questions: []projectadoption.QuestionResolution{}, Scopes: []projectadoption.ScopeResolution{}}
	if err := complete.validate(); err != nil {
		t.Fatalf("complete choices rejected: %v", err)
	}
	for name, choices := range map[string]ResolutionChoices{
		"actor":     {AuthorityClaim: "Owner", DecisionReference: "d", Questions: complete.Questions, Scopes: complete.Scopes},
		"authority": {Actor: "user", DecisionReference: "d", Questions: complete.Questions, Scopes: complete.Scopes},
		"reference": {Actor: "user", AuthorityClaim: "Owner", Questions: complete.Questions, Scopes: complete.Scopes},
		"questions": {Actor: "user", AuthorityClaim: "Owner", DecisionReference: "d", Scopes: complete.Scopes},
		"scopes":    {Actor: "user", AuthorityClaim: "Owner", DecisionReference: "d", Questions: complete.Questions},
	} {
		if err := choices.validate(); err == nil {
			t.Errorf("choices without %s accepted", name)
		}
	}
}

// The Host derives every binding of the sealed Resolution; the caller supplies
// only human decisions.
func TestBuildResolutionBindsTheExactTargetFromHumanChoices(t *testing.T) {
	repo := copyProjectWorld(t)
	commit := gitOutput(t, repo, "rev-parse", "HEAD")
	target, err := projectwork.Load(repo, commit)
	if err != nil {
		t.Fatal(err)
	}
	targetContext, err := projectadoption.TargetContextForProject(target)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := projectadoption.Discover(repo, projectadoption.DiscoveryRequest{
		APIVersion: projectadoption.DiscoveryVersion, ID: "shop-discovery", Purpose: "Resolve selected cancellation evidence",
		Review: "review-1", Commit: commit, ScopeRoots: []string{"docs"},
		Selected:   []projectadoption.SelectedPath{{ID: "intent", Path: "docs/cancellation.md", Reason: "Read the cancellation contract", Basis: "documentation"}},
		Exclusions: []projectadoption.PathReason{}, Unselected: []projectadoption.PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	schemaDigest, buildDigest, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	evidence := discovery.Evidence[0]
	lines := strings.Split(strings.ReplaceAll(evidence.Content, "\r\n", "\n"), "\n")
	if len(lines) < 3 {
		t.Fatalf("fixture evidence unexpectedly short: %q", evidence.Content)
	}
	report := projectadoption.Distillation{
		APIVersion: projectadoption.DistillationVersion, DiscoveryDigest: discovery.Digest, Method: "human-review", SchemaDigest: schemaDigest,
		TargetBasis: target.Digest, TargetRevision: target.Revision, TargetContextDigest: targetContext.Digest,
		Claims: []projectadoption.Claim{{
			ID: "documented-cancellation", ScopeID: "orders", Kind: "documented-intent", Method: "documentation",
			Statement:   "The selected documentation defines when cancellation is allowed.",
			Evidence:    []projectadoption.EvidenceRef{{EvidenceID: evidence.ID, StartLine: 3, EndLine: 3, Excerpt: lines[2]}},
			Uncertainty: []string{},
		}},
		Terms: []projectadoption.Term{}, Contradictions: []projectadoption.Contradiction{}, Questions: []projectadoption.Question{},
		Scopes: []projectadoption.ScopeProposal{{ID: "orders", Name: "Order cancellation", ClaimIDs: []string{"documented-cancellation"}}},
		Proposal: projectadoption.ModelProposal{Goal: "Represent selected order cancellation scope", Files: []projectadoption.ProposedFile{{
			ScopeID: "orders", Path: ".markitect/model/commerce/sales/orders/cancellation.yaml",
			Content: "apiVersion: project.markitect.example.org/v1alpha1\nkind: Statement\nmetadata:\n  name: cancellation\n  namespace: commerce.sales.orders\nspec:\n  category: rule\n  description: Cancel only confirmed orders before shipment.\n",
		}}},
	}
	projectadoption.SealDistillation(&report)
	choices := ResolutionChoices{
		Actor: "user", AuthorityClaim: "Shop owner selected the order scope", DecisionReference: "shop-decision-17",
		Questions: []projectadoption.QuestionResolution{},
		Scopes:    []projectadoption.ScopeResolution{{ScopeID: "orders", Status: "adopt", Reason: "The owner selected this scope for review"}},
	}
	resolution, err := buildResolution(discovery, report, target, choices)
	if err != nil {
		t.Fatalf("build resolution: %v", err)
	}
	if resolution.TargetBasis != target.Digest || resolution.SchemaDigest != schemaDigest || resolution.BuildDigest != buildDigest ||
		resolution.ProposalDigest != projectadoption.ProposalDigest(report.Proposal) || resolution.Authenticated == nil || *resolution.Authenticated {
		t.Fatalf("resolution bindings do not match active Host values: %+v", resolution)
	}
	if _, err := os.Stat(filepath.Join(repo, ".markitect", "model", "commerce", "sales", "orders", "cancellation.yaml")); !os.IsNotExist(err) {
		t.Fatalf("resolution wrote a model proposal to disk: %v", err)
	}

	wrongActor := choices
	wrongActor.Actor = "nobody"
	if _, err := buildResolution(discovery, report, target, wrongActor); err == nil || !strings.Contains(err.Error(), "actor must be user") {
		t.Fatalf("unknown actor accepted: %v", err)
	}
	wrongTarget := report
	wrongTarget.TargetBasis = strings.Repeat("0", 64)
	projectadoption.SealDistillation(&wrongTarget)
	if _, err := buildResolution(discovery, wrongTarget, target, choices); err == nil || !strings.Contains(err.Error(), "distillation target binding does not match") {
		t.Fatalf("report grounded in another target accepted: %v", err)
	}
	provisional, err := projectwork.Load(repo, "")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := buildResolution(discovery, report, provisional, choices); err == nil {
		t.Fatal("provisional target accepted")
	}
}

func copyProjectWorld(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate project fixture test")
	}
	sourceRoot := filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "examples", "project-world")
	destination := t.TempDir()
	err := filepath.WalkDir(sourceRoot, func(sourcePath string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceRoot, sourcePath)
		if err != nil || relative == "." {
			return err
		}
		target := filepath.Join(destination, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(sourcePath)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	if err != nil {
		t.Fatal(err)
	}
	runGit(t, destination, "init", "--initial-branch=feature-project")
	runGit(t, destination, "-c", "user.email=test@example.invalid", "-c", "user.name=Test", "add", ".")
	runGit(t, destination, "-c", "user.email=test@example.invalid", "-c", "user.name=Test", "commit", "-m", "fixture")
	return destination
}
