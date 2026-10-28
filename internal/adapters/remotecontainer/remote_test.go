package remotecontainer

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// These tests start real Run containers. They need
//
//	MARKITECT_TEST_ENGINE         absolute path of the docker CLI
//	MARKITECT_TEST_REMOTE_IMAGE   an image with Git, a POSIX shell and coreutils
//
// and build markitect-remote for the engine with the go command on PATH.

type fixture struct {
	t      *testing.T
	ctx    context.Context
	engine *Engine
	image  string
	binary string
	run    string
}

func newFixture(t *testing.T) *fixture {
	enginePath, image := os.Getenv("MARKITECT_TEST_ENGINE"), os.Getenv("MARKITECT_TEST_REMOTE_IMAGE")
	if enginePath == "" || image == "" {
		t.Skip("set MARKITECT_TEST_ENGINE and MARKITECT_TEST_REMOTE_IMAGE")
	}
	e, err := NewEngine(enginePath)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	t.Cleanup(cancel)
	arch, err := e.Arch(ctx)
	if err != nil {
		t.Fatal(err)
	}
	binary := filepath.Join(t.TempDir(), "markitect-remote")
	build := exec.CommandContext(ctx, "go", "build", "-trimpath", "-o", binary, "github.com/Glacius-Labs/Markitect/cmd/markitect-remote")
	build.Env = append(os.Environ(), "GOOS=linux", "GOARCH="+arch, "CGO_ENABLED=0")
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build markitect-remote: %v\n%s", err, out)
	}
	run := fmt.Sprintf("t3-%d", time.Now().UnixNano())
	t.Cleanup(func() {
		if err := Sweep(context.Background(), e, run); err != nil {
			t.Errorf("sweep: %v", err)
		}
	})
	return &fixture{t: t, ctx: ctx, engine: e, image: image, binary: binary, run: run}
}

// source commits the fixture project in a fresh repository and returns its
// directory and base commit.
func (f *fixture) source() (string, string) {
	dir := f.t.TempDir()
	files := map[string]string{
		"README.md":                      "# Library\n",
		"ci.sh":                          "exit 0\n",
		"src/members/member.go":          "package members\n",
		".markitect/model/project.md":    "# Library\n",
		".markitect/registry.md":         "# Root\n",
		".markitect/registry/members.md": "# members\n",
	}
	for name, content := range files {
		p := filepath.Join(dir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			f.t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			f.t.Fatal(err)
		}
	}
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=Owner", "-c", "user.email=owner@example", "-c", "core.autocrlf=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			f.t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q")
	git("add", "-A")
	git("commit", "-q", "-m", "base")
	return dir, git("rev-parse", "HEAD")
}

func token(user string) string { return "token-" + user }

func (f *fixture) plan(base string) *runremote.Plan {
	p := &runremote.Plan{Run: f.run, Mode: runremote.Apply, Base: base,
		Checks: []project.Check{{Name: "ci", Command: "sh ci.sh", Timeout: 2 * time.Minute, Environment: f.image}},
		Managers: []runremote.Manager{
			{ID: "", Working: true, Checklist: []string{"project"}},
			{ID: "members", Patterns: []string{"src/members/**"}, Working: true, Checklist: []string{"members/member"}},
		},
		Pages: map[string]string{project.RootPage: "", project.RegistryDir + "/members.md": "members"}}
	for _, a := range []struct {
		user, manager string
		role          runremote.Role
	}{{"root-maker", "", runremote.Maker}, {"root-checker", "", runremote.Checker}, {"members-maker", "members", runremote.Maker}, {"members-checker", "members", runremote.Checker}} {
		sum := sha256.Sum256([]byte(token(a.user)))
		p.Agents = append(p.Agents, runremote.Agent{User: a.user, Manager: a.manager, Role: a.role, TokenSHA256: hex.EncodeToString(sum[:])})
	}
	return p
}

func (f *fixture) start(plan *runremote.Plan, repo, commit string) (*Remote, error) {
	var bundle bytes.Buffer
	if err := StartBundle(f.ctx, "git", repo, commit, &bundle); err != nil {
		f.t.Fatal(err)
	}
	return Start(f.ctx, f.engine, Spec{Binary: f.binary, Plan: plan, Limits: Limits{CPUs: "2", Memory: "1g", Pids: 512}}, &bundle)
}

