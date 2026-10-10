package projectmodel

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

// diamondFixture declares two Statements that change together while a third
// one sits between them: changed-a uses between, and between uses changed-b.
// between is reachable from changed-a as used context and from changed-b as a
// direct consumer, so its realization must not depend on which seed Impact
// visits first.
func diamondFixture(t *testing.T) (Report, Report) {
	t.Helper()
	api := APIVersion
	ref := func(kind, namespace, name string) map[string]any {
		return map[string]any{"apiVersion": api, "kind": kind, "namespace": namespace, "name": name}
	}
	definitions := func(description string) []core.Definition {
		return []core.Definition{
			{APIVersion: api, Kind: managerKind, Metadata: core.Metadata{Name: "root"}, Purpose: "Root project manager.", Spec: map[string]any{"owns": []any{"."}}},
			{APIVersion: api, Kind: managerKind, Metadata: core.Metadata{Namespace: "orders", Name: "orders"}, Purpose: "Orders manager.", Spec: map[string]any{"parent": ref(managerKind, "", "root"), "owns": []any{"src/orders/"}}},
			{APIVersion: api, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "changed-a"}, Purpose: "First changed statement.", Spec: map[string]any{"category": "rule", "description": description, "uses": []any{ref(statementKind, "orders", "between")}}},
			{APIVersion: api, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "changed-b"}, Purpose: "Second changed statement.", Spec: map[string]any{"category": "rule", "description": description}},
			{APIVersion: api, Kind: statementKind, Metadata: core.Metadata{Namespace: "orders", Name: "between"}, Purpose: "Statement between the changes.", Spec: map[string]any{"category": "rule", "description": "Unchanged.", "uses": []any{ref(statementKind, "orders", "changed-b")}}},
			{APIVersion: api, Kind: checkKind, Metadata: core.Metadata{Namespace: "orders", Name: "between-tests"}, Purpose: "Run the between tests.", Spec: map[string]any{"command": []any{"go", "test", "./orders"}}},
			{APIVersion: api, Kind: artifactKind, Metadata: core.Metadata{Namespace: "orders", Name: "between-code"}, Purpose: "Between implementation.", Spec: map[string]any{"role": "implementation", "realizes": []any{ref(statementKind, "orders", "between")}, "paths": []any{"src/orders/between.go"}, "checks": []any{ref(checkKind, "orders", "between-tests")}}},
		}
	}
	files := []File{
		{Path: ".markitect/project.yaml", Digest: "sha256:manifest", Mode: "100644"},
		{Path: "src/orders/between.go", Digest: "sha256:between", Mode: "100644"},
	}
	analyze := func(description string) Report {
		model, diagnostics := core.Compile([]core.Schema{Schema()}, definitions(description), "diamond")
		if len(diagnostics) != 0 {
			t.Fatalf("compile diamond fixture: %+v", diagnostics)
		}
		r := Analyze(model, files)
		if r.Status != "succeeded" {
			t.Fatalf("diamond fixture did not analyze cleanly: %s %+v", r.Status, r.Findings)
		}
		return r
	}
	return analyze("Before."), analyze("After.")
}

func TestImpactCoverageIsIndependentOfSeedOrder(t *testing.T) {
	base, candidate := diamondFixture(t)
	first := Impact(base, candidate)
	for i := 0; i < 200; i++ {
		if again := Impact(base, candidate); again.Digest != first.Digest {
			t.Fatalf("run %d changed the impact:\nfirst files=%v checks=%v\nagain files=%v checks=%v", i, first.Files, first.Checks, again.Files, again.Checks)
		}
	}
	if !contains(first.Files, "src/orders/between.go") {
		t.Fatalf("direct consumer of a changed statement lost its realization: files=%v", first.Files)
	}
}

func TestImpactIsIndependentOfReportOrder(t *testing.T) {
	base, candidate := diamondFixture(t)
	want := Impact(base, candidate)
	rng := rand.New(rand.NewPCG(1, 2))
	for i := 0; i < 200; i++ {
		got := Impact(shuffledReport(rng, base), shuffledReport(rng, candidate))
		if got.Digest != want.Digest {
			t.Fatalf("permutation %d changed the impact:\nwant files=%v checks=%v\ngot  files=%v checks=%v", i, want.Files, want.Checks, got.Files, got.Checks)
		}
	}
}

