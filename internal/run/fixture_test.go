package run_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/rungit"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) {
	gitToolsOnPath()
	testkit.Main(m)
}

// gitToolsOnPath puts Git for Windows' POSIX tools on the PATH when no sh is on
// it, so the fake sandbox finds sh for the required checks.
func gitToolsOnPath() {
	if runtime.GOOS != "windows" {
		return
	}
	if _, err := exec.LookPath("sh"); err == nil {
		return
	}
	out, err := exec.Command("git", "--exec-path").Output()
	if err != nil {
		return
	}
	// The exec path is <root>/mingw64/libexec/git-core; the tools are in <root>/usr/bin.
	tools := filepath.Join(filepath.Dir(filepath.Dir(filepath.Dir(strings.TrimSpace(string(out))))), "usr", "bin")
	if _, err := os.Stat(filepath.Join(tools, "sh.exe")); err == nil {
		os.Setenv("PATH", os.Getenv("PATH")+string(os.PathListSeparator)+tools)
	}
}

const (
	m     = project.ModelDir + "/"
	table = "| File | Realizes | Why |\n|---|---|---|\n"
	// failCheck fails when the tree holds a file named FAIL.
	failCheck = `test ! -e FAIL`
)

func def(kind, title, body string) string {
	return "---\nkind: " + kind + "\n---\n# " + title + "\n\n" + body + "\n"
}

// library is the fixture project: Root with a Rule, members with a sub-Package
// loans, and catalog. The required check fails on a file named FAIL.
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

// fixture is one target repository with the host's real Git, a sandbox of plain
// folders and scripted sessions.
type fixture struct {
	t      *testing.T
	src    string // a clone of the target that the test changes
	target string // the bare target repository
	host   *rungit.Repo
	box    *sandboxes
	agents *script
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
	f.box = &sandboxes{t: t, dir: filepath.Join(dir, "boxes")}
	f.agents = &script{t: t, steps: map[agentport.Role][]step{}}
	f.engine = &run.Engine{Git: host, Sandboxes: f.box, Agents: f.agents, Cleanup: time.Minute}
	return f
}

func (f *fixture) write(files map[string]string) {
	f.t.Helper()
	for name, content := range files {
		writeFile(f.t, f.src, name, content)
	}
}

func writeFile(t *testing.T, dir, name, content string) {
	t.Helper()
	p := filepath.Join(dir, filepath.FromSlash(name))
	if content == "" {
		if err := os.Remove(p); err != nil {
			t.Fatal(err)
		}
		return
	}
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
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
	r, err := f.engine.Run(context.Background(), req)
	if err != nil {
		f.t.Fatal(err)
	}
	if !f.box.closed() {
		f.t.Fatal("the Command's container was not closed")
	}
	return r
}

func (f *fixture) tip(branch string) string {
	f.t.Helper()
	cmd := exec.Command("git", "--git-dir="+f.target, "rev-parse", "--verify", "--quiet", "refs/heads/"+branch)
	out, _ := cmd.Output()
	return strings.TrimSpace(string(out))
}

func (f *fixture) show(ref, file string) string {
	f.t.Helper()
	return testkit.Git(f.t, f.target, "show", ref+":"+file)
}

// has says whether a ref holds a file.
func (f *fixture) has(ref, file string) bool {
	return exec.Command("git", "--git-dir="+f.target, "cat-file", "-e", ref+":"+file).Run() == nil
}

// changeBook changes the catalog's book Definition: the item catalog/book.
func changeBook(f *fixture) (base, start string) {
	base = f.tip("main")
	f.write(map[string]string{m + "catalog/book.md": def("std/Concept", "Book", "A book has a title and an `isbn`.")})
	return base, f.commit("book has an isbn")
}

// A scripted session: what it does in its folder, and what its transcript shows.
type step func(s agentport.Session, repo string) agentport.End

// script plays the steps of each role in order. A role that runs out fails the
// test, and every session is recorded.
type script struct {
	t     *testing.T
	mu    sync.Mutex
	steps map[agentport.Role][]step
	// sessions holds every session started, in order.
	sessions []agentport.Session
}

func (a *script) on(role agentport.Role, steps ...step) {
	a.steps[role] = append(a.steps[role], steps...)
}

