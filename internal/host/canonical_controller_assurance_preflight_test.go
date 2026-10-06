package host

import (
	"context"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
)

func TestCanonicalControllerAssuranceChecksAreSourceBoundBeforeExecution(t *testing.T) {
	root, revision, baseConfig := canonicalControllerFixture(t)
	fixed, err := LoadSelectedCanonicalSource(root, revision, "examples/canonical-projection/canonical.yaml", true)
	if err != nil {
		t.Fatalf("load selected fixed source: %v", err)
	}
	requests, err := projectionRequestIndex(fixed)
	if err != nil {
		t.Fatalf("index selected projections: %v", err)
	}
	projectionIDs := make([]string, 0, len(requests))
	for id := range requests {
		projectionIDs = append(projectionIDs, id)
	}
	sort.Strings(projectionIDs)
	if len(projectionIDs) == 0 || len(fixed.Config.Checks) == 0 {
		t.Fatal("fixture must provide a selected Projection and source-declared fixed check")
	}
	configuredChecks := append([]authoring.Check(nil), fixed.Config.Checks...)
	baseConfig.AssuranceRoots = []string{"selected"}
	baseConfig.AssuranceScopes = []CanonicalAssuranceScope{{
		ID: "selected", ProjectionID: projectionIDs[0], Checks: append([]authoring.Check(nil), configuredChecks...),
	}}

	selected, err := canonicalControllerScopedChecks(fixed, requests[projectionIDs[0]].Projector, baseConfig.AssuranceScopes[0].Checks)
	if err != nil || !equalCanonicalValue(selected, configuredChecks) {
		t.Fatalf("unchanged source-declared checks were not accepted exactly: checks=%#v err=%v", selected, err)
	}

	// Keep cycle validation ahead of source-binding diagnostics.
	cyclic := baseConfig
	cyclic.AssuranceScopes = append([]CanonicalAssuranceScope(nil), baseConfig.AssuranceScopes...)
	cyclic.AssuranceScopes[0].Children = []string{"selected"}
	cyclic.AssuranceScopes[0].Checks = append([]authoring.Check(nil), configuredChecks...)
	cyclic.AssuranceScopes[0].Checks[0].Name = "undeclared-c11-check"
	if _, err := canonicalControllerAssuranceGraph(cyclic, fixed); err == nil || !strings.Contains(err.Error(), "cycle detected") {
		t.Fatalf("structural cycle diagnostic was hidden by source check validation: %v", err)
	}

	marker := filepath.Join(filepath.Dir(baseConfig.RecordStore), "actor-invoked")
	t.Setenv(canonicalControllerActorEnv, "1")
	t.Setenv(canonicalControllerMarkerEnv, marker)
	beforeHead, beforeIndex, beforeFiles := canonicalControllerSourceState(t, root)

	cases := []struct {
		name string
		edit func(*authoring.Check)
		want string
	}{
		{
			name: "undeclared scope check",
			edit: func(check *authoring.Check) { check.Name = "c11-workflow-markdown-undeclared" },
			want: "not declared by the selected source config",
		},
		{
			name: "changed command argv",
			edit: func(check *authoring.Check) { check.Run = append(check.Run, "--unapproved") },
			want: "differs from its exact selected source-config definition",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := baseConfig
			cfg.AssuranceScopes = append([]CanonicalAssuranceScope(nil), baseConfig.AssuranceScopes...)
			cfg.AssuranceScopes[0].Checks = append([]authoring.Check(nil), configuredChecks...)
			tc.edit(&cfg.AssuranceScopes[0].Checks[0])

			if _, err := ProposeCanonicalController(root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Propose accepted source-unbound assurance check: %v", err)
			}
			if _, err := ExecuteCanonicalController(context.Background(), root, revision, revision, "examples/canonical-projection/canonical.yaml", cfg, "preflight-test/1", "sha256:"+strings.Repeat("a", 64)); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Execute accepted source-unbound assurance check: %v", err)
			}
			if _, err := os.Stat(marker); !os.IsNotExist(err) {
				t.Fatalf("refused assurance configuration invoked the actor: %v", err)
			}
			if _, err := os.Stat(cfg.RecordStore); !os.IsNotExist(err) {
				t.Fatalf("refused assurance configuration created a ledger: %v", err)
			}
			if head, index, files := canonicalControllerSourceState(t, root); head != beforeHead || index != beforeIndex || !equalCanonicalValue(files, beforeFiles) {
				t.Fatal("refused assurance configuration changed source or target files")
			}
		})
	}

	if _, err := ProposeCanonicalController(root, revision, revision, "examples/canonical-projection/canonical.yaml", baseConfig); err != nil {
		t.Fatalf("Propose rejected unchanged source-declared checks: %v", err)
	}
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("read-only valid Propose invoked the actor: %v", err)
	}
	if _, err := os.Stat(baseConfig.RecordStore); !os.IsNotExist(err) {
		t.Fatalf("read-only valid Propose created a ledger: %v", err)
	}
	if head, index, files := canonicalControllerSourceState(t, root); head != beforeHead || index != beforeIndex || !equalCanonicalValue(files, beforeFiles) {
		t.Fatal("valid read-only Propose changed source or target files")
	}
}
