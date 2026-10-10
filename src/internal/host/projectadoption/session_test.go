package projectadoption

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectcoverage"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"go.yaml.in/yaml/v3"
)

func TestBrownfieldSessionPersistsAndResumesWithoutReplayingStages(t *testing.T) {
	root, commit := committedRepository(t, map[string]string{"src/orders.go": "package src\nfunc Order() {}\n"})
	gitRun(t, root, "checkout", "-b", "codex/brownfield-session-fixture")
	if _, err := projectwork.Init(root, "Target fixture", true); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "initialize target")
	targetRevision := gitRun(t, root, "rev-parse", "HEAD")
	target, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(root, DiscoveryRequest{
		APIVersion: DiscoveryVersion, ID: "brownfield-session", Purpose: "Map order module", Review: "human-review-1", Commit: commit,
		ScopeRoots: []string{"."}, Selected: []SelectedPath{{ID: "orders-source", Path: "src/orders.go", Reason: "Implementation evidence", Basis: "code"}},
		Exclusions: []PathReason{}, Unselected: []PathReason{},
	})
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBrownfieldSession(root, session, session.Digest); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadBrownfieldSession(root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Digest != session.Digest || len(loaded.Iterations) != 0 {
		t.Fatal("recovered session changed its initial bases or replayed work")
	}
	iterated, err := BeginReverseIteration(root, target, loaded, ReverseIterationRequest{ID: "root-pass", ManagerID: loaded.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-source"}, DelegationEvidenceIDs: []string{}, Purpose: "Propose observed hierarchy", Review: "review-2"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBrownfieldSession(root, iterated, loaded.Digest); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBrownfieldSession(root, session, loaded.Digest); err == nil {
		t.Fatal("stale session writer must lose the compare-and-swap")
	}
	resumed, ready, err := ResumeBrownfieldSession(root, root, session.ID)
	if err != nil {
		t.Fatal(err)
	}
	if resumed.Digest != iterated.Digest || len(resumed.Iterations) != 1 || !ready.SourceCurrent || !ready.TargetCurrent || ready.Ready {
		t.Fatalf("resume should verify both bases without claiming readiness: %#v", ready)
	}
	if _, err := WriteBrownfieldSession(root, session, session.Digest); err == nil {
		t.Fatal("CAS create must reject an existing session")
	}
}

func TestReverseIterationRejectsStaleSourceAndTarget(t *testing.T) {
	root, commit := committedRepository(t, map[string]string{"src/orders.go": "package src\nfunc Order() {}\n"})
	gitRun(t, root, "checkout", "-b", "codex/brownfield-stale-fixture")
	if _, err := projectwork.Init(root, "Target fixture", true); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "initialize target")
	targetRevision := gitRun(t, root, "rev-parse", "HEAD")
	target, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "stale-session", Purpose: "Map module", Review: "review", Commit: commit,
		ScopeRoots: []string{"."}, Selected: []SelectedPath{{ID: "source", Path: "src/orders.go", Reason: "Selected implementation", Basis: "code"}}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	changedDiscovery := discovery
	changedDiscovery.Evidence = append([]Evidence(nil), discovery.Evidence...)
	changedDiscovery.Evidence[0].Content += "// changed basis\n"
	changedDiscovery.Evidence[0].Digest = digestBytes([]byte(changedDiscovery.Evidence[0].Content))
	SealDiscovery(&changedDiscovery)
	if _, err := RefreshDiscovery(root, changedDiscovery); err == nil {
		t.Fatal("a changed selected source basis should not refresh as the original Discovery")
	}
	otherTarget, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	otherTarget.Digest = "sha256:" + "aabbcc"
	if _, err := BeginReverseIteration(root, otherTarget, session, ReverseIterationRequest{ID: "root-pass", ManagerID: session.TargetContext.RootManagerID,
		EvidenceIDs: []string{"source"}, DelegationEvidenceIDs: []string{}, Purpose: "Propose a source hierarchy", Review: "review-2"}); err == nil {
		t.Fatal("reverse iteration must reject a target digest changed after session start")
	}
}

