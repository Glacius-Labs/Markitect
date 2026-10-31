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

// Table is the repository's one architecture rule table. The import gate, the
// generated code map and the source packager's roots all derive from it.
var Table = Rules{
	Layers: []Layer{
		{Name: "model", Title: "Canonical model", Meaning: "The model language over supplied bytes: Definitions, Kinds, links, visibility, cycles, manifest and lock."},
		{Name: "assets", Title: "Built-in files", Meaning: "Files built into the binary: the standard Libraries and the skills."},
		{Name: "domain", Title: "Realization record", Meaning: "Registry pages, ownership and file accounting over a compiled model.", Imports: []string{"model"}},
		{Name: "project", Title: "Project", Meaning: "Loads and checks a project's Markitect folder from a repository state.", Imports: []string{"model", "assets", "domain"}},
		{Name: "service", Title: "Analysis", Meaning: "Questions about a loaded project: references, graph, impact with checklists, and lint.", Imports: []string{"model", "assets", "domain", "project"}},
		{Name: "remote", Title: "Run remote rules", Meaning: "A Run's plan, its verdict trailers and the decision of the remote's hooks.", Imports: []string{"model", "assets", "domain", "project", "service"}},
		{Name: "agent-port", Title: "Agent port", Meaning: "How the Run engine starts a maker or checker for a Manager and observes how it ends."},
		{Name: "agent-fake", Title: "Scripted agents", Meaning: "Scripted makers and checkers that work on a Run's remote with real Git, for tests.", Imports: []string{"agent-port"}, TestOnly: true},
		{Name: "run", Title: "Run engine", Meaning: "One Run per change: who works, makers and checkers along the tree, the Result and landing.", Imports: []string{"model", "assets", "domain", "project", "service", "remote", "agent-port"}},
		{Name: "protocol", Title: "Agent protocol", Meaning: "Frame protocol between the Orchestrator and the session helper."},
		{Name: "container", Title: "Container client", Meaning: "The container engine client that the adapters running containers share.", Imports: []string{"protocol"}},
		{Name: "harness", Title: "Manager harness", Meaning: "What a maker or checker receives: its role, the model it knows, its task and the tool contract.", Imports: []string{"model", "assets", "domain", "project", "service", "remote", "agent-port"}},
		{Name: "agents", Title: "Manager agents", Meaning: "One maker or checker in its container: login, clone, provider turns and the Manager tools, with the Claude Code and Codex adapters.", Imports: []string{"model", "assets", "domain", "project", "remote", "agent-port", "protocol", "container", "harness", "agents"}},
		{Name: "guardedwrite", Title: "Guarded write", Meaning: "Guarded working-tree writes for bundle installation, with the Git and path helpers they share with source snapshots."},
		{Name: "build", Title: "Build data", Meaning: "Version identity and embedded license notices."},
		{Name: "adapter", Title: "Adapters", Meaning: "Concrete Git, command-line, desktop and Run remote adapters; no adapter imports another.", Imports: []string{"model", "assets", "domain", "project", "service", "remote", "agent-port", "run", "build"}},
		{Name: "composition", Title: "Composition", Meaning: "Constructs the command line with its adapters.", Imports: []string{"model", "project", "service", "remote", "agent-port", "container", "agents", "run", "adapter"}},
		{Name: "command", Title: "Product commands", Meaning: "Thin entrypoints to product composition.", Imports: []string{"composition"}},
		{Name: "desktop-command", Title: "Desktop app", Meaning: "Markitect Desktop's native shell over the composition: windows, tray and folder dialogs; built only with the native tag.", Imports: []string{"composition"}},
		{Name: "remote-command", Title: "Run remote command", Meaning: "Thin entrypoint of a Run's remote container.", Imports: []string{"adapter"}},
		{Name: "agent", Title: "Session helper", Meaning: "Runs inside each Agent container: provider process, MCP bridge and stdio framing.", Imports: []string{"protocol"}},
		{Name: "agent-command", Title: "Session helper command", Meaning: "Thin entrypoint of the session helper.", Imports: []string{"agent"}},
		{Name: "maintenance", Title: "Maintenance", Meaning: "The import gate and release packaging, separate from product execution.", Imports: []string{"guardedwrite"}},
		{Name: "maintenance-command", Title: "Maintenance command", Meaning: "Release maintenance entrypoint.", Imports: []string{"maintenance", "build"}},
		{Name: "test-support", Title: "Test support", Meaning: "Hermetic fixtures.", TestOnly: true, Standalone: true},
		{Name: "standalone", Title: "Standalone tools", Meaning: "Consumer bootstrap and source smoke.", Standalone: true},
	},
	Packages: []Package{
		{"internal/model", "model", "docs/guides/model-language.md#definitions"},
		{"internal/libraries", "assets", "docs/guides/model-language.md#kinds-and-libraries"},
		{"internal/skills", "assets", "docs/guides/model-language.md#commands"},
		{"internal/realization", "domain", "docs/guides/model-language.md#the-realization-record"},
		{"internal/project", "project", "docs/guides/model-language.md#the-projects-markitect-folder"},
		{"internal/analysis", "service", "docs/guides/model-language.md#impact-and-checklists"},
		{"internal/runremote", "remote", "internal/runremote/README.md"},
		{"internal/agentport", "agent-port", "internal/agentport/README.md"},
		{"internal/agentport/agentfake", "agent-fake", "internal/agentport/README.md"},
		{"internal/run", "run", "internal/run/README.md"},
		{"internal/agentd/frame", "protocol", "internal/adapters/container/README.md"},
		{"internal/adapters/container", "container", "internal/adapters/container/README.md"},
		{"internal/harness", "harness", "internal/agents/README.md#harness"},
		{"internal/agents", "agents", "internal/agents/README.md"},
		{"internal/agents/claude", "agents", "internal/agents/README.md#claude-code"},
		{"internal/agents/codex", "agents", "internal/agents/README.md#codex"},
		{"internal/guardedwrite", "guardedwrite", "docs/architecture.md#distribution"},
		{"internal/buildinfo", "build", "docs/architecture.md#distribution"},
		{"internal/adapters/cli", "adapter", "docs/guides/model-language.md#commands"},
		{"internal/adapters/desktop", "adapter", "internal/adapters/desktop/README.md"},
		{"internal/adapters/remotecontainer", "adapter", "internal/runremote/README.md#the-container"},
		{"internal/adapters/remoteserver", "adapter", "internal/runremote/README.md"},
		{"internal/adapters/repo", "adapter", "docs/guides/model-language.md#the-projects-markitect-folder"},
		{"internal/adapters/rungit", "adapter", "internal/run/README.md#host-git"},
		{"internal/bootstrap", "composition", "docs/architecture.md#product-flow"},
		{"cmd/markitect", "command", "docs/guides/model-language.md#commands"},
		{"cmd/markitect-desktop", "desktop-command", "internal/adapters/desktop/README.md#the-native-app"},
		{"cmd/markitect-remote", "remote-command", "internal/runremote/README.md"},
		{"internal/agentd", "agent", "internal/adapters/container/README.md"},
		{"cmd/markitect-agentd", "agent-command", "internal/adapters/container/README.md"},
		{"internal/tooling/architecture", "maintenance", "docs/architecture.md#layers"},
		{"internal/tooling/release", "maintenance", "docs/architecture.md#distribution"},
		{"cmd/markitect-release", "maintenance-command", "docs/architecture.md#distribution"},
		{"internal/testkit", "test-support", "docs/architecture.md#layers"},
		{"packaging/bootstrap", "standalone", "packaging/README.md"},
		{"tools/smoke", "standalone", "docs/architecture.md#distribution"},
	},
	Exceptions: []Exception{
		{From: "internal/tooling/release", To: "internal/tooling/architecture", Reason: "The packager takes its source roots from this table."},
		{From: "internal/tooling/release", To: "internal/buildinfo", TestOnly: true, Reason: "Tests compare the packaged notices with the embedded ones."},
		{From: "internal/agents", To: "internal/adapters/remotecontainer", TestOnly: true, Reason: "The real-engine test starts real Managers against a real Run remote."},
		{From: "internal/run", To: "internal/adapters/rungit", TestOnly: true, Reason: "The engine's scripted Runs use the host's real Git."},
		{From: "internal/run", To: "internal/adapters/remoteserver", TestOnly: true, Reason: "The engine's scripted Runs push through the Run remote's real hooks."},
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
// duplicate names, a layer importing a layer listed after it, unordered
// packages, standalone layers with imports and exceptions that name unknown
// packages or that the layers already allow.
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
			case order[to] > order[l.Name]:
				report("layer %q: may import only itself and layers listed before it, not %q", l.Name, to)
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
