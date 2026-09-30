package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/inputs"
	"github.com/Glacius-Labs/Markitect/internal/source"
	"go.yaml.in/yaml/v3"
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
	var parseFindings []core.Diagnostic
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
			parseFindings = append(parseFindings, core.Diagnostic{Code: "parse", Path: name, Message: err.Error()})
			continue
		}
		p.Resources = append(p.Resources, r)
	}
	declaredFiles := map[string]bool{}
	for _, resource := range p.Resources {
		if resource.Kind == "Project" {
			continue
		}
		for _, file := range resource.Spec.Files {
			declaredFiles[file] = true
		}
	}
	typedPaths := map[string]*core.Resource{}
	for _, resource := range p.Resources {
		if resource.Kind != "Project" {
			typedPaths[resource.Path] = resource
		}
	}
	for _, diagnostic := range parseFindings {
		if !declaredFiles[diagnostic.Path] || markitectResourceEnvelope(snap.Files[diagnostic.Path]) {
			p.Diagnostics = append(p.Diagnostics, diagnostic)
		}
	}
	p.Graph = core.Build(p.Resources)
	p.Diagnostics = append(p.Diagnostics, p.Graph.Diagnostics...)
	resolved, findings := inputs.Resolve(p.Graph, snap.Files)
	p.InputFiles = resolved
	p.Diagnostics = append(p.Diagnostics, findings...)
	for key, files := range resolved {
		for _, file := range files {
			if typed := typedPaths[file]; typed != nil {
				resource := p.Graph.Resources[key]
				if resource == nil {
					continue
				}
				p.Diagnostics = append(p.Diagnostics, core.Diagnostic{
					Code: "input.typed-source", Path: resource.Path, Line: resource.Line,
					Message: fmt.Sprintf("input file %q is the source of typed resource %s; reference the resource instead of listing it in spec.files", file, typed.Key()),
				})
			}
		}
	}
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

// markitectResourceEnvelope distinguishes an explicitly declared ordinary YAML
// input from a malformed Markitect resource. Malformed YAML remains fail-closed.
// Markitect API versions are reserved under the markitect.example.org/ prefix;
// without apiVersion, a known kind plus metadata and spec is the recognizable
// incomplete envelope. Other YAML shapes, including Kubernetes manifests, are
// ordinary inputs when explicitly declared.
func markitectResourceEnvelope(data []byte) bool {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	for {
		var document yaml.Node
		err := decoder.Decode(&document)
		if err == io.EOF {
			return false
		}
		if err != nil {
			return true
		}
		if len(document.Content) == 0 || document.Content[0].Kind != yaml.MappingNode {
			continue
		}
		root := document.Content[0]
		if hasDuplicateScalarYAMLKeys(root) {
			return true
		}
		var apiVersion, kind, metadata, spec *yaml.Node
		for i := 0; i+1 < len(root.Content); i += 2 {
			key := root.Content[i]
			if key.Kind != yaml.ScalarNode {
				continue
			}
			value := root.Content[i+1]
			switch key.Value {
			case "apiVersion":
				apiVersion = value
			case "kind":
				kind = value
			case "metadata":
				metadata = value
			case "spec":
				spec = value
			}
		}
		if apiVersion != nil {
			apiGroup, _, _ := strings.Cut(core.APIVersion, "/")
			if apiVersion.Kind == yaml.ScalarNode && strings.HasPrefix(apiVersion.Value, apiGroup+"/") {
				return true
			}
			continue
		}
		if kind != nil && kind.Kind == yaml.ScalarNode && format.AllowedSpecFields(kind.Value) != nil && metadata != nil && metadata.Kind == yaml.MappingNode && spec != nil && spec.Kind == yaml.MappingNode {
			return true
		}
	}
}

func hasDuplicateScalarYAMLKeys(node *yaml.Node) bool {
	if node.Kind == yaml.MappingNode {
		seen := make(map[string]bool, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind == yaml.ScalarNode {
				identity := key.Tag + "\x00" + key.Value
				if seen[identity] {
					return true
				}
				seen[identity] = true
			}
		}
	}
	for _, child := range node.Content {
		if hasDuplicateScalarYAMLKeys(child) {
			return true
		}
	}
	return false
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
