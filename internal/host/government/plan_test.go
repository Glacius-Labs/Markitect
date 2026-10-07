package government

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
)

const testAPI = "orders.example/v1"

func testID(kind, name string) core.DefinitionIdentity {
	api := APIVersion
	if kind == "Rule" {
		api = testAPI
	}
	return core.DefinitionIdentity{APIVersion: api, Kind: kind, Namespace: "shop", Name: name}
}
func testRef(kind, name string) map[string]any {
	id := testID(kind, name)
	return map[string]any{"apiVersion": id.APIVersion, "kind": id.Kind, "namespace": id.Namespace, "name": id.Name}
}
func testDef(kind, name string, spec map[string]any) core.Definition {
	id := testID(kind, name)
	return core.Definition{APIVersion: id.APIVersion, Kind: kind, Metadata: core.Metadata{Namespace: id.Namespace, Name: name}, Purpose: "Explain " + name, Spec: spec}
}
func testSource() Source {
	defs := []core.Definition{
		testDef("Constitution", "active", map[string]any{"root": testRef("Area", "root"), "cabinet": []any{testRef("Ressort", "quality")}, "approval": "unanimous-explicit-assent"}),
		testDef("Area", "root", map[string]any{}),
		testDef("Area", "orders", map[string]any{"parent": testRef("Area", "root")}),
		testDef("Area", "cancellation", map[string]any{"parent": testRef("Area", "orders")}),
		testDef("Ressort", "quality", map[string]any{"mandate": testRef("Mandate", "root")}),
		testDef("Mandate", "root", map[string]any{"area": testRef("Area", "root"), "scope": []any{testRef("Rule", "cancel"), testRef("Rule", "release")}, "actions": []any{"implement", "review", "amend-model"}}),
		testDef("Mandate", "orders", map[string]any{"area": testRef("Area", "orders"), "parent": testRef("Mandate", "root"), "scope": []any{testRef("Rule", "cancel"), testRef("Rule", "release")}, "actions": []any{"implement", "review"}}),
		testDef("Rule", "cancel", map[string]any{"dependsOn": testRef("Rule", "release")}),
		testDef("Rule", "release", map[string]any{}),
		testDef("Responsibility", "cancel", map[string]any{"subject": testRef("Rule", "cancel"), "area": testRef("Area", "orders")}),
		testDef("Responsibility", "release", map[string]any{"subject": testRef("Rule", "release"), "area": testRef("Area", "orders")}),
		testDef("Artifact", "handler", map[string]any{"path": "handler.go", "class": "realization", "writer": testRef("Area", "orders")}),
		testDef("Artifact", "test", map[string]any{"path": "handler_test.go", "class": "realization", "writer": testRef("Area", "orders")}),
		testDef("Realization", "cancel-handler", map[string]any{"subject": testRef("Rule", "cancel"), "artifact": testRef("Artifact", "handler"), "role": "transition"}),
		testDef("Realization", "release-handler", map[string]any{"subject": testRef("Rule", "release"), "artifact": testRef("Artifact", "handler"), "role": "stock release"}),
		testDef("Realization", "cancel-test", map[string]any{"subject": testRef("Rule", "cancel"), "artifact": testRef("Artifact", "test"), "role": "behavior check"}),
	}
	return Source{APIVersion: SourceVersion, Kind: "GovernmentSource", Constitution: testID("Constitution", "active"), Definitions: defs, Observation: inventory.Options{Roots: []string{"."}}, Schemas: []core.Schema{{APIVersion: testAPI, Purpose: "Order obligations", Kinds: map[string]core.Kind{"Rule": {Purpose: "Checkable behavior", Properties: map[string]core.Property{"dependsOn": {Purpose: "Needs neighboring behavior", Type: core.TypeReference, MinCount: 0, MaxCount: 1, Target: &core.KindIdentity{APIVersion: testAPI, Kind: "Rule"}}}}}}}}
}

