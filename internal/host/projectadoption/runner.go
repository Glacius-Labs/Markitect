package projectadoption

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

const (
	distillationRunnerVersion = "markitect.example.org/project-distillation-runner/v1alpha1"
	maxDistillationTimeout    = 10 * time.Minute
	maxDistillationStdout     = 4 << 20
	maxDistillationStderr     = 1 << 20
	maxDistillationArtifacts  = 128
	maxDistillationCostMicros = int64(1_000_000_000_000)
	microsPerMillionTokens    = int64(1_000_000)
)

// DistillationRunOptions bounds one agent call. Prices are caller-supplied
// microcurrency per million provider-reported tokens; they estimate accepted
// output cost and cannot cap a provider's pre-receipt invoice.
type DistillationRunOptions struct {
	MaxTimeout                  time.Duration             `json:"maxTimeout"`
	MaxStdoutBytes              int                       `json:"maxStdoutBytes"`
	MaxStderrBytes              int                       `json:"maxStderrBytes"`
	MaxCostMicros               int64                     `json:"maxCostMicros"`
	InputPriceMicrosPerMillion  int64                     `json:"inputPriceMicrosPerMillion"`
	OutputPriceMicrosPerMillion int64                     `json:"outputPriceMicrosPerMillion"`
	TempParent                  string                    `json:"tempParent,omitempty"`
	PrivateLogDirectory         string                    `json:"privateLogDirectory,omitempty"`
	TargetContext               DistillationTargetContext `json:"targetContext"`
}

// DistillationDraft contains only model-proposed content. Binding metadata is
// supplied by the Host after a successful agentexec invocation. Provider DTO
// fields are required even when empty so the closed Python adapter schema can
// require every property. RuntimeObservationJSON is empty when a claim makes
// no submitted-runtime-record assertion.
type DistillationDraft struct {
	Claims         []DistillationDraftClaim    `json:"claims"`
	Terms          []Term                      `json:"terms"`
	Contradictions []Contradiction             `json:"contradictions"`
	Questions      []DistillationDraftQuestion `json:"questions"`
	Scopes         []DistillationDraftScope    `json:"scopes"`
	Proposal       ModelProposal               `json:"proposal"`
}

type DistillationDraftClaim struct {
	ID                     string        `json:"id"`
	ScopeID                string        `json:"scopeId"`
	Kind                   string        `json:"kind"`
	Method                 string        `json:"method"`
	Statement              string        `json:"statement"`
	Evidence               []EvidenceRef `json:"evidence"`
	Uncertainty            []string      `json:"uncertainty"`
	RuntimeObservationJSON string        `json:"runtimeObservationJson"`
}

type DistillationDraftQuestion struct {
	ID           string   `json:"id"`
	ScopeID      string   `json:"scopeId"`
	Prompt       string   `json:"prompt"`
	Alternatives []string `json:"alternatives"`
	ClaimIDs     []string `json:"claimIds"`
	Blocking     bool     `json:"blocking"`
}

type DistillationDraftScope struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	ParentID       string   `json:"parentId"`
	ClaimIDs       []string `json:"claimIds"`
	OwnerCandidate string   `json:"ownerCandidate"`
}

// DistillationReceipt preserves the real runner receipt and explicit cost
// estimate used to accept this generated proposal.
type DistillationReceipt struct {
	APIVersion                  string            `json:"apiVersion"`
	DiscoveryDigest             string            `json:"discoveryDigest"`
	TargetContextDigest         string            `json:"targetContextDigest"`
	SchemaDigest                string            `json:"schemaDigest"`
	DistillationDigest          string            `json:"distillationDigest,omitempty"`
	RunnerIdentity              string            `json:"runnerIdentity"`
	RunnerDigest                string            `json:"runnerDigest"`
	Execution                   agentexec.Receipt `json:"execution"`
	ExecutionReceiptDigest      string            `json:"executionReceiptDigest"`
	MaxCostMicros               int64             `json:"maxCostMicros"`
	InputPriceMicrosPerMillion  int64             `json:"inputPriceMicrosPerMillion"`
	OutputPriceMicrosPerMillion int64             `json:"outputPriceMicrosPerMillion"`
	EstimatedCostMicros         int64             `json:"estimatedCostMicros"`
}

