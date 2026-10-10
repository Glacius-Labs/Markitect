// Package github compares explicitly supplied GitHub repository metadata bytes
// with a normalized Core model. It performs no filesystem or network access.
package github

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
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
	captureAPIVersion = "markitect.github.example.org/repository-capture/v1alpha1"
	githubAPIVersion  = "2026-03-10"
	maxCaptureBytes   = 1 << 20
)

type Config struct {
	Repositories []repositoryMapping `yaml:"repositories"`
}
type repositoryMapping struct {
	Resource     string `yaml:"resource"`
	EvidenceFile string `yaml:"evidenceFile"`
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
	APIVersion  string      `yaml:"apiVersion"`
	Adapter     string      `yaml:"adapter"`
	Action      string      `yaml:"action"`
	Status      string      `yaml:"status"`
	ModelDigest string      `yaml:"modelDigest"`
	Target      string      `yaml:"target,omitempty"`
	Observed    Observation `yaml:"observed"`
	Findings    []Finding   `yaml:"findings"`
	Operations  []Operation `yaml:"operations"`
}
type Observation struct {
	Evidence         string                           `yaml:"evidence"`
	GithubAPIVersion string                           `yaml:"githubApiVersion"`
	Scope            []string                         `yaml:"scope"`
	Repositories     map[string]repositoryObservation `yaml:"repositories"`
}
type repositoryObservation struct {
	RequestPath   string `yaml:"requestPath"`
	FullName      string `yaml:"fullName"`
	DefaultBranch string `yaml:"defaultBranch"`
	CaptureDigest string `yaml:"captureDigest"`
}
type Finding struct {
	Code     string `yaml:"code"`
	Severity string `yaml:"severity"`
	Path     string `yaml:"path,omitempty"`
	Message  string `yaml:"message"`
}
type Operation struct {
	ID      string         `yaml:"id"`
	Action  string         `yaml:"action"`
	Target  string         `yaml:"target"`
	Desired map[string]any `yaml:"desired,omitempty"`
}

type captureEnvelope struct {
	APIVersion string          `json:"apiVersion"`
	Request    captureRequest  `json:"request"`
	Response   captureResponse `json:"response"`
}

type captureRequest struct {
	Method     string `json:"method"`
	Path       string `json:"path"`
	APIVersion string `json:"apiVersion"`
}

type captureResponse struct {
	StatusCode int             `json:"statusCode"`
	Body       json.RawMessage `json:"body"`
}

type repositoryBody struct {
	FullName      string `json:"full_name"`
	DefaultBranch string `json:"default_branch"`
}