func TestReadinessDoesNotPromoteTransitionalCoverage(t *testing.T) {
	root, commit := committedRepository(t, map[string]string{"src/orders.go": "package src\n"})
	gitRun(t, root, "checkout", "-b", "codex/brownfield-readiness-fixture")
	if _, err := projectwork.Init(root, "Target fixture", true); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "initialize target")
	revision := gitRun(t, root, "rev-parse", "HEAD")
	target, err := projectwork.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "readiness-session", Purpose: "Map source", Review: "review", Commit: commit,
		ScopeRoots: []string{"."}, Selected: []SelectedPath{{ID: "source", Path: "src/orders.go", Reason: "Source evidence", Basis: "code"}}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	coverage := &projectcoverage.Report{Accounted: true, Conforming: false, Findings: []projectcoverage.Finding{{Code: "coverage.transitional", Message: "file is transitional", Severity: "warning"}}}
	readiness := AssessReadiness(session, coverage)
	if readiness.Ready || readiness.CoverageConforming {
		t.Fatal("transitional coverage must never be promoted to conformance or readiness")
	}
}

func TestBrownfieldStartRefusesScopeStatusesWithoutProposals(t *testing.T) {
	root, discovery, _, target := distillationDiscovery(t)
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{{ScopeID: "orders", Status: "observed", Reason: "Owner already tracks this area"}})
	if err == nil {
		validateErr := ValidateBrownfieldSession(session)
		_, writeErr := WriteBrownfieldSession(root, session, session.Digest)
		t.Fatalf("start accepted initial scope statuses as session %s that fails validation (%v) and cannot be written (%v)", session.Digest, validateErr, writeErr)
	}
	if !strings.Contains(err.Error(), "initial scope statuses require recorded reverse-model proposals") {
		t.Fatalf("start refused initial scope statuses without saying they need recorded proposals: %v", err)
	}
	session, err = StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := WriteBrownfieldSession(root, session, session.Digest); err != nil {
		t.Fatalf("a start without scope statuses must stay writable: %v", err)
	}
}

func TestSessionLedgerRejectsSymlinkedControlPlane(t *testing.T) {
	root, _ := committedRepository(t, map[string]string{"src/main.go": "package src\n"})
	markitect := filepath.Join(root, ".markitect")
	if err := os.RemoveAll(markitect); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(t.TempDir(), markitect); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	if _, err := sessionDirectory(root, "safe-id", true); err == nil {
		t.Fatal("session ledger must reject a symlinked control-plane directory")
	}
}