type distillationRequestContext struct {
	Instructions    string                    `json:"instructions"`
	DiscoveryDigest string                    `json:"discoveryDigest"`
	Purpose         string                    `json:"purpose"`
	Review          string                    `json:"review"`
	ScopeRoots      []string                  `json:"scopeRoots"`
	Selected        []SelectedPath            `json:"selected"`
	Exclusions      []PathReason              `json:"exclusions"`
	Unselected      []PathReason              `json:"unselected"`
	Schema          any                       `json:"schema"`
	ResponseSchema  json.RawMessage           `json:"responseSchema"`
	TargetContext   DistillationTargetContext `json:"targetContext"`
}

// GenerateDistillation invokes one configured Host executor against the exact
// selected evidence already frozen in Discovery. It returns a proposal and
// receipt only; it does not accept scopes, edit files, or invoke projectwork.
func GenerateDistillation(ctx context.Context, sourceRoot string, discovery Discovery, config agentexec.Config, options DistillationRunOptions) (Distillation, DistillationReceipt, error) {
	var empty Distillation
	var receipt DistillationReceipt
	if ctx == nil {
		return empty, receipt, errors.New("distillation context is required")
	}
	if err := ValidateDiscovery(discovery); err != nil {
		return empty, receipt, fmt.Errorf("validate fixed discovery: %w", err)
	}
	if err := validateDistillationRunnerOptions(sourceRoot, config, options); err != nil {
		return empty, receipt, err
	}
	if err := ValidateTargetContext(options.TargetContext); err != nil {
		return empty, receipt, fmt.Errorf("validate accepted target context: %w", err)
	}
	if len(discovery.Evidence) > maxDistillationArtifacts {
		return empty, receipt, fmt.Errorf("distillation supports at most %d selected evidence files per invocation", maxDistillationArtifacts)
	}
	if _, err := RefreshDiscovery(sourceRoot, discovery); err != nil {
		return empty, receipt, fmt.Errorf("refresh fixed discovery before invocation: %w", err)
	}
	schemaDigest, _, err := CurrentBindings(projectmodel.Schema())
	if err != nil {
		return empty, receipt, fmt.Errorf("bind active project schema and build: %w", err)
	}
	artifacts := make([]agentexec.Artifact, 0, len(discovery.Evidence))
	for _, evidence := range discovery.Evidence {
		artifactPath := "evidence/" + evidence.ID + ".txt"
		mode := "0644"
		if evidence.Mode == "100755" {
			mode = "0755"
		}
		artifacts = append(artifacts, agentexec.Artifact{
			Path: artifactPath, Mode: mode, Digest: "sha256:" + evidence.Digest,
			Content: []byte(evidence.Content),
		})
	}
	contextData := distillationRequestContext{
		Instructions: `Analyze only the supplied fixed Discovery evidence artifacts and project-model schema. targetContext is accepted target guidance, not source evidence; use it only for compatible placement under existing Manager identities/namespaces and to avoid conflicts with public contracts. Produce a proposal only; do not adopt it or modify source. Distinguish static source observations, documented intent, submitted runtime records, and synthesis hypotheses. Never infer behavior from filenames. Do not present runtime records as authenticated execution or claim human acceptance.\n\n` +
			`IDs must match ^[a-z0-9](?:[a-z0-9-]{0,62}[a-z0-9])?$: 1-64 lowercase ASCII letters/digits with only internal hyphens. IDs are local identifiers; use simple scope IDs such as orders, never namespaces such as commerce.sales.orders. Every claim, question, and contradiction scopeId must name a declared local scope. Each parentId must name another declared local scope; each claimIds entry must name a declared claim assigned to that same scope. Every scope claimIds list must include its assigned claims. Proposal file scopeId must name a declared scope. Every claim and term occurrence must cite a selected evidence ID copied exactly from Discovery; each excerpt must be exact contiguous text with inclusive one-based line bounds. Terms must cite occurrences whose excerpt contains the exact term. Questions must cite same-scope claims and provide at least two distinct alternatives. Contradictions must point to a same-scope question whose claimIds include every conflicting claim.\n\n` +
			`Claim kind/method pairs are exactly: observation/static-source (runtimeObservationJson is the empty string); documented-intent/documentation (runtimeObservationJson is the empty string); submitted-runtime-record/submitted-record (runtimeObservationJson is required strict JSON copied from selected runtime-record evidence and the claim must cite that record); hypothesis/synthesis (runtimeObservationJson is the empty string). Do not use kind runtime-observation or combine other pairs. For a submitted record, runtimeObservationJson must contain exactly evidenceId, recordSourceRevision, sourceRelation, command, exitCode, runnerDigest, and inputs; copy record fields exactly and set sourceRelation to same-discovery-commit only when its revision equals the Discovery commit, otherwise historical. Never fabricate runtime metadata for other claim kinds. Return every required property including empty arrays and empty optional-string values, only the closed reportJson object described by responseSchema. Model proposal files must use declared local scope IDs and canonical .markitect/model YAML paths.`,
		DiscoveryDigest: discovery.Digest, Purpose: discovery.Purpose,
		Review: discovery.Review, ScopeRoots: append([]string{}, discovery.ScopeRoots...),
		Selected:   append([]SelectedPath{}, discovery.Selected...),
		Exclusions: append([]PathReason{}, discovery.Exclusions...),
		Unselected: append([]PathReason{}, discovery.Unselected...),
		Schema:     projectmodel.Schema(), ResponseSchema: distillationDraftJSONSchema(),
		TargetContext: options.TargetContext,
	}
	contextJSON, err := json.Marshal(contextData)
	if err != nil {
		return empty, receipt, fmt.Errorf("encode distillation request context: %w", err)
	}
	request := agentexec.Request{
		Role: agentexec.RoleExecutor, SourceRevision: discovery.Commit,
		ModelDigest: "sha256:" + discovery.Digest, ModulePin: "project-adoption/" + DistillationVersion,
		ProjectionID: "brownfield-distillation", ScopeIDs: []string{}, PolicyIDs: []string{"proposal-only"},
		Context: contextJSON, Artifacts: artifacts,
	}
	tempParent, logDirectory := options.TempParent, options.PrivateLogDirectory
	if tempParent == "" {
		tempParent = os.TempDir()
	}
	if logDirectory == "" {
		logDirectory, err = newPrivateLogDirectory(os.TempDir())
		if err != nil {
			return empty, receipt, fmt.Errorf("prepare private agent log location: %w", err)
		}
	}
	result, runErr := agentexec.Run(ctx, config, request, agentexec.RunOptions{
		InputRoots: []string{}, TempParent: tempParent,
		PrivateLogDirectory: logDirectory,
	})
	receipt = bindDistillationReceipt(discovery, schemaDigest, config, result.Receipt, options)
	if runErr != nil {
		return empty, receipt, fmt.Errorf("agent-assisted distillation invocation failed: %w", runErr)
	}
	if result.Response.Outcome != agentexec.OutcomeProposed || len(result.Response.ReportJSON) == 0 || len(result.Response.CandidateFiles) != 0 || len(result.Response.CandidateJSON) != 0 {
		return empty, receipt, errors.New("executor did not return a report-only distillation proposal")
	}
	if err := requireDistillationUsage(result.Response.Usage); err != nil {
		return empty, receipt, err
	}
	inputCost, err := tokenCostMicros(*result.Response.Usage.InputTokens, options.InputPriceMicrosPerMillion)
	if err != nil {
		return empty, receipt, err
	}
	outputCost, err := tokenCostMicros(*result.Response.Usage.OutputTokens, options.OutputPriceMicrosPerMillion)
	if err != nil || inputCost > math.MaxInt64-outputCost {
		return empty, receipt, errors.New("provider-reported token cost exceeds the supported bound")
	}
	receipt.EstimatedCostMicros = inputCost + outputCost
	if receipt.EstimatedCostMicros > options.MaxCostMicros {
		return empty, receipt, fmt.Errorf("provider-reported cost estimate %d micros exceeds configured acceptance ceiling %d micros", receipt.EstimatedCostMicros, options.MaxCostMicros)
	}
	if _, err := RefreshDiscovery(sourceRoot, discovery); err != nil {
		return empty, receipt, fmt.Errorf("refresh fixed discovery after invocation: %w", err)
	}
	var draft DistillationDraft
	if err := decodeClosedJSON(result.Response.ReportJSON, &draft); err != nil {
		return empty, receipt, fmt.Errorf("decode executor reportJson as a closed distillation draft: %w", err)
	}
	report := Distillation{
		APIVersion: DistillationVersion, DiscoveryDigest: discovery.Digest,
		TargetBasis: options.TargetContext.ProjectDigest, TargetRevision: options.TargetContext.Revision,
		TargetContextDigest: options.TargetContext.Digest,
		Method:              "agent-assisted", RunnerIdentity: distillationRunnerIdentity(config),
		RunnerDigest: digestWithoutPrefix(result.Receipt.ConfigDigest), SchemaDigest: schemaDigest,
		Claims: []Claim{}, Terms: draft.Terms, Contradictions: draft.Contradictions,
		Questions: []Question{}, Scopes: []ScopeProposal{}, Proposal: draft.Proposal,
	}
	for _, claim := range draft.Claims {
		var runtimeObservation *RuntimeObservation
		if claim.RuntimeObservationJSON != "" {
			var parsed RuntimeObservation
			if err := decodeClosedJSON([]byte(claim.RuntimeObservationJSON), &parsed); err != nil {
				return empty, receipt, fmt.Errorf("decode claim %q submitted runtime record: %w", claim.ID, err)
			}
			runtimeObservation = &parsed
		}
		report.Claims = append(report.Claims, Claim{
			ID: claim.ID, ScopeID: claim.ScopeID, Kind: claim.Kind, Method: claim.Method,
			Statement: claim.Statement, Evidence: claim.Evidence, Uncertainty: claim.Uncertainty,
			Runtime: runtimeObservation,
		})
	}
	for _, question := range draft.Questions {
		blocking := question.Blocking
		report.Questions = append(report.Questions, Question{
			ID: question.ID, ScopeID: question.ScopeID, Prompt: question.Prompt,
			Alternatives: question.Alternatives, ClaimIDs: question.ClaimIDs, Blocking: &blocking,
		})
	}
	for _, scope := range draft.Scopes {
		report.Scopes = append(report.Scopes, ScopeProposal{
			ID: scope.ID, Name: scope.Name, ParentID: scope.ParentID,
			ClaimIDs: scope.ClaimIDs, OwnerCandidate: scope.OwnerCandidate,
		})
	}
	SealDistillation(&report)
	if err := ValidateDistillation(discovery, report); err != nil {
		return empty, receipt, fmt.Errorf("validate generated distillation: %w", err)
	}
	receipt.DistillationDigest = report.Digest
	return report, receipt, nil
}

