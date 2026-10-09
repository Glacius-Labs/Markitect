package projectapp

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

// ExploreOperation lists, reads, or prepares a guarded one-scope record write.
type ExploreOperation struct {
	Selection      Selection              `json:"selection"`
	ExplorationID  string                 `json:"explorationId,omitempty"`
	Record         *projectexplore.Record `json:"record,omitempty"`
	Write          bool                   `json:"write,omitempty"`
	ExpectedDigest string                 `json:"expectedDigest,omitempty"`
}

// ExploreResult is a typed union: exactly one of Records, Record, or Plan is
// populated according to the request. Persisted is populated after a write.
type ExploreResult struct {
	Records   []projectexplore.Record   `json:"records,omitempty"`
	Record    *projectexplore.Record    `json:"record,omitempty"`
	Plan      *projectexplore.WritePlan `json:"plan,omitempty"`
	Persisted *projectexplore.Record    `json:"persisted,omitempty"`
}

// Explore lists or reads durable records, or previews/writes one record against
// an exact model-only binding. It never loads runtime configuration or invokes
// an agent.
func (o Operations) Explore(operation ExploreOperation) (ExploreResult, error) {
	if err := requireRoot(operation.Selection.Root); err != nil {
		return ExploreResult{}, err
	}
	if operation.Record == nil {
		if operation.Write || operation.ExpectedDigest != "" {
			return ExploreResult{}, errors.New("project explore write requires --input with a record to preview")
		}
		if operation.ExplorationID != "" {
			record, err := projectexplore.Load(operation.Selection.Root, operation.ExplorationID)
			if err != nil {
				return ExploreResult{}, err
			}
			return ExploreResult{Record: &record}, nil
		}
		records, err := projectexplore.List(operation.Selection.Root)
		if err != nil {
			return ExploreResult{}, err
		}
		return ExploreResult{Records: records}, nil
	}
	// Clone and validate transport-owned input before filling derived fields or
	// acknowledgements so the operation never mutates its caller's object.
	encodedRecord, err := json.Marshal(operation.Record)
	if err != nil {
		return ExploreResult{}, err
	}
	record, err := projectexplore.DecodeRecordInput(encodedRecord)
	if err != nil {
		return ExploreResult{}, err
	}
	if operation.ExplorationID != "" && operation.ExplorationID != record.ID {
		return ExploreResult{}, fmt.Errorf("--exploration %q does not match input record ID %q", operation.ExplorationID, record.ID)
	}
	if len(record.Scopes) != 1 {
		return ExploreResult{}, fmt.Errorf("exploration %s must contain exactly one named scope", record.ID)
	}
	root := operation.Selection.Root
	binding, err := explorationCRUDModelBinding(o.Host, root, operation.Selection.Revision, record.Scopes[0])
	if err != nil {
		return ExploreResult{}, err
	}
	records, err := projectexplore.List(root)
	if err != nil {
		return ExploreResult{}, err
	}
	var prior *projectexplore.Record
	for i := range records {
		if records[i].ID == record.ID {
			prior = &records[i]
			break
		}
	}
	var plan projectexplore.WritePlan
	if prior != nil {
		if record.CreatedAgainst == "" {
			record.CreatedAgainst = prior.CreatedAgainst
			record.Digest = ""
		}
		plan, err = projectexplore.UpdatePreview(root, record, prior.Digest, binding)
	} else {
		// CreatePreview fills the initial source binding. The input digest is
		// cleared when that field is absent because it participates in hashing.
		if record.CreatedAgainst == "" {
			record.Digest = ""
		}
		plan, err = projectexplore.CreatePreview(root, record, binding)
	}
	if err != nil {
		return ExploreResult{}, fmt.Errorf("prepare exploration write: %w", err)
	}
	result := ExploreResult{Plan: &plan}
	if !operation.Write {
		return result, nil
	}
	if operation.ExpectedDigest != plan.Digest {
		return ExploreResult{}, fmt.Errorf("--expect does not match the exact exploration write-plan digest %s", plan.Digest)
	}
	persisted, err := projectexplore.Write(root, plan, operation.ExpectedDigest, binding)
	if err != nil {
		return ExploreResult{}, err
	}
	result.Persisted = &persisted
	return result, nil
}

// StructureAcknowledgementInput carries explicit human assertion details.
// Scope and digests are calculated from the selected durable record and current
// binding inside the application operation.
type StructureAcknowledgementInput struct {
	Actor      string    `json:"actor"`
	Authority  string    `json:"authority"`
	Provenance string    `json:"provenance"`
	RecordedAt time.Time `json:"recordedAt"`
}

type ReadinessOperation struct {
	Selection       Selection                      `json:"selection"`
	ExplorationID   string                         `json:"explorationId"`
	ScopeID         string                         `json:"scopeId"`
	Acknowledgement *StructureAcknowledgementInput `json:"acknowledgement,omitempty"`
	Write           bool                           `json:"write,omitempty"`
	ExpectedDigest  string                         `json:"expectedDigest,omitempty"`
}

