package analysis

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// The thresholds of vet. They are provisional, like the code metrics they
// borrow from, and are tuned on real models.
const (
	MaxEfferent         = 5    // Packages one Package may depend on
	MaxDefinitionWords  = 1000 // words in one Definition's body
	MaxPackageSize      = 25   // own Definitions of one Package
	MaxReachShare       = 0.5  // share of all Packages one Definition's change may reach
	MinPackagesForReach = 6    // below this the reach warning stays quiet
)

// Vet analyses the design of a loaded project like code and returns warnings
// only: coupling, size, and how far a change reaches. Change hotspots from Git
// history come later.
func Vet(p *project.Project) model.Diagnostics {
	var ds model.Diagnostics
	m := p.Model
	g := BuildGraph(p)
	for _, n := range g.Packages {
		pkg := m.Packages[n.Path]
		file := packageFile(n.Path)
		if n.Efferent > MaxEfferent {
			ds.Add("vet.coupling", file, 0, "%s depends on %d Packages (more than %d); it may do too much or sit too low", model.PackageName(n.Path), n.Efferent, MaxEfferent)
		}
		if len(pkg.Definitions) > MaxPackageSize {
			ds.Add("vet.package_size", file, 0, "%s has %d Definitions of its own (more than %d); consider splitting it", model.PackageName(n.Path), len(pkg.Definitions), MaxPackageSize)
		}
		if n.Path != model.RootPackage && len(pkg.Definitions) == 0 && len(pkg.Children) == 1 {
			ds.Add("vet.thin_package", file, 0, "%s has no Definitions of its own and only one child, %s; it only adds a level", n.Path, pkg.Children[0])
		}
		// A Package owns the files its path patterns match; without any it owns
		// none, and Root owns what is built for it.
		if page := p.Registry.Pages[n.Path]; n.Path != model.RootPackage {
			switch {
			case page == nil:
				ds.Add("vet.no_paths", project.RegistryDir+"/"+n.Path+".md", 0, "%s has no registry page, so it owns no file; write the page with its first path patterns", n.Path)
			case len(page.Patterns) == 0:
				ds.Add("vet.no_paths", page.File, 0, "%s claims no path pattern, so it owns no file; name its first patterns under paths", n.Path)
			}
		}
	}
	for _, e := range g.Edges {
		from, to := g.node(e.From), g.node(e.To)
		if to.Instability > from.Instability {
			u := e.Uses[0]
			ds.Add("vet.unstable_dependency", u.File, u.Line, "%s (instability %.2f) depends on the less stable %s (instability %.2f); dependencies should point towards stability", model.PackageName(e.From), from.Instability, model.PackageName(e.To), to.Instability)
		}
	}
	total := len(m.Packages)
	for _, id := range m.ModelDefinitions() {
		d := m.Definitions[id]
		if d.Words > MaxDefinitionWords {
			ds.Add("vet.definition_size", d.File, 0, "%d words (more than %d); split it or move detail the model does not need to the realization", d.Words, MaxDefinitionWords)
		}
		if d.Package != model.RootPackage && total >= MinPackagesForReach {
			if n := len(reachOf(m, id)); float64(n) > MaxReachShare*float64(total) {
				ds.Add("vet.reach", d.File, 0, "a change to %s puts Definitions of %d of %d Packages back on the list to verify; it is a broad dependency for its place", id, n, total)
			}
		}
	}
	return ds.Sorted()
}

// reachOf returns the Packages that hold a Definition a change of Definition id
// alone puts back on the list to verify: its own, and those of the Definitions
// that link to it or are of it as their Kind (DEC-099).
func reachOf(m *model.Model, id string) map[string]bool {
	reached := map[string]bool{m.Definitions[id].Package: true}
	for _, other := range m.Definitions {
		if !other.InModel() {
			continue
		}
		for _, l := range other.Links {
			if l.Target == id && !l.Package {
				reached[other.Package] = true
			}
		}
	}
	return reached
}

// packageFile names a Package's folder for diagnostics.
func packageFile(pkg string) string {
	if pkg == model.RootPackage {
		return project.ModelDir
	}
	return fmt.Sprintf("%s/%s", project.ModelDir, pkg)
}
