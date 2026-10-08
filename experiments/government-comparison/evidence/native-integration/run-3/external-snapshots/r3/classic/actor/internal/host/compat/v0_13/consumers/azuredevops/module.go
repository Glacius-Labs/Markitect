// Package azuredevops compares explicitly supplied Azure DevOps Git metadata
// bytes with a normalized Core model. It performs no filesystem or network IO.
package azuredevops

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"path"
	"regexp"
	"sort"
	"strings"

	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
	"go.yaml.in/yaml/v3"
)

const (
	requestAPIVersion = "markitect.example.org/adapter-request/v1alpha1"
	resultAPIVersion  = "markitect.example.org/adapter-result/v1alpha1"
	semanticModelAPI  = "markitect.example.org/semantic-model/v1alpha1"
	azureAPIVersion   = "7.1"
	evidenceKind      = "captured-azure-devops-git-repository-default-branch"
	maxCaptureBytes   = 1 << 20
)

type Config struct {
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
type Result struct {
	APIVersion  string       `yaml:"apiVersion"`
	Adapter     string       `yaml:"adapter"`
	Action      string       `yaml:"action"`
	Status      string       `yaml:"status"`
	ModelDigest string       `yaml:"modelDigest"`
	Target      string       `yaml:"target,omitempty"`
	Observed    *Observation `yaml:"observed,omitempty"`
	Findings    []Finding    `yaml:"findings"`
	Operations  []Operation  `yaml:"operations"`
}
type Observation struct {
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
type Finding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Path     string `yaml:"path,omitempty"`
	Message  string `yaml:"message"`
}
type Operation struct{}

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

func Run(req Request) Result {
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
	if req.ConfigError != "" {
		return fail(res, "adapter-parameters-invalid", req.ConfigError)
	}
	parameters := req.Config
	if parameters.APIVersion != azureAPIVersion || !organizationPattern.MatchString(parameters.Organization) || !guidPattern.MatchString(parameters.ProjectID) {
		return fail(res, "target-parameters-invalid", "apiVersion must be 7.1, organization a single URL-safe name, and projectId a GUID")
	}
	if req.Adapter.Target != targetID(parameters.Organization, parameters.ProjectID) {
		return fail(res, "target-identity-mismatch", "adapter target must exactly identify its Azure DevOps organization and project")
	}
	resources := make(map[string]core.ModelResource, len(req.Model.Resources))
	for _, resource := range req.Model.Resources {
		if resource.Identity.Key != "" {
			resources[resource.Identity.Key] = resource
		}
	}
	if len(parameters.Repositories) == 0 {
		return incomplete(res, "repository-mappings-missing", "adapter.parameters.repositories must declare in-scope canonical resources and capture files")
	}
	seenResource, seenRepo, seenFile := map[string]bool{}, map[string]bool{}, map[string]bool{}
	obs := &Observation{Evidence: evidenceKind, Scope: []string{}, Repositories: []observedRepository{}}
	digests := []string{}
	for _, mapping := range parameters.Repositories {
		if mapping.Resource == "" || mapping.RepositoryID == "" || mapping.CaptureFile == "" || !guidPattern.MatchString(mapping.RepositoryID) {
			return incomplete(res, "repository-mapping-invalid", "each mapping requires an exact resource identity, repository GUID, and captured file")
		}
		if err := cleanCapturePath(mapping.CaptureFile); err != nil {
			return incomplete(res, "capture-file-invalid", err.Error())
		}
		if _, ok := resources[mapping.Resource]; !ok {
			return incomplete(res, "resource-mapping-invalid", "mapping must reference an existing exact semantic-model identity.key: "+mapping.Resource)
		}
		fileKey := path.Clean(mapping.CaptureFile)
		if seenResource[mapping.Resource] || seenRepo[strings.ToLower(mapping.RepositoryID)] || seenFile[fileKey] {
			return incomplete(res, "repository-mapping-ambiguous", "resource, repository, and capture file mappings must be unique")
		}
		seenResource[mapping.Resource], seenRepo[strings.ToLower(mapping.RepositoryID)], seenFile[fileKey] = true, true, true
		data, ok := req.Captures[fileKey]
		if !ok {
			message := "mapped capture bytes were not supplied by the Host"
			if captureErr := req.CaptureErrors[fileKey]; captureErr != "" {
				message = captureErr
			}
			return incomplete(res, "capture-file-invalid", mapping.CaptureFile+": "+message)
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
		res.Operations = []Operation{} // Evidence findings are never executable operations.
		return res
	}
	if req.Plan == nil || req.Plan.Action != "plan" || req.Plan.Status != "complete" || req.Plan.Adapter != req.Adapter.Name || req.Plan.ModelDigest != req.Model.ModelDigest || req.Plan.Target != req.Adapter.Target || !sameObservation(req.Plan.Observed, obs) {
		return fail(res, "verification-drift", "verify requires a completed plan for this model, target, and unchanged capture")
	}
	res.Findings = compareDesired(resources, obs)
	res.Operations = []Operation{}
	if len(res.Findings) > 0 {
		res.Status = "failed"
	}
	return res
}

func parseCapture(data []byte, params Config, mapping repositoryMapping) (capture, gitRepository, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return capture{}, gitRepository{}, err
	}
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
func validateRequestURL(raw string, params Config, mapping repositoryMapping) error {
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
	query, err := url.ParseQuery(u.RawQuery)
	if err != nil || len(query) != 1 || len(query["api-version"]) != 1 || query["api-version"][0] != azureAPIVersion {
		return errors.New("captured URL must use exactly api-version=7.1")
	}
	return nil
}

// rejectDuplicateJSONKeys keeps object interpretation unambiguous without
// rejecting provider fields this adapter does not consume.
func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	if err := walkJSONValue(decoder); err != nil {
		return fmt.Errorf("invalid or ambiguous JSON capture: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		return errors.New("capture must contain exactly one JSON value")
	}
	return nil
}

func walkJSONValue(decoder *json.Decoder) error {
	token, err := decoder.Token()
	if err != nil {
		return err
	}
	switch value := token.(type) {
	case json.Delim:
		switch value {
		case '{':
			seen := map[string]bool{}
			for decoder.More() {
				keyToken, err := decoder.Token()
				if err != nil {
					return err
				}
				key, ok := keyToken.(string)
				if !ok {
					return errors.New("JSON object key is not a string")
				}
				normalizedKey := strings.ToLower(key)
				if seen[normalizedKey] {
					return fmt.Errorf("duplicate JSON object key ignoring case %q", key)
				}
				seen[normalizedKey] = true
				if err := walkJSONValue(decoder); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim('}') {
				return errors.New("unterminated JSON object")
			}
		case '[':
			for decoder.More() {
				if err := walkJSONValue(decoder); err != nil {
					return err
				}
			}
			end, err := decoder.Token()
			if err != nil || end != json.Delim(']') {
				return errors.New("unterminated JSON array")
			}
		default:
			return fmt.Errorf("unexpected JSON delimiter %q", value)
		}
	}
	return nil
}
func compareDesired(resources map[string]core.ModelResource, observed *Observation) []Finding {
	findings := []Finding{}
	for _, got := range observed.Repositories {
		resource := resources[got.Resource]
		value, exists := resource.Data["defaultBranch"]
		branch, ok := value.(string)
		if !exists || !ok || !validBranch(branch) {
			findings = append(findings, Finding{Code: "canonical-default-branch-invalid", Severity: "error", Path: got.Resource, Message: "mapped resource data.defaultBranch must be a refs/heads/<name> string"})
		} else if branch != got.DefaultBranch {
			findings = append(findings, Finding{Code: "default-branch-drift", Severity: "error", Path: got.CaptureFile, Message: fmt.Sprintf("canonical defaultBranch %q differs from captured repository value %q", branch, got.DefaultBranch)})
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
func DecodeConfig(values map[string]any) (Config, error) {
	data, err := yaml.Marshal(values)
	if err != nil {
		return Config{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var params Config
	if err := decoder.Decode(&params); err != nil {
		return params, fmt.Errorf("unsupported or invalid adapter parameter: %w", err)
	}
	return params, nil
}

// CapturePaths returns only clean exact files named by this module's config.
func CapturePaths(config Config) []string {
	seen := map[string]bool{}
	var paths []string
	for _, mapping := range config.Repositories {
		if err := cleanCapturePath(mapping.CaptureFile); err == nil {
			p := path.Clean(mapping.CaptureFile)
			if !seen[p] {
				seen[p] = true
				paths = append(paths, p)
			}
		}
	}
	sort.Strings(paths)
	return paths
}
func cleanCapturePath(raw string) error {
	if raw == "" || strings.ContainsAny(raw, "\\:\x00") || path.IsAbs(raw) || path.Clean(raw) != raw {
		return errors.New("captureFile must be a clean staged-input-relative path")
	}
	for _, part := range strings.Split(raw, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return errors.New("captureFile must stay within staged inputs")
		}
	}
	return nil
}

// InvalidRequest creates the protocol-shaped failure used by the Host.
func InvalidRequest(message string) Result {
	res := baseResult("", "", "", "failed", "")
	res.Findings = append(res.Findings, Finding{Code: "invalid-request", Severity: "error", Message: message})
	return res
}

func targetID(organization, projectID string) string {
	return "azure-devops://" + organization + "/" + strings.ToLower(projectID)
}
func validBranch(value string) bool {
	return strings.HasPrefix(value, "refs/heads/") && len(value) > len("refs/heads/") && !strings.ContainsAny(value, " \t\r\n")
}
func digest(data []byte) string { sum := sha256.Sum256(data); return hex.EncodeToString(sum[:]) }
func baseResult(action, adapter, modelDigest, status, target string) Result {
	return Result{APIVersion: resultAPIVersion, Adapter: adapter, Action: action, Status: status, ModelDigest: modelDigest, Target: target, Findings: []Finding{}, Operations: []Operation{}}
}
func fail(res Result, code, message string) Result {
	res.Status = "failed"
	res.Findings = append(res.Findings, Finding{Code: code, Severity: "error", Message: message})
	return res
}
func incomplete(res Result, code, message string) Result {
	res.Status = "incomplete"
	res.Findings = append(res.Findings, Finding{Code: code, Severity: "error", Message: message})
	return res
}
func sameObservation(a, b *Observation) bool {
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
