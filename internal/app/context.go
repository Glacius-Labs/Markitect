package app

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
)

type Context struct {
	Version        string              `yaml:"version"`
	Revision       string              `yaml:"revision,omitempty"`
	Provisional    bool                `yaml:"provisional"`
	Entry          string              `yaml:"entry"`
	Digest         string              `yaml:"digest"`
	SnapshotDigest string              `yaml:"snapshotDigest"`
	ToolDigest     string              `yaml:"toolDigest,omitempty"`
	Inputs         []ContextInput      `yaml:"inputs"`
	Complete       bool                `yaml:"complete,omitempty"`
	Status         string              `yaml:"status,omitempty"`
	Run            *RunContextEvidence `yaml:"run,omitempty"`
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
	Role           string         `yaml:"role,omitempty"`
	Status         string         `yaml:"status,omitempty"`
	Required       bool           `yaml:"required,omitempty"`
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
	c := &Context{Version: version, Revision: p.Snapshot.ID, Provisional: p.Snapshot.Provisional, Entry: key, SnapshotDigest: p.Snapshot.Digest()}
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

func YAML(value any) ([]byte, error) { return format.Encode(value) }
