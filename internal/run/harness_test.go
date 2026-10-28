package run_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/remoteserver"
	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/agentport/agentfake"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

// The Run remote's pre-receive hook is this test binary, started by Git with
// MARKITECT_REMOTE_TEST_HOOK set, so scripted agents push through the real
// hooks over real Git smart HTTP.
func TestMain(m *testing.M) {
	if os.Getenv("MARKITECT_REMOTE_TEST_HOOK") == "1" {
		os.Exit(remoteserver.Main(context.Background(), os.Args[1:], os.Stdin, os.Stdout, os.Stderr))
	}
	testkit.Main(m)
}

const (
	m     = project.ModelDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
	// failCheck fails when the tree holds a file named FAIL.
	failCheck = `! find . -name FAIL | grep -q .`
)

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

// library is the fixture project: Root with a Rule and the anchor, members
// with a sub-Package loans, and catalog. The required check fails on a file
// named FAIL, which a scripted maker may register to break its submission.
func library() map[string]string {
	files, _ := libraries.Files("std")
	std := model.LibrarySource{Name: "std", Version: libraries.Version}
	for _, f := range files {
		std.Files = append(std.Files, model.File{Path: f.Path, Data: f.Data})
	}
	return map[string]string{
		project.ManifestFile:                      "libraries:\n  std: 1.0.0\n",
		project.LockFile:                          string(model.RenderLock([]model.LibrarySource{std})),
		project.ChecksFile:                        "checks:\n  - name: no-fail\n    command: '" + failCheck + "'\n    timeout: 1m\n    environment: test-image\n",
		m + "project.md":                          def("std/Project", "Library", "Parts: [[members]], [[catalog]]."),
		m + "loan-limit.md":                       def("std/Rule", "Loan limit", "At most `5` open loans."),
		m + "members/member.md":                   def("std/Concept", "Member", "A person."),
		m + "members/loans/loan.md":               def("std/Concept", "Loan", "A [[members/member]] borrows."),
		m + "catalog/book.md":                     def("std/Concept", "Book", "A book has a title."),
		project.RootPage:                          table + "| README.md | [[project]] | Overview |\n",
		project.RegistryDir + "/members.md":       "---\npaths: [src/members/**]\n---\n" + table + "| src/members/member.go | [[members/member]] | Type |\n",
		project.RegistryDir + "/members/loans.md": "---\npaths: [src/members/loans/**]\n---\n" + table + "| src/members/loans/loan.go | [[members/loans/loan]] | Type |\n",
		project.RegistryDir + "/catalog.md":       "---\npaths: [src/catalog/**]\n---\n" + table + "| src/catalog/book.go | [[catalog/book]] | Type |\n",
		"README.md":                               "# Library\n",
		"src/members/member.go":                   "package members\n",
		"src/members/loans/loan.go":               "package loans\n",
		"src/catalog/book.go":                     "package catalog\n",
	}
}

// fixture is one target repository with a host Git, a Run remote that
// serves in this process, and scripted agents.
type fixture struct {
	t      *testing.T
	src    string // a clone of the target that the test changes
	target string // the bare target repository
	host   *rungit.Repo
	agents *agentfake.Agents
	remote *remotes
	engine *run.Engine
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	dir := testkit.TempDir(t)
	f := &fixture{t: t, src: filepath.Join(dir, "src"), target: filepath.Join(dir, "target.git")}
	testkit.Git(t, dir, "init", "-q", "--bare", f.target)
	testkit.Git(t, dir, "init", "-q", "-b", "main", f.src)
	for _, kv := range [][2]string{{"core.autocrlf", "false"}, {"user.name", "Owner"}, {"user.email", "owner@example.invalid"}} {
		testkit.Git(t, f.src, "config", kv[0], kv[1])
	}
	testkit.Git(t, f.src, "remote", "add", "origin", f.target)
	f.write(library())
	f.commit("start")
	host, err := rungit.Open(context.Background(), filepath.Join(dir, "host"), f.target)
	if err != nil {
		t.Fatal(err)
	}
	f.host = host
	f.agents = agentfake.New(filepath.Join(dir, "agents"))
	f.remote = &remotes{t: t, dir: filepath.Join(dir, "remotes")}
	f.engine = &run.Engine{Git: host, Remotes: f.remote, Agents: f.agents, Cleanup: time.Minute}
	return f
}

