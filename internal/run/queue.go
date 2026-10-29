package run

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// impactKeys names what a Run's impact covers: the Definitions it changes or
// reaches and the files it reaches. The target branch's lock reads overlap
// from them, and a landing reads from them which kept Run branches it
// covers, as landing reads incoming changes (concept 2.0 §6.5).
func impactKeys(imp analysis.Impact) []string {
	seen := map[string]bool{}
	add := func(kind, v string) {
		if v != "" {
			seen[kind+":"+v] = true
		}
	}
	for _, c := range imp.Definitions {
		add("def", c.Path)
	}
	for _, it := range imp.Reached {
		add("def", it.Definition)
	}
	for _, f := range imp.ReachedFiles {
		add("file", f.File)
	}
	for _, c := range imp.Checklists {
		for _, f := range c.Findings {
			add("def", f.Definition)
		}
		for _, st := range c.Stale {
			add("file", st.File)
			add("def", st.Target)
			if st.File == "" && st.Target == "" {
				add("page", st.PageFile)
			}
		}
	}
	keys := make([]string, 0, len(seen))
	for k := range seen {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func overlap(a, b []string) []string {
	in := map[string]bool{}
	for _, k := range a {
		in[k] = true
	}
	var both []string
	for _, k := range b {
		if in[k] {
			both = append(both, k)
		}
	}
	return both
}

// lockFormat is the version of a target branch's lock record.
const lockFormat = 1

// lockRecord is the target branch's lock: the Runs of apply that work on it
// now, each with its impact and the time after which its entry no longer
// counts, so a host that died leaves no lock behind for longer than its
// Run's deadline.
type lockRecord struct {
	Format int         `json:"format"`
	Runs   []lockEntry `json:"runs"`
}

type lockEntry struct {
	Run   string    `json:"run"`
	Until time.Time `json:"until"`
	// Nonce tells a Run its own entry from another Run's of the same name,
	// for example after a write that took effect but reported a failure.
	Nonce  string   `json:"nonce"`
	Impact []string `json:"impact"`
}

func readLock(data []byte, now time.Time) ([]lockEntry, error) {
	if len(data) == 0 {
		return nil, nil
	}
	var rec lockRecord
	if err := json.Unmarshal(data, &rec); err != nil || rec.Format != lockFormat {
		return nil, fmt.Errorf("the lock record is unreadable (format %d): %v", rec.Format, err)
	}
	var live []lockEntry
	for _, l := range rec.Runs {
		if l.Until.After(now) {
			live = append(live, l)
		}
	}
	return live, nil
}

func writeLock(entries []lockEntry) ([]byte, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	return json.MarshalIndent(lockRecord{Format: lockFormat, Runs: entries}, "", "  ")
}

// errRunNameTaken refuses a Run whose name another running Run holds.
var errRunNameTaken = errors.New("taken")

// enter takes the Run's place in the target branch's lock: it waits while a
// Run whose impact overlaps its own works on the branch, then enters its
// entry with a compare-and-swap of the lock record. It returns what it
// waited for. Runs whose impact does not overlap enter side by side. After
// an error the Run may still hold its entry; the caller leaves it.
func (e *Engine) enter(ctx context.Context, target string, entry lockEntry) (waited []string, err error) {
	since := e.now()
	var blockers []string
	note := func(suffix string) {
		if len(blockers) > 0 {
			waited = append(waited, fmt.Sprintf("waited %s for %s, whose impact overlaps%s", e.now().Sub(since).Round(time.Second), strings.Join(blockers, " and "), suffix))
		}
	}
	defer func() {
		// The deadline may end the wait in any Git call, not only between
		// two reads.
		if err != nil && ctx.Err() != nil {
			note(", until the deadline")
		}
	}()
	for {
		data, version, err := e.Git.Lock(ctx, target)
		if err != nil {
			return waited, fmt.Errorf("reading the lock of %s: %w", target, err)
		}
		live, err := readLock(data, e.now())
		if err != nil {
			return waited, fmt.Errorf("%s: %w; delete the lock ref of %s if no Run works on it", target, err, target)
		}
		var now []string
		for _, l := range live {
			switch {
			case l.Nonce == entry.Nonce:
				// A write that reported a failure took effect.
				note("")
				return waited, nil
			case l.Run == entry.Run:
				return waited, fmt.Errorf("%w: a Run named %s works on %s until %s at the latest", errRunNameTaken, entry.Run, target, l.Until.UTC().Format(time.RFC3339))
			}
			if both := overlap(l.Impact, entry.Impact); len(both) > 0 {
				now = append(now, fmt.Sprintf("%s (overlapping in %s)", l.Run, strings.Join(firstKeys(both, 3), ", ")))
			}
		}
		if len(now) == 0 {
			out, err := writeLock(append(live, entry))
			if err != nil {
				return waited, err
			}
			if err := e.Git.SetLock(ctx, target, out, version); err != nil {
				// Another Run changed the lock meanwhile, or the write took
				// effect after all; read it again.
				if _, again, rerr := e.Git.Lock(ctx, target); rerr != nil || again == version {
					return waited, fmt.Errorf("writing the lock of %s: %w", target, err)
				}
				continue
			}
			note("")
			return waited, nil
		}
		for _, b := range now {
			if !contains(blockers, b) {
				blockers = append(blockers, b)
			}
		}
		select {
		case <-ctx.Done():
			return waited, ctx.Err()
		case <-time.After(e.lockPoll()):
		}
	}
}

// leave removes the Run's entry, known by its nonce, from the target
// branch's lock, and with it entries whose time has passed. It runs on its
// own cleanup deadline.
func (e *Engine) leave(ctx context.Context, target, nonce string) error {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), e.cleanup())
	defer cancel()
	for attempt := 0; attempt < 5; attempt++ {
		data, version, err := e.Git.Lock(ctx, target)
		if err != nil {
			return err
		}
		live, err := readLock(data, e.now())
		if err != nil {
			return err
		}
		var rest []lockEntry
		found := false
		for _, l := range live {
			if l.Nonce == nonce {
				found = true
			} else {
				rest = append(rest, l)
			}
		}
		if !found {
			return nil
		}
		out, err := writeLock(rest)
		if err != nil {
			return err
		}
		if err := e.Git.SetLock(ctx, target, out, version); err == nil {
			return nil
		}
	}
	return fmt.Errorf("the lock of %s kept changing", target)
}

