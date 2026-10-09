package projectknowledge

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

const testAPI = "example.org/v1"

func testModel(t *testing.T, revision, sourcePath string) core.Model {
	t.Helper()
	ref := core.Property{Purpose: "Resolved statements.", Type: core.TypeReference, Target: &core.KindIdentity{APIVersion: testAPI, Kind: "Statement"}, MinCount: 0, MaxCount: core.Unbounded}
	strList := core.Property{Purpose: "Ordered argv.", Type: core.TypeString, MinCount: 0, MaxCount: core.Unbounded}
	obj := core.Property{Purpose: "Repeated nested details.", Type: core.TypeObject, MinCount: 0, MaxCount: core.Unbounded, Properties: map[string]core.Property{
		"note": {Purpose: "Literal note.", Type: core.TypeString, MinCount: 0, MaxCount: 1}, "links": ref,
	}}
	schema := core.Schema{APIVersion: testAPI, Purpose: "Test vocabulary.", Kinds: map[string]core.Kind{
		"Statement": {Purpose: "A test statement.", Properties: map[string]core.Property{"requires": ref, "argv": strList, "count": {Purpose: "Count.", Type: core.TypeInteger, MinCount: 0, MaxCount: 1}, "details": obj}},
	}}
	defs := []core.Definition{
		{APIVersion: testAPI, Kind: "Statement", Metadata: core.Metadata{Namespace: "sales", Name: "cancel"}, Purpose: "cancel purpose", Spec: map[string]any{"requires": []any{map[string]any{"namespace": "inventory", "name": "release"}}, "argv": []any{"go", "test", "go", "test"}, "count": int64(9007199254740993), "details": []any{map[string]any{"note": "safe", "links": []any{map[string]any{"namespace": "private", "name": "secret"}}}}}, Source: core.Source{Path: sourcePath, Digest: "decl-digest", Line: 7}},
		{APIVersion: testAPI, Kind: "Statement", Metadata: core.Metadata{Namespace: "inventory", Name: "release"}, Purpose: "release", Spec: map[string]any{}, Source: core.Source{Path: "release.yaml"}},
		{APIVersion: testAPI, Kind: "Statement", Metadata: core.Metadata{Namespace: "private", Name: "secret"}, Purpose: "hidden secret", Spec: map[string]any{}, Source: core.Source{Path: "secret.yaml"}},
	}
	m, diags := core.Compile([]core.Schema{schema}, defs, revision)
	if len(diags) > 0 {
		t.Fatalf("compile: %+v", diags)
	}
	return m
}

func stmt(name string) string {
	return core.DefinitionIdentity{APIVersion: testAPI, Kind: "Statement", Namespace: "sales", Name: name}.Key()
}

func testScope() Scope {
	return Scope{ID: "manager:sales", Nodes: []NodeSelection{
		{ID: stmt("cancel"), Fields: []string{"argv", "count", "details", "requires"}, IncludePurpose: true, IncludeSource: true},
		{ID: core.DefinitionIdentity{APIVersion: testAPI, Kind: "Statement", Namespace: "inventory", Name: "release"}.Key()},
		{ID: "file:orders/cancel.go", Fields: []string{"path", "exists", "big", "empty", "absent"}},
	}, Edges: []EdgeKey{{From: stmt("cancel"), To: core.DefinitionIdentity{APIVersion: testAPI, Kind: "Statement", Namespace: "inventory", Name: "release"}.Key(), Property: "requires[0]"}, {From: "file:orders/cancel.go", To: stmt("cancel"), Property: "declares"}}}
}

func testFacts() ProjectFacts {
	return ProjectFacts{SnapshotDigest: "sha256:snapshot", ProjectDigest: "sha256:project", Digest: "sha256:files", Facts: []Fact{{ID: "file:orders/cancel.go", Kind: "File", State: FactKnown, Properties: map[string]json.RawMessage{"path": json.RawMessage(`"src/orders/cancel.go"`), "exists": json.RawMessage(`true`), "big": json.RawMessage(`9007199254740993`), "empty": json.RawMessage(`[]`), "privateTarget": json.RawMessage(`"must-not-be-selected"`)}}}, Relations: []Relation{{From: "file:orders/cancel.go", To: stmt("cancel"), Property: "declares", Source: core.Source{Path: "selector.yaml"}, Basis: "explicit declaration"}}}
}

