package container

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ExecSession retains one hardened container. Commands run through engine exec;
// Close removes only its observed full ID. No command is retried.
type ExecSession interface {
	ID() string
	GitExecCount() int
	Exec(context.Context, []string, io.Reader, io.Writer) error
	Close(context.Context) error
}

type execSession struct {
	engine   *Engine
	id       string
	mu       sync.Mutex
	closed   bool
	err      error
	user     string
	gitExecs atomic.Int64
}

// StartExec starts a long-lived command in a container with fixed mounts.
// The caller owns its lifetime and must Close it, including after cancellation.
func (e *Engine) StartExec(ctx context.Context, spec ContainerSpec, argv []string) (ExecSession, error) {
	return e.startExec(ctx, spec, argv, false)
}

// StartOwnedExec is the Apply Git service's explicit initialization exception.
// Only this container receives CAP_CHOWN for one root exec over its empty
// workspace mounts. The returned interface offers only non-root exec.
func (e *Engine) StartOwnedExec(ctx context.Context, spec ContainerSpec, argv []string) (ExecSession, error) {
	if spec.User != "1000:1000" || len(spec.Mounts) == 0 {
		return nil, fmt.Errorf("owned exec requires user 1000:1000 and workspace mounts")
	}
	for _, mount := range spec.Mounts {
		if !mount.NoCopy || mount.ReadOnly || mount.Subpath != "" || !strings.HasPrefix(mount.Target, "/assignments/") {
			return nil, fmt.Errorf("owned exec only initializes whole private assignment roots")
		}
	}
	return e.startExec(ctx, spec, argv, true)
}

func (e *Engine) startExec(ctx context.Context, spec ContainerSpec, argv []string, initialize bool) (ExecSession, error) {
	args, err := spec.runArgs(argv)
	if err != nil {
		return nil, err
	}
	directory, err := os.MkdirTemp("", "markitect-exec-cid")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(directory)
	cid := filepath.Join(directory, "cid")
	create := []string{"create", "--cidfile", cid}
	imageIndex := len(args) - len(argv)
	for _, arg := range args[1:imageIndex] {
		if arg != "--interactive" && arg != "--rm" {
			create = append(create, arg)
		}
	}
	create = append(create, args[imageIndex:]...)
	if initialize {
		create = append([]string{create[0], "--cap-add", "CHOWN"}, create[1:]...)
	}
	out, createErr := e.output(ctx, create...)
	// The cidfile still identifies our own container when the create response
	// is lost. Never remove by name: a name conflict belongs to someone else.
	raw, readErr := os.ReadFile(cid)
	id := strings.TrimSpace(string(raw))
	if readErr != nil || !containerIDPattern.MatchString(id) {
		return nil, errors.Join(createErr, readErr, fmt.Errorf("created exec container identity was not observed"))
	}
	session := &execSession{engine: e, id: id, user: spec.User}
	fail := func(cause error) (ExecSession, error) {
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		return nil, errors.Join(cause, session.Close(cleanup))
	}
	if createErr != nil || strings.TrimSpace(string(out)) != id {
		return fail(errors.Join(createErr, fmt.Errorf("exec container creation response was incomplete")))
	}
	if _, err := e.output(ctx, "start", id); err != nil {
		return fail(err)
	}
	out, err = e.output(ctx, "inspect", "--type", "container", id)
	if err != nil {
		return fail(err)
	}
	var rows []struct {
		ID, Image string
		State     struct{ Running *bool }
	}
	if err := json.Unmarshal(out, &rows); err != nil || len(rows) != 1 || rows[0].ID != id || rows[0].Image != spec.Image || rows[0].State.Running == nil || !*rows[0].State.Running {
		return fail(fmt.Errorf("exec container running identity was not observed"))
	}
	if initialize {
		// No recursive chown or caller-supplied script. Refuse pre-existing
		// bytes before touching ownership; new volumes stay mode 0755.
		command := []string{"exec", "--user", "0:0", id, "/usr/local/bin/node", "-e", `const fs=require('node:fs'); for(const p of process.argv.slice(1)){const s=fs.lstatSync(p); if(!s.isDirectory()||fs.readdirSync(p).length) throw Error('assignment root is not empty'); if((s.mode&0o777)!==0o755) fs.chmodSync(p,0o755); fs.chownSync(p,1000,1000);}`}
		for _, mount := range spec.Mounts {
			command = append(command, mount.Target)
		}
		if _, err := e.output(ctx, command...); err != nil {
			return fail(err)
		}
	}
	return session, nil
}

func (s *execSession) ID() string { return s.id }

// GitExecCount counts started engine CLI processes for Git exec invocations.
// It does not establish the remote Git process's start, exit or success.
func (s *execSession) GitExecCount() int { return int(s.gitExecs.Load()) }

func (s *execSession) Exec(ctx context.Context, argv []string, stdin io.Reader, stdout io.Writer) error {
	if len(argv) == 0 || !absolute(argv[0]) {
		return fmt.Errorf("exec needs an absolute executable")
	}
	for _, arg := range argv {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("exec argument contains NUL")
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	closed := s.closed
	s.mu.Unlock()
	if closed {
		return fmt.Errorf("exec container is closed")
	}
	cmd := exec.CommandContext(ctx, s.engine.path, append([]string{"exec", "--interactive", "--user", s.user, s.id}, argv...)...)
	cmd.WaitDelay = 2 * time.Second
	diagnostics := &boundedBuffer{max: maxCommandOutput}
	cmd.Stdin, cmd.Stdout, cmd.Stderr = stdin, stdout, diagnostics
	err := cmd.Start()
	if err == nil {
		if argv[0] == "/usr/bin/git" {
			s.gitExecs.Add(1)
		}
		err = cmd.Wait()
	}
	if ctx.Err() != nil {
		// Killing the CLI alone does not prove its remote Git process stopped.
		// End the owned container before returning a cancelled operation.
		cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
		defer cancel()
		return errors.Join(ctx.Err(), s.Close(cleanup))
	}
	if err != nil {
		return fmt.Errorf("exec container %s: %w: %s", s.id, err, strings.TrimSpace(diagnostics.String()))
	}
	return nil
}

func (s *execSession) Close(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return s.err
	}
	s.closed = true
	cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	_, removeErr := s.engine.output(cleanup, "rm", "--force", s.id)
	out, inspectErr := s.engine.output(cleanup, "ps", "--all", "--quiet", "--no-trunc", "--filter", "id="+s.id)
	if inspectErr == nil && strings.TrimSpace(string(out)) != "" {
		inspectErr = fmt.Errorf("exec container removal was not observed")
	}
	s.err = errors.Join(removeErr, inspectErr)
	return s.err
}
