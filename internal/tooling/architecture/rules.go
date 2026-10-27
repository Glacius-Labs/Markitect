package architecture

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strings"
)

// Layer is one row of the layer list. A package may import a package of
// another layer only when its own layer lists that layer in Imports, and a
// package of its own layer only when Imports lists its own layer.
type Layer struct {
	Name    string
	Title   string // section heading in the code map
	Meaning string // one line for the code map
	Imports []string
	// TestOnly layers may be imported only by test files.
	TestOnly bool
	// Standalone layers import no Markitect package, not even from tests.
	Standalone bool
}

// Package assigns one Go package directory to a layer and names the document
// that owns its behavior. The code map takes its summary from the package doc
// comment.
type Package struct {
	Path  string // repository-relative directory with forward slashes
	Layer string
	Doc   string // repository-relative owning document, optionally with #anchor
}

// Exception admits one import edge that the layers do not allow. A TestOnly
// exception admits it only from test files.
type Exception struct {
	From, To string
	TestOnly bool
	Reason   string
}

// Rules is an architecture rule table: the layers, every package's layer and
// the named exceptions.
type Rules struct {
	Layers     []Layer
	Packages   []Package
	Exceptions []Exception
}

// The document writer may use only the reviewed guarded-write API.
const (
	GuardedWritePackage   = "internal/guardedwrite"
	DocumentWriterPackage = "internal/adapters/documents"
)

var guardedWriteAPI = map[string]bool{
	"OpenRoot": true, "CaptureFiles": true, "CaptureV1Files": true, "ApplyOrderedChecked": true, "Change": true, "Result": true,
}