func (a *script) Run(ctx context.Context, sh agentport.Shell, s agentport.Session) agentport.End {
	a.mu.Lock()
	a.sessions = append(a.sessions, s)
	var st step
	if q := a.steps[s.Role]; len(q) > 0 {
		st, a.steps[s.Role] = q[0], q[1:]
	}
	a.mu.Unlock()
	if st == nil {
		return agentport.End{Reason: agentport.Failed, Class: agentport.Other, Detail: fmt.Sprintf("no scripted step for %s %d", s.Role, s.Round)}
	}
	repo := s.Dir
	if s.Role == agentport.Reviewer {
		repo = path.Join(s.Dir, "repo")
	}
	end := st(s, repo)
	if end.Reason == "" {
		end.Reason = agentport.Finished
	}
	if end.Conversation == "" {
		end.Conversation = fmt.Sprintf("%s-%d", s.Role, s.Round)
	}
	end.Agent = "fake:model"
	return end
}

func (a *script) of(role agentport.Role) []agentport.Session {
	var out []agentport.Session
	for _, s := range a.sessions {
		if s.Role == role {
			out = append(out, s)
		}
	}
	return out
}

// edit is an implementer's session that writes files; "" removes one.
func (f *fixture) edit(files map[string]string) step {
	return func(s agentport.Session, repo string) agentport.End {
		for name, content := range files {
			writeFile(f.t, repo, name, content)
		}
		return agentport.End{}
	}
}

var itemLine = regexp.MustCompile(`(?m)^\[ \] (.+?)(?:: ".*")?$`)

// listed reads the items a prompt holds.
func listed(prompt string) []string {
	var keys []string
	for _, m := range itemLine.FindAllStringSubmatch(prompt, -1) {
		keys = append(keys, m[1])
	}
	return keys
}

// answersNote is where the prompt says to answer.
var answerPath = regexp.MustCompile(`(?m)^Write one line per item to (.+), with pointers`)

// answer is a reviewer's session: for each item on its list, the line words
// returns, or an ok that points at README.md, which the transcript shows read.
func (f *fixture) answer(words map[string]string, closing string) step {
	return func(s agentport.Session, repo string) agentport.End {
		note := answerPath.FindStringSubmatch(s.Prompt)
		if note == nil {
			f.t.Fatalf("the prompt says nowhere to answer:\n%s", s.Prompt)
		}
		var b strings.Builder
		for _, key := range listed(s.Prompt) {
			if w, ok := words[key]; ok {
				if w != "" {
					fmt.Fprintf(&b, "%s: %s\n", key, w)
				}
				continue
			}
			fmt.Fprintf(&b, "%s: ok  README.md:1\n", key)
		}
		if err := os.MkdirAll(filepath.Dir(note[1]), 0o755); err != nil {
			f.t.Fatal(err)
		}
		if err := os.WriteFile(note[1], []byte(b.String()), 0o644); err != nil {
			f.t.Fatal(err)
		}
		return agentport.End{Seen: agentport.Seen{Measured: true, Opened: []string{path.Join(repo, "README.md")}}, Text: closing}
	}
}

func (f *fixture) ok() step { return f.answer(nil, "") }

// sandboxes starts plain folders with real Git in place of a container.
type sandboxes struct {
	t    *testing.T
	dir  string
	mu   sync.Mutex
	last *sandbox
	// checkExit makes a check's start fail as an environment that lacks its
	// program does.
	notRun bool
	// onCollect runs just before a commit of the implementer's folder, as a
	// leftover of its session would.
	all []*sandbox
}

func (b *sandboxes) closed() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	for _, s := range b.all {
		if !s.isClosed {
			return false
		}
	}
	return true
}

func (b *sandboxes) Start(ctx context.Context, name, commit string, bundle io.Reader) (run.Sandbox, error) {
	root := filepath.Join(b.dir, name)
	s := &sandbox{t: b.t, root: root, tools: filepath.Join(root, "tools.git"), owner: b}
	if err := os.MkdirAll(root, 0o755); err != nil {
		return nil, err
	}
	s.git(root, "init", "-q", "--bare", s.tools)
	if err := s.bring(bundle); err != nil {
		return nil, err
	}
	b.mu.Lock()
	b.all = append(b.all, s)
	b.last = s
	b.mu.Unlock()
	return s, nil
}

type sandbox struct {
	t        *testing.T
	root     string
	tools    string
	owner    *sandboxes
	mu       sync.Mutex
	n        int
	repo     string // the implementer's folder
	isClosed bool
	// checked lists the commits the required checks ran on.
	checked []string
}

