package host

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
)

const (
	goalRecommendationModulePin = "markitect.host/goal-module-recommendation@v1alpha1"
	goalModelingModulePin       = "markitect.host/goal-modeling-proposal@v1alpha1"
	maxGoalBytes                = 8 << 10
	maxGoalPackages             = 64
	maxGoalCatalogBytes         = 4 << 20
)

// GoalModelingInput contains only caller-supplied goal text and already
// available Module package bytes. The API performs no lookup, filesystem
// access, installation, network operation, or provider loading.
type GoalModelingInput struct {
	Goal                   string
	Packages               []canonical.ModulePackage
	DecisionReferenceClaim string
}

// GoalModuleRecommendation identifies an exact supplied package. ID and Pin
// are Host-derived from package bytes; the provider supplies only its basis
// and uncertainty.
type GoalModuleRecommendation struct {
	ID          string        `json:"id"`
	Pin         canonical.Pin `json:"pin"`
	Name        string        `json:"name"`
	Version     string        `json:"version"`
	Type        string        `json:"type"`
	Basis       string        `json:"basis"`
	Uncertainty []string      `json:"uncertainty"`
}

// GoalModuleRecommendationResult is a noncanonical suggestion. It binds the
// exact goal, supplied catalog, and fresh RoleInfer receipt. A caller must
// still supply selected recommendation IDs to ProposeGoalModel.
type GoalModuleRecommendationResult struct {
	Status                string                     `json:"status"`
	GoalDigest            string                     `json:"goalDigest"`
	CatalogDigest         string                     `json:"catalogDigest"`
	RecommendationsDigest string                     `json:"recommendationsDigest,omitempty"`
	Recommendations       []GoalModuleRecommendation `json:"recommendations"`
	Uncertainty           []string                   `json:"uncertainty"`
	Receipt               agentexec.Receipt          `json:"receipt"`
	Adopted               bool                       `json:"adopted"`
	Accepted              bool                       `json:"accepted"`
}

// GoalModelingOptions names only the external runner's private runtime
// directories. They do not identify a project or a repository to inspect.
type GoalModelingOptions struct {
	TempParent          string
	PrivateLogDirectory string
}

// GoalModelingResult is a compiled, noncanonical proposal. Compilation proves
// only that the supplied Definitions satisfy the selected schemas and Core's
// typed reference rules. The result is never adopted or written to disk.
type GoalModelingResult struct {
	Summary                        string            `json:"summary,omitempty"`
	CandidateJSON                  string            `json:"candidateJson,omitempty"`
	Status                         string            `json:"status"`
	GoalDigest                     string            `json:"goalDigest"`
	CatalogDigest                  string            `json:"catalogDigest"`
	RecommendationsDigest          string            `json:"recommendationsDigest"`
	SelectedRecommendationIDs      []string          `json:"selectedRecommendationIds"`
	SelectedPins                   []canonical.Pin   `json:"selectedPins"`
	InputDigest                    string            `json:"inputDigest"`
	CandidateDigest                string            `json:"candidateDigest,omitempty"`
	DecisionReferenceClaim         string            `json:"decisionReferenceClaim,omitempty"`
	DecisionReferenceAuthenticated bool              `json:"decisionReferenceAuthenticated"`
	Receipt                        agentexec.Receipt `json:"receipt"`
	CompiledCandidate              *core.Model       `json:"compiledCandidate,omitempty"`
	Diagnostics                    []core.Diagnostic `json:"diagnostics"`
	Uncertainty                    []string          `json:"uncertainty"`
	Adopted                        bool              `json:"adopted"`
	Accepted                       bool              `json:"accepted"`
}

type goalCatalogEntry struct {
	ID       string                       `json:"id"`
	Pin      canonical.Pin                `json:"pin"`
	Type     string                       `json:"type"`
	Purpose  string                       `json:"purpose"`
	Requires canonical.ModuleRequirements `json:"requires"`
	Schemas  []goalSchemaInput            `json:"schemas"`
	Package  canonical.ModulePackage      `json:"-"`
}

