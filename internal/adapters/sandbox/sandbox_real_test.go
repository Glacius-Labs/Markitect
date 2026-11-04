package sandbox_test

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/adapters/sandbox"
	"github.com/Glacius-Labs/Markitect/internal/agentport"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
)

// imageVariable names the image of the real-engine tests: one built from
// packaging/runtime. Without it they are skipped.
const imageVariable = "MARKITECT_TEST_IMAGE"

const implementer = agentport.ImplementerUser

var reviewer = agentport.ReviewerUser(1)

// realBox starts a Command's container on the real engine, with a repository of
// two files inside, and removes it at the end.
func realBox(t *testing.T) (*sandbox.Box, string, string) {
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
	dir := t.TempDir()
	git := func(args ...string) string {
		cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=T", "-c", "user.email=t@example.invalid", "-c", "core.autocrlf=false"}, args...)...)
		out, err := cmd.CombinedOutput()
		if err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
		return strings.TrimSpace(string(out))
	}
	git("init", "-q", "-b", "main")
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Start\r\nWith a Windows line end.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte("build/\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	git("add", "-A")
	git("commit", "-q", "-m", "start")
	commit := git("rev-parse", "HEAD")
	file := filepath.Join(dir, "start.bundle")
	git("bundle", "create", "-q", file, "main")
	bundle, err := os.Open(file)
	if err != nil {
		t.Fatal(err)
	}
	defer bundle.Close()
	name := "sandbox-test-" + strings.ToLower(strings.ReplaceAll(t.Name(), "/", "-"))
	if len(name) > 60 {
		name = name[:60]
	}
	s := &sandbox.Sandboxes{Engine: engine, Image: image, Limits: sandbox.Limits{CPUs: "2", Memory: "2g", Pids: 512, Tmp: "256m"}}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	t.Cleanup(cancel)
	_ = sandbox.Sweep(ctx, engine, name)
	box, err := s.Start(ctx, name, commit, bundle)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanup, cancel := context.WithTimeout(context.Background(), time.Minute)
		defer cancel()
		if err := box.Close(cleanup); err != nil {
			t.Errorf("the container did not go: %v", err)
		}
	})
	return box.(*sandbox.Box), commit, dir
}

// as runs a shell line as one of the container's users.
func as(t *testing.T, box *sandbox.Box, user, line string) (string, error) {
	t.Helper()
	var out lockedBuffer
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	err := box.Exec(ctx, agentport.Exec{User: user, Dir: "/", Argv: []string{"/bin/sh", "-c", line}, Stdout: &out, Stderr: &out})
	return out.String(), err
}

// countSleeps counts this user's sleep processes from /proc, since the image has
// no process tools.
const countSleeps = `n=0; me=$(id -u); for d in /proc/[0-9]*; do if [ "$(stat -c %u "$d" 2>/dev/null)" = "$me" ] && [ "$(head -c 5 "$d/cmdline" 2>/dev/null)" = sleep ]; then n=$((n+1)); fi; done; echo $n`

