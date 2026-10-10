package projectmodel

import (
	"encoding/json"
	"errors"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

// A Manager's graph has exactly one edge per reference its Context shows, so
// it neither drops a relation nor adds one Context does not grant.
func TestManagerGraphHasOneEdgePerContextReference(t *testing.T) {
	for seed := uint64(1); seed <= 100; seed++ {
		rng := rand.New(rand.NewPCG(seed, 17))
		report := generateProject(rng).analyze(t, nil)
		for _, m := range report.Managers {
			ctx, err := Context(report, m.ID)
			if err != nil {
				t.Fatal(err)
			}
			want := map[GraphEdge]bool{}
			add := func(from, relation, to string) {
				if from != "" && to != "" {
					want[GraphEdge{From: from, Relation: relation, To: to}] = true
				}
			}
			for _, manager := range append([]Manager{ctx.Manager}, ctx.Children...) {
				add(manager.ID, "parent", manager.Parent)
			}
			for _, s := range append(append([]Statement(nil), ctx.Statements...), ctx.Contracts...) {
				add(s.ID, "owned by", s.Owner)
				for _, id := range s.Uses {
					add(s.ID, "uses", id)
				}
				for _, id := range s.Requires {
					add(s.ID, "requires", id)
				}
			}
			for _, a := range ctx.Artifacts {
				add(a.ID, "owned by", a.Owner)
				for _, id := range a.Realizes {
					add(a.ID, "realizes", id)
				}
				for _, id := range a.Checks {
					add(a.ID, "checked by", id)
				}
				for _, p := range a.Paths {
					add(a.ID, "expects", p)
				}
			}
			for _, c := range ctx.Checks {
				add(c.ID, "owned by", c.Owner)
				for _, id := range c.Uses {
					add(c.ID, "exercises", id)
				}
			}
			for _, d := range ctx.Decisions {
				add(d.ID, "owned by", d.Owner)
				add(d.ID, "decides on", d.Subject)
				add(d.ID, "actor", d.Actor)
			}
			graph := ManagerGraph(ctx)
			if len(graph.Edges) != len(want) {
				t.Fatalf("seed %d, %s: %d edges, Context has %d references", seed, m.ID, len(graph.Edges), len(want))
			}
			for _, e := range graph.Edges {
				if !want[e] {
					t.Fatalf("seed %d, %s: edge %+v is not a Context reference", seed, m.ID, e)
				}
			}
		}
	}
}

// No Manager's Context, graph, trace or explanation shows a Statement private
// to another Manager or another Manager's Decision, and its explanation shows
// only files it owns or its own Artifacts expect. Generated projects make such
// statements and decisions, reference them privately and change them.
func TestManagerViewsShowOnlyWhatTheManagerMaySee(t *testing.T) {
	checked, decisions := 0, 0
	for seed := uint64(1); seed <= 150; seed++ {
		rng := rand.New(rand.NewPCG(seed, 19))
		project := generateProject(rng)
		base := project.analyze(t, nil)
		candidate := project.with(randomEdits(rng, project)...).analyze(t, nil)
		for _, m := range base.Managers {
			var private []string
			for _, s := range append(append([]Statement(nil), base.Statements...), candidate.Statements...) {
				if !s.Public && s.Owner != m.ID {
					private = append(private, s.ID)
				}
			}
			var hidden []string
			for _, d := range append(append([]Decision(nil), base.Decisions...), candidate.Decisions...) {
				if d.Owner != m.ID {
					hidden = append(hidden, d.ID)
				}
			}
			mayShow := map[string]bool{}
			for _, r := range []Report{base, candidate} {
				for _, f := range r.Files {
					if f.Owner == m.ID {
						mayShow[f.Path] = true
					}
				}
				for _, a := range r.Artifacts {
					if a.Owner == m.ID {
						for _, p := range a.Paths {
							mayShow[p] = true
						}
					}
				}
			}
			ctx, err := Context(candidate, m.ID)
			if err != nil {
				t.Fatal(err)
			}
			graph := ManagerGraph(ctx)
			explanation, err := ExplainForManager(base, candidate, m.ID)
			if err != nil {
				t.Fatal(err)
			}
			views := map[string]any{"context": ctx, "graph": graph, "explanation": explanation}
			for _, e := range explanation.Elements {
				files := []string{}
				if e.Kind == "file" {
					files = append(files, e.ID)
				}
				for _, step := range e.Witness {
					if step.FromKind == "file" {
						files = append(files, step.From)
					}
				}
				for _, f := range files {
					if !mayShow[f] {
						t.Fatalf("seed %d: %s's explanation shows file %s it neither owns nor expects", seed, m.ID, f)
					}
				}
			}
			for _, n := range graph.Nodes {
				trace, err := graph.Trace(TraceRequest{From: n.ID, Direction: "both"})
				if err != nil {
					t.Fatal(err)
				}
				views["trace from "+n.ID] = trace
			}
			for name, view := range views {
				data, _ := json.Marshal(view)
				for _, id := range append(append([]string(nil), private...), hidden...) {
					if strings.Contains(string(data), jsonText(id)) {
						t.Fatalf("seed %d: %s's %s shows %s, which it may not see", seed, m.ID, name, id)
					}
				}
			}
			decisions += len(hidden)
			for _, id := range private {
				if _, err := graph.Trace(TraceRequest{From: id}); !errors.Is(err, ErrNodeNotFound) {
					t.Fatalf("seed %d: tracing hidden %s from %s gave %v, want ErrNodeNotFound", seed, id, m.ID, err)
				}
				checked++
			}
		}
	}
	if checked < 100 || decisions < 100 {
		t.Fatalf("only %d private statements and %d foreign decisions were checked", checked, decisions)
	}
}

// Seen from Orders, a change to Inventory's private guard reaches Orders'
// statement through the public contract; the witness starts at the contract.
func TestExplainForManagerCutsWitnessesAtHiddenElements(t *testing.T) {
	model, files := fixture(t, true, true, true)
	guardRef := map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": "inventory", "name": "internal-guard"}
	definitions := append(copyDefinitions(model.Definitions), core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "inventory", Name: "internal-guard"}, Purpose: "Private detail.", Spec: map[string]any{"category": "rule", "description": "Guard."}})
	for i := range definitions {
		if definitions[i].Metadata.Name == "release-reservation" {
			definitions[i].Spec["uses"] = []any{guardRef}
		}
	}
	analyze := func(description string) Report {
		for i := range definitions {
			if definitions[i].Metadata.Name == "internal-guard" {
				definitions[i].Spec["description"] = description
			}
		}
		compiled, diagnostics := core.Compile(model.Schemas, copyDefinitions(definitions), "guard")
		if len(diagnostics) != 0 {
			t.Fatalf("compile: %+v", diagnostics)
		}
		return Analyze(compiled, files)
	}
	base, candidate := analyze("Guard."), analyze("Guard harder.")
	key := func(kind, namespace, name string) string {
		return (core.DefinitionIdentity{APIVersion: APIVersion, Kind: kind, Namespace: namespace, Name: name}).Key()
	}
	guard, contract, cancel := key(statementKind, "inventory", "internal-guard"), key(statementKind, "inventory", "release-reservation"), key(statementKind, "orders", "cancel-order")
	orders, err := ExplainForManager(base, candidate, key(managerKind, "orders", "orders"))
	if err != nil {
		t.Fatal(err)
	}
	data, _ := json.Marshal(orders)
	if strings.Contains(string(data), jsonText(guard)) {
		t.Fatalf("orders sees the private guard: %s", data)
	}
	var found bool
	for _, e := range orders.Elements {
		if e.ID == cancel {
			found = true
			if !e.Partial || len(e.Witness) == 0 || e.Witness[0].From != contract {
				t.Fatalf("cancel-order witness = %+v (partial %v), want it to start at the public contract", e.Witness, e.Partial)
			}
		}
	}
	if !found {
		t.Fatal("orders' explanation lacks its own affected statement")
	}
	inventory, err := ExplainForManager(base, candidate, key(managerKind, "inventory", "inventory"))
	if err != nil {
		t.Fatal(err)
	}
	if !slices.ContainsFunc(inventory.Elements, func(e ExplainedElement) bool { return e.ID == guard && e.Reason == "changed" }) {
		t.Fatalf("inventory does not see its own changed guard: %+v", inventory.Elements)
	}
	if _, err := ExplainForManager(base, candidate, "missing"); !errors.Is(err, ErrManagerNotFound) {
		t.Fatalf("unknown manager error = %v", err)
	}
}

