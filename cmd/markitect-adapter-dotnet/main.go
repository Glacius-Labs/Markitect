// Command markitect-adapter-dotnet is a read-only reference adapter for the
// Markitect normalized semantic-model protocol. It inspects captured MSBuild
// project files and compares ProjectReference edges with canonical Module
// dependsOn relationships. It never loads project code or runs MSBuild.
package main

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	requestAPIVersion = "markitect.example.org/adapter-request/v1alpha1"
	resultAPIVersion  = "markitect.example.org/adapter-result/v1alpha1"
	semanticModelAPI  = "markitect.example.org/semantic-model/v1alpha1"
)

type request struct {
	APIVersion  string         `yaml:"apiVersion"`
	Action      string         `yaml:"action"`
	Adapter     adapterRequest `yaml:"adapter"`
	Model       semanticModel  `yaml:"model"`
	Observation *result        `yaml:"observation,omitempty"`
	Plan        *result        `yaml:"plan,omitempty"`
}

type adapterRequest struct {
	Name       string         `yaml:"name"`
	Type       string         `yaml:"type"`
	Version    string         `yaml:"version"`
	Parameters map[string]any `yaml:"parameters"`
}

type adapterParameters struct {
	ProjectMappings []projectMapping `yaml:"projectMappings"`
}

type projectMapping struct {
	Resource    string `yaml:"resource"`
	ProjectFile string `yaml:"projectFile"`
}

type semanticModel struct {
	APIVersion       string         `yaml:"apiVersion"`
	ModelDigest      string         `yaml:"modelDigest"`
	ValidationStatus string         `yaml:"validationStatus"`
	Resources        []resource     `yaml:"resources"`
	Relationships    []relationship `yaml:"relationships"`
}

type resource struct {
	Identity resourceIdentity `yaml:"identity"`
}

type resourceIdentity struct {
	Kind string `yaml:"kind"`
	Key  string `yaml:"key"`
}

type relationship struct {
	From string `yaml:"from"`
	To   string `yaml:"to"`
	Type string `yaml:"type"`
}

type result struct {
	APIVersion  string       `yaml:"apiVersion"`
	Adapter     string       `yaml:"adapter"`
	Action      string       `yaml:"action"`
	Status      string       `yaml:"status"`
	ModelDigest string       `yaml:"modelDigest"`
	Findings    []finding    `yaml:"findings"`
	Observed    *observation `yaml:"observed,omitempty"`
	Operations  []operation  `yaml:"operations,omitempty"`
}

type observation struct {
	Evidence     string              `yaml:"evidence"`
	Scope        []string            `yaml:"scope"`
	Dependencies map[string][]string `yaml:"dependencies"`
}

type finding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Path     string `yaml:"path,omitempty"`
	Message  string `yaml:"message"`
}

type operation struct {
	ID      string `yaml:"id"`
	Action  string `yaml:"action"`
	Target  string `yaml:"target"`
	Desired any    `yaml:"desired,omitempty"`
}

func main() {
	input, err := io.ReadAll(os.Stdin)
	if err != nil {
		fatal(err)
	}
	var req request
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	if err := decoder.Decode(&req); err != nil {
		fatal(fmt.Errorf("decode adapter request: %w", err))
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		fatal(errors.New("adapter request must contain exactly one YAML document"))
	}
	res := run(req)
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	if err := encoder.Encode(res); err != nil {
		fatal(err)
	}
	_ = encoder.Close()
}

func fatal(err error) {
	res := baseResult("", "", "failed", "")
	res.Findings = []finding{{Code: "invalid-request", Severity: "error", Message: err.Error()}}
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	_ = encoder.Encode(res)
	os.Exit(2)
}

