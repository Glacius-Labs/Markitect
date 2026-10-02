package app

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/inputs"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// Project is one immutable input set and its resolved resources.
type Project struct {
	Snapshot    *snapshot.Snapshot
	Graph       *core.Graph
	Resources   []*core.Resource
	Inventory   []Entry
	Diagnostics []core.Diagnostic
	InputFiles  map[string][]string
	// DomainInputs records the exact, explicitly selected language definitions.
	// These bytes participate in context evidence independently of resource paths.
	DomainInputs []DomainInput
	// PackageFiles is verified, immutable archive content, kept separate from
	// the physical repository snapshot and its writable output paths.
	PackageFiles map[string]map[string][]byte
}

type Entry struct {
	APIVersion string `yaml:"apiVersion,omitempty"`
	Path       string `yaml:"path"`
	Kind       string `yaml:"kind"`
	Name       string `yaml:"name,omitempty"`
	Namespace  string `yaml:"namespace,omitempty"`
	Package    string `yaml:"package,omitempty"`
	Hash       string `yaml:"hash"`
}

func Parse(snap *snapshot.Snapshot) (*Project, error) {
	if snap == nil {
		return nil, fmt.Errorf("project snapshot is required")
	}
	p := &Project{Snapshot: snap, PackageFiles: map[string]map[string][]byte{}}
	data, ok := snap.Files["markitect.yaml"]
	if !ok {
		return nil, fmt.Errorf("markitect.yaml is missing from the selected source; use inventory to inspect existing Markdown and init to preview a new Project with an explicit area")
	}
	config, err := format.Parse("markitect.yaml", data)
	if err != nil {
		return nil, err
	}
	if config.Kind != "Project" {
		return nil, fmt.Errorf("markitect.yaml must contain a Project")
	}
	p.Resources = append(p.Resources, config)
	if err := p.loadPackageInputs(config); err != nil {
		return nil, err
	}
	registry, err := p.loadDomainInputs(config)
	if err != nil {
		return nil, err
	}
	paths := sortedFiles(snap.Files)
	var parseFindings []core.Diagnostic
	for _, name := range paths {
		if name == "markitect.yaml" || (path.Ext(name) != ".yaml" && path.Ext(name) != ".yml") {
			continue
		}
		if p.isDomainInput("", name) {
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
		r, err := format.ParseWithRegistry(name, snap.Files[name], registry)
		if err != nil {
			parseFindings = append(parseFindings, core.Diagnostic{Code: "parse", Path: name, Message: err.Error()})
			continue
		}
		if r.Kind == "Package" {
			return nil, fmt.Errorf("local Package manifest %q is not a project resource; build and pin its archive explicitly", name)
		}
		p.Resources = append(p.Resources, r)
	}
	declaredFiles := map[string]bool{}
	for _, resource := range p.Resources {
		if resource.Kind == "Project" || resource.Package != "" {
			continue
		}
		for _, file := range resource.Spec.Files {
			declaredFiles[file] = true
		}
	}
	typedPaths := map[string]*core.Resource{}
	for _, resource := range p.Resources {
		if resource.Kind != "Project" && resource.Kind != "Package" {
			typedPaths[inputKey(resource.Package, resource.Path)] = resource
		}
	}
	for _, diagnostic := range parseFindings {
		if !declaredFiles[diagnostic.Path] || format.IsResourceEnvelopeWithRegistry(snap.Files[diagnostic.Path], registry) {
			p.Diagnostics = append(p.Diagnostics, diagnostic)
		}
	}
	p.Graph = core.BuildWithRegistry(p.Resources, registry)
	p.Diagnostics = append(p.Diagnostics, p.Graph.Diagnostics...)
	resolved, findings := inputs.ResolveWithPackages(p.Graph, snap.Files, p.PackageFiles)
	p.InputFiles = resolved
	p.Diagnostics = append(p.Diagnostics, findings...)
	for key, files := range resolved {
		resource := p.Graph.Resources[key]
		if resource == nil {
			continue
		}
		for _, file := range files {
			if typed := typedPaths[inputKey(resource.Package, file)]; typed != nil {
				p.Diagnostics = append(p.Diagnostics, core.Diagnostic{
					Code: "input.typed-source", Path: resource.Path, Package: resource.Package, Line: resource.Line,
					Message: fmt.Sprintf("input file %q is the source of typed resource %s; reference the resource instead of listing it in spec.files", file, typed.GraphKey()),
				})
			}
		}
	}
	for _, r := range p.Resources {
		api := r.APIVersion
		if api == core.APIVersion {
			api = ""
		}
		p.Inventory = append(p.Inventory, Entry{APIVersion: api, Path: r.Path, Kind: r.Kind, Name: r.Metadata.Name, Namespace: r.Metadata.Namespace, Package: r.Package, Hash: Hash(p.resourceBytes(r))})
	}
	return p, nil
}

func (p *Project) fileBytes(origin, name string) []byte {
	if origin != "" {
		return p.PackageFiles[origin][name]
	}
	return p.Snapshot.Files[name]
}

func (p *Project) resourceBytes(r *core.Resource) []byte {
	return p.fileBytes(r.Package, r.Path)
}

func (p *Project) packageVersion(origin string) string {
	if manifest := p.Graph.Packages[origin]; manifest != nil {
		return manifest.Spec.Version
	}
	return ""
}

// inputKey keeps physical paths distinct from archive-relative paths without
// presenting archive members as writable files in the repository.
func inputKey(origin, name string) string {
	if origin != "" {
		return "package:" + origin + "/file:" + name
	}
	return "file:" + name
}

func (p *Project) exported(r *core.Resource) bool {
	if r == nil {
		return false
	}
	if r.Package == "" {
		return true
	}
	return p.Graph.IsExported(r)
}

// markitectResourceEnvelope uses the shared strict envelope boundary.
func markitectResourceEnvelope(data []byte) bool { return format.IsResourceEnvelope(data) }

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

// MarkdownInventory lists ordinary Markdown candidates without assigning
// resource kinds from their directories. Typed resources remain represented by
// the parsed Project.Inventory entries.
func MarkdownInventory(snap *snapshot.Snapshot) []Entry {
	var result []Entry
	if snap == nil {
		return result
	}
	for _, name := range sortedFiles(snap.Files) {
		if path.Ext(name) != ".md" || Generated(snap.Files[name]) {
			continue
		}
		result = append(result, Entry{Path: name, Kind: "Markdown", Name: strings.TrimSuffix(path.Base(name), ".md"), Hash: Hash(snap.Files[name])})
	}
	return result
}
