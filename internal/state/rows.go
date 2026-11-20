package state

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
)

// StateFile holds the state: one row per document.
const StateFile = ".markitect/state.jsonl"

// The verdicts.
const (
	VerdictOK   = "ok"
	VerdictFail = "fail"
	VerdictSkip = "skip"
)

// The kinds of input. A key is "self" or a kind and a path, such as
// "link:roles/member.md"; type, link and asset paths are below the spec
// folder, the others below the project folder.
const (
	InputSelf     = "self"
	InputType     = "type:"
	InputLink     = "link:"
	InputAsset    = "asset:"
	InputFile     = "file:"
	InputCheck    = "check:"
	InputEvidence = "evidence:"
)

// Inputs maps each input of a document to its hash.
type Inputs map[string]string

// Row is one line of the state: a document's verdict and the hashes of what
// it rests on at its last confirmation. The field order is the order the
// state file is written in.
type Row struct {
	Doc      string   `json:"doc"`
	Verdict  string   `json:"verdict"`
	Source   string   `json:"source"`
	Inputs   Inputs   `json:"inputs"`
	Evidence []string `json:"evidence"`
	Reason   string   `json:"reason"`
}

// paths returns the paths of the row's inputs of one kind, sorted.
func (r Row) paths(kind string) []string {
	var out []string
	for k := range r.Inputs {
		if p, ok := strings.CutPrefix(k, kind); ok {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}

// Files are the row's artifacts, Checks its counter-checks.
func (r Row) Files() []string  { return r.paths(InputFile) }
func (r Row) Checks() []string { return r.paths(InputCheck) }

// line is the row as one line of JSON, without its line end.
func (r Row) line() []byte {
	if r.Inputs == nil {
		r.Inputs = Inputs{}
	}
	if r.Evidence == nil {
		r.Evidence = []string{}
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	_ = enc.Encode(r)
	return bytes.TrimRight(b.Bytes(), "\n")
}

// Problem is a line of the state file that cannot be read.
type Problem struct {
	Line    int
	Message string
}

// CodeUnreadable is the code of a line of the state that cannot be read.
const CodeUnreadable = "state.unreadable"

func (p Problem) String() string {
	return fmt.Sprintf("%s:%d: %s: %s", StateFile, p.Line, CodeUnreadable, p.Message)
}

// ReadState reads the state of the project at fsys: its rows and the lines it
// cannot read. A project without a state file has no rows.
func ReadState(fsys fs.FS) ([]Row, []Problem, error) {
	data, err := fs.ReadFile(fsys, StateFile)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	rows, problems := ParseState(data)
	return rows, problems, nil
}

// ParseState reads the lines of a state file. Lines may stand in any order
// and twice, as a union merge leaves them; blank lines and CR at line ends
// are ignored, and so are fields it does not know.
func ParseState(data []byte) ([]Row, []Problem) {
	data = bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))
	var rows []Row
	var problems []Problem
	for i, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var r Row
		dec := json.NewDecoder(strings.NewReader(line))
		if err := dec.Decode(&r); err != nil {
			problems = append(problems, Problem{i + 1, "not a JSON object of a row: " + err.Error()})
			continue
		}
		if dec.More() {
			problems = append(problems, Problem{i + 1, "more than one JSON value"})
			continue
		}
		if msg := r.problem(); msg != "" {
			problems = append(problems, Problem{i + 1, msg})
			continue
		}
		rows = append(rows, r)
	}
	return rows, problems
}

// problem says what is wrong with a row read from the file, "" for nothing.
func (r Row) problem() string {
	switch {
	case !cleanPath(r.Doc):
		return fmt.Sprintf("doc %q is not a path below the spec folder", r.Doc)
	case r.Verdict != VerdictOK && r.Verdict != VerdictFail && r.Verdict != VerdictSkip:
		return fmt.Sprintf("verdict %q is none of ok, fail and skip", r.Verdict)
	case r.Source != "mark" && r.Source != "move":
		return fmt.Sprintf("source %q is neither mark nor move", r.Source)
	case r.Inputs[InputSelf] == "":
		return "no self input"
	}
	for k, h := range r.Inputs {
		if !validHash(h) {
			return fmt.Sprintf("input %s: %q is no sha256 hash", k, h)
		}
		if k == InputSelf {
			continue
		}
		known := false
		for _, kind := range []string{InputType, InputLink, InputAsset, InputFile, InputCheck, InputEvidence} {
			if p, ok := strings.CutPrefix(k, kind); ok {
				known = true
				if !cleanPath(p) {
					return fmt.Sprintf("input %s: not a clean relative path", k)
				}
			}
		}
		if !known {
			return fmt.Sprintf("unknown input %s", k)
		}
	}
	return ""
}

// cleanPath says whether p is a clean relative path with forward slashes.
func cleanPath(p string) bool {
	return p != "" && p != "." && path.Clean(p) == p && !strings.HasPrefix(p, "/") && p != ".." &&
		!strings.HasPrefix(p, "../") && !strings.Contains(p, "\\")
}

func validHash(h string) bool {
	hex, ok := strings.CutPrefix(h, "sha256:")
	return ok && len(hex) == 64 && isHex(hex)
}

// FormatState writes rows as a state file: one line each, sorted by document
// and then by line, a line that stands twice once.
func FormatState(rows []Row) []byte {
	type keyed struct {
		doc  string
		line []byte
	}
	var lines []keyed
	for _, r := range rows {
		lines = append(lines, keyed{r.Doc, r.line()})
	}
	sort.Slice(lines, func(i, j int) bool {
		if lines[i].doc != lines[j].doc {
			return lines[i].doc < lines[j].doc
		}
		return bytes.Compare(lines[i].line, lines[j].line) < 0
	})
	var b bytes.Buffer
	for i, l := range lines {
		if i > 0 && bytes.Equal(l.line, lines[i-1].line) {
			continue
		}
		b.Write(l.line)
		b.WriteByte('\n')
	}
	return b.Bytes()
}
