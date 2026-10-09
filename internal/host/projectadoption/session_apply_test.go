package projectadoption

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

func TestApplyAndRecordSessionAdoptionRequiresReviewedPlanAndRecordsGuardedResult(t *testing.T) {
	root, sourceCommit := committedRepository(t, map[string]string{"src/orders.go": "package orders\nfunc Order() {}\n"})
	gitRun(t, root, "checkout", "-b", "codex/session-adoption-fixture")
	if _, err := projectwork.Init(root, "Session adoption fixture", true); err != nil {
		t.Fatal(err)
	}
	manifestPath := filepath.Join(root, filepath.FromSlash(projectwork.ManifestPath))
	manifestBytes, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	config, err := projectwork.DecodeConfig(manifestBytes)
	if err != nil {
		t.Fatal(err)
	}
	config.CoverageMode = "full"
	manifestBytes, err = yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, manifestBytes, 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "initialize target project")
	targetRevision := gitRun(t, root, "rev-parse", "HEAD")
	target, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	schemaDigest, buildDigest, err := CurrentBindings(projectmodel.Schema())
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "session-adoption", Purpose: "Model order responsibility", Review: "source-review",
		Commit: sourceCommit, ScopeRoots: []string{"."}, Selected: []SelectedPath{{ID: "orders-source", Path: "src/orders.go", Reason: "Implementation evidence", Basis: "code"}}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	iterationID := "root-pass"
	session, err = BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: iterationID, ManagerID: session.TargetContext.RootManagerID,
		EvidenceIDs: []string{"orders-source"}, DelegationEvidenceIDs: []string{}, Purpose: "Propose observed order model", Review: "model-review"})
	if err != nil {
		t.Fatal(err)
	}
	report := sessionReport(discovery, "orders-source", "orders", "func Order() {}", "The source defines order behavior.")
	report.SchemaDigest = schemaDigest
	report.Proposal.Files[0].Content = "apiVersion: " + projectmodel.APIVersion + "\nkind: Statement\nmetadata:\n  name: order-responsibility\n  namespace: orders\npurpose: Record order responsibility.\nspec:\n  category: concept\n  description: Manage order behavior.\n"
	SealDistillation(&report)
	session, err = RecordManagerProposal(session, iterationID, ManagerProposal{ManagerID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-source"},
		Hierarchy: []ProposedManager{}, PublicContracts: []ManagerPublicContract{}, Report: report})
	if err != nil {
		t.Fatal(err)
	}
	session, err = IntegrateManagerProposal(session, iterationID, session.TargetContext.RootManagerID, ManagerIntegration{ManagerID: session.TargetContext.RootManagerID,
		ChildProposalDigests: []string{}, ChildContracts: []IntegratedChildContracts{}, Report: report, Conflicts: []SessionConflict{}})
	if err != nil {
		t.Fatal(err)
	}
	resolution := Resolution{APIVersion: ResolutionVersion, DiscoveryDigest: discovery.Digest, DistillationDigest: report.Digest, ProposalDigest: ProposalDigest(report.Proposal),
		TargetBasis: session.Target.ProjectDigest, SchemaDigest: schemaDigest, BuildDigest: buildDigest, Actor: "user", AuthorityClaim: "Project owner", DecisionReference: "order-decision",
		Authenticated: boolValue(false), Questions: []QuestionResolution{}, Scopes: []ScopeResolution{{ScopeID: "orders", Status: "adopt", Reason: "Owner accepted the order scope"}}}
	SealResolution(&resolution)
	session, err = RecordSessionResolution(session, iterationID, resolution)
	if err != nil {
		t.Fatal(err)
	}
	if readiness := AssessReadiness(session, target.Coverage); readiness.Ready || readiness.Scopes[0].Status == "adopted" {
		t.Fatal("owner resolution alone cannot claim completed model adoption or readiness")
	}
	plan, err := PlanSessionAdoption(root, target, session, iterationID, schemaDigest, buildDigest)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := ApplyAndRecordSessionAdoption(root, root, target, session, iterationID, strings.Repeat("0", 64), schemaDigest, buildDigest); err == nil {
		t.Fatal("apply must reject a forged or stale reviewed-plan digest")
	}
	if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(".markitect/model/orders/statement.yaml"))); err == nil {
		t.Fatal("rejected plan digest must not write a candidate model")
	}
	beforeSource, err := os.ReadFile(filepath.Join(root, "src", "orders.go"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBrownfieldSession(root, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	updated, appliedPlan, receipt, err := ApplyAndRecordSessionAdoption(root, root, target, session, iterationID, plan.PlanDigest, schemaDigest, buildDigest)
	if err != nil {
		t.Fatalf("guarded apply should record its own receipt: %v", err)
	}
	if appliedPlan.PlanDigest != plan.PlanDigest || receipt.CandidateDigest != plan.Edit.CandidateDigest || len(updated.Adoptions) != 1 || updated.Adoptions[0].Receipt.CandidateDigest != receipt.CandidateDigest || updated.Adoptions[0].Plan.PlanDigest != plan.PlanDigest {
		t.Fatalf("returned adoption does not bind actual guarded result: plan=%+v receipt=%+v adoptions=%+v", appliedPlan, receipt, updated.Adoptions)
	}
	afterSource, err := os.ReadFile(filepath.Join(root, "src", "orders.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(beforeSource) != string(afterSource) {
		t.Fatal("model adoption changed source implementation")
	}
	if _, err := WriteBrownfieldSession(root, updated, session.Digest); err != nil {
		t.Fatal(err)
	}
	_, readiness, err := ResumeBrownfieldSession(root, root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.TargetCurrent || readiness.Ready {
		t.Fatal("an applied but uncommitted model candidate cannot be reported as accepted or ready")
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "accept adopted order model")
	acceptedRevision := gitRun(t, root, "rev-parse", "HEAD")
	acceptedTarget, err := projectwork.Load(root, acceptedRevision)
	if err != nil {
		t.Fatal(err)
	}
	fixedTarget, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	if acceptedTarget.Coverage.Digest == fixedTarget.Coverage.Digest {
		t.Fatal("adopted model should change the coverage report used for current readiness")
	}
	resumed, readiness, err := ResumeBrownfieldSession(root, root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !readiness.TargetCurrent || readiness.CoverageAccounted != acceptedTarget.Coverage.Accounted || readiness.CoverageConforming != acceptedTarget.Coverage.Conforming {
		t.Fatalf("committed target readiness must use its current coverage report: readiness=%+v coverage=%+v", readiness, acceptedTarget.Coverage)
	}
	refined, err := BeginReverseIteration(root, target, resumed, ReverseIterationRequest{ID: "root-refine", SupersedesIterationID: iterationID, ManagerID: resumed.TargetContext.RootManagerID,
		EvidenceIDs: []string{"orders-source"}, DelegationEvidenceIDs: []string{}, Purpose: "Revisit the accepted order model", Review: "refinement-review"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := PlanSessionAdoption(root, target, refined, iterationID, schemaDigest, buildDigest); err == nil {
		t.Fatal("a resolved iteration superseded by a new root pass cannot be adopted")
	}
	if _, err := WriteBrownfieldSession(root, refined, resumed.Digest); err != nil {
		t.Fatal(err)
	}
	_, readiness, err = ResumeBrownfieldSession(root, root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if readiness.Ready || !readiness.TargetCurrent {
		t.Fatalf("an accepted prior candidate cannot make a new incomplete root pass ready: %+v", readiness)
	}
}
