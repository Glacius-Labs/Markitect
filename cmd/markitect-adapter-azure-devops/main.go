// Command markitect-adapter-azure-devops checks captured Azure DevOps Git
// repository default-branch metadata. It never contacts Azure DevOps.
package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

const (
	requestAPIVersion = "markitect.example.org/adapter-request/v1alpha1"
	resultAPIVersion  = "markitect.example.org/adapter-result/v1alpha1"
	semanticModelAPI  = "markitect.example.org/semantic-model/v1alpha1"
	azureAPIVersion   = "7.1"
	evidenceKind      = "captured-azure-devops-git-repository-default-branch"
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
	Target     string         `yaml:"target"`
	Parameters map[string]any `yaml:"parameters"`
}
type adapterParameters struct {
	APIVersion   string              `yaml:"apiVersion"`
	Organization string              `yaml:"organization"`
	ProjectID    string              `yaml:"projectId"`
	Repositories []repositoryMapping `yaml:"repositories"`
}
type repositoryMapping struct {
	Resource     string `yaml:"resource"`
	RepositoryID string `yaml:"repositoryId"`
	CaptureFile  string `yaml:"captureFile"`
}
type semanticModel struct {
	APIVersion       string          `yaml:"apiVersion"`
	ModelDigest      string          `yaml:"modelDigest"`
	ValidationStatus string          `yaml:"validationStatus"`
	Resources        []modelResource `yaml:"resources"`
}
type modelResource struct {
	Identity modelIdentity  `yaml:"identity"`
	Data     map[string]any `yaml:"data"`
}
type modelIdentity struct {
	Key string `yaml:"key"`
}
type result struct {
	APIVersion  string       `yaml:"apiVersion"`
	Adapter     string       `yaml:"adapter"`
	Action      string       `yaml:"action"`
	Status      string       `yaml:"status"`
	ModelDigest string       `yaml:"modelDigest"`
	Target      string       `yaml:"target,omitempty"`
	Observed    *observation `yaml:"observed,omitempty"`
	Findings    []finding    `yaml:"findings"`
	Operations  []operation  `yaml:"operations"`
}
type observation struct {
	Evidence      string               `yaml:"evidence"`
	Scope         []string             `yaml:"scope"`
	CaptureDigest string               `yaml:"captureDigest"`
	Repositories  []observedRepository `yaml:"repositories"`
}
type observedRepository struct {
	Resource      string `yaml:"resource"`
	RepositoryID  string `yaml:"repositoryId"`
	CaptureFile   string `yaml:"captureFile"`
	DefaultBranch string `yaml:"defaultBranch"`
}
type finding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Path     string `yaml:"path,omitempty"`
	Message  string `yaml:"message"`
}
type operation struct{}

// This envelope records the request metadata that is not present in Azure's response body.
type capture struct {
	Request struct {
		Method string `json:"method"`
		URL    string `json:"url"`
	} `json:"request"`
	Response struct {
		Status int             `json:"status"`
		Body   json.RawMessage `json:"body"`
	} `json:"response"`
}
type gitRepository struct {
	ID            string `json:"id"`
	DefaultBranch string `json:"defaultBranch"`
	Project       struct {
		ID string `json:"id"`
	} `json:"project"`
}

var guidPattern = regexp.MustCompile(`(?i)^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
var organizationPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9-]*$`)

func main() {
	decoder := yaml.NewDecoder(os.Stdin)
	var req request
	if err := decoder.Decode(&req); err != nil {
		fatal(fmt.Errorf("decode adapter request: %w", err))
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		fatal(errors.New("adapter request must contain exactly one YAML document"))
	}
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	if err := encoder.Encode(run(req)); err != nil {
		fatal(err)
	}
	_ = encoder.Close()
}
func fatal(err error) {
	out := baseResult("", "", "", "failed", "")
	out.Findings = append(out.Findings, finding{Code: "invalid-request", Severity: "error", Message: err.Error()})
	encoder := yaml.NewEncoder(os.Stdout)
	encoder.SetIndent(2)
	_ = encoder.Encode(out)
	os.Exit(2)
}

