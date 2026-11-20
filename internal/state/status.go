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

// Waiting is a document that waits on open reports about the spec: it is
// listed only as waiting, not as changed or suspect.
type Waiting struct {
	Doc, Fingerprint string
	// Reports are the names of the open reports it waits on.
	Reports []string
	// Escalated says that every one of them escalated to the owner: the
	// document waits on the owner, which does not make status fail.
	Escalated bool
	// Changes are what status would list for it otherwise, empty when
	// nothing moved.
	Changes []string
}

// Status is what status lists, and the row it takes for each document.
type Status struct {
	Changed, Suspect []Entry
	Unexplained      []string
	Waiting          []Waiting
	Notes            []Note
	docs             map[string]*docState
	folder           *Folder
	project          *Project
	// files is the digest of every file of the walk outside the spec folder,
	// computed once.
	files string
}

// docState is one document as status sees it.
type docState struct {
	doc     *lang.Document // nil for a document that is gone
	path    string
	current Inputs // self, type, links and assets; nil when gone
	// missing are the assets it links that are gone.
	missing map[string]bool
	rows    []Row
	// taken are the rows status takes: those that match the folder, or the
	// closest one.
	taken    []Row
	matching bool
	conflict bool
	entry    *Entry
}

// Lists says whether the state has anything to list: an entry other than a
// document waiting on the owner.
func (s *Status) Lists() bool {
	n := len(s.Changed) + len(s.Suspect) + len(s.Unexplained)
	for _, w := range s.Waiting {
		if !w.Escalated {
			n++
		}
	}
	return n > 0
}