func build(t *testing.T, m core.Model, f ProjectFacts, s Scope) *Index {
	t.Helper()
	i, err := Build(m, f, s)
	if err != nil {
		t.Fatal(err)
	}
	return i
}

func TestBuildDeterministicBindingsAndSourceProvenance(t *testing.T) {
	m := testModel(t, "rev-1", "cancel.yaml")
	i := build(t, m, testFacts(), testScope())
	if got := i.Binding().ModelDigest; got != m.Digest {
		t.Fatalf("model binding=%q", got)
	}
	s2 := testScope()
	for a, b := 0, len(s2.Nodes)-1; a < b; a, b = a+1, b-1 {
		s2.Nodes[a], s2.Nodes[b] = s2.Nodes[b], s2.Nodes[a]
	}
	for n := range s2.Nodes {
		if s2.Nodes[n].ID == stmt("cancel") {
			s2.Nodes[n].Fields = []string{"requires", "details", "count", "argv"}
		}
	}
	i2 := build(t, m, testFacts(), s2)
	if i.Binding().GraphDigest != i2.Binding().GraphDigest {
		t.Fatal("reordered scope changed graph digest")
	}
	changedSource := testModel(t, "rev-2", "renamed.yaml")
	if changedSource.Digest != m.Digest {
		t.Fatal("Core semantic digest unexpectedly includes source/revision")
	}
	i3 := build(t, changedSource, testFacts(), testScope())
	if i3.Binding().GraphDigest == i.Binding().GraphDigest {
		t.Fatal("selected source/revision change did not change graph digest")
	}
}

func TestProjectionRedactsReferencesAndPreservesLiterals(t *testing.T) {
	i := build(t, testModel(t, "rev-1", "cancel.yaml"), testFacts(), testScope())
	n, err := i.Node(stmt("cancel"))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := n.Properties["requires"]; ok {
		t.Fatal("reference value was duplicated as a node literal")
	}
	if n.Properties["details"] == nil {
		t.Fatal("selected property missing")
	}
	var details []map[string]any
	if err := json.Unmarshal(n.Properties["details"], &details); err != nil {
		t.Fatal(err)
	}
	if len(details) != 1 {
		t.Fatalf("repeated object values lost: %#v", details)
	}
	if _, ok := details[0]["links"]; ok {
		t.Fatal("nested reference field leaked")
	}
	var argv []string
	if err := json.Unmarshal(n.Properties["argv"], &argv); err != nil {
		t.Fatal(err)
	}
	if len(argv) != 4 || argv[0] != "go" || argv[2] != "go" {
		t.Fatalf("argv order/repetition lost: %#v", argv)
	}
	if got := string(n.Properties["count"]); got != "9007199254740993" {
		t.Fatalf("int64 precision lost: %s", got)
	}
	if _, err := i.Node(core.DefinitionIdentity{APIVersion: testAPI, Kind: "Statement", Namespace: "private", Name: "secret"}.Key()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("hidden node lookup: %v", err)
	}
	file, err := i.Node("file:orders/cancel.go")
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := file.Properties["privateTarget"]; ok {
		t.Fatal("unselected supplemental fact property leaked")
	}
	if _, ok := file.Properties["absent"]; ok {
		t.Fatal("absent fact property was materialized")
	}
	if string(file.Properties["empty"]) != "[]" {
		t.Fatalf("explicit empty property lost: %s", file.Properties["empty"])
	}
	if string(file.Properties["big"]) != "9007199254740993" {
		t.Fatalf("supplemental int64 precision lost: %s", file.Properties["big"])
	}
}