func validateDistillationRunnerOptions(sourceRoot string, config agentexec.Config, options DistillationRunOptions) error {
	if sourceRoot == "" {
		return errors.New("source repository root is required")
	}
	if config.EnvironmentAllowlist == nil {
		return errors.New("distillation runner requires an explicit environment allowlist")
	}
	if config.Timeout <= 0 || options.MaxTimeout <= 0 || options.MaxTimeout > maxDistillationTimeout || config.Timeout > options.MaxTimeout {
		return fmt.Errorf("runner timeout must be positive and no greater than the explicit maximum of %s", maxDistillationTimeout)
	}
	if options.MaxStdoutBytes <= 0 || options.MaxStdoutBytes > maxDistillationStdout || config.MaxStdoutBytes <= 0 || config.MaxStdoutBytes > options.MaxStdoutBytes {
		return fmt.Errorf("executor stdout limit must be positive and no greater than %d bytes or its explicit option bound", maxDistillationStdout)
	}
	if options.MaxStderrBytes <= 0 || options.MaxStderrBytes > maxDistillationStderr || config.MaxStderrBytes <= 0 || config.MaxStderrBytes > options.MaxStderrBytes {
		return fmt.Errorf("executor stderr limit must be positive and no greater than %d bytes or its explicit option bound", maxDistillationStderr)
	}
	if options.MaxCostMicros <= 0 || options.MaxCostMicros > maxDistillationCostMicros || options.InputPriceMicrosPerMillion <= 0 || options.OutputPriceMicrosPerMillion <= 0 || options.InputPriceMicrosPerMillion > maxDistillationCostMicros || options.OutputPriceMicrosPerMillion > maxDistillationCostMicros {
		return errors.New("distillation requires finite positive cost ceiling and input/output token prices")
	}
	root, err := filepath.Abs(sourceRoot)
	if err != nil {
		return fmt.Errorf("resolve source repository root: %w", err)
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("resolve source repository identity: %w", err)
	}
	tempParent := options.TempParent
	if tempParent == "" {
		tempParent = os.TempDir()
	}
	if err := requireOutsideRepository(root, tempParent); err != nil {
		return fmt.Errorf("temporary parent: %w", err)
	}
	logDirectory := options.PrivateLogDirectory
	if logDirectory == "" {
		logDirectory = os.TempDir()
	}
	if err := requireOutsideRepository(root, logDirectory); err != nil {
		return fmt.Errorf("private log directory: %w", err)
	}
	return nil
}

