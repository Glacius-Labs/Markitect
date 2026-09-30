package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

type Context struct {
	Version        string         `yaml:"version"`
	Revision       string         `yaml:"revision,omitempty"`
	Provisional    bool           `yaml:"provisional"`
	Entry          string         `yaml:"entry"`
	Digest         string         `yaml:"digest"`
	SnapshotDigest string         `yaml:"snapshotDigest"`
	ToolDigest     string         `yaml:"toolDigest,omitempty"`
	Inputs         []ContextInput `yaml:"inputs"`
}
type ContextInput struct {
	Key            string         `yaml:"key"`
	Path           string         `yaml:"path"`
	Package        string         `yaml:"package,omitempty"`
	PackageVersion string         `yaml:"packageVersion,omitempty"`
	Hash           string         `yaml:"hash"`
	Reason         string         `yaml:"reason"`
	Resource       *core.Resource `yaml:"resource,omitempty"`
	Text           string         `yaml:"text,omitempty"`
}

func CompileContext(p *Project, key, version string, toolDigest ...string) (*Context, error) {
	if len(p.Diagnostics) > 0 {
		return nil, fmt.Errorf("context cannot be compiled while project diagnostics remain")
	}
	entry, ok := p.Graph.Resources[key]
	if !ok {
		return nil, fmt.Errorf("unknown entry %s", key)
	}
	if !p.exported(entry) {
		return nil, fmt.Errorf("package entry %s is not exported", key)
	}
	c := &Context{Version: version, Revision: p.Snapshot.Revision, Provisional: p.Snapshot.Provisional, Entry: key, SnapshotDigest: p.Snapshot.Digest()}
	if len(toolDigest) > 0 {
		c.ToolDigest = toolDigest[0]
	}
	reasons := map[string]string{key: "entry"}
	queue := []string{key}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		edges := append([]string(nil), p.Graph.Edges[current]...)
		sort.Strings(edges)
		for _, dest := range edges {
			if _, ok := reasons[dest]; !ok {
				reasons[dest] = "required by " + current
				queue = append(queue, dest)
			}
		}
	}
	// Configuration controls ownership, bindings and implicit rule applicability.
	reasons[p.Graph.Project.GraphKey()] = "project policy and bindings"
	for k := range reasons {
		r := p.Graph.Resources[k]
		if r != nil && r.Package != "" {
			manifest := p.Graph.Packages[r.Package]
			if manifest == nil {
				return nil, fmt.Errorf("package manifest for %s is missing", k)
			}
			reasons[manifest.GraphKey()] = "package policy and exports for " + r.Package
		}
	}
	keys := make([]string, 0, len(reasons))
	for k := range reasons {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var fingerprint strings.Builder
	fingerprint.WriteString(version + "\x00" + key + "\x00")
	fingerprint.WriteString(c.ToolDigest + "\x00")
	for _, k := range keys {
		r := p.Graph.Resources[k]
		if r == nil {
			return nil, fmt.Errorf("unresolved context resource %s", k)
		}
		h := Hash(p.resourceBytes(r))
		c.Inputs = append(c.Inputs, ContextInput{Key: k, Path: r.Path, Package: r.Package, PackageVersion: p.packageVersion(r.Package), Hash: h, Reason: reasons[k], Resource: r})
		fmt.Fprintf(&fingerprint, "%d:%s%d:%s%d:%s", len(k), k, len(r.Path), r.Path, len(h), h)
	}
	type declaredFile struct{ origin, path, reason string }
	fileReasons := map[string]declaredFile{}
	for _, k := range keys {
		r := p.Graph.Resources[k]
		for _, name := range p.InputFiles[k] {
			identity := inputKey(r.Package, name)
			if _, ok := fileReasons[identity]; !ok {
				fileReasons[identity] = declaredFile{r.Package, name, "declared input of " + k}
			}
		}
	}
	fileNames := make([]string, 0, len(fileReasons))
	for name := range fileReasons {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)
	for _, identity := range fileNames {
		file := fileReasons[identity]
		data := p.fileBytes(file.origin, file.path)
		h := Hash(data)
		c.Inputs = append(c.Inputs, ContextInput{Key: identity, Path: file.path, Package: file.origin, PackageVersion: p.packageVersion(file.origin), Hash: h, Reason: file.reason, Text: string(data)})
		// Preserve local fingerprints; package identities additionally separate
		// identically named archive members from local and other package inputs.
		fingerprintPath := file.path
		if file.origin != "" {
			fingerprintPath = identity
		}
		fmt.Fprintf(&fingerprint, "file:%d:%s%d:%s", len(fingerprintPath), fingerprintPath, len(h), h)
	}
	c.Digest = Hash([]byte(fingerprint.String()))
	return c, nil
}