func TestIndexCopiesAreImmutableAndWalkProvidesForwardReverseWitnesses(t *testing.T) {
	i := build(t, testModel(t, "rev-1", "cancel.yaml"), testFacts(), testScope())
	n, _ := i.Node(stmt("cancel"))
	n.Properties["argv"][0] = 'x'
	all := i.Nodes()
	all[0].ID = "mutated"
	edges := i.Edges()
	edges[0].From = "mutated"
	if _, err := i.Node(stmt("cancel")); err != nil {
		t.Fatal("caller mutated index", err)
	}
	w, err := i.Walk(WalkRequest{Start: "file:orders/cancel.go", MaxDepth: 2, MaxSteps: 20, MaxResults: 10})
	if err != nil {
		t.Fatal(err)
	}
	if !w.Complete || len(w.Results) != 3 {
		t.Fatalf("forward walk: %+v", w)
	}
	r, err := i.Walk(WalkRequest{Start: stmt("cancel"), Reverse: true, MaxDepth: 2, MaxSteps: 20, MaxResults: 10})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, v := range r.Results {
		if v.Node == "file:orders/cancel.go" {
			found = true
			if len(v.Path) != 1 || v.Path[0].From != "file:orders/cancel.go" {
				t.Fatalf("reverse witness lost original edge: %+v", v)
			}
		}
	}
	if !found {
		t.Fatal("reverse traversal missed incoming relation")
	}
	var wg sync.WaitGroup
	for j := 0; j < 8; j++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := i.Walk(WalkRequest{Start: stmt("cancel"), Reverse: true, MaxDepth: 3, MaxSteps: 20, MaxResults: 10}); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
}

func TestExplicitScopeAndBoundedWalkFailures(t *testing.T) {
	m := testModel(t, "rev-1", "cancel.yaml")
	facts := testFacts()
	scope := testScope()
	scope.Nodes = nil
	if _, err := Build(m, facts, scope); err == nil {
		t.Fatal("empty scope was treated as all")
	}
	scope = testScope()
	scope.Edges[0].To = core.DefinitionIdentity{APIVersion: testAPI, Kind: "Statement", Namespace: "private", Name: "secret"}.Key()
	if _, err := Build(m, facts, scope); err == nil {
		t.Fatal("edge to absent selected node accepted")
	}
	facts = testFacts()
	facts.Relations = append(facts.Relations, Relation{From: stmt("cancel"), To: core.DefinitionIdentity{APIVersion: testAPI, Kind: "Statement", Namespace: "inventory", Name: "release"}.Key(), Property: "requires[0]", Basis: "conflicting adapter edge"})
	if _, err := Build(m, facts, testScope()); err == nil {
		t.Fatal("Core/fact duplicate edge conflict accepted")
	}
	facts = testFacts()
	i := build(t, m, facts, testScope())
	if _, err := i.Walk(WalkRequest{Start: stmt("cancel"), MaxDepth: 0, MaxSteps: 10, MaxResults: 2}); err == nil {
		t.Fatal("zero depth accepted")
	}
	if _, err := i.Walk(WalkRequest{Start: stmt("cancel"), MaxDepth: 33, MaxSteps: 10, MaxResults: 2}); err == nil {
		t.Fatal("depth over hard cap accepted")
	}
	partial, err := i.Walk(WalkRequest{Start: "file:orders/cancel.go", MaxDepth: 10, MaxSteps: 1, MaxResults: 4})
	if err != nil {
		t.Fatal(err)
	}
	if partial.Complete || partial.Reason != "step_limit" {
		t.Fatalf("bounded partial result not explicit: %+v", partial)
	}
	depth, err := i.Walk(WalkRequest{Start: "file:orders/cancel.go", MaxDepth: 1, MaxSteps: 10, MaxResults: 4})
	if err != nil {
		t.Fatal(err)
	}
	if depth.Complete || depth.Reason != "depth_limit" {
		t.Fatalf("depth truncation was reported complete: %+v", depth)
	}
}

func TestCycleFanoutAndResultBound(t *testing.T) {
	m := testModel(t, "rev-1", "cancel.yaml")
	facts := ProjectFacts{SnapshotDigest: "snapshot", Digest: "facts", Facts: []Fact{
		{ID: "a", Kind: "Fact", State: FactKnown}, {ID: "b", Kind: "Fact", State: FactPartial}, {ID: "c", Kind: "Fact", State: FactUnknown}, {ID: "d", Kind: "Fact", State: FactKnown},
	}, Relations: []Relation{
		{From: "a", To: "b", Property: "next"}, {From: "a", To: "c", Property: "next"}, {From: "a", To: "d", Property: "next"}, {From: "b", To: "a", Property: "back"},
	}}
	scope := Scope{ID: "cycle", Nodes: []NodeSelection{{ID: "a"}, {ID: "b"}, {ID: "c"}, {ID: "d"}}, Edges: []EdgeKey{{From: "a", To: "b", Property: "next"}, {From: "a", To: "c", Property: "next"}, {From: "a", To: "d", Property: "next"}, {From: "b", To: "a", Property: "back"}}}
	i := build(t, m, facts, scope)
	partial, err := i.Walk(WalkRequest{Start: "a", MaxDepth: 4, MaxSteps: 20, MaxResults: 2})
	if err != nil {
		t.Fatal(err)
	}
	if partial.Complete || partial.Reason != "result_limit" || len(partial.Results) != 2 {
		t.Fatalf("fanout result cap not explicit: %+v", partial)
	}
	complete, err := i.Walk(WalkRequest{Start: "a", MaxDepth: 4, MaxSteps: 20, MaxResults: 4})
	if err != nil {
		t.Fatal(err)
	}
	if !complete.Complete || len(complete.Results) != 4 {
		t.Fatalf("cycle traversal failed to terminate with one witness per node: %+v", complete)
	}
}

