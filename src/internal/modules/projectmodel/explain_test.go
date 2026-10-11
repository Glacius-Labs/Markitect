package projectmodel

import (
	"math/rand/v2"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

// DEC-023: Explain covers exactly the elements of the impact, each with a
// witness built only from real model relations, and is deterministic.
func TestExplainCoversTheImpactWithRealWitnesses(t *testing.T) {
	explained := 0
	for seed := uint64(1); seed <= 200; seed++ {
		rng := rand.New(rand.NewPCG(seed, 13))
		base := generateProject(rng)
		b, c := base.analyze(t, nil), base.with(randomEdits(rng, base)...).analyze(t, nil)
		impact, explanation := Impact(b, c), Explain(b, c)
		if explanation.ImpactDigest != impact.Digest {
			t.Fatalf("seed %d: explanation is bound to %s, impact is %s", seed, explanation.ImpactDigest, impact.Digest)
		}
		byKind := map[string][]string{}
		for _, e := range explanation.Elements {
			byKind[e.Kind] = append(byKind[e.Kind], e.ID)
			if e.Reason == "unexplained" || e.Class != "change" && e.Class != "context" {
				t.Fatalf("seed %d: %s %s has reason %q and class %q", seed, e.Kind, e.ID, e.Reason, e.Class)
			}
			assertRealWitness(t, b, c, e)
			explained++
		}
		for kind, want := range map[string][]string{"statement": impact.AffectedStatements, "manager": impact.Managers, "file": impact.Files, "check": impact.Checks} {
			if !slices.Equal(byKind[kind], want) && len(byKind[kind])+len(want) > 0 {
				t.Fatalf("seed %d: explained %ss %v, impact has %v", seed, kind, byKind[kind], want)
			}
		}
		if again := Explain(shuffledReport(rng, b), shuffledReport(rng, c)); again.Digest != explanation.Digest {
			t.Fatalf("seed %d: explanation depends on report order", seed)
		}
	}
	if explained < 1000 {
		t.Fatalf("only %d elements explained over 200 generated changes", explained)
	}
}

// assertRealWitness checks that the witness is a connected path ending at the
// element and that every statement and artifact step is a declared relation.
func assertRealWitness(t *testing.T, base, candidate Report, e ExplainedElement) {
	t.Helper()
	declares := func(relation, from, to string) bool {
		for _, r := range []Report{base, candidate} {
			for _, s := range r.Statements {
				switch {
				case relation == "uses" && s.ID == from && contains(s.Uses, to),
					relation == "requires" && s.ID == from && contains(s.Requires, to),
					relation == "used by" && s.ID == to && contains(s.Uses, from),
					relation == "required by" && s.ID == to && contains(s.Requires, from):
					return true
				}
			}
			for _, a := range r.Artifacts {
				switch {
				case relation == "realized by" && a.ID == to && contains(a.Realizes, from),
					relation == "realizes" && a.ID == from && contains(a.Realizes, to),
					relation == "expects" && a.ID == from && contains(a.Paths, to),
					relation == "checked by" && a.ID == from && contains(a.Checks, to):
					return true
				}
			}
			for _, c := range r.Checks {
				switch {
				case relation == "exercised by" && c.ID == to && contains(c.Uses, from),
					relation == "exercises" && c.ID == from && contains(c.Uses, to):
					return true
				}
			}
		}
		return false
	}
	modelRelations := map[string]bool{"uses": true, "requires": true, "used by": true, "required by": true, "realized by": true, "realizes": true, "expects": true, "checked by": true, "exercised by": true, "exercises": true}
	for i, step := range e.Witness {
		if i > 0 && (e.Witness[i-1].To != step.From || e.Witness[i-1].ToKind != step.FromKind) {
			t.Fatalf("%s %s: witness is not connected: %+v", e.Kind, e.ID, e.Witness)
		}
		if modelRelations[step.Relation] && !declares(step.Relation, step.From, step.To) {
			t.Fatalf("%s %s: witness step %+v is not a declared relation", e.Kind, e.ID, step)
		}
	}
	if n := len(e.Witness); n > 0 && (e.Witness[n-1].To != e.ID || e.Witness[n-1].ToKind != e.Kind) {
		t.Fatalf("%s %s: witness does not end at the element: %+v", e.Kind, e.ID, e.Witness)
	}
}

// DEC-023: a changed contract, its direct consumers and their realizations
// must change; what the contract only uses, transitive consumers and ancestor
// Managers are context.
func TestExplainClassifiesChangeAndContext(t *testing.T) {
	model, files := fixture(t, true, true, true)
	ref := func(namespace, name string) map[string]any {
		return map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": namespace, "name": name}
	}
	definitions := append(copyDefinitions(model.Definitions),
		core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "inventory", Name: "reservation"}, Purpose: "Reservation concept.", Spec: map[string]any{"category": "concept", "description": "Held stock.", "public": true}},
		core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "operator-workflow"}, Purpose: "Operator flow.", Spec: map[string]any{"category": "workflow", "description": "Follow cancellation.", "uses": []any{ref("orders", "cancel-order")}}},
	)
	for i := range definitions {
		if definitions[i].Metadata.Name == "release-reservation" {
			definitions[i].Spec["uses"] = []any{ref("inventory", "reservation")}
		}
	}
	analyze := func(description string) Report {
		for i := range definitions {
			if definitions[i].Metadata.Name == "release-reservation" {
				definitions[i].Spec["description"] = description
			}
		}
		compiled, diagnostics := core.Compile(model.Schemas, copyDefinitions(definitions), "classes")
		if len(diagnostics) != 0 {
			t.Fatalf("compile: %+v", diagnostics)
		}
		return Analyze(compiled, files)
	}
	base := analyze("Release reservation once.")
	explanation := Explain(base, analyze("Release reservation at most once."))
	key := func(kind, namespace, name string) string {
		return (core.DefinitionIdentity{APIVersion: APIVersion, Kind: kind, Namespace: namespace, Name: name}).Key()
	}
	want := map[string]string{
		key(statementKind, "inventory", "release-reservation"): "change",
		key(statementKind, "orders", "cancel-order"):           "change",
		"src/inventory/release.go":                             "change",
		"src/orders/cancel.go":                                 "change",
		key(statementKind, "inventory", "reservation"):         "context",
		key(statementKind, "orders", "operator-workflow"):      "context",
		rootManagerKey():                                       "context",
	}
	got := map[string]ExplainedElement{}
	for _, e := range explanation.Elements {
		got[e.ID] = e
	}
	for id, class := range want {
		if got[id].Class != class {
			t.Fatalf("%s is %q (reason %q, witness %+v), want %q", id, got[id].Class, got[id].Reason, got[id].Witness, class)
		}
	}
	cancel := got[key(statementKind, "orders", "cancel-order")]
	if cancel.Reason != "consumer" || len(cancel.Witness) != 1 || cancel.Witness[0].Relation != "required by" || cancel.Witness[0].From != key(statementKind, "inventory", "release-reservation") {
		t.Fatalf("cancel-order is not explained as a direct consumer: %+v", cancel)
	}
}