// Every part of every definition is either projected, so Impact names the
// definition as changed, or unprojected, so Impact widens to the declared
// project. Each edit is paired with an unrelated projected change, next to
// which an unprojected change used to be lost.
func TestImpactSeesEveryDefinitionPropertyChange(t *testing.T) {
	model, files := fixture(t, true, true, true)
	ref := func(kind, namespace, name string) map[string]any {
		return map[string]any{"apiVersion": APIVersion, "kind": kind, "namespace": namespace, "name": name}
	}
	contract := ref(statementKind, "inventory", "release-reservation")
	decision := core.Definition{APIVersion: APIVersion, Kind: decisionKind, Metadata: core.Metadata{Namespace: "inventory", Name: "release-once"}, Purpose: "Why reservations are released once.", Spec: map[string]any{
		"subject": contract, "decision": "Release each reservation exactly once.", "reason": "Double release corrupts stock.", "actor": ref(managerKind, "inventory", "inventory"),
	}}
	baseModel, diagnostics := core.Compile(model.Schemas, append(copyDefinitions(model.Definitions), decision), "base")
	if len(diagnostics) != 0 {
		t.Fatalf("compile base: %+v", diagnostics)
	}
	base := Analyze(baseModel, files)
	targets := map[string]core.DefinitionIdentity{
		managerKind:   {APIVersion: APIVersion, Kind: managerKind, Namespace: "orders", Name: "orders"},
		statementKind: {APIVersion: APIVersion, Kind: statementKind, Namespace: "orders", Name: "cancel-order"},
		artifactKind:  {APIVersion: APIVersion, Kind: artifactKind, Namespace: "orders", Name: "cancel-order-code"},
		checkKind:     {APIVersion: APIVersion, Kind: checkKind, Namespace: "orders", Name: "cancel-order-tests"},
		decisionKind:  {APIVersion: APIVersion, Kind: decisionKind, Namespace: "inventory", Name: "release-once"},
	}
	edits := map[string]struct {
		projected bool
		edit      func(*core.Definition)
	}{
		"Manager.purpose":       {true, func(d *core.Definition) { d.Purpose = "Orders and returns." }},
		"Manager.parent":        {true, func(d *core.Definition) { d.Spec["parent"] = ref(managerKind, "inventory", "inventory") }},
		"Manager.owns":          {true, func(d *core.Definition) { d.Spec["owns"] = []any{"src/orders/", "src/returns/"} }},
		"Manager.instructions":  {true, func(d *core.Definition) { d.Spec["instructions"] = "Keep returns consistent." }},
		"Statement.purpose":     {false, func(d *core.Definition) { d.Purpose = "Order cancellation and refunds." }},
		"Statement.category":    {true, func(d *core.Definition) { d.Spec["category"] = "workflow" }},
		"Statement.description": {true, func(d *core.Definition) { d.Spec["description"] = "Cancel before packing." }},
		"Statement.public":      {true, func(d *core.Definition) { d.Spec["public"] = true }},
		"Statement.uses":        {true, func(d *core.Definition) { d.Spec["uses"] = []any{contract} }},
		"Statement.requires":    {true, func(d *core.Definition) { delete(d.Spec, "requires") }},
		"Artifact.purpose":      {false, func(d *core.Definition) { d.Purpose = "Cancellation and refund code." }},
		"Artifact.role":         {true, func(d *core.Definition) { d.Spec["role"] = "documentation" }},
		"Artifact.realizes":     {true, func(d *core.Definition) { d.Spec["realizes"] = []any{contract} }},
		"Artifact.paths":        {true, func(d *core.Definition) { d.Spec["paths"] = []any{"src/orders/refund.go"} }},
		"Artifact.checks":       {true, func(d *core.Definition) { delete(d.Spec, "checks") }},
		"Artifact.required":     {true, func(d *core.Definition) { d.Spec["required"] = false }},
		"Artifact.reason":       {true, func(d *core.Definition) { d.Spec["reason"] = "Refunds need it." }},
		"Check.purpose":         {false, func(d *core.Definition) { d.Purpose = "Run refund tests." }},
		"Check.command":         {true, func(d *core.Definition) { d.Spec["command"] = []any{"go", "test", "./orders", "-race"} }},
		"Check.uses":            {true, func(d *core.Definition) { d.Spec["uses"] = []any{contract} }},
		"Check.limitation":      {true, func(d *core.Definition) { d.Spec["limitation"] = "Does not prove refunds." }},
		"Decision.purpose":      {false, func(d *core.Definition) { d.Purpose = "Why release is idempotent." }},
		"Decision.subject":      {false, func(d *core.Definition) { d.Spec["subject"] = ref(statementKind, "orders", "cancel-order") }},
		"Decision.decision":     {false, func(d *core.Definition) { d.Spec["decision"] = "Release at most once." }},
		"Decision.reason":       {false, func(d *core.Definition) { d.Spec["reason"] = "Stock must stay exact." }},
		"Decision.actor":        {false, func(d *core.Definition) { d.Spec["actor"] = ref(managerKind, "orders", "orders") }},
	}
	// A new schema property needs a row here, so it cannot be silently unprojected.
	for kind, k := range Schema().Kinds {
		for _, property := range append(mapKeysOf(k.Properties), "purpose") {
			if _, ok := edits[kind+"."+property]; !ok {
				t.Errorf("no edit covers %s.%s", kind, property)
			}
		}
	}
	for name, tc := range edits {
		t.Run(name, func(t *testing.T) {
			kind := name[:strings.Index(name, ".")]
			definitions := copyDefinitions(baseModel.Definitions)
			for i := range definitions {
				if definitions[i].Identity().Key() == targets[kind].Key() {
					tc.edit(&definitions[i])
				}
				if definitions[i].Kind == managerKind && definitions[i].Metadata.Namespace == "" {
					definitions[i].Purpose = "Root project manager for the shop."
				}
			}
			candidateModel, diagnostics := core.Compile(model.Schemas, definitions, "candidate")
			if len(diagnostics) != 0 {
				t.Fatalf("compile candidate: %+v", diagnostics)
			}
			impact := Impact(base, Analyze(candidateModel, files))
			if tc.projected && !contains(impact.ChangedDefinitions, targets[kind].Key()) {
				t.Fatalf("projected change was not named: changed=%v", impact.ChangedDefinitions)
			}
			if !tc.projected && (len(impact.Unknown) == 0 || !strings.HasPrefix(impact.Unknown[0], "model digest changed beyond")) {
				t.Fatalf("unprojected change was lost next to a projected one: unknown=%v", impact.Unknown)
			}
		})
	}
}

