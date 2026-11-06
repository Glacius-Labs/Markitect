package run

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/analysis"
)

// The work without a Run. Between checklist and gate, what the tool knows of the
// work lives outside the project, in the user's cache folder, one file per project
// folder: the change, the hashes of the spec and the harness when it began, the
// list, and each item's latest answer bound to the hashes of its files.

// Work is one piece of work without a Run.
type Work struct {
	// Project is the project folder.
	Project string `json:"project"`
	// Base is the certificate the change starts from, a project-relative path, or
	// "" when there is none. Full puts every Definition on the list.
	Base     string             `json:"base,omitempty"`
	Full     bool               `json:"full,omitempty"`
	Findings []analysis.Finding `json:"findings,omitempty"`
	// Named says checklist was given the base, --full or Findings; otherwise the
	// work follows the newest certificate.
	Named bool `json:"named,omitempty"`
	// Spec holds the hashes of the spec and the harness when the work began; the
	// work may not change them.
	Spec map[string]string `json:"spec"`
	// Items is the list as it stands: every item any state put on it, as a Run
	// keeps them (mergeItems).
	Items []Item `json:"items"`
	// Answers holds each item's latest answer.
	Answers map[string]Given `json:"answers"`
	// Reviews are the states on which every item had an answer.
	Reviews    []Reviewed `json:"reviews,omitempty"`
	MissingFor []string   `json:"missingFor,omitempty"`
	Notes      []string   `json:"notes,omitempty"`
	Began      time.Time  `json:"began"`
}

// Given is an answer with what it is bound to: the state it was given on, and the
// hashes then of the files its item names and the files it points to.
type Given struct {
	Answer
	At    string            `json:"at"`
	Files map[string]string `json:"files"`
}

// Reviewed is a state on which every item had an answer: what was wrong there, and
// what only the owner can settle.
type Reviewed struct {
	At           string          `json:"at"`
	Wrong        []string        `json:"wrong,omitempty"`
	SpecProblems []OwnerDecision `json:"specProblems,omitempty"`
}

// workFolder is the work states' folder in the cache.
const workFolder = "work"

// workKey names the work of a project folder in the cache.
func workKey(root string) string {
	p := filepath.Clean(root)
	if runtime.GOOS == "windows" {
		p = strings.ToLower(p)
	}
	sum := sha256.Sum256([]byte(p))
	return hex.EncodeToString(sum[:8])
}

func (e *Engine) cache() (Store, error) {
	if e.Cache == "" {
		return Store{}, errors.New("no cache folder for the work without a run")
	}
	return Store{Dir: e.Cache}, nil
}

// loadWork reads the work of a project folder; nil when there is none.
func (e *Engine) loadWork(root string) (*Work, error) {
	s, err := e.cache()
	if err != nil {
		return nil, err
	}
	var data []byte
	file := filepath.Join(s.Dir, workFolder, workKey(root)+".json")
	for i := 0; i < 20; i++ {
		if data, err = os.ReadFile(file); err == nil || errors.Is(err, os.ErrNotExist) {
			break
		}
		// A rename beside a read is refused for a moment on Windows.
		time.Sleep(50 * time.Millisecond)
	}
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var w Work
	if err := json.Unmarshal(data, &w); err != nil {
		return nil, fmt.Errorf("the work of %s: %w", root, err)
	}
	if w.Answers == nil {
		w.Answers = map[string]Given{}
	}
	return &w, nil
}

// updateWork changes the work of a project folder under a short lock. change gets
// nil when there is none, and returns the work to store, or nil to remove it.
func (e *Engine) updateWork(root string, change func(*Work) (*Work, error)) error {
	s, err := e.cache()
	if err != nil {
		return err
	}
	key := workKey(root)
	release, err := s.lockIn(workFolder, key, lockWait)
	if err != nil {
		return err
	}
	defer release()
	w, err := e.loadWork(root)
	if err != nil {
		return err
	}
	if w, err = change(w); err != nil {
		return err
	}
	file := filepath.Join(s.Dir, workFolder, key+".json")
	if w == nil {
		if err := os.Remove(file); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
		return nil
	}
	data, err := json.MarshalIndent(w, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(s.Dir, workFolder), key+".json", data)
}

// writeAtomic writes a file of a folder in one step: a crash leaves the old one or
// the new one.
func writeAtomic(dir, name string, data []byte) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, name+".*.tmp")
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
	for i := 0; ; i++ {
		err = os.Rename(temp, filepath.Join(dir, name))
		if err == nil || i == 20 {
			return err
		}
		time.Sleep(50 * time.Millisecond)
	}
}

// bound lists the files an answer is bound to: the files its item names and the
// files it points to.
func bound(it Item, a Answer) []string {
	files := append([]string(nil), it.Look...)
	for _, p := range a.Pointers {
		files = append(files, p.File)
	}
	return dedupe(files)
}

// given binds an answer to the folder's state.
func given(it Item, a Answer, f *folder) Given {
	g := Given{Answer: a, At: f.fingerprint, Files: map[string]string{}}
	for _, file := range bound(it, a) {
		g.Files[file] = f.hashes[file]
	}
	return g
}

// standing returns the answers that stand on the folder's state, and the items
// that have none. An answer stands on the state it was given on, and on a later
// one where none of the files it is bound to changed and its item names no file it
// did not know, as a repair review works in a Run. A wrong answer stands only on
// its own state: the work went back for it.
func (w *Work) standing(f *folder) (map[string]Given, []string) {
	valid := map[string]Given{}
	var missing []string
	for _, it := range w.Items {
		g, ok := w.Answers[it.Key]
		stands := ok && g.At == f.fingerprint
		if ok && !stands && g.Word != AnswerWrong {
			stands = true
			for _, file := range bound(it, g.Answer) {
				if was, known := g.Files[file]; !known || was != f.hashes[file] {
					stands = false
				}
			}
		}
		if stands {
			valid[it.Key] = g
		} else {
			missing = append(missing, it.Key)
		}
	}
	return valid, missing
}

// review records that every item has an answer on a state; one record per state,
// the latest.
func (w *Work) review(at string, valid map[string]Given) {
	wrong, decisions := verdictsFor(w.Items, func(key string) Answer { return valid[key].Answer }, settleLean)
	r := Reviewed{At: at, Wrong: wrong, SpecProblems: decisions}
	for i := range w.Reviews {
		if w.Reviews[i].At == at {
			w.Reviews[i] = r
			return
		}
	}
	w.Reviews = append(w.Reviews, r)
}

// rounds counts the finished reviews that sent something wrong back, which are the
// repair rounds, and says what each went back for. A review with a spec problem
// sent nothing back: the owner decides first.
func (w *Work) rounds() (int, []string) {
	var sent []string
	for _, r := range w.Reviews {
		if len(r.Wrong) > 0 && len(r.SpecProblems) == 0 {
			sent = append(sent, strings.Join(r.Wrong, "; "))
		}
	}
	return len(sent), sent
}

// specProblems lists what any finished review left to the owner. It stands until
// the spec changes, which starts the work afresh.
func (w *Work) specProblems() []OwnerDecision {
	seen := map[string]bool{}
	var out []OwnerDecision
	for _, r := range w.Reviews {
		for _, d := range r.SpecProblems {
			if !seen[d.Matter] {
				seen[d.Matter] = true
				out = append(out, d)
			}
		}
	}
	return out
}

// settleLean is what the owner is told to do about a spec problem without a Run.
const settleLean = "Settle it in the spec, and start the work again."
