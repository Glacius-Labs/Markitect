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
	PolicyResults  []core.PolicyResult `yaml:"policyResults,omitempty"`
	Analysis       *AnalysisEvidence   `yaml:"analysis,omitempty"`
}
type ContextInput struct {
	Key              string            `yaml:"key"`
	Path             string            `yaml:"path"`
	Package          string            `yaml:"package,omitempty"`
	PackageVersion   string            `yaml:"packageVersion,omitempty"`
	Hash             string            `yaml:"hash"`
	Reason           string            `yaml:"reason"`
	Via              []ContextRelation `yaml:"via,omitempty"`
	DomainAPIVersion string            `yaml:"domainApiVersion,omitempty"`
	DomainName       string            `yaml:"domainName,omitempty"`
	Resource         *core.Resource    `yaml:"resource,omitempty"`
	Text             string            `yaml:"text,omitempty"`
	Role             string            `yaml:"role,omitempty"`
	Status           string            `yaml:"status,omitempty"`
	Required         bool              `yaml:"required,omitempty"`
}

// ContextRelation explains one declared graph relationship through which a
// resource was included in the entry's bounded context.
type ContextRelation struct {
	From             string `yaml:"from"`
	Relation         string `yaml:"relation"`
	DomainAPIVersion string `yaml:"domainApiVersion,omitempty"`
	Path             string `yaml:"path,omitempty"`
	Line             int    `yaml:"line,omitempty"`
}

func CompileContext(p *Project, key, version string, toolDigest ...string) (*Context, error) {
	return compileContext(p, key, version, false, toolDigest...)
}

// AnalyzeContext compiles the ordinary declared context closure for a
// structurally valid candidate, even when ordinary policy results fail. The
// output is explicitly marked as analysis and remains non-acceptance evidence.
func AnalyzeContext(p *Project, key, version string, toolDigest ...string) (*Context, error) {
	return compileContext(p, key, version, true, toolDigest...)
}

func compileContext(p *Project, key, version string, allowPolicyFailures bool, toolDigest ...string) (*Context, error) {
	if p == nil || p.Graph == nil || p.Snapshot == nil {
		return nil, fmt.Errorf("a parsed project and fixed snapshot are required")
	}
	if allowPolicyFailures {
		if len(p.StructuralDiagnostics()) > 0 {
			return nil, fmt.Errorf("context analysis is blocked while structural diagnostics remain")
		}
	} else if len(p.Diagnostics) > 0 {
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
	parents := map[string]string{}
	queue := []string{key}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		edges := append([]string(nil), p.Graph.Edges[current]...)
		sort.Strings(edges)
		for _, dest := range edges {
			if _, ok := reasons[dest]; !ok {
				parents[dest] = current
				reasons[dest] = "required by " + current
				queue = append(queue, dest)
			}
		}
	}
	closure := make(map[string]bool, len(reasons))
	for resourceKey := range reasons {
		closure[resourceKey] = true
	}
	// Retain every declared context relationship between reachable resources,
	// not merely the first BFS path. This makes multiple inclusion causes
	// reviewable without changing the compact adjacency used by graph traversal.
	viaByResource := map[string][]ContextRelation{}
	for _, relationship := range p.Graph.Relationships {
		if !relationship.Context || relationship.From == relationship.To || !closure[relationship.From] || !closure[relationship.To] {
			continue
		}
		viaByResource[relationship.To] = append(viaByResource[relationship.To], ContextRelation{
			From: relationship.From, Relation: relationship.Relation, DomainAPIVersion: relationship.DomainAPIVersion,
			Path: relationship.Path, Line: relationship.Line,
		})
	}
	for resourceKey := range viaByResource {
		via := viaByResource[resourceKey]
		sort.Slice(via, func(i, j int) bool { return contextRelationLess(via[i], via[j]) })
		viaByResource[resourceKey] = via
	}
	for resourceKey, parent := range parents {
		via := viaByResource[resourceKey]
		for _, relation := range via {
			if relation.From == parent {
				reasons[resourceKey] = "required by " + parent + " via " + relation.Relation
				break
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
		via := append([]ContextRelation(nil), viaByResource[k]...)
		c.Inputs = append(c.Inputs, ContextInput{Key: k, Path: r.Path, Package: r.Package, PackageVersion: p.packageVersion(r.Package), Hash: h, Reason: reasons[k], Via: via, Resource: r})
		fmt.Fprintf(&fingerprint, "%d:%s%d:%s%d:%s", len(k), k, len(r.Path), r.Path, len(h), h)
		if len(via) > 0 {
			viaBytes, err := YAML(via)
			if err != nil {
				return nil, fmt.Errorf("encode context relationship provenance for %s: %w", k, err)
			}
			fmt.Fprintf(&fingerprint, "via:%s", Hash(viaBytes))
		}
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
	// A domain definition controls types, relationships and constraints even
	// when its file is outside resource Areas. Include its exact bytes, not just
	// the selected path in the Project configuration.
	for _, input := range p.DomainInputs {
		identity := "domain:" + inputKey(input.Package, input.Path)
		data := p.fileBytes(input.Package, input.Path)
		h := Hash(data)
		c.Inputs = append(c.Inputs, ContextInput{Key: identity, Path: input.Path, Package: input.Package, PackageVersion: p.packageVersion(input.Package), Hash: h, Reason: "selected language definition", Text: string(data), Role: "domain", DomainAPIVersion: input.APIVersion, DomainName: input.Name})
		fmt.Fprintf(&fingerprint, "domain:%d:%s%d:%s", len(identity), identity, len(h), h)
	}
	// Show the policy outcomes for this closure, including any explicitly
	// recorded exception. Collection assertions describe the selected domain
	// scope and remain visible independently of one subject's closure.
	for _, result := range p.Graph.PolicyResults {
		if _, included := reasons[result.Subject]; included || result.Subject == "" {
			c.PolicyResults = append(c.PolicyResults, result)
		}
	}
	if len(c.PolicyResults) > 0 {
		c.PolicyResults = clonePolicyResults(c.PolicyResults)
		policyBytes, err := YAML(c.PolicyResults)
		if err != nil {
			return nil, fmt.Errorf("encode context policy results: %w", err)
		}
		fmt.Fprintf(&fingerprint, "policy:%s", Hash(policyBytes))
	}
	if allowPolicyFailures {
		model, err := CompileModel(p)
		if err != nil {
			return nil, fmt.Errorf("compile context analysis identity: %w", err)
		}
		c.Analysis = &AnalysisEvidence{
			Mode: PolicyFailureAnalysisMode, Complete: true, Candidate: analysisSnapshot(model),
			Notice: "Policy analysis is read-only diagnostic evidence; it is not verification or acceptance.",
		}
		analysisBytes, err := YAML(c.Analysis)
		if err != nil {
			return nil, fmt.Errorf("encode context analysis evidence: %w", err)
		}
		fmt.Fprintf(&fingerprint, "analysis:%s", Hash(analysisBytes))
	}
	c.Digest = Hash([]byte(fingerprint.String()))
	return c, nil
}

func contextRelationLess(a, b ContextRelation) bool {
	if a.From != b.From {
		return a.From < b.From
	}
	if a.DomainAPIVersion != b.DomainAPIVersion {
		return a.DomainAPIVersion < b.DomainAPIVersion
	}
	if a.Relation != b.Relation {
		return a.Relation < b.Relation
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return a.Line < b.Line
}

func YAML(value any) ([]byte, error) { return format.Encode(value) }