// Wait lists as waiting every document of the spec that an open report names
// as its place, unless the report's violated property is none, which says
// that an artifact deviates from a spec that holds. Such a document leaves
// the changed and suspect entries and the notes. reports should have no
// errors of check.
func (s *Status) Wait(reports []*Report) {
	byDoc := map[string][]*Report{}
	for _, r := range reports {
		if !r.Open() || r.Property == "" || r.Property == "none" {
			continue
		}
		if doc, ok := r.SpecDoc(s.project); ok {
			byDoc[doc] = append(byDoc[doc], r)
		}
	}
	var docs []string
	for doc := range byDoc {
		docs = append(docs, doc)
	}
	sort.Strings(docs)
	for _, doc := range docs {
		w := Waiting{Doc: doc, Escalated: true, Changes: []string{}}
		for _, r := range byDoc[doc] {
			w.Reports = append(w.Reports, r.Name)
			w.Escalated = w.Escalated && r.Escalated
		}
		sort.Strings(w.Reports)
		take := func(list []Entry) []Entry {
			var out []Entry
			for _, e := range list {
				if e.Doc == doc {
					w.Changes = e.Changes
				} else {
					out = append(out, e)
				}
			}
			return out
		}
		s.Changed, s.Suspect = take(s.Changed), take(s.Suspect)
		var notes []Note
		for _, n := range s.Notes {
			if n.Doc != doc {
				notes = append(notes, n)
			}
		}
		s.Notes = notes
		s.Waiting = append(s.Waiting, w)
	}
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
	inputs := currentInputs(p.Spec)
	assets := map[string]string{}
	for _, file := range f.Files {
		if rel, ok := p.specFile(file); ok && !p.isDocument(file) {
			assets[rel] = file
		}
	}
	linked := map[string]bool{}
	for _, d := range p.Spec.Documents {
		ds := &docState{doc: d, path: d.Path, current: inputs[d.Path], missing: map[string]bool{}}
		for _, r := range d.Paths {
			if r.Target == "" || lang.IsDocumentName(path.Base(r.Target)) || !cleanPath(r.Target) {
				continue
			}
			file, ok := assets[r.Target]
			if !ok {
				ds.missing[InputAsset+r.Target] = true
				continue
			}
			h, err := f.hash(file)
			if err != nil {
				return nil, err
			}
			ds.current[InputAsset+r.Target] = h
			linked[file] = true
		}
		s.docs[d.Path] = ds
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
	for _, file := range f.Files {
		if !mapped[file] && !linked[file] && !p.isDocument(file) {
			s.Unexplained = append(s.Unexplained, file)
		}
	}
	for _, doc := range docs {
		if e := s.docs[doc].entry; e != nil && e.Kind == Changed {
			s.Changed = append(s.Changed, *e)
		} else if e != nil {
			s.Suspect = append(s.Suspect, *e)
		}
	}
	return s, nil
}

// Fingerprints computes the fingerprint of every entry on the list. Evaluate
// leaves them out, since they hash every file of the walk.
func (s *Status) Fingerprints() error {
	for _, list := range [][]Entry{s.Changed, s.Suspect} {
		for i := range list {
			fp, _, err := s.Fingerprint(list[i].Doc)
			if err != nil {
				return err
			}
			list[i].Fingerprint = fp
		}
	}
	for i := range s.Waiting {
		fp, _, err := s.Fingerprint(s.Waiting[i].Doc)
		if err != nil {
			return err
		}
		s.Waiting[i].Fingerprint = fp
	}
	return nil
}

// currentInputs computes self, type and links of every document: the
// documents it links to and the documents that link to it, not itself.
func currentInputs(spec *lang.Spec) map[string]Inputs {
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
	return out
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
	changes, changed, err := s.changes(r, d)
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
	look := s.lookAt(d)
	// The files of the spec folder that changed or are new are worth a look.
	for _, c := range changes {
		if p, ok := strings.CutPrefix(c, InputAsset); ok && !strings.HasSuffix(p, " (gone)") && !strings.HasSuffix(p, " (dropped)") {
			look = append(look, path.Join(s.project.SpecFolder, strings.TrimSuffix(p, " (new)")))
		}
	}
	d.entry = &Entry{Doc: d.path, Kind: kind, Changes: changes, LookAt: look}
	return nil
}

// changes names what differs between a row and the document as it is, in
// the order self, type, link, file, check, evidence, and whether any of it
// makes the document changed rather than suspect.
func (s *Status) changes(r Row, d *docState) ([]string, bool, error) {
	current := d.current
	keys := map[string]bool{}
	for k := range r.Inputs {
		keys[k] = true
	}
	for k := range current {
		keys[k] = true
	}
	order := []string{InputSelf, InputType, InputLink, InputAsset, InputFile, InputCheck, InputEvidence}
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
		local := rank(k) <= 3
		switch {
		case local && !had:
			out = append(out, k+" (new)")
		case local && !has && d.missing[k]:
			out = append(out, k+" (gone)")
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
// taken as they are now, and the digest of every file of the walk outside the
// spec folder. A mark changes only .markitect/, so marks leave one another's
// fingerprints alone, and any edit of a project file changes them all.
func (s *Status) fingerprint(d *docState) (string, error) {
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
	files, err := s.filesDigest()
	if err != nil {
		return "", err
	}
	lines = append(lines, "files "+files)
	sort.Strings(lines)
	lines = compact(lines)
	sum := sha256.Sum256([]byte(strings.Join(lines, "\n")))
	return hex.EncodeToString(sum[:])[:FingerprintLen], nil
}

// filesDigest is the SHA-256 of the path and hash of every file of the walk
// outside the spec folder, in path order.
func (s *Status) filesDigest() (string, error) {
	if s.files != "" {
		return s.files, nil
	}
	var b strings.Builder
	for _, file := range s.folder.Files {
		if s.project.isDocument(file) {
			continue
		}
		h, err := s.folder.hash(file)
		if err != nil {
			return "", err
		}
		b.WriteString(file + " " + h + "\n")
	}
	sum := sha256.Sum256([]byte(b.String()))
	s.files = hex.EncodeToString(sum[:])
	return s.files, nil
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
