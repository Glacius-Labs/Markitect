package sandbox_test

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/adapters/sandbox"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// The whole Command on the real engine: the host's real Git, the real container
// and its users, and sessions scripted as shell lines in that container instead
// of a provider. These are the planted refusals of U1: each holds whatever an
// agent did, and they start no provider.

const (
	mdir  = project.ModelDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
	token = "sk-ant-oat01-planted-token-value"
)

func spec(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

func project1() map[string]string {
	files, _ := libraries.Files("std")
	std := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, model.File{Path: f.Path, Data: f.Data})
	}
	return map[string]string{
		project.ManifestFile:                "libraries:\n  std: 1.0.0\n",
		project.LockFile:                    string(model.RenderLock([]model.LibrarySource{std})),
		project.ChecksFile:                  "checks:\n  - name: no-fail\n    command: 'test ! -e FAIL'\n    timeout: 1m\n    environment: test-image\n",
		mdir + "project.md":                 spec("std/Project", "Library", "Parts: [[catalog]]."),
		mdir + "catalog/book.md":            spec("std/Concept", "Book", "A book has a title."),
		project.RootPage:                    table + "| README.md | [[project]] | Overview |\n",
		project.RegistryDir + "/catalog.md": "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Type |\n",
		"README.md":                         "# Library\n",
		"src/catalog/book.go":               "package catalog\n",
	}
}

// scripted plays each role's sessions as shell lines, run in the session's folder
// as the session's user with the token in its environment, as a provider's
// session would be.
type scripted struct {
	t     *testing.T
	lines map[agentport.Role][]string
	// answers replies with ok for every item, pointing at README.md.
	prompts map[agentport.Role][]string
}

var listLine = regexp.MustCompile(`(?m)^\[ \] (.+?)(?:: ".*")?$`)
var answerFile = regexp.MustCompile(`(?m)^Write one line per item to (.+), with pointers`)

func (a *scripted) Run(ctx context.Context, sh agentport.Shell, s agentport.Session) agentport.End {
	a.prompts[s.Role] = append(a.prompts[s.Role], s.Prompt)
	line := ""
	if q := a.lines[s.Role]; len(q) > 0 {
		line, a.lines[s.Role] = q[0], q[1:]
	}
	var out lockedBuffer
	secrets := []string{"CLAUDE_CODE_OAUTH_TOKEN=" + token}
	if s.Role == agentport.Reviewer {
		m := answerFile.FindStringSubmatch(s.Prompt)
		var answers strings.Builder
		for _, item := range listLine.FindAllStringSubmatch(s.Prompt, -1) {
			fmt.Fprintf(&answers, "%s: ok  README.md:1\n", item[1])
		}
		// The note goes through the same shell, as the session's own write would.
		if err := sh.Exec(ctx, agentport.Exec{User: s.User, Dir: s.Dir, Argv: []string{"/bin/sh", "-c", `cat > "$1"`, "note", m[1]}, Stdin: strings.NewReader(answers.String())}); err != nil {
			a.t.Errorf("writing the answers: %v", err)
		}
	}
	if line != "" {
		if err := sh.Exec(ctx, agentport.Exec{User: s.User, Dir: s.Dir, Argv: []string{"/bin/sh", "-c", line}, Secrets: secrets, Stdout: &out, Stderr: &out}); err != nil {
			a.t.Logf("%s's line: %v: %s", s.Role, err, out.String())
		}
	}
	return agentport.End{Reason: agentport.Finished, Conversation: string(s.Role), Agent: "scripted", Seen: agentport.Seen{Measured: true, Commands: []string{line}, Opened: []string{path.Join(s.Dir, "repo", "README.md")}}}
}

