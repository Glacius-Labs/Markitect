package state

import (
	"errors"
	"fmt"
	"io/fs"
	"path"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/lang"
	"github.com/Glacius-Labs/Markitect/internal/library"
)

// The files of the factory in .markitect/, by paths below the project folder.
const (
	ReportsDir   = ".markitect/reports"
	SamplesDir   = ".markitect/samples"
	SettingsFile = ".markitect/settings.md"
)

// The codes of the factory's files that check reports beside the codes of
// the language.
const (
	CodeReportForm      = "report.form"
	CodeReportPlace     = "report.place"
	CodeReportDuplicate = "report.duplicate"
	CodeSettingsForm    = "settings.form"
	CodeSampleForm      = "sample.form"
	CodeSampleVerdict   = "sample.verdict"
)

// Roles are the roles the settings and the sources of reports name.
var Roles = []string{"architect", "builder", "reviewer", "counter-checker", "sampler", "reader", "triage", "intake"}

// Properties are the values of a report's violated property: the five
// properties of a good spec, correct, and none for an artifact that deviates
// from a spec that holds.
var Properties = []string{"necessary", "complete", "consistent", "unambiguous and verifiable", "structured", "correct", "none"}

// The outcomes of a closing.
const (
	ClosedDone      = "done"
	ClosedDuplicate = "duplicate"
	ClosedNothing   = "nothing to do"
)

// Report is one report of .markitect/reports/, with the values its sections
// hold. A value that is not of its form is empty.
type Report struct {
	// Name is the file name without .md, Path the file below the project
	// folder.
	Name, Path string
	Doc        *lang.Document
	// Raised is the day of its source, as YYYY-MM-DD; By is "owner", "intake"
	// or "job", with Job and Role for a job.
	Raised, By, Job, Role string
	// Place is the path below the project folder it concerns.
	Place    string
	Occurred string // "occurred" or "risk"
	Property string
	// Impact and Likelihood are "low", "medium" or "high".
	Impact, Likelihood string
	Branches           []string
	Escalated          bool
	// Closed says that it has a closing; ClosedOn is its day, Outcome one of
	// ClosedDone, ClosedDuplicate and ClosedNothing, and DuplicateOf the name
	// of the report it repeats.
	Closed               bool
	ClosedOn, Outcome    string
	DuplicateOf          string
	placeLine, closeLine int
}

// Open says whether the report has no closing.
func (r *Report) Open() bool { return !r.Closed }

// Sample is the result of one control sample in .markitect/samples/.
type Sample struct {
	Name, Path string
	Doc        *lang.Document
	Day        string
	// Planted are the documents a deviation was planted in, below the spec
	// folder, each once, in the order of the file.
	Planted []string
	// Verdicts are the reviewer's verdicts on the copy, by document.
	Verdicts map[string]string
}

// Settings are the settings of .markitect/settings.md.
type Settings struct {
	Doc *lang.Document
	// Models maps a role to its model, Cadence a job to its text.
	Models, Cadence map[string]string
	// CounterChecks is the folder of the counter-checks below the project
	// folder, "" when the settings name none.
	CounterChecks string
	WriteTable    []WriteRule
}

// WriteRule is one line of the settings' write table.
type WriteRule struct{ Role, Pattern string }

// Factory is what the jobs and the user keep in .markitect/ beside the state
// and the history: the reports, the results of control samples and the
// settings, with what check finds in them.
type Factory struct {
	Reports []*Report
	Samples []*Sample
	// Settings is nil when the project has no settings file.
	Settings    *Settings
	Diagnostics []lang.Diagnostic
}

// DefaultSettings is the settings file init writes.
const DefaultSettings = `# Settings

## Models

- architect: opus
- builder: sonnet
- reviewer: sonnet
- counter-checker: sonnet
- sampler: sonnet
- reader: sonnet
- triage: sonnet
- intake: sonnet

## Counter-checks

tests/counter-checks
`

