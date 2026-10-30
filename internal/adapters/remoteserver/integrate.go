package remoteserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// Integrate pushes the host's integration of a parent without items of its
// own, a commit the bundle holds, to the parent's branch (long-lived
// Managers). It pushes as user host through the repository's own
// pre-receive hook, so the hook decides it as any push, and returns that
// decision: a refusal is a result, an error means it could not push. env
// is added to the push's environment, as Server.Env is.
func Integrate(l Layout, git string, env []string, bundle io.Reader, branch, commit string) (runremote.Decision, error) {
	if !Seeded(l) {
		return runremote.Decision{}, fmt.Errorf("the remote is not seeded")
	}
	scratch, err := os.MkdirTemp(l.Tmp(), "integrate-")
	if err != nil {
		return runremote.Decision{}, err
	}
	defer os.RemoveAll(scratch)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	base := append(gitConfigEnv(), "PATH="+os.Getenv("PATH"), "HOME="+l.Root)
	g := func(extra []string, args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, git, append([]string{"--git-dir=" + scratch}, args...)...)
		cmd.Env = append(append(append([]string(nil), base...), extra...), env...)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return strings.TrimSpace(stderr.String()), fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := g(nil, "init", "-q", "--bare"); err != nil {
		return runremote.Decision{}, err
	}
	if err := os.WriteFile(filepath.Join(scratch, "objects", "info", "alternates"), []byte(filepath.ToSlash(filepath.Join(l.Repo(), "objects"))+"\n"), 0o600); err != nil {
		return runremote.Decision{}, err
	}
	file := filepath.Join(scratch, "integration.bundle")
	if err := writeFile(file, bundle); err != nil {
		return runremote.Decision{}, err
	}
	if _, err := g(nil, "bundle", "unbundle", file); err != nil {
		return runremote.Decision{}, err
	}
	if t, err := g(nil, "cat-file", "-t", commit); err != nil || t != "commit" {
		return runremote.Decision{}, fmt.Errorf("the bundle does not hold the commit %s", commit)
	}
	unlock, err := lockPushes(l)
	if err != nil {
		return runremote.Decision{}, err
	}
	defer unlock()
	before, err := logLength(l)
	if err != nil {
		return runremote.Decision{}, err
	}
	// A local push runs the repository's receive-pack and its hook, which
	// read the pusher from REMOTE_USER as behind the server.
	diagnostics, pushErr := g([]string{"REMOTE_USER=" + runremote.HostUser}, "push", "--quiet", "--no-verify", filepath.ToSlash(l.Repo()), commit+":refs/heads/"+branch)
	entries, err := logEntries(l)
	if err != nil {
		return runremote.Decision{}, err
	}
	if len(entries) > before {
		if e := entries[len(entries)-1]; e.User == runremote.HostUser {
			return e.Decision, nil
		}
	}
	if pushErr != nil {
		// Git refused before the hook decided, for example a branch that
		// moved meanwhile.
		return runremote.Decision{Reason: "the push did not reach the hook: " + diagnostics}, nil
	}
	return runremote.Decision{}, fmt.Errorf("the push of %s left no decision in the log", commit)
}

func logLength(l Layout) (int, error) {
	entries, err := logEntries(l)
	return len(entries), err
}

// logEntries reads the decision log; a remote without pushes has none. An
// entry still being written, the last line without its newline, is left out.
func logEntries(l Layout) ([]Entry, error) {
	data, err := os.ReadFile(l.Log())
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	data = data[:bytes.LastIndexByte(data, '\n')+1]
	var entries []Entry
	sc := bufio.NewScanner(bytes.NewReader(data))
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		var e Entry
		if err := json.Unmarshal(sc.Bytes(), &e); err != nil {
			return nil, err
		}
		entries = append(entries, e)
	}
	return entries, sc.Err()
}

func integrate(l Layout, branch, commit string, stdin io.Reader, stdout io.Writer) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	dec, err := Integrate(l, git, nil, stdin, branch, commit)
	if err != nil {
		return err
	}
	return json.NewEncoder(stdout).Encode(dec)
}