func (s *sandbox) git(dir string, args ...string) string {
	cmd := exec.Command("git", append([]string{"-c", "core.autocrlf=false", "-c", "user.name=Markitect", "-c", "user.email=m@example.invalid"}, args...)...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		s.t.Fatalf("git %v in %s: %v: %s", args, dir, err, out)
	}
	return strings.TrimSpace(string(out))
}

func (s *sandbox) bring(bundle io.Reader) error {
	f, err := os.CreateTemp(s.root, "in-*.bundle")
	if err != nil {
		return err
	}
	name := f.Name()
	if _, err := io.Copy(f, bundle); err != nil {
		f.Close()
		return err
	}
	f.Close()
	defer os.Remove(name)
	s.git(s.root, "--git-dir="+s.tools, "fetch", "-q", name, "+refs/*:refs/in/*")
	return nil
}

func (s *sandbox) copyOf(user, commit string) string {
	s.mu.Lock()
	s.n++
	folder := filepath.Join(s.root, user, fmt.Sprint(s.n))
	s.mu.Unlock()
	repo := filepath.Join(folder, "repo")
	if err := os.MkdirAll(folder, 0o755); err != nil {
		s.t.Fatal(err)
	}
	s.git(folder, "clone", "-q", "--no-checkout", s.tools, "repo")
	s.git(repo, "fetch", "-q", s.tools, commit)
	s.git(repo, "checkout", "-q", "--detach", commit)
	return filepath.ToSlash(repo)
}

func (s *sandbox) Deliver(ctx context.Context, user, commit string) (string, error) {
	if err := os.RemoveAll(filepath.Join(s.root, "notes", user)); err != nil {
		return "", err
	}
	repo := s.copyOf(user, commit)
	if user == agentport.ImplementerUser {
		s.repo = repo
	}
	return repo, nil
}

func (s *sandbox) Collect(ctx context.Context, parent, message string) (string, io.ReadCloser, error) {
	s.git(s.repo, "add", "-A")
	tree := s.git(s.repo, "write-tree")
	if tree == s.git(s.repo, "rev-parse", parent+"^{tree}") {
		return parent, nil, nil
	}
	commit := s.git(s.repo, "commit-tree", tree, "-p", parent, "-m", message)
	s.git(s.repo, "branch", "-f", "work", commit)
	s.git(s.repo, "push", "-q", s.tools, "+"+commit+":refs/heads/work")
	file := filepath.Join(s.root, "out.bundle")
	s.git(s.repo, "bundle", "create", "-q", file, "refs/heads/work", "^"+parent)
	data, err := os.ReadFile(file)
	return commit, io.NopCloser(bytes.NewReader(data)), err
}

func (s *sandbox) Check(ctx context.Context, bundle io.Reader, commit string, checks []project.Check) ([]run.CheckResult, error) {
	if bundle != nil {
		if err := s.bring(bundle); err != nil {
			return nil, err
		}
	}
	s.mu.Lock()
	s.checked = append(s.checked, commit)
	s.mu.Unlock()
	if s.owner.notRun {
		return []run.CheckResult{{Name: checks[0].Name, Exit: 127, NotRun: true, Output: "sh: no such program"}}, nil
	}
	repo := s.copyOf("checks", commit)
	var results []run.CheckResult
	for _, c := range checks {
		cmd := exec.Command("sh", "-c", c.Command)
		cmd.Dir = repo
		var out bytes.Buffer
		cmd.Stdout, cmd.Stderr = &out, &out
		err := cmd.Run()
		results = append(results, run.CheckResult{Name: c.Name, Passed: err == nil, Exit: cmd.ProcessState.ExitCode(), Output: out.String()})
		if err != nil {
			break
		}
	}
	return results, nil
}

func (s *sandbox) NotePath(user, name string) string {
	return filepath.ToSlash(filepath.Join(s.root, "notes", user, name))
}

func (s *sandbox) Note(ctx context.Context, user, name string) ([]byte, bool, error) {
	data, err := os.ReadFile(s.NotePath(user, name))
	if os.IsNotExist(err) {
		return nil, false, nil
	}
	return data, err == nil, err
}

func (s *sandbox) Exec(ctx context.Context, e agentport.Exec) error { return nil }

func (s *sandbox) Close(ctx context.Context) error {
	s.isClosed = true
	return nil
}
