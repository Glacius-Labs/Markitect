package state

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// Folder is the files of a project at one moment, as the walk lists them,
// with their hashes computed once.
type Folder struct {
	fsys   fs.FS
	Files  []string
	set    map[string]bool
	hashes map[string]string
}

// ReadFolder walks the project at fsys.
func ReadFolder(fsys fs.FS) (*Folder, error) {
	files, err := Walk(fsys)
	if err != nil {
		return nil, err
	}
	f := &Folder{fsys: fsys, Files: files, set: map[string]bool{}, hashes: map[string]string{}}
	for _, p := range files {
		f.set[p] = true
	}
	return f, nil
}

// Has says whether the walk lists a file.
func (f *Folder) Has(p string) bool { return f.set[p] }

// hash is a file's hash, "" when the walk does not list it.
func (f *Folder) hash(p string) (string, error) {
	if !f.set[p] {
		return "", nil
	}
	if h, ok := f.hashes[p]; ok {
		return h, nil
	}
	h, err := HashFile(f.fsys, p)
	if err != nil {
		return "", err
	}
	f.hashes[p] = h
	return h, nil
}

// The kinds of entry of status.
const (
	Changed = "changed"
	Suspect = "suspect"
)

// Entry is one document on the list of status.
type Entry struct {
	Doc         string
	Kind        string
	Fingerprint string
	// Changes says what moved: "no row", "gone", "self", an input such as
	// "link:roles/member.md" with " (new)", " (dropped)" or " (gone)", "fail:
	// REASON" or "conflict: fail, ok".
	Changes []string
	// LookAt is where to look: the document's file, its artifacts and
	// counter-checks, below the project folder, and its last evidence.
	LookAt []string
}

// Note is a document that needs nothing but says something: a skip at its
// current state.
type Note struct {
	Doc, Verdict, Reason string
}

// Status is what status lists, and the row it takes for each document.
type Status struct {
	Changed, Suspect []Entry
	Unexplained      []string
	Notes            []Note
	docs             map[string]*docState
	folder           *Folder
	project          *Project
}

// docState is one document as status sees it.
type docState struct {
	doc     *lang.Document // nil for a document that is gone
	path    string
	current Inputs // self, type and links; nil when gone
	rows    []Row
	// taken are the rows status takes: those that match the folder, or the
	// closest one.
	taken    []Row
	matching bool
	conflict bool
	entry    *Entry
}

// Lists says whether the state has anything to list.
func (s *Status) Lists() bool {
	return len(s.Changed)+len(s.Suspect)+len(s.Unexplained) > 0
}

// Row returns the row status takes for a document, and whether rows of it
// conflict.
func (s *Status) Row(doc string) (Row, bool, bool) {
	d, ok := s.docs[doc]
	if !ok || len(d.taken) == 0 {
		return Row{}, false, false
	}
	return d.taken[0], d.conflict, true
}

// Evaluate compares the state with the project as it is. The spec should
// have no errors; a link to no document counts for nothing.
func Evaluate(p *Project, f *Folder, rows []Row) (*Status, error) {
	s := &Status{docs: map[string]*docState{}, folder: f, project: p}
	inputs, err := currentInputs(p.Spec)
	if err != nil {
		return nil, err
	}
	for _, d := range p.Spec.Documents {
		s.docs[d.Path] = &docState{doc: d, path: d.Path, current: inputs[d.Path]}
	}
	sorted := append([]Row(nil), rows...)
	sort.SliceStable(sorted, func(i, j int) bool { return string(sorted[i].line()) < string(sorted[j].line()) })
	for _, r := range sorted {
		d, ok := s.docs[r.Doc]
		if !ok {
			d = &docState{path: r.Doc}
			s.docs[r.Doc] = d
		}
		if n := len(d.rows); n == 0 || string(d.rows[n-1].line()) != string(r.line()) {
			d.rows = append(d.rows, r)
		}
	}
	var docs []string
	for doc := range s.docs {
		docs = append(docs, doc)
	}
	sort.Strings(docs)
	mapped := map[string]bool{}
	for _, doc := range docs {
		d := s.docs[doc]
		if err := s.judge(d); err != nil {
			return nil, err
		}
		if d.doc != nil {
			for _, r := range d.taken {
				for _, p := range append(r.Files(), r.Checks()...) {
					mapped[p] = true
				}
			}
		}
	}
	prefix := p.SpecFolder + "/"
	for _, file := range f.Files {
		if !mapped[file] && !strings.HasPrefix(file, prefix) {
			s.Unexplained = append(s.Unexplained, file)
		}
	}
	for _, doc := range docs {
		d := s.docs[doc]
		if d.entry == nil {
			continue
		}
		fp, err := s.fingerprint(d, d.entry.Kind == Changed)
		if err != nil {
			return nil, err
		}
		d.entry.Fingerprint = fp
		if d.entry.Kind == Changed {
			s.Changed = append(s.Changed, *d.entry)
		} else {
			s.Suspect = append(s.Suspect, *d.entry)
		}
	}
	return s, nil
}