func mapKeysOf[V any](values map[string]V) []string {
	out := make([]string, 0, len(values))
	for key := range values {
		out = append(out, key)
	}
	return out
}

// Generated projects with one to three edits must give one impact, however
// often it runs and however its definitions, inventory and reports are ordered.
func TestImpactIsDeterministicOnGeneratedProjects(t *testing.T) {
	scoped := 0
	for seed := uint64(1); seed <= 200; seed++ {
		rng := rand.New(rand.NewPCG(seed, 7))
		base := generateProject(rng)
		candidate := mutateProject(rng, base)
		baseReport, candidateReport := base.analyze(t, nil), candidate.analyze(t, nil)
		want := Impact(baseReport, candidateReport)
		if len(want.Unknown) == 0 && len(want.Files) > 0 {
			scoped++
		}
		for i := 0; i < 2; i++ {
			for name, got := range map[string]ChangeImpact{
				"repeated":         Impact(baseReport, candidateReport),
				"shuffled reports": Impact(shuffledReport(rng, baseReport), shuffledReport(rng, candidateReport)),
				"shuffled input":   Impact(base.analyze(t, rng), candidate.analyze(t, rng)),
			} {
				if got.Digest != want.Digest {
					t.Fatalf("seed %d, %s: impact changed:\nwant statements=%v files=%v checks=%v managers=%v\ngot  statements=%v files=%v checks=%v managers=%v", seed, name, want.AffectedStatements, want.Files, want.Checks, want.Managers, got.AffectedStatements, got.Files, got.Checks, got.Managers)
				}
			}
		}
	}
	// Unknown scope routes everything and would make the comparison trivial.
	if scoped < 100 {
		t.Fatalf("only %d of 200 generated changes had a scoped impact", scoped)
	}
}

