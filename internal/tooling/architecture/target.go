package architecture

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strings"
)

// Current explicit responsibility roots, most specific first. Every actual Go
// package additionally needs its own checked code-map entry. There is no broad
// internal/tooling or adapter import exemption and no legacy product allowance.
var targetRoots = []struct{ path, layer string }{
	{"internal/tooling/modules/pipelines", "maintenance"},
	{"internal/tooling/modules/githooks", "maintenance"},
	{"cmd/markitect-check-architecture", "maintenance-command"},
	{"internal/adapters/codexappserver", "apply-adapter"},
	{"internal/adapters/claudecode", "apply-adapter"},
	{"internal/adapters/container", "apply-adapter"},
	{"internal/tooling/architecturecli", "maintenance"},
	{"internal/tooling/release/install", "maintenance"},
	{"internal/tooling/architecture", "maintenance"},
	{"internal/tooling/guardedwrite", "maintenance"},
	{"internal/tooling/modulechecks", "maintenance"},
	{"cmd/markitect-check-modules", "maintenance-command"},
	{"internal/adapters/documents", "apply-adapter"},
	{"internal/tooling/releasecli", "maintenance"},
	{"internal/tooling/modulecli", "maintenance"},
	{"internal/adapters/gitrepo", "apply-adapter"},
	{"internal/tooling/licenses", "build-data"},
	{"internal/adapters/checks", "apply-adapter"},
	{"internal/agentd/frame", "agent-protocol"},
	{"cmd/markitect-agentd", "runtime-command"},
	{"internal/tooling/publish", "maintenance"},
	{"internal/tooling/release", "maintenance"},
	{"internal/tooling/source", "maintenance"},
	{"cmd/markitect-release", "maintenance-command"},
	{"internal/adapters/cli", "apply-adapter"},
	{"internal/organization", "apply-organization"},
	{"internal/realization", "apply-realization"},
	{"internal/bootstrap", "apply-bootstrap"},
	{"internal/buildinfo", "build-data"},
	{"internal/modeldocs", "model-view"},
	{"internal/snapshot", "apply-snapshot"},
	{"internal/agentd", "agent-runtime"},
	{"internal/testkit", "test-support"},
	{"internal/run", "apply-run"},
	{"internal/apply", "apply-usecase"},
	{"internal/model", "apply-model"},
	{"cmd/markitect", "apply-command"},
	{"integration", "standalone"},
	{"tools/smoke", "standalone"},
}

func targetLayer(p string) (string, string) {
	for _, r := range targetRoots {
		if p == r.path || strings.HasPrefix(p, r.path+"/") {
			return r.layer, r.path
		}
	}
	return "", ""
}

var responsibilityImports = map[string][]string{
	"cmd/markitect":                    {"internal/bootstrap"},
	"cmd/markitect-agentd":             {"internal/agentd"},
	"cmd/markitect-check-architecture": {"internal/tooling/architecturecli"},
	"cmd/markitect-check-modules":      {"internal/tooling/modulecli"},
	"cmd/markitect-release":            {"internal/tooling/releasecli"},
	"internal/adapters/checks":         {"internal/apply", "internal/model", "internal/organization", "internal/snapshot", "internal/run", "internal/adapters/container"},
	"internal/adapters/cli":            {"internal/apply", "internal/model", "internal/organization", "internal/realization", "internal/run", "internal/snapshot", "internal/modeldocs", "internal/buildinfo", "internal/tooling/licenses"},
	"internal/adapters/codexappserver": {"internal/apply", "internal/model", "internal/organization", "internal/snapshot", "internal/adapters/container", "internal/run", "internal/agentd/frame"},
	"internal/adapters/claudecode":     {"internal/apply", "internal/model", "internal/organization", "internal/snapshot", "internal/run", "internal/adapters/container", "internal/agentd/frame"},
	"internal/adapters/container":      {"internal/agentd/frame"},
	"internal/adapters/documents":      {"internal/modeldocs", "internal/tooling/guardedwrite"},
	"internal/adapters/gitrepo":        {"internal/apply", "internal/model", "internal/organization", "internal/realization", "internal/snapshot", "internal/run", "internal/adapters/container"},
	"internal/agentd":                  {"internal/agentd/frame"},
	"internal/apply":                   {"internal/model", "internal/snapshot", "internal/organization", "internal/realization", "internal/run"},
	"internal/bootstrap":               {"internal/adapters/checks", "internal/adapters/claudecode", "internal/adapters/cli", "internal/adapters/codexappserver", "internal/adapters/container", "internal/adapters/documents", "internal/adapters/gitrepo", "internal/apply", "internal/model", "internal/modeldocs", "internal/run", "internal/snapshot"},
	"internal/modeldocs":               {"internal/apply", "internal/model", "internal/snapshot"},
	"internal/organization":            {"internal/model", "internal/snapshot"},
	"internal/realization":             {"internal/model", "internal/snapshot"},
	"internal/run":                     {"internal/snapshot"},
	"internal/tooling/architecturecli": {"internal/tooling/architecture"},
	"internal/tooling/guardedwrite":    {"internal/tooling/source"},
	"internal/tooling/modulechecks":    {"internal/adapters/gitrepo", "internal/apply", "internal/model", "internal/snapshot", "internal/tooling/modules/githooks", "internal/tooling/modules/pipelines"},
	"internal/tooling/modulecli":       {"internal/tooling/modulechecks"},
	"internal/tooling/publish":         {"internal/tooling/source", "internal/tooling/release"},
	"internal/tooling/release":         {"internal/tooling/source", "internal/tooling/licenses"},
	"internal/tooling/release/install": {"internal/tooling/source", "internal/tooling/release", "internal/tooling/guardedwrite"},
	"internal/tooling/releasecli":      {"internal/buildinfo", "internal/tooling/source", "internal/tooling/licenses", "internal/tooling/release", "internal/tooling/release/install", "internal/tooling/publish"},
}

// Only concrete fixture and composition needs receive test-only edges;
// production keeps its separately declared responsibility imports.
var responsibilityTestImports = map[string][]string{
	"internal/adapters/checks": {"internal/adapters/gitrepo"},
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