// realCommand makes a target repository and an engine over the real container.
func realCommand(t *testing.T, a *scripted) (*run.Engine, string, string) {
	t.Helper()
	image := os.Getenv(imageVariable)
	if image == "" {
		t.Skip("set " + imageVariable + " to an image built from packaging/runtime")
	}
	cli, err := exec.LookPath("docker")
	if err != nil {
		t.Skip("docker is not installed")
	}
	engine, err := sandbox.NewEngine(cli)
	if err != nil {
		t.Fatal(err)
	}
	dir := testkit.TempDir(t)
	src, target := filepath.Join(dir, "src"), filepath.Join(dir, "target.git")
	testkit.Git(t, dir, "init", "-q", "--bare", target)
	testkit.Git(t, dir, "init", "-q", "-b", "main", src)
	for _, kv := range [][2]string{{"core.autocrlf", "false"}, {"user.name", "Owner"}, {"user.email", "o@example.invalid"}} {
		testkit.Git(t, src, "config", kv[0], kv[1])
	}
	testkit.Git(t, src, "remote", "add", "origin", target)
	write := func(files map[string]string, message string) string {
		for name, content := range files {
			p := filepath.Join(src, filepath.FromSlash(name))
			if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}
		}
		testkit.Git(t, src, "add", "-A")
		testkit.Git(t, src, "commit", "-q", "-m", message)
		testkit.Git(t, src, "push", "-q", "origin", "HEAD:main")
		return strings.TrimSpace(testkit.Git(t, src, "rev-parse", "HEAD"))
	}
	base := write(project1(), "start")
	write(map[string]string{mdir + "catalog/book.md": spec("std/Concept", "Book", "A book has a title and an `isbn`.")}, "book has an isbn")
	host, err := rungit.Open(context.Background(), filepath.Join(dir, "host"), target)
	if err != nil {
		t.Fatal(err)
	}
	return &run.Engine{Git: host, Agents: a, Secrets: [][]byte{[]byte(token)}, Cleanup: time.Minute,
		Sandboxes: &sandbox.Sandboxes{Engine: engine, Image: image, Limits: sandbox.Limits{CPUs: "2", Memory: "2g", Pids: 512, Tmp: "256m"}}}, base, target
}

func newScript(t *testing.T, implementerLines ...string) *scripted {
	return &scripted{t: t, lines: map[agentport.Role][]string{agentport.Implementer: implementerLines}, prompts: map[agentport.Role][]string{}}
}