// ReadFactory reads the reports, the results of control samples and the
// settings of a project and checks them against the types the tool owns and
// the forms of their values. A file that cannot be read fails; one that is
// not of its form is a diagnostic.
func ReadFactory(p *Project) (*Factory, error) {
	f := &Factory{}
	types := map[string]*lang.Document{}
	for _, t := range []string{library.ReportType, library.SettingsType, library.SampleType} {
		data, err := fs.ReadFile(library.FS(), t)
		if err != nil {
			return nil, err
		}
		types[t] = lang.Parse(t, data)
	}
	reports, err := f.readFolder(p, ReportsDir, CodeReportForm, "report")
	if err != nil {
		return nil, err
	}
	for _, d := range reports {
		f.Diagnostics = append(f.Diagnostics, lang.CheckAgainst(d, types[library.ReportType], "the report type")...)
		f.Reports = append(f.Reports, f.readReport(p, d))
	}
	names := map[string]bool{}
	for _, r := range f.Reports {
		names[r.Name] = true
	}
	for _, r := range f.Reports {
		if r.Outcome == ClosedDuplicate && (!names[r.DuplicateOf] || r.DuplicateOf == r.Name) {
			f.add(r.Path, r.closeLine, CodeReportDuplicate, "duplicate of %s, which is no other report", r.DuplicateOf)
		}
	}
	samples, err := f.readFolder(p, SamplesDir, CodeSampleForm, "sample")
	if err != nil {
		return nil, err
	}
	for _, d := range samples {
		f.Diagnostics = append(f.Diagnostics, lang.CheckAgainst(d, types[library.SampleType], "the sample type")...)
		f.Samples = append(f.Samples, f.readSample(d))
	}
	data, err := fs.ReadFile(p.FS, SettingsFile)
	switch {
	case errors.Is(err, fs.ErrNotExist):
	case err != nil:
		return nil, err
	default:
		d := lang.Parse(SettingsFile, data)
		f.Diagnostics = append(f.Diagnostics, lang.CheckAgainst(d, types[library.SettingsType], "the settings type")...)
		f.Settings = f.readSettings(p, d)
	}
	sort.SliceStable(f.Diagnostics, func(i, j int) bool {
		if f.Diagnostics[i].Path != f.Diagnostics[j].Path {
			return f.Diagnostics[i].Path < f.Diagnostics[j].Path
		}
		return f.Diagnostics[i].Line < f.Diagnostics[j].Line
	})
	return f, nil
}

func (f *Factory) add(file string, line int, code, format string, args ...any) {
	f.Diagnostics = append(f.Diagnostics, lang.Diagnostic{Path: file, Line: max(line, 1), Code: code, Message: fmt.Sprintf(format, args...)})
}

// nameForm is the form of the name of a report, a sample, a job and a branch
// of a duplicate: lowercase letters and digits in words joined by hyphens.
var nameForm = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

// readFolder parses the documents of a folder of .markitect/, each NAME.md
// with a name of lowercase words joined by hyphens. Files whose name starts
// with a dot, such as .gitkeep, are passed over; anything else is a
// diagnostic.
func (f *Factory) readFolder(p *Project, dir, code, what string) ([]*lang.Document, error) {
	entries, err := fs.ReadDir(p.FS, dir)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var out []*lang.Document
	for _, e := range entries {
		file := dir + "/" + e.Name()
		base, isMD := strings.CutSuffix(e.Name(), ".md")
		switch {
		case strings.HasPrefix(e.Name(), "."):
			continue
		case e.IsDir() || !isMD || !nameForm.MatchString(base):
			f.add(file, 1, code, "a %s is a file NAME.md, its name lowercase words joined by hyphens", what)
			continue
		}
		data, err := fs.ReadFile(p.FS, file)
		if err != nil {
			return nil, err
		}
		out = append(out, lang.Parse(file, data))
	}
	return out, nil
}

// values are the value lines of a section: its lines as written without
// comments and blank lines, each without a leading list marker and without
// the backticks of a line written wholly as code. ok is false when the
// document lacks the section.
func values(d *lang.Document, section string) ([]lang.SourceLine, bool) {
	s, ok := d.Section(section)
	if !ok {
		return nil, false
	}
	out := s.Values()
	for i, l := range out {
		for _, marker := range []string{"- ", "* ", "+ "} {
			if rest, ok := strings.CutPrefix(l.Text, marker); ok {
				out[i].Text = strings.TrimSpace(rest)
				break
			}
		}
		out[i].Text = unquote(out[i].Text)
	}
	return out, true
}

// unquote drops the backticks of a value written as code.
func unquote(s string) string {
	if len(s) >= 2 && s[0] == '`' && s[len(s)-1] == '`' && !strings.Contains(s[1:len(s)-1], "`") {
		return s[1 : len(s)-1]
	}
	return s
}

// day reads a day as YYYY-MM-DD.
func day(s string) bool {
	_, err := time.Parse(time.DateOnly, s)
	return err == nil
}

func oneOf(s string, list []string) bool {
	for _, x := range list {
		if s == x {
			return true
		}
	}
	return false
}