// DEC-023 with DEC-021: an edit that changes only how a statement is written
// makes that statement and its file change, but not the realization the
// statement has as transitive context of another change.
func TestExplainKeepsRewriteFromSpreadingTheChangeClass(t *testing.T) {
	model, files := fixture(t, true, true, true)
	ref := func(kind, namespace, name string) map[string]any {
		return map[string]any{"apiVersion": APIVersion, "kind": kind, "namespace": namespace, "name": name}
	}
	definitions := append(copyDefinitions(model.Definitions),
		core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "s"}, Purpose: "Follows cancellation.", Spec: map[string]any{"category": "rule", "description": "Follow.", "requires": []any{ref(statementKind, "orders", "cancel-order")}}},
		core.Definition{APIVersion: APIVersion, Kind: artifactKind, Metadata: core.Metadata{Namespace: "orders", Name: "s-code"}, Purpose: "Realizes s.", Spec: map[string]any{"role": "implementation", "realizes": []any{ref(statementKind, "orders", "s")}, "paths": []any{"src/orders/s.go"}}},
	)
	files = append(files, File{Path: "src/orders/s.go", Digest: "sha256:s", Mode: "100644"})
	analyze := func(description string, explicitPrivate bool) Report {
		changed := copyDefinitions(definitions)
		for i := range changed {
			switch changed[i].Metadata.Name {
			case "release-reservation":
				changed[i].Spec["description"] = description
			case "s":
				if explicitPrivate {
					changed[i].Spec["public"] = false
				}
			}
		}
		compiled, diagnostics := core.Compile(model.Schemas, changed, "rewrite-class")
		if len(diagnostics) != 0 {
			t.Fatalf("compile: %+v", diagnostics)
		}
		return Analyze(compiled, files)
	}
	base := analyze("Release reservation once.", false)
	sID := (core.DefinitionIdentity{APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "s"}).Key()
	for name, candidate := range map[string]Report{
		"contract change only":            analyze("Release reservation at most once.", false),
		"contract change and rewritten s": analyze("Release reservation at most once.", true),
	} {
		classes := map[string]string{}
		for _, e := range Explain(base, candidate).Elements {
			classes[e.ID] = e.Class
		}
		if classes["src/orders/s.go"] != "context" {
			t.Fatalf("%s: src/orders/s.go is %q, want context", name, classes["src/orders/s.go"])
		}
		if name == "contract change and rewritten s" && classes[sID] != "change" {
			t.Fatalf("%s: rewritten s is %q, want change", name, classes[sID])
		}
	}
}

