package run

import (
	"bufio"
	"context"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// Trailers of the Command's Result commits. A landing commit names the work it
// lands, which is the default base of the next Command: what a landing merged in
// from a moved target was not realized by the Run, so it stays in the next
// impact. A Result commit on a Run branch names the Run's target, start and
// impact, from which a later landing reads whether it covers that Run.
const (
	trailerPrefix = "Run-"
	trailerRun    = "Run-Name"
	trailerTarget = "Run-Target"
	trailerWork   = "Run-Work"
	trailerStart  = "Run-Start"
	trailerFull   = "Run-Full"
	trailerImpact = "Run-Impact"
)

// impactKeys lists what a Command's impact covers: the Definitions the change
// changes or reaches, Findings' Definitions, and the files it reaches.
func impactKeys(imp analysis.Impact, items []Item) []string {
	keys := map[string]bool{}
	for _, c := range imp.Definitions {
		keys["def:"+c.Path] = true
	}
	for _, it := range imp.Reached {
		keys["def:"+it.Definition] = true
	}
	for _, f := range imp.ReachedFiles {
		keys["file:"+f.File] = true
	}
	for _, it := range items {
		if it.Definition != "" {
			keys["def:"+it.Definition] = true
		}
		for _, f := range it.Look {
			keys["file:"+f] = true
		}
	}
	return sortedKeys(keys)
}

// resultMessage is the message of a Result commit.
func resultMessage(r *Result, work string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Markitect Run %s: %s\n\n%s: %s\n", r.Run, r.State, trailerRun, r.Run)
	if work != "" {
		fmt.Fprintf(&b, "%s: %s\n", trailerWork, work)
		switch {
		case r.Full:
			fmt.Fprintf(&b, "%s: true\n", trailerFull)
		case r.FullCheck.Since != "":
			fmt.Fprintf(&b, "%s: %s\n", trailerFullCheck, r.FullCheck.Since)
		case r.FullCheck.Note == "":
			// Known to know no full check; an unread count claims nothing.
			fmt.Fprintf(&b, "%s: %s\n", trailerFullCheck, fullCheckNone)
		}
		return b.String()
	}
	fmt.Fprintf(&b, "%s: %s\n%s: %s\n%s: %s\n%s: %s\n", trailerTarget, r.Target, trailerStart, r.Start, trailerCommand, r.Command, trailerState, r.State)
	if r.Full {
		fmt.Fprintf(&b, "%s: true\n", trailerFull)
	}
	for _, k := range r.impact {
		fmt.Fprintf(&b, "%s: %s\n", trailerImpact, k)
	}
	return b.String()
}

// trailers reads the Command's trailers from a commit message.
func trailers(message string) map[string][]string {
	t := map[string][]string{}
	sc := bufio.NewScanner(strings.NewReader(message))
	sc.Buffer(make([]byte, 64<<10), 16<<20)
	for sc.Scan() {
		k, v, ok := strings.Cut(sc.Text(), ": ")
		if ok && strings.HasPrefix(k, trailerPrefix) {
			t[k] = append(t[k], strings.TrimSpace(v))
		}
	}
	return t
}

// lastBase is the default base: the work of the last landing on the target,
// read from the landing commit's trailer. A landing commit without one is its
// own base.
func (e *Engine) lastBase(ctx context.Context, tip string) (string, bool, error) {
	landed, ok, err := e.Git.LastLanded(ctx, tip)
	if err != nil || !ok {
		return "", ok, err
	}
	c, err := e.Git.Commit(ctx, landed)
	if err != nil {
		return "", false, err
	}
	if w := trailers(c.Message)[trailerWork]; len(w) == 1 {
		if ok, err := e.Git.Contains(ctx, landed, w[0]); err == nil && ok {
			return w[0], true, nil
		}
	}
	return landed, true, nil
}

// covered lists the kept Run branches this landing covers, with their tips: a
// kept Run of the same target branch is covered when this Run started from a
// state that holds its start and this Run's impact holds its impact, or this Run
// covered everything. Only a branch whose tip is the Command's Result commit
// counts: a commit that adds nothing but that Run's Result page, which no agent
// can write.
func (e *Engine) covered(ctx context.Context, r *Result) (map[string]string, error) {
	tips, err := e.Git.Branches(ctx, RunBranchPrefix)
	if err != nil {
		return nil, err
	}
	ours := map[string]bool{}
	for _, k := range r.impact {
		ours[k] = true
	}
	out := map[string]string{}
	for _, branch := range sortedKeys(tips) {
		tip := tips[branch]
		name := strings.TrimPrefix(branch, RunBranchPrefix)
		if name == r.Run {
			continue
		}
		c, err := e.Git.Commit(ctx, tip)
		if err != nil {
			return nil, err
		}
		t := trailers(c.Message)
		if len(t[trailerRun]) != 1 || t[trailerRun][0] != name || len(t[trailerStart]) != 1 ||
			len(t[trailerTarget]) != 1 || t[trailerTarget][0] != r.Target || len(c.Parents) != 1 {
			continue
		}
		changed, err := e.Git.Changed(ctx, c.Parents[0], tip)
		if err != nil {
			return nil, err
		}
		if len(changed) != 1 || changed[0].Path != ResultFile(name) {
			continue
		}
		if ok, err := e.Git.Contains(ctx, r.Start, t[trailerStart][0]); err != nil || !ok {
			continue
		}
		all := true
		if !r.Full {
			if len(t[trailerFull]) > 0 {
				all = false
			}
			for _, k := range t[trailerImpact] {
				if !ours[k] {
					all = false
					break
				}
			}
		}
		if all {
			out[branch] = tip
		}
	}
	return out, nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