var (
	sourceForm  = regexp.MustCompile(`^(owner|intake|job ([a-z0-9]+(?:-[a-z0-9]+)*) ([a-z]+(?:-[a-z]+)*)), (\d{4}-\d{2}-\d{2})$`)
	ratingForm  = regexp.MustCompile(`^impact (low|medium|high), likelihood (low|medium|high)$`)
	branchForm  = regexp.MustCompile(`^branch (\S+)$`)
	closingForm = regexp.MustCompile(`^(\d{4}-\d{2}-\d{2}): (done|nothing to do|duplicate of (\S+))$`)
	sampleForm  = regexp.MustCompile(`^job control-sample sampler, (\d{4}-\d{2}-\d{2})$`)
)

// readReport reads the values of a report and reports each that is not of
// its form. The sections themselves, missing or empty, are checked against
// the report type.
func (f *Factory) readReport(p *Project, d *lang.Document) *Report {
	r := &Report{Doc: d, Path: d.Path, Name: strings.TrimSuffix(path.Base(d.Path), ".md")}
	bad := func(l lang.SourceLine, format string, args ...any) {
		f.add(d.Path, l.Line, CodeReportForm, format, args...)
	}
	if vs, _ := values(d, "Source"); len(vs) > 0 {
		m := sourceForm.FindStringSubmatch(vs[0].Text)
		switch {
		case m == nil || !day(m[4]):
			bad(vs[0], "source: write owner, intake or job JOB ROLE, then a comma and the day as YYYY-MM-DD")
		case m[3] != "" && !oneOf(m[3], Roles):
			bad(vs[0], "source: %s is no role; the roles are %s", m[3], strings.Join(Roles, ", "))
		default:
			r.Raised, r.By, r.Job, r.Role = m[4], strings.Fields(m[1])[0], m[2], m[3]
		}
	}
	if vs, _ := values(d, "Place"); len(vs) > 0 {
		r.placeLine = vs[0].Line
		place, ok := placeOf(d, vs[0])
		switch {
		case !ok:
			bad(vs[0], "place: write one path below the project folder, or a Markdown link to it relative to this file")
		case reserved(place):
			bad(vs[0], "place: %s lies in %s/; a report concerns a spec document, an artifact or a folder", place, Dir)
		default:
			r.Place = place
		}
	}
	if vs, _ := values(d, "Occurred or risk"); len(vs) > 0 {
		if v := vs[0].Text; v == "occurred" || v == "risk" {
			r.Occurred = v
		} else {
			bad(vs[0], "occurred or risk: write occurred or risk")
		}
	}
	if vs, _ := values(d, "Violated property"); len(vs) > 0 {
		if v := vs[0].Text; oneOf(v, Properties) {
			r.Property = v
		} else {
			bad(vs[0], "violated property: write one of %s", strings.Join(Properties, ", "))
		}
	}
	if vs, _ := values(d, "Rating"); len(vs) > 0 {
		if m := ratingForm.FindStringSubmatch(vs[0].Text); m != nil {
			r.Impact, r.Likelihood = m[1], m[2]
		} else {
			bad(vs[0], "rating: write impact LEVEL, likelihood LEVEL, each low, medium or high")
		}
	}
	if vs, _ := values(d, "Actions"); len(vs) > 0 {
		proposals := 0
		for _, l := range vs {
			switch m := branchForm.FindStringSubmatch(l.Text); {
			case r.Escalated:
				proposals++
			case l.Text == "escalated":
				r.Escalated = true
			case m != nil:
				r.Branches = append(r.Branches, m[1])
			default:
				bad(l, "actions: write a line branch NAME per branch, or a line escalated followed by the proposals")
			}
		}
		if r.Escalated && proposals == 0 {
			bad(vs[len(vs)-1], "actions: escalated is followed by the proposals for the owner")
		}
	}
	if vs, has := values(d, "Closing"); has {
		r.Closed = true
		if len(vs) > 0 {
			r.closeLine = vs[0].Line
			m := closingForm.FindStringSubmatch(strings.TrimSuffix(vs[0].Text, "."))
			switch {
			case m == nil || !day(m[1]) || m[3] != "" && !nameForm.MatchString(m[3]):
				bad(vs[0], "closing: write YYYY-MM-DD: done, YYYY-MM-DD: duplicate of NAME or YYYY-MM-DD: nothing to do")
			case len(vs) < 2:
				bad(vs[0], "closing: the reason follows on the lines below")
			case r.Raised != "" && m[1] < r.Raised:
				bad(vs[0], "closing: closed on %s, before it was raised on %s", m[1], r.Raised)
			default:
				r.ClosedOn, r.Outcome = m[1], m[2]
				if m[3] != "" {
					r.Outcome, r.DuplicateOf = ClosedDuplicate, m[3]
				}
			}
		}
	}
	if r.Place != "" && r.Open() {
		if _, err := fs.Stat(p.FS, r.Place); err != nil {
			f.add(d.Path, r.placeLine, CodeReportPlace, "place: %s does not exist; an open report concerns something there is", r.Place)
		}
	}
	return r
}

