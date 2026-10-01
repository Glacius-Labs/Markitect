package app

import (
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestStrictProviderDiagnosticsHaveStableResourceOrder(t *testing.T) {
	project := &core.Resource{Kind: "Project", Spec: core.Spec{
		Targets:          []string{"claude"},
		ProviderAdapters: &core.ProviderAdapters{StrictInventory: true},
	}}
	resources := map[string]*core.Resource{}
	for _, name := range []string{"zeta", "alpha", "middle"} {
		rule := &core.Resource{Kind: "Rule", Metadata: core.Metadata{Name: name}, Path: "docs/rules/" + name + ".yaml", Line: 1}
		resources[rule.Key()] = rule
	}
	p := &Project{Snapshot: &snapshot.Snapshot{Files: map[string][]byte{}}, Graph: &core.Graph{Project: project, Resources: resources}}
	want := []string{"docs/rules/alpha.yaml", "docs/rules/middle.yaml", "docs/rules/zeta.yaml"}
	for run := 0; run < 32; run++ {
		findings := checkProviderAdapterInputs(p)
		if len(findings) != len(want) {
			t.Fatalf("run %d: findings = %+v", run, findings)
		}
		for i, path := range want {
			if findings[i].Code != "provider-adapter.unmapped-rule" || findings[i].Path != path {
				t.Fatalf("run %d: finding %d = %+v, want %s", run, i, findings[i], path)
			}
		}
	}
}