type goalSchemaInput struct {
	Path    string `json:"path"`
	Content []byte `json:"content"`
}

// RecommendGoalModules makes one fresh RoleInfer call over exactly the
// explicit goal and bounded package input. Provider output cannot create pins
// or expand the supplied package set.
func RecommendGoalModules(ctx context.Context, input GoalModelingInput, config agentexec.Config, options GoalModelingOptions) (GoalModuleRecommendationResult, error) {
	goalDigest, catalogDigest, entries, err := prepareGoalCatalog(input)
	if err != nil {
		return GoalModuleRecommendationResult{}, err
	}
	contextValue := struct {
		Phase      string             `json:"phase"`
		Goal       string             `json:"goal"`
		GoalDigest string             `json:"goalDigest"`
		Catalog    []goalCatalogEntry `json:"suppliedModules"`
		Task       string             `json:"task"`
		Output     any                `json:"output"`
	}{
		Phase: "recommend-modules", Goal: input.Goal, GoalDigest: goalDigest,
		Catalog: entries,
		Task:    "Recommend only exact modules from suppliedModules that appear useful for this goal. Give a concise basis and explicit uncertainty for each recommendation. Do not invent module identities, pins, schemas, or project facts.",
		Output:  map[string]any{"recommendations": []any{map[string]any{"id": "supplied module id", "basis": "why this exact supplied module may help", "uncertainty": []string{"what is unknown"}}}, "uncertainty": []string{"overall limitation"}},
	}
	request, err := goalInferenceRequest(contextValue, goalDigest, catalogDigest, goalRecommendationModulePin, "goal/module-recommendations", catalogIDs(entries))
	if err != nil {
		return GoalModuleRecommendationResult{}, err
	}
	configDigest, err := agentexec.Fingerprint(config)
	if err != nil {
		return GoalModuleRecommendationResult{}, fmt.Errorf("fingerprint goal recommendation runner: %w", err)
	}
	run, runErr := agentexec.Run(ctx, config, request, agentexec.RunOptions{TempParent: options.TempParent, PrivateLogDirectory: options.PrivateLogDirectory})
	result := GoalModuleRecommendationResult{
		Status: "incomplete", GoalDigest: goalDigest, CatalogDigest: catalogDigest,
		Recommendations: []GoalModuleRecommendation{}, Uncertainty: []string{},
		Receipt: run.Receipt, Adopted: false, Accepted: false,
	}
	if runErr != nil {
		return result, runErr
	}
	if run.Receipt.ConfigDigest != configDigest {
		return result, errors.New("goal recommendation receipt configuration differs from the prepared runner fingerprint")
	}
	result.Status = run.Response.Outcome
	if run.Response.Outcome != agentexec.OutcomeProposed {
		result.Uncertainty = append([]string(nil), run.Response.Uncertainty...)
		return result, nil
	}
	if !hasGoalUncertainty(run.Response.Uncertainty) {
		return result, errors.New("goal recommendations must include explicit runner uncertainty")
	}
	var wire goalRecommendationOutput
	if err := strictGoalJSON(run.Response.CandidateJSON, &wire); err != nil {
		return result, fmt.Errorf("decode goal recommendations: %w", err)
	}
	if len(wire.Recommendations) > len(entries) || !hasGoalUncertainty(wire.Uncertainty) || !hasGoalUncertainty(run.Response.Uncertainty) {
		return result, errors.New("goal recommendation output exceeds the supplied catalog or omits uncertainty")
	}
	byID := make(map[string]goalCatalogEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	seen := map[string]bool{}
	for _, recommendation := range wire.Recommendations {
		entry, ok := byID[recommendation.ID]
		if !ok || seen[recommendation.ID] {
			return result, fmt.Errorf("goal recommendation names an unknown or repeated supplied Module ID %q", recommendation.ID)
		}
		if strings.TrimSpace(recommendation.Basis) == "" || !hasGoalUncertainty(recommendation.Uncertainty) {
			return result, fmt.Errorf("goal recommendation %q must include a basis and uncertainty", recommendation.ID)
		}
		seen[recommendation.ID] = true
		result.Recommendations = append(result.Recommendations, GoalModuleRecommendation{
			ID: entry.ID, Pin: entry.Pin, Name: entry.Pin.Name, Version: entry.Pin.Version,
			Type: entry.Type, Basis: recommendation.Basis, Uncertainty: append([]string(nil), recommendation.Uncertainty...),
		})
	}
	result.Uncertainty = append([]string(nil), wire.Uncertainty...)
	result.RecommendationsDigest, err = recommendationDigest(result.Recommendations)
	if err != nil {
		return result, err
	}
	result.Status = "proposed"
	return result, nil
}

type goalRecommendationOutput struct {
	Recommendations []struct {
		ID          string   `json:"id"`
		Basis       string   `json:"basis"`
		Uncertainty []string `json:"uncertainty"`
	} `json:"recommendations"`
	Uncertainty []string `json:"uncertainty"`
}

// ProposeGoalModel requires explicit selected IDs from a prior recommendation
// result, activates only their exact supplied pins, and asks a fresh RoleInfer
// process for Definitions only. Candidate YAML is validated with the canonical
// decoder and the same Core compiler used for canonical source.
func ProposeGoalModel(ctx context.Context, input GoalModelingInput, prior GoalModuleRecommendationResult, selectedIDs []string, config agentexec.Config, options GoalModelingOptions) (GoalModelingResult, error) {
	goalDigest, catalogDigest, entries, err := prepareGoalCatalog(input)
	if err != nil {
		return GoalModelingResult{}, err
	}
	result := GoalModelingResult{
		Status: "incomplete", GoalDigest: goalDigest, CatalogDigest: catalogDigest,
		RecommendationsDigest:     prior.RecommendationsDigest,
		SelectedRecommendationIDs: []string{}, SelectedPins: []canonical.Pin{}, Diagnostics: []core.Diagnostic{},
		DecisionReferenceClaim: input.DecisionReferenceClaim,
		Uncertainty:            []string{}, Adopted: false, Accepted: false,
	}
	if prior.Status != "proposed" || prior.GoalDigest != goalDigest || prior.CatalogDigest != catalogDigest || prior.Adopted || prior.Accepted {
		return result, errors.New("goal modeling requires an unchanged, noncanonical recommendation result for this exact goal and supplied catalog")
	}
	wantRecommendationDigest, err := recommendationDigest(prior.Recommendations)
	if err != nil || wantRecommendationDigest == "" || wantRecommendationDigest != prior.RecommendationsDigest {
		return result, errors.New("goal recommendation result binding is invalid")
	}
	if len(selectedIDs) == 0 || len(selectedIDs) > maxGoalPackages {
		return result, fmt.Errorf("explicitly selected recommendation IDs must contain 1 to %d entries", maxGoalPackages)
	}
	byID := make(map[string]goalCatalogEntry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	recommendations := make(map[string]GoalModuleRecommendation, len(prior.Recommendations))
	for _, recommendation := range prior.Recommendations {
		recommendations[recommendation.ID] = recommendation
	}
	selectedIDs = append([]string(nil), selectedIDs...)
	sort.Strings(selectedIDs)
	selectedPins := make([]canonical.Pin, 0, len(selectedIDs))
	selectedEntries := make([]goalCatalogEntry, 0, len(selectedIDs))
	for i, id := range selectedIDs {
		if id == "" || i > 0 && selectedIDs[i-1] == id {
			return result, errors.New("selected recommendation IDs must be nonempty and unique")
		}
		recommendation, recommended := recommendations[id]
		entry, supplied := byID[id]
		if !recommended || !supplied || recommendation.Pin != entry.Pin {
			return result, fmt.Errorf("selected ID %q is not an exact supplied recommendation", id)
		}
		selectedPins = append(selectedPins, entry.Pin)
		selectedEntries = append(selectedEntries, entry)
	}
	activation, err := canonical.Resolve(packagesOf(entries), selectedPins)
	if err != nil {
		return result, fmt.Errorf("activate selected Module pins: %w", err)
	}
	result.SelectedRecommendationIDs = append([]string(nil), selectedIDs...)
	result.SelectedPins = append([]canonical.Pin(nil), activation.Modules...)
	schemaDigest, err := digestJSON(activation.Schemas)
	if err != nil {
		return result, err
	}
	contextValue := struct {
		Phase        string             `json:"phase"`
		Goal         string             `json:"goal"`
		GoalDigest   string             `json:"goalDigest"`
		Modules      []goalCatalogEntry `json:"selectedModules"`
		Schemas      []core.Schema      `json:"selectedCoreSchemas"`
		SchemaDigest string             `json:"schemaDigest"`
		Task         string             `json:"task"`
		Output       any                `json:"output"`
	}{
		Phase: "model-from-selected-modules", Goal: input.Goal, GoalDigest: goalDigest,
		Modules: selectedEntries, Schemas: activation.Schemas, SchemaDigest: schemaDigest,
		Task:   "Propose canonical Definition values that express the user's goal using only the supplied selectedCoreSchemas. Return no Schema changes, module changes, code, or adoption decision. State uncertainty explicitly. Values must satisfy every closed Property contract; references must use exact declared target kinds.",
		Output: map[string]any{"definitions": []any{map[string]any{"apiVersion": "schema-declared value", "kind": "schema-declared Kind", "metadata": map[string]any{"name": "stable identifier", "namespace": "owner namespace"}, "purpose": "human-readable intent", "spec": map[string]any{"schema-declared-property": "schema-typed value"}}}, "summary": "brief proposal description", "uncertainty": []string{"unresolved modeling question"}},
	}
	inputBinding, err := digestJSON(struct {
		GoalDigest, CatalogDigest, RecommendationsDigest, SchemaDigest, DecisionReferenceClaim string
		SelectedIDs                                                                            []string
	}{goalDigest, catalogDigest, prior.RecommendationsDigest, schemaDigest, input.DecisionReferenceClaim, selectedIDs})
	if err != nil {
		return result, err
	}
	request, err := goalInferenceRequest(contextValue, goalDigest, inputBinding, goalModelingModulePin, "goal/model-proposal", selectedIDs)
	if err != nil {
		return result, err
	}
	configDigest, err := agentexec.Fingerprint(config)
	if err != nil {
		return result, fmt.Errorf("fingerprint goal modeling runner: %w", err)
	}
	run, runErr := agentexec.Run(ctx, config, request, agentexec.RunOptions{TempParent: options.TempParent, PrivateLogDirectory: options.PrivateLogDirectory})
	result.InputDigest = run.Receipt.InputDigest
	result.Receipt = run.Receipt
	if runErr != nil {
		return result, runErr
	}
	if run.Receipt.ConfigDigest != configDigest {
		return result, errors.New("goal modeling receipt configuration differs from the prepared runner fingerprint")
	}
	result.Status = run.Response.Outcome
	if run.Response.Outcome != agentexec.OutcomeProposed {
		result.Uncertainty = append([]string(nil), run.Response.Uncertainty...)
		return result, nil
	}
	if !hasGoalUncertainty(run.Response.Uncertainty) {
		return result, errors.New("goal modeling proposal must include explicit runner uncertainty")
	}
	definitions, summary, candidateDigest, uncertainties, err := decodeGoalDefinitions(run.Response.CandidateJSON)
	if err != nil {
		return result, err
	}
	if !hasGoalUncertainty(uncertainties) {
		return result, errors.New("goal modeling candidate must include explicit proposal uncertainty")
	}
	result.CandidateDigest = candidateDigest
	result.Summary = summary
	result.CandidateJSON = string(append([]byte(nil), run.Response.CandidateJSON...))
	result.Uncertainty = append(append([]string(nil), run.Response.Uncertainty...), uncertainties...)
	model, diagnostics := core.Compile(activation.Schemas, definitions, "goal/"+strings.TrimPrefix(goalDigest, "sha256:"))
	result.Diagnostics = append([]core.Diagnostic(nil), diagnostics...)
	if len(diagnostics) != 0 {
		result.Status = "invalid"
		return result, fmt.Errorf("compile goal modeling candidate: %d Core diagnostic(s)", len(diagnostics))
	}
	result.CompiledCandidate = &model
	result.Status = "proposed"
	return result, nil
}

type goalModelOutput struct {
	Definitions []json.RawMessage `json:"definitions"`
	Summary     string            `json:"summary"`
	Uncertainty []string          `json:"uncertainty"`
}

func decodeGoalDefinitions(data []byte) ([]core.Definition, string, string, []string, error) {
	var output goalModelOutput
	if err := strictGoalJSON(data, &output); err != nil {
		return nil, "", "", nil, fmt.Errorf("decode goal modeling candidate: %w", err)
	}
	if len(output.Definitions) == 0 || len(output.Definitions) > core.MaxDefinitions || strings.TrimSpace(output.Summary) == "" {
		return nil, "", "", nil, errors.New("goal modeling output must include Definitions and a summary")
	}
	definitions := make([]core.Definition, 0, len(output.Definitions))
	for i, raw := range output.Definitions {
		path := fmt.Sprintf("goal-proposal/definition-%06d.yaml", i+1)
		definition, err := canonical.DecodeDefinition(path, raw)
		if err != nil {
			return nil, "", "", nil, fmt.Errorf("decode proposed Definition %d: %w", i+1, err)
		}
		definitions = append(definitions, definition)
	}
	digest := fileDigestBytes(data)
	return definitions, output.Summary, digest, append([]string(nil), output.Uncertainty...), nil
}

func hasGoalUncertainty(values []string) bool {
	if len(values) == 0 {
		return false
	}
	for _, value := range values {
		if strings.TrimSpace(value) == "" {
			return false
		}
	}
	return true
}

func strictGoalJSON(data []byte, dst any) error {
	if len(data) == 0 || len(data) > 8<<20 {
		return errors.New("JSON is empty or exceeds the 8 MiB goal-modeling bound")
	}
	if err := rejectDuplicateJSONKeys(data); err != nil {
		return err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errors.New("JSON contains trailing data")
	}
	return nil
}

func prepareGoalCatalog(input GoalModelingInput) (string, string, []goalCatalogEntry, error) {
	if len(input.Goal) == 0 || len(input.Goal) > maxGoalBytes || !utf8.ValidString(input.Goal) || strings.TrimSpace(input.Goal) == "" {
		return "", "", nil, fmt.Errorf("goal must be valid nonempty UTF-8 text of at most %d bytes", maxGoalBytes)
	}
	if len(input.Packages) == 0 || len(input.Packages) > maxGoalPackages {
		return "", "", nil, fmt.Errorf("supplied Module package count must be between 1 and %d", maxGoalPackages)
	}
	if len(input.DecisionReferenceClaim) > 4096 || !utf8.ValidString(input.DecisionReferenceClaim) {
		return "", "", nil, errors.New("decision reference claim must be valid UTF-8 text of at most 4096 bytes")
	}
	goalDigest := fileDigestBytes([]byte(input.Goal))
	entries := make([]goalCatalogEntry, 0, len(input.Packages))
	seen := map[string]bool{}
	totalBytes := len(input.Goal)
	for _, pkg := range input.Packages {
		manifest, err := canonical.DecodeManifest(pkg.ManifestBytes)
		if err != nil {
			return "", "", nil, fmt.Errorf("decode supplied Module manifest: %w", err)
		}
		digest, err := canonical.DigestPackage(pkg)
		if err != nil {
			return "", "", nil, fmt.Errorf("digest supplied Module %s@%s: %w", manifest.Name, manifest.Version, err)
		}
		pin := canonical.Pin{Name: manifest.Name, Version: manifest.Version, Digest: digest}
		id := goalModuleID(pin)
		if seen[id] {
			return "", "", nil, fmt.Errorf("supplied Module %q is duplicated", id)
		}
		seen[id] = true
		totalBytes += len(pkg.ManifestBytes)
		entry := goalCatalogEntry{ID: id, Pin: pin, Type: manifest.Type, Purpose: manifest.Purpose, Requires: manifest.Requires, Schemas: []goalSchemaInput{}, Package: pkg}
		for _, path := range manifest.Provides.Schemas {
			data, ok := pkg.Files[path]
			if !ok {
				return "", "", nil, fmt.Errorf("supplied Module %s schema %q is missing", id, path)
			}
			totalBytes += len(data)
			entry.Schemas = append(entry.Schemas, goalSchemaInput{Path: path, Content: append([]byte(nil), data...)})
		}
		if totalBytes > maxGoalCatalogBytes {
			return "", "", nil, fmt.Errorf("goal and supplied Module manifest/schema bytes exceed %d bytes", maxGoalCatalogBytes)
		}
		entries = append(entries, entry)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	bindings := make([]struct {
		ID   string        `json:"id"`
		Pin  canonical.Pin `json:"pin"`
		Hash string        `json:"packageDigest"`
	}, len(entries))
	for i, entry := range entries {
		bindings[i] = struct {
			ID   string        `json:"id"`
			Pin  canonical.Pin `json:"pin"`
			Hash string        `json:"packageDigest"`
		}{entry.ID, entry.Pin, entry.Pin.Digest}
	}
	catalogDigest, err := digestJSON(bindings)
	return goalDigest, catalogDigest, entries, err
}

func goalInferenceRequest(contextValue any, goalDigest, inputDigest, modulePin, projectionID string, scopes []string) (agentexec.Request, error) {
	contextBytes, err := json.Marshal(contextValue)
	if err != nil {
		return agentexec.Request{}, fmt.Errorf("encode bounded goal-modeling context: %w", err)
	}
	return agentexec.Request{
		Role: agentexec.RoleInfer, SourceRevision: "goal/" + strings.TrimPrefix(goalDigest, "sha256:"),
		ModelDigest: inputDigest, ModulePin: modulePin, ProjectionID: projectionID,
		ScopeIDs: scopes, PolicyIDs: []string{}, Context: contextBytes, Artifacts: []agentexec.Artifact{},
	}, nil
}

func catalogIDs(entries []goalCatalogEntry) []string {
	ids := make([]string, 0, len(entries))
	for _, entry := range entries {
		ids = append(ids, entry.ID)
	}
	return ids
}

func packagesOf(entries []goalCatalogEntry) []canonical.ModulePackage {
	packages := make([]canonical.ModulePackage, 0, len(entries))
	for _, entry := range entries {
		packages = append(packages, entry.Package)
	}
	return packages
}

func goalModuleID(pin canonical.Pin) string {
	return pin.Name + "@" + pin.Version + "#" + pin.Digest
}

func recommendationDigest(values []GoalModuleRecommendation) (string, error) {
	copy := append([]GoalModuleRecommendation(nil), values...)
	sort.Slice(copy, func(i, j int) bool { return copy[i].ID < copy[j].ID })
	return digestJSON(copy)
}

func digestJSON(value any) (string, error) {
	return digestCanonicalValue(value)
}
