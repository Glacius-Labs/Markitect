package projectadoption

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/testkit"
)

func TestDiscoveryUsesOnlySelectedBlobsAtFullCommit(t *testing.T) {
	repo, commit := committedRepository(t, map[string]string{
		"src/orders/cancel.go": "package orders\nfunc Cancel() {}\n",
		"docs/order.md":        "Cancellation is permitted before dispatch.\n",
		"runtime/test.log":     testRuntimeRecord("package orders\nfunc Cancel() {}\n"),
		"docs/unselected.md":   "not evidence for this review\n",
	})
	root := repo.Dir
	request := DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: "discovery-1", Purpose: "Assess order cancellation model",
		Review: "review-17", Commit: commit, ScopeRoots: []string{"."},
		Selected: []SelectedPath{
			{ID: "implementation", Path: "src/orders/cancel.go", Reason: "Implementation candidate", Basis: "code"},
			{ID: "intent", Path: "docs/order.md", Reason: "Documented intent", Basis: "documentation"},
			{ID: "test-run", Path: "runtime/test.log", Reason: "Recorded test run", Basis: "runtime-record"},
		},
		Exclusions: []PathReason{{Path: "docs/drafts", Reason: "Unreviewed drafts"}},
		Unselected: []PathReason{{Path: "docs/unselected.md", Reason: "Outside current review question"}},
	}
	first, err := Discover(root, request)
	if err != nil {
		t.Fatal(err)
	}
	if first.Commit != commit || len(first.Evidence) != len(request.Selected) || first.Evidence[0].Mode != "100644" {
		t.Fatalf("discovery did not bind selected commit, paths and regular-file modes: %#v", first)
	}
	if err := os.WriteFile(filepath.Join(root, "docs", "unselected.md"), []byte("changed working tree only\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := RefreshDiscovery(root, first)
	if err != nil {
		t.Fatal(err)
	}
	if second.Digest != first.Digest {
		t.Fatal("changing an unselected working-tree file changed fixed-commit discovery")
	}
	if _, err := Discover(root, DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: "bad", Purpose: "x", Review: "r", Commit: commit,
		ScopeRoots: []string{"."}, Selected: []SelectedPath{{ID: "wide", Path: "docs", Reason: "directory", Basis: "documentation"}},
		Exclusions: []PathReason{}, Unselected: []PathReason{},
	}); err == nil {
		t.Fatal("directory selection should not expand to recursive evidence")
	}
}

func TestStrictRecordsRejectDuplicateAndUnknownFields(t *testing.T) {
	if _, err := DecodeDiscoveryRequest([]byte(`{"apiVersion":"x","apiVersion":"y"}`)); err == nil {
		t.Fatal("duplicate fields should be rejected before decoding")
	}
	if _, err := DecodeDiscoveryRequest([]byte(`{"apiVersion":"x","apiversion":"y"}`)); err == nil {
		t.Fatal("case-aliased fields should be rejected")
	}
	if _, err := DecodeDiscoveryRequest([]byte(`{"apiVersion":"x","unknown":true}`)); err == nil {
		t.Fatal("unknown fields should be rejected")
	}
}

func TestCurrentBindingsDeriveSchemaAndBuildIdentity(t *testing.T) {
	schemaDigest, buildDigest, err := CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	if !validDigest(schemaDigest) || !validDigest(buildDigest) {
		t.Fatalf("bindings must be lowercase SHA-256 digests: schema=%q build=%q", schemaDigest, buildDigest)
	}
	schema := projectmodel.Schema()
	schema.Purpose += " changed"
	changedSchemaDigest, _, err := CurrentBindings(schema)
	if err != nil {
		t.Fatal(err)
	}
	if changedSchemaDigest == schemaDigest {
		t.Fatal("changing the active schema did not change its binding")
	}
}

