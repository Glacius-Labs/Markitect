package projectcli

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// runExplore lists, reads, or previews a durable one-scope work-item record.
// Record writes are always tied to the current model-only binding; this path
// deliberately does not load a runtime or start an agent.
func runExplore(opts options, out io.Writer) error {
	if opts.input == "" {
		if opts.write || opts.expect != "" {
			return fmt.Errorf("project explore write requires --input with a record to preview")
		}
		if opts.explorationID != "" {
			record, err := projectexplore.Load(opts.repo, opts.explorationID)
			if err != nil {
				return err
			}
			return writeJSON(out, record)
		}
		records, err := projectexplore.List(opts.repo)
		if err != nil {
			return err
		}
		return writeJSON(out, records)
	}

	data, err := readRecord(opts.repo, opts.input)
	if err != nil {
		return fmt.Errorf("read exploration input: %w", err)
	}
	record, err := projectexplore.DecodeRecordInput(data)
	if err != nil {
		return err
	}
	if opts.explorationID != "" && opts.explorationID != record.ID {
		return fmt.Errorf("--exploration %q does not match input record ID %q", opts.explorationID, record.ID)
	}
	if len(record.Scopes) != 1 {
		return fmt.Errorf("exploration %s must contain exactly one named scope", record.ID)
	}
	binding, err := explorationCRUDBinding(opts.repo, opts.revision, record.Scopes[0])
	if err != nil {
		return err
	}

	var prior *projectexplore.Record
	records, err := projectexplore.List(opts.repo)
	if err != nil {
		return err
	}
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
		plan, err = projectexplore.UpdatePreview(opts.repo, record, prior.Digest, binding)
	} else {
		// CreatePreview fills the initial source binding. Clear the input digest
		// because that single field is part of the record digest.
		if record.CreatedAgainst == "" {
			record.Digest = ""
		}
		plan, err = projectexplore.CreatePreview(opts.repo, record, binding)
	}
	if err != nil {
		return fmt.Errorf("prepare exploration write: %w", err)
	}
	if opts.write {
		if opts.expect != plan.Digest {
			return fmt.Errorf("--expect does not match the exact exploration write-plan digest %s", plan.Digest)
		}
		persisted, writeErr := projectexplore.Write(opts.repo, plan, opts.expect, binding)
		if writeErr != nil {
			return writeErr
		}
		return writeJSON(out, struct {
			Record projectexplore.Record    `json:"record"`
			Plan   projectexplore.WritePlan `json:"plan"`
		}{persisted, plan})
	}
	return writeJSON(out, plan)
}