// Table is the repository's one architecture rule table. The import gate, the
// generated code map and the source packager's roots all derive from it.
var Table = Rules{
	Layers: []Layer{
		{Name: "model", Title: "Canonical model", Meaning: "The canonical language check over supplied bytes."},
		{Name: "snapshot", Title: "Snapshots", Meaning: "Immutable repository tree values."},
		{Name: "domain", Title: "Organization and realization", Meaning: "Manager topology, ownership and accounting over model and snapshot values.", Imports: []string{"model", "snapshot"}},
		{Name: "project", Title: "Project", Meaning: "Checked structural preparation that every command starts from.", Imports: []string{"model", "snapshot", "domain"}},
		{Name: "service", Title: "Services", Meaning: "Run execution, static runtime metadata, harness instructions and the model view over a checked Project.", Imports: []string{"model", "snapshot", "domain", "project"}},
		{Name: "usecase", Title: "Apply", Meaning: "Command preflight, the sequence of closed Runs and stored-candidate publication.", Imports: []string{"model", "project", "service"}},
		{Name: "protocol", Title: "Agent protocol", Meaning: "Frame protocol between the Orchestrator and the session helper."},
		{Name: "container", Title: "Container client", Meaning: "The container engine client that the adapters running containers share.", Imports: []string{"protocol"}},
		{Name: "guardedwrite", Title: "Guarded write", Meaning: "Guarded working-tree writes for publication, docs --write and bundle installation, with the Git and path helpers they share with source snapshots."},
		{Name: "build", Title: "Build data", Meaning: "Version identity and embedded license notices."},
		{Name: "adapter", Title: "Adapters", Meaning: "Concrete repository, Session, check, record, runner, command-line and document adapters; no adapter imports another.", Imports: []string{"model", "snapshot", "domain", "project", "service", "usecase", "container", "guardedwrite", "build"}},
		{Name: "composition", Title: "Composition", Meaning: "Constructs the actual services and their adapters.", Imports: []string{"model", "project", "service", "usecase", "container", "guardedwrite", "adapter"}},
		{Name: "command", Title: "Product commands", Meaning: "Thin entrypoints to product composition.", Imports: []string{"composition"}},
		{Name: "agent", Title: "Session helper", Meaning: "Runs inside each Agent container: provider process, MCP bridge and stdio framing.", Imports: []string{"protocol"}},
		{Name: "agent-command", Title: "Session helper command", Meaning: "Thin entrypoint of the session helper.", Imports: []string{"agent"}},
		{Name: "maintenance", Title: "Maintenance", Meaning: "The import gate and release packaging, separate from product execution.", Imports: []string{"guardedwrite"}},
		{Name: "maintenance-command", Title: "Maintenance command", Meaning: "Release maintenance entrypoint.", Imports: []string{"maintenance", "build"}},
		{Name: "test-support", Title: "Test support", Meaning: "Hermetic fixtures.", TestOnly: true, Standalone: true},
		{Name: "standalone", Title: "Standalone tools", Meaning: "Consumer bootstrap and source smoke.", Standalone: true},
	},
	Packages: []Package{
		{"internal/model", "model", "docs/architecture.md#foundations-and-project"},
		{"internal/snapshot", "snapshot", "docs/architecture.md#foundations-and-project"},
		{"internal/organization", "domain", "docs/architecture.md#foundations-and-project"},
		{"internal/realization", "domain", "docs/architecture.md#foundations-and-project"},
		{"internal/project", "project", "internal/project/README.md"},
		{"internal/harness", "service", "internal/harness/README.md"},
		{"internal/modeldocs", "service", "docs/architecture.md#product-flow"},
		{"internal/run", "service", "internal/run/README.md"},
		{"internal/runtimeconfig", "service", "docs/architecture.md#foundations-and-project"},
		{"internal/apply", "usecase", "internal/apply/README.md"},
		{"internal/workflow", "usecase", "internal/apply/README.md"},
		{"internal/agentd/frame", "protocol", "internal/adapters/container/README.md"},
		{"internal/adapters/container", "container", "internal/adapters/container/README.md"},
		{"internal/guardedwrite", "guardedwrite", "docs/architecture.md#state-and-evidence-storage"},
		{"internal/buildinfo", "build", "docs/architecture.md#distribution"},
		{"internal/adapters/checks", "adapter", "internal/adapters/checks/README.md"},
		{"internal/adapters/cli", "adapter", "docs/architecture.md#adapters"},
		{"internal/adapters/codexappserver", "adapter", "internal/adapters/codexappserver/README.md"},
		{"internal/adapters/documents", "adapter", "docs/architecture.md#adapters"},
		{"internal/adapters/gitrepo", "adapter", "docs/architecture.md#adapters"},
		{"internal/adapters/runner", "adapter", "docs/design/linux-runner.md"},
		{"internal/adapters/runstore", "adapter", "internal/run/README.md"},
		{"internal/bootstrap", "composition", "docs/architecture.md#product-flow"},
		{"cmd/markitect", "command", "docs/architecture.md#product-flow"},
		{"cmd/markitect-runner", "command", "docs/design/linux-runner.md"},
		{"internal/agentd", "agent", "internal/adapters/container/README.md"},
		{"cmd/markitect-agentd", "agent-command", "internal/adapters/container/README.md"},
		{"internal/tooling/architecture", "maintenance", "docs/architecture.md#layers"},
		{"internal/tooling/release", "maintenance", "docs/architecture.md#distribution"},
		{"internal/tooling/source", "maintenance", "docs/architecture.md#distribution"},
		{"cmd/markitect-release", "maintenance-command", "docs/architecture.md#distribution"},
		{"internal/testkit", "test-support", "docs/architecture.md#layers"},
		{"packaging/bootstrap", "standalone", "packaging/README.md"},
		{"tools/smoke", "standalone", "docs/architecture.md#distribution"},
	},
	Exceptions: []Exception{
		{From: "cmd/markitect-runner", To: "internal/adapters/runner", Reason: "The runner image entrypoint starts the runner adapter directly."},
		{From: "internal/adapters/codexappserver", To: "internal/agentd/frame", Reason: "Codex Sessions use the frame values that the container client exposes."},
		{From: "internal/apply", To: "internal/workflow", Reason: "Apply runs the sequence of closed Runs."},
		{From: "internal/tooling/release", To: "internal/tooling/architecture", Reason: "The packager takes its source roots from this table."},
		{From: "internal/tooling/release", To: "internal/tooling/source", Reason: "Packaging and installation load fixed source snapshots."},
		{From: "internal/adapters/codexappserver", To: "internal/adapters/gitrepo", TestOnly: true, Reason: "Hierarchy tests run Codex Sessions over real Run Git copies."},
		{From: "internal/adapters/codexappserver", To: "internal/adapters/runstore", TestOnly: true, Reason: "Hierarchy tests read the stored Run records."},
		{From: "internal/adapters/gitrepo", To: "internal/adapters/checks", TestOnly: true, Reason: "The real-Git vertical tests compose workspaces with declared checks."},
		{From: "internal/adapters/gitrepo", To: "internal/adapters/runstore", TestOnly: true, Reason: "The real-Git vertical tests store and read back Run records."},
		{From: "internal/bootstrap", To: "internal/snapshot", TestOnly: true, Reason: "Composition tests build snapshot check values."},
		{From: "internal/tooling/release", To: "internal/buildinfo", TestOnly: true, Reason: "Tests compare the packaged notices with the embedded ones."},
	},
}

// SourceRoots returns the packages below cmd/ and internal/, the only roots a
// source archive may contain.
func (r Rules) SourceRoots() []string {
	var roots []string
	for _, p := range r.Packages {
		if strings.HasPrefix(p.Path, "cmd/") || strings.HasPrefix(p.Path, "internal/") {
			roots = append(roots, p.Path)
		}
	}
	return roots
}