func TestFilesystemPathComparisonUsesDirectoryIdentity(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows path components are case-insensitive")
	}
	parent := t.TempDir()
	upper := filepath.Join(parent, "Repo")
	lower := filepath.Join(parent, "repo")
	if err := os.Mkdir(upper, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(lower, 0o755); err != nil {
		if os.IsExist(err) {
			t.Skip("filesystem treats case-variant directory names as aliases")
		}
		t.Fatal(err)
	}
	upperInfo, err := os.Stat(upper)
	if err != nil {
		t.Fatal(err)
	}
	lowerInfo, err := os.Stat(lower)
	if err != nil {
		t.Fatal(err)
	}
	if os.SameFile(upperInfo, lowerInfo) {
		t.Skip("filesystem treats case-variant directory names as aliases")
	}
	if sameFilesystemPath(upper, lower) {
		t.Fatal("distinct case-sensitive POSIX directories must not alias")
	}
	if !sameFilesystemPath(upper, filepath.Join(upper, ".")) {
		t.Fatal("the same directory through a path alias should compare equal")
	}
}

func TestDistillationPreservesContradictionsAndSeparatesEvidenceMethods(t *testing.T) {
	repo, commit := committedRepository(t, map[string]string{
		"src/orders/cancel.go": "package orders\nfunc Cancel() {}\n",
		"docs/order.md":        "Cancellation is permitted before dispatch.\n",
		"runtime/test.log":     testRuntimeRecord("package orders\nfunc Cancel() {}\n"),
	})
	root := repo.Dir
	request := DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: "discovery-2", Purpose: "Assess cancellation", Review: "review-18",
		Commit: commit, ScopeRoots: []string{"."},
		Selected: []SelectedPath{
			{ID: "implementation", Path: "src/orders/cancel.go", Reason: "Implementation candidate", Basis: "code"},
			{ID: "intent", Path: "docs/order.md", Reason: "Documented intent", Basis: "documentation"},
			{ID: "test-run", Path: "runtime/test.log", Reason: "Recorded test run", Basis: "runtime-record"},
		}, Exclusions: []PathReason{}, Unselected: []PathReason{},
	}
	discovery, err := Discover(root, request)
	if err != nil {
		t.Fatal(err)
	}
	report := validDistillation(discovery)
	if err := ValidateDistillation(discovery, report); err != nil {
		t.Fatalf("valid mixed-evidence distillation rejected: %v", err)
	}
	if len(report.Contradictions) != 1 || report.Contradictions[0].QuestionID != "clarify-cancellation" {
		t.Fatalf("contradiction was lost from validated report: %#v", report.Contradictions)
	}
	bad := report
	bad.Claims = append([]Claim(nil), report.Claims...)
	bad.Claims[0].Method = "submitted-record"
	SealDistillation(&bad)
	if err := ValidateDistillation(discovery, bad); err == nil {
		t.Fatal("static source observation must not be relabeled as runtime evidence")
	}
	bad = report
	bad.Claims = append([]Claim(nil), report.Claims...)
	bad.Claims[0].Evidence = append([]EvidenceRef(nil), report.Claims[0].Evidence...)
	bad.Claims[0].Evidence[0].StartLine = 99
	SealDistillation(&bad)
	if err := ValidateDistillation(discovery, bad); err == nil {
		t.Fatal("out-of-range line evidence should be rejected")
	}
	bad = report
	bad.Claims = append([]Claim(nil), report.Claims...)
	for i, claim := range bad.Claims {
		if claim.Kind == "submitted-runtime-record" {
			changedRuntime := *claim.Runtime
			changedRuntime.ExitCode = intValue(1)
			bad.Claims[i].Runtime = &changedRuntime
		}
	}
	SealDistillation(&bad)
	if err := ValidateDistillation(discovery, bad); err == nil {
		t.Fatal("exit code 1 must not match the structured record with exit code 0")
	}
	duplicateDiscovery := discovery
	duplicateDiscovery.Evidence = append([]Evidence(nil), discovery.Evidence...)
	var duplicateRecord string
	for i, evidence := range duplicateDiscovery.Evidence {
		if evidence.ID == "test-run" {
			duplicateRecord = strings.Replace(evidence.Content, `"exitCode":0`, `"exitCode":0,"exitCode":10`, 1)
			if duplicateRecord == evidence.Content {
				t.Fatal("test runtime record did not contain the expected numeric exit code")
			}
			duplicateDiscovery.Evidence[i].Content = duplicateRecord
			duplicateDiscovery.Evidence[i].Digest = digestBytes([]byte(duplicateRecord))
		}
	}
	SealDiscovery(&duplicateDiscovery)
	duplicateReport := report
	duplicateReport.DiscoveryDigest = duplicateDiscovery.Digest
	duplicateReport.Claims = append([]Claim(nil), report.Claims...)
	for i, claim := range duplicateReport.Claims {
		if claim.Kind == "submitted-runtime-record" {
			claim.Evidence = append([]EvidenceRef(nil), claim.Evidence...)
			claim.Evidence[0].Excerpt = strings.TrimSpace(duplicateRecord)
			duplicateReport.Claims[i] = claim
		}
	}
	SealDistillation(&duplicateReport)
	if err := ValidateDistillation(duplicateDiscovery, duplicateReport); err == nil {
		t.Fatal("duplicate structured runtime-record fields must be rejected")
	}
	sameCommitDiscovery := discovery
	sameCommitDiscovery.Evidence = append([]Evidence(nil), discovery.Evidence...)
	for i, evidence := range sameCommitDiscovery.Evidence {
		if evidence.ID == "test-run" {
			content := testRuntimeRecordAtRevision("package orders\nfunc Cancel() {}\n", discovery.Commit)
			sameCommitDiscovery.Evidence[i].Content = content
			sameCommitDiscovery.Evidence[i].Digest = digestBytes([]byte(content))
		}
	}
	SealDiscovery(&sameCommitDiscovery)
	sameCommitReport := validDistillation(sameCommitDiscovery)
	if err := ValidateDistillation(sameCommitDiscovery, sameCommitReport); err != nil {
		t.Fatalf("same-commit submitted record with selected matching inputs rejected: %v", err)
	}
	bad = sameCommitReport
	bad.Claims = append([]Claim(nil), sameCommitReport.Claims...)
	for i, claim := range bad.Claims {
		if claim.Kind == "submitted-runtime-record" {
			changedRuntime := *claim.Runtime
			changedRuntime.Inputs = append([]RuntimeInput(nil), claim.Runtime.Inputs...)
			changedRuntime.Inputs[0].Digest = strings.Repeat("b", 64)
			bad.Claims[i].Runtime = &changedRuntime
		}
	}
	SealDistillation(&bad)
	if err := ValidateDistillation(sameCommitDiscovery, bad); err == nil {
		t.Fatal("same-commit runtime input must bind the selected source digest")
	}
	bad = report
	bad.Questions = append([]Question(nil), report.Questions...)
	bad.Questions[0].ClaimIDs = []string{"source-observation", "ownership-hypothesis"}
	SealDistillation(&bad)
	if err := ValidateDistillation(discovery, bad); err == nil {
		t.Fatal("same-scope question that omits one conflicting claim must not resolve the contradiction")
	}
}