func mutate(s *Source, kind, name string, fn func(*core.Definition)) {
	for i := range s.Definitions {
		if s.Definitions[i].Kind == kind && s.Definitions[i].Metadata.Name == name {
			fn(&s.Definitions[i])
			return
		}
	}
	panic("test identity missing")
}
func hasFinding(findings []Finding, code string) bool {
	for _, f := range findings {
		if f.Code == code {
			return true
		}
	}
	return false
}
func testOrder(m Model) Order {
	return Order{APIVersion: OrderVersion, Kind: "Order", Purpose: "Repair cancellation", ActiveConstitution: m.Digest, Action: "implement", Subjects: []core.DefinitionIdentity{testID("Rule", "cancel")}}
}
func realReport(t *testing.T, extra bool) inventory.Report {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{"handler.go": "package orders\nfunc Cancel() {}\n", "handler_test.go": "package orders\n"}
	if extra {
		files["unmapped.txt"] = "unknown purpose"
	}
	for p, b := range files {
		if err := os.WriteFile(filepath.Join(dir, p), []byte(b), 0600); err != nil {
			t.Fatal(err)
		}
	}
	r, err := inventory.Capture(dir, inventory.Options{Roots: []string{"."}})
	if err != nil {
		t.Fatal(err)
	}
	return r
}

func TestAmendmentPlanRequiresCanonicalWriterPriorAmendmentMandate(t *testing.T) {
	s := testSource()
	// The root owns the business decision; another Area writes its canonical
	// representation. The owner's permission cannot substitute for the writer's.
	for _, name := range []string{"cancel", "release"} {
		mutate(&s, "Responsibility", name, func(d *core.Definition) { d.Spec["area"] = testRef("Area", "root") })
	}
	s.Definitions = append(s.Definitions,
		testDef("Artifact", "model", map[string]any{"path": "government.yaml", "class": "canonical", "writer": testRef("Area", "orders")}),
		testDef("Realization", "cancel-model", map[string]any{"subject": testRef("Rule", "cancel"), "artifact": testRef("Artifact", "model"), "role": "canonical cancellation rule"}),
	)
	m := Compile(s)
	if len(m.Findings) != 0 {
		t.Fatal(m.Findings)
	}
	o := testOrder(m)
	o.Action = "amend-model"
	p := BuildPlan(m, o, realReport(t, false), "government.yaml")
	if p.Status != "blocked" || !hasFinding(p.Findings, "writer.unauthorized") {
		t.Fatalf("canonical writer inherited owner's authority: %+v", p)
	}
	mutate(&s, "Mandate", "orders", func(d *core.Definition) { d.Spec["actions"] = []any{"implement", "review", "amend-model"} })
	m = Compile(s)
	o.ActiveConstitution = m.Digest
	p = BuildPlan(m, o, realReport(t, false), "government.yaml")
	if p.Status == "blocked" || p.Action != "amend-model" {
		t.Fatalf("explicit prior canonical writer mandate not used: %+v", p)
	}
	dp := BuildDelegationPlan(m, p, DelegationLimits{MaxDepth: 4, MaxFanout: 4, MaxCalls: 32})
	if dp.Status != "planned" || len(dp.Root.Children) != 1 || !contains(dp.Root.Children[0].Actions, "amend-model") {
		t.Fatalf("canonical writer amendment action missing: %+v", dp)
	}
	if len(dp.Root.Actions) != 0 {
		t.Fatalf("structural parent inherited child write action: %v", dp.Root.Actions)
	}
}