func Run(req Request) Result {
	res := baseResult(req.Action, req.Adapter.Name, "complete", req.Model.ModelDigest, req.Adapter.Target)
	if req.APIVersion != requestAPIVersion {
		return fail(res, "unsupported-request-version", "expected apiVersion "+requestAPIVersion)
	}
	if req.Action != "observe" && req.Action != "plan" && req.Action != "verify" {
		return fail(res, "unsupported-action", "action "+quote(req.Action)+" is not supported; this adapter is read-only")
	}
	if req.Adapter.Name == "" || req.Adapter.Type != "command" || req.Adapter.Version == "" {
		return fail(res, "adapter-identity-invalid", "adapter identity requires a name, type command, and version")
	}
	if req.Model.APIVersion != semanticModelAPI {
		return fail(res, "unsupported-model-version", "unsupported normalized semantic model apiVersion "+quote(req.Model.APIVersion))
	}
	if req.Model.ModelDigest == "" || req.Model.ValidationStatus != "passed" {
		return incomplete(res, "model-not-validated", "a normalized model digest and validationStatus passed are required")
	}
	if req.ConfigError != "" {
		return fail(res, "adapter-parameters-invalid", req.ConfigError)
	}
	if len(req.Config.Repositories) == 0 {
		return incomplete(res, "repository-mappings-missing", "adapter.parameters.repositories must select the canonical repositories in scope")
	}

	resources := make(map[string]core.ModelResource, len(req.Model.Resources))
	for _, resource := range req.Model.Resources {
		if resource.Identity.Key != "" {
			resources[resource.Identity.Key] = resource
		}
	}
	mappings := append([]repositoryMapping(nil), req.Config.Repositories...)
	sort.Slice(mappings, func(i, j int) bool { return mappings[i].Resource < mappings[j].Resource })
	seenResources, seenFiles := map[string]bool{}, map[string]bool{}
	for _, mapping := range mappings {
		if mapping.Resource == "" || seenResources[mapping.Resource] {
			return incomplete(res, "repository-mapping-ambiguous", "each exact canonical resource may be mapped only once")
		}
		seenResources[mapping.Resource] = true
		if mapping.EvidenceFile == "" || seenFiles[strings.ToLower(mapping.EvidenceFile)] {
			return incomplete(res, "capture-mapping-ambiguous", "each declared capture input must be mapped exactly once")
		}
		seenFiles[strings.ToLower(mapping.EvidenceFile)] = true
		if _, err := cleanRelativePath(mapping.EvidenceFile); err != nil {
			return incomplete(res, "capture-path-invalid", err.Error())
		}
		if _, ok := resources[mapping.Resource]; !ok {
			return incomplete(res, "resource-mapping-invalid", "mapping must name an existing exact canonical identity.key: "+quote(mapping.Resource))
		}
	}

	observed := Observation{
		Evidence:         "github-rest-repository-metadata-capture",
		GithubAPIVersion: githubAPIVersion,
		Scope:            make([]string, 0, len(mappings)),
		Repositories:     make(map[string]repositoryObservation, len(mappings)),
	}
	for _, mapping := range mappings {
		resource := resources[mapping.Resource]
		if resource.Identity.Kind != "Repository" {
			return incomplete(res, "resource-kind-unsupported", "mapped resource "+quote(mapping.Resource)+" must have kind Repository")
		}
		fullName, ok := resource.Data["fullName"].(string)
		if !ok || !validFullName(fullName) {
			return incomplete(res, "canonical-repository-invalid", "Repository "+quote(mapping.Resource)+" must declare a valid top-level data.fullName as owner/repository")
		}
		defaultBranch, ok := resource.Data["defaultBranch"].(string)
		if !ok || strings.TrimSpace(defaultBranch) == "" {
			return incomplete(res, "canonical-repository-invalid", "Repository "+quote(mapping.Resource)+" must declare non-empty top-level data.defaultBranch")
		}
		data, ok := req.Captures[mapping.EvidenceFile]
		if !ok {
			message := "mapped capture bytes were not supplied by the Host"
			if captureErr := req.CaptureErrors[mapping.EvidenceFile]; captureErr != "" {
				message = captureErr
			}
			return incomplete(res, "capture-invalid", mapping.EvidenceFile+": "+message)
		}
		capture, err := decodeCapture(data)
		if err != nil {
			return incomplete(res, "capture-invalid", mapping.EvidenceFile+": "+err.Error())
		}
		if capture.APIVersion != captureAPIVersion {
			return incomplete(res, "capture-version-unsupported", mapping.EvidenceFile+": expected apiVersion "+captureAPIVersion)
		}
		if capture.Request.Method != "GET" || capture.Request.APIVersion != githubAPIVersion {
			return incomplete(res, "capture-request-unsupported", mapping.EvidenceFile+": request must be GET using GitHub API version "+githubAPIVersion)
		}
		owner, repository, _ := strings.Cut(fullName, "/")
		wantPath := "/repos/" + owner + "/" + repository
		if capture.Request.Path != wantPath {
			return incomplete(res, "capture-target-mismatch", mapping.EvidenceFile+": requested path does not match canonical fullName "+quote(fullName))
		}
		if capture.Response.StatusCode != 200 {
			code := "capture-http-status"
			if capture.Response.StatusCode == 403 {
				code = "capture-unauthorized"
			} else if capture.Response.StatusCode == 404 {
				code = "capture-repository-missing"
			}
			return incomplete(res, code, mapping.EvidenceFile+": captured GET returned HTTP "+fmt.Sprint(capture.Response.StatusCode)+"; repository conformance was not evaluated")
		}
		var body repositoryBody
		if len(capture.Response.Body) == 0 || string(capture.Response.Body) == "null" {
			return incomplete(res, "capture-body-missing", mapping.EvidenceFile+": successful capture has no JSON response body")
		}
		if err := json.Unmarshal(capture.Response.Body, &body); err != nil {
			return incomplete(res, "capture-body-invalid", mapping.EvidenceFile+": decode GitHub repository response: "+err.Error())
		}
		if body.FullName != fullName {
			return incomplete(res, "capture-target-mismatch", mapping.EvidenceFile+": response full_name does not match canonical fullName "+quote(fullName))
		}
		if strings.TrimSpace(body.DefaultBranch) == "" {
			return incomplete(res, "capture-field-missing", mapping.EvidenceFile+": response default_branch is missing or empty")
		}
		captureHash := sha256.Sum256(data)
		observed.Scope = append(observed.Scope, mapping.Resource)
		observed.Repositories[mapping.Resource] = repositoryObservation{
			RequestPath: capture.Request.Path, FullName: body.FullName, DefaultBranch: body.DefaultBranch,
			CaptureDigest: "sha256:" + hex.EncodeToString(captureHash[:]),
		}
		if body.DefaultBranch != defaultBranch {
			res.Findings = append(res.Findings, Finding{
				Code: "default-branch-drift", Severity: "error", Path: mapping.EvidenceFile,
				Message: "captured default_branch " + quote(body.DefaultBranch) + " differs from canonical data.defaultBranch " + quote(defaultBranch) + " for " + quote(mapping.Resource),
			})
		}
	}
	res.Observed = observed
	if req.Action == "plan" {
		if req.Observation == nil || req.Observation.APIVersion != resultAPIVersion || req.Observation.Adapter != req.Adapter.Name || req.Observation.Target != req.Adapter.Target || req.Observation.Action != "observe" || req.Observation.Status != "complete" || req.Observation.ModelDigest != req.Model.ModelDigest || !sameObserved(req.Observation.Observed, res.Observed) {
			return incomplete(res, "observation-invalid", "plan requires the matching complete observe result for this adapter, model and captured input set")
		}
	}
	if req.Action == "verify" {
		if req.Observation == nil || req.Plan == nil || req.Observation.APIVersion != resultAPIVersion || req.Plan.APIVersion != resultAPIVersion || req.Observation.Adapter != req.Adapter.Name || req.Observation.Target != req.Adapter.Target || req.Observation.Action != "observe" || req.Observation.Status != "complete" || req.Observation.ModelDigest != req.Model.ModelDigest || req.Plan.Adapter != req.Adapter.Name || req.Plan.Target != req.Adapter.Target || req.Plan.Action != "plan" || req.Plan.Status != "complete" || req.Plan.ModelDigest != req.Model.ModelDigest {
			return incomplete(res, "plan-invalid", "verify requires matching complete observation and plan results for this adapter and model")
		}
		if !sameObserved(req.Observation.Observed, req.Plan.Observed) || !sameObserved(req.Observation.Observed, res.Observed) {
			return fail(res, "verification-drift", "current fixed-input capture differs from the observation bound into the saved plan")
		}
		planned := res
		planned.Action = "plan"
		if !sameResult(req.Plan, planned) {
			return fail(res, "saved-plan-mismatch", "saved plan findings or evidence differ from a fresh deterministic plan over the captured inputs")
		}
	}
	if req.Action == "verify" && len(res.Findings) > 0 {
		return sortedResult(res, "failed")
	}
	return sortedResult(res, "complete")
}

