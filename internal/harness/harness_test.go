package harness

import (
	"strings"
	"testing"
)

func mustParse(t *testing.T, text string) File {
	t.Helper()
	f, err := Parse([]byte(text))
	if err != nil {
		t.Fatalf("parse %q: %v", text, err)
	}
	return f
}

func text(t *testing.T, f File) string {
	t.Helper()
	data, err := f.Bytes()
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// The project's value beats the user's, the user's the built-in one, key by key;
// a skill without a strategy of its own takes strategy.default from wherever it
// comes.
func TestResolveKeyByKey(t *testing.T) {
	user := mustParse(t, "agents:\n  builder:\n    model: opus\n    effort: max\nstrategy:\n  default: parallel\n")
	project := mustParse(t, "agents:\n  builder:\n    model: haiku\nstrategy:\n  review: sequential\n")
	c := Resolve(user, project)
	for key, want := range map[string]Value{
		"agents.builder.model":  {Value: "haiku", Origin: Project},
		"agents.builder.effort": {Value: "max", Origin: User},
		"agents.reviewer.model": {Value: "sonnet", Origin: BuiltIn},
		"agents.auditor.effort": {Value: "high", Origin: BuiltIn},
		"strategy.default":      {Value: "parallel", Origin: User},
		"strategy.review":       {Value: "sequential", Origin: Project},
		"strategy.build":        {Value: "parallel", Origin: User, Via: "strategy.default"},
		"strategy.audit":        {Value: "parallel", Origin: User, Via: "strategy.default"},
	} {
		got, ok := c.Get(key)
		if !ok || got.Value != want.Value || got.Origin != want.Origin || got.Via != want.Via {
			t.Errorf("%s = %+v, want %+v", key, got, want)
		}
	}
	if model, effort := c.Agent("builder"); model != "haiku" || effort != "max" {
		t.Errorf("builder = %s %s", model, effort)
	}
	if got := Resolve(File{}, File{}).Strategy("build"); got != Auto {
		t.Errorf("the built-in strategy is %q", got)
	}
	if n := len(c.Values()); n != 4*2+1+len(Skills) {
		t.Errorf("%d values: %+v", n, c.Values())
	}
}

// A file names only keys the configuration takes, with values it takes; nothing
// else reaches an agent file.
func TestParseRefusesWhatItDoesNotTake(t *testing.T) {
	for _, bad := range []string{
		"agents:\n  builder:\n    model: \"opus; rm -rf /\"\n",
		"agents:\n  builder:\n    effort: huge\n",
		"agents:\n  tester:\n    model: opus\n",
		"agents:\n  builder:\n    tools: Bash\n",
		"strategy:\n  deploy: parallel\n",
		"strategy:\n  build: Not A Name\n",
		"workflows:\n  parallel: x.js\n",
		"workflows:\n  markitect-parallel: x.js\n",
		"workflows:\n  mine: \"\"\n",
		"colour: blue\n",
		"agents: [builder]\n",
		"- just a list\n",
		"agents:\n  builder:\n    model: opus\n    model: haiku\n",
	} {
		if _, err := Parse([]byte(bad)); err == nil {
			t.Errorf("took %q", bad)
		}
	}
	for _, good := range []string{"", "# nothing yet\n", "agents:\n  builder:\n    model: claude-opus-5-5\n    effort: xhigh\n  auditor:\n    model: inherit\n", "strategy:\n  build: my-flow\nworkflows:\n  my-flow: .claude/workflows/my-flow.js\n", "agents:\n  builder:\n    model: opus[1m]\n"} {
		mustParse(t, good)
	}
}

// Set and Unset keep the owner's comments and order, and Unset removes the
// mappings it leaves empty.
func TestSetAndUnsetKeepTheFile(t *testing.T) {
	f := mustParse(t, "# mine\nstrategy:\n  default: sequential # for now\n")
	if err := f.Set("agents.builder.model", "opus"); err != nil {
		t.Fatal(err)
	}
	if err := f.Set("strategy.default", "parallel"); err != nil {
		t.Fatal(err)
	}
	got := text(t, f)
	for _, want := range []string{"# mine\n", "default: parallel # for now\n", "agents:\n  builder:\n    model: opus\n"} {
		if !strings.Contains(got, want) {
			t.Errorf("lacks %q:\n%s", want, got)
		}
	}
	if strings.Index(got, "strategy:") > strings.Index(got, "agents:") {
		t.Errorf("the order changed:\n%s", got)
	}
	if err := f.Set("agents.builder.model", "bad name!"); err == nil {
		t.Error("set a bad model")
	}
	if !f.Unset("agents.builder.model") || f.Unset("agents.builder.model") || f.Unset("agents.reviewer.effort") {
		t.Error("unset did not say what it removed")
	}
	if got := text(t, f); strings.Contains(got, "agents") {
		t.Errorf("an empty mapping stayed:\n%s", got)
	}
	again := mustParse(t, text(t, f))
	if v := again.Values(); len(v) != 1 || v["strategy.default"] != "parallel" {
		t.Errorf("read back: %v", v)
	}
}

// A file that holds only comments keeps them when a value is set, and a new file
// starts with the comment it was made with.
func TestCommentsOfAnEmptyFileStay(t *testing.T) {
	f := mustParse(t, "# the owner's note\n# two lines\n")
	if got := text(t, f); got != "# the owner's note\n# two lines\n" {
		t.Errorf("an untouched file: %q", got)
	}
	if err := f.Set("strategy.build", "parallel"); err != nil {
		t.Fatal(err)
	}
	got := text(t, f)
	if !strings.HasPrefix(got, "# the owner's note\n# two lines\n") || !strings.Contains(got, "strategy:\n  build: parallel\n") {
		t.Errorf("after a set:\n%s", got)
	}
	n := New("# header\n")
	if err := n.Set("agents.reviewer.effort", "max"); err != nil {
		t.Fatal(err)
	}
	if got := text(t, n); !strings.HasPrefix(got, "# header\n") || !strings.Contains(got, "reviewer:\n    effort: max\n") {
		t.Errorf("a new file:\n%s", got)
	}
	n.Unset("agents.reviewer.effort")
	if got := text(t, n); got != "# header\n" {
		t.Errorf("emptied again: %q", got)
	}
}

// A strategy that names a custom workflow needs the workflow defined at some
// level; a skill that takes strategy.default is not counted twice.
func TestProblemsNameAMissingWorkflow(t *testing.T) {
	project := mustParse(t, "strategy:\n  default: quick\n  build: mine\n")
	user := mustParse(t, "workflows:\n  mine: ~/.claude/workflows/mine.js\n")
	problems := Resolve(user, project).Problems()
	if len(problems) != 1 || !strings.Contains(problems[0], "strategy.default names the workflow quick") {
		t.Fatalf("problems: %v", problems)
	}
	if p := Resolve(user, mustParse(t, "strategy:\n  build: mine\n")).Problems(); len(p) != 0 {
		t.Fatalf("problems: %v", p)
	}
}

// The agent files get the model and effort of their role, and nothing else of
// them changes.
func TestRenderAgent(t *testing.T) {
	file := "---\nname: markitect-builder\ndescription: model: not this one\nmodel: sonnet\neffort: high\n---\nmodel: in the body stays\n"
	c := Resolve(File{}, mustParse(t, "agents:\n  builder:\n    model: opus\n    effort: max\n"))
	got := string(c.RenderAgent("builder", []byte(file)))
	want := "---\nname: markitect-builder\ndescription: model: not this one\nmodel: opus\neffort: max\n---\nmodel: in the body stays\n"
	if got != want {
		t.Errorf("rendered:\n%s", got)
	}
	if got := string(c.RenderAgent("reviewer", []byte(file))); !strings.Contains(got, "\nmodel: sonnet\neffort: high\n") {
		t.Errorf("the reviewer:\n%s", got)
	}
	if got := string(c.RenderAgent("builder", []byte("no front matter\nmodel: x\n"))); got != "no front matter\nmodel: x\n" {
		t.Errorf("without front matter: %q", got)
	}
	for file, want := range map[string]string{".claude/agents/markitect-builder.md": "builder", ".claude/agents/markitect-auditor.md": "auditor", ".claude/agents/markitect-other.md": "", ".claude/skills/markitect-build/SKILL.md": ""} {
		if role, _ := RoleOf(file); role != want {
			t.Errorf("RoleOf(%s) = %q", file, role)
		}
	}
}