func TestAmendmentPlanSeparatesModelAndRealizationAuthorityPerSubject(t *testing.T) {
	s := testSource()
	for _, name := range []string{"cancel", "release"} {
		mutate(&s, "Responsibility", name, func(d *core.Definition) { d.Spec["area"] = testRef("Area", "root") })
	}
	s.Definitions = append(s.Definitions,
		testDef("Area", "model", map[string]any{"parent": testRef("Area", "root")}),
		testDef("Mandate", "model-amend", map[string]any{"area": testRef("Area", "model"), "parent": testRef("Mandate", "root"), "scope": []any{testRef("Rule", "cancel")}, "actions": []any{"amend-model"}}),
		testDef("Mandate", "model-code", map[string]any{"area": testRef("Area", "model"), "parent": testRef("Mandate", "root"), "scope": []any{testRef("Rule", "release")}, "actions": []any{"implement"}}),
		testDef("Artifact", "model", map[string]any{"path": "government.yaml", "class": "canonical", "writer": testRef("Area", "model")}),
		testDef("Artifact", "release-helper", map[string]any{"path": "release.go", "class": "realization", "writer": testRef("Area", "model")}),
		testDef("Realization", "cancel-model", map[string]any{"subject": testRef("Rule", "cancel"), "artifact": testRef("Artifact", "model"), "role": "canonical cancellation rule"}),
		testDef("Realization", "release-helper", map[string]any{"subject": testRef("Rule", "release"), "artifact": testRef("Artifact", "release-helper"), "role": "stock release helper"}),
	)
	m := Compile(s)
	if len(m.Findings) != 0 {
		t.Fatal(m.Findings)
	}
	o := testOrder(m)
	o.Action = "amend-model"
	p := BuildPlan(m, o, realReport(t, false), "government.yaml")
	if p.Status == "blocked" {
		t.Fatalf("model-only mandate required implement or unrelated amendment rights: %+v", p.Findings)
	}
	dp := BuildDelegationPlan(m, p, DelegationLimits{MaxDepth: 4, MaxFanout: 4, MaxCalls: 32})
	if dp.Status != "planned" {
		t.Fatalf("path/subject actions were merged: %+v", dp.Findings)
	}
	for _, child := range dp.Root.Children {
		if child.Area == testID("Area", "model") && !reflect.DeepEqual(child.Actions, []string{"amend-model", "implement"}) {
			t.Fatalf("missing exact local action classes: %v", child.Actions)
		}
	}
}

func TestGovernmentPlanJoinsRealFilesAndKeepsUnknowns(t *testing.T) {
	s := testSource()
	m := Compile(s)
	if len(m.Findings) > 0 {
		t.Fatalf("model: %+v", m.Findings)
	}
	r := realReport(t, true)
	p := BuildPlan(m, testOrder(m), r)
	if p.Status != "planned-scoped" || !p.Provisional || len(p.Findings) > 0 {
		t.Fatalf("plan: %+v", p)
	}
	if len(p.Affected) != 2 || len(p.Work) != 1 || len(p.Work[0].Paths) != 2 || len(p.IntegrationReviews) != 2 {
		t.Fatalf("lost responsibility/dependencies: %+v", p)
	}
	if !reflect.DeepEqual(p.Survey.Unknown, []string{"unmapped.txt"}) || p.Survey.Coverage != "incomplete" {
		t.Fatalf("unknown became pass: %+v", p.Survey)
	}
	for _, a := range p.Survey.Artifacts {
		if a.Path == "handler.go" && (len(a.Subjects) != 2 || a.Writer == nil || a.Observed.Digest == "") {
			t.Fatalf("bad realization: %+v", a)
		}
	}
	if !reflect.DeepEqual(p, BuildPlan(m, testOrder(m), r)) {
		t.Fatal("same fixed values changed plan")
	}
	clean := BuildPlan(m, testOrder(m), realReport(t, false))
	if clean.Status != "planned-within-declared-boundary" {
		t.Fatalf("clean plan: %+v", clean)
	}
}

