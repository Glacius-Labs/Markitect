package run

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The migration from markitect 0.20 (DEC-099, decision 7): update writes the
// record from the newest certificate under .markitect/results/, from the spec and
// the file hashes the certificate holds, and removes the results folder, whose
// certificates and Result pages the record replaces. Every Definition is recorded
// ok at the certified state, with the certificate's pointers as evidence where it
// has them; a "not shown" answer becomes skip.

// ResultsDir is where markitect 0.20 wrote its certificates and Result pages.
const ResultsDir = project.Dir + "/results"

// certificateKind and certificateFormat mark a certificate of markitect gate.
const (
	certificateKind   = "markitect-certificate"
	certificateFormat = 1
)

// certificate is what the migration reads of a certificate.
type certificate struct {
	Kind        string            `json:"kind"`
	Format      int               `json:"format"`
	Created     time.Time         `json:"created"`
	Base        string            `json:"base,omitempty"`
	Files       map[string]string `json:"files"`
	Definitions map[string]string `json:"definitions"`
	Spec        map[string]string `json:"spec"`
	Items       []struct {
		Kind       string `json:"kind"`
		Definition string `json:"definition,omitempty"`
		Answer     string `json:"answer,omitempty"`
		Pointers   []struct {
			File string `json:"file"`
			Line int    `json:"line,omitempty"`
		} `json:"pointers,omitempty"`
		Explanation string `json:"explanation,omitempty"`
	} `json:"items"`
}

// Migrate turns the newest certificate of the project at root into its record,
// unless it has one, and removes the results folder. It returns the files it
// wrote or removed; with dry it changes nothing and returns what it would.
func Migrate(root string, dry bool) ([]string, error) {
	results, err := resultFiles(root)
	if err != nil || len(results) == 0 {
		return nil, err
	}
	var changed []string
	if existing, err := readRecordFile(root); err != nil {
		return nil, err
	} else if existing == nil {
		data, err := recordFromCertificate(root)
		if err != nil {
			return nil, err
		}
		if data == nil && holdsJSON(results) {
			// Something there may be a certificate this markitect cannot read: it
			// stays, rather than be lost.
			return nil, fmt.Errorf("%s holds no certificate this markitect can read; move its JSON files away, or remove the folder, and run markitect update again", ResultsDir)
		}
		if data != nil {
			changed = append(changed, project.ReviewFile)
			if !dry {
				if err := writeAtomic(filepath.Join(root, filepath.FromSlash(project.Dir)), path.Base(project.ReviewFile), data); err != nil {
					return nil, err
				}
			}
		}
	}
	for _, file := range results {
		changed = append(changed, file)
		if !dry {
			if err := os.Remove(filepath.Join(root, filepath.FromSlash(file))); err != nil && !errors.Is(err, os.ErrNotExist) {
				return nil, err
			}
		}
	}
	if !dry {
		_ = os.Remove(filepath.Join(root, filepath.FromSlash(ResultsDir)))
	}
	return changed, nil
}

// holdsJSON says whether a list of files holds a JSON file.
func holdsJSON(files []string) bool {
	for _, f := range files {
		if strings.HasSuffix(f, ".json") {
			return true
		}
	}
	return false
}

// resultFiles lists the files of the results folder, project-relative.
func resultFiles(root string) ([]string, error) {
	var out []string
	dir := filepath.Join(root, filepath.FromSlash(ResultsDir))
	err := filepath.WalkDir(dir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return filepath.SkipDir
			}
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, p)
		if err != nil {
			return err
		}
		out = append(out, filepath.ToSlash(rel))
		return nil
	})
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	return out, err
}

// recordFromCertificate is the record the newest certificate gives, or nil when
// the results folder holds no certificate.
func recordFromCertificate(root string) ([]byte, error) {
	c, err := newestCertificate(root)
	if err != nil || c == nil {
		return nil, err
	}
	c.moveSpec()
	p, err := project.Load(certified{c})
	if err != nil {
		return nil, err
	}
	hash := func(file string) (string, error) {
		if h, ok := c.Files[file]; ok {
			return "sha256:" + h, nil
		}
		return "", nil
	}
	all, err := state{p: p, hash: hash}.inputs()
	if err != nil {
		return nil, err
	}
	lines := map[string]*Line{}
	for _, id := range sortedKeys(all) {
		lines[id] = &Line{Definition: id, Verdict: VerdictOK, Inputs: all[id]}
	}
	for _, it := range c.Items {
		l := lines[it.Definition]
		if l == nil || it.Kind != "definition" {
			continue
		}
		if it.Answer == "not shown" {
			l.Verdict, l.Why = VerdictSkip, oneLine(it.Explanation)
			if l.Why == "" {
				l.Why = "not shown"
			}
		}
		for _, ptr := range it.Pointers {
			text := ptr.File
			if ptr.Line > 0 {
				text += fmt.Sprintf(":%d", ptr.Line)
			}
			l.Evidence = appendNew(l.Evidence, text)
			if h, ok := c.Files[ptr.File]; ok {
				l.Inputs[InputEvidence+ptr.File] = "sha256:" + h
			}
		}
	}
	var out []Line
	for _, id := range sortedKeys(lines) {
		out = append(out, *lines[id])
	}
	return Render(out), nil
}

// newestCertificate finds the newest certificate in the results folder, by what
// the certificates record, never by file times: one that another names as its
// base is older than that one; of the rest, as after a merge of two certified
// branches, the one made last.
func newestCertificate(root string) (*certificate, error) {
	entries, err := os.ReadDir(filepath.Join(root, filepath.FromSlash(ResultsDir)))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	all := map[string]*certificate{}
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(ResultsDir), e.Name()))
		if err != nil {
			return nil, err
		}
		var c certificate
		if json.Unmarshal(data, &c) != nil || c.Kind != certificateKind || c.Format != certificateFormat || c.Files == nil {
			continue
		}
		all[path.Join(ResultsDir, e.Name())] = &c
	}
	succeeded := map[string]bool{}
	for _, c := range all {
		succeeded[c.Base] = true
	}
	var best string
	for _, pass := range []bool{true, false} {
		for _, file := range sortedKeys(all) {
			if pass && succeeded[file] {
				continue
			}
			if best == "" || all[file].Created.After(all[best].Created) || (all[file].Created.Equal(all[best].Created) && file > best) {
				best = file
			}
		}
		if best != "" {
			break
		}
	}
	if best == "" {
		return nil, nil
	}
	return all[best], nil
}

// moveSpec names the files of a certificate as the layout names them now:
// markitect 0.20 kept the spec below project.OldSpecDir.
func (c *certificate) moveSpec() {
	moved := func(in map[string]string) map[string]string {
		out := make(map[string]string, len(in))
		for file, v := range in {
			out[project.SpecPath(file)] = v
		}
		return out
	}
	c.Files, c.Spec = moved(c.Files), moved(c.Spec)
	for i := range c.Items {
		for j := range c.Items[i].Pointers {
			c.Items[i].Pointers[j].File = project.SpecPath(c.Items[i].Pointers[j].File)
		}
	}
}

// certified is a certificate as a set of a project's files: the files it hashed
// and the text of its spec.
type certified struct{ c *certificate }

func (s certified) Files() ([]string, error) { return sortedKeys(s.c.Files), nil }

func (s certified) Read(file string) ([]byte, error) {
	if text, ok := s.c.Spec[file]; ok {
		return []byte(text), nil
	}
	return nil, fmt.Errorf("the certificate holds no text of %s", file)
}

func (s certified) GitIgnored(string) bool { return false }