// manager starts one scripted agent's container on the Run's network.
func (f *fixture) manager(r *Remote, user string) string {
	name := r.Container[:len(r.Container)-len("remote")] + user
	if err := f.engine.run(f.ctx, nil, nil, "run", "--detach", "--name", name, "--label", RunLabel+"="+f.run,
		"--network", r.Network, "--user", "1000:1000", "--env", "HOME=/tmp", "--entrypoint", "sleep", f.image, "infinity"); err != nil {
		f.t.Fatal(err)
	}
	url := strings.Replace(r.URL, "http://", "http://"+user+":"+token(user)+"@", 1)
	f.sh(name, `git clone -q `+url+` /tmp/w && cd /tmp/w && git config user.name `+user+` && git config user.email `+user+`@run`)
	return name
}

// sh runs a script in a container and fails the test if it fails.
func (f *fixture) sh(container, script string) string {
	f.t.Helper()
	out, err := f.try(container, script)
	if err != nil {
		f.t.Fatalf("%s: %v\n%s", container, err, out)
	}
	return out
}

func (f *fixture) try(container, script string) (string, error) {
	var out bytes.Buffer
	err := f.engine.run(f.ctx, strings.NewReader("set -e\n"+script+"\n"), &out, "exec", "-i", container, "sh", "-s")
	return out.String(), err
}

// refused runs a push script that must fail with want in its output.
func (f *fixture) refused(container, script, want string) {
	f.t.Helper()
	var out bytes.Buffer
	err := f.engine.run(f.ctx, strings.NewReader("cd /tmp/w\n"+script+" 2>&1\n"), &out, "exec", "-i", container, "sh", "-s")
	if err == nil || !strings.Contains(err.Error()+out.String(), want) {
		f.t.Fatalf("%s: %q was not refused with %q: %v\n%s", container, script, want, err, out.String())
	}
}

// trailers is a commit message paragraph for a single-quoted -m argument.
func trailers(t runremote.Trailers) string { return t.String() }

