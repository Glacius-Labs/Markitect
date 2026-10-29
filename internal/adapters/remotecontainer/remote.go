package remotecontainer

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
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

const (
	// RunLabel marks every container and network of a Run with its ID.
	RunLabel = "markitect.run"
	// Alias is the remote's host name on the Run's network.
	Alias = "remote"
	// program is where the remote's program lives in its container.
	program = "/opt/markitect/markitect-remote"
	root    = "/srv/markitect"
	port    = 8080
)

// Limits bound the remote container, its required checks included.
type Limits struct {
	CPUs   string // docker --cpus, for example "2"
	Memory string // docker --memory, for example "4g"
	Pids   int
}

// Spec is one Run's remote.
type Spec struct {
	// Image is the remote's image when the plan has no required checks; with
	// checks it is the image they name. It holds Git, a POSIX shell and
	// coreutils besides what the checks need.
	Image string
	// Binary is markitect-remote built for Linux on the engine's architecture.
	Binary string
	Plan   *runremote.Plan
	Limits Limits
}

// Remote is a started Run remote.
type Remote struct {
	// Network is the Run's internal network; Manager containers join it.
	Network   string
	Container string
	// URL is the repository's address on the Run's network.
	URL    string
	engine *Engine
	run    string
}

var runID = regexp.MustCompile(`^[a-z0-9][a-z0-9_.-]{0,62}$`)

// Start creates the Run's internal network and its remote container, puts the
// program and the plan in, starts it, seeds the bare repository from bundle
// (which must hold the plan's base commit) and waits until it serves. On
// failure it removes everything it created.
func Start(ctx context.Context, e *Engine, spec Spec, bundle io.Reader) (r *Remote, err error) {
	if err := spec.Plan.Validate(); err != nil {
		return nil, err
	}
	run := spec.Plan.Run
	if !runID.MatchString(run) {
		return nil, fmt.Errorf("run %q: a container name needs lower-case letters, digits, '.', '_' or '-'", run)
	}
	image := spec.Plan.Image()
	switch {
	case image == "" && spec.Image == "":
		return nil, errors.New("the plan has no required checks, so the remote needs an image")
	case image == "":
		image = spec.Image
	case spec.Image != "" && spec.Image != image:
		return nil, fmt.Errorf("the required checks run in %q, not in %q", image, spec.Image)
	}
	files, err := payload(spec)
	if err != nil {
		return nil, err
	}
	r = &Remote{Network: "markitect-run-" + run, Container: "markitect-run-" + run + "-remote",
		URL: fmt.Sprintf("http://%s:%d/run.git", Alias, port), engine: e, run: run}
	if left, err := e.labelled(ctx, run); err != nil {
		return nil, err
	} else if len(left) > 0 {
		return nil, fmt.Errorf("run %q already has containers or networks %v; remove them with Sweep first", run, left)
	}
	defer func() {
		if err != nil {
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
			defer cancel()
			err = errors.Join(err, Sweep(cleanup, e, run))
		}
	}()
	label := RunLabel + "=" + run
	// Internal alone still gives the bridge a gateway address on the
	// engine's host; isolated gateway mode removes it, so no container of
	// the Run reaches the host.
	if err := e.run(ctx, nil, nil, "network", "create", "--internal", "--opt", "com.docker.network.bridge.gateway_mode_ipv4=isolated", "--label", label, r.Network); err != nil {
		return nil, err
	}
	args := []string{"create", "--name", r.Container, "--label", label, "--label", "markitect.role=remote",
		"--network", r.Network, "--network-alias", Alias,
		"--user", "0:0", "--init", "--security-opt", "no-new-privileges", "--cap-drop", "ALL"}
	for _, c := range []string{"CHOWN", "DAC_OVERRIDE", "FOWNER", "KILL", "SETUID", "SETGID", "SYS_RESOURCE"} {
		args = append(args, "--cap-add", c)
	}
	if spec.Limits.CPUs != "" {
		args = append(args, "--cpus", spec.Limits.CPUs)
	}
	if spec.Limits.Memory != "" {
		args = append(args, "--memory", spec.Limits.Memory)
	}
	if spec.Limits.Pids > 0 {
		args = append(args, "--pids-limit", strconv.Itoa(spec.Limits.Pids))
	}
	args = append(args, "--entrypoint", program, image, "serve", "--root", root)
	if err := e.run(ctx, nil, nil, args...); err != nil {
		return nil, err
	}
	if err := e.run(ctx, bytes.NewReader(files), nil, "cp", "-", r.Container+":/"); err != nil {
		return nil, err
	}
	if err := e.run(ctx, nil, nil, "start", r.Container); err != nil {
		return nil, err
	}
	if err := e.run(ctx, bundle, nil, "exec", "-i", r.Container, program, "seed", "--root", root); err != nil {
		return nil, err
	}
	if err := e.run(ctx, nil, nil, "exec", r.Container, program, "ready", "--timeout", "60s"); err != nil {
		return nil, err
	}
	return r, nil
}