// currentInputs computes self, type and links of every document: the
// documents it links to and the documents that link to it, not itself.
func currentInputs(spec *lang.Spec) (map[string]Inputs, error) {
	self := map[string]string{}
	for _, d := range spec.Documents {
		self[d.Path] = Hash(d.Source)
	}
	out := map[string]Inputs{}
	for _, d := range spec.Documents {
		out[d.Path] = Inputs{InputSelf: self[d.Path]}
	}
	for _, d := range spec.Documents {
		if h, ok := self[d.TypePath]; d.HasType() && ok {
			out[d.Path][InputType+d.TypePath] = h
		}
		for _, l := range d.Links {
			h, ok := self[l.Target]
			if !ok || l.Target == d.Path {
				continue
			}
			out[d.Path][InputLink+l.Target] = h
			out[l.Target][InputLink+d.Path] = self[d.Path]
		}
	}
	return out, nil
}

// compare matches a row against the document as it is: whether every input
// matches and the row has every current input, and how many of its inputs
// match.
func (s *Status) compare(r Row, current Inputs) (bool, int, error) {
	match, score := true, 0
	for k, h := range r.Inputs {
		now, err := s.now(k, current)
		if err != nil {
			return false, 0, err
		}
		if now == h {
			score++
		} else {
			match = false
		}
	}
	for k := range current {
		if _, ok := r.Inputs[k]; !ok {
			match = false
		}
	}
	return match, score, nil
}

// now is an input's hash as the project is: "" for one that is gone.
func (s *Status) now(key string, current Inputs) (string, error) {
	for _, kind := range []string{InputFile, InputCheck, InputEvidence} {
		if p, ok := strings.CutPrefix(key, kind); ok {
			return s.folder.hash(p)
		}
	}
	return current[key], nil
}

// judge takes the rows of a document and puts it on the list when it is to be
// confirmed.
func (s *Status) judge(d *docState) error {
	if d.doc == nil {
		d.taken = d.rows[:1]
		d.entry = &Entry{Doc: d.path, Kind: Changed, Changes: []string{"gone"}, LookAt: s.lookAt(d)}
		return nil
	}
	if len(d.rows) == 0 {
		d.entry = &Entry{Doc: d.path, Kind: Changed, Changes: []string{"no row"}, LookAt: s.lookAt(d)}
		return nil
	}
	best, bestScore := -1, -1
	var matching []Row
	for i, r := range d.rows {
		match, score, err := s.compare(r, d.current)
		if err != nil {
			return err
		}
		if match {
			matching = append(matching, r)
		}
		if score > bestScore {
			best, bestScore = i, score
		}
	}
	if len(matching) > 0 {
		d.taken, d.matching = matching, true
		verdicts := map[string]bool{}
		for _, r := range matching {
			verdicts[r.Verdict] = true
		}
		if len(verdicts) > 1 {
			d.conflict = true
			var vs []string
			for v := range verdicts {
				vs = append(vs, v)
			}
			sort.Strings(vs)
			d.entry = &Entry{Doc: d.path, Kind: Changed, Changes: []string{"conflict: " + strings.Join(vs, ", ")}, LookAt: s.lookAt(d)}
			return nil
		}
		switch r := matching[0]; r.Verdict {
		case VerdictFail:
			d.entry = &Entry{Doc: d.path, Kind: Changed, Changes: []string{"fail: " + r.Reason}, LookAt: s.lookAt(d)}
		case VerdictSkip:
			s.Notes = append(s.Notes, Note{Doc: d.path, Verdict: r.Verdict, Reason: r.Reason})
		}
		return nil
	}
	r := d.rows[best]
	d.taken = []Row{r}
	changes, changed, err := s.changes(r, d.current)
	if err != nil {
		return err
	}
	kind := Suspect
	if changed || r.Verdict == VerdictFail {
		kind = Changed
	}
	if r.Verdict == VerdictFail {
		changes = append(changes, "fail: "+r.Reason)
	}
	d.entry = &Entry{Doc: d.path, Kind: kind, Changes: changes, LookAt: s.lookAt(d)}
	return nil
}

