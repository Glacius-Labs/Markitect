package analysis

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// The thresholds of vet. They are provisional, like the code metrics they
// borrow from, and are tuned on real specs (DEC-102).
const (
	MaxEfferent        = 5    // Packages one Package may depend on
	MaxDefinitionWords = 1000 // words in one Definition's body
	MaxPackageSize     = 25   // own Definitions of one Package
	// A change to one Definition alone may put at most this share of the spec's
	// Definitions back on the list, and never fewer than MinHub, before vet.hub
	// warns.
	MaxHubShare = 0.2
	MinHub      = 10
	// MaxStatements is the number of list items and table rows one Definition may
	// hold before vet.statements notes it.
	MaxStatements = 20
	// Fan-in times revisions may reach this share of the spec's Definitions, and
	// never fewer than MinHotspot, before vet.hotspot notes it.
	MaxHotspotShare = 0.1
	MinHotspot      = 20
	// MaxRisk is the risk a verified Definition may have before vet.risk notes
	// it: the line CRAP draws for code. With c one more than the cases a reviewer
	// counted and e the share of them the evidence demonstrates, the risk is
	// c² × (1 − e)³ + c.
	MaxRisk = 30
)

// Report is what vet finds. Warnings are the measures that were precise on real
// specs: vet --strict exits 1 on them. Notes have a known blind spot or no
// calibration yet, and never change the exit code.
type Report struct {
	Warnings spec.Diagnostics
	Notes    []Note
}

// Note is one note of vet.
type Note struct {
	spec.Diagnostic
	// Targets and Structure belong to vet.unlinked_mention: the Definitions named
	// without a link, and where a link would go: "" within the Package structure,
	// StructureAgainst or StructureNew.
	Targets   []string
	Structure string
}

// History is what the record says about the spec's past. vet reads no Git
// (DEC-097), so this is all it knows of change.
type History struct {
	// Revisions counts, per Definition, how often its own text changed between
	// two verdicts on it.
	Revisions map[string]int
	// Counts holds, per Definition with an ok or a skip at its text as it is,
	// what the reviewer counted with it.
	Counts map[string]Count
}

// Count is a reviewer's count for one Definition: the cases the Definition
// demands and how many of them the evidence demonstrates. Skip says that the
// verdict was a skip, which demonstrates none.
type Count struct {
	Cases, Covered int
	Skip           bool
}

// Risk is the change risk of a Definition verified with these counts, after
// CRAP: large and thinly demonstrated is a risk, and so is very large.
func (c Count) Risk() float64 {
	comp := float64(1 + c.Cases)
	uncovered := 0.0
	switch {
	case c.Skip:
		uncovered = 1
	case c.Cases > 0:
		uncovered = math.Max(0, 1-float64(c.Covered)/float64(c.Cases))
	}
	return comp*comp*uncovered*uncovered*uncovered + comp
}