// payload is the tar stream the host copies into the created container: the
// program, executable by everyone, and the plan, readable by root only.
func payload(spec Spec) ([]byte, error) {
	bin, err := os.ReadFile(spec.Binary)
	if err != nil {
		return nil, err
	}
	plan, err := json.Marshal(spec.Plan)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	tw := tar.NewWriter(&buf)
	entries := []struct {
		name string
		mode int64
		data []byte
	}{
		{"opt/markitect/", 0o755, nil},
		{"opt/markitect/markitect-remote", 0o755, bin},
		{"srv/markitect/", 0o700, nil},
		{"srv/markitect/plan.json", 0o600, plan},
	}
	for _, f := range entries {
		h := &tar.Header{Name: f.name, Mode: f.mode, Size: int64(len(f.data)), Typeflag: tar.TypeReg, ModTime: time.Unix(0, 0)}
		if strings.HasSuffix(f.name, "/") {
			h.Typeflag, h.Size = tar.TypeDir, 0
		}
		if err := tw.WriteHeader(h); err != nil {
			return nil, err
		}
		if _, err := tw.Write(f.data); err != nil {
			return nil, err
		}
	}
	if err := tw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Export writes a bundle of every branch of the remote.
func (r *Remote) Export(ctx context.Context, w io.Writer) error {
	return r.engine.run(ctx, nil, w, "exec", r.Container, program, "export", "--root", root)
}

// Check runs the required checks on a commit the host holds, such as the
// merge of a moved target branch before landing, isolated as the hook runs
// them and without changing any branch of the Run. The bundle holds the
// commit and may be thin against the Run's branches. A failing check is a
// result; an error means the checks could not run.
func (r *Remote) Check(ctx context.Context, bundle io.Reader, commit string) ([]runremote.CheckResult, error) {
	var out bytes.Buffer
	if err := r.engine.run(ctx, bundle, &out, "exec", "-i", r.Container, program, "check", "--root", root, "--commit", commit); err != nil {
		return nil, err
	}
	var results []runremote.CheckResult
	if err := json.Unmarshal(out.Bytes(), &results); err != nil {
		return nil, fmt.Errorf("the remote's check results: %w", err)
	}
	return results, nil
}

// StartFrom moves a Manager's branch, still at the base, to a state of its
// parent's branch before the Manager starts.
func (r *Remote) StartFrom(ctx context.Context, branch, commit string) error {
	return r.engine.run(ctx, nil, nil, "exec", r.Container, program, "start", "--root", root, "--branch", branch, "--commit", commit)
}

// Log returns the remote's decision log, one JSON entry per push.
func (r *Remote) Log(ctx context.Context) ([]byte, error) {
	var out bytes.Buffer
	err := r.engine.run(ctx, nil, &out, "exec", r.Container, program, "log", "--root", root)
	return out.Bytes(), err
}

// Close removes the remote container with its volumes and the Run's
// network, as far as they still exist. The Run's Manager containers must be
// gone first; Sweep removes everything of a Run at once.
func (r *Remote) Close(ctx context.Context) error {
	label := "label=" + RunLabel + "=" + r.run
	if ids, err := r.engine.output(ctx, "ps", "--all", "--quiet", "--filter", label, "--filter", "name=^"+r.Container+"$"); err != nil {
		return err
	} else if ids != "" {
		if err := r.engine.run(ctx, nil, nil, "rm", "--force", "--volumes", ids); err != nil {
			return err
		}
	}
	if nets, err := r.engine.output(ctx, "network", "ls", "--quiet", "--filter", label, "--filter", "name=^"+r.Network+"$"); err != nil {
		return err
	} else if nets != "" {
		if err := r.engine.run(ctx, nil, nil, "network", "rm", nets); err != nil {
			return err
		}
	}
	return r.engine.gone(ctx, r.run)
}

// Sweep removes every container and network that carries the Run's label,
// also those of a host that stopped without cleaning up, and verifies that
// none is left.
func Sweep(ctx context.Context, e *Engine, run string) error {
	label := "label=" + RunLabel + "=" + run
	ids, err := e.output(ctx, "ps", "--all", "--quiet", "--filter", label)
	if err != nil {
		return err
	}
	var errs []error
	if f := strings.Fields(ids); len(f) > 0 {
		errs = append(errs, e.run(ctx, nil, nil, append([]string{"rm", "--force", "--volumes"}, f...)...))
	}
	nets, err := e.output(ctx, "network", "ls", "--quiet", "--filter", label)
	if err != nil {
		return errors.Join(append(errs, err)...)
	}
	if f := strings.Fields(nets); len(f) > 0 {
		errs = append(errs, e.run(ctx, nil, nil, append([]string{"network", "rm"}, f...)...))
	}
	if err := errors.Join(errs...); err != nil {
		return err
	}
	return e.gone(ctx, run)
}

func (e *Engine) labelled(ctx context.Context, run string) ([]string, error) {
	label := "label=" + RunLabel + "=" + run
	ids, err := e.output(ctx, "ps", "--all", "--quiet", "--filter", label)
	if err != nil {
		return nil, err
	}
	nets, err := e.output(ctx, "network", "ls", "--quiet", "--filter", label)
	if err != nil {
		return nil, err
	}
	return append(strings.Fields(ids), strings.Fields(nets)...), nil
}

func (e *Engine) gone(ctx context.Context, run string) error {
	left, err := e.labelled(ctx, run)
	if err != nil {
		return err
	}
	if len(left) > 0 {
		return fmt.Errorf("run %q still has containers or networks %v", run, left)
	}
	return nil
}

// StartBundle writes a bundle that holds commit of the repository at
// repoDir, for Start. It leaves the repository untouched: the bundle comes
// from a scratch repository that borrows the objects.
func StartBundle(ctx context.Context, git, repoDir, commit string, w io.Writer) error {
	g := func(dir string, stdout io.Writer, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, git, append([]string{"-C", dir}, args...)...)
		for _, kv := range os.Environ() {
			if !strings.HasPrefix(strings.ToUpper(kv), "GIT_") {
				cmd.Env = append(cmd.Env, kv)
			}
		}
		cmd.Env = append(cmd.Env, "GIT_CONFIG_NOSYSTEM=1", "GIT_CONFIG_GLOBAL="+os.DevNull, "GIT_TERMINAL_PROMPT=0")
		var out, stderr bytes.Buffer
		if stdout == nil {
			stdout = &out
		}
		cmd.Stdout, cmd.Stderr = stdout, &stderr
		if err := cmd.Run(); err != nil {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(out.String()), nil
	}
	common, err := g(repoDir, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return err
	}
	scratch, err := os.MkdirTemp("", "markitect-start-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(scratch)
	if _, err := g(scratch, nil, "init", "-q", "--bare"); err != nil {
		return err
	}
	alternates := filepath.Join(scratch, "objects", "info", "alternates")
	if err := os.WriteFile(alternates, []byte(filepath.ToSlash(filepath.Join(common, "objects"))+"\n"), 0o644); err != nil {
		return err
	}
	if t, err := g(scratch, nil, "cat-file", "-t", commit); err != nil || t != "commit" {
		return errors.Join(err, fmt.Errorf("%s is no commit of %s", commit, repoDir))
	}
	if _, err := g(scratch, nil, "update-ref", "refs/heads/base", commit); err != nil {
		return err
	}
	_, err = g(scratch, w, "bundle", "create", "-q", "-", "refs/heads/base")
	return err
}
