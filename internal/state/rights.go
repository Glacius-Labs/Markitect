package state

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/lang"
)

// CodeWriteDenied is the code of a change a step's role may not make.
const CodeWriteDenied = "write.denied"

// RulesFile holds the rules the system learned of the owner's decisions.
const RulesFile = ".markitect/rules.md"

// ControlFiles are the control files of .markitect/; the CI definition, the
// settings' CI patterns, is one too. Only the owner writes them.
var ControlFiles = []string{VersionFile, SpecFile, SettingsFile, AttributesFile, RulesFile}

// Violation is a change a step's role may not make, at a file below the
// project folder.
type Violation struct{ Path, Message string }

func (v Violation) String() string { return v.Path + ": " + CodeWriteDenied + ": " + v.Message }

// The kinds of file the write table tells apart.
const (
	kindControl      = "control"
	kindState        = "state"
	kindHistory      = "history"
	kindReport       = "report"
	kindSample       = "sample"
	kindIgnore       = "ignore"
	kindMarkitect    = "markitect"
	kindSpec         = "spec"
	kindCounterCheck = "counter-check"
	kindArtifact     = "artifact"
	kindRepository   = "repository"
)

// writers are the roles the shipped write table lets change a kind of file,
// besides the owner, who may change every kind. The state, the history and
// the reports are then checked by row, line and section.
var writers = map[string][]string{
	kindState:        {"reviewer", "architect"},
	kindHistory:      {"reviewer", "architect"},
	kindReport:       {"architect", "job", "triage"},
	kindSample:       {"job"},
	kindIgnore:       {"architect"},
	kindSpec:         {"architect"},
	kindCounterCheck: {"counter-checker"},
	kindArtifact:     {"builder"},
	kindRepository:   {"builder"},
}

// nouns name each kind of file in a violation.
var nouns = map[string]string{
	kindControl:      "a control file",
	kindState:        "the state",
	kindHistory:      "the history",
	kindReport:       "a report",
	kindSample:       "the results of control samples",
	kindIgnore:       "the ignore file",
	kindMarkitect:    "a file of .markitect/ other than the reports, the samples, the state, the history and the ignore file",
	kindSpec:         "the spec",
	kindCounterCheck: "the counter-check folder",
	kindArtifact:     "an artifact",
	kindRepository:   "the commit of a nested repository",
}

// rights is what the write table says for one step: the layout and settings
// it is read from, and the role's narrowing.
type rights struct {
	role          string
	spec          string
	counterChecks string
	ci            []pattern
	repositories  map[string]bool
	narrow        []pattern
	narrowText    []string
}

// CheckStep checks what a step changed between the project before it and
// after it against the write table for its role, read with the settings and
// the layout from control: the target branch's project, or the project
// before the step. It compares the files Git keeps, the rows of the state,
// the lines of the history and the sections of the reports, and returns
// every change the role may not make, sorted.
func CheckStep(role string, before, after, control *Project) ([]Violation, error) {
	if !oneOf(role, StepRoles) {
		return nil, fmt.Errorf("%s is no role; the roles are %s", role, strings.Join(StepRoles, ", "))
	}
	f, err := ReadFactory(control)
	if err != nil {
		return nil, err
	}
	for _, d := range f.Diagnostics {
		if d.Path == SettingsFile {
			return nil, fmt.Errorf("the settings the write table is read from have errors: %s", d)
		}
	}
	r := rights{role: role, spec: control.SpecFolder, ci: lowerPatterns(DefaultCI), repositories: map[string]bool{}}
	if s := f.Settings; s != nil {
		r.counterChecks, r.ci = s.CounterChecks, lowerPatterns(s.CI)
		for _, w := range s.WriteTable {
			if w.Role == role {
				r.narrowText = append(r.narrowText, w.Pattern)
			}
		}
		r.narrow = patterns(r.narrowText)
	}
	was, err := trackedHashes(before.FS, r.repositories)
	if err != nil {
		return nil, err
	}
	is, err := trackedHashes(after.FS, r.repositories)
	if err != nil {
		return nil, err
	}
	var changed []string
	for p, h := range is {
		if was[p] != h {
			changed = append(changed, p)
		}
	}
	for p := range was {
		if _, ok := is[p]; !ok {
			changed = append(changed, p)
		}
	}
	sort.Strings(changed)
	var out []Violation
	deny := func(p, format string, args ...any) {
		out = append(out, Violation{p, fmt.Sprintf(format, args...)})
	}
	// The reports whose place a move of this step rewrote.
	var moved map[string]bool
	if role == "architect" {
		if moved, err = movedReports(before.FS, after.FS); err != nil {
			return nil, err
		}
	}
	for _, p := range changed {
		kind := r.kind(p)
		if role != "owner" && !oneOf(role, writers[kind]) {
			deny(p, "%s may not change %s", role, nouns[kind])
			continue
		}
		if len(r.narrow) > 0 && !matchesPath(r.narrow, p) {
			deny(p, "%s may not change %s: the settings narrow %s to %s", role, p, role, strings.Join(r.narrowText, ", "))
			continue
		}
		var msgs []string
		switch kind {
		case kindState:
			msgs, err = stateRights(role, before, after)
		case kindHistory:
			msgs, err = historyRights(role, before.FS, after.FS)
		case kindReport:
			msgs, err = reportRights(role, p, before.FS, after.FS, moved)
		}
		if err != nil {
			return nil, err
		}
		for _, m := range msgs {
			deny(p, "%s", m)
		}
	}
	return out, nil
}