func TestManagerIterationUsesAssignedEvidenceAndParentIntegratesChildProposal(t *testing.T) {
	root, sourceCommit := committedRepository(t, map[string]string{
		"src/orders.go":  "package orders\nfunc Order() {}\n// another valid source line\n",
		"docs/orders.md": "Orders are managed by the order module.\n",
	})
	gitRun(t, root, "checkout", "-b", "codex/brownfield-manager-fixture")
	if _, err := projectwork.Init(root, "Manager fixture", true); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "initialize manager tree")
	targetRevision := gitRun(t, root, "rev-parse", "HEAD")
	target, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "manager-iteration", Purpose: "Reverse model order module", Review: "review-iteration", Commit: sourceCommit,
		ScopeRoots: []string{"."}, Selected: []SelectedPath{
			{ID: "orders-code", Path: "src/orders.go", Reason: "Implementation selected for order Manager", Basis: "code"},
			{ID: "orders-doc", Path: "docs/orders.md", Reason: "Intent selected for root Manager", Basis: "documentation"},
		}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	childManager := ProposedManager{ID: "orders-manager", Name: "Orders Manager", Purpose: "Reverse model order implementation.", ParentID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}}
	rootIteration, err := BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "root", ManagerID: session.TargetContext.RootManagerID,
		EvidenceIDs: []string{"orders-doc"}, DelegationEvidenceIDs: []string{"orders-code"}, Purpose: "Propose source hierarchy", Review: "root-review"})
	if err != nil {
		t.Fatal(err)
	}
	rootReport := sessionReport(discovery, "orders-doc", "orders", "Orders are managed by the order module.", "Root scope hypothesis.")
	rootContract := ManagerPublicContract{Contract: DistillationTargetContract{ID: "orders.order-intent", Name: "order-intent", Namespace: "orders", Owner: session.TargetContext.RootManagerID, Category: "concept", Description: "Order management intent.", Uses: []string{}, Requires: []string{}}, ClaimIDs: []string{"orders-claim"}}
	rootProposal := ManagerProposal{ManagerID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-doc"}, Hierarchy: []ProposedManager{childManager}, PublicContracts: []ManagerPublicContract{rootContract}, Report: rootReport}
	rootIteration, err = RecordManagerProposal(rootIteration, "root", rootProposal)
	if err != nil {
		t.Fatal(err)
	}
	childIteration, err := BeginReverseIteration(root, target, rootIteration, ReverseIterationRequest{ID: "orders-manager", ParentIterationID: "root", ManagerID: childManager.ID,
		EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}, Purpose: "Inspect order implementation", Review: "orders-review"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BeginReverseIteration(root, target, childIteration, ReverseIterationRequest{ID: "orders-manager-second", ParentIterationID: "root", ManagerID: childManager.ID,
		EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}, Purpose: "Retry order inspection", Review: "orders-review-2"}); err == nil {
		t.Fatal("a parent cannot open duplicate child iterations for the same Manager")
	}
	duplicateStored := cloneSession(childIteration)
	duplicateChild := duplicateStored.Iterations[1]
	duplicateChild.ID = "orders-manager-second"
	duplicateStored.Iterations = append(duplicateStored.Iterations, duplicateChild)
	sealSession(&duplicateStored)
	if err := ValidateBrownfieldSession(duplicateStored); err == nil {
		t.Fatal("stored session validation must reject duplicate parent/Manager child iterations")
	}
	duplicateBytes, err := json.Marshal(duplicateStored)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBrownfieldSession(duplicateBytes); err == nil {
		t.Fatal("ledger decoding must enforce the same duplicate child iteration rule")
	}
	managerContext, err := BuildManagerReverseContext(childIteration, "orders-manager")
	if err != nil {
		t.Fatal(err)
	}
	if len(managerContext.Evidence) != 1 || managerContext.Evidence[0].EvidenceID != "orders-code" || managerContext.Evidence[0].Classification != "observed-implementation" || managerContext.ManagerOrigin != "proposed-by-parent" {
		t.Fatalf("Manager context must contain only its selected source evidence with explicit classification: %+v", managerContext)
	}
	if len(managerContext.ProposedNeighborContracts) != 1 || managerContext.ProposedNeighborContracts[0].Classification != "manager-proposed-public-contract" || managerContext.ProposedNeighborContracts[0].Contract.Contract.ID != "orders.order-intent" {
		t.Fatalf("child context must expose the parent's explicit proposed public contract separately from accepted desired contracts: %+v", managerContext)
	}
	rootContext, err := BuildManagerReverseContext(rootIteration, "root")
	if err != nil {
		t.Fatal(err)
	}
	if len(rootContext.Evidence) != 1 || rootContext.Evidence[0].Classification != "documented-intent" {
		t.Fatalf("root bootstrap context should contain only its selected documented intent: %+v", rootContext)
	}
	wrongEvidence := sessionReport(discovery, "orders-doc", "orders", "Orders are managed by the order module.", "Child proposal.")
	if _, err := RecordManagerProposal(childIteration, "orders-manager", ManagerProposal{ManagerID: childManager.ID, EvidenceIDs: []string{"orders-code"}, Hierarchy: []ProposedManager{}, PublicContracts: []ManagerPublicContract{}, Report: wrongEvidence}); err == nil {
		t.Fatal("child Manager must not cite evidence outside its assignment")
	}
	childReport := sessionReport(discovery, "orders-code", "orders", "func Order() {}", "Order code declares the order module.")
	childReport.Terms = []Term{{ID: "order-term", Text: "Order", Context: "The selected implementation names the order operation.",
		Occurrences: []TermOccurrence{{EvidenceID: "orders-code", StartLine: 2, EndLine: 2, Excerpt: "func Order() {}"}}, Synonyms: []string{}, Ambiguities: []string{}}}
	SealDistillation(&childReport)
	publicOrderContract := ManagerPublicContract{Contract: DistillationTargetContract{ID: "orders.accept-order", Name: "accept-order", Namespace: "orders", Owner: childManager.ID, Category: "use-case", Description: "Accept a valid order.", Uses: []string{}, Requires: []string{}}, ClaimIDs: []string{"orders-claim"}}
	submanager := ProposedManager{ID: "orders-submanager", Name: "Orders Submanager", Purpose: "Inspect order implementation details.", ParentID: childManager.ID, EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}}
	childProposal := ManagerProposal{ManagerID: childManager.ID, EvidenceIDs: []string{"orders-code"}, Hierarchy: []ProposedManager{submanager}, PublicContracts: []ManagerPublicContract{publicOrderContract}, Report: childReport}
	delegationAttempt := childProposal
	delegationAttempt.Hierarchy = []ProposedManager{{ID: "orders-submanager", Name: "Orders Submanager", Purpose: "Inspect assigned implementation.", ParentID: childManager.ID, EvidenceIDs: []string{"orders-doc"}, DelegationEvidenceIDs: []string{}}}
	if _, err := RecordManagerProposal(childIteration, "orders-manager", delegationAttempt); err == nil {
		t.Fatal("a non-root Manager cannot delegate evidence outside its exact parent assignment")
	}
	childIteration, err = RecordManagerProposal(childIteration, "orders-manager", childProposal)
	if err != nil {
		t.Fatal(err)
	}
	childDigest := childIteration.Iterations[1].Proposal.Digest
	integrated := sessionReport(discovery, "orders-code", "orders", "func Order() {}", "Integrated order contract.")
	integrated.Terms = []Term{{ID: "order-term", Text: "Order", Context: "The selected implementation names the order operation.",
		Occurrences: []TermOccurrence{{EvidenceID: "orders-code", StartLine: 2, EndLine: 2, Excerpt: "func Order() {}"}}, Synonyms: []string{}, Ambiguities: []string{}}}
	integrated.Claims = append(integrated.Claims, Claim{ID: "root-orders-intent", ScopeID: "orders", Kind: "documented-intent", Method: "documentation", Statement: "The documentation assigns order management to the order module.",
		Evidence: []EvidenceRef{{EvidenceID: "orders-doc", StartLine: 1, EndLine: 1, Excerpt: "Orders are managed by the order module."}}, Uncertainty: []string{}})
	integrated.Scopes[0].ClaimIDs = []string{"orders-claim", "root-orders-intent"}
	integrated.Questions = []Question{{ID: "clarify-orders", ScopeID: "orders", Prompt: "Which behavior is accepted?", Alternatives: []string{"implementation", "documentation"}, ClaimIDs: []string{"orders-claim", "root-orders-intent"}, Blocking: boolValue(true)}}
	integrated.Proposal.Files[0].Content = "apiVersion: " + projectwork.APIVersion + "\nkind: Statement\n"
	SealDistillation(&integrated)
	integration := ManagerIntegration{ManagerID: session.TargetContext.RootManagerID, ChildProposalDigests: []string{childDigest}, ChildContracts: []IntegratedChildContracts{{ManagerID: childManager.ID, ProposalDigest: childDigest, Contracts: []ManagerPublicContract{publicOrderContract}}}, Report: integrated, Conflicts: []SessionConflict{{
		ID: "orders-intent-conflict", ScopeID: "orders", QuestionID: "clarify-orders", Description: "Implementation and documentation differ.", EvidenceIDs: []string{"orders-code", "orders-doc"}, Disposition: "unresolved", Reason: "The manager cannot decide desired behavior.",
	}}}
	if _, err := IntegrateManagerProposal(childIteration, "root", session.TargetContext.RootManagerID, integration); err == nil {
		t.Fatal("parent integration cannot accept a non-leaf child before that child integrates its own branch")
	}
	if _, err := BuildManagerIntegrationContext(childIteration, "root"); err == nil {
		t.Fatal("parent integration context cannot be built before its non-leaf child completes integration")
	}
	grandchildIteration, err := BeginReverseIteration(root, target, childIteration, ReverseIterationRequest{ID: "orders-submanager", ParentIterationID: "orders-manager", ManagerID: submanager.ID,
		EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}, Purpose: "Inspect order implementation details", Review: "submanager-review"})
	if err != nil {
		t.Fatal(err)
	}
	grandchildReport := sessionReport(discovery, "orders-code", "orders", "func Order() {}", "The submanager observed the order implementation.")
	grandchildProposal := ManagerProposal{ManagerID: submanager.ID, EvidenceIDs: []string{"orders-code"}, Hierarchy: []ProposedManager{}, PublicContracts: []ManagerPublicContract{}, Report: grandchildReport}
	grandchildIteration, err = RecordManagerProposal(grandchildIteration, "orders-submanager", grandchildProposal)
	if err != nil {
		t.Fatal(err)
	}
	grandchildDigest := grandchildIteration.Iterations[2].Proposal.Digest
	childFinalReport := childReport
	childFinalReport.Claims = append([]Claim(nil), childReport.Claims...)
	childFinalReport.Claims[0].Statement = "The order manager integrated its submanager's finding."
	SealDistillation(&childFinalReport)
	childIntegration := ManagerIntegration{ManagerID: childManager.ID, ChildProposalDigests: []string{grandchildDigest},
		ChildContracts: []IntegratedChildContracts{{ManagerID: submanager.ID, ProposalDigest: grandchildDigest, Contracts: []ManagerPublicContract{}}},
		Report:         childFinalReport, Conflicts: []SessionConflict{}}
	childIteration, err = IntegrateManagerProposal(grandchildIteration, "orders-manager", childManager.ID, childIntegration)
	if err != nil {
		t.Fatalf("child Manager should integrate its submanager before the root: %v", err)
	}
	integrationContext, err := BuildManagerIntegrationContext(childIteration, "root")
	if err != nil {
		t.Fatal(err)
	}
	if integrationContext.ParentProposal.ManagerID != session.TargetContext.RootManagerID || len(integrationContext.Children) != 1 || integrationContext.Children[0].ManagerID != childManager.ID || integrationContext.Children[0].IntegrationDigest != childIteration.Iterations[1].Integration.Digest || integrationContext.Children[0].ReportDigest != childFinalReport.Digest || integrationContext.Children[0].Report.Claims[0].Statement != childFinalReport.Claims[0].Statement {
		t.Fatalf("parent integration context must include the completed direct-child report after grandchild integration: %+v", integrationContext)
	}
	integration.ChildIntegrationDigests = []ChildIntegrationDigest{{ManagerID: childManager.ID, ProposalDigest: childDigest,
		IntegrationDigest: childIteration.Iterations[1].Integration.Digest, ReportDigest: childFinalReport.Digest}}
	unauthorizedReport := integrated
	unauthorizedReport.Claims = append([]Claim(nil), integrated.Claims...)
	unauthorizedReport.Claims[0].Evidence = append([]EvidenceRef(nil), integrated.Claims[0].Evidence...)
	unauthorizedReport.Claims[0].Evidence[0] = EvidenceRef{EvidenceID: "orders-code", StartLine: 1, EndLine: 1, Excerpt: "package orders"}
	SealDistillation(&unauthorizedReport)
	unauthorizedIntegration := integration
	unauthorizedIntegration.Report = unauthorizedReport
	if _, err := IntegrateManagerProposal(childIteration, "root", session.TargetContext.RootManagerID, unauthorizedIntegration); err == nil {
		t.Fatal("parent integration must not invent a new valid line citation from delegated evidence")
	}
	unauthorizedTermReport := integrated
	unauthorizedTermReport.Terms = append([]Term(nil), integrated.Terms...)
	unauthorizedTermReport.Terms[0].Occurrences = append([]TermOccurrence(nil), integrated.Terms[0].Occurrences...)
	unauthorizedTermReport.Terms[0].Occurrences[0] = TermOccurrence{EvidenceID: "orders-code", StartLine: 1, EndLine: 1, Excerpt: "package orders"}
	SealDistillation(&unauthorizedTermReport)
	unauthorizedTermIntegration := integration
	unauthorizedTermIntegration.Report = unauthorizedTermReport
	if _, err := IntegrateManagerProposal(childIteration, "root", session.TargetContext.RootManagerID, unauthorizedTermIntegration); err == nil {
		t.Fatal("parent integration must not invent a new term occurrence from delegated evidence")
	}
	childIteration, err = IntegrateManagerProposal(childIteration, "root", session.TargetContext.RootManagerID, integration)
	if err != nil {
		t.Fatalf("parent integration should bind its child's public proposal: %v", err)
	}
	wrongContracts := *childIteration.Iterations[0].Integration
	wrongContracts.ChildContracts = append([]IntegratedChildContracts(nil), wrongContracts.ChildContracts...)
	wrongContracts.ChildContracts[0].Contracts = []ManagerPublicContract{{Contract: DistillationTargetContract{ID: "orders.accept-order", Name: "accept-order", Namespace: "orders", Owner: childManager.ID, Category: "use-case", Description: "Changed after child proposal.", Uses: []string{}, Requires: []string{}}, ClaimIDs: []string{"orders-claim"}}}
	wrongContracts.Digest = ""
	preIntegration := cloneSession(childIteration)
	preIntegration.Iterations[0].Integration = nil
	sealSession(&preIntegration)
	if _, err := IntegrateManagerProposal(preIntegration, "root", session.TargetContext.RootManagerID, wrongContracts); err == nil {
		t.Fatal("parent cannot silently alter the child's public contract during integration")
	}
	wrongChildIntegration := *childIteration.Iterations[0].Integration
	wrongChildIntegration.ChildIntegrationDigests = append([]ChildIntegrationDigest(nil), wrongChildIntegration.ChildIntegrationDigests...)
	wrongChildIntegration.ChildIntegrationDigests[0].ReportDigest = strings.Repeat("a", 64)
	wrongChildIntegration.Digest = ""
	if _, err := IntegrateManagerProposal(preIntegration, "root", session.TargetContext.RootManagerID, wrongChildIntegration); err == nil {
		t.Fatal("parent integration must bind the exact final child integration and report digests")
	}
	if childIteration.Iterations[0].Integration == nil || childIteration.Iterations[0].Integration.ChildProposalDigests[0] != childDigest {
		t.Fatal("parent integration did not record child proposal digest")
	}
	resolution := Resolution{APIVersion: ResolutionVersion, DiscoveryDigest: discovery.Digest, DistillationDigest: integrated.Digest, ProposalDigest: ProposalDigest(integrated.Proposal),
		TargetBasis: session.Target.ProjectDigest, SchemaDigest: integrated.SchemaDigest, BuildDigest: strings.Repeat("c", 64), Actor: "user", AuthorityClaim: "Project owner", DecisionReference: "decision-orders",
		Authenticated: boolValue(false), Questions: []QuestionResolution{}, Scopes: []ScopeResolution{{ScopeID: "orders", Status: "adopt", Reason: "Awaiting explicit behavior answer"}}}
	SealResolution(&resolution)
	if _, err := RecordSessionResolution(childIteration, "root", resolution); err == nil {
		t.Fatal("manager conflict must not be adopted without its explicit owner answer")
	}
	resolution.Questions = []QuestionResolution{{QuestionID: "clarify-orders", ScopeID: "orders", Disposition: "answer", Answer: "Documented behavior is intended", Reason: "Owner confirmed desired behavior"}}
	SealResolution(&resolution)
	resolved, err := RecordSessionResolution(childIteration, "root", resolution)
	if err != nil {
		t.Fatalf("explicit owner resolution should settle the scoped conflict: %v", err)
	}
	if readiness := AssessReadiness(resolved, &projectcoverage.Report{Accounted: true, Conforming: false}); len(readiness.UnresolvedConflicts) != 0 || readiness.Ready {
		t.Fatalf("resolved conflict should be cleared while nonconforming coverage still blocks readiness: %+v", readiness)
	}
	transitional := cloneSession(childIteration)
	setOneScopeStatus(&transitional, "orders", "transitional", "Behavior remains outside the accepted model")
	sealSession(&transitional)
	conformingCoverage := &projectcoverage.Report{Accounted: true, Conforming: true}
	if readiness := AssessReadiness(transitional, conformingCoverage); readiness.Ready {
		t.Fatal("a transitional scope cannot be made ready by a conforming coverage input")
	}
	repeatedRoot, err := BeginReverseIteration(root, target, childIteration, ReverseIterationRequest{ID: "root-refine", SupersedesIterationID: "root", ManagerID: session.TargetContext.RootManagerID,
		EvidenceIDs: []string{"orders-doc"}, DelegationEvidenceIDs: []string{}, Purpose: "Revisit integrated source model", Review: "root-review-2"})
	if err != nil {
		t.Fatalf("root Manager should be able to repeat reverse inference after integrating its prior tree: %v", err)
	}
	repeatedProposal := ManagerProposal{ManagerID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-doc"}, Hierarchy: []ProposedManager{}, PublicContracts: []ManagerPublicContract{}, Report: rootReport}
	repeatedRoot, err = RecordManagerProposal(repeatedRoot, "root-refine", repeatedProposal)
	if err != nil {
		t.Fatal(err)
	}
	repeatedRoot, err = IntegrateManagerProposal(repeatedRoot, "root-refine", session.TargetContext.RootManagerID, ManagerIntegration{ManagerID: session.TargetContext.RootManagerID, ChildProposalDigests: []string{}, ChildContracts: []IntegratedChildContracts{}, Report: rootReport, Conflicts: []SessionConflict{}})
	if err != nil {
		t.Fatalf("repeated root pass should integrate as a new immutable iteration: %v", err)
	}
	if complete, _ := activeTreeComplete(repeatedRoot); complete {
		t.Fatal("an active root with no owner resolution cannot claim a complete reverse-model tree")
	}
	latestReadiness := AssessReadiness(repeatedRoot, conformingCoverage)
	if len(latestReadiness.UnresolvedConflicts) != 0 || len(latestReadiness.BlockingQuestions) != 0 {
		t.Fatalf("superseded root conflicts and questions must remain historical without blocking the active clarified pass: %+v", latestReadiness)
	}
}