// Trace follows edges breadth first within its limits and reports when a
// limit cut it short.
func TestTraceRespectsDirectionAndLimits(t *testing.T) {
	statement := func(uses ...int) generatedStatement {
		return generatedStatement{description: "Rule.", purpose: "Statement.", uses: uses, public: true}
	}
	// sa uses sb, sb uses sc, sc uses sd.
	report := generatedProject{managers: []string{""}, statements: []generatedStatement{statement(1), statement(2), statement(3), statement()}}.analyze(t, nil)
	graph := ProjectGraph(report)
	id := func(name string) string {
		return (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Name: name}).Key()
	}
	full, err := graph.Trace(TraceRequest{From: id("sa"), Direction: "out"})
	if err != nil {
		t.Fatal(err)
	}
	var last TracedNode
	for _, n := range full.Reached {
		if n.Node.ID == id("sd") {
			last = n
		}
	}
	if !full.Complete || last.Depth != 3 || len(last.Witness) != 3 || last.Witness[2].To != id("sd") {
		t.Fatalf("full trace = complete %v, sd at depth %d with witness %+v", full.Complete, last.Depth, last.Witness)
	}
	short, _ := graph.Trace(TraceRequest{From: id("sa"), Direction: "out", MaxDepth: 2})
	if short.Complete || slices.ContainsFunc(short.Reached, func(n TracedNode) bool { return n.Node.ID == id("sd") }) {
		t.Fatalf("depth-limited trace = complete %v, reached %+v", short.Complete, short.Reached)
	}
	back, _ := graph.Trace(TraceRequest{From: id("sd"), Direction: "in"})
	if !slices.ContainsFunc(back.Reached, func(n TracedNode) bool { return n.Node.ID == id("sa") }) {
		t.Fatalf("incoming trace did not reach sa: %+v", back.Reached)
	}
	again, _ := graph.Trace(TraceRequest{From: id("sa"), Direction: "both"})
	if repeat, _ := graph.Trace(TraceRequest{From: id("sa"), Direction: "both"}); digest(repeat) != digest(again) {
		t.Fatal("trace is not deterministic")
	}
	if _, err := graph.Trace(TraceRequest{From: "absent"}); !errors.Is(err, ErrNodeNotFound) {
		t.Fatalf("absent start error = %v", err)
	}
}

