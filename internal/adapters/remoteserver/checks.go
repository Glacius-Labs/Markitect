package remoteserver

import (
	"archive/tar"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

const (
	// checkUIDs are the users the checks run as: each check run takes the
	// next one, and everything it leaves is removed before the next starts.
	checkUIDBase  = 61000
	checkUIDCount = 2000
	// outputTail is how much of a check's output the pusher and the log see.
	outputTail = 16 << 10
)

// checkRunner runs the required checks on a copy of a commit's tree. When the
// remote runs as root on Linux, as it does in its container, every check runs
// as an unprivileged user of its own that can neither read the plan nor write
// the repository; afterwards every process, file and IPC object of that user
// is removed. Elsewhere (tests) checks run as the current user.
type checkRunner struct {
	// progress tells the pusher which check runs.
	progress io.Writer
	layout   Layout
	repo     gitRepo
	dir      string
	isolate  bool
}

func newCheckRunner(l Layout, repo gitRepo, progress io.Writer) *checkRunner {
	dir := filepath.Join(l.Tmp(), "checks")
	isolate := isolated()
	if isolate {
		// Outside the root-only remote directory, so a check user can reach its own copy.
		dir = filepath.Join(filepath.Dir(l.Root), "checks")
	}
	return &checkRunner{progress: progress, layout: l, repo: repo, dir: dir, isolate: isolate}
}

func (c *checkRunner) Run(commit runremote.Commit, checks []project.Check) (results []runremote.CheckResult, err error) {
	if err := os.MkdirAll(c.dir, 0o711); err != nil {
		return nil, err
	}
	// The host's landing check runs beside the hooks; check runs take turns.
	unlock, err := lockChecks(c.layout)
	if err != nil {
		return nil, err
	}
	defer unlock()
	image, err := readEnvironment(c.layout)
	if err != nil {
		return nil, err
	}
	uid, err := c.nextUID()
	if err != nil {
		return nil, err
	}
	if c.isolate {
		// Checks inherit this: the first victims of the memory limit.
		if err := victim(); err != nil {
			return nil, err
		}
	}
	work, err := os.MkdirTemp(c.dir, "check-")
	if err != nil {
		return nil, err
	}
	defer func() {
		if serr := sweep(c.layout, uid, c.isolate); serr != nil && err == nil {
			results, err = nil, serr
		}
		if rerr := os.RemoveAll(work); rerr != nil && err == nil {
			results, err = nil, rerr
		}
	}()
	repo, home, tmp := filepath.Join(work, "repo"), filepath.Join(work, "home"), filepath.Join(work, "tmp")
	for _, d := range []string{repo, home, tmp} {
		if err := os.Mkdir(d, 0o700); err != nil {
			return nil, err
		}
	}
	if err := c.copyTree(commit, repo, home); err != nil {
		return nil, fmt.Errorf("copy of %s: %w", commit.ID, err)
	}
	if c.isolate {
		if err := chownTree(work, uid); err != nil {
			return nil, err
		}
	}
	env := checkEnv(image, home, tmp)
	for _, check := range checks {
		fmt.Fprintf(c.progress, "running the required check %s on %.12s\n", check.Name, commit.ID)
		r := c.runOne(check, filepath.Join(repo, filepath.FromSlash(check.Directory)), env, uid)
		results = append(results, r)
		if !r.Passed {
			break
		}
	}
	return results, nil
}

func (c *checkRunner) nextUID() (int, error) {
	seq, file := 0, filepath.Join(c.layout.Root, "check-seq")
	if b, err := os.ReadFile(file); err == nil {
		seq, _ = strconv.Atoi(strings.TrimSpace(string(b)))
	} else if !errors.Is(err, os.ErrNotExist) {
		return 0, err
	}
	if err := os.WriteFile(file, []byte(strconv.Itoa(seq+1)), 0o600); err != nil {
		return 0, err
	}
	return checkUIDBase + seq%checkUIDCount, nil
}

// copyTree writes the commit's tree into dir and makes dir a Git repository
// whose one commit has exactly that tree, so a check sees the branch's full
// state as a clean checkout. Git's archive applies no attribute here: the
// remote's info/attributes switches conversions off.
func (c *checkRunner) copyTree(commit runremote.Commit, dir, home string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	archive := exec.CommandContext(ctx, c.repo.git, "--git-dir="+c.repo.dir, "archive", "--format=tar", commit.ID)
	archive.Env = c.repo.env
	var stderr bytes.Buffer
	archive.Stderr = &stderr
	out, err := archive.StdoutPipe()
	if err != nil {
		return err
	}
	if err := archive.Start(); err != nil {
		return err
	}
	executables, xerr := extract(out, dir)
	_, _ = io.Copy(io.Discard, out)
	if err := archive.Wait(); err != nil {
		return fmt.Errorf("git archive: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	if xerr != nil {
		return xerr
	}
	g := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, c.repo.git, append([]string{"-c", "core.autocrlf=false", "-c", "core.safecrlf=false",
			"-c", "core.hooksPath=" + os.DevNull, "-c", "safe.directory=*", "-C", dir}, args...)...)
		cmd.Env = append(gitConfigEnv(), "PATH="+os.Getenv("PATH"), "HOME="+home,
			"GIT_AUTHOR_NAME=Markitect", "GIT_AUTHOR_EMAIL=markitect@run", "GIT_COMMITTER_NAME=Markitect", "GIT_COMMITTER_EMAIL=markitect@run")
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := g("init", "-q"); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(dir, ".git", "info", "attributes"), []byte(neutralAttributes), 0o644); err != nil {
		return err
	}
	if _, err := g("add", "-A", "-f"); err != nil {
		return err
	}
	for len(executables) > 0 {
		n := min(len(executables), 200)
		if _, err := g(append([]string{"update-index", "--chmod=+x", "--"}, executables[:n]...)...); err != nil {
			return err
		}
		executables = executables[n:]
	}
	if _, err := g("commit", "-q", "--allow-empty", "--no-verify", "-m", "Markitect check of "+commit.ID); err != nil {
		return err
	}
	tree, err := g("rev-parse", "HEAD^{tree}")
	if err != nil {
		return err
	}
	if tree != commit.Tree {
		return fmt.Errorf("the copy's tree %s differs from the commit's tree %s; submodules and other entries a checkout cannot hold are not supported in a Run", tree, commit.Tree)
	}
	return nil
}