func sessionReport(discovery Discovery, evidenceID, scopeID, excerpt, statement string) Distillation {
	evidence := discoveryEvidence(discovery, evidenceID)
	line := 0
	for index, candidate := range strings.Split(evidence.Content, "\n") {
		if candidate == excerpt {
			line = index + 1
			break
		}
	}
	if line == 0 {
		panic("test excerpt not found in evidence")
	}
	claimID := scopeID + "-claim"
	kind, method := "observation", "static-source"
	if evidence.Basis == "documentation" {
		kind, method = "documented-intent", "documentation"
	}
	report := Distillation{APIVersion: DistillationVersion, DiscoveryDigest: discovery.Digest, Method: "human-review", SchemaDigest: strings.Repeat("d", 64),
		Claims: []Claim{{ID: claimID, ScopeID: scopeID, Kind: kind, Method: method, Statement: statement,
			Evidence: []EvidenceRef{{EvidenceID: evidenceID, StartLine: line, EndLine: line, Excerpt: excerpt}}, Uncertainty: []string{}}},
		Terms: []Term{}, Contradictions: []Contradiction{}, Questions: []Question{},
		Scopes:   []ScopeProposal{{ID: scopeID, Name: "Order responsibilities", ClaimIDs: []string{claimID}}},
		Proposal: ModelProposal{Goal: "Represent observed order responsibility", Files: []ProposedFile{{ScopeID: scopeID, Path: ".markitect/model/orders/statement.yaml", Content: "apiVersion: " + projectwork.APIVersion + "\nkind: Statement\n"}}},
	}
	SealDistillation(&report)
	return report
}