func TestResolutionRequiresExplicitPerScopeDecisionAndAnswers(t *testing.T) {
	repo, commit := committedRepository(t, map[string]string{
		"src/orders/cancel.go": "package orders\nfunc Cancel() {}\n",
		"docs/order.md":        "Cancellation is permitted before dispatch.\n",
		"runtime/test.log":     testRuntimeRecord("package orders\nfunc Cancel() {}\n"),
	})
	root := repo.Dir
	discovery, err := Discover(root, DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: "discovery-3", Purpose: "Assess cancellation", Review: "review-19",
		Commit: commit, ScopeRoots: []string{"."}, Selected: []SelectedPath{
			{ID: "implementation", Path: "src/orders/cancel.go", Reason: "Implementation candidate", Basis: "code"},
			{ID: "intent", Path: "docs/order.md", Reason: "Documented intent", Basis: "documentation"},
			{ID: "test-run", Path: "runtime/test.log", Reason: "Recorded test run", Basis: "runtime-record"},
		}, Exclusions: []PathReason{}, Unselected: []PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	report := validDistillation(discovery)
	resolution := Resolution{
		APIVersion: ResolutionVersion, DiscoveryDigest: discovery.Digest, DistillationDigest: report.Digest,
		ProposalDigest: ProposalDigest(report.Proposal), TargetBasis: strings.Repeat("b", 64),
		SchemaDigest: report.SchemaDigest, BuildDigest: strings.Repeat("c", 64), Actor: "user",
		AuthorityClaim: "Project owner for orders scope", DecisionReference: "decision-19", Authenticated: boolValue(false),
		Questions: []QuestionResolution{{QuestionID: "clarify-cancellation", ScopeID: "orders", Disposition: "answer", Answer: "Documented behavior is intended", Reason: "Owner confirmed"}},
		Scopes:    []ScopeResolution{{ScopeID: "orders", Status: "adopt", Reason: "Confirmed by project owner"}, {ScopeID: "inventory", Status: "defer", Reason: "Out of current scope"}},
	}
	SealResolution(&resolution)
	if err := ValidateResolution(discovery, report, resolution); err != nil {
		t.Fatalf("valid partial scope resolution rejected: %v", err)
	}
	resolution.Questions[0].Disposition = "defer"
	SealResolution(&resolution)
	if err := ValidateResolution(discovery, report, resolution); err == nil {
		t.Fatal("adopted selected scope must not retain an unanswered ambiguity")
	}
}

func TestPlanAndApplyAdoptOnlyResolvedModelScope(t *testing.T) {
	repo, _ := committedRepository(t, map[string]string{
		"src/orders/cancel.go": "package orders\nfunc Cancel() {}\n",
		"docs/order.md":        "Cancellation is permitted before dispatch.\n",
		"runtime/test.log":     testRuntimeRecord("package orders\nfunc Cancel() {}\n"),
	})
	root := repo.Dir
	repo.Git("checkout", "-b", "codex/project-adoption-test")
	if _, err := projectwork.Init(root, "Brownfield fixture", true); err != nil {
		t.Fatal(err)
	}
	commit := repo.Commit("initialize Markitect project")
	codeBefore, err := os.ReadFile(filepath.Join(root, "src", "orders", "cancel.go"))
	if err != nil {
		t.Fatal(err)
	}
	docsBefore, err := os.ReadFile(filepath.Join(root, "docs", "order.md"))
	if err != nil {
		t.Fatal(err)
	}
	target, err := projectwork.Load(root, commit)
	if err != nil {
		t.Fatal(err)
	}
	schemaDigest, buildDigest, err := CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	request := DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: "adopt-discovery", Purpose: "Assess order cancellation", Review: "review-20",
		Commit: commit, ScopeRoots: []string{"."},
		Selected: []SelectedPath{
			{ID: "implementation", Path: "src/orders/cancel.go", Reason: "Implementation candidate", Basis: "code"},
			{ID: "intent", Path: "docs/order.md", Reason: "Documented intent", Basis: "documentation"},
			{ID: "test-run", Path: "runtime/test.log", Reason: "Recorded test run", Basis: "runtime-record"},
		}, Exclusions: []PathReason{}, Unselected: []PathReason{},
	}
	discovery, err := Discover(root, request)
	if err != nil {
		t.Fatal(err)
	}
	report := validDistillation(discovery)
	report.SchemaDigest = schemaDigest
	targetContext, err := TargetContextForProject(target)
	if err != nil {
		t.Fatal(err)
	}
	report.TargetBasis = target.Digest
	report.TargetRevision = target.Revision
	report.TargetContextDigest = targetContext.Digest
	statement := "apiVersion: " + projectmodel.APIVersion + "\nkind: Statement\nmetadata:\n  name: cancel-order\n  namespace: orders\npurpose: Record the cancellation context.\nspec:\n  category: concept\n  description: Cancel an order before dispatch.\n"
	deferredStatement := strings.ReplaceAll(statement, "namespace: orders", "namespace: inventory")
	report.Proposal.Files[0].Content = statement
	report.Proposal.Files[0].Path = ".markitect/model/orders/statement.yaml"
	report.Proposal.Files[1].Content = deferredStatement
	report.Proposal.Files[1].Path = ".markitect/model/inventory/statement.yaml"
	SealDistillation(&report)
	resolution := Resolution{
		APIVersion: ResolutionVersion, DiscoveryDigest: discovery.Digest, DistillationDigest: report.Digest,
		ProposalDigest: ProposalDigest(report.Proposal), TargetBasis: target.Digest,
		SchemaDigest: schemaDigest, BuildDigest: buildDigest, Actor: "user",
		AuthorityClaim: "Project owner confirmed order cancellation scope", DecisionReference: "decision-20", Authenticated: boolValue(false),
		Questions: []QuestionResolution{{QuestionID: "clarify-cancellation", ScopeID: "orders", Disposition: "answer", Answer: "Documented behavior is intended", Reason: "Project owner confirmed"}},
		Scopes:    []ScopeResolution{{ScopeID: "orders", Status: "adopt", Reason: "Confirmed scope"}, {ScopeID: "inventory", Status: "defer", Reason: "Not reviewed"}},
	}
	SealResolution(&resolution)
	retargetedReport := report
	retargetedReport.TargetBasis = strings.Repeat("a", 64)
	retargetedReport.TargetRevision = strings.Repeat("b", 40)
	retargetedReport.TargetContextDigest = strings.Repeat("c", 64)
	SealDistillation(&retargetedReport)
	retargetedResolution := resolution
	retargetedResolution.DistillationDigest = retargetedReport.Digest
	SealResolution(&retargetedResolution)
	if _, err := PlanAdoption(root, target, discovery, retargetedReport, retargetedResolution, schemaDigest, buildDigest); err == nil {
		t.Fatal("adoption must reject a generated report bound to another target project")
	}
	if _, err := PlanAdoption(root, target, discovery, report, resolution, strings.Repeat("e", 64), strings.Repeat("f", 64)); err == nil {
		t.Fatal("adoption must reject caller-invented schema/build bindings")
	}
	plan, err := PlanAdoption(root, target, discovery, report, resolution, schemaDigest, buildDigest)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "partial" || len(plan.AdoptedScopes) != 1 || plan.AdoptedScopes[0] != "orders" || len(plan.DeferredScopes) != 1 || plan.DeferredScopes[0] != "inventory" {
		t.Fatalf("plan did not preserve partial scope outcome: %#v", plan)
	}
	for _, file := range plan.Edit.Mutation.Files {
		if file.Path != projectwork.ManifestPath && file.Path != ".markitect/model/orders/statement.yaml" {
			t.Fatalf("plan includes non-adopted or noncanonical output %q", file.Path)
		}
	}
	planBytes, err := EncodeAdoptionPlan(plan)
	if err != nil {
		t.Fatal(err)
	}
	decodedPlan, err := DecodeAdoptionPlan(planBytes)
	if err != nil {
		t.Fatal(err)
	}
	changedReport := report
	changedReport.Claims = append([]Claim(nil), report.Claims...)
	changedReport.Claims[0].Statement += " Changed after preview."
	SealDistillation(&changedReport)
	changedResolution := resolution
	changedResolution.DistillationDigest = changedReport.Digest
	SealResolution(&changedResolution)
	if _, err := ApplyAdoption(root, root, target, discovery, changedReport, changedResolution, decodedPlan, plan.PlanDigest, schemaDigest, buildDigest); err == nil {
		t.Fatal("apply must reject a report changed after preview")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(".markitect/model/orders/statement.yaml"))); !os.IsNotExist(err) {
		t.Fatalf("stale preview wrote adoption output: %v", err)
	}
	if _, err := ApplyAdoption(root, root, target, discovery, report, resolution, decodedPlan, plan.PlanDigest, schemaDigest, buildDigest); err != nil {
		t.Fatal(err)
	}
	codeAfter, err := os.ReadFile(filepath.Join(root, "src", "orders", "cancel.go"))
	if err != nil {
		t.Fatal(err)
	}
	docsAfter, err := os.ReadFile(filepath.Join(root, "docs", "order.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(codeAfter) != string(codeBefore) || string(docsAfter) != string(docsBefore) {
		t.Fatal("adoption changed source code or documentation")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(".markitect/model/inventory/statement.yaml"))); !os.IsNotExist(err) {
		t.Fatalf("deferred inventory scope was written: %v", err)
	}
}