// changes names what differs between a row and the document as it is, in
// the order self, type, link, file, check, evidence, and whether any of it
// makes the document changed rather than suspect.
func (s *Status) changes(r Row, current Inputs) ([]string, bool, error) {
	keys := map[string]bool{}
	for k := range r.Inputs {
		keys[k] = true
	}
	for k := range current {
		keys[k] = true
	}
	order := []string{InputSelf, InputType, InputLink, InputFile, InputCheck, InputEvidence}
	rank := func(k string) int {
		for i, kind := range order {
			if k == kind || i > 0 && strings.HasPrefix(k, kind) {
				return i
			}
		}
		return len(order)
	}
	var sorted []string
	for k := range keys {
		sorted = append(sorted, k)
	}
	sort.Slice(sorted, func(i, j int) bool {
		if a, b := rank(sorted[i]), rank(sorted[j]); a != b {
			return a < b
		}
		return sorted[i] < sorted[j]
	})
	var out []string
	changed := false
	for _, k := range sorted {
		was, had := r.Inputs[k]
		now, err := s.now(k, current)
		if err != nil {
			return nil, false, err
		}
		_, has := current[k]
		local := rank(k) <= 2
		switch {
		case local && !had:
			out = append(out, k+" (new)")
		case local && !has:
			out = append(out, k+" (dropped)")
		case now == was:
			continue
		case now == "":
			out = append(out, k+" (gone)")
		default:
			out = append(out, k)
		}
		if !strings.HasPrefix(k, InputType) && !strings.HasPrefix(k, InputLink) {
			changed = true
		}
	}
	return out, changed, nil
}

// lookAt is where to look for a document: its file, the artifacts and
// counter-checks of the rows taken, and their evidence.
func (s *Status) lookAt(d *docState) []string {
	var out []string
	if d.doc != nil {
		out = append(out, path.Join(s.project.SpecFolder, d.path))
	}
	seen := map[string]bool{}
	add := func(p string) {
		if !seen[p] {
			seen[p] = true
			out = append(out, p)
		}
	}
	rows := d.taken
	if len(rows) == 0 {
		rows = d.rows
	}
	for _, r := range rows {
		for _, p := range append(r.Files(), r.Checks()...) {
			add(p)
		}
	}
	for _, r := range rows {
		for _, e := range r.Evidence {
			add(e)
		}
	}
	return out
}

// FingerprintLen is how many hexadecimal digits of the fingerprint status
// prints and mark compares.
const FingerprintLen = 8

// fingerprint hashes what a verdict on the document will rest on, without
// the evidence: its own inputs, the artifacts and counter-checks of the rows
// taken as they are now, and for a changed document every unexplained file,
// which the reviewer may map.
func (s *Status) fingerprint(d *docState, changed bool) (string, error) {
	var lines []string
	for k, h := range d.current {
		lines = append(lines, k+" "+h)
	}
	if d.doc == nil {
		lines = append(lines, "gone "+d.path)
	}
	for _, r := range d.taken {
		for _, kind := range []string{InputFile, InputCheck} {
			for _, p := range r.paths(kind) {
				h, err := s.folder.hash(p)
				if err != nil {
					return "", err
				}
				if h == "" {
					h = "gone"
				}
				lines = append(lines, kind+p+" "+h)
			}
		}
	}
	if changed {
		for _, u := range s.Unexplained {
			h, err := s.folder.hash(u)
			if err != nil {
				return "", err
			}
			lines = append(lines, "unexplained:"+u+" "+h)
		}
	}
	sort.Strings(lines)
	lines = compact(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])[:FingerprintLen], nil
}

// compact drops repeated neighbours of a sorted list.
func compact(list []string) []string {
	var out []string
	for i, s := range list {
		if i == 0 || s != list[i-1] {
			out = append(out, s)
		}
	}
	return out
}