func DecodeConfig(values map[string]any) (Config, error) {
	encoded, err := yaml.Marshal(values)
	if err != nil {
		return Config{}, err
	}
	decoder := yaml.NewDecoder(bytes.NewReader(encoded))
	decoder.KnownFields(true)
	var config Config
	if err := decoder.Decode(&config); err != nil {
		return config, fmt.Errorf("unsupported or invalid adapter parameter: %w", err)
	}
	return config, nil
}

// CapturePaths returns only the exact clean capture paths named by this config.
func CapturePaths(config Config) []string {
	seen := map[string]bool{}
	var paths []string
	for _, mapping := range config.Repositories {
		if _, err := cleanRelativePath(mapping.EvidenceFile); err == nil && !seen[mapping.EvidenceFile] {
			seen[mapping.EvidenceFile] = true
			paths = append(paths, mapping.EvidenceFile)
		}
	}
	sort.Strings(paths)
	return paths
}

// InvalidRequest creates the protocol-shaped failure used by the Host.
func InvalidRequest(message string) Result {
	res := baseResult("", "", "failed", "", "")
	res.Findings = append(res.Findings, Finding{Code: "invalid-request", Severity: "error", Message: message})
	return sortedResult(res, "failed")
}

func decodeCapture(data []byte) (captureEnvelope, error) {
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return captureEnvelope{}, fmt.Errorf("capture JSON is ambiguous or malformed: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var capture captureEnvelope
	if err := decoder.Decode(&capture); err != nil {
		return capture, fmt.Errorf("decode capture envelope: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		return capture, errors.New("capture must contain exactly one JSON value")
	}
	if capture.Request.Method == "" || capture.Request.Path == "" || capture.Request.APIVersion == "" || capture.Response.StatusCode < 100 || capture.Response.StatusCode > 599 {
		return capture, errors.New("capture request and HTTP response status are required")
	}
	return capture, nil
}

// rejectDuplicateJSONKeys rejects duplicate object keys at every depth before
// decoding into structs. encoding/json otherwise accepts duplicates and uses
// the last encountered value, making captured evidence ambiguous.
func rejectDuplicateJSONKeys(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	first, err := decoder.Token()
	if err != nil {
		return err
	}
	if err := scanJSONValue(decoder, first); err != nil {
		return err
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("capture must contain exactly one JSON value")
		}
		return err
	}
	return nil
}

func scanJSONValue(decoder *json.Decoder, token json.Token) error {
	delim, isDelimiter := token.(json.Delim)
	if !isDelimiter {
		return nil
	}
	switch delim {
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
				return fmt.Errorf("duplicate JSON object key %q", key)
			}
			seen[normalizedKey] = true
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := scanJSONValue(decoder, value); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim('}') {
			return errors.New("unterminated JSON object")
		}
	case '[':
		for decoder.More() {
			value, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := scanJSONValue(decoder, value); err != nil {
				return err
			}
		}
		end, err := decoder.Token()
		if err != nil {
			return err
		}
		if end != json.Delim(']') {
			return errors.New("unterminated JSON array")
		}
	default:
		return errors.New("unexpected JSON delimiter")
	}
	return nil
}