func TestAdoptionValidationErrorsAreDeterministic(t *testing.T) {
	root, discovery, _, target := distillationDiscovery(t)
	schemaDigest, buildDigest, err := CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	report := multiScopeDistillation(discovery, schemaDigest, []string{"alpha", "beta"}, true, []string{"alpha", "beta"})
	if err := ValidateDistillation(discovery, report); err != nil {
		t.Fatalf("precondition: two-scope report should be valid: %v", err)
	}
	adoptAll := func(r Distillation, scopes []string, basis string) Resolution {
		resolution := Resolution{APIVersion: ResolutionVersion, DiscoveryDigest: discovery.Digest, DistillationDigest: r.Digest, ProposalDigest: ProposalDigest(r.Proposal),
			TargetBasis: basis, SchemaDigest: r.SchemaDigest, BuildDigest: buildDigest, Actor: "user", AuthorityClaim: "Project owner",
			DecisionReference: "decision-determinism", Authenticated: boolValue(false), Questions: []QuestionResolution{}, Scopes: []ScopeResolution{}}
		for _, id := range scopes {
			resolution.Scopes = append(resolution.Scopes, ScopeResolution{ScopeID: id, Status: "adopt", Reason: "Owner adopts " + id})
		}
		SealResolution(&resolution)
		return resolution
	}

	t.Run("ValidateResolution unanswered questions", func(t *testing.T) {
		resolution := adoptAll(report, []string{"alpha", "beta"}, strings.Repeat("b", 64))
		assertSameErrorEveryRun(t, 200, func() error { return ValidateResolution(discovery, report, resolution) })
	})
	t.Run("validateScopeTree unknown parents", func(t *testing.T) {
		broken := multiScopeDistillation(discovery, schemaDigest, []string{"alpha", "beta"}, false, []string{"alpha", "beta"})
		broken.Scopes[0].ParentID, broken.Scopes[1].ParentID = "missing-a", "missing-b"
		SealDistillation(&broken)
		assertSameErrorEveryRun(t, 200, func() error { return ValidateDistillation(discovery, broken) })
	})
	t.Run("ValidateDistillation scopes without claims", func(t *testing.T) {
		broken := multiScopeDistillation(discovery, schemaDigest, []string{"alpha", "beta", "gamma"}, false, []string{"alpha", "beta", "gamma"})
		broken.Claims = broken.Claims[:1]
		broken.Scopes[1].ClaimIDs, broken.Scopes[2].ClaimIDs = []string{}, []string{}
		SealDistillation(&broken)
		assertSameErrorEveryRun(t, 200, func() error { return ValidateDistillation(discovery, broken) })
	})
	t.Run("PlanAdoption adopted scopes without files", func(t *testing.T) {
		planned := multiScopeDistillation(discovery, schemaDigest, []string{"alpha", "beta", "gamma"}, false, []string{"alpha"})
		resolution := adoptAll(planned, []string{"alpha", "beta", "gamma"}, target.Digest)
		assertSameErrorEveryRun(t, 5, func() error {
			_, err := PlanAdoption(root, target, discovery, planned, resolution, schemaDigest, buildDigest)
			return err
		})
	})
}