type ReadinessResult struct {
	Exploration projectexplore.Record          `json:"exploration"`
	Scope       projectexplore.Scope           `json:"scope"`
	Binding     projectexplore.Binding         `json:"binding"`
	Readiness   projectexplore.ReadinessReport `json:"readiness"`
	WritePlan   *projectexplore.WritePlan      `json:"writePlan,omitempty"`
	Persisted   *projectexplore.Record         `json:"persisted,omitempty"`
}

// Readiness computes the current named-scope binding and optionally prepares or
// writes a digest-bound explicit acknowledgement through the store CAS.
func (o Operations) Readiness(operation ReadinessOperation) (ReadinessResult, error) {
	if err := requireRoot(operation.Selection.Root); err != nil {
		return ReadinessResult{}, err
	}
	if operation.Write && operation.Acknowledgement == nil {
		return ReadinessResult{}, errors.New("project readiness --write requires --acknowledge-structure")
	}
	if operation.Acknowledgement != nil {
		ack := operation.Acknowledgement
		if strings.TrimSpace(ack.Actor) == "" || strings.TrimSpace(ack.Authority) == "" || strings.TrimSpace(ack.Provenance) == "" {
			return ReadinessResult{}, errors.New("--acknowledge-structure requires --actor, --authority, and --decision-ref")
		}
		if ack.RecordedAt.IsZero() {
			return ReadinessResult{}, errors.New("--acknowledged-at must be an explicit RFC3339 timestamp")
		}
	}
	root := operation.Selection.Root
	record, err := projectexplore.Load(root, operation.ExplorationID)
	if err != nil {
		return ReadinessResult{}, err
	}
	scope, ok := findScope(record, operation.ScopeID)
	if !ok {
		return ReadinessResult{}, fmt.Errorf("exploration %s has no scope %q", record.ID, operation.ScopeID)
	}
	binding, err := readinessBinding(o.Host, root, operation.Selection.Revision, record, scope)
	if err != nil {
		return ReadinessResult{}, err
	}
	readiness, err := projectexplore.EvaluateReadiness(record, scope.ID, binding)
	if err != nil {
		return ReadinessResult{}, err
	}
	result := ReadinessResult{Exploration: record, Scope: scope, Binding: binding, Readiness: readiness}
	if operation.Acknowledgement == nil {
		return result, nil
	}
	priorDigest := record.Digest
	ackInput := operation.Acknowledgement
	ack := projectexplore.StructureAcknowledgement{
		ScopeID: scope.ID, BindingDigest: readiness.BindingDigest, StructureDigest: readiness.StructureDigest,
		Actor: ackInput.Actor, Authority: ackInput.Authority, Provenance: ackInput.Provenance, RecordedAt: ackInput.RecordedAt.UTC(),
	}
	if err := projectexplore.AcknowledgeStructure(&record, scope.ID, binding, ack); err != nil {
		return ReadinessResult{}, err
	}
	plan, err := projectexplore.UpdatePreview(root, record, priorDigest, binding)
	if err != nil {
		return ReadinessResult{}, err
	}
	readiness, err = projectexplore.EvaluateReadiness(record, scope.ID, binding)
	if err != nil {
		return ReadinessResult{}, err
	}
	result.Exploration, result.Readiness, result.WritePlan = record, readiness, &plan
	if !operation.Write {
		return result, nil
	}
	if operation.ExpectedDigest != plan.Digest {
		return ReadinessResult{}, fmt.Errorf("--expect does not match the exact readiness acknowledgement plan digest %s", plan.Digest)
	}
	written, err := projectexplore.Write(root, plan, operation.ExpectedDigest, binding)
	if err != nil {
		return ReadinessResult{}, err
	}
	result.Persisted = &written
	return result, nil
}

func readinessBinding(host projectrun.Host, root, revision string, record projectexplore.Record, scope projectexplore.Scope) (projectexplore.Binding, error) {
	if strings.TrimSpace(scope.Goal) == "" {
		return projectexplore.Binding{}, fmt.Errorf("exploration scope %q has no goal", scope.ID)
	}
	return projectrun.ExplorationBinding(host, root, revision, projectrun.PlanRequest{
		ExplorationID: record.ID, ScopeID: scope.ID, Goal: scope.Goal, Operation: scope.Operation,
		Managers: append([]string(nil), scope.ManagerIDs...),
	})
}

