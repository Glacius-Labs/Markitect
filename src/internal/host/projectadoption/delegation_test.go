package projectadoption

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

func TestDelegationPoolsAreExplicitAndContextMetadataCarriesNoContent(t *testing.T) {
	repo, sourceCommit := committedRepository(t, map[string]string{
		"src/root.go":    "package src\nfunc Root() {}\n",
		"src/child.go":   "package src\nfunc Child() {}\n",
		"src/leaf.go":    "package src\nfunc Leaf() {}\n",
		"src/private.go": "package src\nfunc Private() {}\n",
	})
	root := repo.Dir
	repo.Git("checkout", "-b", "codex/delegation-pools")
	if _, err := projectwork.Init(root, "Delegation fixture", true); err != nil {
		t.Fatal(err)
	}
	targetRevision := repo.Commit("initialize delegation target")
	target, err := projectwork.Load(root, targetRevision)
	if err != nil {
		t.Fatal(err)
	}
	discovery, err := Discover(root, DiscoveryRequest{APIVersion: DiscoveryVersion, ID: "delegation-session", Purpose: "Route selected evidence", Review: "review-delegation", Commit: sourceCommit,
		ScopeRoots: []string{"src"}, Selected: []SelectedPath{
			{ID: "root-evidence", Path: "src/root.go", Reason: "Root responsibility", Basis: "code"},
			{ID: "child-evidence", Path: "src/child.go", Reason: "Child responsibility", Basis: "code"},
			{ID: "leaf-evidence", Path: "src/leaf.go", Reason: "Leaf responsibility", Basis: "code"},
			{ID: "private-evidence", Path: "src/private.go", Reason: "Not delegated to child", Basis: "code"},
		}, Exclusions: []PathReason{}, Unselected: []PathReason{}})
	if err != nil {
		t.Fatal(err)
	}
	session, err := StartBrownfieldSession(root, target, discovery, []ScopeStatus{})
	if err != nil {
		t.Fatal(err)
	}
	rootManager := session.TargetContext.RootManagerID
	boundedRoot, err := BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "bounded-root", ManagerID: rootManager,
		EvidenceIDs: []string{"root-evidence"}, DelegationEvidenceIDs: []string{}, Purpose: "Keep assignment local", Review: "bounded-root-review"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "missing-pool", ManagerID: rootManager,
		EvidenceIDs: []string{"root-evidence"}, Purpose: "Must fail closed", Review: "missing-pool-review"}); err == nil {
		t.Fatal("a missing delegation pool must fail closed instead of expanding to all Discovery evidence")
	}
	encodedBounded, err := EncodeBrownfieldSession(boundedRoot)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(encodedBounded), `"delegationEvidenceIds": []`) || strings.Contains(string(encodedBounded), "delegationEvidenceExplicit") {
		t.Fatalf("the closed ledger must store an explicit empty pool without compatibility marker bookkeeping: %s", encodedBounded)
	}
	decodedBounded, err := DecodeBrownfieldSession(encodedBounded)
	if err != nil {
		t.Fatal(err)
	}
	decodedBounded = cloneSession(decodedBounded)
	if _, err := WriteBrownfieldSession(root, boundedRoot, boundedRoot.Digest); err != nil {
		t.Fatal(err)
	}
	loadedBounded, err := LoadBrownfieldSession(root, boundedRoot.ID)
	if err != nil {
		t.Fatal(err)
	}
	var legacyShape map[string]any
	if err := json.Unmarshal(encodedBounded, &legacyShape); err != nil {
		t.Fatal(err)
	}
	iterations := legacyShape["iterations"].([]any)
	delete(iterations[0].(map[string]any), "delegationEvidenceIds")
	legacyBytes, err := json.Marshal(legacyShape)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeBrownfieldSession(legacyBytes); err == nil {
		t.Fatal("an ambiguous historical iteration without an explicit delegation pool must fail closed")
	}
	for name, candidate := range map[string]BrownfieldSession{"decode": decodedBounded, "write-load": loadedBounded} {
		outside := ProposedManager{ID: "unauthorized-child", Name: "Unauthorized child", Purpose: "Inspect unauthorized evidence", ParentID: rootManager, EvidenceIDs: []string{"child-evidence"}, DelegationEvidenceIDs: []string{}}
		outsideProposal := ManagerProposal{ManagerID: rootManager, EvidenceIDs: []string{"root-evidence"}, Hierarchy: []ProposedManager{outside}, PublicContracts: []ManagerPublicContract{},
			Report: sessionReport(discovery, "root-evidence", "root", "func Root() {}", "The root source defines the root behavior.")}
		if _, err := RecordManagerProposal(candidate, "bounded-root", outsideProposal); err == nil {
			t.Fatalf("explicitly empty root pool broadened after %s roundtrip", name)
		}
	}
	rootIteration, err := BeginReverseIteration(root, target, session, ReverseIterationRequest{ID: "root", ManagerID: rootManager,
		EvidenceIDs: []string{"root-evidence"}, DelegationEvidenceIDs: []string{"child-evidence", "leaf-evidence"}, Purpose: "Route module inspection", Review: "root-review"})
	if err != nil {
		t.Fatal(err)
	}
	rootContext, err := BuildManagerReverseContext(rootIteration, "root")
	if err != nil {
		t.Fatal(err)
	}
	if len(rootContext.Evidence) != 1 || rootContext.Evidence[0].EvidenceID != "root-evidence" || rootContext.Evidence[0].Content == "" || len(rootContext.DelegationEvidence) != 2 || len(rootContext.DiscoveryInventory) != 4 {
		t.Fatalf("root context must separate its raw evidence from authorized metadata and full metadata inventory: %+v", rootContext)
	}
	metadataBytes, err := json.Marshal(struct {
		Delegation []ManagerReverseEvidenceMetadata `json:"delegation"`
		Inventory  []ManagerReverseEvidenceMetadata `json:"inventory"`
	}{rootContext.DelegationEvidence, rootContext.DiscoveryInventory})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(metadataBytes), "func Child") || strings.Contains(string(metadataBytes), "func Leaf") {
		t.Fatal("delegation and Discovery inventory metadata must never carry source content")
	}
	child := ProposedManager{ID: "child-manager", Name: "Child manager", Purpose: "Inspect child code", ParentID: rootManager,
		EvidenceIDs: []string{"child-evidence"}, DelegationEvidenceIDs: []string{"leaf-evidence"}}
	rootReport := sessionReport(discovery, "root-evidence", "root", "func Root() {}", "The root source defines the root behavior.")
	proposal := ManagerProposal{ManagerID: rootManager, EvidenceIDs: []string{"root-evidence"}, Hierarchy: []ProposedManager{child}, PublicContracts: []ManagerPublicContract{}, Report: rootReport}
	badOverlap := proposal
	badOverlap.Hierarchy = []ProposedManager{{ID: "child-manager", Name: "Child manager", Purpose: "Inspect child code", ParentID: rootManager,
		EvidenceIDs: []string{"child-evidence"}, DelegationEvidenceIDs: []string{"child-evidence"}}}
	if _, err := RecordManagerProposal(rootIteration, "root", badOverlap); err == nil {
		t.Fatal("a child assignment cannot overlap its own evidence and delegation pools")
	}
	badDuplicate := proposal
	badDuplicate.Hierarchy = []ProposedManager{{ID: "child-manager", Name: "Child manager", Purpose: "Inspect child code", ParentID: rootManager,
		EvidenceIDs: []string{"child-evidence", "child-evidence"}, DelegationEvidenceIDs: []string{"leaf-evidence"}}}
	if _, err := RecordManagerProposal(rootIteration, "root", badDuplicate); err == nil {
		t.Fatal("stored child evidence assignments must reject duplicate IDs")
	}
	outsideRootPool := proposal
	outsideRootPool.Hierarchy = []ProposedManager{{ID: "child-manager", Name: "Child manager", Purpose: "Inspect child code", ParentID: rootManager,
		EvidenceIDs: []string{"child-evidence"}, DelegationEvidenceIDs: []string{"leaf-evidence", "private-evidence"}}}
	if _, err := RecordManagerProposal(rootIteration, "root", outsideRootPool); err == nil {
		t.Fatal("root's metadata inventory does not grant delegation outside its explicit pool")
	}
	rootIteration, err = RecordManagerProposal(rootIteration, "root", proposal)
	if err != nil {
		t.Fatal(err)
	}
	childIteration, err := BeginReverseIteration(root, target, rootIteration, ReverseIterationRequest{ID: "child", ParentIterationID: "root", ManagerID: child.ID,
		EvidenceIDs: []string{"child-evidence"}, DelegationEvidenceIDs: []string{"leaf-evidence"}, Purpose: child.Purpose, Review: "child-review"})
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := EncodeBrownfieldSession(childIteration)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeBrownfieldSession(encoded)
	if err != nil || !sameStrings(decoded.Iterations[1].DelegationEvidenceIDs, []string{"leaf-evidence"}) {
		t.Fatalf("explicit delegation assignment must survive closed ledger encoding: iteration=%+v err=%v", decoded.Iterations[1], err)
	}
	badStoredOverlap := cloneSession(childIteration)
	badStoredOverlap.Iterations[1].DelegationEvidenceIDs = []string{"child-evidence"}
	sealSession(&badStoredOverlap)
	if err := ValidateBrownfieldSession(badStoredOverlap); err == nil {
		t.Fatal("stored iteration validation must reject overlapping own and delegation pools")
	}
	childContext, err := BuildManagerReverseContext(childIteration, "child")
	if err != nil {
		t.Fatal(err)
	}
	if len(childContext.Evidence) != 1 || childContext.Evidence[0].EvidenceID != "child-evidence" || childContext.Evidence[0].Content == "" || len(childContext.DelegationEvidence) != 1 || childContext.DelegationEvidence[0].EvidenceID != "leaf-evidence" || childContext.DiscoveryInventory != nil && len(childContext.DiscoveryInventory) != 0 {
		t.Fatalf("non-root context must include own raw content and only its authorized delegation metadata: %+v", childContext)
	}
	if _, err := BeginReverseIteration(root, target, rootIteration, ReverseIterationRequest{ID: "child-mismatch", ParentIterationID: "root", ManagerID: child.ID,
		EvidenceIDs: []string{"child-evidence"}, DelegationEvidenceIDs: []string{}, Purpose: child.Purpose, Review: "child-review"}); err == nil {
		t.Fatal("child iteration must match the parent's exact delegation pool")
	}
	grandchild := ProposedManager{ID: "leaf-manager", Name: "Leaf manager", Purpose: "Inspect leaf code", ParentID: child.ID, EvidenceIDs: []string{"leaf-evidence"}, DelegationEvidenceIDs: []string{}}
	childReport := sessionReport(discovery, "child-evidence", "child", "func Child() {}", "The child source defines child behavior.")
	childProposal := ManagerProposal{ManagerID: child.ID, EvidenceIDs: []string{"child-evidence"}, Hierarchy: []ProposedManager{grandchild}, PublicContracts: []ManagerPublicContract{}, Report: childReport}
	childIteration, err = RecordManagerProposal(childIteration, "child", childProposal)
	if err != nil {
		t.Fatalf("non-root child should delegate only from its own and authorized pools: %v", err)
	}
	leafIteration, err := BeginReverseIteration(root, target, childIteration, ReverseIterationRequest{ID: "leaf", ParentIterationID: "child", ManagerID: grandchild.ID,
		EvidenceIDs: []string{"leaf-evidence"}, DelegationEvidenceIDs: []string{}, Purpose: grandchild.Purpose, Review: "leaf-review"})
	if err != nil {
		t.Fatalf("grandchild should receive the exact parent assignment: %v", err)
	}
	leafContext, err := BuildManagerReverseContext(leafIteration, "leaf")
	if err != nil {
		t.Fatal(err)
	}
	if len(leafContext.Evidence) != 1 || leafContext.Evidence[0].EvidenceID != "leaf-evidence" || leafContext.Evidence[0].Content == "" {
		t.Fatalf("grandchild should receive its own selected raw evidence: %+v", leafContext)
	}
	if _, err := RecordManagerProposal(rootIteration, "root", proposal); err == nil {
		t.Fatal("recording the same immutable proposal twice must remain rejected")
	}
}