func multiScopeDistillation(discovery Discovery, schemaDigest string, scopes []string, withQuestions bool, fileScopes []string) Distillation {
	evidence := discoveryEvidence(discovery, "implementation")
	report := Distillation{APIVersion: DistillationVersion, DiscoveryDigest: discovery.Digest, Method: "human-review", SchemaDigest: schemaDigest,
		Claims: []Claim{}, Terms: []Term{}, Contradictions: []Contradiction{}, Questions: []Question{}, Scopes: []ScopeProposal{},
		Proposal: ModelProposal{Goal: "Represent observed scopes", Files: []ProposedFile{}}}
	for _, id := range scopes {
		claimID := id + "-claim"
		report.Claims = append(report.Claims, Claim{ID: claimID, ScopeID: id, Kind: "observation", Method: "static-source", Statement: "Source declares " + id + ".",
			Evidence: []EvidenceRef{{EvidenceID: evidence.ID, StartLine: 2, EndLine: 2, Excerpt: "func Cancel() {}"}}, Uncertainty: []string{}})
		report.Scopes = append(report.Scopes, ScopeProposal{ID: id, Name: "Scope " + id, ClaimIDs: []string{claimID}})
		if withQuestions {
			report.Questions = append(report.Questions, Question{ID: id + "-question", ScopeID: id, Prompt: "Which " + id + " behavior is intended?",
				Alternatives: []string{"implementation", "documentation"}, ClaimIDs: []string{claimID}, Blocking: boolValue(false)})
		}
	}
	for _, id := range fileScopes {
		report.Proposal.Files = append(report.Proposal.Files, ProposedFile{ScopeID: id, Path: ".markitect/model/" + id + "/statement.yaml",
			Content: "apiVersion: " + projectwork.APIVersion + "\nkind: Statement\nmetadata:\n  name: " + id + "\n  namespace: " + id + "\npurpose: Scope " + id + "\nspec:\n  category: concept\n  description: Scope " + id + "\n  public: false\n  uses: []\n  requires: []\n"})
	}
	SealDistillation(&report)
	return report
}