// explorationCRUDModelBinding is model-only: record edits remain possible
// before runtime setup and model acceptance, while CAS still binds exact source.
func explorationCRUDModelBinding(host projectrun.Host, root, revision string, scope projectexplore.Scope) (projectexplore.Binding, error) {
	var binding projectexplore.Binding
	project, err := host.Load(root, revision)
	if err != nil {
		return binding, err
	}
	if project == nil || project.Snapshot == nil {
		return binding, errors.New("exploration requires an exact project model snapshot")
	}
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return binding, err
	}
	branchBytes, err := source.GitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || strings.TrimSpace(string(branchBytes)) == "" {
		return binding, errors.New("exploration record writes require a named Git branch")
	}
	headBytes, headErr := source.GitOutput(root, "rev-parse", "--verify", "HEAD^{commit}")
	head := ""
	if headErr == nil {
		head = strings.TrimSpace(string(headBytes))
	}
	selectionBytes, err := json.Marshal(struct {
		ProjectDigest string   `json:"projectDigest"`
		ModelDigest   string   `json:"modelDigest"`
		ScopeID       string   `json:"scopeId"`
		ScopeName     string   `json:"scopeName"`
		Goal          string   `json:"goal"`
		Operation     string   `json:"operation"`
		ManagerIDs    []string `json:"managerIds"`
	}{project.Digest, project.Report.ModelDigest, scope.ID, scope.Name, scope.Goal, scope.Operation, scope.ManagerIDs})
	if err != nil {
		return binding, err
	}
	selectionSum := sha256.Sum256(selectionBytes)
	managerSet := make(map[string]bool, len(scope.ManagerIDs))
	for _, id := range scope.ManagerIDs {
		managerSet[id] = true
	}
	managerSelected := func(id string) bool { return len(managerSet) == 0 || managerSet[id] }
	artifacts, files, checks := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, artifact := range project.Report.Artifacts {
		if !managerSelected(artifact.Owner) {
			continue
		}
		if artifact.Required {
			artifacts[artifact.ID] = true
		}
		for _, path := range artifact.Paths {
			files[path] = true
		}
	}
	for _, entry := range project.Report.Files {
		if entry.Owner == "" || managerSelected(entry.Owner) {
			files[entry.Path] = true
		}
	}
	for _, check := range project.Report.Checks {
		if managerSelected(check.Owner) {
			checks[check.ID] = true
		}
	}
	if project.Config.DocumentPath != "" {
		files[project.Config.DocumentPath] = true
	}
	stringKeys := func(values map[string]bool) []string {
		out := make([]string, 0, len(values))
		for value := range values {
			out = append(out, value)
		}
		sort.Strings(out)
		return out
	}
	basisByPath := make(map[string]string, len(project.Snapshot.Files)+1)
	for path, data := range project.Snapshot.Files {
		basisByPath[path] = bytesDigest(data)
	}
	runtime, runtimeErr := source.ObserveSelectedWorking(root, []string{projectwork.RuntimePath})
	if runtimeErr != nil {
		return binding, fmt.Errorf("capture optional runtime basis: %w", runtimeErr)
	}
	if data, ok := runtime.Snapshot.Files[projectwork.RuntimePath]; ok {
		basisByPath[projectwork.RuntimePath] = bytesDigest(data)
	}
	basisFiles := make([]projectexplore.BasisFile, 0, len(basisByPath))
	for path, digest := range basisByPath {
		basisFiles = append(basisFiles, projectexplore.BasisFile{Path: path, Digest: digest})
	}
	sort.Slice(basisFiles, func(i, j int) bool { return basisFiles[i].Path < basisFiles[j].Path })
	binding = projectexplore.Binding{
		RepositoryRoot: identity.Root, Branch: strings.TrimSpace(string(branchBytes)), Head: head,
		ModelRevision: project.Revision, ModelAccepted: false, AcceptancePolicy: project.Config.AcceptancePolicy,
		ProjectDigest: prefixedDigest(project.Digest), ModelDigest: prefixedDigest(project.Report.ModelDigest), SnapshotDigest: prefixedDigest(project.Snapshot.Digest()),
		SelectionDigest: "sha256:" + hex.EncodeToString(selectionSum[:]), ScopeID: scope.ID, ScopeName: scope.Name,
		Goal: scope.Goal, Operation: scope.Operation, ManagerIDs: append([]string{}, scope.ManagerIDs...),
		RequiredArtifacts: stringKeys(artifacts), FileStructure: stringKeys(files), Checks: stringKeys(checks), BasisFiles: basisFiles,
	}
	_, err = projectexplore.BindingDigest(binding)
	if err != nil {
		return binding, fmt.Errorf("validate model-only scope binding %q (scopeName=%q goal=%q operation=%q managers=%d artifacts=%d files=%d checks=%d basis=%d): %w",
			scope.ID, binding.ScopeName, binding.Goal, binding.Operation, len(binding.ManagerIDs), len(binding.RequiredArtifacts), len(binding.FileStructure), len(binding.Checks), len(binding.BasisFiles), err)
	}
	return binding, nil
}

func bytesDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func prefixedDigest(value string) string {
	if strings.HasPrefix(value, "sha256:") {
		return value
	}
	return "sha256:" + value
}

func findScope(record projectexplore.Record, id string) (projectexplore.Scope, bool) {
	for _, scope := range record.Scopes {
		if scope.ID == id {
			return scope, true
		}
	}
	return projectexplore.Scope{}, false
}
