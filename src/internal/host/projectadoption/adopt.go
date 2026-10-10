package projectadoption

import (
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"go.yaml.in/yaml/v3"
)

// PlanAdoption turns only named, resolved scopes into canonical project model
// writes. It validates against the selected target project through the frozen
// projectwork edit planner; it never changes implementation or documentation.
func PlanAdoption(sourceRoot string, target *projectwork.Project, discovery Discovery, report Distillation, resolution Resolution, schemaDigest, buildDigest string) (AdoptionPlan, error) {
	if target == nil || target.Snapshot == nil || target.Provisional {
		return AdoptionPlan{}, errors.New("adoption requires a loaded committed target project snapshot")
	}
	activeSchemaDigest, activeBuildDigest, err := CurrentBindings(projectmodel.Schema())
	if err != nil {
		return AdoptionPlan{}, fmt.Errorf("derive trusted active project bindings: %w", err)
	}
	if schemaDigest != activeSchemaDigest || buildDigest != activeBuildDigest {
		return AdoptionPlan{}, errors.New("caller schema/build bindings do not match the active schema and running executable")
	}
	if err := RefreshSource(sourceRoot, discovery); err != nil {
		return AdoptionPlan{}, err
	}
	if err := ValidateDistillation(discovery, report); err != nil {
		return AdoptionPlan{}, err
	}
	if err := ValidateResolution(discovery, report, resolution); err != nil {
		return AdoptionPlan{}, err
	}
	if resolution.TargetBasis != target.Digest {
		return AdoptionPlan{}, errors.New("resolution target basis differs from the loaded project snapshot")
	}
	if err := ValidateDistillationTarget(report, target); err != nil {
		return AdoptionPlan{}, err
	}
	if resolution.SchemaDigest != schemaDigest || !validDigest(schemaDigest) {
		return AdoptionPlan{}, errors.New("resolution schema digest differs from the current project schema")
	}
	if resolution.BuildDigest != buildDigest || !validDigest(buildDigest) {
		return AdoptionPlan{}, errors.New("resolution build digest differs from the current Markitect build")
	}
	selectedScopes := map[string]bool{}
	deferredScopes := map[string]bool{}
	for _, decision := range resolution.Scopes {
		if decision.Status == "adopt" {
			selectedScopes[decision.ScopeID] = true
		} else {
			deferredScopes[decision.ScopeID] = true
		}
	}
	files := make([]projectwork.FileChange, 0, len(report.Proposal.Files)+1)
	modelFiles := make([]string, 0, len(report.Proposal.Files))
	seenPaths := map[string]bool{}
	for _, proposed := range report.Proposal.Files {
		if !selectedScopes[proposed.ScopeID] {
			continue
		}
		key := strings.ToLower(proposed.Path)
		if seenPaths[key] {
			return AdoptionPlan{}, fmt.Errorf("duplicate or case-aliased proposal path %q", proposed.Path)
		}
		seenPaths[key] = true
		files = append(files, projectwork.FileChange{Path: proposed.Path, Content: proposed.Content})
		modelFiles = append(modelFiles, proposed.Path)
	}
	if len(selectedScopes) == 0 {
		return AdoptionPlan{}, errors.New("resolution defers every scope; there is no adoption plan")
	}
	for _, id := range sortedMapKeys(selectedScopes) {
		found := false
		for _, proposed := range report.Proposal.Files {
			if proposed.ScopeID == id {
				found = true
				break
			}
		}
		if !found {
			return AdoptionPlan{}, fmt.Errorf("adopted scope %q has no canonical model files", id)
		}
	}
	updatedManifest, err := manifestWithModelFiles(target.Config, modelFiles)
	if err != nil {
		return AdoptionPlan{}, err
	}
	files = append(files, projectwork.FileChange{Path: projectwork.ManifestPath, Content: updatedManifest})
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	mutation := projectwork.Mutation{
		APIVersion: projectmodel.APIVersion,
		BaseDigest: target.Digest,
		Actor:      resolution.Actor,
		Goal:       "Brownfield adoption: " + resolution.DecisionReference,
		Files:      files,
	}
	edit, err := projectwork.PlanEdit(target, mutation)
	if err != nil {
		return AdoptionPlan{}, fmt.Errorf("validate canonical model adoption proposal: %w", err)
	}
	status := "complete"
	if len(deferredScopes) > 0 {
		status = "partial"
	}
	plan := AdoptionPlan{
		APIVersion: AdoptionPlanVersion, DiscoveryDigest: discovery.Digest,
		DistillationDigest: report.Digest, ResolutionDigest: resolution.Digest,
		TargetBasis: target.Digest, Status: status, AdoptedScopes: sortedMapKeys(selectedScopes),
		DeferredScopes: sortedMapKeys(deferredScopes), Edit: edit,
	}
	copy := plan
	copy.PlanDigest = ""
	plan.PlanDigest = digestValue(copy)
	return plan, nil
}

