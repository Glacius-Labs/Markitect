// Package dotnet compares supplied MSBuild project-file bytes with a normalized
// Core model. It performs no filesystem access or process execution.
package dotnet

import (
	"bytes"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"

	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

const (
	requestAPIVersion = "markitect.example.org/adapter-request/v1alpha1"
	resultAPIVersion  = "markitect.example.org/adapter-result/v1alpha1"
	semanticModelAPI  = "markitect.example.org/semantic-model/v1alpha1"
	msbuildNamespace  = "http://schemas.microsoft.com/developer/msbuild/2003"
)

type Config struct {
	ProjectMappings []projectMapping `yaml:"projectMappings"`
}

type projectMapping struct {
	Resource    string `yaml:"resource"`
	ProjectFile string `yaml:"projectFile"`
}

type Identity struct {
	Name    string
	Type    string
	Version string
	Target  string
}

type Request struct {
	APIVersion    string
	Action        string
	Adapter       Identity
	Model         core.SemanticModel
	Config        Config
	ConfigError   string
	Captures      map[string][]byte
	CaptureErrors map[string]string
	Observation   *Result
	Plan          *Result
}

// CapturePaths returns only clean project-file paths named by this module's
// explicit configuration. The Host remains responsible for reading them.
func CapturePaths(config Config) []string {
	seen := map[string]bool{}
	var paths []string
	for _, mapping := range config.ProjectMappings {
		clean, err := cleanRelativePath(mapping.ProjectFile)
		if err == nil && strings.EqualFold(path.Ext(clean), ".csproj") && !seen[clean] {
			seen[clean] = true
			paths = append(paths, clean)
		}
	}
	sort.Strings(paths)
	return paths
}

// InvalidRequest creates the protocol-shaped failure used by the Host when
// it cannot decode the outer adapter request.
func InvalidRequest(message string) Result {
	res := baseResult("", "", "failed", "")
	return sortedResult(Result{APIVersion: res.APIVersion, Adapter: res.Adapter, Action: res.Action, Status: res.Status, ModelDigest: res.ModelDigest, Findings: []Finding{{Code: "invalid-request", Severity: "error", Message: message}}, Operations: []Operation{}}, "failed")
}

type Result struct {
	APIVersion  string       `yaml:"apiVersion"`
	Adapter     string       `yaml:"adapter"`
	Action      string       `yaml:"action"`
	Status      string       `yaml:"status"`
	ModelDigest string       `yaml:"modelDigest"`
	Findings    []Finding    `yaml:"findings"`
	Observed    *Observation `yaml:"observed,omitempty"`
	Operations  []Operation  `yaml:"operations,omitempty"`
}

type Observation struct {
	Evidence     string              `yaml:"evidence"`
	Scope        []string            `yaml:"scope"`
	Dependencies map[string][]string `yaml:"dependencies"`
}

type Finding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Path     string `yaml:"path,omitempty"`
	Message  string `yaml:"message"`
}

type Operation struct {
	ID      string `yaml:"id"`
	Action  string `yaml:"action"`
	Target  string `yaml:"target"`
	Desired any    `yaml:"desired,omitempty"`
}