func (r Rules) layer(name string) (Layer, bool) {
	for _, l := range r.Layers {
		if l.Name == name {
			return l, true
		}
	}
	return Layer{}, false
}

func (r Rules) pkg(p string) (Package, bool) {
	for _, candidate := range r.Packages {
		if candidate.Path == p {
			return candidate, true
		}
	}
	return Package{}, false
}

// allows reports whether a package of layer from may import one of layer to.
func (r Rules) allows(from, to string) bool {
	l, _ := r.layer(from)
	return contains(l.Imports, to)
}

func (r Rules) exception(from, to string, test bool) bool {
	for _, x := range r.Exceptions {
		if x.From == from && x.To == to && (test || !x.TestOnly) {
			return true
		}
	}
	return false
}

var layerName = regexp.MustCompile(`^[a-z][a-z-]*[a-z]$`)

// Validate reports every inconsistency of the table itself: unknown or
// duplicate names, unordered packages, standalone layers with imports and
// exceptions that name unknown packages or that the layers already allow.
func (r Rules) Validate() []string {
	var problems []string
	report := func(format string, args ...any) { problems = append(problems, fmt.Sprintf(format, args...)) }
	line := func(s string) bool { return strings.TrimSpace(s) != "" && !strings.ContainsAny(s, "|\n") }
	order := map[string]int{}
	for i, l := range r.Layers {
		if !layerName.MatchString(l.Name) {
			report("layer %q: name must be lower-case words joined by hyphens", l.Name)
		}
		if _, dup := order[l.Name]; dup {
			report("layer %q: duplicate", l.Name)
		}
		order[l.Name] = i
		if !line(l.Title) || !line(l.Meaning) {
			report("layer %q: title and meaning must be single non-empty lines without '|'", l.Name)
		}
		if l.Standalone && len(l.Imports) != 0 {
			report("layer %q: a standalone layer imports nothing", l.Name)
		}
	}
	for _, l := range r.Layers {
		seen := map[string]bool{}
		for _, to := range l.Imports {
			target, known := r.layer(to)
			switch {
			case !known:
				report("layer %q: imports unknown layer %q", l.Name, to)
			case seen[to]:
				report("layer %q: imports layer %q twice", l.Name, to)
			case target.TestOnly:
				report("layer %q: test-only layer %q needs no import rule", l.Name, to)
			}
			seen[to] = true
		}
	}
	seen := map[string]bool{}
	for i, p := range r.Packages {
		if p.Path == "" || p.Path != path.Clean(p.Path) || path.IsAbs(p.Path) || strings.HasPrefix(p.Path, "../") || strings.Contains(p.Path, "\\") {
			report("package %q: path must be a clean repository-relative path with forward slashes", p.Path)
		}
		if seen[p.Path] {
			report("package %q: duplicate", p.Path)
		}
		seen[p.Path] = true
		rank, known := order[p.Layer]
		if !known {
			report("package %q: unknown layer %q", p.Path, p.Layer)
		} else if i > 0 {
			prev := r.Packages[i-1]
			if prevRank, ok := order[prev.Layer]; ok && (prevRank > rank || prevRank == rank && prev.Path >= p.Path) {
				report("package %q: packages must be ordered by layer, then by path", p.Path)
			}
		}
		if strings.TrimSpace(p.Doc) == "" {
			report("package %q: owning document is missing", p.Path)
		}
	}
	pairs := map[[2]string]bool{}
	for _, x := range r.Exceptions {
		from, fromKnown := r.pkg(x.From)
		to, toKnown := r.pkg(x.To)
		name := x.From + " -> " + x.To
		switch {
		case !fromKnown || !toKnown:
			report("exception %s: names an unknown package", name)
		case x.From == x.To:
			report("exception %s: a package may not except itself", name)
		case pairs[[2]string{x.From, x.To}]:
			report("exception %s: duplicate", name)
		case r.allows(from.Layer, to.Layer):
			report("exception %s: layer %s already may import layer %s", name, from.Layer, to.Layer)
		default:
			fl, _ := r.layer(from.Layer)
			tl, _ := r.layer(to.Layer)
			if fl.Standalone || tl.TestOnly {
				report("exception %s: standalone and test-only layers take no exceptions", name)
			}
		}
		pairs[[2]string{x.From, x.To}] = true
		if !line(x.Reason) {
			report("exception %s: reason must be a single non-empty line without '|'", name)
		}
	}
	// A table that classifies only one package of the guarded-write rule would
	// silently check nothing; a partial fixture table may have neither.
	_, writer := r.pkg(GuardedWritePackage)
	_, user := r.pkg(DocumentWriterPackage)
	if writer != user {
		report("the guarded-write rule needs rows for both %s and %s", GuardedWritePackage, DocumentWriterPackage)
	}
	return problems
}

func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}

func sortViolations(out []Violation) {
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
}
