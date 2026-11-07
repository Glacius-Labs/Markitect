package run

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The inputs of a Definition: everything its verification depended on, each by
// its hash. A change of any input puts the Definition back on the list. They
// reach what concept 4.0's checklist reached at distance 0 and 1: a Definition
// that changed, one that links to a changed Definition or is of a changed Kind,
// one whose files changed, and one whose registry rows gained or lost it.

// Keys of the inputs: the Definition's own text, its Kind, a Definition it links
// to, a file a registry row names for it, and a file a reviewer gave as evidence.
const (
	InputSelf     = "self"
	InputKind     = "kind:"
	InputLink     = "link:"
	InputFile     = "file:"
	InputEvidence = "evidence:"
)

// Inputs are a Definition's inputs, by key, each a "sha256:" hash.
type Inputs map[string]string

// Hash is the SHA-256 of a file's content as an input; in text, without a NUL
// byte, line endings do not count, so a checkout on another system has the same
// hashes.
func Hash(data []byte) string {
	if bytes.IndexByte(data, 0) < 0 {
		data = project.Normalize(data)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

// fingerprintLen is how much of the fingerprint status prints and review takes.
const fingerprintLen = 12

// Fingerprint is one short hash of every input but the evidence, which the
// reviewer chooses after status printed it.
func (in Inputs) Fingerprint() string {
	h := sha256.New()
	for _, k := range sortedKeys(in) {
		if !strings.HasPrefix(k, InputEvidence) {
			fmt.Fprintf(h, "%s\x00%s\n", k, in[k])
		}
	}
	return hex.EncodeToString(h.Sum(nil))[:fingerprintLen]
}

// Files are the files of the inputs with a key prefix, such as InputFile.
func (in Inputs) Files(prefix string) []string {
	var out []string
	for _, k := range sortedKeys(in) {
		if f, ok := strings.CutPrefix(k, prefix); ok {
			out = append(out, f)
		}
	}
	return out
}

// state is what the inputs of a project's Definitions are computed from: its
// spec and registry, and a way to hash its files.
type state struct {
	p    *project.Project
	hash func(file string) (string, error)
}

// inputs computes the inputs of every Definition of the project's model, without
// evidence.
func (s state) inputs() (map[string]Inputs, error) {
	m := s.p.Model
	rows := map[string][]string{}
	if s.p.Registry != nil {
		for _, file := range sortedKeys(s.p.Registry.Entries) {
			for _, id := range s.p.Registry.Entries[file].Realizes {
				rows[id] = append(rows[id], file)
			}
		}
	}
	out := map[string]Inputs{}
	for _, id := range m.ModelDefinitions() {
		d := m.Definitions[id]
		in := Inputs{InputSelf: Hash(d.Data)}
		for _, l := range d.Links {
			if l.Target == "" || l.Package || l.Target == d.ID {
				continue
			}
			v, ok := s.linked(l.Target)
			if !ok {
				continue
			}
			key := InputLink + l.Target
			if l.Kind {
				key = InputKind + l.Target
			}
			in[key] = v
		}
		for _, file := range rows[id] {
			h, err := s.hash(file)
			if err != nil {
				return nil, err
			}
			in[InputFile+file] = hashOrGone(h)
		}
		out[id] = in
	}
	return out, nil
}

// linked is the input a link contributes: a model Definition's text, or for a
// Library Definition its Library's digest from the lock, so that a newer binary
// changes nothing until the lock is written again. Core's Kind never changes and
// is no input.
func (s state) linked(target string) (string, bool) {
	t := s.p.Model.Definitions[target]
	switch {
	case t == nil, t.Library == "core":
		return "", false
	case t.InModel():
		return Hash(t.Data), true
	}
	if locked, ok := s.p.Lock.Libraries[t.Library]; ok {
		return locked.Digest, true
	}
	for _, lib := range s.p.Libraries {
		if lib.Name == t.Library {
			return model.LibraryDigest(lib.Files), true
		}
	}
	return "", false
}

// gone is the input of a file the walk does not find.
const gone = "gone"

func hashOrGone(h string) string {
	if h == "" {
		return gone
	}
	return h
}

// Evidence is one place a reviewer gives as evidence: a file, with a line and a
// note if the reviewer gave them, as in "tests/fees_test.go:12 (TestWaiver)".
type Evidence struct {
	Text string
	File string
}

// ParseEvidence reads what a reviewer gives as evidence: FILE, FILE:LINE,
// FILE:LINE-LINE or FILE:LINE:COLUMN, and after a space a note.
func ParseEvidence(text string) (Evidence, error) {
	text = oneLine(text)
	token, note, _ := strings.Cut(text, " ")
	file, lines := token, ""
	// FILE:LINE, FILE:LINE-LINE or, as compilers print it, FILE:LINE:COLUMN.
	for n := 0; n < 2; n++ {
		i := strings.LastIndexByte(file, ':')
		if i <= 0 || !lineSpec(file[i+1:]) {
			break
		}
		file, lines = file[:i], file[i:]+lines
	}
	file = strings.TrimPrefix(strings.ReplaceAll(file, `\`, "/"), "./")
	text = strings.TrimSpace(file + lines + " " + note)
	switch {
	case file == "":
		return Evidence{}, fmt.Errorf("evidence %q names no file", text)
	case strings.HasPrefix(file, "/") || strings.Contains(file, ":") || strings.HasPrefix(file, "../") || strings.Contains(file, "/../"):
		return Evidence{}, fmt.Errorf("evidence %q: name the file relative to the project folder", text)
	}
	return Evidence{Text: text, File: file}, nil
}

// lineSpec says whether s is a line or a range of lines, such as 12 or 12-20.
func lineSpec(s string) bool {
	from, to, ranged := strings.Cut(s, "-")
	return digits(from) && (!ranged || digits(to))
}

func digits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// oneLine keeps a text on one line, with single spaces.
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

func sortedKeys[V any](m map[string]V) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}