// Conservative invalidation: adding a change to a revision never removes
// anything from its impact, so impact(A and B) contains impact(A) and impact(B).
func TestImpactNeverShrinksWhenChangesCombine(t *testing.T) {
	description := func(i int) projectEdit {
		return func(q *generatedProject) { q.statements[i].description = "Changed rule." }
	}
	purpose := func(i int) projectEdit {
		return func(q *generatedProject) { q.statements[i].purpose = "Changed statement." }
	}
	check := func(t *testing.T, label string, base generatedProject, a, b []projectEdit) {
		t.Helper()
		report := base.analyze(t, nil)
		combined := Impact(report, base.with(append(append([]projectEdit(nil), a...), b...)...).analyze(t, nil))
		for name, part := range map[string][]projectEdit{"first": a, "second": b} {
			alone := Impact(report, base.with(part...).analyze(t, nil))
			for set, values := range map[string][2][]string{
				"statements": {alone.AffectedStatements, combined.AffectedStatements},
				"managers":   {alone.Managers, combined.Managers},
				"files":      {alone.Files, combined.Files},
				"checks":     {alone.Checks, combined.Checks},
			} {
				for _, v := range values[0] {
					if !contains(values[1], v) {
						t.Fatalf("%s: the %s change alone routes %s %s, combined it does not", label, name, set, v)
					}
				}
			}
			if len(alone.Unknown) > 0 && len(combined.Unknown) == 0 {
				t.Fatalf("%s: the %s change alone widens to unknown scope, combined it does not: %v", label, name, alone.Unknown)
			}
		}
	}
	t.Run("purpose and description of one statement", func(t *testing.T) {
		base := generatedProject{managers: []string{""}, statements: []generatedStatement{{description: "Rule.", purpose: "Statement."}},
			artifacts: []generatedArtifact{{realizes: []int{0}}}, digests: []string{"sha256:a"}}
		check(t, "purpose then description", base, []projectEdit{purpose(0)}, []projectEdit{description(0)})
	})
	t.Run("statement reached through uses first", func(t *testing.T) {
		// 0 and 1 change; 2 uses 0, 3 uses 2, 1 uses 3. Statement 3 consumes 2, so its
		// realization stays in the impact although 1 reaches it through uses first.
		statement := func(uses ...int) generatedStatement {
			return generatedStatement{description: "Rule.", purpose: "Statement.", uses: uses}
		}
		base := generatedProject{managers: []string{""}, statements: []generatedStatement{statement(), statement(3), statement(0), statement(2)},
			artifacts: []generatedArtifact{{realizes: []int{3}}}, digests: []string{"sha256:a"}}
		check(t, "uses chain", base, []projectEdit{description(0)}, []projectEdit{description(1)})
	})
	t.Run("generated projects", func(t *testing.T) {
		for seed := uint64(1); seed <= 200; seed++ {
			rng := rand.New(rand.NewPCG(seed, 11))
			base := generateProject(rng)
			check(t, fmt.Sprintf("seed %d", seed), base, randomEdits(rng, base), randomEdits(rng, base))
		}
	})
}

type generatedStatement struct {
	namespace, description, purpose string
	public                          bool
	uses, requires                  []int
}

type generatedArtifact struct {
	namespace string
	realizes  []int
	check     int
}

type generatedProject struct {
	managers   []string
	statements []generatedStatement
	artifacts  []generatedArtifact
	checks     []string
	command    []string
	digests    []string
}