// neutralAttributes switches off every attribute that would let a file's
// checked-out bytes differ from its blob.
const neutralAttributes = "* -text -eol -crlf -ident -filter -working-tree-encoding -export-ignore -export-subst\n"

// extract writes a Git archive into dir through os.Root, so no entry can
// write outside dir, and returns the paths of executable files.
func extract(r io.Reader, dir string) ([]string, error) {
	root, err := os.OpenRoot(dir)
	if err != nil {
		return nil, err
	}
	defer root.Close()
	var executables []string
	tr := tar.NewReader(r)
	for {
		h, err := tr.Next()
		if errors.Is(err, io.EOF) {
			return executables, nil
		}
		if err != nil {
			return nil, err
		}
		name := path.Clean(h.Name)
		if !filepath.IsLocal(filepath.FromSlash(name)) {
			return nil, fmt.Errorf("archive entry %q leaves the copy", h.Name)
		}
		switch h.Typeflag {
		case tar.TypeXGlobalHeader:
		case tar.TypeDir:
			if err := root.MkdirAll(name, 0o755); err != nil {
				return nil, err
			}
		case tar.TypeReg:
			mode := os.FileMode(0o644)
			if h.Mode&0o100 != 0 {
				mode = 0o755
				executables = append(executables, name)
			}
			f, err := root.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_EXCL, mode)
			if err != nil {
				return nil, err
			}
			_, err = io.Copy(f, tr)
			if cerr := f.Close(); err == nil {
				err = cerr
			}
			if err != nil {
				return nil, err
			}
		case tar.TypeSymlink:
			if err := root.Symlink(h.Linkname, name); err != nil {
				return nil, err
			}
		default:
			return nil, fmt.Errorf("archive entry %q has unsupported type %q", h.Name, h.Typeflag)
		}
	}
}