// kind is the kind of a file below the project folder. The control files,
// .markitect/ and the CI definition are told apart in any case, so that a
// name in another case, another file on a system that tells case apart and
// the same on one that does not, is never an artifact.
func (r rights) kind(p string) string {
	first, _, _ := strings.Cut(p, "/")
	switch {
	case r.repositories[p]:
		return kindRepository
	case oneOfFold(p, ControlFiles) || matchesPath(r.ci, strings.ToLower(p)):
		return kindControl
	case strings.EqualFold(first, Dir) && first != Dir:
		return kindMarkitect
	case p == StateFile:
		return kindState
	case p == HistoryFile:
		return kindHistory
	case p == IgnoreFile:
		return kindIgnore
	case path.Dir(p) == ReportsDir:
		return kindReport
	case strings.HasPrefix(p, SamplesDir+"/"):
		return kindSample
	case strings.HasPrefix(p, Dir+"/"):
		return kindMarkitect
	case p == r.spec || strings.HasPrefix(p, r.spec+"/"):
		return kindSpec
	case r.counterChecks != "" && strings.HasPrefix(p, r.counterChecks+"/"):
		return kindCounterCheck
	}
	return kindArtifact
}

func oneOfFold(s string, list []string) bool {
	for _, x := range list {
		if strings.EqualFold(s, x) {
			return true
		}
	}
	return false
}

// lowerPatterns parses patterns in lowercase, to match paths in lowercase.
func lowerPatterns(lines []string) []pattern {
	var lower []string
	for _, l := range lines {
		lower = append(lower, strings.ToLower(l))
	}
	return patterns(lower)
}

// patterns parses lines written like those of a .gitignore file at the top
// of the project.
func patterns(lines []string) []pattern {
	return parseIgnore("", []byte(strings.Join(lines, "\n")))
}

// matchesPath says whether patterns take a file, as a .gitignore file at the
// top would exclude it: by itself or by a folder above it.
func matchesPath(ps []pattern, p string) bool {
	parts := strings.Split(p, "/")
	for i := 1; i < len(parts); i++ {
		if lastMatch(ps, strings.Join(parts[:i], "/"), true) {
			return true
		}
	}
	return lastMatch(ps, p, false)
}

// trackedHashes hashes every file Git keeps of the project at fsys. A
// nested repository counts by the commit its HEAD names, as Git records it,
// so that what is not committed in it does not count; it is added to repos.
func trackedHashes(fsys fs.FS, repos map[string]bool) (map[string]string, error) {
	files, err := TrackedFiles(fsys)
	if err != nil {
		return nil, err
	}
	out := map[string]string{}
	for _, p := range files {
		if isFolder(fsys, p) && nestedRepository(fsys, p) {
			repos[p] = true
			out[p] = Hash([]byte("a repository at " + HeadCommit(fsys, p) + "\n"))
			continue
		}
		if out[p], err = HashFile(fsys, p); err != nil {
			return nil, err
		}
	}
	return out, nil
}

