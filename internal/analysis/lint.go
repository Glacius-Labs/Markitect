package analysis

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The lint thresholds. They are provisional, like the code metrics they
// borrow from, and are tuned on real models.
const (
	MaxEfferent         = 5    // Packages one Package may depend on
	MaxDefinitionWords  = 1000 // words in one Definition's body
	MaxPackageSize      = 25   // own Definitions of one Package
	MaxReachShare       = 0.5  // share of all Managers one Definition's change may reach
	MinPackagesForReach = 6    // below this the reach warning stays quiet
)

// Lint measures a loaded project like code and returns warnings only. Change
// hotspots from Git history come later.
func Lint(p *project.Project) model.Diagnostics {
	var ds model.Diagnostics
	m := p.Model
	g := BuildGraph(p)
	for _, n := range g.Packages {
		pkg := m.Packages[n.Path]
		file := packageFile(n.Path)
		if n.Efferent > MaxEfferent {
			ds.Add("lint.coupling", file, 0, "%s depends on %d Packages (more than %d); it may do too much or sit too low", model.PackageName(n.Path), n.Efferent, MaxEfferent)
		}
		if len(pkg.Definitions) > MaxPackageSize {
			ds.Add("lint.package_size", file, 0, "%s has %d Definitions of its own (more than %d); consider splitting it", model.PackageName(n.Path), len(pkg.Definitions), MaxPackageSize)
		}
		if n.Path != model.RootPackage && len(pkg.Definitions) == 0 && len(pkg.Children) == 1 {
			ds.Add("lint.thin_package", file, 0, "%s has no Definitions of its own and only one child, %s; its Manager only passes work through", n.Path, pkg.Children[0])
		}
		// The hooks take a Manager's paths from the Run's base, so a Package
		// without patterns can change no file in its next Run.
		if page := p.Registry.Pages[n.Path]; n.Path != model.RootPackage && (page == nil || len(page.Patterns) == 0) {
			pageFile := project.RegistryDir + "/" + n.Path + ".md"
			if page != nil {
				pageFile = page.File
			}
			ds.Add("lint.no_paths", pageFile, 0, "%s claims no path pattern, so it owns no file and its Manager can change none; name its first patterns under paths", n.Path)
		}
	}
	for _, e := range g.Edges {
		from, to := g.node(e.From), g.node(e.To)
		if to.Instability > from.Instability {
			u := e.Uses[0]
			ds.Add("lint.unstable_dependency", u.File, u.Line, "%s (instability %.2f) depends on the less stable %s (instability %.2f); dependencies should point towards stability", model.PackageName(e.From), from.Instability, model.PackageName(e.To), to.Instability)
		}
	}
	incoming := map[string]bool{}
	for _, d := range m.Definitions {
		for _, l := range d.Links {
			if l.Target != d.ID {
				incoming[l.Target] = true
			}
		}
	}
	realized := map[string]bool{}
	for _, e := range p.Registry.Entries {
		for _, id := range e.Realizes {
			realized[id] = true
		}
	}
	total := len(m.Packages)
	for _, id := range m.ModelDefinitions() {
		d := m.Definitions[id]
		if d.Words > MaxDefinitionWords {
			ds.Add("lint.definition_size", d.File, 0, "%d words (more than %d); split it or move detail the model does not need to the realization", d.Words, MaxDefinitionWords)
		}
		if !incoming[id] && !realized[id] {
			ds.Add("lint.unused", d.File, 0, "nothing links to %s and no file realizes it", id)
		}
		if d.Package != model.RootPackage && total >= MinPackagesForReach {
			if n := len(reachOf(p, id)); float64(n) > MaxReachShare*float64(total) {
				ds.Add("lint.reach", d.File, 0, "a change to %s reaches %d of %d Managers; it is a broad dependency for its place", id, n, total)
			}
		}
	}
	for _, s := range p.Registry.Stale {
		switch s.Code {
		case "stale.definition":
			ds.Add("lint.stale", s.PageFile, s.Line, "%s links [[%s]], which no longer exists; %s must realign or remove it", s.File, s.Target, model.PackageName(s.Package))
		case "stale.file":
			ds.Add("lint.stale", s.PageFile, s.Line, "%s no longer exists; %s must remove its entry", s.File, model.PackageName(s.Package))
		case "stale.page":
			if s.File == "" {
				ds.Add("lint.stale", s.PageFile, 0, "the Package of this page no longer exists; Root must remove the page")
			} else {
				ds.Add("lint.stale", s.PageFile, s.Line, "%s is listed only on the page of a removed Package; its owner %s must register or remove it", s.File, model.PackageName(s.Package))
			}
		}
	}
	return ds.Sorted()
}

// reachOf returns the Managers that would get a checklist item or a reached
// file if Definition id alone changed, as ComputeImpact would find them. The
// parents that only integrate are not counted.
func reachOf(p *project.Project, id string) map[string]bool {
	m := p.Model
	d := m.Definitions[id]
	working := map[string]bool{}
	add := func(pkg string) { working[pkg] = true }
	for _, pkg := range m.Subtree(d.Package) {
		add(pkg) // its own Package and, through the inherited item, every one below
	}
	reached := map[string]bool{id: true}
	for _, other := range m.Definitions {
		if !other.InModel() {
			continue
		}
		for _, l := range other.Links {
			if l.Target == id {
				add(other.Package)
				reached[other.ID] = true
			}
		}
	}
	// The files that realize a reached Definition are reached too.
	for file, e := range p.Registry.Entries {
		for _, r := range e.Realizes {
			if reached[r] {
				add(p.Registry.Owner[file])
			}
		}
	}
	return working
}

// packageFile names a Package's folder for diagnostics.
func packageFile(pkg string) string {
	if pkg == model.RootPackage {
		return project.ModelDir
	}
	return fmt.Sprintf("%s/%s", project.ModelDir, pkg)
}
