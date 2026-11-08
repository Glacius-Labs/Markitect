package analysis

import (
	"math"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// Graph is the Package dependency graph with its measures.
type Graph struct {
	Packages []PackageNode `json:"packages"`
	Edges    []Edge        `json:"edges"`
}

// PackageNode is one Package with Robert Martin's coupling measures over its
// explicit edges: Ca, the Packages that depend on it; Ce, the Packages it
// depends on; and the instability Ce / (Ca + Ce), 0 when both are 0. The
// implicit edge from a child to its parent is not counted. Its abstractness is
// the share of its own Definitions that are Kinds, as Martin counts interfaces
// among classes, 0 without Definitions; its distance from the main sequence is
// |A + I - 1|.
type PackageNode struct {
	Path         string  `json:"path"`
	Parent       string  `json:"parent"`
	Definitions  int     `json:"definitions"`
	Afferent     int     `json:"afferent"`
	Efferent     int     `json:"efferent"`
	Instability  float64 `json:"instability"`
	Abstractness float64 `json:"abstractness"`
	Distance     float64 `json:"distance"`
}

// Edge is one explicit dependency and the links that make it.
type Edge struct {
	From string      `json:"from"`
	To   string      `json:"to"`
	Uses []model.Use `json:"uses"`
}

// BuildGraph derives the graph of a loaded project.
func BuildGraph(p *project.Project) Graph {
	m := p.Model
	var g Graph
	afferent := map[string]int{}
	for _, from := range sortedKeys(m.Edges) {
		for _, to := range sortedKeys(m.Edges[from]) {
			afferent[to]++
			g.Edges = append(g.Edges, Edge{From: from, To: to, Uses: m.Edges[from][to]})
		}
	}
	for _, path := range m.SortedPackages() {
		pkg := m.Packages[path]
		n := PackageNode{Path: path, Parent: pkg.Parent, Definitions: len(pkg.Definitions), Afferent: afferent[path], Efferent: len(m.Edges[path])}
		if path == model.RootPackage {
			n.Parent = ""
		}
		if total := n.Afferent + n.Efferent; total > 0 {
			n.Instability = float64(n.Efferent) / float64(total)
		}
		kinds := 0
		for _, id := range pkg.Definitions {
			if m.Definitions[id].Kind == model.CoreKind {
				kinds++
			}
		}
		if len(pkg.Definitions) > 0 {
			n.Abstractness = float64(kinds) / float64(len(pkg.Definitions))
		}
		n.Distance = math.Abs(n.Abstractness + n.Instability - 1)
		g.Packages = append(g.Packages, n)
	}
	return g
}

// fanIn returns, for each model Definition, the other model Definitions that link
// to it or are of it as their Kind: what a change to it alone puts back on the
// list to verify besides itself (DEC-099). A link to a Package is none.
func fanIn(m *model.Model) map[string]map[string]bool {
	in := map[string]map[string]bool{}
	for _, id := range m.ModelDefinitions() {
		for _, l := range m.Definitions[id].Links {
			if l.Package || l.Target == "" || l.Target == id {
				continue
			}
			if t := m.Definitions[l.Target]; t == nil || !t.InModel() {
				continue
			}
			if in[l.Target] == nil {
				in[l.Target] = map[string]bool{}
			}
			in[l.Target][id] = true
		}
	}
	return in
}

// node returns the measures of one Package.
func (g Graph) node(path string) PackageNode {
	for _, n := range g.Packages {
		if n.Path == path {
			return n
		}
	}
	return PackageNode{Path: path}
}