// TrackedFiles are the files of the project at fsys that Git keeps: those
// of the walk, with what .markitect/ignore excludes, since that file says
// only which files the project does not explain, and every file of
// .markitect/.
func TrackedFiles(fsys fs.FS) ([]string, error) {
	w := &walker{fsys: fsys, git: map[string][]pattern{}}
	if err := w.walk("", nil); err != nil {
		return nil, err
	}
	err := fs.WalkDir(fsys, Dir, func(p string, e fs.DirEntry, err error) error {
		switch {
		case err != nil && p == Dir && errors.Is(err, fs.ErrNotExist):
			return fs.SkipDir
		case err != nil:
			return err
		case strings.EqualFold(e.Name(), ".git") && e.IsDir():
			return fs.SkipDir
		case !e.IsDir():
			w.files = append(w.files, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(w.files)
	return w.files, nil
}

// lines are the non-blank lines of a file, without CR and white space
// around them; a missing file has none.
func lines(fsys fs.FS, name string) ([]string, error) {
	data, err := fs.ReadFile(fsys, name)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range strings.Split(string(bytes.TrimPrefix(data, []byte("\xef\xbb\xbf"))), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}

// stateRights checks the rows a step added to the state: the reviewer's
// rows have the source mark, each with a verdict event of the step on its
// document with its verdict and inputs; the architect's and the owner's the
// source move, each carried by a move event of the step from a row there
// was. Rows are compared as the tool writes them, so a rewritten file
// changes no row that stayed.
func stateRights(role string, beforeP, afterP *Project) ([]string, error) {
	before, after := beforeP.FS, afterP.FS
	canonical := func(fsys fs.FS) (map[string]Row, []string, error) {
		ls, err := lines(fsys, StateFile)
		if err != nil {
			return nil, nil, err
		}
		rows := map[string]Row{}
		var bad []string
		for _, l := range ls {
			parsed, problems := ParseState([]byte(l))
			if len(problems) > 0 || len(parsed) != 1 {
				bad = append(bad, l)
				continue
			}
			rows[string(parsed[0].line())] = parsed[0]
		}
		return rows, bad, nil
	}
	was, wasBad, err := canonical(before)
	if err != nil {
		return nil, err
	}
	is, isBad, err := canonical(after)
	if err != nil {
		return nil, err
	}
	var out []string
	seen := map[string]bool{}
	for _, l := range wasBad {
		seen[l] = true
	}
	for _, l := range isBad {
		if !seen[l] {
			out = append(out, fmt.Sprintf("%s may not write a line of the state that is no row: %s", role, l))
		}
	}
	var keys []string
	for k := range is {
		if _, ok := was[k]; !ok {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	events, err := addedEvents(before, after)
	if err != nil {
		return nil, err
	}
	var old []Row
	for _, r := range was {
		old = append(old, r)
	}
	for _, k := range keys {
		row := is[k]
		switch {
		case row.Source == "mark" && role == "reviewer":
			if !verdictFor(row, events) {
				out = append(out, fmt.Sprintf("%s may not write a row with source mark without a verdict event of the step on it with its verdict and inputs: %s", role, row.Doc))
			}
		case row.Source == "move" && (role == "architect" || role == "owner"):
			if !carried(row, old, events, beforeP, afterP) {
				out = append(out, fmt.Sprintf("%s may not write a row with source move that no move of the step carried from a row there was: %s", role, row.Doc))
			}
		default:
			out = append(out, fmt.Sprintf("%s may not write a row with source %s: %s", role, row.Source, row.Doc))
		}
	}
	return out, nil
}

// addedEvents are the events of the lines a step added to the history, in
// order.
func addedEvents(before, after fs.FS) ([]Event, error) {
	was, err := lines(before, HistoryFile)
	if err != nil {
		return nil, err
	}
	is, err := lines(after, HistoryFile)
	if err != nil {
		return nil, err
	}
	had := map[string]bool{}
	for _, l := range was {
		had[l] = true
	}
	var added []string
	for _, l := range is {
		if !had[l] {
			added = append(added, l)
		}
	}
	events, _ := ParseHistory([]byte(strings.Join(added, "\n")))
	return events, nil
}

// verdictFor says whether a verdict event names a row's document, verdict
// and inputs.
func verdictFor(row Row, events []Event) bool {
	for _, e := range events {
		if e.Type == EventVerdict && e.Doc == row.Doc && e.Verdict == row.Verdict && e.Inputs == InputsDigest(row.Inputs) {
			return true
		}
	}
	return false
}

// carried says whether a row with source move is what the moves of the
// step make of a row there was, as MoveRows carries it: the same verdict,
// reason and evidence, its document and its type: and link: inputs at the
// paths the moves lead to, and every hash as it was, but a self, type: or
// link: hash of a document a move rewrote, which matched its text before
// and matches it after.
func carried(row Row, old []Row, events []Event, before, after *Project) bool {
	var moves []Event
	rewritten := map[string]bool{}
	for _, e := range events {
		if e.Type == EventMove {
			moves = append(moves, e)
			rewritten[e.To] = true
			for _, p := range e.Rewritten {
				rewritten[p] = true
			}
		}
	}
	if len(moves) == 0 {
		return false
	}
	lead := func(p string) string {
		for _, m := range moves {
			if p == m.From {
				p = m.To
			}
		}
		return p
	}
	back := func(p string) string {
		for i := len(moves) - 1; i >= 0; i-- {
			if p == moves[i].To {
				p = moves[i].From
			}
		}
		return p
	}
	text := func(p *Project, doc string) string {
		if d, ok := p.Spec.Document(doc); ok {
			return Hash(d.Source)
		}
		return ""
	}
	target := func(key, doc string) string {
		switch {
		case key == InputSelf:
			return doc
		case strings.HasPrefix(key, InputType):
			return strings.TrimPrefix(key, InputType)
		case strings.HasPrefix(key, InputLink):
			return strings.TrimPrefix(key, InputLink)
		}
		return ""
	}
next:
	for _, o := range old {
		if lead(o.Doc) != row.Doc || o.Verdict != row.Verdict || o.Reason != row.Reason ||
			strings.Join(o.Evidence, "\n") != strings.Join(row.Evidence, "\n") || len(o.Inputs) != len(row.Inputs) {
			continue
		}
		for k, h := range o.Inputs {
			key := k
			if t := target(k, o.Doc); t != "" && k != InputSelf {
				key = strings.TrimSuffix(k, t) + lead(t)
			}
			now, ok := row.Inputs[key]
			if !ok {
				continue next
			}
			if now == h {
				continue
			}
			t := target(key, row.Doc)
			if t == "" || !rewritten[t] || h != text(before, back(t)) || now != text(after, t) {
				continue next
			}
		}
		return true
	}
	return false
}

// historyRights checks the lines a step added to the history, which is only
// appended to: the reviewer adds verdict events, the architect and the owner
// move events.
func historyRights(role string, before, after fs.FS) ([]string, error) {
	was, err := lines(before, HistoryFile)
	if err != nil {
		return nil, err
	}
	is, err := lines(after, HistoryFile)
	if err != nil {
		return nil, err
	}
	had, has := map[string]bool{}, map[string]bool{}
	for _, l := range was {
		had[l] = true
	}
	for _, l := range is {
		has[l] = true
	}
	var out []string
	for _, l := range was {
		if !has[l] {
			out = append(out, fmt.Sprintf("%s may not remove a line of the history, which is only appended to: %s", role, l))
		}
	}
	for _, l := range is {
		if had[l] {
			continue
		}
		events, problems := ParseHistory([]byte(l))
		switch {
		case len(problems) > 0 || len(events) != 1:
			if role != "owner" {
				out = append(out, fmt.Sprintf("%s may not add a line of the history that is no verdict or move event: %s", role, l))
			}
		case events[0].Type == EventVerdict && role != "reviewer":
			out = append(out, fmt.Sprintf("%s may not add a verdict event: %s", role, events[0].Doc))
		case events[0].Type == EventMove && role != "architect" && role != "owner":
			out = append(out, fmt.Sprintf("%s may not add a move event: %s -> %s", role, events[0].From, events[0].To))
		}
	}
	return out, nil
}

// movedReports are the reports the move events a step added name.
func movedReports(before, after fs.FS) (map[string]bool, error) {
	was, err := lines(before, HistoryFile)
	if err != nil {
		return nil, err
	}
	is, err := lines(after, HistoryFile)
	if err != nil {
		return nil, err
	}
	had := map[string]bool{}
	for _, l := range was {
		had[l] = true
	}
	out := map[string]bool{}
	for _, l := range is {
		if had[l] {
			continue
		}
		events, _ := ParseHistory([]byte(l))
		for _, e := range events {
			if e.Type == EventMove {
				for _, r := range e.Reports {
					out[r] = true
				}
			}
		}
	}
	return out, nil
}

// reportSections are the parts of a report's text: "" for the title and
// the text before the first section, and each section by its heading in
// lowercase.
func reportSections(fsys fs.FS, p string) (map[string]string, *lang.Document, error) {
	data, err := fs.ReadFile(fsys, p)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	d := lang.Parse(p, data)
	text := strings.ReplaceAll(strings.ReplaceAll(string(data), "\r\n", "\n"), "\r", "\n")
	all := strings.Split(text, "\n")
	first := len(all)
	if len(d.Sections) > 0 {
		first = d.Sections[0].Line - 1
	}
	out := map[string]string{"": strings.TrimSpace(strings.Join(all[:first], "\n"))}
	for _, s := range d.Sections {
		var b []string
		for _, l := range s.Raw {
			b = append(b, l.Text)
		}
		out[strings.ToLower(s.Heading)] += strings.TrimSpace(strings.Join(b, "\n")) + "\n"
	}
	return out, d, nil
}

// reportSource is how a report's source reads: "owner", "intake", "job" or
// "" for a source not of its form.
func reportSource(d *lang.Document) string {
	vs, _ := values(d, "Source")
	if len(vs) == 0 {
		return ""
	}
	m := sourceForm.FindStringSubmatch(vs[0].Text)
	if m == nil {
		return ""
	}
	return strings.Fields(m[1])[0]
}

// jobSections are the sections a new report of the job may hold.
var jobSections = []string{"source", "place", "finding", "occurred or risk", "violated property", "prevention"}

// sectionsOf are the sections, by their headings in lowercase, each role
// may change in a report it did not write.
var sectionsOf = map[string][]string{
	"triage":    {"rating", "actions", "closing"},
	"job":       {"prevention"},
	"architect": {"place"},
}

// reportRights checks a change of one report. The job writes new reports
// with a job or intake as source, the owner new reports with the owner as
// source; in a report there was, triage changes only the rating, the actions
// and the closing, the job only the prevention, and the architect only a
// place that a move of the step rewrote. Only the owner removes a report.
func reportRights(role, p string, before, after fs.FS, moved map[string]bool) ([]string, error) {
	was, _, err := reportSections(before, p)
	if err != nil {
		return nil, err
	}
	is, d, err := reportSections(after, p)
	if err != nil {
		return nil, err
	}
	switch {
	case is == nil && role != "owner":
		return []string{role + " may not remove a report"}, nil
	case is == nil:
		return nil, nil
	case was == nil:
		source := reportSource(d)
		switch {
		case role == "job" && (source == "job" || source == "intake"):
			var extra []string
			for k := range is {
				if k != "" && !oneOf(k, jobSections) {
					extra = append(extra, k)
				}
			}
			if len(extra) == 0 {
				return nil, nil
			}
			sort.Strings(extra)
			return []string{fmt.Sprintf("job may not write %s in a new report; triage sets them", strings.Join(extra, ", "))}, nil
		case role == "owner" && source == "owner":
			return nil, nil
		case role == "job" || role == "owner":
			return []string{fmt.Sprintf("%s may not write a report whose source is not %s", role, map[string]string{"job": "a job or intake", "owner": "the owner"}[role])}, nil
		}
		return []string{role + " may not write a new report"}, nil
	case role == "owner":
		return nil, nil
	case role == "architect" && !moved[p]:
		return []string{"architect may change a report only where a move of the step rewrote its place"}, nil
	}
	keys := map[string]bool{}
	for k := range was {
		keys[k] = true
	}
	for k := range is {
		keys[k] = true
	}
	var denied []string
	for k := range keys {
		if was[k] != is[k] && !oneOf(k, sectionsOf[role]) {
			if k == "" {
				k = "the title"
			}
			denied = append(denied, k)
		}
	}
	if len(denied) == 0 {
		return nil, nil
	}
	sort.Strings(denied)
	return []string{fmt.Sprintf("%s may not change %s of a report", role, strings.Join(denied, ", "))}, nil
}

// Added is an entry of status on a branch that status on its target does not
// show: its kind, "changed", "suspect", "unexplained" or "waiting", the
// document or, for an unexplained one, the file, and the changes.
type Added struct {
	Kind, Doc string
	Changes   []string
}

// Landing compares status on a branch with status on its target, both with
// their waiting entries, by document and kind: the entries the branch adds
// keep it from landing. A document waiting on the owner with no change its
// reports do not excuse is no entry.
func Landing(branch, target *Status) []Added {
	keys := func(s *Status) []Added {
		var out []Added
		for _, e := range s.Changed {
			out = append(out, Added{Changed, e.Doc, e.Changes})
		}
		for _, e := range s.Suspect {
			out = append(out, Added{Suspect, e.Doc, e.Changes})
		}
		for _, u := range s.Unexplained {
			out = append(out, Added{"unexplained", u, nil})
		}
		for _, w := range s.Waiting {
			if w.Counts() {
				out = append(out, Added{"waiting", w.Doc, w.Changes})
			}
		}
		return out
	}
	shown := map[string]bool{}
	for _, a := range keys(target) {
		shown[a.Kind+"\x00"+a.Doc] = true
	}
	var out []Added
	for _, a := range keys(branch) {
		if !shown[a.Kind+"\x00"+a.Doc] {
			out = append(out, a)
		}
	}
	return out
}