func assertSameErrorEveryRun(t *testing.T, runs int, call func() error) {
	t.Helper()
	seen := map[string]int{}
	for i := 0; i < runs; i++ {
		err := call()
		if err == nil {
			t.Fatal("precondition: call should fail")
		}
		seen[err.Error()]++
	}
	if len(seen) != 1 {
		messages := make([]string, 0, len(seen))
		for message, count := range seen {
			messages = append(messages, fmt.Sprintf("%s (x%d)", message, count))
		}
		sort.Strings(messages)
		t.Errorf("same input produced %d distinct errors over %d runs:\n  %s", len(seen), runs, strings.Join(messages, "\n  "))
	}
}

func validDistillation(discovery Discovery) Distillation {
	code := discoveryEvidence(discovery, "implementation")
	doc := discoveryEvidence(discovery, "intent")
	run := discoveryEvidence(discovery, "test-run")
	statement := Claim{
		ID: "source-observation", ScopeID: "orders", Kind: "observation", Method: "static-source",
		Statement: "The selected source declares a cancellation function.",
		Evidence:  []EvidenceRef{{EvidenceID: code.ID, StartLine: 2, EndLine: 2, Excerpt: "func Cancel() {}"}}, Uncertainty: []string{},
	}
	intent := Claim{
		ID: "documented-intent", ScopeID: "orders", Kind: "documented-intent", Method: "documentation",
		Statement: "Documentation permits cancellation before dispatch.",
		Evidence:  []EvidenceRef{{EvidenceID: doc.ID, StartLine: 1, EndLine: 1, Excerpt: "Cancellation is permitted before dispatch."}}, Uncertainty: []string{"No runtime conformance evidence is implied."},
	}
	runtimeClaim := Claim{
		ID: "test-observation", ScopeID: "orders", Kind: "submitted-runtime-record", Method: "submitted-record",
		Statement: "The submitted runtime record reports a successful test command; execution is not authenticated.",
		Evidence:  []EvidenceRef{{EvidenceID: run.ID, StartLine: 1, EndLine: 1, Excerpt: strings.TrimSpace(run.Content)}}, Uncertainty: []string{"This does not establish production behavior."},
		Runtime: runtimeObservationFromEvidence(run, discovery.Commit),
	}
	hypothesis := Claim{
		ID: "ownership-hypothesis", ScopeID: "orders", Kind: "hypothesis", Method: "synthesis",
		Statement: "The order module may own cancellation policy.",
		Evidence:  []EvidenceRef{{EvidenceID: code.ID, StartLine: 2, EndLine: 2, Excerpt: "func Cancel() {}"}}, Uncertainty: []string{"Ownership has not been confirmed."},
	}
	inventoryHypothesis := Claim{
		ID: "inventory-hypothesis", ScopeID: "inventory", Kind: "hypothesis", Method: "synthesis",
		Statement: "Inventory may be an independent adoption scope.",
		Evidence:  []EvidenceRef{{EvidenceID: code.ID, StartLine: 1, EndLine: 1, Excerpt: "package orders"}}, Uncertainty: []string{"No inventory source was selected."},
	}
	report := Distillation{
		APIVersion: DistillationVersion, DiscoveryDigest: discovery.Digest, Method: "human-review",
		SchemaDigest: strings.Repeat("d", 64), Claims: []Claim{statement, intent, runtimeClaim, hypothesis, inventoryHypothesis},
		Terms:          []Term{{ID: "cancellation-term", Text: "Cancellation", Context: "Business term in the selected documentation", Occurrences: []TermOccurrence{{EvidenceID: doc.ID, StartLine: 1, EndLine: 1, Excerpt: "Cancellation"}}, Synonyms: []string{"Cancel"}, Ambiguities: []string{"Could describe a user action or system command."}}},
		Contradictions: []Contradiction{{ID: "implementation-intent-gap", ScopeID: "orders", ClaimIDs: []string{statement.ID, intent.ID}, QuestionID: "clarify-cancellation", Description: "Implementation evidence and documented intent do not establish the same behavior."}},
		Questions:      []Question{{ID: "clarify-cancellation", ScopeID: "orders", Prompt: "Which source should define the adopted cancellation behavior?", Alternatives: []string{"Documented intent", "Observed implementation"}, ClaimIDs: []string{statement.ID, intent.ID}, Blocking: boolValue(true)}},
		Scopes:         []ScopeProposal{{ID: "orders", Name: "Order management", ClaimIDs: []string{statement.ID, intent.ID, runtimeClaim.ID, hypothesis.ID}}, {ID: "inventory", Name: "Inventory", ClaimIDs: []string{inventoryHypothesis.ID}}},
		Proposal:       ModelProposal{Goal: "Represent confirmed order cancellation scope", Files: []ProposedFile{{ScopeID: "orders", Path: ".markitect/model/orders/statement.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\n"}, {ScopeID: "inventory", Path: ".markitect/model/inventory/statement.yaml", Content: "apiVersion: project.markitect.example.org/v1alpha1\n"}}},
	}
	SealDistillation(&report)
	return report
}