func generateProject(rng *rand.Rand) generatedProject {
	p := generatedProject{managers: []string{""}}
	for i := 0; i < 1+rng.IntN(3); i++ {
		p.managers = append(p.managers, "m"+string(rune('a'+i)))
	}
	if rng.IntN(2) == 0 {
		p.managers = append(p.managers, p.managers[1]+".sub")
	}
	for i := 0; i < 4+rng.IntN(6); i++ {
		p.statements = append(p.statements, generatedStatement{namespace: p.managers[rng.IntN(len(p.managers))], description: "Rule.", purpose: "Statement.", public: rng.IntN(3) > 0})
	}
	for i := range p.statements {
		for j := range p.statements {
			if i != j && p.canReference(i, j) {
				switch rng.IntN(8) {
				case 0:
					p.statements[i].uses = append(p.statements[i].uses, j)
				case 1:
					p.statements[i].requires = append(p.statements[i].requires, j)
				}
			}
		}
	}
	for _, namespace := range p.managers {
		if rng.IntN(2) == 0 {
			p.checks = append(p.checks, namespace)
			p.command = append(p.command, "test")
		}
	}
	for i, s := range p.statements {
		if rng.IntN(3) > 0 {
			p.artifacts = append(p.artifacts, generatedArtifact{namespace: s.namespace, realizes: []int{i}, check: rng.IntN(len(p.checks) + 1)})
			p.digests = append(p.digests, "sha256:a")
		}
	}
	return p
}

// mutateProject applies one to three edits that keep the project valid.
func mutateProject(rng *rand.Rand, p generatedProject) generatedProject {
	return p.with(randomEdits(rng, p)...)
}

type projectEdit func(*generatedProject)

// randomEdits picks one to three edits. None undoes another, so any subset of
// them combines into a larger change.
func randomEdits(rng *rand.Rand, p generatedProject) []projectEdit {
	var edits []projectEdit
	for n := 1 + rng.IntN(3); n > 0; n-- {
		i, j, k := rng.IntN(len(p.statements)), rng.IntN(len(p.statements)), rng.Int()
		switch rng.IntN(6) {
		case 0:
			edits = append(edits, func(q *generatedProject) { q.statements[i].description = "Changed rule." })
		case 1:
			edits = append(edits, func(q *generatedProject) { q.statements[i].purpose = "Changed statement." })
		case 2:
			edits = append(edits, func(q *generatedProject) {
				if i != j && q.canReference(i, j) && !slices.Contains(q.statements[i].uses, j) {
					q.statements[i].uses = append(q.statements[i].uses, j)
				}
			})
		case 3:
			edits = append(edits, func(q *generatedProject) { q.statements[i].requires = nil })
		case 4:
			edits = append(edits, func(q *generatedProject) {
				if len(q.command) > 0 {
					q.command[k%len(q.command)] = "vet"
				}
			})
		case 5:
			edits = append(edits, func(q *generatedProject) {
				if len(q.digests) > 0 {
					q.digests[k%len(q.digests)] = "sha256:b"
				}
			})
		}
	}
	return edits
}

// with returns a copy of p after the edits.
func (p generatedProject) with(edits ...projectEdit) generatedProject {
	q := p
	q.statements = append([]generatedStatement(nil), p.statements...)
	for i := range q.statements {
		q.statements[i].uses = append([]int(nil), p.statements[i].uses...)
		q.statements[i].requires = append([]int(nil), p.statements[i].requires...)
	}
	q.command = append([]string(nil), p.command...)
	q.digests = append([]string(nil), p.digests...)
	for _, edit := range edits {
		edit(&q)
	}
	return q
}

func (p generatedProject) canReference(from, to int) bool {
	return p.statements[to].public || p.statements[to].namespace == p.statements[from].namespace
}

