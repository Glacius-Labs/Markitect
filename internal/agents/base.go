package agents

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/runremote"
)

// maxBase bounds the Markitect folder read from the base commit.
const maxBase = 64 << 20

// baseRepo is the project at the Run's base commit as the clone holds it:
// every tracked file, and the bytes of Markitect's own folder, the only
// files project.Load reads. A commit holds only what Git tracks, so nothing
// is ignored.
type baseRepo struct {
	files []string
	data  map[string][]byte
}

func (r baseRepo) Files() ([]string, error) { return r.files, nil }
func (r baseRepo) Read(f string) ([]byte, error) {
	if d, ok := r.data[f]; ok {
		return d, nil
	}
	return nil, fmt.Errorf("%s is not in the Markitect folder at the base", f)
}
func (baseRepo) GitIgnored(string) bool { return false }

// loadBase loads the project at the Run's base from the clone. The engine
// checked it before the Run started; a project with compile errors starts
// no agent.
func (a *Agent) loadBase(ctx context.Context) (*project.Project, error) {
	base := a.spec.Start.Git.Base
	var list bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "ls-tree", "-r", "-z", "--name-only", base}, "", &list); err != nil {
		return nil, err
	}
	repo := baseRepo{data: map[string][]byte{}}
	for _, f := range strings.Split(strings.TrimRight(list.String(), "\x00"), "\x00") {
		if f != "" {
			repo.files = append(repo.files, f)
		}
	}
	var archive bytes.Buffer
	bounded := &limited{w: &archive, n: maxBase}
	if err := a.git(ctx, []string{gitPath, "archive", "--format=tar", base, "--", project.Dir}, "", bounded); err != nil {
		if bounded.hit {
			return nil, fmt.Errorf("agents: the Markitect folder at the base exceeds %d bytes", maxBase)
		}
		return nil, err
	}
	tr := tar.NewReader(&archive)
	for {
		h, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read the Markitect folder at the base: %w", err)
		}
		if h.Typeflag != tar.TypeReg {
			continue
		}
		data, err := io.ReadAll(tr)
		if err != nil {
			return nil, err
		}
		repo.data[h.Name] = data
	}
	p, err := project.Load(repo)
	if err != nil {
		return nil, err
	}
	if len(p.Errors) > 0 {
		return nil, fmt.Errorf("agents: the project at the base %s has %d compile errors; no agent starts", base, len(p.Errors))
	}
	return p, nil
}

// submission reads the submission a checker in apply reviews: the tip of
// the Manager's branch, its verdicts and its diff from the commit from, or,
// when from is empty, from the commit its work started from (Git.Base).
func (a *Agent) submission(ctx context.Context, from string) (*harness.Submission, error) {
	tip, t, err := a.tip(ctx)
	if err != nil {
		return nil, err
	}
	if t.Kind != runremote.Submission {
		return nil, fmt.Errorf("agents: the tip %s of %s is no submission to review", tip, a.branch())
	}
	sub := &harness.Submission{Commit: tip, From: from}
	for _, v := range t.Verdicts {
		sub.Verdicts = append(sub.Verdicts, harness.Verdict{Item: v.Item, Verdict: v.Word, Evidence: v.Evidence})
	}
	base := from
	if base == "" {
		base = a.spec.Start.Git.Base
	}
	var diff bytes.Buffer
	// A diff above the bound is cut; the harness then lists the files.
	bounded := &limited{w: &diff, n: harness.MaxInlineDiff + 1}
	if err := a.git(ctx, []string{gitPath, "diff", "--no-color", "--no-ext-diff", base, tip}, "", bounded); err != nil && !bounded.hit {
		return nil, err
	}
	sub.Diff = diff.String()
	var files bytes.Buffer
	if err := a.git(ctx, []string{gitPath, "diff", "--no-color", "--name-status", base, tip}, "", &files); err != nil {
		return nil, err
	}
	for _, line := range strings.Split(strings.TrimSpace(files.String()), "\n") {
		if line != "" {
			sub.Files = append(sub.Files, strings.ReplaceAll(line, "\t", " "))
		}
	}
	return sub, nil
}

// limited keeps at most n bytes and then fails, so that a command whose
// output would exceed its bound ends; hit records that.
type limited struct {
	w   io.Writer
	n   int
	hit bool
}

func (l *limited) Write(p []byte) (int, error) {
	if len(p) > l.n {
		_, _ = l.w.Write(p[:l.n])
		l.n, l.hit = 0, true
		return 0, errors.New("output exceeds its bound")
	}
	l.n -= len(p)
	return l.w.Write(p)
}
