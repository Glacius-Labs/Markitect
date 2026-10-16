package architecture

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// Target responsibilities are separate from the current src/ layers. Each
// actual package still needs an exact entry in codemap.yaml; these roots do
// not assert that a package has already been implemented.
var targetRoots = []struct{ path, layer string }{
	{"internal/model", "apply-model"},
	{"internal/snapshot", "apply-snapshot"},
	{"internal/organization", "apply-organization"},
	{"internal/realization", "apply-realization"},
	{"internal/run", "apply-run"},
	{"internal/apply", "apply-usecase"},
	{"internal/adapters/cli", "apply-adapter"},
	{"internal/adapters/mcp", "apply-adapter"},
	{"internal/adapters/gitrepo", "apply-adapter"},
	{"internal/adapters/journal", "apply-adapter"},
	{"internal/adapters/codexappserver", "apply-adapter"},
	{"internal/adapters/checks", "apply-adapter"},
	{"internal/adapters/process", "apply-process"},
	{"internal/bootstrap", "apply-bootstrap"},
	{"cmd/markitect", "apply-command"},
}

func targetLayer(p string) (string, string) {
	for _, root := range targetRoots {
		if p == root.path || strings.HasPrefix(p, root.path+"/") {
			return root.layer, root.path
		}
	}
	return "", ""
}

// Cross-responsibility edges only. Packages within one responsibility may
// use its private subpackages. Process mechanics have only two consumers.
var targetImports = map[string][]string{
	"apply-model":        {},
	"apply-snapshot":     {},
	"apply-organization": {"apply-model", "apply-snapshot"},
	"apply-realization":  {"apply-model", "apply-snapshot"},
	"apply-run":          {"apply-model", "apply-snapshot", "apply-organization", "apply-realization"},
	"apply-usecase":      {"apply-model", "apply-snapshot", "apply-organization", "apply-realization", "apply-run"},
	"apply-adapter":      {"apply-model", "apply-snapshot", "apply-organization", "apply-realization", "apply-run", "apply-usecase"},
	"apply-process":      {},
	"apply-bootstrap":    {"apply-model", "apply-snapshot", "apply-organization", "apply-realization", "apply-run", "apply-usecase", "apply-adapter", "apply-process"},
	"apply-command":      {"apply-bootstrap"},
}

func targetRule(e Edge) string {
	from, owner := targetLayer(e.From)
	to, target := targetLayer(e.To)
	if from == "" || to == "" {
		return "target and current packages may not import each other"
	}
	if e.From == e.To && !e.Test {
		return "self-import is forbidden"
	}
	if owner == target || contains(targetImports[from], to) {
		return ""
	}
	if to == "apply-process" && (owner == "internal/adapters/codexappserver" || owner == "internal/adapters/checks") {
		return ""
	}
	return "Apply responsibility direction forbids " + from + " -> " + to
}

// Edge.Test alone cannot distinguish internal from external Go tests.
// CheckRepository verifies the package clause before admitting a test's
// import of its own directory; ordinary test imports keep the same policy.
func checkTargetSelfImports(root string, edges []Edge) ([]Violation, error) {
	var out []Violation
	for _, e := range edges {
		if target, _ := targetLayer(e.From); target == "" || !e.Test || e.From != e.To {
			continue
		}
		f, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, filepath.FromSlash(e.File)), nil, parser.PackageClauseOnly)
		if err != nil {
			return nil, err
		}
		if !strings.HasSuffix(f.Name.Name, "_test") {
			out = append(out, Violation{e, "only external test packages may import their own package"})
		}
	}
	return out, nil
}