// ApplyAdoption rechecks source, target, proposal and resolution bindings,
// then delegates only the validated Markitect-owned changes to projectwork.
func ApplyAdoption(sourceRoot, targetRoot string, target *projectwork.Project, discovery Discovery, report Distillation, resolution Resolution, plan AdoptionPlan, expectedPlanDigest, schemaDigest, buildDigest string) (AdoptionReceipt, error) {
	if err := ValidateAdoptionPlan(plan); err != nil {
		return AdoptionReceipt{}, err
	}
	if expectedPlanDigest == "" || expectedPlanDigest != plan.PlanDigest {
		return AdoptionReceipt{}, errors.New("apply requires the exact reviewed adoption plan digest")
	}
	current, err := PlanAdoption(sourceRoot, target, discovery, report, resolution, schemaDigest, buildDigest)
	if err != nil {
		return AdoptionReceipt{}, err
	}
	if current.PlanDigest != plan.PlanDigest {
		return AdoptionReceipt{}, errors.New("adoption plan is stale or modified; plan again against the current target")
	}
	if !sameFilesystemPath(target.Root, targetRoot) {
		return AdoptionReceipt{}, errors.New("target root differs from the loaded project root")
	}
	if err := validateCanonicalOnly(current.Edit.Mutation.Files); err != nil {
		return AdoptionReceipt{}, err
	}
	result, err := projectwork.ApplyEdit(targetRoot, current.Edit, current.Edit.BaseDigest)
	if err != nil {
		return AdoptionReceipt{}, fmt.Errorf("apply canonical project model adoption: %w", err)
	}
	if result.CandidateDigest != current.Edit.CandidateDigest {
		return AdoptionReceipt{}, errors.New("applied adoption candidate differs from the reviewed candidate")
	}
	status := "adopted"
	if current.Status == "partial" {
		status = "adopted-partial"
	}
	return AdoptionReceipt{
		Status: status, Adopted: append([]string(nil), current.AdoptedScopes...),
		Deferred:        append([]string(nil), current.DeferredScopes...),
		DiscoveryDigest: current.DiscoveryDigest, DistillationDigest: current.DistillationDigest,
		ResolutionDigest: current.ResolutionDigest, PriorTargetBasis: current.TargetBasis,
		CandidateDigest: result.CandidateDigest,
	}, nil
}

// ValidateAdoptionPlan checks the closed digest and write boundary of a plan
// record. ApplyAdoption additionally recomputes it from current bound inputs.
func ValidateAdoptionPlan(plan AdoptionPlan) error {
	if plan.APIVersion != AdoptionPlanVersion || !validDigest(plan.DiscoveryDigest) || !validDigest(plan.DistillationDigest) || !validDigest(plan.ResolutionDigest) || !validDigest(plan.TargetBasis) {
		return errors.New("adoption plan requires supported version and exact input digests")
	}
	if plan.Status != "complete" && plan.Status != "partial" {
		return errors.New("adoption plan status must be complete or partial")
	}
	if plan.Status == "complete" && len(plan.DeferredScopes) != 0 || plan.Status == "partial" && len(plan.DeferredScopes) == 0 {
		return errors.New("adoption plan status does not match its deferred scopes")
	}
	if plan.AdoptedScopes == nil || plan.DeferredScopes == nil || len(plan.AdoptedScopes) == 0 {
		return errors.New("adoption plan must explicitly bind adopted and deferred scopes")
	}
	seen := map[string]bool{}
	for _, id := range append(append([]string(nil), plan.AdoptedScopes...), plan.DeferredScopes...) {
		if !validID(id) || seen[id] {
			return fmt.Errorf("adoption plan has invalid or duplicate scope %q", id)
		}
		seen[id] = true
	}
	if plan.Edit.BaseDigest != plan.TargetBasis || plan.Edit.Mutation.BaseDigest != plan.TargetBasis || len(plan.Edit.Mutation.Files) == 0 {
		return errors.New("adoption plan edit does not bind the exact target snapshot")
	}
	if err := validateCanonicalOnly(plan.Edit.Mutation.Files); err != nil {
		return err
	}
	copy := plan
	copy.PlanDigest = ""
	if !validDigest(plan.PlanDigest) || digestValue(copy) != plan.PlanDigest {
		return errors.New("adoption plan digest mismatch")
	}
	return nil
}

type AdoptionReceipt struct {
	Status             string   `json:"status"`
	Adopted            []string `json:"adoptedScopes"`
	Deferred           []string `json:"deferredScopes"`
	DiscoveryDigest    string   `json:"discoveryDigest"`
	DistillationDigest string   `json:"distillationDigest"`
	ResolutionDigest   string   `json:"resolutionDigest"`
	PriorTargetBasis   string   `json:"priorTargetBasis"`
	CandidateDigest    string   `json:"candidateDigest"`
}

func RefreshSource(root string, discovery Discovery) error {
	_, err := RefreshDiscovery(root, discovery)
	return err
}

func manifestWithModelFiles(config projectwork.Config, additions []string) (string, error) {
	seen := map[string]bool{}
	files := make([]string, 0, len(config.ModelFiles)+len(additions))
	for _, name := range config.ModelFiles {
		if seen[strings.ToLower(name)] {
			return "", fmt.Errorf("target manifest contains duplicate model path %q", name)
		}
		seen[strings.ToLower(name)] = true
		files = append(files, name)
	}
	for _, name := range additions {
		if !seen[strings.ToLower(name)] {
			seen[strings.ToLower(name)] = true
			files = append(files, name)
		}
	}
	sort.Strings(files)
	config.ModelFiles = files
	data, err := yaml.Marshal(config)
	if err != nil {
		return "", fmt.Errorf("encode updated project manifest: %w", err)
	}
	return string(data), nil
}

func validateCanonicalOnly(files []projectwork.FileChange) error {
	for _, file := range files {
		if file.Delete || (file.Path != projectwork.ManifestPath && !strings.HasPrefix(file.Path, projectwork.ModelRoot+"/")) {
			return fmt.Errorf("adoption cannot write outside canonical project metadata/model paths: %s", file.Path)
		}
	}
	return nil
}

func sameFilesystemPath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	left, leftErr := os.Stat(a)
	right, rightErr := os.Stat(b)
	if leftErr != nil || rightErr != nil || !left.IsDir() || !right.IsDir() {
		return false
	}
	return os.SameFile(left, right)
}

func sortedMapKeys(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}