func requireOutsideRepository(root, candidate string) error {
	if candidate == "" {
		return errors.New("directory is required")
	}
	absolute, err := filepath.Abs(candidate)
	if err != nil {
		return errors.New("directory path is invalid")
	}
	absolute, err = resolveExistingPrefix(absolute)
	if err != nil {
		return errors.New("directory path could not be resolved")
	}
	relative, err := filepath.Rel(root, absolute)
	if err != nil || relative == "." || relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return errors.New("directory must be outside the source repository")
	}
	return nil
}

func newPrivateLogDirectory(parent string) (string, error) {
	absolute, err := filepath.Abs(parent)
	if err != nil {
		return "", errors.New("private log parent path is invalid")
	}
	info, err := os.Stat(absolute)
	if err != nil || !info.IsDir() {
		return "", errors.New("private log parent must be an existing directory")
	}
	for attempt := 0; attempt < 8; attempt++ {
		var token [16]byte
		if _, err := rand.Read(token[:]); err != nil {
			return "", errors.New("could not allocate a unique private log path")
		}
		candidate := filepath.Join(absolute, "markitect-distillation-"+hex.EncodeToString(token[:]))
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", errors.New("private log path could not be inspected")
		}
	}
	return "", errors.New("could not allocate an unused private log path")
}