// BUG-01: the class does not depend on how definitions are split into model
// files. A transitive consumer that shares the changed Statement's file is
// itself change, but its realization stays context, as with separate files.
func TestExplainClassIgnoresSharedModelFiles(t *testing.T) {
	for _, shared := range []bool{false, true} {
		definitions, model, files := sourcedFixture(t)
		ref := func(namespace, name string) map[string]any {
			return map[string]any{"apiVersion": APIVersion, "kind": statementKind, "namespace": namespace, "name": name}
		}
		workflowFile := ".markitect/model/orders/operator-workflow.yaml"
		for _, d := range definitions {
			if shared && d.Metadata.Name == "release-reservation" {
				workflowFile = d.Source.Path
			}
		}
		definitions = append(definitions,
			core.Definition{APIVersion: APIVersion, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "operator-workflow"}, Purpose: "Operator flow.", Spec: map[string]any{"category": "workflow", "description": "Follow cancellation.", "uses": []any{ref("orders", "cancel-order")}},
				Source: core.Source{Path: workflowFile}},
			core.Definition{APIVersion: APIVersion, Kind: artifactKind, Metadata: core.Metadata{Namespace: "orders", Name: "workflow-doc"}, Purpose: "Documents the flow.", Spec: map[string]any{"role": "documentation", "realizes": []any{ref("orders", "operator-workflow")}, "paths": []any{"src/orders/workflow.md"}},
				Source: core.Source{Path: ".markitect/model/orders/workflow-doc.yaml"}},
		)
		files = append(files, File{Path: "src/orders/workflow.md", Digest: "sha256:doc", Mode: "100644"})
		analyze := func(description string) Report {
			changed := copyDefinitions(definitions)
			for i := range changed {
				if changed[i].Metadata.Name == "release-reservation" {
					changed[i].Spec["description"] = description
				}
			}
			return analyzeDefinitions(t, model, changed, files)
		}
		for _, e := range Explain(analyze("Release reservation once."), analyze("Release reservation at most once.")).Elements {
			if e.ID == "src/orders/workflow.md" && e.Class != "context" {
				t.Fatalf("shared file %v: the transitive consumer's realization is %q (reason %q), want context", shared, e.Class, e.Reason)
			}
		}
	}
}
