package remoteserver

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http/httptest"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// The scripted Managers of these tests are Git clients that push to a real
// remote over smart HTTP; the remote's hook is this test binary, started by
// Git with MARKITECT_REMOTE_TEST_HOOK set.
func TestMain(m *testing.M) {
	if os.Getenv("MARKITECT_REMOTE_TEST_HOOK") == "1" {
		os.Exit(Main(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
	}
	os.Exit(m.Run())
}

// The fixture project: Root with the children members and loans, and cards
// below members. Every Package works; each has a maker and a checker.
var fixtureFiles = map[string]string{
	"README.md":                            "# Library\n",
	"go.work":                              "go 1.27\n",
	"src/members/member.go":                "package members\n",
	"src/members/cards/card.go":            "package cards\n",
	"src/loans/loan.go":                    "package loans\n",
	".markitect/manifest.yaml":             "libraries:\n  std: 1.0.0\n",
	".markitect/lock.yaml":                 "std 1.0.0\n",
	".markitect/execution.yaml":            "checks: []\n",
	".markitect/ignore.md":                 "| Path | Why |\n|---|---|\n",
	".markitect/model/project.md":          "---\nkind: std/Project\n---\n# Library\n",
	".markitect/model/members/member.md":   "---\nkind: std/Concept\n---\n# Member\n",
	".markitect/model/loans/loan.md":       "---\nkind: std/Concept\n---\n# Loan\n",
	".markitect/registry.md":               "# Root\n",
	".markitect/registry/members.md":       "# members\n",
	".markitect/registry/members/cards.md": "# cards\n",
	".markitect/registry/loans.md":         "# loans\n",
}

// failCheck fails when the tree holds a file named FAIL, so a scripted maker
// decides whether its submission passes.
const failCheck = `test -z "$(find . -name FAIL -not -path './.git/*')"`

// The tests name Managers as trailers do: "." for Root.
func id(name string) string {
	if name == "." {
		return model.RootPackage
	}
	return name
}

func fixturePlan(base string) *runremote.Plan {
	p := &runremote.Plan{
		Run:    "test-run",
		Mode:   runremote.Apply,
		Base:   base,
		Checks: []project.Check{{Name: "no-fail-marker", Command: failCheck, Timeout: time.Minute, Environment: "test-image"}},
		Managers: []runremote.Manager{
			{ID: "", Working: true, Checklist: []string{"project"}},
			{ID: "loans", Patterns: []string{"src/loans/**"}, Working: true, Checklist: []string{"loans/loan"}},
			{ID: "members", Patterns: []string{"src/members/**"}, Working: true, Checklist: []string{"members/member"}},
			{ID: "members/cards", Patterns: []string{"src/members/cards/**"}, Working: true, Checklist: []string{"members/cards/card"}},
		},
		Pages: map[string]string{},
	}
	for _, m := range p.Managers {
		p.Pages[runremote.PageFile(m.ID)] = m.ID
		for _, role := range []runremote.Role{runremote.Maker, runremote.Checker} {
			user := userOf(runremote.Name(m.ID), role)
			p.Agents = append(p.Agents, runremote.Agent{User: user, Manager: m.ID, Role: role, TokenSHA256: tokenHash(user)})
		}
	}
	return p
}

func userOf(name string, role runremote.Role) string {
	short := map[string]string{".": "root", "members": "members", "members/cards": "cards", "loans": "loans"}[name]
	return short + "-" + string(role)
}

func tokenOf(user string) string { return "token-" + user }

func tokenHash(user string) string {
	sum := sha256.Sum256([]byte(tokenOf(user)))
	return hex.EncodeToString(sum[:])
}

type remote struct {
	t    *testing.T
	l    Layout
	url  string
	plan *runremote.Plan
	git  string
}

// newRemote seeds a remote from the fixture and serves it; mutate may change
// the plan before seeding.
func newRemote(t *testing.T, mutate func(*runremote.Plan)) *remote {
	t.Helper()
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is not installed")
	}
	src := t.TempDir()
	run(t, src, "init", "-q", "-b", "main")
	for name, content := range fixtureFiles {
		write(t, src, name, content)
	}
	run(t, src, "add", "-A")
	run(t, src, "commit", "-q", "-m", "base")
	base := run(t, src, "rev-parse", "HEAD")
	bundle := filepath.Join(t.TempDir(), "start.bundle")
	run(t, src, "bundle", "create", "-q", bundle, "main")

	plan := fixturePlan(base)
	if mutate != nil {
		mutate(plan)
	}
	l := Layout{Root: filepath.Join(t.TempDir(), "markitect")}
	if err := os.MkdirAll(l.Root, 0o700); err != nil {
		t.Fatal(err)
	}
	b, _ := json.Marshal(plan)
	if err := os.WriteFile(l.Plan(), b, 0o600); err != nil {
		t.Fatal(err)
	}
	f, err := os.Open(bundle)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	self, err := filepath.Abs(os.Args[0])
	if err != nil {
		t.Fatal(err)
	}
	if err := Seed(l, git, f, []string{self}); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadPlan(l)
	if err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(&Server{Layout: l, Plan: loaded, Git: git, Env: []string{"MARKITECT_REMOTE_TEST_HOOK=1"}})
	t.Cleanup(srv.Close)
	return &remote{t: t, l: l, url: srv.URL + "/" + repoName, plan: plan, git: git}
}

// agent is one scripted maker or checker with its own clone, on its own
// branch.
type agent struct {
	t      *testing.T
	dir    string
	user   string
	branch string
}

func (r *remote) agent(manager string, role runremote.Role) *agent {
	r.t.Helper()
	user := userOf(manager, role)
	u, _ := url.Parse(r.url)
	u.User = url.UserPassword(user, tokenOf(user))
	dir := filepath.Join(r.t.TempDir(), user)
	run(r.t, "", "clone", "-q", u.String(), dir)
	a := &agent{t: r.t, dir: dir, user: user, branch: runremote.Branch(id(manager))}
	a.git("checkout", "-q", "-B", "work", "origin/"+a.branch)
	return a
}

func (a *agent) git(args ...string) string {
	a.t.Helper()
	return run(a.t, a.dir, args...)
}

func (a *agent) write(name, content string) { write(a.t, a.dir, name, content) }

func (a *agent) commit(message string) string {
	a.t.Helper()
	a.git("add", "-A")
	a.git("commit", "-q", "--allow-empty", "-m", message)
	return a.git("rev-parse", "HEAD")
}

// push pushes HEAD to the agent's own branch and returns the remote's
// answer and whether it was accepted.
func (a *agent) push(extra ...string) (string, bool) {
	a.t.Helper()
	return a.pushTo("HEAD:refs/heads/"+a.branch, extra...)
}

func (a *agent) pushTo(refspec string, extra ...string) (string, bool) {
	a.t.Helper()
	out, err := gitCmd(a.dir, append(append([]string{"push", "--porcelain", "origin"}, extra...), refspec)...)
	return out, err == nil
}

// sync fetches every branch of the remote.
func (a *agent) sync() { a.git("fetch", "-q", "origin") }

func (a *agent) mustPush() {
	a.t.Helper()
	if out, ok := a.push(); !ok {
		a.t.Fatalf("%s: push refused:\n%s", a.user, out)
	}
}

func (a *agent) mustRefuse(want string) string {
	a.t.Helper()
	out, ok := a.push()
	if ok {
		a.t.Fatalf("%s: push accepted, want a refusal containing %q:\n%s", a.user, want, out)
	}
	if !strings.Contains(out, want) {
		a.t.Fatalf("%s: refusal lacks %q:\n%s", a.user, want, out)
	}
	return out
}

// submission returns a message whose trailers give every item a verdict.
func submission(manager string, items ...string) string {
	t := runremote.Trailers{Manager: manager}
	for _, item := range items {
		t.Verdicts = append(t.Verdicts, runremote.Line{Item: item, Word: runremote.Changed, Evidence: "realized and tested"})
	}
	return "Submit " + manager + "\n\n" + t.String()
}

// review returns a checker's message answering every item.
func review(manager, outcome string, answers map[string]string) string {
	t := runremote.Trailers{Manager: manager, Review: outcome}
	for _, item := range sortedKeys(answers) {
		t.Checks = append(t.Checks, runremote.Line{Item: item, Word: answers[item], Evidence: "re-checked"})
	}
	return "Review " + manager + "\n\n" + t.String()
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// approve has the Manager's checker approve the submission at its branch tip.
func (r *remote) approve(manager string, items ...string) string {
	r.t.Helper()
	c := r.agent(manager, runremote.Checker)
	answers := map[string]string{}
	for _, item := range items {
		answers[item] = runremote.Confirmed
	}
	id := c.commit(review(manager, runremote.Approved, answers))
	c.mustPush()
	return id
}

func (r *remote) log() []Entry {
	r.t.Helper()
	b, err := os.ReadFile(r.l.Log())
	if err != nil {
		r.t.Fatal(err)
	}
	var entries []Entry
	for _, line := range bytes.Split(bytes.TrimSpace(b), []byte("\n")) {
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			r.t.Fatal(err)
		}
		entries = append(entries, e)
	}
	return entries
}

func write(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func run(t *testing.T, dir string, args ...string) string {
	t.Helper()
	out, err := gitCmd(dir, args...)
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return strings.TrimSpace(out)
}

func gitCmd(dir string, args ...string) (string, error) {
	home := os.TempDir()
	cmd := exec.Command("git", append([]string{"-c", "user.name=Scripted Manager", "-c", "user.email=manager@run",
		"-c", "core.autocrlf=false", "-c", "credential.helper=", "-c", "init.defaultBranch=main", "-c", "commit.gpgsign=false"}, args...)...)
	cmd.Dir = dir
	cmd.Env = append(gitConfigEnv(), "PATH="+os.Getenv("PATH"), "HOME="+home, "GCM_INTERACTIVE=never")
	for _, k := range []string{"SYSTEMROOT", "WINDIR", "TEMP", "TMP", "COMSPEC", "PATHEXT"} {
		if v, ok := os.LookupEnv(k); ok {
			cmd.Env = append(cmd.Env, k+"="+v)
		}
	}
	var out bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &out
	err := cmd.Run()
	return out.String(), err
}