func TestLargeStarAndInputCapsAreBoundedAndDeterministic(t *testing.T) {
	m := testModel(t, "rev-1", "cancel.yaml")
	const fanout = 1_000
	facts := ProjectFacts{SnapshotDigest: "snapshot", Digest: "large-facts", Facts: make([]Fact, 0, fanout+1), Relations: make([]Relation, 0, fanout)}
	scope := Scope{ID: "large-star", Nodes: make([]NodeSelection, 0, fanout+1), Edges: make([]EdgeKey, 0, fanout)}
	facts.Facts = append(facts.Facts, Fact{ID: "root", Kind: "Fact", State: FactKnown})
	scope.Nodes = append(scope.Nodes, NodeSelection{ID: "root"})
	for n := 0; n < fanout; n++ {
		id := fmt.Sprintf("leaf:%04d", n)
		facts.Facts = append(facts.Facts, Fact{ID: id, Kind: "Fact", State: FactKnown})
		facts.Relations = append(facts.Relations, Relation{From: "root", To: id, Property: "child"})
		scope.Nodes = append(scope.Nodes, NodeSelection{ID: id})
		scope.Edges = append(scope.Edges, EdgeKey{From: "root", To: id, Property: "child"})
	}
	i := build(t, m, facts, scope)
	w, err := i.Walk(WalkRequest{Start: "root", MaxDepth: 2, MaxSteps: 5, MaxResults: 20})
	if err != nil {
		t.Fatal(err)
	}
	if w.Complete || w.Reason != "step_limit" || w.Steps != 5 {
		t.Fatalf("large fanout ignored the step bound: %+v", w)
	}
	first, err := i.Walk(WalkRequest{Start: "root", MaxDepth: 2, MaxSteps: 2_000, MaxResults: 1_001})
	if err != nil {
		t.Fatal(err)
	}
	second, err := i.Walk(WalkRequest{Start: "root", MaxDepth: 2, MaxSteps: 2_000, MaxResults: 1_001})
	if err != nil {
		t.Fatal(err)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) || !first.Complete || len(first.Results) != fanout+1 {
		t.Fatal("repeated bounded query was not deterministic")
	}

	tooMany := ProjectFacts{SnapshotDigest: "snapshot", Digest: "too-many", Facts: make([]Fact, maxNodes+1)}
	if _, err := Build(m, tooMany, Scope{ID: "limit", Nodes: []NodeSelection{{ID: "unused"}}}); err == nil {
		t.Fatal("fact count above hard node cap accepted")
	}
	large := json.RawMessage(mustJSON(t, strings.Repeat("x", maxPayloadBytes)))
	tooLarge := ProjectFacts{SnapshotDigest: "snapshot", Digest: "too-large", Facts: []Fact{{ID: "big", Kind: "Fact", State: FactKnown, Properties: map[string]json.RawMessage{"text": large}}}}
	if _, err := Build(m, tooLarge, Scope{ID: "payload", Nodes: []NodeSelection{{ID: "big", Fields: []string{"text"}}}}); err == nil {
		t.Fatal("fact payload above hard byte cap accepted")
	}
	largeLabel := strings.Repeat("x", maxPayloadBytes+1)
	for _, label := range []string{"model", "revision", "snapshot", "project", "facts", "scope"} {
		model := m
		bindingFacts := ProjectFacts{SnapshotDigest: "snapshot", ProjectDigest: "project", Digest: "facts"}
		s := Scope{ID: "scope", Nodes: []NodeSelection{{ID: "unused"}}}
		switch label {
		case "model":
			model.Digest = largeLabel
		case "revision":
			model.Revision = largeLabel
		case "snapshot":
			bindingFacts.SnapshotDigest = largeLabel
		case "project":
			bindingFacts.ProjectDigest = largeLabel
		case "facts":
			bindingFacts.Digest = largeLabel
		case "scope":
			s.ID = largeLabel
		}
		if _, err := Build(model, bindingFacts, s); err == nil {
			t.Fatalf("oversized %s binding label accepted", label)
		}
	}
	aggregateFacts := ProjectFacts{SnapshotDigest: strings.Repeat("s", 2<<20), ProjectDigest: strings.Repeat("p", 2<<20), Digest: strings.Repeat("f", 2<<20)}
	aggregateModel := m
	aggregateModel.Digest = strings.Repeat("m", 2<<20)
	if _, err := Build(aggregateModel, aggregateFacts, Scope{ID: "aggregate", Nodes: []NodeSelection{{ID: "unused"}}}); err == nil {
		t.Fatal("aggregate binding labels above payload cap accepted")
	}
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// This synthetic mapping only exercises graph navigation. It is not an
// observation of Shop inventory, execution, or runtime behavior.
func TestSyntheticShopCancellationOwnershipArtifactAndCheckNavigation(t *testing.T) {
	m := testModel(t, "shop-fixture-rev", "orders/cancel-order.yaml")
	cancel := stmt("cancel")
	release := core.DefinitionIdentity{APIVersion: testAPI, Kind: "Statement", Namespace: "inventory", Name: "release"}.Key()
	owner, artifact, check, file := "manager:inventory", "artifact:reservation-lifecycle", "check:cancellation-tests", "file:src/shop/inventory/reservations.go"
	facts := ProjectFacts{SnapshotDigest: "fixture-snapshot", ProjectDigest: "fixture-project", Digest: "fixture-facts", Facts: []Fact{
		{ID: owner, Kind: "Manager", State: FactKnown},
		{ID: artifact, Kind: "Artifact", State: FactKnown, Properties: map[string]json.RawMessage{"path": json.RawMessage(`"src/shop/inventory/"`)}},
		{ID: check, Kind: "Check", State: FactPartial},
		{ID: file, Kind: "File", State: FactKnown, Properties: map[string]json.RawMessage{"path": json.RawMessage(`"src/shop/inventory/reservations.go"`)}},
	}, Relations: []Relation{
		{From: release, To: owner, Property: "ownedBy", Source: core.Source{Path: "manager.yaml"}, Basis: "synthetic namespace ownership mapping"},
		{From: artifact, To: release, Property: "realizes[0]", Source: core.Source{Path: "reservation-lifecycle.yaml"}, Basis: "declared artifact relation"},
		{From: artifact, To: check, Property: "checks[0]", Source: core.Source{Path: "reservation-lifecycle.yaml"}, Basis: "declared check relation"},
		{From: file, To: artifact, Property: "memberOf", Source: core.Source{Path: "selectors.yaml"}, Basis: "synthetic explicit path selector"},
	}}
	scope := Scope{ID: "synthetic-shop-cancellation", Nodes: []NodeSelection{{ID: cancel}, {ID: release}, {ID: owner}, {ID: artifact, Fields: []string{"path"}}, {ID: check}, {ID: file, Fields: []string{"path"}}}, Edges: []EdgeKey{
		{From: cancel, To: release, Property: "requires[0]"},
		{From: release, To: owner, Property: "ownedBy"},
		{From: artifact, To: release, Property: "realizes[0]"},
		{From: artifact, To: check, Property: "checks[0]"},
		{From: file, To: artifact, Property: "memberOf"},
	}}
	i := build(t, m, facts, scope)
	walk := func(start string, reverse bool, target string, wantPathLen int) {
		t.Helper()
		result, err := i.Walk(WalkRequest{Start: start, Reverse: reverse, MaxDepth: 5, MaxSteps: 100, MaxResults: 20})
		if err != nil {
			t.Fatal(err)
		}
		for _, witness := range result.Results {
			if witness.Node == target {
				if len(witness.Path) != wantPathLen {
					t.Fatalf("witness %s->%s has path len %d: %+v", start, target, len(witness.Path), witness.Path)
				}
				return
			}
		}
		t.Fatalf("no witness from %s to %s (reverse=%v)", start, target, reverse)
	}
	walk(cancel, false, owner, 2)    // cancellation requires release, mapped to Inventory owner
	walk(release, true, artifact, 1) // reverse realization witness retains artifact -> release orientation
	walk(artifact, false, check, 1)
	walk(file, false, release, 2) // file membership -> artifact realization -> release
	walk(release, true, file, 2)
}

func TestDecisionProjectionRetainsReasonAndTypesReferencesAsEdges(t *testing.T) {
	statementRef := core.Property{Purpose: "subject", Type: core.TypeReference, Target: &core.KindIdentity{APIVersion: testAPI, Kind: "Statement"}, MinCount: 1, MaxCount: 1}
	managerRef := core.Property{Purpose: "actor", Type: core.TypeReference, Target: &core.KindIdentity{APIVersion: testAPI, Kind: "Manager"}, MinCount: 1, MaxCount: 1}
	str := func(p string) core.Property {
		return core.Property{Purpose: p, Type: core.TypeString, MinCount: 1, MaxCount: 1}
	}
	optionalString := core.Property{Purpose: "Private child instructions.", Type: core.TypeString, MinCount: 0, MaxCount: 1}
	schema := core.Schema{APIVersion: testAPI, Purpose: "Decision fixture", Kinds: map[string]core.Kind{
		"Statement": {Purpose: "rule", Properties: map[string]core.Property{}},
		"Manager":   {Purpose: "owner", Properties: map[string]core.Property{"instructions": optionalString}},
		"Decision":  {Purpose: "recorded decision", Properties: map[string]core.Property{"subject": statementRef, "actor": managerRef, "decision": str("decision text"), "reason": str("decision reason")}},
	}}
	defs := []core.Definition{
		{APIVersion: testAPI, Kind: "Statement", Metadata: core.Metadata{Namespace: "inventory", Name: "release"}, Purpose: "Release reservations", Spec: map[string]any{}},
		{APIVersion: testAPI, Kind: "Manager", Metadata: core.Metadata{Namespace: "inventory", Name: "inventory"}, Purpose: "Inventory owner", Spec: map[string]any{"instructions": "child private instructions"}},
		{APIVersion: testAPI, Kind: "Decision", Metadata: core.Metadata{Namespace: "inventory", Name: "D-1"}, Purpose: "Recorded rationale", Spec: map[string]any{"subject": map[string]any{"namespace": "inventory", "name": "release"}, "actor": map[string]any{"namespace": "inventory", "name": "inventory"}, "decision": "Keep release reservation checks", "reason": "Prevent double allocation"}},
	}
	m, diagnostics := core.Compile([]core.Schema{schema}, defs, "decision-rev")
	if len(diagnostics) > 0 {
		t.Fatalf("compile: %+v", diagnostics)
	}
	did := defs[2].Identity().Key()
	sid := defs[0].Identity().Key()
	mid := defs[1].Identity().Key()
	i, err := Build(m, ProjectFacts{SnapshotDigest: "snapshot", Digest: "facts"}, Scope{ID: "decision-review", Nodes: []NodeSelection{{ID: did, Fields: []string{"decision", "reason", "subject", "actor"}, IncludePurpose: true}, {ID: sid}, {ID: mid}}, Edges: []EdgeKey{{From: did, To: sid, Property: "subject"}, {From: did, To: mid, Property: "actor"}}})
	if err != nil {
		t.Fatal(err)
	}
	n, err := i.Node(did)
	if err != nil {
		t.Fatal(err)
	}
	if string(n.Properties["reason"]) != "\"Prevent double allocation\"" {
		t.Fatalf("decision reason lost: %s", n.Properties["reason"])
	}
	if _, ok := n.Properties["subject"]; ok {
		t.Fatal("Decision subject duplicated in literals")
	}
	if _, ok := n.Properties["actor"]; ok {
		t.Fatal("Decision actor duplicated in literals")
	}
	manager, err := i.Node(mid)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := manager.Properties["instructions"]; ok {
		t.Fatal("unselected child instructions leaked")
	}
}