func resolveExistingPrefix(absolute string) (string, error) {
	absolute = filepath.Clean(absolute)
	for current := absolute; ; current = filepath.Dir(current) {
		_, err := os.Lstat(current)
		if err == nil {
			resolved, err := filepath.EvalSymlinks(current)
			if err != nil {
				return "", err
			}
			relative, err := filepath.Rel(current, absolute)
			if err != nil {
				return "", err
			}
			return filepath.Clean(filepath.Join(resolved, relative)), nil
		}
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", os.ErrNotExist
		}
	}
}

func requireDistillationUsage(usage *agentexec.Usage) error {
	if usage == nil || usage.Source != "provider-reported" || usage.InputTokens == nil || usage.OutputTokens == nil {
		return errors.New("distillation result is incomplete: provider-reported input and output token usage are required")
	}
	if usage.ToolCalls != nil && *usage.ToolCalls != 0 {
		return errors.New("distillation result is incomplete: tool-call cost is outside the configured token pricing")
	}
	return nil
}

func tokenCostMicros(tokens, pricePerMillion int64) (int64, error) {
	if tokens < 0 || pricePerMillion <= 0 || pricePerMillion > maxDistillationCostMicros {
		return 0, errors.New("provider-reported usage or token price is outside the supported bound")
	}
	whole := tokens / microsPerMillionTokens
	remainder := tokens % microsPerMillionTokens
	if whole > math.MaxInt64/pricePerMillion {
		return 0, errors.New("provider-reported token cost exceeds the supported bound")
	}
	cost := whole * pricePerMillion
	partialProduct := remainder * pricePerMillion
	partial := partialProduct / microsPerMillionTokens
	if partialProduct%microsPerMillionTokens != 0 {
		partial++
	}
	if cost > math.MaxInt64-partial {
		return 0, errors.New("provider-reported token cost exceeds the supported bound")
	}
	return cost + partial, nil
}