// lockedBuffer takes a command's two output streams, which arrive side by side.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// A folder comes out as a commit with what Git takes and nothing else, byte for
// byte, and each role's copy is its own and out of the others' reach.
func TestAFolderComesOutAsACommitByteForByte(t *testing.T) {
	box, start, _ := realBox(t)
	ctx := t.Context()
	repo, err := box.Deliver(ctx, implementer, start)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := as(t, box, implementer, `od -c "`+repo+`/README.md" | head -1`); err != nil || !strings.Contains(got, `\r`) {
		t.Fatalf("the copy's line ends were converted: %q, %v", got, err)
	}
	if _, err := as(t, box, implementer, `cd "`+repo+`"; printf 'new\r\n' > new.txt; mkdir -p build; echo left > build/out; printf 'x' > locked; chmod 000 locked; rm .gitignore; printf 'build/\n' > .gitignore; git -c user.name=a -c user.email=a@b.c commit -q --allow-empty -m "a commit of its own"; sleep 300 &`); err != nil {
		t.Fatal(err)
	}
	commit, bundle, err := box.Collect(ctx, start, "the Command's message\n")
	if err != nil {
		t.Fatal(err)
	}
	if commit == start || bundle == nil {
		t.Fatalf("commit %s, bundle %v", commit, bundle)
	}
	bundle.Close()
	if got, _ := as(t, box, implementer, countSleeps); strings.TrimSpace(got) != "0" {
		t.Fatalf("the implementer's processes outlived the commit: %q", got)
	}
	copied, err := box.Deliver(ctx, reviewer, commit)
	if err != nil {
		t.Fatal(err)
	}
	got, err := as(t, box, reviewer, `cd "`+copied+`"; git log -1 --format=%s; git log -1 --format=%an; git rev-parse HEAD; ls -a; od -c new.txt | head -1`)
	if err != nil {
		t.Fatalf("%v: %s", err, got)
	}
	for _, want := range []string{"the Command's message", "Markitect", commit, "new.txt", "locked", `\r`} {
		if !strings.Contains(got, want) {
			t.Fatalf("the reviewer's copy lacks %q:\n%s", want, got)
		}
	}
	if strings.Contains(got, "build") && !strings.Contains(got, ".gitignore") || strings.Contains(got, "\nbuild\n") {
		t.Fatalf("what Git does not take is in the copy:\n%s", got)
	}
	// An unchanged folder hands out no new commit.
	again, bundle, err := box.Collect(ctx, commit, "again\n")
	if err != nil || again != commit || bundle != nil {
		t.Fatalf("an unchanged folder: %s, %v, %v", again, bundle, err)
	}
}

// What one user holds no other can reach, and a required check gets neither the
// token nor anything the sessions left (planted refusals of concept 3.0 §7, §9).
func TestNoUserReachesAnothersFolderOrHome(t *testing.T) {
	box, start, _ := realBox(t)
	ctx := t.Context()
	repo, err := box.Deliver(ctx, implementer, start)
	if err != nil {
		t.Fatal(err)
	}
	copied, err := box.Deliver(ctx, reviewer, start)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := as(t, box, implementer, `echo SECRETVALUE > "$HOME/left.txt"; echo PLANTEDVALUE > /tmp/planted.txt; chmod 600 /tmp/planted.txt`); err != nil {
		t.Fatal(err)
	}
	if got, err := as(t, box, reviewer, `id -u; cat /home/implementer/left.txt || echo refused; ls `+repo+` || echo refused; cat /tmp/planted.txt || echo refused; echo x > /work/tools/x || echo refused`); err != nil || strings.Contains(got, "SECRETVALUE") || strings.Contains(got, "PLANTEDVALUE") || strings.Count(got, "refused") != 4 {
		t.Fatalf("the reviewer reaches what is not its own: %v\n%s", err, got)
	}
	if got, err := as(t, box, implementer, `ls `+copied+` || echo refused; cat /work/tools/repo.git/HEAD || echo refused`); err != nil || strings.Count(got, "refused") != 2 {
		t.Fatalf("the implementer reaches what is not its own: %v\n%s", err, got)
	}
	// Root's own programs belong to no user, and root holds no capability.
	if got, err := as(t, box, implementer, `echo x > /usr/local/bin/markitect || echo refused; touch /etc/x || echo refused`); err != nil || strings.Count(got, "refused") != 2 {
		t.Fatalf("the tools can be changed: %v\n%s", err, got)
	}
	// A process a session left behind does not outlive the start of the checks.
	if _, err := as(t, box, reviewer, `sleep 300 >/dev/null 2>&1 &`); err != nil {
		t.Fatal(err)
	}
	checks := []project.Check{{Name: "probe", Directory: ".", Timeout: time.Minute, Command: `set -u
for f in /home/implementer/left.txt /home/reviewer /work/implementer /work/reviewer /work/tools/repo.git/HEAD; do
  if cat "$f" 2>/dev/null || ls "$f" 2>/dev/null; then echo "REACHED $f"; exit 1; fi
done
if env | grep -q TOKEN; then echo "A TOKEN"; exit 1; fi
test -f README.md
sleep 300 &
echo left > "$HOME/mine.txt"
echo left > /tmp/checks-left.txt
`}}
	results, err := box.Check(ctx, nil, start, checks)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || !results[0].Passed {
		t.Fatalf("the planted check: %+v", results)
	}
	if got, _ := as(t, box, reviewer, countSleeps); strings.TrimSpace(got) != "0" {
		t.Fatalf("a session's process outlived the start of the checks: %q", got)
	}
	// Its processes are ended and its folder is thrown away.
	if got, err := as(t, box, "checks", countSleeps+`; ls /work/checks | wc -l; ls "$HOME"; ls /tmp`); err != nil || !strings.HasPrefix(got, "0\n0\n") || strings.Contains(got, "mine.txt") || strings.Contains(got, "checks-left") {
		t.Fatalf("the checks left something: %v\n%s", err, got)
	}
}