func cleanRelativePath(raw string) (string, error) {
	if raw == "" || strings.ContainsAny(raw, "\\:\x00") || path.IsAbs(raw) || path.Clean(raw) != raw {
		return "", errors.New("evidenceFile must be a clean repository-relative path")
	}
	for _, part := range strings.Split(raw, "/") {
		if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return "", errors.New("evidenceFile must stay within staged declared inputs")
		}
	}
	return raw, nil
}

func validFullName(value string) bool {
	if strings.TrimSpace(value) != value || value == "" || strings.ContainsAny(value, "\\:\x00") {
		return false
	}
	owner, repository, found := strings.Cut(value, "/")
	return found && owner != "" && repository != "" && !strings.Contains(repository, "/") && owner != "." && owner != ".." && repository != "." && repository != ".."
}

func baseResult(action, adapter, status, digest, target string) Result {
	return Result{
		APIVersion: resultAPIVersion, Adapter: adapter, Action: action, Status: status,
		ModelDigest: digest, Target: target, Findings: []Finding{}, Operations: []Operation{},
		Observed: Observation{Evidence: "github-rest-repository-metadata-capture", GithubAPIVersion: githubAPIVersion, Scope: []string{}, Repositories: map[string]repositoryObservation{}},
	}
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

func sameObserved(a, b Observation) bool {
	x, err := yaml.Marshal(a)
	if err != nil {
		return false
	}
	y, err := yaml.Marshal(b)
	return err == nil && bytes.Equal(x, y)
}

func sameResult(a *Result, b Result) bool {
	if a == nil {
		return false
	}
	x, err := yaml.Marshal(a)
	if err != nil {
		return false
	}
	y, err := yaml.Marshal(b)
	return err == nil && bytes.Equal(x, y)
}

func quote(value string) string { return fmt.Sprintf("%q", value) }