func (f *fixture) write(files map[string]string) {
	f.t.Helper()
	for name, content := range files {
		p := filepath.Join(f.src, filepath.FromSlash(name))
		if content == "" {
			if err := os.Remove(p); err != nil {
				f.t.Fatal(err)
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			f.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
}

// commit commits the clone and pushes it to the target's main.
func (f *fixture) commit(message string) string {
	f.t.Helper()
	testkit.Git(f.t, f.src, "add", "-A")
	testkit.Git(f.t, f.src, "commit", "-q", "-m", message)
	testkit.Git(f.t, f.src, "push", "-q", "origin", "HEAD:main")
	return strings.TrimSpace(testkit.Git(f.t, f.src, "rev-parse", "HEAD"))
}

// pull brings the clone to the target's main.
func (f *fixture) pull() {
	f.t.Helper()
	testkit.Git(f.t, f.src, "pull", "-q", "--ff-only", "origin", "main")
}

func (f *fixture) run(req run.Request) *run.Result {
	f.t.Helper()
	if req.Target == "" {
		req.Target = "main"
	}
	if req.Mode == "" {
		req.Mode = agentport.Apply
	}
	if req.Limits == (run.Limits{}) {
		req.Limits = run.Limits{Deadline: time.Now().Add(5 * time.Minute), Starts: 40, Rounds: 4, Start: 2 * time.Minute}
	}
	r, err := f.engine.Run(context.Background(), req)
	if err != nil {
		f.t.Fatal(err)
	}
	for _, st := range r.Starts {
		f.t.Logf("start %s %s %d: %s %s", st.Manager, st.Role, st.Round, st.Reason, st.Detail)
	}
	if left := f.agents.Left(); len(left) > 0 {
		f.t.Errorf("scripted steps not used: %v", left)
	}
	if len(r.Cleanup) > 0 {
		f.t.Errorf("cleanup: %v", r.Cleanup)
	}
	return r
}

// tip is the target's branch tip, or "" when the branch does not exist.
func (f *fixture) tip(branch string) string {
	f.t.Helper()
	out, err := exec.Command("git", "--git-dir="+f.target, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func (f *fixture) show(rev, file string) string {
	f.t.Helper()
	return testkit.Git(f.t, f.target, "show", rev+":"+file)
}

// remotes starts Run remotes in this process: a seeded bare repository with
// the real hooks behind Git smart HTTP.
type remotes struct {
	t       *testing.T
	dir     string
	started []*remote
}

type remote struct {
	l    remoteserver.Layout
	srv  *httptest.Server
	plan *runremote.Plan
	git  string
}

func (rs *remotes) Start(ctx context.Context, plan *runremote.Plan, bundle io.Reader) (run.Remote, error) {
	git, err := exec.LookPath("git")
	if err != nil {
		return nil, err
	}
	l := remoteserver.Layout{Root: filepath.Join(rs.dir, plan.Run)}
	if err := os.MkdirAll(l.Root, 0o700); err != nil {
		return nil, err
	}
	b, err := json.Marshal(plan)
	if err != nil {
		return nil, err
	}
	if err := os.WriteFile(l.Plan(), b, 0o600); err != nil {
		return nil, err
	}
	self, err := filepath.Abs(os.Args[0])
	if err != nil {
		return nil, err
	}
	if err := remoteserver.Seed(l, git, bundle, []string{self}); err != nil {
		return nil, err
	}
	loaded, err := remoteserver.LoadPlan(l)
	if err != nil {
		return nil, err
	}
	srv := httptest.NewServer(&remoteserver.Server{Layout: l, Plan: loaded, Git: git, Env: []string{"MARKITECT_REMOTE_TEST_HOOK=1"}})
	r := &remote{l: l, srv: srv, plan: loaded, git: git}
	rs.started = append(rs.started, r)
	return r, nil
}

func (r *remote) URL() string     { return r.srv.URL + "/run.git" }
func (r *remote) Network() string { return "test-network" }

func (r *remote) Export(_ context.Context, w io.Writer) error {
	return remoteserver.Export(r.l, r.git, w)
}

func (r *remote) Log(context.Context) ([]run.Push, error) {
	data, err := os.ReadFile(r.l.Log())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var pushes []run.Push
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		var e remoteserver.Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, err
		}
		pushes = append(pushes, run.Push{User: e.User, Updates: e.Updates, Decision: e.Decision})
	}
	return pushes, sc.Err()
}

// Check runs the required checks on a commit the host made, as the
// container's remote does.
func (r *remote) Check(_ context.Context, bundle io.Reader, commit string) ([]runremote.CheckResult, error) {
	return remoteserver.LandingCheck(r.l, r.git, bundle, commit, io.Discard)
}

func (r *remote) Close(context.Context) error {
	r.srv.Close()
	return nil
}

func errorf(format string, args ...any) error { return fmt.Errorf(format, args...) }