// The Run container starts, serves the scripted Managers over the Run's
// network, refuses their attempts to get around the hooks, keeps its checks
// away from its own state, exports the result and is removed without a trace.
func TestTheRunContainerHoldsItsGuaranteesAndLeavesNothing(t *testing.T) {
	f := newFixture(t)
	repo, base := f.source()
	r, err := f.start(f.plan(base), repo, base)
	if err != nil {
		t.Fatal(err)
	}
	hookSum := func() string {
		return f.sh(r.Container, `sha256sum /srv/markitect/run.git/hooks/pre-receive /srv/markitect/plan.json /opt/markitect/markitect-remote`)
	}
	before := hookSum()

	m := f.manager(r, "members-maker")
	mc := f.manager(r, "members-checker")
	root := f.manager(r, "root-maker")
	rc := f.manager(r, "root-checker")

	// Attempts to get around the hooks.
	f.sh(m, `cd /tmp/w && git checkout -q -b w origin/managers/members && echo 'package members // v2' > src/members/member.go && git commit -qam work`)
	f.refused(m, `git push origin HEAD:refs/heads/root`, "is not your branch")
	f.refused(m, `git push origin HEAD:refs/heads/base`, "is not your branch")
	f.refused(m, `git push origin HEAD:refs/heads/managers/new`, "is not your branch")
	f.refused(m, `echo x > README.md && git commit -qam readme && git push origin HEAD:refs/heads/managers/members`, "outside your area")
	f.sh(m, `cd /tmp/w && git reset -q --hard HEAD~1 && git push -q origin HEAD:refs/heads/managers/members`)
	f.refused(m, `git reset -q --hard HEAD~1 && echo other > src/members/member.go && git commit -qam other && git push --force origin HEAD:refs/heads/managers/members`, "never rewritten")
	f.refused(m, `git push origin :refs/heads/managers/members`, "deleting")
	f.refused(m, `GIT_TERMINAL_PROMPT=0 git ls-remote http://remote:8080/run.git`, "fatal:")
	f.refused(m, `git ls-remote http://members-maker:wrong@remote:8080/run.git`, "Authentication failed")

	// The maker submits, the checker approves.
	f.sh(m, `cd /tmp/w && git fetch -q && git reset -q --hard origin/managers/members && git commit -q --allow-empty -m submit -m '`+
		trailers(runremote.Trailers{Manager: "members", Verdicts: []runremote.Line{{Item: "members/member", Word: runremote.Changed, Evidence: "v2"}}})+`' && git push -q origin HEAD:refs/heads/managers/members`)
	f.sh(mc, `cd /tmp/w && git fetch -q && git checkout -q -b w origin/managers/members && git commit -q --allow-empty -m approve -m '`+
		trailers(runremote.Trailers{Manager: "members", Review: runremote.Approved, Checks: []runremote.Line{{Item: "members/member", Word: runremote.Confirmed, Evidence: "read"}}})+`' && git push -q origin HEAD:refs/heads/managers/members`)

	// Root's check runs code Root wrote: it tries to reach the remote's state.
	attack := `echo "uid=$(id -u)"
cat /srv/markitect/plan.json >/dev/null 2>&1 && echo "ATTACK read the plan"
ls /srv/markitect >/dev/null 2>&1 && echo "ATTACK listed the remote"
echo x >> /srv/markitect/run.git/hooks/pre-receive 2>/dev/null && echo "ATTACK wrote the hook"
echo x >> /opt/markitect/markitect-remote 2>/dev/null && echo "ATTACK wrote the program"
echo x >> /srv/markitect/log.jsonl 2>/dev/null && echo "ATTACK wrote the log"
me=$(id -u)
for p in /proc/[0-9]*; do
  [ "$(stat -c %u "$p" 2>/dev/null)" = "$me" ] && continue
  kill -9 "${p#/proc/}" 2>/dev/null && echo "ATTACK killed ${p#/proc/}"
done
echo 1000 > /proc/self/oom_score_adj 2>/dev/null; echo 0 > /proc/self/oom_score_adj 2>/dev/null && echo "ATTACK left the memory limit's victims"
setsid sleep 600 >/dev/null 2>&1 < /dev/null &
for d in /tmp /var/tmp /dev/shm /run/lock; do echo planted > "$d/t3-marker" 2>/dev/null; done
command -v ipcmk >/dev/null && ipcmk -M 1024 >/dev/null
git ls-remote http://remote:8080/run.git >/dev/null 2>&1 && echo "ATTACK read the repository"
exit 0
`
	// The next check, of another user, must find nothing of the first.
	probe := `echo "uid=$(id -u)"
for d in /tmp /var/tmp /dev/shm /run/lock; do [ -e "$d/t3-marker" ] && echo "LEFT $d/t3-marker"; done
ipcs -m 2>/dev/null | grep -q "^0x" && echo "LEFT shared memory"
exit 0
`
	f.sh(root, `cd /tmp/w && git checkout -q -b w origin/root && git merge -q --no-edit origin/managers/members && cat > ci.sh <<'EOF'
`+attack+`EOF
git add ci.sh && git commit -q -m attack -m '`+trailers(runremote.Trailers{Manager: ".", Verdicts: []runremote.Line{{Item: "project", Word: runremote.Changed, Evidence: "merged members"}}})+`' && git push -q origin HEAD:refs/heads/root`)

	lastCheck := func() string {
		log, err := r.Log(f.ctx)
		if err != nil {
			t.Fatal(err)
		}
		var last struct {
			Accepted bool
			Checks   []runremote.CheckResult
		}
		lines := bytes.Split(bytes.TrimSpace(log), []byte("\n"))
		if err := json.Unmarshal(lines[len(lines)-1], &last); err != nil {
			t.Fatal(err)
		}
		if !last.Accepted || len(last.Checks) != 1 || !strings.Contains(last.Checks[0].Output, "uid=61") {
			t.Fatalf("the check did not run as a check user: %+v", last)
		}
		return last.Checks[0].Output
	}
	if out := lastCheck(); strings.Contains(out, "ATTACK") {
		t.Fatalf("the check reached the remote:\n%s", out)
	}
	if after := hookSum(); after != before {
		t.Fatalf("the remote's files changed:\n%s\n%s", before, after)
	}
	if left := f.sh(r.Container, `for p in /proc/[0-9]*; do u=$(stat -c %u $p 2>/dev/null) || continue; [ "$u" -ge 61000 ] && echo "$p $u"; done; true`); strings.TrimSpace(left) != "" {
		t.Fatalf("check processes outlived the check:\n%s", left)
	}
	f.sh(root, `cd /tmp/w && cat > ci.sh <<'EOF'
`+probe+`EOF
git add ci.sh && git commit -q -m probe -m '`+trailers(runremote.Trailers{Manager: ".", Verdicts: []runremote.Line{{Item: "project", Word: runremote.Changed, Evidence: "probe"}}})+`' && git push -q origin HEAD:refs/heads/root`)
	if out := lastCheck(); strings.Contains(out, "LEFT") {
		t.Fatalf("a check found what an earlier one left:\n%s", out)
	}
	f.sh(rc, `cd /tmp/w && git fetch -q && git checkout -q -b w origin/root && git commit -q --allow-empty -m approve -m '`+
		trailers(runremote.Trailers{Manager: ".", Review: runremote.Approved, Checks: []runremote.Line{{Item: "project", Word: runremote.Confirmed, Evidence: "read"}}})+`' && git push -q origin HEAD:refs/heads/root`)
	approved := strings.TrimSpace(f.sh(rc, `cd /tmp/w && git rev-parse HEAD`))

	// The result comes out as a bundle.
	bundle := filepath.Join(t.TempDir(), "result.bundle")
	out, err := os.Create(bundle)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Export(f.ctx, out); err != nil {
		t.Fatal(err)
	}
	out.Close()
	heads, err := exec.Command("git", "bundle", "list-heads", bundle).CombinedOutput()
	if err != nil || !strings.Contains(string(heads), approved+" refs/heads/root") {
		t.Fatalf("exported heads lack Root's approval %s: %v\n%s", approved, err, heads)
	}

	// The host checks a merged state only it holds before landing.
	hostGit := func(args ...string) string {
		out, err := exec.Command("git", append([]string{"-C", repo, "-c", "user.name=Host", "-c", "user.email=host@example"}, args...)...).CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v\n%s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	hostGit("fetch", "-q", bundle, "refs/heads/root:refs/heads/landed")
	merged := hostGit("commit-tree", "-p", "landed", "-m", "merged with the moved target", "landed^{tree}")
	hostGit("update-ref", "refs/heads/merged", merged)
	var thin bytes.Buffer
	cmd := exec.Command("git", "-C", repo, "bundle", "create", "-q", "-", "merged", "--not", approved)
	cmd.Stdout = &thin
	if err := cmd.Run(); err != nil {
		t.Fatalf("bundle: %v", err)
	}
	results, err := r.Check(f.ctx, &thin, merged)
	if err != nil || len(results) != 1 || !results[0].Passed {
		t.Fatalf("landing check: %+v %v", results, err)
	}
	if again := f.sh(r.Container, `git --git-dir=/srv/markitect/run.git for-each-ref --format='%(objectname)' refs/heads/root`); strings.TrimSpace(again) != approved {
		t.Fatalf("the landing check moved root to %s", again)
	}

	// Removal leaves nothing of the Run.
	if err := Sweep(f.ctx, f.engine, f.run); err != nil {
		t.Fatal(err)
	}
}

func TestAFailedStartLeavesNothing(t *testing.T) {
	f := newFixture(t)
	repo, base := f.source()
	plan := f.plan(strings.Repeat("1", 40))
	if _, err := f.start(plan, repo, base); err == nil || !strings.Contains(err.Error(), "does not hold the base commit") {
		t.Fatalf("start with a missing base: %v", err)
	}
	if err := f.engine.gone(f.ctx, f.run); err != nil {
		t.Fatal(err)
	}
	// Close of a healthy remote removes it and its network.
	r, err := f.start(f.plan(base), repo, base)
	if err != nil {
		t.Fatal(err)
	}
	if err := r.Close(f.ctx); err != nil {
		t.Fatal(err)
	}
}