func run(req request) result {
	res := baseResult(req.Action, req.Adapter.Name, "complete", req.Model.ModelDigest)
	if req.APIVersion != requestAPIVersion {
		return fail(res, "unsupported-request-version", "expected apiVersion "+requestAPIVersion)
	}
	if req.Action != "observe" && req.Action != "plan" && req.Action != "verify" {
		return fail(res, "unsupported-action", "action "+quote(req.Action)+" is not supported; this adapter is read-only")
	}
	if req.Adapter.Name == "" || req.Adapter.Type != "command" || req.Adapter.Version == "" {
		return fail(res, "adapter-identity-mismatch", "request adapter identity requires a name, type command, and version")
	}
	parameters, err := decodeParameters(req.Adapter.Parameters)
	if err != nil {
		return fail(res, "adapter-parameters-invalid", err.Error())
	}
	if req.Model.ModelDigest == "" {
		return incomplete(res, "model-digest-missing", "normalized model does not declare modelDigest")
	}
	if req.Model.APIVersion != semanticModelAPI {
		return fail(res, "unsupported-model-version", "unsupported normalized model apiVersion "+quote(req.Model.APIVersion))
	}
	if req.Model.ValidationStatus != "passed" {
		return fail(res, "model-not-validated", "adapter requires a normalized semantic model with validationStatus passed")
	}
	if len(parameters.ProjectMappings) == 0 {
		return incomplete(res, "project-mappings-missing", "adapter.parameters.projectMappings must declare the canonical resources in this adapter's check scope")
	}

	resourceKeys := map[string]bool{}
	for _, r := range req.Model.Resources {
		if r.Identity.Key != "" {
			resourceKeys[r.Identity.Key] = true
		}
	}
	if len(resourceKeys) == 0 {
		return incomplete(res, "resources-missing", "normalized model contains no resources with canonical identity keys")
	}

	mappings := map[string]string{}
	pathOwners := map[string]string{}
	for _, item := range parameters.ProjectMappings {
		if item.Resource == "" || !resourceKeys[item.Resource] {
			res.Findings = append(res.Findings, finding{Code: "resource-mapping-invalid", Severity: "error", Path: item.ProjectFile, Message: "mapping must reference an existing exact canonical identity.key: " + quote(item.Resource)})
			continue
		}
		clean, err := cleanRelativePath(item.ProjectFile)
		if err != nil {
			res.Findings = append(res.Findings, finding{Code: "project-path-invalid", Severity: "error", Path: item.ProjectFile, Message: err.Error()})
			continue
		}
		if !strings.EqualFold(filepath.Ext(clean), ".csproj") {
			res.Findings = append(res.Findings, finding{Code: "project-path-invalid", Severity: "error", Path: clean, Message: "projectFile must identify a .csproj input"})
			continue
		}
		if previous, ok := mappings[item.Resource]; ok {
			res.Findings = append(res.Findings, finding{Code: "resource-mapping-ambiguous", Severity: "error", Path: clean, Message: "canonical resource has multiple project mappings: " + quote(previous) + " and " + quote(clean)})
			continue
		}
		if previous, ok := pathOwners[clean]; ok {
			res.Findings = append(res.Findings, finding{Code: "project-mapping-ambiguous", Severity: "error", Path: clean, Message: "project input maps to multiple canonical resources: " + quote(previous) + " and " + quote(item.Resource)})
			continue
		}
		mappings[item.Resource] = clean
		pathOwners[clean] = item.Resource
	}
	if len(res.Findings) > 0 {
		return sortedResult(res, "incomplete")
	}
	resourceDeps := map[string]map[string]bool{}
	for _, edge := range req.Model.Relationships {
		if edge.Type != "dependsOn" || mappings[edge.From] == "" {
			continue
		}
		if !resourceKeys[edge.To] {
			res.Findings = append(res.Findings, finding{Code: "dependency-target-invalid", Severity: "error", Message: "dependsOn target is absent from the normalized model: " + quote(edge.To)})
			continue
		}
		if mappings[edge.To] == "" {
			res.Findings = append(res.Findings, finding{Code: "dependency-target-unmapped", Severity: "error", Message: "dependsOn target must be mapped to a captured project input: " + quote(edge.From) + " -> " + quote(edge.To)})
			continue
		}
		if resourceDeps[edge.From] == nil {
			resourceDeps[edge.From] = map[string]bool{}
		}
		resourceDeps[edge.From][edge.To] = true
	}
	if len(res.Findings) > 0 {
		return sortedResult(res, "incomplete")
	}

	projectToModule := map[string]string{}
	for key, path := range mappings {
		projectToModule[path] = key
	}
	observed := map[string][]string{}
	for _, key := range sortedKeys(mappings) {
		projectFile := mappings[key]
		actualPath, err := stagedPath(projectFile)
		if err != nil {
			res.Findings = append(res.Findings, finding{Code: "project-input-missing", Severity: "error", Path: projectFile, Message: err.Error()})
			continue
		}
		data, err := os.ReadFile(actualPath)
		if err != nil {
			res.Findings = append(res.Findings, finding{Code: "project-input-missing", Severity: "error", Path: projectFile, Message: "cannot read staged project input: " + err.Error()})
			continue
		}
		targetPaths, unsupported, err := projectReferences(data)
		if err != nil {
			res.Findings = append(res.Findings, finding{Code: "project-xml-invalid", Severity: "error", Path: projectFile, Message: err.Error()})
			continue
		}
		if unsupported != "" {
			res.Findings = append(res.Findings, finding{Code: "project-semantics-unsupported", Severity: "error", Path: projectFile, Message: unsupported})
			continue
		}
		actual := []string{}
		for _, reference := range targetPaths {
			resolved, err := resolveProjectReference(projectFile, reference)
			if err != nil {
				res.Findings = append(res.Findings, finding{Code: "project-reference-unsupported", Severity: "error", Path: projectFile, Message: "ProjectReference " + quote(reference) + ": " + err.Error()})
				continue
			}
			target, ok := projectToModule[resolved]
			if !ok {
				res.Findings = append(res.Findings, finding{Code: "project-reference-unmapped", Severity: "error", Path: projectFile, Message: "ProjectReference target " + quote(resolved) + " is not mapped to a canonical Module"})
				continue
			}
			actual = append(actual, target)
		}
		observed[key] = uniqueSorted(actual)
	}
	if hasIncompleteFinding(res.Findings) {
		return sortedResult(res, "incomplete")
	}
	res.Observed = &observation{Evidence: "literal-unconditional-project-reference-xml", Scope: sortedKeys(mappings), Dependencies: observed}
	if req.Action == "plan" {
		if req.Observation == nil {
			return incomplete(res, "observation-missing", "plan requires the completed observation from this adapter")
		}
		if req.Observation.Adapter != req.Adapter.Name || req.Observation.Action != "observe" || req.Observation.Status != "complete" || req.Observation.ModelDigest != req.Model.ModelDigest {
			return fail(res, "observation-identity-mismatch", "plan observation must be a complete observe result for this adapter and model")
		}
		if !sameObservation(req.Observation.Observed, res.Observed) {
			return fail(res, "observation-drift", "captured project inputs changed between observe and plan")
		}
	}

	for _, from := range sortedKeys(mappings) {
		want := resourceDeps[from]
		got := make(map[string]bool, len(observed[from]))
		for _, to := range observed[from] {
			got[to] = true
		}
		for _, to := range sortedKeys(want) {
			if !got[to] {
				res.Findings = append(res.Findings, finding{Code: "dependency-missing", Severity: "error", Path: mappings[from], Message: "canonical dependsOn target " + quote(to) + " is missing from project references for " + quote(from)})
			}
		}
		for _, to := range sortedKeys(got) {
			if !want[to] {
				res.Findings = append(res.Findings, finding{Code: "dependency-forbidden", Severity: "error", Path: mappings[from], Message: "project reference to " + quote(to) + " is not declared by canonical dependsOn for " + quote(from)})
			}
		}
	}
	if req.Action == "verify" {
		if req.Plan == nil {
			return incomplete(res, "plan-missing", "verify requires a saved plan containing modelDigest and observed module dependencies")
		}
		if req.Plan.Action != "plan" || req.Plan.Status != "complete" || req.Plan.Adapter != req.Adapter.Name || req.Plan.ModelDigest != req.Model.ModelDigest {
			return fail(res, "plan-model-drift", "plan modelDigest does not match current normalized model")
		}
		if !sameObservation(req.Plan.Observed, res.Observed) {
			res.Findings = append(res.Findings, finding{Code: "verification-drift", Severity: "error", Message: "observed project references differ from the saved plan"})
		}
	}
	return sortedResult(res, "complete")
}