// runReadiness computes and displays the exact current named-scope structure.
// Acknowledgment is a separate, digest-bound, CAS-protected user assertion.
func runReadiness(opts options, out io.Writer) error {
	if opts.write && !opts.acknowledgeStructure {
		return fmt.Errorf("project readiness --write requires --acknowledge-structure")
	}
	if opts.acknowledgeStructure && (strings.TrimSpace(opts.actor) == "" || strings.TrimSpace(opts.authority) == "" || strings.TrimSpace(opts.decisionRef) == "") {
		return fmt.Errorf("--acknowledge-structure requires --actor, --authority, and --decision-ref")
	}
	record, err := projectexplore.Load(opts.repo, opts.explorationID)
	if err != nil {
		return err
	}
	scope, ok := findExplorationScope(record, opts.scope)
	if !ok {
		return fmt.Errorf("exploration %s has no scope %q", record.ID, opts.scope)
	}
	binding, err := readinessBinding(opts.repo, opts.revision, record, scope)
	if err != nil {
		return err
	}
	readiness, err := projectexplore.EvaluateReadiness(record, scope.ID, binding)
	if err != nil {
		return err
	}
	var preview *projectexplore.WritePlan
	var persisted *projectexplore.Record
	if opts.acknowledgeStructure {
		priorDigest := record.Digest
		recordedAt, timeErr := time.Parse(time.RFC3339Nano, opts.acknowledgedAt)
		if timeErr != nil || strings.TrimSpace(opts.acknowledgedAt) == "" {
			return fmt.Errorf("--acknowledged-at must be an explicit RFC3339 timestamp")
		}
		ack := projectexplore.StructureAcknowledgement{
			ScopeID: scope.ID, BindingDigest: readiness.BindingDigest, StructureDigest: readiness.StructureDigest,
			Actor: opts.actor, Authority: opts.authority, Provenance: opts.decisionRef, RecordedAt: recordedAt.UTC(),
		}
		if err := projectexplore.AcknowledgeStructure(&record, scope.ID, binding, ack); err != nil {
			return err
		}
		plan, planErr := projectexplore.UpdatePreview(opts.repo, record, priorDigest, binding)
		if planErr != nil {
			return planErr
		}
		preview = &plan
		readiness, err = projectexplore.EvaluateReadiness(record, scope.ID, binding)
		if err != nil {
			return err
		}
		if opts.write {
			if opts.expect != plan.Digest {
				return fmt.Errorf("--expect does not match the exact readiness acknowledgement plan digest %s", plan.Digest)
			}
			written, writeErr := projectexplore.Write(opts.repo, plan, opts.expect, binding)
			if writeErr != nil {
				return writeErr
			}
			persisted = &written
		}
	}

	return writeJSON(out, struct {
		Exploration projectexplore.Record          `json:"exploration"`
		Scope       projectexplore.Scope           `json:"scope"`
		Binding     projectexplore.Binding         `json:"binding"`
		Readiness   projectexplore.ReadinessReport `json:"readiness"`
		WritePlan   *projectexplore.WritePlan      `json:"writePlan,omitempty"`
		Persisted   *projectexplore.Record         `json:"persisted,omitempty"`
	}{record, scope, binding, readiness, preview, persisted})
}

func readinessBinding(root, revision string, record projectexplore.Record, scope projectexplore.Scope) (projectexplore.Binding, error) {
	if strings.TrimSpace(scope.Goal) == "" {
		return projectexplore.Binding{}, fmt.Errorf("exploration scope %q has no goal", scope.ID)
	}
	return projectrun.ExplorationBinding(projectRunHost(), root, revision, projectrun.PlanRequest{
		ExplorationID: record.ID,
		ScopeID:       scope.ID,
		Goal:          scope.Goal,
		Operation:     scope.Operation,
		Managers:      append([]string(nil), scope.ManagerIDs...),
	})
}

// explorationCRUDBinding is deliberately model-only. Creating or updating a
// work item must remain possible before runtime setup and before model
// acceptance; it still binds the exact selected source bytes under the guarded
// store CAS so a preview cannot survive a source change.
func explorationCRUDBinding(root, revision string, scope projectexplore.Scope) (projectexplore.Binding, error) {
	var binding projectexplore.Binding
	project, err := projectwork.Load(root, revision)
	if err != nil {
		return binding, err
	}
	if project == nil || project.Snapshot == nil {
		return binding, fmt.Errorf("exploration requires an exact project model snapshot")
	}
	identity, err := source.IdentifyGit(root)
	if err != nil {
		return binding, err
	}
	branchBytes, err := source.GitOutput(root, "symbolic-ref", "--quiet", "--short", "HEAD")
	if err != nil || strings.TrimSpace(string(branchBytes)) == "" {
		return binding, fmt.Errorf("exploration record writes require a named Git branch")
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
		basisByPath[path] = projectBytesDigest(data)
	}
	runtime, runtimeErr := source.ObserveSelectedWorking(root, []string{projectwork.RuntimePath})
	if runtimeErr != nil {
		return binding, fmt.Errorf("capture optional runtime basis: %w", runtimeErr)
	}
	if data, ok := runtime.Snapshot.Files[projectwork.RuntimePath]; ok {
		basisByPath[projectwork.RuntimePath] = projectBytesDigest(data)
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

func projectBytesDigest(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func prefixedDigest(value string) string {
	if strings.HasPrefix(value, "sha256:") {
		return value
	}
	return "sha256:" + value
}

func findExplorationScope(record projectexplore.Record, id string) (projectexplore.Scope, bool) {
	for _, scope := range record.Scopes {
		if scope.ID == id {
			return scope, true
		}
	}
	return projectexplore.Scope{}, false
}