// Vet analyses the design of a loaded project like code: coupling, size, hubs
// and, as notes, unlinked mentions, statements, placement, hotspots and risk.
func Vet(p *project.Project, h History) Report {
	var r Report
	sp := p.Spec
	g := BuildGraph(p)
	for _, n := range g.Packages {
		pkg := sp.Packages[n.Path]
		file := packageFile(n.Path)
		if n.Efferent > MaxEfferent {
			r.Warnings.Add("vet.coupling", file, 0, "%s depends on %d Packages (more than %d); it may do too much or sit too low", spec.PackageName(n.Path), n.Efferent, MaxEfferent)
		}
		// The parts of a thing do not count: a Package is large by its things.
		things := 0
		for _, id := range pkg.Definitions {
			if sp.Definitions[id].Whole == "" {
				things++
			}
		}
		// Root holds what binds the whole project. In a spec with Packages that
		// cannot be split off, so only a spec without any is measured there.
		if things > MaxPackageSize && (n.Path != spec.RootPackage || len(sp.Packages) == 1) {
			r.Warnings.Add("vet.package_size", file, 0, "%s has %d Definitions of its own, parts not counted (more than %d); consider splitting it", spec.PackageName(n.Path), things, MaxPackageSize)
		}
		if n.Path != spec.RootPackage && len(pkg.Definitions) == 1 && len(pkg.Children) == 1 {
			r.Warnings.Add("vet.thin_package", file, 0, "%s holds only its anchor and one child, %s; it only adds a level", n.Path, pkg.Children[0])
		}
		// A Package owns the files its path patterns match; without any it owns
		// none, and Root owns what is built for it.
		// Until something is built the registry is empty, and the builder writes
		// the pages.
		if page := p.Registry.Pages[n.Path]; n.Path != spec.RootPackage && len(p.Registry.Entries) > 0 {
			switch {
			case page == nil:
				r.Warnings.Add("vet.no_paths", project.RegistryDir+"/"+n.Path+".md", 0, "%s has no registry page, so it owns no file; write the page with its first path patterns", n.Path)
			case len(page.Patterns) == 0:
				r.Warnings.Add("vet.no_paths", page.File, 0, "%s claims no path pattern, so it owns no file; name its first patterns under paths", n.Path)
			}
		}
	}
	for _, e := range g.Edges {
		from, to := g.node(e.From), g.node(e.To)
		if to.Instability > from.Instability {
			u := e.Uses[0]
			r.Warnings.Add("vet.unstable_dependency", u.File, u.Line, "%s (instability %.2f) depends on the less stable %s (instability %.2f); dependencies should point towards stability", spec.PackageName(e.From), from.Instability, spec.PackageName(e.To), to.Instability)
		}
	}
	ids := sp.OwnDefinitions()
	in := fanIn(sp)
	total := len(ids)
	hub := math.Max(MinHub, MaxHubShare*float64(total))
	hotspot := math.Max(MinHotspot, MaxHotspotShare*float64(total))
	var notes spec.Diagnostics
	for _, id := range ids {
		d := sp.Definitions[id]
		fan := len(in[id])
		if d.Words > MaxDefinitionWords {
			r.Warnings.Add("vet.definition_size", d.File, 0, "%d words (more than %d); split it or move detail the spec does not need to the realization", d.Words, MaxDefinitionWords)
		}
		// A Kind's change concerns all its Definitions by design, so a Kind is no
		// hub (DEC-102). A hub is counted by its links: what a Definition binds
		// by its place is declared, and no finding (DEC-113).
		if d.Kind != spec.CoreKind && float64(fan+1) > hub {
			r.Warnings.Add("vet.hub", d.File, 0, "%d Definitions link to %s or are of it as Kind, so a change to it alone puts %d of the spec's %d back on the list to verify (more than a fifth); link the smallest Definition that holds what they use, and keep this one small and settled", fan, id, fan+1, total)
		}
		if page, err := spec.ParsePage(d.Data); err == nil {
			if n := page.Statements(); n > MaxStatements {
				notes.Add("vet.statements", d.File, 0, "%d statements, list items and table rows (more than %d); a reviewer verifies each, so split it along its ideas", n, MaxStatements)
			}
		}
		if below := placement(sp, d, in[id]); below != "" {
			notes.Add("vet.placement", d.File, 0, "every Definition that links to %s lies in %s; it could move down there, unless it binds %s by scope", id, below, spec.PackageName(d.Package))
		}
		if c, ok := h.Counts[id]; ok && c.Risk() > MaxRisk {
			risk := math.Ceil(c.Risk())
			switch {
			case c.Cases+1 > MaxRisk:
				notes.Add("vet.risk", d.File, 0, "%d cases, too many to verify well however much evidence there is (risk %.0f, more than %d); split it along its ideas", c.Cases, risk, MaxRisk)
			case c.Skip:
				notes.Add("vet.risk", d.File, 0, "recorded skip with %d cases, and a skip counts none of them as demonstrated (risk %.0f, more than %d); find a way to show them, or split the Definition", c.Cases, risk, MaxRisk)
			default:
				notes.Add("vet.risk", d.File, 0, "its ok rests on evidence for %d of its %d cases (risk %.0f, more than %d); the next review demonstrates the others, or the Definition is split", c.Covered, c.Cases, risk, MaxRisk)
			}
		}
		if rev := h.Revisions[id]; rev > 0 && float64(fan*rev) > hotspot {
			linkers := fmt.Sprintf("%d Definitions link to it or are of it as Kind", fan)
			if fan == 1 {
				linkers = "1 Definition links to it or is of it as Kind"
			}
			notes.Add("vet.hotspot", d.File, 0, "its own text changed %d %s between verdicts, and %s, so its changes put about %d back on the list to verify, more than a tenth of the spec; split what changes from what relies on it", rev, plural(rev, "time", "times"), linkers, fan*rev)
		}
	}
	r.Warnings = r.Warnings.Sorted()
	for _, d := range notes {
		r.Notes = append(r.Notes, Note{Diagnostic: d})
	}
	r.Notes = append(r.Notes, unlinkedMentions(p)...)
	sort.SliceStable(r.Notes, func(i, j int) bool { return less(r.Notes[i].Diagnostic, r.Notes[j].Diagnostic) })
	return r
}

// placement returns the Package below a Definition's own into which it could move,
// because every Definition that links to it, or is of it as Kind, lies there: the
// lowest Package that holds them all, when that lies below the Definition's own.
// It returns "" otherwise, or when nothing links to it.
func placement(sp *spec.Spec, d *spec.Definition, linkers map[string]bool) string {
	common := ""
	first := true
	for id := range linkers {
		pkg := sp.Definitions[id].Package
		if pkg == d.Package || !spec.Within(pkg, d.Package) {
			return ""
		}
		if first {
			common, first = pkg, false
			continue
		}
		for !spec.Within(pkg, common) {
			common = sp.Parent(common)
		}
	}
	if first || common == d.Package {
		return ""
	}
	return common
}

// less orders diagnostics by file, line, code and message, as Sorted does.
func less(a, b spec.Diagnostic) bool {
	if a.File != b.File {
		return a.File < b.File
	}
	if a.Line != b.Line {
		return a.Line < b.Line
	}
	if a.Code != b.Code {
		return a.Code < b.Code
	}
	return a.Message < b.Message
}

// packageFile names a Package's folder for diagnostics.
func packageFile(pkg string) string {
	if pkg == spec.RootPackage {
		return project.SpecDir
	}
	return fmt.Sprintf("%s/%s", project.SpecDir, pkg)
}

func plural(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

// names joins names as prose: "a", "a and b", "a, b and c".
func names(list []string) string {
	switch len(list) {
	case 0:
		return ""
	case 1:
		return list[0]
	}
	return strings.Join(list[:len(list)-1], ", ") + " and " + list[len(list)-1]
}