func run(req request) result {
	res := baseResult(req.Action, req.Adapter.Name, req.Model.ModelDigest, "complete", req.Adapter.Target)
	if req.APIVersion != requestAPIVersion {
		return fail(res, "unsupported-request-version", "expected apiVersion "+requestAPIVersion)
	}
	if req.Action != "observe" && req.Action != "plan" && req.Action != "verify" {
		return fail(res, "unsupported-action", "this adapter supports only observe, plan, and verify")
	}
	if req.Adapter.Name == "" || req.Adapter.Type != "command" || req.Adapter.Version == "" {
		return fail(res, "adapter-identity-mismatch", "adapter requires a name, type command, and version")
	}
	if req.Model.APIVersion != semanticModelAPI || req.Model.ModelDigest == "" {
		return fail(res, "unsupported-model", "a supported semantic model apiVersion and modelDigest are required")
	}
	if req.Model.ValidationStatus != "passed" {
		return fail(res, "model-not-validated", "adapter requires validationStatus passed")
	}
	parameters, err := decodeParameters(req.Adapter.Parameters)
	if err != nil {
		return fail(res, "adapter-parameters-invalid", err.Error())
	}
	if parameters.APIVersion != azureAPIVersion || !organizationPattern.MatchString(parameters.Organization) || !guidPattern.MatchString(parameters.ProjectID) {
		return fail(res, "target-parameters-invalid", "apiVersion must be 7.1, organization a single URL-safe name, and projectId a GUID")
	}
	if req.Adapter.Target != targetID(parameters.Organization, parameters.ProjectID) {
		return fail(res, "target-identity-mismatch", "adapter target must exactly identify its Azure DevOps organization and project")
	}
	resources := make(map[string]modelResource, len(req.Model.Resources))
	for _, resource := range req.Model.Resources {
		if resource.Identity.Key != "" {
			resources[resource.Identity.Key] = resource
		}
	}
	if len(parameters.Repositories) == 0 {
		return incomplete(res, "repository-mappings-missing", "adapter.parameters.repositories must declare in-scope canonical resources and capture files")
	}
	seenResource, seenRepo, seenFile := map[string]bool{}, map[string]bool{}, map[string]bool{}
	obs := &observation{Evidence: evidenceKind, Scope: []string{}, Repositories: []observedRepository{}}
	digests := []string{}
	for _, mapping := range parameters.Repositories {
		if mapping.Resource == "" || mapping.RepositoryID == "" || mapping.CaptureFile == "" || !guidPattern.MatchString(mapping.RepositoryID) {
			return incomplete(res, "repository-mapping-invalid", "each mapping requires an exact resource identity, repository GUID, and captured file")
		}
		if _, ok := resources[mapping.Resource]; !ok {
			return incomplete(res, "resource-mapping-invalid", "mapping must reference an existing exact semantic-model identity.key: "+mapping.Resource)
		}
		fileKey := filepath.ToSlash(filepath.Clean(mapping.CaptureFile))
		if seenResource[mapping.Resource] || seenRepo[strings.ToLower(mapping.RepositoryID)] || seenFile[fileKey] {
			return incomplete(res, "repository-mapping-ambiguous", "resource, repository, and capture file mappings must be unique")
		}
		seenResource[mapping.Resource], seenRepo[strings.ToLower(mapping.RepositoryID)], seenFile[fileKey] = true, true, true
		path, pathErr := safeCapturePath(mapping.CaptureFile)
		if pathErr != nil {
			return incomplete(res, "capture-file-invalid", pathErr.Error())
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return incomplete(res, "capture-file-missing", "cannot read staged capture "+mapping.CaptureFile)
		}
		_, repo, capErr := parseCapture(data, parameters, mapping)
		if capErr != nil {
			return incomplete(res, "capture-invalid", mapping.CaptureFile+": "+capErr.Error())
		}
		digests = append(digests, fileKey+":"+digest(data))
		obs.Scope = append(obs.Scope, mapping.Resource)
		obs.Repositories = append(obs.Repositories, observedRepository{Resource: mapping.Resource, RepositoryID: mapping.RepositoryID, CaptureFile: fileKey, DefaultBranch: repo.DefaultBranch})
	}
	sort.Strings(obs.Scope)
	sort.Strings(digests)
	sort.Slice(obs.Repositories, func(i, j int) bool { return obs.Repositories[i].Resource < obs.Repositories[j].Resource })
	obs.CaptureDigest = digest([]byte(strings.Join(digests, "\n")))
	res.Observed = obs
	if req.Action == "observe" {
		return res
	}
	if req.Action == "plan" {
		if req.Observation == nil || req.Observation.Status != "complete" || req.Observation.Action != "observe" || req.Observation.Adapter != req.Adapter.Name || req.Observation.ModelDigest != req.Model.ModelDigest || req.Observation.Target != req.Adapter.Target || !sameObservation(req.Observation.Observed, obs) {
			return fail(res, "observation-drift", "plan requires the same completed observation, adapter, model, and target")
		}
		res.Findings = compareDesired(resources, obs)
		res.Operations = []operation{} // Evidence findings are never executable operations.
		return res
	}
	if req.Plan == nil || req.Plan.Action != "plan" || req.Plan.Status != "complete" || req.Plan.Adapter != req.Adapter.Name || req.Plan.ModelDigest != req.Model.ModelDigest || req.Plan.Target != req.Adapter.Target || !sameObservation(req.Plan.Observed, obs) {
		return fail(res, "verification-drift", "verify requires a completed plan for this model, target, and unchanged capture")
	}
	res.Findings = compareDesired(resources, obs)
	res.Operations = []operation{}
	if len(res.Findings) > 0 {
		res.Status = "failed"
	}
	return res
}