func TestAFailingCheckIsAResultAndAMissingProgramTheEnvironments(t *testing.T) {
	box, start, _ := realBox(t)
	ctx := t.Context()
	results, err := box.Check(ctx, nil, start, []project.Check{{Name: "fails", Directory: ".", Timeout: time.Minute, Command: "echo no; exit 3"}})
	if err != nil || len(results) != 1 || results[0].Passed || results[0].NotRun || results[0].Exit != 3 || !strings.Contains(results[0].Output, "no") {
		t.Fatalf("%+v, %v", results, err)
	}
	results, err = box.Check(ctx, nil, start, []project.Check{{Name: "missing", Directory: ".", Timeout: time.Minute, Command: "no-such-program"}})
	if err != nil || len(results) != 1 || !results[0].NotRun {
		t.Fatalf("%+v, %v", results, err)
	}
	results, err = box.Check(ctx, nil, start, []project.Check{{Name: "slow", Directory: ".", Timeout: time.Second, Command: "sleep 300"}})
	if err != nil || len(results) != 1 || results[0].Passed || !results[0].TimedOut {
		t.Fatalf("%+v, %v", results, err)
	}
	results, err = box.Check(ctx, nil, start, []project.Check{{Name: "nowhere", Directory: "no/such", Timeout: time.Minute, Command: "true"}})
	if err != nil || len(results) != 1 || results[0].Passed || results[0].NotRun {
		t.Fatalf("%+v, %v", results, err)
	}
}

// A secret stays off the engine's command line and reaches only that exec.
func TestASecretReachesOnlyItsExec(t *testing.T) {
	box, _, _ := realBox(t)
	ctx := t.Context()
	var out bytes.Buffer
	err := box.Exec(ctx, agentport.Exec{User: implementer, Dir: "/", Argv: []string{"/bin/sh", "-c", `echo "$MARKITECT_TEST_SECRET"; echo "$HOME"; test -d "$TMPDIR"`},
		Secrets: []string{"MARKITECT_TEST_SECRET=sk-planted"}, Stdout: &out})
	if err != nil || !strings.Contains(out.String(), "sk-planted") || !strings.Contains(out.String(), "/home/implementer") {
		t.Fatalf("%v: %q", err, out.String())
	}
	if got, _ := as(t, box, reviewer, `echo "[$MARKITECT_TEST_SECRET]"`); !strings.Contains(got, "[]") {
		t.Fatalf("the secret outlived its exec: %q", got)
	}
}

func TestNotes(t *testing.T) {
	box, _, _ := realBox(t)
	ctx := t.Context()
	if _, ok, err := box.Note(ctx, reviewer, "answers.md"); err != nil || ok {
		t.Fatalf("no note yet: %v, %v", ok, err)
	}
	if _, err := as(t, box, reviewer, `mkdir -p "$(dirname "`+box.NotePath(reviewer, "answers.md")+`")"; printf 'an answer\n' > "`+box.NotePath(reviewer, "answers.md")+`"`); err != nil {
		t.Fatal(err)
	}
	if got, ok, err := box.Note(ctx, reviewer, "answers.md"); err != nil || !ok || string(got) != "an answer\n" {
		t.Fatalf("%q, %v, %v", got, ok, err)
	}
	if _, ok, _ := box.Note(ctx, implementer, "answers.md"); ok {
		t.Fatal("a note is another role's")
	}
	if _, err := as(t, box, reviewer, `rm "`+box.NotePath(reviewer, "answers.md")+`"`); err != nil {
		t.Fatal(err)
	}
	// A note that is a link is no note: it could point anywhere.
	if _, err := as(t, box, reviewer, `ln -s /etc/passwd "`+box.NotePath(reviewer, "answers.md")+`"`); err != nil {
		t.Fatal(err)
	}
	if _, ok, _ := box.Note(ctx, reviewer, "answers.md"); ok {
		t.Fatal("a link was read as a note")
	}
}