// placeOf reads a place: a Markdown link, resolved relative to the report,
// or a path below the project folder, as a clean path below the project
// folder.
func placeOf(d *lang.Document, l lang.SourceLine) (string, bool) {
	if strings.HasPrefix(l.Text, "[") {
		var found []string
		for _, r := range d.Paths {
			if r.Line == l.Line {
				found = append(found, r.Target)
			}
		}
		if len(found) != 1 || !cleanPath(found[0]) {
			return "", false
		}
		return found[0], true
	}
	text := unquote(l.Text)
	if strings.ContainsAny(text, " \t") {
		return "", false
	}
	return ProjectPath(text)
}

// SpecDoc is the document of the spec a report's place names, below the
// spec folder, and whether it names one.
func (r *Report) SpecDoc(p *Project) (string, bool) {
	rel, ok := p.specFile(r.Place)
	if !ok || !lang.IsDocumentName(path.Base(rel)) {
		return "", false
	}
	_, ok = p.Spec.Document(rel)
	return rel, ok
}

// readSample reads the values of the result of a control sample, and checks
// that every planted document has a verdict.
func (f *Factory) readSample(d *lang.Document) *Sample {
	s := &Sample{Doc: d, Path: d.Path, Name: strings.TrimSuffix(path.Base(d.Path), ".md"), Verdicts: map[string]string{}}
	bad := func(l lang.SourceLine, format string, args ...any) {
		f.add(d.Path, l.Line, CodeSampleForm, format, args...)
	}
	if vs, _ := values(d, "Source"); len(vs) > 0 {
		if m := sampleForm.FindStringSubmatch(vs[0].Text); m != nil && day(m[1]) {
			s.Day = m[1]
		} else {
			bad(vs[0], "source: write job control-sample sampler, then a comma and the day as YYYY-MM-DD")
		}
	}
	type planted struct {
		doc  string
		line lang.SourceLine
	}
	var plants []planted
	seen := map[string]bool{}
	vs, _ := values(d, "Planted")
	for _, l := range vs {
		doc, deviation, ok := pathValue(l.Text)
		if !ok || deviation == "" {
			bad(l, "planted: write PATH: DEVIATION, the path that of a document below the spec folder")
			continue
		}
		plants = append(plants, planted{doc, l})
		if !seen[doc] {
			seen[doc] = true
			s.Planted = append(s.Planted, doc)
		}
	}
	vs, _ = values(d, "Verdicts")
	for _, l := range vs {
		doc, verdict, ok := pathValue(l.Text)
		switch {
		case !ok || verdict != VerdictOK && verdict != VerdictFail && verdict != VerdictSkip:
			bad(l, "verdicts: write PATH: VERDICT, the path that of a document below the spec folder, the verdict ok, fail or skip")
		case s.Verdicts[doc] != "":
			bad(l, "verdicts: %s has a verdict already", doc)
		default:
			s.Verdicts[doc] = verdict
		}
	}
	reported := map[string]bool{}
	for _, pl := range plants {
		if s.Verdicts[pl.doc] == "" && !reported[pl.doc] {
			reported[pl.doc] = true
			f.add(d.Path, pl.line.Line, CodeSampleVerdict, "%s was planted and has no line in Verdicts", pl.doc)
		}
	}
	return s
}

// pathValue splits PATH: VALUE, the path that of a document below the spec
// folder.
func pathValue(text string) (doc, value string, ok bool) {
	doc, value, ok = strings.Cut(text, ": ")
	doc = unquote(strings.TrimSpace(doc))
	if !ok || !cleanPath(doc) || !lang.IsDocumentName(path.Base(doc)) {
		return "", "", false
	}
	return doc, strings.TrimSpace(value), true
}

var (
	roleLine = regexp.MustCompile(`^([a-z]+(?:-[a-z]+)*): (\S.*)$`)
	jobLine  = regexp.MustCompile(`^([a-z0-9]+(?:-[a-z0-9]+)*): (\S.*)$`)
)