// environmentName holds the image's environment, recorded when the remote is
// seeded: the environment a check runs in.
const environmentName = "environment.json"

func recordEnvironment(l Layout) error {
	b, err := json.Marshal(os.Environ())
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(l.Root, environmentName), b, 0o600)
}

func readEnvironment(l Layout) ([]string, error) {
	b, err := os.ReadFile(filepath.Join(l.Root, environmentName))
	if err != nil {
		return nil, err
	}
	var env []string
	return env, json.Unmarshal(b, &env)
}

// checkEnv is the image's environment with a fresh HOME and TMPDIR; every
// other variable that points into the image's home, PATH excepted, points
// into the fresh one, so no cache is shared between check runs.
func checkEnv(image []string, home, tmp string) []string {
	old := ""
	for _, kv := range image {
		if v, ok := strings.CutPrefix(kv, "HOME="); ok && v != "/" {
			old = strings.TrimSuffix(v, "/")
		}
	}
	var env []string
	for _, kv := range image {
		k, v, _ := strings.Cut(kv, "=")
		switch {
		case k == "HOME" || k == "TMPDIR" || k == "PWD" || k == "OLDPWD" || k == "REMOTE_USER",
			strings.HasPrefix(k, "GIT_"), strings.HasPrefix(k, "MARKITECT_"):
			continue
		case old != "" && k != "PATH" && (v == old || strings.HasPrefix(v, old+"/")):
			v = home + v[len(old):]
		}
		env = append(env, k+"="+v)
	}
	return append(env, "HOME="+home, "TMPDIR="+tmp)
}

func (c *checkRunner) runOne(check project.Check, dir string, env []string, uid int) runremote.CheckResult {
	r := runremote.CheckResult{Name: check.Name, Exit: -1}
	ctx, cancel := context.WithTimeout(context.Background(), check.Timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, shell(), "-c", check.Command)
	cmd.Dir = dir
	cmd.Env = env
	out := &tailBuffer{limit: outputTail}
	cmd.Stdout, cmd.Stderr = out, out
	cmd.WaitDelay = 10 * time.Second
	prepare(cmd, uid, c.isolate)
	start := time.Now()
	err := cmd.Start()
	if err == nil {
		err = cmd.Wait()
		finished(cmd)
	}
	r.Seconds = time.Since(start).Round(10 * time.Millisecond).Seconds()
	if cmd.ProcessState != nil {
		r.Exit = cmd.ProcessState.ExitCode()
	}
	r.TimedOut = errors.Is(ctx.Err(), context.DeadlineExceeded)
	r.Passed = r.Exit == 0 && !r.TimedOut && (err == nil || errors.Is(err, exec.ErrWaitDelay))
	if err != nil && !r.Passed && cmd.ProcessState == nil {
		fmt.Fprintf(out, "\n[the check did not start: %v]\n", err)
	}
	r.Output = out.String()
	return r
}

func shell() string {
	if runtime.GOOS == "windows" {
		if p, err := exec.LookPath("sh"); err == nil {
			return p
		}
	}
	return "/bin/sh"
}

// tailBuffer keeps the last limit bytes written to it.
type tailBuffer struct {
	limit int
	buf   []byte
	cut   bool
}

func (t *tailBuffer) Write(p []byte) (int, error) {
	t.buf = append(t.buf, p...)
	if len(t.buf) > t.limit {
		t.buf = append([]byte(nil), t.buf[len(t.buf)-t.limit:]...)
		t.cut = true
	}
	return len(p), nil
}

func (t *tailBuffer) String() string {
	if t.cut {
		return "[…]" + string(t.buf)
	}
	return string(t.buf)
}