// When ctx ends, the user's processes end with it.
func TestACancelledExecEndsItsUsersProcesses(t *testing.T) {
	box, _, _ := realBox(t)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := box.Exec(ctx, agentport.Exec{User: implementer, Dir: "/", Argv: []string{"/bin/sh", "-c", "sleep 300 & sleep 300"}})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("the exec returned %v", err)
	}
	if got, _ := as(t, box, implementer, countSleeps); strings.TrimSpace(got) != "0" {
		t.Fatalf("its processes outlived it: %q", got)
	}
}

var _ run.Sandbox = (*sandbox.Box)(nil)

// A process a reviewer left cannot change its answers once it has ended, and a
// fresh review starts with nothing of the last one, its configuration included.
func TestAReviewersLeftoversCannotChangeItsAnswersOrTheNextReview(t *testing.T) {
	box, start, _ := realBox(t)
	ctx := t.Context()
	if _, err := box.Deliver(ctx, reviewer, start); err != nil {
		t.Fatal(err)
	}
	note := box.NotePath(reviewer, "answers.md")
	if _, err := as(t, box, reviewer, `printf 'catalog/book: wrong\n' > "`+note+`"; (while true; do printf 'catalog/book: ok\n' > "`+note+`"; sleep 0.2; done) >/dev/null 2>&1 &`); err != nil {
		t.Fatal(err)
	}
	if _, err := as(t, box, reviewer, `printf 'catalog/book: wrong\n' > "`+note+`"; mkdir -p "$HOME/.claude"; echo '{"hooks":{}}' > "$HOME/.claude/settings.json"; echo left > /dev/shm/reviewer-left; chmod 111 /dev/shm/reviewer-left`); err != nil {
		t.Fatal(err)
	}
	first, ok, err := box.Note(ctx, reviewer, "answers.md")
	if err != nil || !ok {
		t.Fatalf("%v, %v", ok, err)
	}
	time.Sleep(time.Second)
	again, _, err := box.Note(ctx, reviewer, "answers.md")
	if err != nil || string(again) != string(first) {
		t.Fatalf("the note changed after it was read: %q then %q, %v", first, again, err)
	}
	if _, err := box.Deliver(ctx, reviewer, start); err != nil {
		t.Fatal(err)
	}
	if got, err := as(t, box, reviewer, `ls -A "$HOME/.claude" 2>/dev/null | wc -l; ls /dev/shm | wc -l; test ! -e "`+note+`" && echo gone`); err != nil || !strings.HasPrefix(got, "0\n0\n") || !strings.Contains(got, "gone") {
		t.Fatalf("a fresh reviewer found what the last one left: %v\n%s", err, got)
	}
}

// Reviewers side by side reach neither each other's folders nor each other's notes.
func TestReviewersCannotReachEachOther(t *testing.T) {
	box, start, _ := realBox(t)
	ctx := t.Context()
	other := agentport.ReviewerUser(2)
	repo, err := box.Deliver(ctx, reviewer, start)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := box.Deliver(ctx, other, start); err != nil {
		t.Fatal(err)
	}
	if got, err := as(t, box, other, `ls `+repo+` || echo refused; echo x > "`+box.NotePath(reviewer, "answers.md")+`" || echo refused`); err != nil || strings.Count(got, "refused") != 2 {
		t.Fatalf("%v\n%s", err, got)
	}
}

// What the implementer leaves in shared memory is gone before a required check or
// a reviewer runs.
func TestSharedMemoryIsClearedWithTheCommit(t *testing.T) {
	box, start, _ := realBox(t)
	ctx := t.Context()
	repo, err := box.Deliver(ctx, implementer, start)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := as(t, box, implementer, `echo "sk-planted" > /dev/shm/t; mkdir /tmp/locked; chmod 111 /tmp/locked; cd "`+repo+`"; echo work > work.txt`); err != nil {
		t.Fatal(err)
	}
	if _, _, err := box.Collect(ctx, start, "work\n"); err != nil {
		t.Fatal(err)
	}
	if got, err := as(t, box, reviewer, `ls /dev/shm | wc -l; ls /tmp | grep -c locked || true`); err != nil || !strings.HasPrefix(got, "0\n0") {
		t.Fatalf("%v\n%s", err, got)
	}
}
