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
	{"cmd/markitect-check-architecture", "maintenance-command"},
	{"internal/adapters/codexappserver", "apply-adapter"},
	{"internal/adapters/container", "apply-adapter"},
	{"internal/tooling/architecturecli", "maintenance"},
	{"internal/tooling/release/install", "maintenance"},
	{"internal/tooling/architecture", "maintenance"},
	{"internal/tooling/guardedwrite", "maintenance"},
	{"internal/adapters/documents", "apply-adapter"},
	{"internal/tooling/releasecli", "maintenance"},
	{"internal/adapters/gitrepo", "apply-adapter"},
	{"internal/tooling/licenses", "build-data"},
	{"internal/adapters/checks", "apply-adapter"},
	{"internal/agentd/frame", "agent-protocol"},
	{"cmd/markitect-agentd", "runtime-command"},
	{"cmd/markitect-runner", "apply-command"},
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
	{"internal/project", "apply-project"},
	{"internal/runtimeconfig", "runtime-config"},
	{"internal/adapters/runstore", "apply-adapter"},
	{"internal/adapters/runner", "apply-adapter"},
	{"internal/harness", "native-harness"},
	{"internal/apply", "apply-usecase"},
	{"internal/workflow", "apply-workflow"},
	{"internal/model", "apply-model"},
	{"cmd/markitect", "apply-command"},
	{"packaging/bootstrap", "standalone"},
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
	"internal/workflow":                {"internal/run"},
	"internal/apply":                   {"internal/model", "internal/project", "internal/run", "internal/workflow"},
	"internal/adapters/runstore":       {"internal/run"},
	"internal/runtimeconfig":           {"internal/model", "internal/organization", "internal/project", "internal/snapshot"},
	"internal/project":                 {"internal/model", "internal/organization", "internal/realization", "internal/snapshot"},
	"cmd/markitect":                    {"internal/bootstrap"},
	"cmd/markitect-agentd":             {"internal/agentd"},
	"cmd/markitect-runner":             {"internal/adapters/runner"},
	"cmd/markitect-check-architecture": {"internal/tooling/architecturecli"},
	"cmd/markitect-release":            {"internal/tooling/releasecli"},
	"internal/adapters/checks":         {"internal/adapters/container", "internal/model", "internal/organization", "internal/project", "internal/run", "internal/runtimeconfig", "internal/snapshot"},
	"internal/adapters/cli":            {"internal/apply", "internal/buildinfo", "internal/model", "internal/modeldocs", "internal/project", "internal/realization", "internal/runtimeconfig", "internal/snapshot", "internal/tooling/licenses"},
	"internal/adapters/codexappserver": {"internal/adapters/container", "internal/agentd/frame", "internal/harness", "internal/model", "internal/organization", "internal/project", "internal/run", "internal/runtimeconfig", "internal/snapshot"},
	"internal/adapters/container":      {"internal/agentd/frame"},
	"internal/adapters/documents":      {"internal/modeldocs", "internal/tooling/guardedwrite"},
	"internal/adapters/gitrepo":        {"internal/adapters/container", "internal/apply", "internal/model", "internal/project", "internal/run", "internal/snapshot", "internal/tooling/guardedwrite"},
	"internal/agentd":                  {"internal/agentd/frame"},
	"internal/harness":                 {"internal/model", "internal/organization", "internal/project"},
	"internal/bootstrap":               {"internal/adapters/checks", "internal/adapters/cli", "internal/adapters/codexappserver", "internal/adapters/container", "internal/adapters/documents", "internal/adapters/gitrepo", "internal/adapters/runner", "internal/adapters/runstore", "internal/apply", "internal/model", "internal/modeldocs", "internal/project", "internal/run", "internal/runtimeconfig", "internal/tooling/guardedwrite"},
	"internal/modeldocs":               {"internal/model", "internal/project", "internal/snapshot"},
	"internal/organization":            {"internal/model", "internal/snapshot"},
	"internal/realization":             {"internal/model", "internal/snapshot"},
	"internal/run":                     {"internal/organization", "internal/project", "internal/snapshot"},
	"internal/tooling/architecturecli": {"internal/tooling/architecture"},
	"internal/tooling/guardedwrite":    {"internal/tooling/source"},
	"internal/tooling/publish":         {"internal/tooling/source", "internal/tooling/release"},
	"internal/tooling/release":         {"internal/tooling/source", "internal/tooling/licenses"},
	"internal/tooling/release/install": {"internal/tooling/source", "internal/tooling/release", "internal/tooling/guardedwrite"},
	"internal/tooling/releasecli":      {"internal/buildinfo", "internal/tooling/source", "internal/tooling/licenses", "internal/tooling/release", "internal/tooling/release/install", "internal/tooling/publish"},
}

// Only concrete fixture and composition needs receive test-only edges;
// production keeps its separately declared responsibility imports.
var responsibilityTestImports = map[string][]string{
	"internal/bootstrap":               {"internal/snapshot"},
	"internal/adapters/codexappserver": {"internal/adapters/gitrepo", "internal/adapters/runstore"},
	"internal/adapters/gitrepo":        {"internal/adapters/runstore", "internal/adapters/checks", "internal/runtimeconfig", "internal/organization", "internal/realization"},
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
