package remoteserver

import (
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

// LandingCheck runs the required checks on a commit the host holds, such as
// the merge of a moved target branch before landing (concept §6.5). The
// bundle holds the commit and may be thin against the Run's branches. It
// changes no ref of the Run's repository: the commit goes into a scratch
// repository that borrows its objects. The checks run isolated as in the
// hook, and the decision log records them as the host's landing check. A
// failing check is a result; an error means the checks could not run.
func LandingCheck(l Layout, git string, bundle io.Reader, commit string, progress io.Writer) ([]runremote.CheckResult, error) {
	plan, err := LoadPlan(l)
	if err != nil {
		return nil, err
	}
	if !Seeded(l) {
		return nil, fmt.Errorf("the remote is not seeded")
	}
	scratch, err := os.MkdirTemp(l.Tmp(), "landing-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(scratch)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()
	env := append(gitConfigEnv(), "PATH="+os.Getenv("PATH"), "HOME="+l.Root)
	g := func(args ...string) (string, error) {
		cmd := exec.CommandContext(ctx, git, append([]string{"--git-dir=" + scratch}, args...)...)
		cmd.Env = env
		var stderr bytes.Buffer
		cmd.Stderr = &stderr
		out, err := cmd.Output()
		if err != nil {
			return "", fmt.Errorf("git %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
		}
		return strings.TrimSpace(string(out)), nil
	}
	if _, err := g("init", "-q", "--bare"); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(scratch, "objects", "info", "alternates"), []byte(filepath.ToSlash(filepath.Join(l.Repo(), "objects"))+"\n"), 0o600); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(scratch, "info", "attributes"), []byte(neutralAttributes), 0o600); err != nil {
		return nil, err
	}
	file := filepath.Join(scratch, "landing.bundle")
	if err := writeFile(file, bundle); err != nil {
		return nil, err
	}
	// Objects only, whatever refs the bundle names.
	if _, err := g("bundle", "unbundle", file); err != nil {
		return nil, err
	}
	if t, err := g("cat-file", "-t", commit); err != nil || t != "commit" {
		return nil, fmt.Errorf("the bundle does not hold the commit %s", commit)
	}
	repo := gitRepo{git: git, dir: scratch, env: env}
	c, err := repo.Commit(commit)
	if err != nil {
		return nil, err
	}
	var results []runremote.CheckResult
	if len(plan.Checks) > 0 {
		results, err = newCheckRunner(l, repo, progress).Run(c, plan.Checks)
	}
	// Accepted only when every check passed, as for a submission.
	dec := runremote.Decision{Accepted: err == nil, Kind: "landing-check", Checks: results}
	if err != nil {
		dec.Reason = fmt.Sprintf("the required checks could not run on %s: %v", commit, err)
	}
	for _, r := range results {
		if !r.Passed && dec.Accepted {
			dec.Accepted, dec.Reason = false, fmt.Sprintf("the required check %q failed on %s", r.Name, commit)
		}
	}
	if lerr := appendLog(l, Entry{Time: time.Now().UTC().Format(time.RFC3339Nano), User: "host",
		Updates: []runremote.Update{{New: commit}}, Decision: dec}); lerr != nil && err == nil {
		err = lerr
	}
	return results, err
}

func landing(l Layout, commit string, stdin io.Reader, stdout, stderr io.Writer) error {
	git, err := exec.LookPath("git")
	if err != nil {
		return err
	}
	results, err := LandingCheck(l, git, stdin, commit, stderr)
	if err != nil {
		return err
	}
	if results == nil {
		results = []runremote.CheckResult{}
	}
	return json.NewEncoder(stdout).Encode(results)
}