// analyze compiles the project. A non-nil rng supplies the definitions and the
// inventory in a random order. Reference lists keep their order: reordering one
// is a source edit that changes the model digest.
func (p generatedProject) analyze(t *testing.T, rng *rand.Rand) Report {
	t.Helper()
	api := APIVersion
	ref := func(kind, namespace, name string) map[string]any {
		return map[string]any{"apiVersion": api, "kind": kind, "namespace": namespace, "name": name}
	}
	statementRefs := func(ids []int) []any {
		out := []any{}
		for _, id := range ids {
			out = append(out, ref(statementKind, p.statements[id].namespace, "s"+string(rune('a'+id))))
		}
		return out
	}
	dir := func(namespace string) string {
		if namespace == "" {
			return "src/root/"
		}
		return "src/" + strings.ReplaceAll(namespace, ".", "/") + "/"
	}
	var definitions []core.Definition
	for _, namespace := range p.managers {
		spec := map[string]any{"owns": []any{"."}}
		if namespace != "" {
			parent := ""
			if dot := strings.LastIndex(namespace, "."); dot > 0 {
				parent = namespace[:dot]
			}
			spec = map[string]any{"parent": ref(managerKind, parent, managerName(parent)), "owns": []any{dir(namespace)}}
		}
		definitions = append(definitions, core.Definition{APIVersion: api, Kind: managerKind, Metadata: core.Metadata{Namespace: namespace, Name: managerName(namespace)}, Purpose: "Manager.", Spec: spec})
	}
	for i, s := range p.statements {
		definitions = append(definitions, core.Definition{APIVersion: api, Kind: statementKind, Metadata: core.Metadata{Namespace: s.namespace, Name: "s" + string(rune('a'+i))}, Purpose: s.purpose, Spec: map[string]any{"category": "rule", "description": s.description, "public": s.public, "uses": statementRefs(s.uses), "requires": statementRefs(s.requires)}})
	}
	for i, namespace := range p.checks {
		definitions = append(definitions, core.Definition{APIVersion: api, Kind: checkKind, Metadata: core.Metadata{Namespace: namespace, Name: "check"}, Purpose: "Check.", Spec: map[string]any{"command": []any{"go", p.command[i]}}})
	}
	files := []File{{Path: ".markitect/project.yaml", Digest: "sha256:manifest", Mode: "100644"}}
	for i, a := range p.artifacts {
		path := dir(a.namespace) + "a" + string(rune('a'+i)) + ".go"
		spec := map[string]any{"role": "implementation", "realizes": statementRefs(a.realizes), "paths": []any{path}, "required": true}
		if a.check < len(p.checks) {
			spec["checks"] = []any{ref(checkKind, p.checks[a.check], "check")}
		}
		definitions = append(definitions, core.Definition{APIVersion: api, Kind: artifactKind, Metadata: core.Metadata{Namespace: a.namespace, Name: "a" + string(rune('a'+i))}, Purpose: "Artifact.", Spec: spec})
		files = append(files, File{Path: path, Digest: p.digests[i], Mode: "100644"})
	}
	if rng != nil {
		definitions, files = shuffled(rng, definitions), shuffled(rng, files)
	}
	model, diagnostics := core.Compile([]core.Schema{Schema()}, definitions, "generated")
	if len(diagnostics) != 0 {
		t.Fatalf("compile generated project: %+v", diagnostics)
	}
	return Analyze(model, files)
}

func managerName(namespace string) string {
	if namespace == "" {
		return "root"
	}
	return "manager"
}

// shuffledReport permutes every top-level collection. Analyze emits them
// sorted, but Impact must not rely on that order.
func shuffledReport(rng *rand.Rand, r Report) Report {
	out := r
	out.Managers = shuffled(rng, r.Managers)
	out.Statements = shuffled(rng, r.Statements)
	out.Artifacts = shuffled(rng, r.Artifacts)
	out.Checks = shuffled(rng, r.Checks)
	out.Files = shuffled(rng, r.Files)
	out.Findings = shuffled(rng, r.Findings)
	out.Unknown = shuffled(rng, r.Unknown)
	return out
}

func shuffled[T any](rng *rand.Rand, values []T) []T {
	out := append([]T(nil), values...)
	rng.Shuffle(len(out), func(i, j int) { out[i], out[j] = out[j], out[i] })
	return out
}