func TestGovernmentModelRejectsConflictsAndSelfDelegation(t *testing.T) {
	cases := []struct {
		name, code string
		change     func(*Source)
	}{
		{"purpose", "definition.purpose", func(s *Source) { s.Definitions[0].Purpose = " " }},
		{"unknown identity", "subject.unresolved", func(s *Source) {
			mutate(s, "Responsibility", "cancel", func(d *core.Definition) { d.Spec["subject"] = testRef("Rule", "absent") })
		}},
		{"two writers", "artifact.conflict", func(s *Source) {
			s.Definitions = append(s.Definitions, testDef("Artifact", "duplicate", map[string]any{"path": "handler.go", "class": "realization", "writer": testRef("Area", "root")}))
		}},
		{"two owners", "responsibility.conflict", func(s *Source) {
			s.Definitions = append(s.Definitions, testDef("Responsibility", "duplicate", map[string]any{"subject": testRef("Rule", "cancel"), "area": testRef("Area", "root")}))
		}},
		{"missing writer", "artifact.writer-missing", func(s *Source) {
			mutate(s, "Artifact", "handler", func(d *core.Definition) { delete(d.Spec, "writer") })
		}},
		{"cycle", "area.cycle", func(s *Source) {
			mutate(s, "Area", "orders", func(d *core.Definition) { d.Spec["parent"] = testRef("Area", "cancellation") })
		}},
		{"authority expansion", "mandate.expansion", func(s *Source) {
			mutate(s, "Mandate", "root", func(d *core.Definition) { d.Spec["actions"] = []any{"review"} })
		}},
		{"undelegated", "mandate.undelegated", func(s *Source) { mutate(s, "Mandate", "orders", func(d *core.Definition) { delete(d.Spec, "parent") }) }},
		{"unsafe path", "artifact.path", func(s *Source) {
			mutate(s, "Artifact", "handler", func(d *core.Definition) { d.Spec["path"] = "../outside" })
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			s := testSource()
			tc.change(&s)
			m := Compile(s)
			if !hasFinding(m.Findings, tc.code) {
				t.Fatalf("want %s got %+v", tc.code, m.Findings)
			}
		})
	}
}

func TestGovernmentPlanRefusesStaleAuthorityAndUnavailableScope(t *testing.T) {
	m := Compile(testSource())
	r := realReport(t, false)
	cases := []struct {
		name, code string
		change     func(*Order, *inventory.Report)
	}{
		{"stale", "order.stale", func(o *Order, _ *inventory.Report) { o.ActiveConstitution = "sha256:" + strings.Repeat("0", 64) }},
		{"empty", "order.empty", func(o *Order, _ *inventory.Report) { o.Subjects = nil }},
		{"unknown identity", "order.subject", func(o *Order, _ *inventory.Report) { o.Subjects = []core.DefinitionIdentity{testID("Rule", "absent")} }},
		{"unknown scope", "scope.unknown", func(o *Order, _ *inventory.Report) { o.UnknownScope = true }},
		{"self authority", "authority.protected", func(o *Order, _ *inventory.Report) {
			o.Action = "amend-model"
			o.Subjects = []core.DefinitionIdentity{m.Constitution}
		}},
		{"nondelegated amendment", "authority.missing", func(o *Order, _ *inventory.Report) { o.Action = "amend-model" }},
		{"unmapped path", "order.unmapped-path", func(o *Order, _ *inventory.Report) { o.Paths = []string{"unknown.go"} }},
		{"excluded input", "input.unavailable", func(_ *Order, r *inventory.Report) {
			for i := range r.Entries {
				if r.Entries[i].Path == "handler.go" {
					r.Entries[i].Status = "excluded"
					r.Entries[i].Digest = ""
				}
			}
		}},
		{"outside scope", "input.outside-boundary", func(_ *Order, r *inventory.Report) { r.Roots = []string{"different"}; r.Entries = nil }},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			o := testOrder(m)
			copyR := r
			copyR.Entries = append([]inventory.Entry(nil), r.Entries...)
			tc.change(&o, &copyR)
			p := BuildPlan(m, o, copyR)
			if p.Status != "blocked" || !hasFinding(p.Findings, tc.code) {
				t.Fatalf("want %s got %+v", tc.code, p)
			}
		})
	}
}

