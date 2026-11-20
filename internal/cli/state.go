package cli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/lang"
	"github.com/Glacius-Labs/Markitect/internal/state"
)

// evaluate reads the state and the files of a loaded project and compares
// them. problems are the lines of the state it cannot read, which it leaves
// out.
func evaluate(w *workspace) (*state.Status, []state.Problem, error) {
	rows, problems, err := state.ReadState(w.FS)
	if err != nil {
		return nil, nil, err
	}
	folder, err := state.ReadFolder(w.FS)
	if err != nil {
		return nil, nil, err
	}
	s, err := state.Evaluate(w.Project, folder, rows)
	return s, problems, err
}

// statusDocument is the JSON document of status.
type statusDocument struct {
	header
	Errors      []diagnostic  `json:"errors"`
	Changed     []statusEntry `json:"changed"`
	Suspect     []statusEntry `json:"suspect"`
	Unexplained []string      `json:"unexplained"`
	Notes       []statusNote  `json:"notes"`
}

type statusEntry struct {
	Doc         string   `json:"doc"`
	Fingerprint string   `json:"fingerprint"`
	Changes     []string `json:"changes"`
	LookAt      []string `json:"look_at"`
}

type statusNote struct {
	Doc     string `json:"doc"`
	Verdict string `json:"verdict"`
	Reason  string `json:"reason"`
}

// blockers are the errors of the spec and the unreadable lines of the state,
// which keep status from giving a list and mark from writing.
func blockers(w *workspace, problems []state.Problem) []diagnostic {
	out := []diagnostic{}
	for _, d := range lang.Check(w.Spec) {
		out = append(out, diagnostic{File: d.Path, Line: d.Line, Code: d.Code, Message: d.Message})
	}
	for _, p := range problems {
		out = append(out, diagnostic{File: state.StateFile, Line: p.Line, Code: state.CodeUnreadable, Message: p.Message})
	}
	return out
}

func (d diagnostic) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", d.File, d.Line, d.Code, d.Message)
}

func status(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	s, problems, err := evaluate(w)
	if err != nil {
		return ExitFailed, err
	}
	doc := statusDocument{header: newHeader("status"), Errors: blockers(w, problems),
		Changed: []statusEntry{}, Suspect: []statusEntry{}, Unexplained: []string{}, Notes: []statusNote{}}
	var b strings.Builder
	if len(doc.Errors) > 0 {
		for _, d := range doc.Errors {
			b.WriteString(d.String() + "\n")
		}
		b.WriteString("no list while the spec or the state has errors\n")
		code, err := output(o, stdout, doc, b.String())
		if err == nil {
			code = ExitFound
		}
		return code, err
	}
	entries := func(kind string, list []state.Entry) []statusEntry {
		out := []statusEntry{}
		for _, e := range list {
			out = append(out, statusEntry{Doc: e.Doc, Fingerprint: e.Fingerprint, Changes: e.Changes, LookAt: e.LookAt})
			fmt.Fprintf(&b, "%s: %s@%s: %s\n", kind, e.Doc, e.Fingerprint, strings.Join(e.Changes, ", "))
			if len(e.LookAt) > 0 {
				fmt.Fprintf(&b, "  look at: %s\n", strings.Join(e.LookAt, ", "))
			}
		}
		return out
	}
	doc.Changed = entries(state.Changed, s.Changed)
	doc.Suspect = entries(state.Suspect, s.Suspect)
	for _, u := range s.Unexplained {
		doc.Unexplained = append(doc.Unexplained, u)
		fmt.Fprintf(&b, "unexplained: %s\n", u)
	}
	for _, n := range s.Notes {
		doc.Notes = append(doc.Notes, statusNote{n.Doc, n.Verdict, n.Reason})
		fmt.Fprintf(&b, "note: %s: %s: %s\n", n.Doc, n.Verdict, n.Reason)
	}
	fmt.Fprintf(&b, "%d changed, %d suspect, %d unexplained\n", len(s.Changed), len(s.Suspect), len(s.Unexplained))
	code, err := output(o, stdout, doc, b.String())
	if err == nil && s.Lists() {
		code = ExitFound
	}
	return code, err
}

// markResult is the JSON document of mark.
type markResult struct {
	header
	Doc     string `json:"doc"`
	Verdict string `json:"verdict"`
	Removed bool   `json:"removed"`
}