func TestACommandLandsOnTheRealEngine(t *testing.T) {
	a := newScript(t, `printf 'package catalog\n\ntype Book struct{ Title, ISBN string }\n' > src/catalog/book.go; mkdir -p build; echo left > build/out; printf 'build/\n' > .gitignore`)
	e, base, target := realCommand(t, a)
	// A file the registry does not list fails `check`, and the gate refuses.
	a.lines[agentport.Implementer] = []string{
		`printf 'package catalog\n\ntype Book struct{ Title, ISBN string }\n' > src/catalog/book.go; echo extra > src/catalog/extra.txt`,
		`rm src/catalog/extra.txt`,
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	r, err := e.Run(ctx, run.Request{Mode: agentport.Apply, Target: "main", Base: base, Run: "real-lands"})
	if err != nil {
		t.Fatal(err)
	}
	if r.State != run.StateLanded || r.Repairs != 1 {
		t.Fatalf("%s after %d repairs: %s", r.State, r.Repairs, r.Why)
	}
	if !strings.Contains(a.prompts[agentport.Implementer][1], "file.unregistered") {
		t.Fatalf("the repair prompt:\n%s", a.prompts[agentport.Implementer][1])
	}
	if got := testkit.Git(t, target, "show", "main:src/catalog/book.go"); !strings.Contains(got, "ISBN") {
		t.Fatalf("the work did not land: %q", got)
	}
	if !r.RequiredChecks.Ran || r.RequiredChecks.Commit != r.Work {
		t.Fatalf("required checks %+v", r.RequiredChecks)
	}
	if len(r.Cleanup) != 0 || len(r.AfterLanding) != 0 {
		t.Fatalf("left behind: %v %v", r.Cleanup, r.AfterLanding)
	}
}

// The gate's planted refusals: each session changes what it must not, the gate
// refuses through both repair rounds, and nothing lands.
func TestTheGatesRefusalsHoldOnTheRealEngine(t *testing.T) {
	for name, c := range map[string]struct{ line, want string }{
		"the spec":         {`echo changed >> .markitect/model/catalog/book.md`, "is the owner's"},
		"a required check": {`printf 'checks: []\n' > .markitect/execution.yaml`, "execution.yaml changed"},
		"a failing check":  {`echo x > FAIL; printf '| FAIL | [[project]] | Planted |\n' >> .markitect/registry.md`, "the required check no-fail failed"},
		"a link out":       {`ln -s /dev/shm/good.md notes.md; printf '| notes.md | [[project]] | Planted |\n' >> .markitect/registry.md`, "a link that leaves the repository"},
	} {
		t.Run(name, func(t *testing.T) {
			a := newScript(t, c.line, ":", ":")
			e, base, target := realCommand(t, a)
			before := strings.TrimSpace(testkit.Git(t, target, "rev-parse", "main"))
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			r, err := e.Run(ctx, run.Request{Mode: agentport.Apply, Target: "main", Base: base, Run: "real-" + strings.ReplaceAll(name, " ", "-")})
			if err != nil {
				t.Fatal(err)
			}
			if r.State != run.StateRefused || r.Repairs != 2 || !strings.Contains(r.Why, c.want) {
				t.Fatalf("%s after %d repairs: %s", r.State, r.Repairs, r.Why)
			}
			if got := strings.TrimSpace(testkit.Git(t, target, "rev-parse", "main")); got != before {
				t.Fatal("refused work landed")
			}
			if len(a.prompts[agentport.Reviewer]) != 0 {
				t.Fatal("a reviewer ran on refused work")
			}
		})
	}
}

// A required check is agent-written code: a check the implementer plants to read
// the token or the sessions' leavings finds nothing, and neither does the
// reviewer's copy hold what the implementer left outside Git.
func TestAPlantedCheckFindsNoTokenAndTheReviewerNoLeavings(t *testing.T) {
	spy := `env | grep -q sk-ant && echo TOKEN > FOUND; cat /home/implementer/left.txt /dev/shm/t >> FOUND 2>/dev/null; ls /work/implementer >> FOUND 2>/dev/null; test ! -s FOUND`
	a := newScript(t, `echo "$CLAUDE_CODE_OAUTH_TOKEN" > "$HOME/left.txt"; echo "$CLAUDE_CODE_OAUTH_TOKEN" > /dev/shm/t; chmod 644 /dev/shm/t; echo left > untracked.tmp; printf 'untracked.tmp\n' > .gitignore; printf '| .gitignore | [[project]] | Ignores |\n' >> .markitect/registry.md`)
	a.lines[agentport.Reviewer] = []string{`ls repo > "$HOME/seen.txt"; cat repo/untracked.tmp >> "$HOME/seen.txt" 2>/dev/null; cat /home/implementer/left.txt >> "$HOME/seen.txt" 2>/dev/null; grep -q sk-ant "$HOME/seen.txt" && exit 7; exit 0`}
	e, base, target := realCommand(t, a)
	// The planted check is in the project's own required checks, from the start.
	src := filepath.Join(t.TempDir(), "edit")
	testkit.Git(t, filepath.Dir(src), "clone", "-q", "-b", "main", target, src)
	checks := fmt.Sprintf("checks:\n  - name: spy\n    command: %q\n    timeout: 1m\n    environment: test-image\n", spy)
	if err := os.WriteFile(filepath.Join(src, filepath.FromSlash(project.ChecksFile)), []byte(checks), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, kv := range [][2]string{{"user.name", "Owner"}, {"user.email", "o@example.invalid"}} {
		testkit.Git(t, src, "config", kv[0], kv[1])
	}
	testkit.Git(t, src, "add", "-A")
	testkit.Git(t, src, "commit", "-q", "-m", "a check that spies")
	testkit.Git(t, src, "push", "-q", "origin", "HEAD:main")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	r, err := e.Run(ctx, run.Request{Mode: agentport.Apply, Target: "main", Base: base, Run: "real-spy"})
	if err != nil {
		t.Fatal(err)
	}
	if r.State != run.StateLanded || !r.RequiredChecks.Ran {
		t.Fatalf("%s: %s; checks %+v", r.State, r.Why, r.RequiredChecks)
	}
}

// Work that holds the token does not land.
func TestWorkThatHoldsTheTokenDoesNotLand(t *testing.T) {
	a := newScript(t, `echo "$CLAUDE_CODE_OAUTH_TOKEN" >> src/catalog/book.go`)
	e, base, target := realCommand(t, a)
	before := strings.TrimSpace(testkit.Git(t, target, "rev-parse", "main"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	r, err := e.Run(ctx, run.Request{Mode: agentport.Apply, Target: "main", Base: base, Run: "real-token"})
	if err != nil {
		t.Fatal(err)
	}
	if r.State != run.StateKept || !strings.Contains(r.Why, "token") {
		t.Fatalf("%s: %s", r.State, r.Why)
	}
	if got := strings.TrimSpace(testkit.Git(t, target, "rev-parse", "main")); got != before {
		t.Fatal("the token landed")
	}
	if strings.Contains(testkit.Git(t, target, "log", "--all", "-p"), token) {
		t.Fatal("the token reached the target repository")
	}
}

func TestVerifyOnTheRealEngine(t *testing.T) {
	a := newScript(t)
	e, base, target := realCommand(t, a)
	before := strings.TrimSpace(testkit.Git(t, target, "rev-parse", "main"))
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	r, err := e.Run(ctx, run.Request{Mode: agentport.Verify, Target: "main", Base: base, Run: "real-verify"})
	if err != nil {
		t.Fatal(err)
	}
	if r.State != run.StateVerified || !r.RequiredChecks.Ran {
		t.Fatalf("%s: %s; checks %+v", r.State, r.Why, r.RequiredChecks)
	}
	if got := strings.TrimSpace(testkit.Git(t, target, "rev-parse", "main")); got != before {
		t.Fatal("verify changed the target")
	}
	if !strings.Contains(testkit.Git(t, target, "show", run.RunBranchPrefix+r.Run+":"+run.ResultFile(r.Run)), "verified") {
		t.Fatal("verify's Result is not on its branch")
	}
}
