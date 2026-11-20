package state

import (
	"fmt"
	"path"
	"regexp"
	"strings"
)

// Refusal is a mark that cannot be made, with every reason found. Nothing is
// written.
type Refusal struct{ Problems []string }

func (r *Refusal) Error() string { return strings.Join(r.Problems, "\n") }

// MarkRequest is a verdict the reviewer binds: the document by its path below
// the spec folder, the fingerprint status printed, the verdict, the artifacts,
// counter-checks and evidence, below the project folder, and the reason.
type MarkRequest struct {
	Doc, Fingerprint, Verdict string
	Files, Checks, Evidence   []string
	Reason                    string
}

// Fingerprint is the fingerprint of a document as status computes it, and
// whether status knows the document: it is in the spec or has a row.
func (s *Status) Fingerprint(doc string) (string, bool, error) {
	d, ok := s.docs[doc]
	if !ok {
		return "", false, nil
	}
	if d.entry != nil {
		return d.entry.Fingerprint, true, nil
	}
	fp, err := s.fingerprint(d, false)
	return fp, true, err
}

// evidenceForm is path, path:line, path:line-line or path:line:col.
var evidenceForm = regexp.MustCompile(`^(.+?)((?::[0-9]+)(?:-[0-9]+|:[0-9]+)?)?$`)

// note is a trailing note in parentheses, which is dropped.
var note = regexp.MustCompile(`\s*\([^()]*\)\s*$`)

// ParseEvidence splits a piece of evidence into its file, below the project
// folder, and the place in it, and returns it as it is stored.
func ParseEvidence(arg string) (file, stored string, ok bool) {
	e := strings.TrimSpace(note.ReplaceAllString(arg, ""))
	m := evidenceForm.FindStringSubmatch(e)
	if m == nil {
		return "", "", false
	}
	file, ok = ProjectPath(m[1])
	if !ok {
		return "", "", false
	}
	return file, file + m[2], true
}

// ProjectPath turns a path on the command line into a clean path below the
// project folder with forward slashes.
func ProjectPath(arg string) (string, bool) {
	p := strings.TrimSpace(strings.ReplaceAll(arg, "\\", "/"))
	if p == "" || strings.HasPrefix(p, "/") || len(p) >= 2 && p[1] == ':' {
		return "", false
	}
	p = path.Clean(p)
	return p, cleanPath(p)
}

// Mark binds a verdict to the document at the state status computed, and
// returns the rows of the new state. It refuses when the fingerprint is not
// the document's now, when the document is unknown or when a path is bad.
// The new state holds one row per document, the one status takes, except
// for the rows of a conflict, which stay until a mark settles it.
func Mark(s *Status, req MarkRequest) ([]Row, error) {
	var problems []string
	refuse := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	d, ok := s.docs[req.Doc]
	if !ok {
		return nil, &Refusal{[]string{"unknown document: " + req.Doc}}
	}
	fp, _, err := s.Fingerprint(req.Doc)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(req.Fingerprint, fp) {
		return nil, &Refusal{[]string{fmt.Sprintf("%s@%s is not the state now, %s@%s; run status again", req.Doc, req.Fingerprint, req.Doc, fp)}}
	}
	if d.doc == nil {
		if req.Verdict != VerdictOK {
			refuse("%s is gone from the spec: it takes only ok, which removes its row", req.Doc)
		}
		if len(req.Files)+len(req.Checks)+len(req.Evidence) > 0 {
			refuse("%s is gone from the spec: it takes no artifacts, counter-checks or evidence", req.Doc)
		}
	}
	row := Row{Doc: req.Doc, Verdict: req.Verdict, Source: "mark", Inputs: Inputs{}, Evidence: []string{},
		Reason: strings.Join(strings.Fields(req.Reason), " ")}
	for k, h := range d.current {
		row.Inputs[k] = h
	}
	prefix := s.project.SpecFolder + "/"
	named := map[string]string{}
	for _, list := range []struct {
		kind string
		args []string
	}{{InputFile, req.Files}, {InputCheck, req.Checks}} {
		for _, arg := range list.args {
			p, ok := ProjectPath(arg)
			switch {
			case !ok:
				refuse("%s is not a path below the project folder", arg)
			case strings.HasPrefix(p, prefix):
				refuse("%s lies in the spec folder: an artifact or a counter-check lies outside it", p)
			case !s.folder.Has(p):
				refuse("%s is no file of the project: it does not exist, lies in .markitect/ or is ignored", p)
			case named[p] != "" && named[p] != list.kind:
				refuse("%s is named as an artifact and as a counter-check", p)
			default:
				named[p] = list.kind
				row.Inputs[list.kind+p], err = s.folder.hash(p)
				if err != nil {
					return nil, err
				}
			}
		}
	}
	seen := map[string]bool{}
	for _, arg := range req.Evidence {
		file, stored, ok := ParseEvidence(arg)
		switch {
		case !ok:
			refuse("evidence %q is not path, path:line, path:line-line or path:line:col below the project folder", arg)
		case !s.folder.Has(file):
			refuse("evidence %s is no file of the project: it does not exist, lies in .markitect/ or is ignored", file)
		default:
			row.Inputs[InputEvidence+file], err = s.folder.hash(file)
			if err != nil {
				return nil, err
			}
			if !seen[stored] {
				seen[stored] = true
				row.Evidence = append(row.Evidence, stored)
			}
		}
	}
	if len(problems) > 0 {
		return nil, &Refusal{problems}
	}
	var out []Row
	for doc, other := range s.docs {
		switch {
		case doc == req.Doc:
		case other.conflict:
			out = append(out, other.taken...)
		case len(other.taken) > 0:
			out = append(out, other.taken[0])
		}
	}
	if d.doc != nil {
		out = append(out, row)
	}
	return out, nil
}
