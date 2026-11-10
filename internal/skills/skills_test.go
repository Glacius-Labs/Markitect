package skills

import (
	"os/exec"
	"strings"
	"testing"
)

// The parallel workflow is a script Claude Code runs, which no Go code executes:
// testdata/simulate.mjs runs it against stub agents and checks its waves, merges
// and stop rule (DEC-103). It needs node, which the Linux CI runners have.
func TestParallelWorkflowAgainstStubAgents(t *testing.T) {
	node, err := exec.LookPath("node")
	if err != nil {
		t.Skip("node is not on the PATH; the workflow's simulation needs it")
	}
	out, err := exec.Command(node, "testdata/simulate.mjs", "claude/workflows/markitect-parallel.js").CombinedOutput()
	if err != nil {
		t.Fatalf("the simulation failed: %v\n%s", err, out)
	}
}

// Every agent file names a model and an effort in its front matter, since init
// and update write the configuration's values over those lines and add none.
func TestEveryAgentNamesItsModelAndEffort(t *testing.T) {
	agents := 0
	for _, f := range Harness() {
		if !strings.HasPrefix(f.Path, ".claude/agents/") {
			continue
		}
		agents++
		front, _, ok := strings.Cut(strings.TrimPrefix(string(f.Data), "---\n"), "\n---\n")
		if !ok || !strings.HasPrefix(string(f.Data), "---\n") {
			t.Fatalf("%s has no front matter", f.Path)
		}
		for _, key := range []string{"\nmodel: sonnet\n", "\neffort: high\n"} {
			if !strings.Contains("\n"+front+"\n", key) {
				t.Errorf("%s lacks %q in its front matter", f.Path, strings.TrimSpace(key))
			}
		}
	}
	if agents != 4 {
		t.Fatalf("%d agents", agents)
	}
	workflows := 0
	for _, f := range Harness() {
		if strings.HasPrefix(f.Path, ".claude/workflows/") {
			workflows++
			if f.Path != ".claude/workflows/markitect-parallel.js" {
				t.Errorf("an unexpected workflow %s", f.Path)
			}
		}
	}
	if workflows != 1 {
		t.Fatalf("%d workflows", workflows)
	}
}

// The architect agent reads the architect skill by its path in a project, so
// that path must be a file init installs: a rename of the skill must reach the
// agent's file too.
func TestTheArchitectAgentNamesAnInstalledSkillFile(t *testing.T) {
	const path = "markitect-architect/SKILL.md"
	installed := false
	for _, f := range Files() {
		installed = installed || f.Path == path
	}
	if !installed {
		t.Fatalf("%s is not among the skill files", path)
	}
	for _, f := range Harness() {
		if f.Path != ".claude/agents/markitect-architect.md" {
			continue
		}
		if !strings.Contains(string(f.Data), ".claude/skills/"+path) {
			t.Fatalf("%s does not name .claude/skills/%s", f.Path, path)
		}
		return
	}
	t.Fatal("the harness has no architect agent")
}