func decodeParameters(values map[string]any) (adapterParameters, error) {
	if len(values) == 0 {
		return adapterParameters{}, nil
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return adapterParameters{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var parameters adapterParameters
	if err := decoder.Decode(&parameters); err != nil {
		return parameters, fmt.Errorf("unsupported or invalid adapter parameter: %w", err)
	}
	return parameters, nil
}

func baseResult(action, adapter, status, digest string) result {
	return result{APIVersion: resultAPIVersion, Adapter: adapter, Action: action, Status: status, ModelDigest: digest, Findings: []finding{}, Operations: []operation{}}
}

func fail(res result, code, message string) result {
	res.Findings = append(res.Findings, finding{Code: code, Severity: "error", Message: message})
	return sortedResult(res, "failed")
}

func incomplete(res result, code, message string) result {
	res.Findings = append(res.Findings, finding{Code: code, Severity: "error", Message: message})
	return sortedResult(res, "incomplete")
}

func sortedResult(res result, status string) result {
	res.Status = status
	sort.Slice(res.Findings, func(i, j int) bool {
		a, b := res.Findings[i], res.Findings[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		return a.Message < b.Message
	})
	return res
}

func cleanRelativePath(path string) (string, error) {
	if strings.TrimSpace(path) == "" {
		return "", errors.New("projectFile must be a non-empty repository-relative path")
	}
	path = strings.ReplaceAll(path, "\\", "/")
	if strings.HasPrefix(path, "/") || filepath.IsAbs(path) || (len(path) > 1 && path[1] == ':') {
		return "", errors.New("projectFile must be a repository-relative path")
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(path)))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("projectFile must stay within the staged input directory")
	}
	return clean, nil
}

func stagedPath(relative string) (string, error) {
	root, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	joined := filepath.Join(root, filepath.FromSlash(relative))
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolvedFile, err := filepath.EvalSymlinks(joined)
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolvedFile)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("project input resolves outside the staged input directory")
	}
	return resolvedFile, nil
}

