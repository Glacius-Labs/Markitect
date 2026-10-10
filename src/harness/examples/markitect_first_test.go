package examples

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// The canonical intent checkpoint adds accounting limits to development
// context before this technical regression is introduced. It works in Verify's
// Git-free materialized tree as well as a development checkout.
func TestMarkitectFirstProjectContextAndProjections(t *testing.T) {
	root := harnessRepositoryRoot(t)
	snap, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	project, err := host.Parse(snap)
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Diagnostics) != 0 {
		t.Fatalf("root Project is invalid: %#v", project.Diagnostics)
	}
	const entry = "development/Skill/engineering-change"
	context, err := host.CompileContext(project, entry, "test", "sha256:fixed-test-tool")
	if err != nil {
		t.Fatal(err)
	}
	foundProtocol, foundLimits := false, false
	for _, input := range context.Inputs {
		if input.Key == "core/Workflow/markitect-first-change" {
			foundProtocol = true
			if len(input.Via) == 0 {
				t.Fatal("protocol lacks inclusion relation provenance")
			}
		}
		if input.Path == "docs/design/managed-artifact-coverage.md" {
			foundLimits = true
			if input.Hash != host.Hash(snap.Files[input.Path]) || input.Reason == "" {
				t.Fatalf("accounting limits lack exact input evidence: %#v", input)
			}
		}
		if strings.HasPrefix(input.Path, "examples/") {
			t.Fatalf("independent example entered development context: %#v", input)
		}
	}
	if !foundProtocol || !foundLimits {
		t.Fatalf("needed context missing: protocol=%v accounting limits=%v", foundProtocol, foundLimits)
	}
	repeated, err := host.CompileContext(project, entry, "test", "sha256:fixed-test-tool")
	if err != nil {
		t.Fatal(err)
	}
	first, err := host.YAML(context)
	if err != nil {
		t.Fatal(err)
	}
	second, err := host.YAML(repeated)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("fixed model/context inputs produced different output")
	}
	_, owners, err := host.GenerateOutputsWithOwners(project)
	if err != nil {
		t.Fatal(err)
	}
	for _, output := range []string{".agents/skills/engineering-change/SKILL.md", ".claude/skills/engineering-change/SKILL.md"} {
		if len(owners[output]) != 1 || owners[output][0] != entry {
			t.Fatalf("entry projection %s has wrong canonical owner: %v", output, owners[output])
		}
	}
	if findings := host.CheckOutputs(project); len(findings) != 0 {
		t.Fatalf("root projections are not converged: %#v", findings)
	}
	observation, err := host.ObserveProjection(project, "test", "sha256:fixed-test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if len(observation.Stale) != 0 || len(observation.Drift) != 0 || len(observation.Conflicts) != 0 {
		t.Fatalf("native projection observation does not converge: %#v", observation)
	}
	plan, err := host.PlanProjection(project, "test", "sha256:fixed-test-tool")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "complete" || len(plan.Operations) != 0 || len(plan.Stale) != 0 {
		t.Fatalf("converged root must have an empty complete plan: %#v", plan)
	}
	if err := host.VerifyProjectionPlan(project, plan, "test", "sha256:fixed-test-tool"); err != nil {
		t.Fatalf("converged root plan does not verify: %v", err)
	}
}