func TestAcceptedManagerTreeRequiresParentAssignmentAndIntegratesRecursively(t *testing.T) {
	root, sourceCommit := committedRepository(t, map[string]string{"src/orders.go": "package orders\nfunc Order() {}\n", "docs/orders.md": "Orders are managed by the order module.\n"})
	gitRun(t, root, "checkout", "-b", "codex/brownfield-accepted-tree")
	if _, err := projectwork.Init(root, "Accepted tree fixture", true); err != nil {
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
	childPath := ".markitect/model/orders/manager.yaml"
	config.ModelFiles = append(config.ModelFiles, childPath)
	encodedConfig, err := yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifestPath, encodedConfig, 0o644); err != nil {
		t.Fatal(err)
	}
	managerYAML := "apiVersion: " + projectwork.APIVersion + "\nkind: Manager\nmetadata:\n  name: orders\n  namespace: orders\npurpose: Owns order responsibilities.\nspec:\n  parent:\n    apiVersion: " + projectwork.APIVersion + "\n    kind: Manager\n    namespace: \"\"\n    name: project-owner\n  owns: [src/]\n"
	if err := os.MkdirAll(filepath.Dir(filepath.Join(root, filepath.FromSlash(childPath))), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(childPath)), []byte(managerYAML), 0o644); err != nil {
		t.Fatal(err)
	}
	gitRun(t, root, "add", "--all")
	gitRun(t, root, "commit", "--quiet", "-m", "add accepted order Manager")
	targetRevision := gitRun(t, root, "rev-parse", "HEAD")
	target, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "accepted-manager-tree", Purpose: "Reverse model order module", Review: "review-accepted-tree", Commit: sourceCommit,
		ScopeRoots: []string{"."}, Selected: []SelectedPath{{ID: "orders-code", Path: "src/orders.go", Reason: "Implementation evidence", Basis: "code"}, {ID: "orders-doc", Path: "docs/orders.md", Reason: "Documented intent", Basis: "documentation"}}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	var accepted DistillationTargetManager
	for _, manager := range session.TargetContext.Managers {
		if manager.Namespace == "orders" {
			accepted = manager
		}
	}
	if accepted.ID == "" {
		t.Fatalf("accepted child Manager missing: %+v", session.TargetContext.Managers)
	}
	rootIteration, err := BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "root", ManagerID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-doc"}, DelegationEvidenceIDs: []string{"orders-code"}, Purpose: "Propose Manager hierarchy", Review: "root-review"})
	if err != nil {
		t.Fatal(err)
	}
	rootReport := sessionReport(discovery, "orders-doc", "orders", "Orders are managed by the order module.", "The order module is responsible for order management.")
	rootProposal := ManagerProposal{ManagerID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-doc"}, Hierarchy: []ProposedManager{{ID: accepted.ID, Name: accepted.Name, Purpose: accepted.Purpose, ParentID: session.TargetContext.RootManagerID, EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}}}, PublicContracts: []ManagerPublicContract{}, Report: rootReport}
	rootIteration, err = RecordManagerProposal(rootIteration, "root", rootProposal)
	if err != nil {
		t.Fatal(err)
	}
	childIteration, err := BeginReverseIteration(root, target, rootIteration, ReverseIterationRequest{ID: "orders", ParentIterationID: "root", ManagerID: accepted.ID, EvidenceIDs: []string{"orders-code"}, DelegationEvidenceIDs: []string{}, Purpose: "Inspect order implementation", Review: "orders-review"})
	if err != nil {
		t.Fatal(err)
	}
	context, err := BuildManagerReverseContext(childIteration, "orders")
	if err != nil {
		t.Fatal(err)
	}
	if context.ManagerOrigin != "accepted-target" || context.Manager.ID != accepted.ID || len(context.Evidence) != 1 || context.Evidence[0].EvidenceID != "orders-code" {
		t.Fatalf("accepted child context is not bounded: %+v", context)
	}
	childReport := sessionReport(discovery, "orders-code", "orders", "func Order() {}", "Order module declares an order function.")
	childPublicContract := ManagerPublicContract{Contract: DistillationTargetContract{ID: "orders.order", Name: "order", Namespace: "orders", Owner: accepted.ID, Category: "concept", Description: "An order managed by the module.", Uses: []string{}, Requires: []string{}}, ClaimIDs: []string{"orders-claim"}}
	childProposal := ManagerProposal{ManagerID: accepted.ID, EvidenceIDs: []string{"orders-code"}, Hierarchy: []ProposedManager{}, PublicContracts: []ManagerPublicContract{childPublicContract}, Report: childReport}
	childIteration, err = RecordManagerProposal(childIteration, "orders", childProposal)
	if err != nil {
		t.Fatal(err)
	}
	integrated := sessionReport(discovery, "orders-code", "orders", "func Order() {}", "Integrated order behavior.")
	childProposalDigest := childIteration.Iterations[1].Proposal.Digest
	integration := ManagerIntegration{ManagerID: session.TargetContext.RootManagerID, ChildProposalDigests: []string{childProposalDigest}, ChildContracts: []IntegratedChildContracts{{ManagerID: accepted.ID, ProposalDigest: childProposalDigest, Contracts: []ManagerPublicContract{childPublicContract}}}, Report: integrated, Conflicts: []SessionConflict{}}
	childIteration, err = IntegrateManagerProposal(childIteration, "root", session.TargetContext.RootManagerID, integration)
	if err != nil {
		t.Fatal(err)
	}
	if childIteration.Iterations[0].Integration == nil {
		t.Fatal("root Manager did not integrate the accepted child's public proposal")
	}
}