func parseCapture(data []byte, params adapterParameters, mapping repositoryMapping) (capture, gitRepository, error) {
	var cap capture
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&cap); err != nil {
		return cap, gitRepository{}, fmt.Errorf("invalid JSON envelope: %w", err)
	}
	var extra any
	if decoder.Decode(&extra) != io.EOF {
		return cap, gitRepository{}, errors.New("capture must contain exactly one JSON value")
	}
	if cap.Request.Method != "GET" {
		return cap, gitRepository{}, errors.New("captured request method must be GET")
	}
	if err := validateRequestURL(cap.Request.URL, params, mapping); err != nil {
		return cap, gitRepository{}, err
	}
	if cap.Response.Status != 200 {
		return cap, gitRepository{}, fmt.Errorf("captured HTTP status %d is not a successful repository response", cap.Response.Status)
	}
	var repo gitRepository
	if len(cap.Response.Body) == 0 || string(cap.Response.Body) == "null" {
		return cap, repo, errors.New("response body is missing")
	}
	if err := json.Unmarshal(cap.Response.Body, &repo); err != nil {
		return cap, repo, fmt.Errorf("invalid GitRepository response body: %w", err)
	}
	if !strings.EqualFold(repo.ID, mapping.RepositoryID) || !strings.EqualFold(repo.Project.ID, params.ProjectID) {
		return cap, repo, errors.New("response repository or project ID does not match the mapped target")
	}
	if !validBranch(repo.DefaultBranch) {
		return cap, repo, errors.New("defaultBranch must be a non-empty refs/heads/<name> value without whitespace")
	}
	return cap, repo, nil
}
func validateRequestURL(raw string, params adapterParameters, mapping repositoryMapping) error {
	u, err := url.Parse(raw)
	if err != nil || u.Scheme != "https" || u.Host != "dev.azure.com" || u.User != nil || u.Fragment != "" {
		return errors.New("captured URL must be an HTTPS dev.azure.com Get Repository URL")
	}
	parts := strings.Split(strings.Trim(u.EscapedPath(), "/"), "/")
	if len(parts) != 6 || parts[1] != params.ProjectID || parts[2] != "_apis" || parts[3] != "git" || parts[4] != "repositories" || parts[5] != mapping.RepositoryID {
		return errors.New("captured URL path does not match the selected project and repository")
	}
	org, err := url.PathUnescape(parts[0])
	if err != nil || org != params.Organization {
		return errors.New("captured URL organization does not match the selected target")
	}
	query := u.Query()
	if len(query) != 1 || query.Get("api-version") != azureAPIVersion {
		return errors.New("captured URL must use exactly api-version=7.1")
	}
	return nil
}
func compareDesired(resources map[string]modelResource, observed *observation) []finding {
	findings := []finding{}
	for _, got := range observed.Repositories {
		resource := resources[got.Resource]
		value, exists := resource.Data["defaultBranch"]
		branch, ok := value.(string)
		if !exists || !ok || !validBranch(branch) {
			findings = append(findings, finding{Code: "canonical-default-branch-invalid", Severity: "error", Path: got.Resource, Message: "mapped resource data.defaultBranch must be a refs/heads/<name> string"})
		} else if branch != got.DefaultBranch {
			findings = append(findings, finding{Code: "default-branch-drift", Severity: "error", Path: got.CaptureFile, Message: fmt.Sprintf("canonical defaultBranch %q differs from captured repository value %q", branch, got.DefaultBranch)})
		}
	}
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Path != findings[j].Path {
			return findings[i].Path < findings[j].Path
		}
		return findings[i].Code < findings[j].Code
	})
	return findings
}
func decodeParameters(values map[string]any) (adapterParameters, error) {
	data, err := yaml.Marshal(values)
	if err != nil {
		return adapterParameters{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var params adapterParameters
	if err := decoder.Decode(&params); err != nil {
		return params, fmt.Errorf("unsupported or invalid adapter parameter: %w", err)
	}
	return params, nil
}
func safeCapturePath(relative string) (string, error) {
	if strings.TrimSpace(relative) == "" || filepath.IsAbs(relative) || filepath.VolumeName(relative) != "" {
		return "", errors.New("captureFile must be a staged-input-relative path")
	}
	clean := filepath.Clean(relative)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("captureFile must stay within staged inputs")
	}
	root, err := filepath.Abs(".")
	if err != nil {
		return "", err
	}
	resolvedRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(filepath.Join(root, clean))
	if err != nil {
		return "", err
	}
	rel, err := filepath.Rel(resolvedRoot, resolved)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", errors.New("captureFile resolves outside staged inputs")
	}
	return resolved, nil
}
func targetID(organization, projectID string) string {
	return "azure-devops://" + organization + "/" + strings.ToLower(projectID)
}
func validBranch(value string) bool {
	return strings.HasPrefix(value, "refs/heads/") && len(value) > len("refs/heads/") && !strings.ContainsAny(value, " \t\r\n")
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func baseResult(action, adapter, modelDigest, status, target string) result {
	return result{APIVersion: resultAPIVersion, Adapter: adapter, Action: action, Status: status, ModelDigest: modelDigest, Target: target, Findings: []finding{}, Operations: []operation{}}
}
func fail(res result, code, message string) result {
	res.Status = "failed"
	res.Findings = append(res.Findings, finding{Code: code, Severity: "error", Message: message})
	return res
}
func incomplete(res result, code, message string) result {
	res.Status = "incomplete"
	res.Findings = append(res.Findings, finding{Code: code, Severity: "error", Message: message})
	return res
}
func sameObservation(a, b *observation) bool {
	if a == nil || b == nil || a.Evidence != b.Evidence || a.CaptureDigest != b.CaptureDigest || strings.Join(a.Scope, "\x00") != strings.Join(b.Scope, "\x00") || len(a.Repositories) != len(b.Repositories) {
		return false
	}
	for i := range a.Repositories {
		if a.Repositories[i] != b.Repositories[i] {
			return false
		}
	}
	return true
}
