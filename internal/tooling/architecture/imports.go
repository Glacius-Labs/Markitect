// Package architecture enforces repository import ownership without compiling
// the inspected code. It also inspects tests and all platform-specific files.
package architecture

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

const ModulePath = "github.com/Glacius-Labs/Markitect/"

type Edge struct {
	File     string
	Line     int
	From, To string
	Test     bool
}
type Violation struct {
	Edge Edge
	Rule string
}

func (v Violation) String() string {
	return fmt.Sprintf("%s:%d: %s imports %s: %s", v.Edge.File, v.Edge.Line, v.Edge.From, v.Edge.To, v.Rule)
}

// layer classifies every product package by its concrete owner, not an edge
// allowlist. Old paths retain their intended responsibility during migration
// and fail the same rules until they are removed.
func layer(p string) (string, string) {
	switch {
	case p == "internal/core" || strings.HasPrefix(p, "internal/core/"):
		return "core", "core"
	case p == "internal/host" || strings.HasPrefix(p, "internal/host/"):
		return "host", "host"
	case p == "internal/modules" || strings.HasPrefix(p, "internal/modules/"):
		parts := strings.Split(p, "/")
		if len(parts) >= 3 {
			return "module", strings.Join(parts[:3], "/")
		}
		return "unknown", p
	case p == "internal/infrastructure" || strings.HasPrefix(p, "internal/infrastructure/"):
		return "infrastructure", "infrastructure"
	case p == "internal/tooling" || strings.HasPrefix(p, "internal/tooling/") || p == "tools" || strings.HasPrefix(p, "tools/"):
		return "tooling", "tooling"
	case p == "cmd" || strings.HasPrefix(p, "cmd/"):
		return "cli", p
	// This isolated adopting-code fixture is compiled by Go, but is not a
	// Markitect capability. It cannot import any Markitect product package.
	case p == "examples/documentation/docs/implementation/src":
		return "fixture", p
	case p == "examples" || p == "integration" || p == "benchmark":
		return "harness-tests", p
	case p == "examples/engineering-discovery" || p == "examples/selective-adoption" || p == "examples/selective-adoption/pathspell" || p == "experiments/mcp-pilot":
		return "harness-runtime", p
	case p == "internal/app" || p == "internal/format" || p == "internal/inputs":
		return "host", "host"
	case p == "internal/snapshot":
		return "core", "core"
	case p == "internal/source":
		return "infrastructure", "infrastructure"
	case p == "internal/release" || p == "internal/publish" || p == "internal/licenses":
		return "tooling", "tooling"
	case p == "internal/render" || p == "internal/markdownlinks":
		return "module", "legacy-projections"
	case p == "internal/adoption" || p == "internal/copyme":
		return "module", "adoption"
	case p == "internal/authoring":
		return "module", "authoring"
	case p == "internal/artifactcoverage":
		return "module", "artifactcoverage"
	case p == "internal/contentpackage":
		return "host", "host"
	default:
		return "unknown", p
	}
}

func Check(edges []Edge) []Violation {
	var out []Violation
	for _, e := range edges {
		from, owner := layer(e.From)
		if e.To == "" {
			if from == "unknown" || from == "harness-tests" && !e.Test {
				out = append(out, Violation{e, "unclassified product package"})
			}
			continue
		}
		to, target := layer(e.To)
		rule := ""
		switch {
		case from == "fixture":
			rule = "adopting-code fixture may not import Markitect product packages"
		case e.From == e.To:
			rule = "self-import is forbidden"
		case strings.HasPrefix(to, "harness") && !strings.HasPrefix(from, "harness"):
			rule = "product may not import test/example Harness"
		case from == "harness-tests" && !e.Test:
			rule = "Harness root may contain only test code"
		case from == "harness-runtime" && to != "host" && !(strings.HasPrefix(to, "harness") && strings.HasPrefix(e.To, strings.TrimSuffix(e.From, "/pathspell"))):
			rule = "example/experiment runtime may import only Host and its own demonstration subtree"
		case from == "unknown" || to == "unknown":
			rule = "unclassified internal package"
		case from == "core" && to != "core":
			rule = "Core may import only Core"
		case from == "module" && to != "core" && !(to == "module" && owner == target):
			rule = "Module may import only Core and its own subtree"
		case from == "cli" && to != "host":
			rule = "CLI must delegate to Host"
		case from == "infrastructure" && (to == "module" || to == "host" || to == "cli" || to == "tooling"):
			rule = "Infrastructure may not import product composition"
		case from == "tooling" && (to == "module" || to == "host" || to == "cli"):
			rule = "Tooling may not import runtime composition"
		}
		if rule != "" {
			out = append(out, Violation{e, rule})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].String() < out[j].String() })
	return out
}

func Inspect(root string) ([]Edge, error) {
	var edges []Edge
	seenPackages := map[string]bool{}
	err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			if name != root && (entry.Name() == ".git" || entry.Name() == ".cache" || entry.Name() == ".artifacts" || entry.Name() == "vendor" || entry.Name() == "testdata" || entry.Name() == "node_modules") {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(name, ".go") {
			return nil
		}
		rel, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		// Only product source and explicitly classified test/tool harnesses.
		if !strings.HasPrefix(rel, "internal/") && !strings.HasPrefix(rel, "cmd/") && !strings.HasPrefix(rel, "tools/") && !strings.HasPrefix(rel, "examples/") && !strings.HasPrefix(rel, "integration/") && !strings.HasPrefix(rel, "experiments/") && !strings.HasPrefix(rel, "benchmark/") {
			return nil
		}
		packagePath := filepath.ToSlash(filepath.Dir(rel))
		if !seenPackages[packagePath] {
			seenPackages[packagePath] = true
			edges = append(edges, Edge{File: rel, From: packagePath, Test: strings.HasSuffix(rel, "_test.go")})
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			imported, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				return err
			}
			if !strings.HasPrefix(imported, ModulePath) {
				continue
			}
			edges = append(edges, Edge{rel, fset.Position(imp.Pos()).Line, filepath.ToSlash(filepath.Dir(rel)), strings.TrimPrefix(imported, ModulePath), strings.HasSuffix(rel, "_test.go")})
		}
		return nil
	})
	sort.Slice(edges, func(i, j int) bool {
		a, b := edges[i], edges[j]
		if a.File != b.File {
			return a.File < b.File
		}
		if a.Line != b.Line {
			return a.Line < b.Line
		}
		return a.To < b.To
	})
	return edges, err
}