func mark(o options, stdout io.Writer) (int, error) {
	docArg, fingerprint, ok := cutLast(o.positional[0], "@")
	if !ok || fingerprint == "" {
		return ExitFailed, fmt.Errorf("%s: name the document as DOCUMENT@FINGERPRINT, as status prints it", o.positional[0])
	}
	verdict := o.positional[1]
	switch {
	case verdict != state.VerdictOK && verdict != state.VerdictFail && verdict != state.VerdictSkip:
		return ExitFailed, fmt.Errorf("verdict %q is none of ok, fail and skip", verdict)
	case verdict != state.VerdictOK && strings.TrimSpace(o.reason) == "":
		return ExitFailed, fmt.Errorf("%s needs a reason with -m", verdict)
	}
	doc, ok := edit.DocumentPath(docArg)
	if !ok {
		return ExitFound, &state.Refusal{Problems: []string{"unknown document: " + docArg}}
	}
	dir, err := mustProject(o)
	if err != nil {
		return ExitFailed, err
	}
	release, err := lockProject(dir)
	if err != nil {
		return ExitFailed, err
	}
	defer release()
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	s, problems, err := evaluate(w)
	if err != nil {
		return ExitFailed, err
	}
	if blocked := blockers(w, problems); len(blocked) > 0 {
		r := &state.Refusal{}
		for _, d := range blocked {
			r.Problems = append(r.Problems, d.String())
		}
		r.Problems = append(r.Problems, "nothing is marked while the spec or the state has errors")
		return ExitFound, r
	}
	_, inSpec := w.Spec.Document(doc)
	if verdict == state.VerdictOK && inSpec && len(o.evidence) == 0 {
		return ExitFailed, errors.New("ok needs evidence with -e")
	}
	rows, err := state.Mark(s, state.MarkRequest{Doc: doc, Fingerprint: fingerprint, Verdict: verdict,
		Files: o.files, Checks: o.checks, Evidence: o.evidence, Reason: o.reason})
	if err != nil {
		var r *state.Refusal
		if errors.As(err, &r) {
			return ExitFound, err
		}
		return ExitFailed, err
	}
	if err := writeAtomic(filepath.Join(dir, filepath.FromSlash(state.StateFile)), state.FormatState(rows)); err != nil {
		return ExitFailed, err
	}
	text := "marked " + doc + " " + verdict + "\n"
	if !inSpec {
		text = "removed the row of " + doc + ", which is gone\n"
	}
	return output(o, stdout, markResult{newHeader("mark"), doc, verdict, !inSpec}, text)
}

// cutLast cuts s around the last sep.
func cutLast(s, sep string) (string, string, bool) {
	i := strings.LastIndex(s, sep)
	if i < 0 {
		return s, "", false
	}
	return s[:i], s[i+len(sep):], true
}

// cacheDir is the user's cache folder, where the locks lie; tests replace it.
var cacheDir = os.UserCacheDir

// lockWait bounds how long mark waits for another mark of the same project.
const lockWait = time.Minute

// lockLifetime is the longest a lock can be held; an older one is stale.
const lockLifetime = 2 * time.Hour

// lockProject takes the lock of a project folder: a file named after the
// folder in markitect/locks/ of the user's cache folder, holding this
// process's ID. A lock whose process is gone, or that is too old, is taken
// over.
func lockProject(dir string) (func(), error) {
	cache, err := cacheDir()
	if err != nil {
		return nil, fmt.Errorf("no cache folder for the lock of the state: %w", err)
	}
	locks := filepath.Join(cache, "markitect", "locks")
	if err := os.MkdirAll(locks, 0o755); err != nil {
		return nil, err
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(strings.ToLower(filepath.Clean(abs))))
	file := filepath.Join(locks, hex.EncodeToString(sum[:8])+".lock")
	until := time.Now().Add(lockWait)
	for {
		f, err := os.OpenFile(file, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			fmt.Fprintf(f, "%d\n", os.Getpid())
			f.Close()
			return func() { os.Remove(file) }, nil
		}
		if staleLock(file) {
			os.Remove(file)
			continue
		}
		if time.Now().After(until) {
			return nil, fmt.Errorf("another mark holds the lock %s", file)
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// staleLock says a lock can no longer be held: it is too old, or its process
// is gone.
func staleLock(file string) bool {
	info, err := os.Stat(file)
	if err != nil {
		return false
	}
	if time.Since(info.ModTime()) > lockLifetime {
		return true
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return false
	}
	pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return time.Since(info.ModTime()) > 5*time.Second
	}
	p, err := os.FindProcess(pid)
	if err != nil {
		return true
	}
	defer p.Release()
	if runtime.GOOS == "windows" {
		// FindProcess opens the process, which fails once it is gone.
		return false
	}
	return p.Signal(syscall.Signal(0)) != nil
}

// writeAtomic replaces a file in one step: a crash leaves the old one or the
// new one.
func writeAtomic(file string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(file), ".markitect-*.tmp")
	if err != nil {
		return err
	}
	temp := tmp.Name()
	defer os.Remove(temp)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Chmod(temp, 0o644); err != nil {
		return err
	}
	for i := 0; ; i++ {
		// A rename beside a reader is refused for a moment on Windows.
		if err = os.Rename(temp, file); err == nil || i == 20 {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}