// jsonText is s as it appears inside a JSON string, so a search in marshalled
// output finds identity keys, whose quotes JSON escapes.
func jsonText(s string) string {
	data, _ := json.Marshal(s)
	return string(data[1 : len(data)-1])
}

// A file is visible to the Manager that owns it, as for runs and reviews, not
// to an ancestor whose wider selector the file falls under.
func TestExplainForManagerShowsOnlyOwnedFiles(t *testing.T) {
	model, files := fixture(t, true, true, true)
	definitions := append(copyDefinitions(model.Definitions), core.Definition{APIVersion: APIVersion, Kind: managerKind, Metadata: core.Metadata{Namespace: "orders.returns", Name: "returns"}, Purpose: "Returns.", Spec: map[string]any{
		"parent": map[string]any{"apiVersion": APIVersion, "kind": managerKind, "namespace": "orders", "name": "orders"}, "owns": []any{"src/orders/returns/"},
	}})
	compiled, diagnostics := core.Compile(model.Schemas, definitions, "returns")
	if len(diagnostics) != 0 {
		t.Fatalf("compile: %+v", diagnostics)
	}
	const secret = "src/orders/returns/secret-refund.go"
	base := Analyze(compiled, append(append([]File(nil), files...), File{Path: secret, Digest: "sha256:a", Mode: "100644"}))
	candidate := Analyze(compiled, append(append([]File(nil), files...), File{Path: secret, Digest: "sha256:b", Mode: "100644"}))
	key := func(namespace, name string) string {
		return (core.DefinitionIdentity{APIVersion: APIVersion, Kind: managerKind, Namespace: namespace, Name: name}).Key()
	}
	for manager, visible := range map[string]bool{key("orders.returns", "returns"): true, key("orders", "orders"): false, rootManagerKey(): false} {
		explanation, err := ExplainForManager(base, candidate, manager)
		if err != nil {
			t.Fatal(err)
		}
		if shown := slices.ContainsFunc(explanation.Elements, func(e ExplainedElement) bool { return e.ID == secret }); shown != visible {
			t.Fatalf("%s sees %s: %v, want %v", manager, secret, shown, visible)
		}
	}
}