func Run(req Request) Result {
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
	if req.ConfigError != "" {
		return fail(res, "adapter-parameters-invalid", req.ConfigError)
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
	if len(req.Config.ProjectMappings) == 0 {
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
	for _, item := range req.Config.ProjectMappings {
		if item.Resource == "" || !resourceKeys[item.Resource] {
			res.Findings = append(res.Findings, Finding{Code: "resource-mapping-invalid", Severity: "error", Path: item.ProjectFile, Message: "mapping must reference an existing exact canonical identity.key: " + quote(item.Resource)})
			continue
		}
		clean, err := cleanRelativePath(item.ProjectFile)
		if err != nil {
			res.Findings = append(res.Findings, Finding{Code: "project-path-invalid", Severity: "error", Path: item.ProjectFile, Message: err.Error()})
			continue
		}
		if !strings.EqualFold(path.Ext(clean), ".csproj") {
			res.Findings = append(res.Findings, Finding{Code: "project-path-invalid", Severity: "error", Path: clean, Message: "projectFile must identify a .csproj input"})
			continue
		}
		if previous, ok := mappings[item.Resource]; ok {
			res.Findings = append(res.Findings, Finding{Code: "resource-mapping-ambiguous", Severity: "error", Path: clean, Message: "canonical resource has multiple project mappings: " + quote(previous) + " and " + quote(clean)})
			continue
		}
		if previous, ok := pathOwners[clean]; ok {
			res.Findings = append(res.Findings, Finding{Code: "project-mapping-ambiguous", Severity: "error", Path: clean, Message: "project input maps to multiple canonical resources: " + quote(previous) + " and " + quote(item.Resource)})
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
			res.Findings = append(res.Findings, Finding{Code: "dependency-target-invalid", Severity: "error", Message: "dependsOn target is absent from the normalized model: " + quote(edge.To)})
			continue
		}
		if mappings[edge.To] == "" {
			res.Findings = append(res.Findings, Finding{Code: "dependency-target-unmapped", Severity: "error", Message: "dependsOn target must be mapped to a captured project input: " + quote(edge.From) + " -> " + quote(edge.To)})
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

	projectToResource := map[string]string{}
	for key, path := range mappings {
		projectToResource[path] = key
	}
	observed := map[string][]string{}
	for _, key := range sortedKeys(mappings) {
		projectFile := mappings[key]
		data, ok := req.Captures[projectFile]
		if !ok {
			message := "cannot read staged project input: mapped bytes were not supplied by the Host"
			if captureErr := req.CaptureErrors[projectFile]; captureErr != "" {
				message = "cannot read staged project input: " + captureErr
			}
			res.Findings = append(res.Findings, Finding{Code: "project-input-missing", Severity: "error", Path: projectFile, Message: message})
			continue
		}
		targetPaths, unsupported, err := projectReferences(data)
		if err != nil {
			res.Findings = append(res.Findings, Finding{Code: "project-xml-invalid", Severity: "error", Path: projectFile, Message: err.Error()})
			continue
		}
		if unsupported != "" {
			res.Findings = append(res.Findings, Finding{Code: "project-semantics-unsupported", Severity: "error", Path: projectFile, Message: unsupported})
			continue
		}
		actual := []string{}
		for _, reference := range targetPaths {
			resolved, err := resolveProjectReference(projectFile, reference)
			if err != nil {
				res.Findings = append(res.Findings, Finding{Code: "project-reference-unsupported", Severity: "error", Path: projectFile, Message: "ProjectReference " + quote(reference) + ": " + err.Error()})
				continue
			}
			target, ok := projectToResource[resolved]
			if !ok {
				res.Findings = append(res.Findings, Finding{Code: "project-reference-unmapped", Severity: "error", Path: projectFile, Message: "ProjectReference target " + quote(resolved) + " is not mapped to a canonical resource"})
				continue
			}
			actual = append(actual, target)
		}
		observed[key] = uniqueSorted(actual)
	}
	if hasIncompleteFinding(res.Findings) {
		return sortedResult(res, "incomplete")
	}
	res.Observed = &Observation{Evidence: "literal-unconditional-project-reference-xml", Scope: sortedKeys(mappings), Dependencies: observed}
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
				res.Findings = append(res.Findings, Finding{Code: "dependency-missing", Severity: "error", Path: mappings[from], Message: "canonical dependsOn target " + quote(to) + " is missing from project references for " + quote(from)})
			}
		}
		for _, to := range sortedKeys(got) {
			if !want[to] {
				res.Findings = append(res.Findings, Finding{Code: "dependency-forbidden", Severity: "error", Path: mappings[from], Message: "project reference to " + quote(to) + " is not declared by canonical dependsOn for " + quote(from)})
			}
		}
	}
	if req.Action == "verify" {
		if req.Plan == nil {
			return incomplete(res, "plan-missing", "verify requires a saved plan containing modelDigest and observed resource dependencies")
		}
		if req.Plan.Action != "plan" || req.Plan.Status != "complete" || req.Plan.Adapter != req.Adapter.Name || req.Plan.ModelDigest != req.Model.ModelDigest {
			return fail(res, "plan-model-drift", "plan modelDigest does not match current normalized model")
		}
		if !sameObservation(req.Plan.Observed, res.Observed) {
			res.Findings = append(res.Findings, Finding{Code: "verification-drift", Severity: "error", Message: "observed project references differ from the saved plan"})
		}
	}
	return sortedResult(res, "complete")
}

func DecodeConfig(values map[string]any) (Config, error) {
	if len(values) == 0 {
		return Config{}, nil
	}
	data, err := yaml.Marshal(values)
	if err != nil {
		return Config{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("unsupported or invalid adapter parameter: %w", err)
	}
	return config, nil
}

func baseResult(action, adapter, status, digest string) Result {
	return Result{APIVersion: resultAPIVersion, Adapter: adapter, Action: action, Status: status, ModelDigest: digest, Findings: []Finding{}, Operations: []Operation{}}
}

func fail(res Result, code, message string) Result {
	res.Findings = append(res.Findings, Finding{Code: code, Severity: "error", Message: message})
	return sortedResult(res, "failed")
}

func incomplete(res Result, code, message string) Result {
	res.Findings = append(res.Findings, Finding{Code: code, Severity: "error", Message: message})
	return sortedResult(res, "incomplete")
}

func sortedResult(res Result, status string) Result {
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

func cleanRelativePath(raw string) (string, error) {
	if strings.TrimSpace(raw) == "" {
		return "", errors.New("projectFile must be a non-empty repository-relative path")
	}
	raw = strings.ReplaceAll(raw, "\\", "/")
	if strings.HasPrefix(raw, "/") || (len(raw) > 1 && raw[1] == ':') {
		return "", errors.New("projectFile must be a repository-relative path")
	}
	clean := path.Clean(raw)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, "../") {
		return "", errors.New("projectFile must stay within the staged input directory")
	}
	return clean, nil
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
			if start.Name.Space != "" && start.Name.Space != msbuildNamespace {
				return nil, "project uses an unsupported XML namespace; only unnamespaced or standard MSBuild project XML is supported", nil
			}
			if hasForeignAttribute(start, "Condition") {
				return nil, "project Condition attribute uses an unsupported XML namespace", nil
			}
			if hasAttribute(start, "Condition") {
				return nil, "project-level conditions are outside the captured XML subset", nil
			}
			ancestors = append(ancestors, start)
			continue
		}
		projectNamespace := ancestors[0].Name.Space
		if isMSBuildElement(start.Name.Local) && start.Name.Space != projectNamespace {
			return nil, "MSBuild element " + quote(start.Name.Local) + " uses a different XML namespace from the Project root", nil
		}
		unsupportedNode := ""
		if start.Name.Local == "Import" || start.Name.Local == "ImportGroup" || start.Name.Local == "Sdk" {
			unsupportedNode = "project uses imports or SDK declarations whose evaluated items are outside the captured XML subset"
		}
		if start.Name.Local == "Project" && hasAttribute(start, "Condition") {
			unsupportedNode = "project-level conditions are outside the captured XML subset"
		}
		if start.Name.Local == "ItemGroup" && hasForeignAttribute(start, "Condition") {
			unsupportedNode = "ItemGroup Condition attribute uses an unsupported XML namespace"
		}
		if start.Name.Local == "ProjectReference" {
			inItemGroup := false
			for _, parent := range ancestors {
				switch parent.Name.Local {
				case "ItemGroup":
					inItemGroup = true
					if hasForeignAttribute(parent, "Condition") {
						unsupportedNode = "ItemGroup Condition attribute uses an unsupported XML namespace"
					}
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
			if hasForeignAttribute(start, "Include", "Condition", "Update", "Remove", "Exclude") {
				unsupportedNode = "ProjectReference uses an attribute in an unsupported XML namespace"
			}
			if hasAttribute(start, "Condition") || hasAttribute(start, "Update") || hasAttribute(start, "Remove") || hasAttribute(start, "Exclude") {
				unsupportedNode = "conditioned or transformed ProjectReference items are not supported"
			}
			if !hasAttribute(start, "Include") && !hasForeignAttribute(start, "Include") {
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
			if attribute.Name.Space == "" && attribute.Name.Local == "Include" {
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

func isMSBuildElement(name string) bool {
	switch name {
	case "Project", "Import", "ImportGroup", "Sdk", "ItemGroup", "ProjectReference", "Choose", "When", "Otherwise", "Target":
		return true
	default:
		return false
	}
}

func hasAttribute(element xml.StartElement, name string) bool {
	for _, attribute := range element.Attr {
		if attribute.Name.Space == "" && attribute.Name.Local == name {
			return true
		}
	}
	return false
}

func hasForeignAttribute(element xml.StartElement, names ...string) bool {
	for _, attribute := range element.Attr {
		if attribute.Name.Space == "" {
			continue
		}
		for _, name := range names {
			if attribute.Name.Local == name {
				return true
			}
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
	base := path.Dir(projectFile)
	resolved := path.Clean(path.Join(base, reference))
	if resolved == ".." || strings.HasPrefix(resolved, "../") {
		return "", errors.New("reference resolves outside the captured repository inputs")
	}
	if !strings.EqualFold(path.Ext(resolved), ".csproj") {
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

func hasIncompleteFinding(findings []Finding) bool { return len(findings) > 0 }

func sameObservation(a, b *Observation) bool {
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