type Impact struct {
	Base      string   `yaml:"base"`
	Candidate string   `yaml:"candidate"`
	Changed   []string `yaml:"changed"`
	Affected  []string `yaml:"affected"`
	Reason    string   `yaml:"reason"`
}

func Changes(before, after *Project) *Impact {
	result := &Impact{Base: before.Snapshot.Revision, Candidate: after.Snapshot.Revision, Reason: "Union of old and new dependency closures; configuration or inventory changes conservatively affect all resources."}
	changed := map[string]bool{}
	for name, b := range before.Snapshot.Files {
		a, ok := after.Snapshot.Files[name]
		if !ok || Hash(b) != Hash(a) || before.Snapshot.Modes[name] != after.Snapshot.Modes[name] {
			changed[name] = true
		}
	}
	for name := range after.Snapshot.Files {
		if _, ok := before.Snapshot.Files[name]; !ok {
			changed[name] = true
		}
	}
	for n := range changed {
		result.Changed = append(result.Changed, n)
	}
	sort.Strings(result.Changed)
	all := false
	seeds := map[string]bool{}
	ownedInputs := map[string]map[string]bool{}
	for _, p := range []*Project{before, after} {
		for _, r := range p.Resources {
			if r.Package == "" && changed[r.Path] {
				seeds[r.GraphKey()] = true
				if r.Kind == "Project" {
					all = true
				}
			}
		}
		fileOwners, err := impactFileOwners(p)
		if err != nil {
			all = true
			continue
		}
		for file, owners := range fileOwners {
			for owner := range owners {
				if ownedInputs[file] == nil {
					ownedInputs[file] = map[string]bool{}
				}
				ownedInputs[file][owner] = true
			}
		}
	}
	// New/deleted symbols may affect global constraints and scope-level discovery.
	if len(before.Graph.Resources) != len(after.Graph.Resources) {
		all = true
	}
	for k := range before.Graph.Resources {
		if _, ok := after.Graph.Resources[k]; !ok {
			all = true
		}
	}
	for k := range after.Graph.Resources {
		if _, ok := before.Graph.Resources[k]; !ok {
			all = true
		}
	}
	// Unmodelled inputs may contain normative contracts, router membership or
	// checker code. Until their independence is declared, never reuse evidence.
	modelled := map[string]bool{"markitect.yaml": true}
	for _, p := range []*Project{before, after} {
		for _, r := range p.Resources {
			if r.Package == "" {
				modelled[r.Path] = true
			}
		}
	}
	for name := range changed {
		if owners := ownedInputs[name]; len(owners) > 0 {
			for owner := range owners {
				seeds[owner] = true
			}
		} else if !modelled[name] {
			all = true
		}
	}
	if all {
		for _, p := range []*Project{before, after} {
			for k := range p.Graph.Resources {
				seeds[k] = true
			}
		}
	}
	for again := true; again; {
		again = false
		for _, p := range []*Project{before, after} {
			for from, edges := range p.Graph.Edges {
				if seeds[from] {
					continue
				}
				for _, to := range edges {
					if seeds[to] {
						seeds[from] = true
						again = true
						break
					}
				}
			}
		}
	}
	for k := range seeds {
		result.Affected = append(result.Affected, k)
	}
	sort.Strings(result.Affected)
	return result
}

// impactFileOwners maps only paths whose content is already represented by a
// typed resource or by an explicit Spec.Files declaration. Unknown generated
// files remain broad-impact inputs until their ownership is modeled.
func impactFileOwners(p *Project) (map[string]map[string]bool, error) {
	owners := map[string]map[string]bool{}
	add := func(file, key string) {
		if file == "" || key == "" {
			return
		}
		if owners[file] == nil {
			owners[file] = map[string]bool{}
		}
		owners[file][key] = true
	}
	if p == nil {
		return owners, nil
	}
	for _, r := range p.Resources {
		if r == nil || r.Kind == "Project" || r.Package != "" {
			continue
		}
		key := r.GraphKey()
		for _, file := range r.Spec.Files {
			add(file, key)
		}
		for _, file := range p.InputFiles[key] {
			add(file, key)
		}
	}
	_, generatedOwners, err := render.GenerateWithOwners(p.Graph)
	if err != nil {
		return nil, err
	}
	for file, resources := range generatedOwners {
		for _, resource := range resources {
			add(file, resource)
		}
	}
	return owners, nil
}

func YAML(value any) ([]byte, error) { return format.Encode(value) }
