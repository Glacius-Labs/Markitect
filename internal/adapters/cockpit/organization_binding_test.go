package cockpit

import "testing"

// Synthetic read-contract evidence, never a provider workflow.
func TestOrganizationReadsRecordedProfilesAndHelpersOnlyAtTheirBoundary(t *testing.T) {
	s := newScenario(t)
	s.add("run.start", "", "", "", map[string]any{"run": "r", "managers": []string{".", "stock", "legacy"}, "binding": map[string]any{
		"head": "source-head", "strategy": "direct", "execution": "execution@digest", "defaults": "defaults@digest",
		"managers": []map[string]any{
			{"manager": ".", "execute": map[string]any{"profile": "writer", "identity": "bound-writer"}, "verify": map[string]any{"profile": "reviewer", "identity": "bound-reviewer"}, "maxHelpers": 0, "helpers": []any{}},
			{"manager": "stock", "execute": map[string]any{"profile": "other-writer"}, "verify": map[string]any{"profile": "other-reviewer"}, "helpers": []map[string]any{{"role": "worker", "writes": true, "profile": "worker-profile", "identity": "bound-helper"}}, "maxHelpers": 2},
			{"manager": "legacy"},
		},
		"checks": []map[string]any{{"name": "unit", "requiredBy": "stock/check", "image": "image", "command": "go test", "timeout": "1m"}},
	}})
	records := s.records()
	if v := Project("r", records, 0, Source{}, nil); v.Run.Binding != nil {
		t.Fatal("binding leaked before run.start")
	}
	v := Project("r", records, 1, Source{}, nil)
	b := v.Run.Binding
	if b == nil || len(b.Managers) != 3 || b.Head != "source-head" || b.Execution != "execution@digest" || b.Defaults != "defaults@digest" {
		t.Fatalf("binding = %+v", b)
	}
	root, stock := b.Managers[0], b.Managers[1]
	if root.Execute.Profile != "writer" || root.Verify.Identity != "bound-reviewer" || root.MaxHelpers == nil || *root.MaxHelpers != 0 || root.Helpers == nil {
		t.Fatalf("Root = %+v", root)
	}
	if stock.MaxHelpers == nil || *stock.MaxHelpers != 2 || len(stock.Helpers) != 1 || stock.Helpers[0].Profile != "worker-profile" || stock.Helpers[0].Identity != "bound-helper" || !stock.Helpers[0].Writes {
		t.Fatalf("stock = %+v", stock)
	}
	if b.Managers[2].MaxHelpers != nil || b.Managers[2].Execute.Profile != "" {
		t.Fatal("unrecorded role or helper limit was invented")
	}
	if len(b.Checks) != 1 || b.Checks[0].Command != "go test" || b.Checks[0].RequiredBy != "stock/check" {
		t.Fatalf("checks = %+v", b.Checks)
	}
}

func TestHistoricalOrganizationDoesNotInventBinding(t *testing.T) {
	s := newScenario(t)
	s.add("run.start", "", "", "", map[string]any{"run": "old", "managers": []string{"."}})
	v := Project("old", s.records(), 1, Source{}, nil)
	if v.Run.Binding != nil {
		t.Fatal("historical Run acquired a current binding")
	}
}