func projectReferences(data []byte) ([]string, string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(data))
	refs := []string{}
	ancestors := []xml.StartElement{}
	rootSeen := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, "", fmt.Errorf("invalid project XML: %w", err)
		}
		switch token.(type) {
		case xml.EndElement:
			if len(ancestors) > 0 {
				ancestors = ancestors[:len(ancestors)-1]
			}
			continue
		}
		start, ok := token.(xml.StartElement)
		if !ok {
			continue
		}
		if !rootSeen {
			rootSeen = true
			if start.Name.Local != "Project" {
				return nil, "", errors.New(".csproj XML root element must be Project")
			}
		}
		unsupportedNode := ""
		if start.Name.Local == "Import" || start.Name.Local == "ImportGroup" || start.Name.Local == "Sdk" {
			unsupportedNode = "project uses imports or SDK declarations whose evaluated items are outside the captured XML subset"
		}
		if start.Name.Local == "Project" && hasAttribute(start, "Condition") {
			unsupportedNode = "project-level conditions are outside the captured XML subset"
		}
		if start.Name.Local == "ProjectReference" {
			inItemGroup := false
			for _, parent := range ancestors {
				switch parent.Name.Local {
				case "ItemGroup":
					inItemGroup = true
					if hasAttribute(parent, "Condition") {
						unsupportedNode = "conditioned ItemGroup may add or remove ProjectReference items"
					}
				case "Choose", "When", "Otherwise", "Target":
					unsupportedNode = "ProjectReference is controlled by a conditional or target-time MSBuild construct"
				}
			}
			if !inItemGroup {
				unsupportedNode = "ProjectReference outside an unconditional ItemGroup is not supported"
			}
			if hasAttribute(start, "Condition") || hasAttribute(start, "Update") || hasAttribute(start, "Remove") || hasAttribute(start, "Exclude") {
				unsupportedNode = "conditioned or transformed ProjectReference items are not supported"
			}
			if hasAttribute(start, "Include") == false {
				return nil, "", errors.New("ProjectReference is missing Include")
			}
		}
		if unsupportedNode != "" {
			return nil, unsupportedNode, nil
		}
		if start.Name.Local != "ProjectReference" {
			ancestors = append(ancestors, start)
			continue
		}
		include := ""
		for _, attribute := range start.Attr {
			if attribute.Name.Local == "Include" {
				if include != "" {
					return nil, "", errors.New("ProjectReference has multiple Include attributes")
				}
				include = strings.TrimSpace(attribute.Value)
			}
		}
		if include == "" {
			return nil, "", errors.New("ProjectReference is missing Include")
		}
		refs = append(refs, include)
		ancestors = append(ancestors, start)
	}
	if !rootSeen {
		return nil, "", errors.New(".csproj XML document is empty")
	}
	return uniqueSorted(refs), "", nil
}

func hasAttribute(element xml.StartElement, name string) bool {
	for _, attribute := range element.Attr {
		if attribute.Name.Local == name {
			return true
		}
	}
	return false
}

func resolveProjectReference(projectFile, reference string) (string, error) {
	if strings.ContainsAny(reference, "$@%*?") {
		return "", errors.New("MSBuild expressions and wildcard references are not supported by this deterministic check")
	}
	reference = strings.ReplaceAll(strings.TrimSpace(reference), "\\", "/")
	if reference == "" || strings.HasPrefix(reference, "/") || (len(reference) > 1 && reference[1] == ':') {
		return "", errors.New("reference must resolve to a repository-relative captured project path")
	}
	base := filepath.ToSlash(filepath.Dir(filepath.FromSlash(projectFile)))
	resolved := filepath.ToSlash(filepath.Clean(filepath.Join(filepath.FromSlash(base), filepath.FromSlash(reference))))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", errors.New("reference resolves outside the captured repository inputs")
	}
	if !strings.EqualFold(filepath.Ext(resolved), ".csproj") {
		return "", errors.New("reference does not identify a .csproj project input")
	}
	return resolved, nil
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	for _, value := range values {
		seen[value] = true
	}
	return sortedKeys(seen)
}

func sortedKeys[V any](values map[string]V) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func hasIncompleteFinding(findings []finding) bool { return len(findings) > 0 }

func sameObservation(a, b *observation) bool {
	if a == nil || b == nil || a.Evidence != b.Evidence || strings.Join(a.Scope, "\x00") != strings.Join(b.Scope, "\x00") || len(a.Dependencies) != len(b.Dependencies) {
		return false
	}
	for key, values := range a.Dependencies {
		if strings.Join(uniqueSorted(values), "\x00") != strings.Join(uniqueSorted(b.Dependencies[key]), "\x00") {
			return false
		}
	}
	return true
}

func quote(value string) string { return fmt.Sprintf("%q", value) }
