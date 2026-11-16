package skills

import (
	"os/exec"
	"regexp"
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
		for _, key := range []string{"\nmodel: ", "\neffort: "} {
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
	const path = ".claude/skills/markitect-architect/SKILL.md"
	installed := false
	for _, f := range Harness() {
		installed = installed || f.Path == path
	}
	if !installed {
		t.Fatalf("%s is not among the skill files", path)
	}
	for _, f := range Harness() {
		if f.Path != ".claude/agents/markitect-architect.md" {
			continue
		}
		if !strings.Contains(string(f.Data), path) {
			t.Fatalf("%s does not name %s", f.Path, path)
		}
		return
	}
	t.Fatal("the harness has no architect agent")
}

// The skills that explain the spec to a person change nothing in the project and
// start no agent (concept, §8): Claude may start them, and they allow without a
// prompt only markitect's commands that read. Their last look at a project the
// catch-up skill keeps in the user's own folder, outside the project.
func TestTheReadingSkillsOnlyRead(t *testing.T) {
	allowed := regexp.MustCompile(`^Bash\(markitect ((status|doc|list|vet)( \*)?|inbox( --all)?)\)$`)
	for _, name := range []string{"tour", "catchup", "explain"} {
		path := ".claude/skills/markitect-" + name + "/SKILL.md"
		var text string
		for _, f := range Harness() {
			if f.Path == path {
				text = string(f.Data)
			}
		}
		if text == "" {
			t.Fatalf("the harness has no %s", path)
		}
		front, body, ok := strings.Cut(strings.TrimPrefix(text, "---\n"), "\n---\n")
		if !ok || !strings.HasPrefix(text, "---\n") {
			t.Fatalf("%s has no front matter", path)
		}
		front = "\n" + front + "\n"
		if !strings.Contains(front, "\nname: markitect-"+name+"\n") || !strings.Contains(front, "\ndescription: ") {
			t.Errorf("%s lacks its name or description", path)
		}
		if strings.Contains(front, "disable-model-invocation") {
			t.Errorf("%s: Claude may start a skill that only reads", path)
		}
		_, tools, _ := strings.Cut(front, "\nallowed-tools: ")
		tools, _, _ = strings.Cut(tools, "\n")
		for _, tool := range strings.Split(tools, ", ") {
			if !allowed.MatchString(tool) {
				t.Errorf("%s allows %q without a prompt, which is no reading command of markitect", path, tool)
			}
		}
		for _, starts := range []string{"Agent tool", "Workflow tool", "markitect-builder", "markitect-reviewer", "markitect-auditor", "markitect run", "markitect mark"} {
			if strings.Contains(body, starts) {
				t.Errorf("%s names %q, and a reading skill starts no agent and records nothing", path, starts)
			}
		}
	}
	for _, f := range Harness() {
		if f.Path != ".claude/skills/markitect-catchup/SKILL.md" {
			continue
		}
		text := strings.ReplaceAll(string(f.Data), "\r\n", "\n")
		// Each shell block names the user's own folder S in its first line, since
		// a shell keeps no variable from one command to the next, and removes
		// only under ${S:?}, so that an unset S stops it instead of reaching /.
		blocks := strings.Split(text, "```bash\n")[1:]
		if len(blocks) == 0 {
			t.Fatal("the catch-up skill has no shell block")
		}
		for _, b := range blocks {
			b = b[:strings.Index(b, "```")]
			if !strings.HasPrefix(b, `S="$HOME/.claude/markitect-seen/`) {
				t.Errorf("a shell block of the catch-up skill does not start by naming the user's own folder:\n%s", b)
			}
		}
		for _, rm := range strings.Split(text, "rm -rf ")[1:] {
			if !strings.HasPrefix(rm, `"${S:?}/`) {
				t.Errorf("the catch-up skill removes %.30q, outside ${S:?}", rm)
			}
		}
	}
}