func (e *Engine) lockPoll() time.Duration {
	if e.LockPoll > 0 {
		return e.LockPoll
	}
	return 15 * time.Second
}

func firstKeys(keys []string, n int) []string {
	if len(keys) <= n {
		return keys
	}
	return append(append([]string{}, keys[:n]...), fmt.Sprintf("%d more", len(keys)-n))
}

func contains(list []string, v string) bool {
	for _, l := range list {
		if l == v {
			return true
		}
	}
	return false
}

// Trailers of the host's Result commits. A landing commit names the
// approved work it lands, which is the default base of the next Run: what a
// landing merged in from a moved target was not realized by the Run, so it
// stays in the next Run's impact. A Result commit on a Run branch names the
// Run's target, start and impact, from which a later landing reads whether
// it covers that Run. They are not Markitect trailers in the sense of
// runremote: those belong to the agents' submissions and reviews, and every
// Run starts from a landing commit that carries these.
const (
	trailerPrefix = "Run-"
	trailerRun    = "Run-Name"
	trailerTarget = "Run-Target"
	trailerWork   = "Run-Work"
	trailerStart  = "Run-Start"
	trailerFull   = "Run-Full"
	trailerImpact = "Run-Impact"
)

// resultMessage is the message of a Result commit.
func resultMessage(r *Result, work string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Markitect Run %s: %s\n\n%s: %s\n", r.Run, r.State, trailerRun, r.Run)
	if work != "" {
		fmt.Fprintf(&b, "%s: %s\n", trailerWork, work)
		return b.String()
	}
	fmt.Fprintf(&b, "%s: %s\n%s: %s\n", trailerTarget, r.Target, trailerStart, r.Start)
	if r.Full {
		fmt.Fprintf(&b, "%s: true\n", trailerFull)
	}
	for _, k := range r.impact {
		fmt.Fprintf(&b, "%s: %s\n", trailerImpact, k)
	}
	return b.String()
}

// trailers reads the host's trailers from a commit message.
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

// lastBase is the default base: the approved work of the last landing on
// the target, read from the landing commit's trailer. A landing commit
// without one is its own base.
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

// covered lists the kept Run branches this landing covers, with their tips
// (concept 2.0 §6.5): a kept Run of the same target branch is covered when
// this Run started from a state that holds its start and this Run's impact
// holds its impact, or this Run covered everything. Only a branch whose tip
// is the host's Result commit counts: a commit that adds nothing but that
// Run's Result page, which no agent can write. A Run that still holds an
// entry in the lock is never covered.
func (e *Engine) covered(ctx context.Context, r *Result) (map[string]string, error) {
	tips, err := e.Git.Branches(ctx, RunBranchPrefix)
	if err != nil {
		return nil, err
	}
	data, _, err := e.Git.Lock(ctx, r.Target)
	if err != nil {
		return nil, err
	}
	live, err := readLock(data, e.now())
	if err != nil {
		return nil, err
	}
	running := map[string]bool{}
	for _, l := range live {
		running[l.Run] = true
	}
	ours := map[string]bool{}
	for _, k := range r.impact {
		ours[k] = true
	}
	out := map[string]string{}
	for _, branch := range sortedKeys(tips) {
		tip := tips[branch]
		name := strings.TrimPrefix(branch, RunBranchPrefix)
		if name == r.Run || running[name] {
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