func TestGovernmentIntentIdentityExcludesObservedBytes(t *testing.T) {
	s := testSource()
	m := Compile(s)
	s.Definitions[0].Source = core.Source{Path: "different", Digest: "observed"}
	same := Compile(s)
	if same.Digest != m.Digest {
		t.Fatal("provenance altered accepted intent")
	}
	s.Definitions[0].Purpose = "Changed root goal"
	if Compile(s).Digest == m.Digest {
		t.Fatal("root intent not bound")
	}
}

func TestGovernmentMissingBelowBoundaryIsUnavailable(t *testing.T) {
	s := testSource()
	mutate(&s, "Artifact", "handler", func(d *core.Definition) { d.Spec["path"] = "private/handler.go" })
	m := Compile(s)
	r := realReport(t, false)
	r.Entries = append(r.Entries, inventory.Entry{Path: "private", Type: "directory", Source: "native", Status: "excluded", Reason: "private owner boundary"})
	p := BuildPlan(m, testOrder(m), r)
	if p.Status != "blocked" || !hasFinding(p.Findings, "input.unavailable") || !reflect.DeepEqual(p.Survey.Unavailable, []string{"private/handler.go"}) {
		t.Fatalf("excluded descendant became missing create: %+v", p)
	}
}

func TestGovernmentGitMetadataIsReasonedExclusion(t *testing.T) {
	m := Compile(testSource())
	r := realReport(t, false)
	r.Entries = append(r.Entries, inventory.Entry{Path: ".git", Type: "directory", Source: "native", Status: "boundary", Reason: ".git is Git metadata"})
	s := SurveyRepository(m, r)
	if len(s.Unknown) > 0 || s.Coverage == "incomplete" {
		t.Fatalf("Git metadata became unexplained artifact: %+v", s)
	}
}

func TestGovernmentUnknownScopeBroadensAndForeignOnlyCannotExecute(t *testing.T) {
	s := testSource()
	m := Compile(s)
	o := testOrder(m)
	o.Subjects = nil
	o.UnknownScope = true
	p := BuildPlan(m, o, realReport(t, false))
	if p.Status != "blocked" || len(p.Affected) != 2 || len(p.Work[0].Paths) != 2 || len(p.IntegrationReviews) != 2 {
		t.Fatalf("unknown scope lost domain work/root review: %+v", p)
	}
	for _, name := range []string{"handler", "test"} {
		mutate(&s, "Artifact", name, func(d *core.Definition) { d.Spec["class"] = "foreign"; delete(d.Spec, "writer") })
	}
	m = Compile(s)
	p = BuildPlan(m, testOrder(m), realReport(t, false))
	if p.Status != "blocked" || !hasFinding(p.Findings, "realization.missing") {
		t.Fatalf("foreign-only input became executable work: %+v", p)
	}
}

func TestGovernmentSurveyShowsUnassignedAndUnrealizedModelSubjects(t *testing.T) {
	s := testSource()
	s.Definitions = append(s.Definitions, testDef("Rule", "not-yet-understood", map[string]any{}))
	m := Compile(s)
	survey := SurveyRepository(m, realReport(t, false))
	want := []core.DefinitionIdentity{testID("Rule", "not-yet-understood")}
	if survey.Coverage != "incomplete" || !reflect.DeepEqual(survey.UnassignedSubjects, want) || !reflect.DeepEqual(survey.UnrealizedSubjects, want) {
		t.Fatalf("unknown model responsibilities disappeared: %+v", survey)
	}
	p := BuildPlan(m, testOrder(m), survey.Observation)
	if p.Status != "planned-scoped" {
		t.Fatalf("unrelated unknown should retain scoped work: %+v", p)
	}
}

func TestGovernmentDecodeClosedDocument(t *testing.T) {
	for _, data := range []string{"kind: GovernmentSource\nunknown: value\n", "kind: GovernmentSource\n---\nkind: GovernmentSource\n", "kind: &name GovernmentSource\n"} {
		var s Source
		if Decode([]byte(data), &s) == nil {
			t.Fatalf("accepted %q", data)
		}
	}
}
