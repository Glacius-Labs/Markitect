package remoteserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// LoadPlan reads and validates the remote's plan.
func LoadPlan(l Layout) (*runremote.Plan, error) {
	b, err := os.ReadFile(l.Plan())
	if err != nil {
		return nil, err
	}
	var p runremote.Plan
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&p); err != nil {
		return nil, fmt.Errorf("plan: %w", err)
	}
	if err := p.Validate(); err != nil {
		return nil, err
	}
	return &p, nil
}

// repoConfig is the bare repository's configuration. The hook decides every
// push; the settings below hold even if it were missing.
var repoConfig = [][2]string{
	{"http.receivepack", "true"},
	{"http.uploadpack", "true"},
	{"receive.denyDeletes", "true"},
	{"receive.denyNonFastForwards", "true"},
	{"receive.denyCurrentBranch", "refuse"},
	{"receive.fsckObjects", "true"},
	{"receive.advertisePushOptions", "false"},
	{"receive.autogc", "false"},
	{"receive.maxInputSize", "1073741824"},
	{"core.logAllRefUpdates", "true"},
	{"gc.auto", "0"},
	{"gc.reflogExpire", "never"},
	{"gc.reflogExpireUnreachable", "never"},
}

// Seed creates the Run's bare repository from a bundle that holds the plan's
// base commit: base and every working Manager's branch start there, and the
// pre-receive hook runs hook (an argv) with the remote's root.
func Seed(l Layout, git string, bundle io.Reader, hook []string) error {
	plan, err := LoadPlan(l)
	if err != nil {
		return err
	}
	if _, err := os.Stat(l.Repo()); err == nil {
		return fmt.Errorf("%s exists; a remote is seeded once", l.Repo())
	}
	if err := os.MkdirAll(l.Tmp(), 0o700); err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	g := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, git, args...)
		cmd.Env = append(gitConfigEnv(), "PATH="+os.Getenv("PATH"), "HOME="+l.Root)
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(string(out)), nil
	}
	repo := l.Repo()
	if _, err := g("init", "-q", "--bare", repo); err != nil {
		return err
	}
	for _, kv := range repoConfig {
		if _, err := g("--git-dir="+repo, "config", kv[0], kv[1]); err != nil {
			return err
		}
	}
	file := filepath.Join(l.Tmp(), "start.bundle")
	if err := writeFile(file, bundle); err != nil {
		return err
	}
	defer os.Remove(file)
	if _, err := g("--git-dir="+repo, "fetch", "-q", "--no-tags", file, "+refs/*:refs/seed/*"); err != nil {
		return err
	}
	if t, err := g("--git-dir="+repo, "cat-file", "-t", plan.Base); err != nil || t != "commit" {
		return fmt.Errorf("the start bundle does not hold the base commit %s", plan.Base)
	}
	seeds, err := g("--git-dir="+repo, "for-each-ref", "--format=%(refname)", "refs/seed/")
	if err != nil {
		return err
	}
	for _, ref := range strings.Fields(seeds) {
		if _, err := g("--git-dir="+repo, "update-ref", "-d", ref); err != nil {
			return err
		}
	}
	branches := []string{runremote.BaseBranch}
	for _, m := range plan.Managers {
		if m.Working {
			branches = append(branches, runremote.Branch(m.ID))
		}
	}
	for _, b := range branches {
		if _, err := g("--git-dir="+repo, "update-ref", "--create-reflog", "-m", "markitect: Run start", "refs/heads/"+b, plan.Base, runremote.ZeroID); err != nil {
			return err
		}
	}
	if _, err := g("--git-dir="+repo, "symbolic-ref", "HEAD", "refs/heads/"+runremote.BaseBranch); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(repo, "info", "attributes"), []byte(neutralAttributes), 0o644); err != nil {
		return err
	}
	hooks := filepath.Join(repo, "hooks")
	if err := os.RemoveAll(hooks); err != nil {
		return err
	}
	if err := os.Mkdir(hooks, 0o755); err != nil {
		return err
	}
	script := "#!/bin/sh\nexec " + shellQuote(append(append([]string(nil), hook...), "hook", "--root", filepath.ToSlash(l.Root))) + "\n"
	if err := os.WriteFile(filepath.Join(hooks, "pre-receive"), []byte(script), 0o755); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(l.Root, seedName), []byte(plan.Base+"\n"), 0o600)
}

// Seeded reports whether the remote serves.
func Seeded(l Layout) bool {
	_, err := os.Stat(filepath.Join(l.Root, seedName))
	return err == nil
}

// Export writes a bundle of every branch of the remote.
func Export(l Layout, git string, w io.Writer) error {
	if !Seeded(l) {
		return errors.New("the remote is not seeded")
	}
	cmd := exec.Command(git, "--git-dir="+l.Repo(), "bundle", "create", "-", "--branches")
	cmd.Env = append(gitConfigEnv(), "PATH="+os.Getenv("PATH"), "HOME="+l.Root)
	var stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = w, &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git bundle: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func writeFile(name string, r io.Reader) error {
	f, err := os.OpenFile(name, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	_, err = io.Copy(f, r)
	if cerr := f.Close(); err == nil {
		err = cerr
	}
	return err
}

func shellQuote(argv []string) string {
	q := make([]string, len(argv))
	for i, a := range argv {
		q[i] = "'" + strings.ReplaceAll(filepath.ToSlash(a), "'", `'\''`) + "'"
	}
	return strings.Join(q, " ")
}