// readSettings reads the values of the settings.
func (f *Factory) readSettings(p *Project, d *lang.Document) *Settings {
	s := &Settings{Doc: d, Models: map[string]string{}, Cadence: map[string]string{}}
	bad := func(l lang.SourceLine, format string, args ...any) {
		f.add(d.Path, l.Line, CodeSettingsForm, format, args...)
	}
	vs, _ := values(d, "Models")
	for _, l := range vs {
		m := roleLine.FindStringSubmatch(l.Text)
		model := ""
		if m != nil {
			model = unquote(strings.TrimSpace(m[2]))
		}
		switch {
		case m == nil || strings.ContainsAny(model, " \t") || model == "":
			bad(l, "models: write ROLE: MODEL")
		case !oneOf(m[1], Roles):
			bad(l, "models: %s is no role; the roles are %s", m[1], strings.Join(Roles, ", "))
		case s.Models[m[1]] != "":
			bad(l, "models: %s has a model already", m[1])
		default:
			s.Models[m[1]] = model
		}
	}
	vs, _ = values(d, "Counter-checks")
	if len(vs) > 1 {
		bad(vs[1], "counter-checks: one folder, on one line")
	}
	if len(vs) > 0 {
		folder, ok := ProjectPath(strings.TrimSuffix(unquote(vs[0].Text), "/"))
		switch {
		case !ok || strings.ContainsAny(folder, " \t"):
			bad(vs[0], "counter-checks: write one folder below the project folder")
		case reserved(folder):
			bad(vs[0], "counter-checks: %s lies in %s/ or a .git folder", folder, Dir)
		case folder == p.SpecFolder || strings.HasPrefix(folder, p.SpecFolder+"/"):
			bad(vs[0], "counter-checks: %s lies in the spec folder; counter-checks are files of the project outside it", folder)
		default:
			s.CounterChecks = folder
		}
	}
	vs, _ = values(d, "Cadence")
	for _, l := range vs {
		m := jobLine.FindStringSubmatch(l.Text)
		switch {
		case m == nil:
			bad(l, "cadence: write JOB: TEXT, the job's name lowercase words joined by hyphens")
		case s.Cadence[m[1]] != "":
			bad(l, "cadence: %s has a cadence already", m[1])
		default:
			s.Cadence[m[1]] = strings.TrimSpace(m[2])
		}
	}
	vs, _ = values(d, "Write table")
	for _, l := range vs {
		m := roleLine.FindStringSubmatch(l.Text)
		switch {
		case m == nil:
			bad(l, "write table: write ROLE: PATTERN")
		case !oneOf(m[1], Roles):
			bad(l, "write table: %s is no role; the roles are %s", m[1], strings.Join(Roles, ", "))
		default:
			pattern := unquote(strings.TrimSpace(m[2]))
			if msg := patternProblem(pattern); msg != "" {
				bad(l, "write table: %s: %s", pattern, msg)
				continue
			}
			s.WriteTable = append(s.WriteTable, WriteRule{m[1], pattern})
		}
	}
	return s
}

// reserved says whether a path lies in .markitect/ or a .git folder, in any
// case.
func reserved(p string) bool {
	for i, part := range strings.Split(p, "/") {
		if i == 0 && strings.EqualFold(part, Dir) || strings.EqualFold(part, ".git") {
			return true
		}
	}
	return false
}

// posixClasses are the classes a bracket expression may name.
var posixClasses = []string{"alnum", "alpha", "blank", "cntrl", "digit", "graph", "lower", "print", "punct", "space", "upper", "xdigit"}

// patternProblem says why a pattern of the write table does not parse, ""
// when it does: it is a line of a .gitignore file that names something, with
// every bracket closed, every class known and no backslash at its end.
func patternProblem(p string) string {
	body := strings.TrimPrefix(p, "!")
	body = strings.TrimSuffix(strings.TrimPrefix(body, "/"), "/")
	if body == "" || strings.HasPrefix(p, "#") {
		return "names nothing"
	}
	for i := 0; i < len(body); i++ {
		switch body[i] {
		case '\\':
			if i++; i == len(body) {
				return "ends in a backslash"
			}
		case '[':
			j := i + 1
			if j < len(body) && (body[j] == '!' || body[j] == '^') {
				j++
			}
			if j < len(body) && body[j] == ']' {
				j++
			}
			for ; j < len(body) && body[j] != ']'; j++ {
				if strings.HasPrefix(body[j:], "[:") {
					end := strings.Index(body[j+2:], ":]")
					if end < 0 {
						return "a class is not closed"
					}
					if class := body[j+2 : j+2+end]; !oneOf(class, posixClasses) {
						return "no class [:" + class + ":]"
					}
					j += 2 + end + 1
				}
			}
			if j >= len(body) {
				return "a [ is not closed"
			}
			i = j
		}
	}
	return ""
}