func discoveryEvidence(discovery Discovery, id string) Evidence {
	for _, item := range discovery.Evidence {
		if item.ID == id {
			return item
		}
	}
	panic("missing test evidence " + id)
}

func testRuntimeRecord(source string) string {
	return testRuntimeRecordAtRevision(source, strings.Repeat("0", 40))
}

func testRuntimeRecordAtRevision(source, revision string) string {
	record := submittedRuntimeRecord{
		APIVersion: RuntimeRecordVersion, RecordSourceRevision: revision,
		Command: []string{"go", "test", "./orders"}, ExitCode: intValue(0),
		RunnerDigest: strings.Repeat("a", 64),
		Inputs:       []RuntimeInput{{Path: "src/orders/cancel.go", Digest: digestBytes([]byte(source))}},
	}
	data, err := json.Marshal(record)
	if err != nil {
		panic(err)
	}
	return string(data) + "\n"
}

func runtimeObservationFromEvidence(evidence Evidence, discoveryCommit string) *RuntimeObservation {
	var record submittedRuntimeRecord
	if err := decodeClosedJSON([]byte(evidence.Content), &record); err != nil {
		panic(err)
	}
	relation := "historical"
	if record.RecordSourceRevision == discoveryCommit {
		relation = "same-discovery-commit"
	}
	return &RuntimeObservation{
		EvidenceID: evidence.ID, RecordSourceRevision: record.RecordSourceRevision,
		SourceRelation: relation, Command: append([]string(nil), record.Command...),
		ExitCode: intValue(*record.ExitCode), RunnerDigest: record.RunnerDigest,
		Inputs: append([]RuntimeInput(nil), record.Inputs...),
	}
}

func boolValue(value bool) *bool { return &value }
func intValue(value int) *int    { return &value }

// committedRepository returns a repository on branch main whose first commit
// holds files, and that commit.
func committedRepository(t *testing.T, files map[string]string) (*testkit.Repo, string) {
	t.Helper()
	repo := testkit.NewRepo(t)
	for name, content := range files {
		repo.Write(name, content)
	}
	return repo, repo.Commit("fixture")
}
