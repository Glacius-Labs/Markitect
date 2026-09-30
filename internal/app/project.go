package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"markitect/internal/core"
	"markitect/internal/format"
	"markitect/internal/inputs"
	"markitect/internal/source"
)

// Project is one immutable input set and its resolved resources.
type Project struct {
	Snapshot    *source.Snapshot
	Graph       *core.Graph
	Resources   []*core.Resource
	Inventory   []Entry
	Diagnostics []core.Diagnostic
	InputFiles  map[string][]string
}

type Entry struct {
	Path      string `yaml:"path"`
	Kind      string `yaml:"kind"`
	Name      string `yaml:"name,omitempty"`
	Namespace string `yaml:"namespace,omitempty"`
	Hash      string `yaml:"hash"`
}

func Load(root, revision string) (*Project, error) {
	snap, err := source.Load(root, revision)
	if err != nil {
		return nil, err
	}
	return Parse(snap)
}

func Parse(snap *source.Snapshot) (*Project, error) {
	p := &Project{Snapshot: snap}
	data, ok := snap.Files["markitect.yaml"]
	if !ok {
		return nil, fmt.Errorf("markitect.yaml is missing from the selected source; run inventory before migration")
	}
	config, err := format.Parse("markitect.yaml", data)
	if err != nil {
		return nil, err
	}
	if config.Kind != "Project" {
		return nil, fmt.Errorf("markitect.yaml must contain a Project")
	}
	p.Resources = append(p.Resources, config)
	paths := sortedFiles(snap.Files)
	for _, name := range paths {
		if name == "markitect.yaml" || (path.Ext(name) != ".yaml" && path.Ext(name) != ".yml") {
			continue
		}
		scoped := false
		for _, area := range config.Spec.Areas {
			if Within(name, area.Path) {
				scoped = true
				break
			}
		}
		if !scoped {
			continue
		}
		r, err := format.Parse(name, snap.Files[name])
		if err != nil {
			p.Diagnostics = append(p.Diagnostics, core.Diagnostic{Code: "parse", Path: name, Message: err.Error()})
			continue
		}
		p.Resources = append(p.Resources, r)
	}
	p.Graph = core.Build(p.Resources)
	p.Diagnostics = append(p.Diagnostics, p.Graph.Diagnostics...)
	resolved, findings := inputs.Resolve(p.Graph, snap.Files)
	p.InputFiles = resolved
	p.Diagnostics = append(p.Diagnostics, findings...)
	for _, r := range p.Resources {
		p.Inventory = append(p.Inventory, Entry{Path: r.Path, Kind: r.Kind, Name: r.Metadata.Name, Namespace: r.Metadata.Namespace, Hash: Hash(snap.Files[r.Path])})
	}
	if config.Spec.Profile == "konfyra" {
		for _, candidate := range LegacyInventory(snap) {
			companion := strings.TrimSuffix(candidate.Path, ".md") + ".yaml"
			if _, ok := snap.Files[companion]; !ok {
				p.Diagnostics = append(p.Diagnostics, core.Diagnostic{Code: "unclassified-mechanism", Path: candidate.Path, Message: "active AI mechanism has no canonical YAML resource; migrate or explicitly retire it"})
			}
		}
	}
	return p, nil
}

func Hash(b []byte) string { h := sha256.Sum256(b); return "sha256:" + hex.EncodeToString(h[:]) }
func Within(name, root string) bool {
	return name == root || strings.HasPrefix(name, strings.TrimSuffix(root, "/")+"/")
}
func sortedFiles(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for n := range files {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// LegacyInventory reports candidates without inferring normative dependencies.
func LegacyInventory(snap *source.Snapshot) []Entry {
	var result []Entry
	for _, name := range sortedFiles(snap.Files) {
		if Generated(snap.Files[name]) {
			continue
		}
		if !strings.HasPrefix(name, "docs/") || path.Ext(name) != ".md" || path.Base(name) == "README.md" {
			continue
		}
		kind := ""
		switch path.Base(path.Dir(name)) {
		case "rules":
			kind = "Rule"
		case "skills":
			kind = "Skill"
		case "agents":
			kind = "Agent"
		case "workflows":
			kind = "Workflow"
		}
		if kind != "" {
			result = append(result, Entry{Path: name, Kind: kind, Name: strings.TrimSuffix(path.Base(name), ".md"), Hash: Hash(snap.Files[name])})
		}
	}
	return result
}