func distillationRunnerIdentity(config agentexec.Config) string {
	return "agentexec/" + config.ProviderVersion + "/" + config.Model
}

func bindDistillationReceipt(discovery Discovery, schemaDigest string, config agentexec.Config, execution agentexec.Receipt, options DistillationRunOptions) DistillationReceipt {
	return DistillationReceipt{
		APIVersion: distillationRunnerVersion, DiscoveryDigest: discovery.Digest,
		TargetContextDigest: options.TargetContext.Digest,
		SchemaDigest:        schemaDigest, RunnerIdentity: distillationRunnerIdentity(config),
		RunnerDigest: digestWithoutPrefix(execution.ConfigDigest), Execution: execution,
		ExecutionReceiptDigest: digestValue(execution), MaxCostMicros: options.MaxCostMicros,
		InputPriceMicrosPerMillion:  options.InputPriceMicrosPerMillion,
		OutputPriceMicrosPerMillion: options.OutputPriceMicrosPerMillion,
	}
}

func digestWithoutPrefix(value string) string { return strings.TrimPrefix(value, "sha256:") }

func distillationDraftJSONSchema() json.RawMessage {
	data, err := json.Marshal(jsonSchemaForType(reflect.TypeOf(DistillationDraft{})))
	if err != nil {
		panic(err)
	}
	return data
}

func jsonSchemaForType(value reflect.Type) map[string]any {
	if value.Kind() == reflect.Pointer {
		return jsonSchemaForType(value.Elem())
	}
	switch value.Kind() {
	case reflect.Struct:
		properties := map[string]any{}
		required := []string{}
		for i := 0; i < value.NumField(); i++ {
			field := value.Field(i)
			if field.PkgPath != "" {
				continue
			}
			parts := strings.Split(field.Tag.Get("json"), ",")
			name := parts[0]
			if name == "" || name == "-" {
				continue
			}
			propertySchema := jsonSchemaForType(field.Type)
			switch name {
			case "id", "scopeId", "questionId", "evidenceId":
				propertySchema["minLength"] = 1
				propertySchema["maxLength"] = 64
			case "parentId":
				propertySchema["minLength"] = 0
				propertySchema["maxLength"] = 64
			case "kind":
				propertySchema["enum"] = []string{"observation", "documented-intent", "submitted-runtime-record", "hypothesis"}
			case "method":
				propertySchema["enum"] = []string{"static-source", "documentation", "submitted-record", "synthesis"}
			case "claimIds":
				if items, ok := propertySchema["items"].(map[string]any); ok {
					items["minLength"] = 1
					items["maxLength"] = 64
				}
			}
			properties[name] = propertySchema
			optional := false
			for _, option := range parts[1:] {
				if option == "omitempty" {
					optional = true
				}
			}
			if !optional {
				required = append(required, name)
			}
		}
		return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
	case reflect.Slice, reflect.Array:
		return map[string]any{"type": "array", "items": jsonSchemaForType(value.Elem())}
	case reflect.String:
		return map[string]any{"type": "string"}
	case reflect.Bool:
		return map[string]any{"type": "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return map[string]any{"type": "integer"}
	default:
		return map[string]any{}
	}
}
