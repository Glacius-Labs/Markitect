// Package knowledgeevidence projects explicitly selected, validated Host
// records into the neutral project knowledge facts model. It is read-only and
// never becomes an authority or a second evidence ledger.
package knowledgeevidence

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/internal/host/projectexplore"
	"github.com/Glacius-Labs/Markitect/internal/host/projectrun"
	"github.com/Glacius-Labs/Markitect/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectknowledge"
)

const maxSelectedRecords = 64

type Scope struct {
	ManagerID            string   `json:"managerId,omitempty"`
	WholeProject         bool     `json:"wholeProject,omitempty"`
	VisibleDefinitionIDs []string `json:"visibleDefinitionIds,omitempty"`
	VisiblePaths         []string `json:"visiblePaths,omitempty"`
}

// Selection is deliberately explicit. Empty IDs mean that the corresponding
// operational ledger is not opened. Briefing history is opt-in separately.
type Selection struct {
	RunIDs                 []string `json:"runIds,omitempty"`
	ExplorationIDs         []string `json:"explorationIds,omitempty"`
	BrownfieldSessionIDs   []string `json:"brownfieldSessionIds,omitempty"`
	IncludeBriefingHistory bool     `json:"includeBriefingHistory,omitempty"`
}

type SourceBinding struct {
	Kind          string `json:"kind"`
	RecordID      string `json:"recordId"`
	Digest        string `json:"digest"`
	Schema        string `json:"schema"`
	Source        string `json:"source"` // live-operational-record or selected-model-history
	ModelDigest   string `json:"modelDigest,omitempty"`
	Revision      string `json:"revision,omitempty"`
	RuntimeDigest string `json:"runtimeDigest,omitempty"`
	DigestKind    string `json:"digestKind,omitempty"`
}

type UnknownRecord struct {
	Kind   string `json:"kind"`
	ID     string `json:"id,omitempty"`
	Reason string `json:"reason"`
}

type Capture struct {
	Facts          projectknowledge.ProjectFacts `json:"facts"`
	CaptureDigest  string                        `json:"captureDigest"`
	Completeness   string                        `json:"completeness"`
	Unknown        []UnknownRecord               `json:"unknown,omitempty"`
	SourceBindings []SourceBinding               `json:"sourceBindings,omitempty"`
}

func Read(project *projectwork.Project, scope Scope, selection Selection) (Capture, error) {
	if project == nil || project.Snapshot == nil || project.Digest == "" || project.Model.Digest == "" {
		return Capture{}, errors.New("knowledge evidence requires a loaded project and source snapshot")
	}
	if err := validateScope(scope); err != nil {
		return Capture{}, err
	}
	if len(selection.RunIDs)+len(selection.ExplorationIDs)+len(selection.BrownfieldSessionIDs) > maxSelectedRecords {
		return Capture{}, fmt.Errorf("selected record count exceeds %d", maxSelectedRecords)
	}
	out := Capture{Completeness: projectknowledge.FactKnown, Unknown: []UnknownRecord{}, SourceBindings: []SourceBinding{}}
	root := project.Root
	if root == "" {
		return Capture{}, errors.New("loaded project has no operational record root")
	}
	snapshotDigest := project.Snapshot.Digest()
	currentDefs := map[string]bool{}
	for _, definition := range project.Model.Definitions {
		currentDefs[definition.Identity().Key()] = true
	}
	facts := factBuilder{out: &out, scope: scope, defs: stringSet(scope.VisibleDefinitionIDs), currentDefs: currentDefs, paths: stringSet(scope.VisiblePaths), projectRevision: project.Revision, snapshotDigest: snapshotDigest, modelDigest: project.Model.Digest, projectDigest: project.Digest, added: map[string]bool{}, runCaptureConsistent: map[string]bool{}}
	for _, id := range sortedUnique(selection.RunIDs) {
		if id == "" {
			return Capture{}, errors.New("selected run ID is empty")
		}
		records, err := projectrun.KnowledgeRecords(root, id)
		if err != nil {
			if isMissing(err) || scope.ManagerID != "" {
				facts.unknown("run", id, "selected run is unavailable")
				continue
			}
			return Capture{}, fmt.Errorf("selected run %q is invalid or unavailable", id)
		}
		if records.PlanState != projectrun.KnowledgeRecordPresent || !facts.runVisible(records.Status.Plan) {
			facts.unknown("run", id, "selected run is unavailable")
			continue
		}
		if records.RunState != projectrun.KnowledgeRecordPresent {
			facts.addIncompleteRun(id, records)
			continue
		}
		facts.addRun(records)
	}
	for _, id := range sortedUnique(selection.ExplorationIDs) {
		if id == "" {
			return Capture{}, errors.New("selected exploration ID is empty")
		}
		record, err := projectexplore.Load(root, id)
		if err != nil {
			if isMissing(err) || scope.ManagerID != "" {
				facts.unknown("exploration", id, "selected exploration is unavailable")
				continue
			}
			return Capture{}, fmt.Errorf("selected exploration %q is invalid or unavailable", id)
		}
		if !facts.explorationVisible(record) {
			facts.unknown("exploration", id, "selected exploration is unavailable")
			continue
		}
		fenced, fenceErr := projectexplore.Load(root, id)
		if fenceErr != nil || fenced.Digest != record.Digest {
			facts.unknown("exploration", id, "selected exploration changed during capture")
			continue
		}
		facts.addExploration(record)
	}
	for _, id := range sortedUnique(selection.BrownfieldSessionIDs) {
		if id == "" {
			return Capture{}, errors.New("selected Brownfield session ID is empty")
		}
		session, err := projectadoption.LoadBrownfieldSession(root, id)
		if err != nil {
			if isMissing(err) || scope.ManagerID != "" {
				facts.unknown("brownfield-session", id, "selected Brownfield session is unavailable")
				continue
			}
			return Capture{}, fmt.Errorf("selected Brownfield session %q is invalid or unavailable", id)
		}
		if !facts.brownfieldVisible(session) {
			facts.unknown("brownfield-session", id, "selected Brownfield session is unavailable")
			continue
		}
		fenced, fenceErr := projectadoption.LoadBrownfieldSession(root, id)
		if fenceErr != nil || fenced.Digest != session.Digest {
			facts.unknown("brownfield-session", id, "selected Brownfield session changed during capture")
			continue
		}
		facts.addBrownfield(session)
	}
	if selection.IncludeBriefingHistory {
		if scope.ManagerID == "" {
			return Capture{}, errors.New("briefing history requires a named Manager scope")
		}
		store, beforeDigest, beforeErr := projectbriefing.Read(root)
		briefings, events, digest, err := projectbriefing.LoadForManager(root, project.Model.Digest, scope.ManagerID, project.Revision)
		_, afterDigest, afterErr := projectbriefing.Read(root)
		if err != nil || beforeErr != nil || afterErr != nil || beforeDigest != afterDigest {
			facts.unknown("briefing-history", scope.ManagerID, "selected briefing history is unavailable or stale")
		} else {
			facts.addHistory(briefings, events, digest, store)
		}
	}
	partialFacts := false
	for _, fact := range facts.facts {
		if fact.State != projectknowledge.FactKnown {
			partialFacts = true
			break
		}
	}
	if len(out.Unknown) != 0 || partialFacts {
		out.Completeness = projectknowledge.FactPartial
	}
	sort.Slice(out.SourceBindings, func(i, j int) bool {
		a, b := out.SourceBindings[i], out.SourceBindings[j]
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return a.RecordID < b.RecordID
	})
	out.Facts.SnapshotDigest = snapshotDigest
	out.Facts.ProjectDigest = project.Digest
	out.Facts.Facts = facts.facts
	out.Facts.Relations = facts.relations
	digest, err := digestJSON(struct {
		Facts     []projectknowledge.Fact
		Relations []projectknowledge.Relation
	}{facts.facts, facts.relations})
	if err != nil {
		return Capture{}, err
	}
	out.Facts.Digest = digest
	out.CaptureDigest, err = digestJSON(struct {
		SnapshotDigest string
		ProjectDigest  string
		FactsDigest    string
		Completeness   string
		Unknown        []UnknownRecord
		Sources        []SourceBinding
	}{out.Facts.SnapshotDigest, out.Facts.ProjectDigest, out.Facts.Digest, out.Completeness, out.Unknown, out.SourceBindings})
	if err != nil {
		return Capture{}, err
	}
	return out, nil
}

func validateScope(s Scope) error {
	if (s.ManagerID != "") == s.WholeProject {
		return errors.New("scope must select exactly one Manager or the whole project")
	}
	if s.ManagerID != "" && len(s.VisibleDefinitionIDs) == 0 && len(s.VisiblePaths) == 0 {
		return errors.New("Manager scope requires caller-selected visible definition IDs or paths")
	}
	return nil
}

type factBuilder struct {
	out                                          *Capture
	scope                                        Scope
	defs, currentDefs, paths                     map[string]bool
	facts                                        []projectknowledge.Fact
	relations                                    []projectknowledge.Relation
	projectRevision, snapshotDigest, modelDigest string
	projectDigest                                string
	added                                        map[string]bool
	runCaptureConsistent                         map[string]bool
}

func (b *factBuilder) unknown(kind, id, reason string) {
	if !b.scope.WholeProject {
		id = ""
	}
	b.out.Unknown = append(b.out.Unknown, UnknownRecord{kind, id, reason})
}
func (b *factBuilder) binding(kind, id, digest, schema, model, revision, runtime string) {
	b.out.SourceBindings = append(b.out.SourceBindings, SourceBinding{Kind: kind, RecordID: id, Digest: digest, Schema: schema, Source: "live-operational-record", ModelDigest: model, Revision: revision, RuntimeDigest: runtime})
}
func (b *factBuilder) add(id, kind, state string, props map[string]any, source string) {
	if b.added[id] {
		return
	}
	b.added[id] = true
	raw := map[string]json.RawMessage{}
	for k, v := range props {
		d, e := json.Marshal(v)
		if e == nil {
			raw[k] = d
		}
	}
	b.facts = append(b.facts, projectknowledge.Fact{ID: id, Kind: kind, State: state, Properties: raw, Source: core.Source{Path: "operational-record", Digest: source}})
}
func (b *factBuilder) edge(from, to, property, source, basis string) {
	b.relations = append(b.relations, projectknowledge.Relation{From: from, To: to, Property: property, Source: core.Source{Path: "operational-record", Digest: source}, Basis: basis})
}

func (b *factBuilder) runVisible(plan projectrun.PlanRecord) bool {
	if b.scope.WholeProject {
		return true
	}
	for _, task := range plan.Managers {
		if task.ManagerID == b.scope.ManagerID {
			return true
		}
	}
	return false
}

func (b *factBuilder) modelBindingStatus(revision, snapshot, model string) string {
	if revision == b.projectRevision && snapshot == b.snapshotDigest && model == b.modelDigest {
		return "current"
	}
	return "stale"
}

func (b *factBuilder) addIncompleteRun(selectedID string, records projectrun.KnowledgeRecordSet) {
	plan := records.Status.Plan
	planID, runID := "record/plan/"+plan.ID, "record/run/"+selectedID
	b.add(planID, "ExecutionPlan", projectknowledge.FactPartial, map[string]any{"digest": records.PlanDigest, "schema": records.PlanAPIVersion, "modelDigest": plan.BaseModelDigest, "sourceSnapshot": plan.BaseSnapshot, "baseRevision": plan.BaseRevision, "status": plan.Status, "modelBindingStatus": b.modelBindingStatus(plan.BaseRevision, plan.BaseSnapshot, plan.BaseModelDigest), "runtimeBindingStatus": string(records.RuntimeBindingState)}, records.PlanDigest)
	b.add(runID, "ExecutionRun", projectknowledge.FactUnknown, nil, "")
	b.edge(planID, runID, "expectsRun", records.PlanDigest, "selected plan exists while its execution record is unavailable")
	b.binding("plan", plan.ID, records.PlanDigest, records.PlanAPIVersion, plan.BaseModelDigest, plan.BaseRevision, records.RuntimeDigest)
	b.unknown("run", selectedID, "selected run state is unavailable")
	for _, planned := range plan.Managers {
		if !b.scope.WholeProject && planned.ManagerID != b.scope.ManagerID {
			continue
		}
		taskID := "record/task/" + selectedID + "/" + planned.ID
		props := map[string]any{"managerId": planned.ManagerID, "present": false, "expectedArtifacts": filteredDefinitions(planned.Artifacts, b.defs, b.scope.WholeProject), "expectedChecks": filteredDefinitions(planned.Checks, b.defs, b.scope.WholeProject)}
		statements := []string{}
		for _, statement := range planned.Statements {
			if b.scope.WholeProject || b.defs[statement] {
				statements = append(statements, statement)
			}
		}
		if b.scope.WholeProject || len(statements) > 0 {
			props["expectedStatements"] = statements
		}
		b.add(taskID, "ManagerTask", projectknowledge.FactPartial, props, records.PlanDigest)
		b.edge(planID, taskID, "plansTask", records.PlanDigest, "validated plan declares the task; no run task record is present")
		b.unknown("run-task", planned.ID, "planned task has no validated observed task result")
	}
	if records.CandidateState == projectrun.KnowledgeRecordPresent {
		candidateID := "record/candidate/" + selectedID + "/selected-candidate"
		props := map[string]any{"digest": records.CandidateHash, "bindingStatus": "unknown"}
		if b.scope.WholeProject {
			props["id"] = records.CandidateID
		}
		b.add(candidateID, "Candidate", projectknowledge.FactPartial, props, records.CandidateHash)
		b.edge(planID, candidateID, "hasInitialCandidate", records.PlanDigest, "validated planned candidate exists without a run state")
		b.binding("candidate", selectedID, records.CandidateHash, records.CandidateAPIVersion, plan.BaseModelDigest, plan.BaseRevision, "")
	} else {
		b.unknown("candidate", selectedID, "selected candidate is unavailable")
	}
}

func (b *factBuilder) addRun(records projectrun.KnowledgeRecordSet) {
	p, r := records.Status.Plan, records.Status.Run
	b.runCaptureConsistent[r.ID] = records.CaptureState == projectrun.KnowledgeCaptureConsistent
	modelBinding := b.modelBindingStatus(p.BaseRevision, p.BaseSnapshot, p.BaseModelDigest)
	baseState := "unknown"
	runID, planID := "record/run/"+r.ID, "record/plan/"+p.ID
	b.add(planID, "ExecutionPlan", projectknowledge.FactPartial, map[string]any{"digest": records.PlanDigest, "schema": records.PlanAPIVersion, "modelDigest": p.BaseModelDigest, "sourceSnapshot": p.BaseSnapshot, "baseRevision": p.BaseRevision, "status": p.Status, "modelBindingStatus": modelBinding, "runtimeBindingStatus": string(records.RuntimeBindingState)}, records.PlanDigest)
	runProps := map[string]any{"digest": records.RunDigest, "schema": records.RunAPIVersion, "planId": p.ID, "status": r.Status, "candidateDigest": r.Candidate.Snapshot, "bindingStatus": baseState, "modelBindingStatus": modelBinding, "runtimeDigest": records.RuntimeDigest}
	if b.scope.WholeProject {
		runProps["candidateId"] = r.Candidate.ID
	} else {
		runProps["candidateBindingPresent"] = r.Candidate.ID != ""
	}
	runProps["runtimeBindingStatus"] = string(records.RuntimeBindingState)
	runState := projectknowledge.FactPartial
	if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
		runProps["captureState"] = "unknown"
	} else {
		runProps["captureState"] = "consistent"
	}
	b.add(runID, "ExecutionRun", runState, runProps, records.RunDigest)
	if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
		b.unknown("run-capture", r.ID, "selected operational record links changed or could not be fenced consistently")
	}
	if records.RuntimeBindingState == projectrun.KnowledgeRuntimeBindingNotCompared {
		b.unknown("run-runtime-binding", r.ID, "runtime binding was not compared with a current validated runtime")
	}
	b.edge(runID, planID, "usesPlan", r.Digest, "validated run plan link")
	b.binding("run", r.ID, records.RunDigest, records.RunAPIVersion, r.ModelDigest, r.BaseRevision, records.RuntimeDigest)
	b.binding("plan", p.ID, records.PlanDigest, records.PlanAPIVersion, p.BaseModelDigest, p.BaseRevision, records.RuntimeDigest)
	owned := map[string]bool{}
	for _, task := range p.Managers {
		if b.scope.WholeProject || task.ManagerID == b.scope.ManagerID {
			owned[task.ID] = true
		}
	}
	tasks := map[string]projectrun.ManagerTask{}
	for _, task := range r.Tasks {
		tasks[task.ID] = task
	}
	for _, planned := range p.Managers {
		if !owned[planned.ID] {
			continue
		}
		observed, ok := tasks[planned.ID]
		taskID := "record/task/" + r.ID + "/" + planned.ID
		state := projectknowledge.FactPartial
		if ok && records.CaptureState == projectrun.KnowledgeCaptureConsistent {
			state = "known"
		}
		taskProps := map[string]any{"managerId": planned.ManagerID, "present": ok, "expectedArtifacts": filteredDefinitions(planned.Artifacts, b.defs, b.scope.WholeProject), "expectedChecks": filteredDefinitions(planned.Checks, b.defs, b.scope.WholeProject)}
		if ok {
			taskProps["state"] = observed.State
			taskProps["reportStatus"] = observed.ReportStatus
		}
		statements := []string{}
		for _, statement := range planned.Statements {
			if b.scope.WholeProject || b.defs[statement] {
				statements = append(statements, statement)
			}
		}
		if b.scope.WholeProject || len(statements) > 0 {
			taskProps["expectedStatements"] = statements
		}
		b.add(taskID, "ManagerTask", state, taskProps, p.Digest)
		b.edge(planID, taskID, "plansTask", p.Digest, "validated plan declares the task")
		if ok {
			b.edge(runID, taskID, "hasTask", r.Digest, "validated run contains the observed task result")
			for _, path := range filteredPaths(observed.WrittenPaths, b.paths, b.scope.WholeProject) {
				pid := pathFactID(r.ID, "task-"+planned.ID, path)
				pathState := projectknowledge.FactKnown
				if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
					pathState = projectknowledge.FactPartial
				}
				b.add(pid, "CandidatePath", pathState, map[string]any{"path": path}, r.Digest)
				b.edge(taskID, pid, "wrotePath", r.Digest, "validated task write report")
			}
		} else {
			b.unknown("run-task", planned.ID, "planned task has no validated observed task result")
		}
	}
	ownReviews := []projectrun.ReviewRecord{}
	for _, review := range r.Reviews {
		if b.scope.WholeProject || review.ManagerID == b.scope.ManagerID {
			ownReviews = append(ownReviews, review)
		}
	}
	hasCandidate := records.CandidateState == projectrun.KnowledgeRecordPresent && r.Candidate.ID != "" && (b.scope.WholeProject || len(filteredPaths(mapKeys(r.Candidate.Files), b.paths, false)) > 0 || len(ownReviews) > 0)
	candidateNodeID := ""
	if hasCandidate {
		candidateLabel := "selected-candidate"
		candidateProps := map[string]any{"digest": records.CandidateHash, "schema": records.CandidateAPIVersion, "integrated": r.Candidate.Integrated, "visiblePaths": filteredPaths(mapKeys(r.Candidate.Files), b.paths, b.scope.WholeProject)}
		if b.scope.WholeProject {
			candidateLabel = r.Candidate.ID
			candidateProps["id"] = r.Candidate.ID
		}
		cid := "record/candidate/" + r.ID + "/" + candidateLabel
		candidateNodeID = cid
		candidateState := projectknowledge.FactKnown
		if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
			candidateState = projectknowledge.FactPartial
		}
		b.add(cid, "Candidate", candidateState, candidateProps, records.CandidateHash)
		b.binding("candidate", r.ID, records.CandidateHash, records.CandidateAPIVersion, r.ModelDigest, r.BaseRevision, records.RuntimeDigest)
		b.edge(runID, cid, "producedCandidate", r.Digest, "validated run candidate reference")
		for _, path := range filteredPaths(mapKeys(r.Candidate.Files), b.paths, b.scope.WholeProject) {
			candidatePathIdentity := r.Candidate.ID
			if !b.scope.WholeProject {
				candidatePathIdentity = "selected-candidate"
			}
			pid := pathFactID(r.ID, candidatePathIdentity, path)
			pathState := projectknowledge.FactKnown
			if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
				pathState = projectknowledge.FactPartial
			}
			b.add(pid, "CandidatePath", pathState, map[string]any{"path": path, "digest": r.Candidate.Files[path]}, r.Digest)
			b.edge(cid, pid, "containsPath", r.Digest, "candidate manifest digest")
		}
	}
	checkPlans := map[string]projectrun.CheckPlan{}
	for _, c := range p.Checks {
		checkPlans[c.ID] = c
	}
	verifiedChecks := map[string]projectrun.CheckResult{}
	verificationCurrent := records.CaptureState == projectrun.KnowledgeCaptureConsistent && records.VerificationState == projectrun.KnowledgeRecordPresent && records.Verification != nil
	if verificationCurrent {
		for _, check := range records.Verification.Checks {
			if check.CandidateID == records.CandidateID {
				verifiedChecks[check.ID] = check
			}
		}
	}
	for _, c := range r.Checks {
		cp, exists := checkPlans[c.ID]
		if !b.scope.WholeProject && !ownedTaskManager(p, cp.Owner, b.scope.ManagerID) {
			continue
		}
		id := "record/check/" + r.ID + "/" + c.ID
		verifyStatus := "unknown"
		if verify, ok := verifiedChecks[c.ID]; ok {
			verifyStatus = verify.Outcome
		}
		props := map[string]any{"id": c.ID, "outcome": c.Outcome, "exitCode": c.ExitCode, "expected": exists, "checked": true, "checkPassed": c.Outcome == "passed" && c.ExitCode == 0, "verificationStatus": verifyStatus}
		if b.scope.WholeProject {
			props["candidateId"] = c.CandidateID
		}
		if exists {
			props["required"] = cp.Required
			props["owner"] = cp.Owner
		}
		checkState := projectknowledge.FactKnown
		if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
			checkState = projectknowledge.FactPartial
		}
		b.add(id, "CheckExecution", checkState, props, r.Digest)
		b.edge(runID, id, "hasCheckExecution", r.Digest, "validated durable check result")
		if hasCandidate && c.CandidateID == r.Candidate.ID {
			b.edge(id, candidateNodeID, "checkedCandidate", r.Digest, "check result candidate ID matched the validated run candidate")
		}
	}
	for _, review := range ownReviews {
		reviewID := "record/review/" + r.ID + "/" + review.TaskID + "/" + review.Phase + "/" + fmt.Sprint(review.Round)
		props := map[string]any{"managerId": review.ManagerID, "phase": review.Phase, "round": review.Round, "outcome": review.Outcome, "candidateDigest": review.CandidateDigest, "scopeDigest": review.ScopeDigest, "inputDigest": review.InputDigest, "candidateBindingStatus": "validated-digest-binding"}
		if b.scope.WholeProject {
			props["candidateId"] = review.CandidateID
		}
		reviewState := projectknowledge.FactKnown
		if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
			reviewState = projectknowledge.FactPartial
		}
		b.add(reviewID, "CandidateReview", reviewState, props, r.Digest)
		b.edge(runID, reviewID, "hasReview", r.Digest, "validated durable review record; findings omitted")
		taskID := "record/task/" + r.ID + "/" + review.TaskID
		if owned[review.TaskID] {
			b.edge(taskID, reviewID, "reviewedBy", r.Digest, "validated review task identity")
		}
		if hasCandidate && review.CandidateID == r.Candidate.ID {
			b.edge(reviewID, candidateNodeID, "reviewedCandidate", r.Digest, "review candidate ID matched validated run candidate")
		}
	}
	if records.CandidateState == projectrun.KnowledgeRecordMissing || records.CandidateState == projectrun.KnowledgeRecordUnknown {
		b.unknown("candidate", r.ID, "selected candidate is unavailable")
	}
	verificationStatus := "unknown"
	verified := false
	if records.VerificationState == projectrun.KnowledgeRecordPresent && records.Verification != nil && records.CaptureState == projectrun.KnowledgeCaptureConsistent {
		verificationStatus = records.Verification.Status
		verified = verificationStatus == "verified"
	}
	if hasCandidate {
		verifyID := "record/verification/" + r.ID + "/selected"
		verifyState := projectknowledge.FactPartial
		verifyProps := map[string]any{"verificationStatus": verificationStatus, "candidateDigest": records.CandidateHash}
		if records.VerificationState == projectrun.KnowledgeRecordPresent && records.Verification != nil {
			verifyProps["schema"] = records.VerificationAPIVersion
			verifyProps["digest"] = records.VerificationDigest
			verifyProps["verified"] = verified
			if records.CaptureState != projectrun.KnowledgeCaptureConsistent {
				verifyProps["verificationStatus"] = "unknown"
				verifyProps["verified"] = false
			} else {
				verifyState = projectknowledge.FactKnown
			}
			b.binding("verification", r.ID, records.VerificationDigest, records.VerificationAPIVersion, r.ModelDigest, r.BaseRevision, records.RuntimeDigest)
		} else {
			b.unknown("verification", r.ID, "selected candidate has no validated verification record")
		}
		b.add(verifyID, "CandidateVerification", verifyState, verifyProps, records.VerificationDigest)
		b.edge(candidateNodeID, verifyID, "hasVerification", records.CandidateHash, "validated candidate and verification records")
	}
	if records.ApplyState == projectrun.KnowledgeRecordPresent && records.Apply != nil {
		applyID := "record/apply/" + r.ID + "/selected"
		applyCurrent := records.CaptureState == projectrun.KnowledgeCaptureConsistent && records.Apply.Status == projectrun.StatusApplied
		applyProps := map[string]any{"schema": records.ApplyAPIVersion, "contentDigest": records.ApplyContentDigest, "digestKind": "derived-read-fence-only", "recordedStatus": records.Apply.Status, "appliedStatus": "unknown", "applied": false, "authenticated": false, "appliedAt": records.Apply.AppliedAt, "writtenPaths": filteredPaths(records.Apply.Written, b.paths, b.scope.WholeProject)}
		applyState := projectknowledge.FactPartial
		if applyCurrent {
			applyProps["appliedStatus"] = "applied"
			applyProps["applied"] = true
			applyState = projectknowledge.FactKnown
		}
		if b.scope.WholeProject {
			applyProps["candidateId"] = records.Apply.CandidateID
		}
		b.add(applyID, "ApplyReceipt", applyState, applyProps, records.ApplyContentDigest)
		b.edge(runID, applyID, "hasApplyReceipt", records.RunDigest, "validated selected Apply receipt")
		if hasCandidate {
			b.edge(candidateNodeID, applyID, "appliedBy", records.ApplyContentDigest, "Apply receipt candidate binding validated")
		}
		b.out.SourceBindings = append(b.out.SourceBindings, SourceBinding{Kind: "apply", RecordID: r.ID, Digest: records.ApplyContentDigest, Schema: records.ApplyAPIVersion, Source: "live-operational-record", ModelDigest: r.ModelDigest, Revision: r.BaseRevision, RuntimeDigest: records.RuntimeDigest, DigestKind: "derived-read-fence-only"})
	} else if r.Status == projectrun.StatusApplied {
		b.unknown("apply", r.ID, "run status reports applied but no validated Apply receipt is available")
	}
	for _, c := range p.Checks {
		if !b.scope.WholeProject && !ownedTaskManager(p, c.Owner, b.scope.ManagerID) {
			continue
		}
		found := false
		for _, result := range r.Checks {
			if result.ID == c.ID {
				found = true
				break
			}
		}
		if found {
			continue
		}
		id := "record/check/" + r.ID + "/" + c.ID
		b.add(id, "CheckExecution", projectknowledge.FactPartial, map[string]any{"id": c.ID, "expected": true, "checked": false, "verificationStatus": "unknown", "required": c.Required, "owner": c.Owner}, p.Digest)
		b.edge(runID, id, "expectsCheck", p.Digest, "validated declared check plan")
	}
}

func (b *factBuilder) addExploration(r projectexplore.Record) {
	id := "record/exploration/" + r.ID
	visible := func(scopeID string) bool {
		if b.scope.WholeProject {
			return true
		}
		for _, s := range r.Scopes {
			if s.ID == scopeID {
				for _, m := range s.ManagerIDs {
					if m == b.scope.ManagerID {
						return true
					}
				}
			}
		}
		return false
	}
	b.binding("exploration", r.ID, r.Digest, r.APIVersion, "", "", "")
	b.add(id, "Exploration", projectknowledge.FactPartial, map[string]any{"digest": r.Digest, "status": r.Status, "createdAgainstBindingDigest": r.CreatedAgainst, "bindingStatus": "unknown"}, r.Digest)
	b.unknown("exploration-binding", r.ID, "creation binding cannot be compared with a current validated readiness basis")
	for _, s := range r.Scopes {
		if !visible(s.ID) {
			continue
		}
		sid := id + "/scope/" + s.ID
		managerIDs := append([]string(nil), s.ManagerIDs...)
		if !b.scope.WholeProject {
			managerIDs = []string{b.scope.ManagerID}
		}
		b.add(sid, "WorkScope", "known", map[string]any{"id": s.ID, "operation": s.Operation, "managerIds": managerIDs}, r.Digest)
		b.edge(id, sid, "containsScope", r.Digest, "validated exploration record")
		for _, d := range r.Decisions {
			if !contains(d.ScopeIDs, s.ID) {
				continue
			}
			did := sid + "/decision/" + d.ID
			b.add(did, "WorkDecision", "known", map[string]any{"status": d.Status, "blocking": d.Blocking, "answer": d.Answer, "reason": d.Reason, "callerAccepted": d.Status == "answered" || d.Status == "deferred", "authenticated": false}, r.Digest)
			b.edge(sid, did, "hasDecision", r.Digest, "validated scope link")
		}
		for _, a := range r.Acknowledgements {
			if a.ScopeID != s.ID {
				continue
			}
			aid := sid + "/acknowledgement/" + a.StructureDigest
			b.add(aid, "StructureAcknowledgement", "known", map[string]any{"bindingDigest": a.BindingDigest, "structureDigest": a.StructureDigest, "actor": a.Actor, "authority": a.Authority, "provenance": a.Provenance, "callerAccepted": true, "authenticated": false}, r.Digest)
			b.edge(sid, aid, "hasAcknowledgement", r.Digest, "validated acknowledgement scope")
		}
		for _, c := range r.Completions {
			if c.ScopeID != s.ID {
				continue
			}
			cid := sid + "/apply/" + c.ApplyID
			props := map[string]any{"recordedStatus": c.Status, "runId": c.RunID, "planId": c.PlanID, "planDigest": c.PlanDigest, "candidateDigest": c.CandidateDigest, "recordedVerificationStatus": c.VerificationStatus, "verificationDigest": c.VerificationDigest, "applyId": c.ApplyID, "applyDigest": c.ApplyDigest, "appliedAt": c.AppliedAt, "verificationStatus": "unknown", "applyStatus": "unknown"}
			if b.scope.WholeProject {
				props["candidateId"] = c.CandidateID
				props["verificationId"] = c.VerificationID
			}
			b.add(cid, "ApplyReceipt", "known", props, r.Digest)
			b.edge(sid, cid, "completedBy", r.Digest, "validated completion receipt")
		}
	}
}

func (b *factBuilder) addBrownfield(s projectadoption.BrownfieldSession) {
	id := "record/brownfield/" + s.ID
	// Session-level source paths and discovery body are intentionally omitted.
	targetStatus := "stale"
	if s.Target.Revision == b.projectRevision && s.Target.ProjectDigest == b.projectDigest && s.Target.ModelDigest == b.modelDigest {
		targetStatus = "current"
	}
	b.add(id, "BrownfieldSession", projectknowledge.FactPartial, map[string]any{"digest": s.Digest, "schema": s.APIVersion, "targetRevision": s.Target.Revision, "targetProjectDigest": s.Target.ProjectDigest, "targetModelDigest": s.Target.ModelDigest, "targetBindingStatus": targetStatus, "sourceBindingStatus": "unknown", "sourceCommit": s.Source.Commit, "sourceDigest": s.Source.Digest}, s.Digest)
	b.unknown("brownfield-source-binding", s.ID, "discovery source binding cannot be compared with a current validated source snapshot")
	b.binding("brownfield-session", s.ID, s.Digest, s.APIVersion, s.Target.ModelDigest, s.Target.Revision, "")
	ownedScopes := map[string]bool{}
	if b.scope.WholeProject {
		for _, scope := range s.Scopes {
			ownedScopes[scope.ScopeID] = true
		}
	} else {
		for _, it := range s.Iterations {
			if it.ManagerID == b.scope.ManagerID && it.Proposal != nil {
				for _, proposed := range it.Proposal.Report.Scopes {
					ownedScopes[proposed.ID] = true
				}
			}
		}
	}
	for _, scope := range s.Scopes {
		if !ownedScopes[scope.ScopeID] {
			continue
		}
		sid := id + "/scope/" + scope.ScopeID
		b.add(sid, "BrownfieldScope", "known", map[string]any{"status": scope.Status, "reason": scope.Reason}, s.Digest)
		b.edge(id, sid, "containsScope", s.Digest, "validated session status")
	}
	for _, it := range s.Iterations {
		if !b.scope.WholeProject && it.ManagerID != b.scope.ManagerID {
			continue
		}
		iid := id + "/iteration/" + it.ID
		b.add(iid, "BrownfieldIteration", "known", map[string]any{"managerId": it.ManagerID, "purposeRecorded": it.Purpose != "", "reviewRecorded": it.Review != "", "targetContextDigest": it.TargetContextDigest, "proposalPresent": it.Proposal != nil, "integrationPresent": it.Integration != nil, "resolutionPresent": it.Resolution != nil}, s.Digest)
		b.edge(id, iid, "hasIteration", s.Digest, "validated session iteration")
		if it.Proposal == nil {
			continue
		}
		for _, claim := range it.Proposal.Report.Claims {
			cid := iid + "/claim/" + claim.ID
			b.add(cid, "BrownfieldClaim", "known", map[string]any{"kind": claim.Kind, "method": claim.Method, "statement": claim.Statement, "uncertainty": claim.Uncertainty, "scopeId": claim.ScopeID}, it.Proposal.Digest)
			b.edge(iid, cid, "proposedClaim", it.Proposal.Digest, "validated proposal claim")
			for n, ref := range claim.Evidence {
				eid := iid + "/evidence/" + ref.EvidenceID + fmt.Sprintf("/%d", n)
				props := map[string]any{"evidenceId": ref.EvidenceID, "startLine": ref.StartLine, "endLine": ref.EndLine}
				if b.scope.WholeProject || b.paths[discoveryPath(s, ref.EvidenceID)] {
					props["path"] = discoveryPath(s, ref.EvidenceID)
				}
				b.add(eid, "BrownfieldEvidenceRef", "known", props, it.Proposal.Digest)
				b.edge(cid, eid, "citesEvidence", it.Proposal.Digest, "validated claim evidence link")
			}
		}
		for _, c := range it.Proposal.Report.Contradictions {
			cid := iid + "/contradiction/" + c.ID
			b.add(cid, "BrownfieldContradiction", "known", map[string]any{"scopeId": c.ScopeID, "description": c.Description, "claimIds": c.ClaimIDs, "questionId": c.QuestionID}, it.Proposal.Digest)
			b.edge(iid, cid, "recordsContradiction", it.Proposal.Digest, "caller-supplied validated contradiction")
		}
		for _, q := range it.Proposal.Report.Questions {
			qid := iid + "/question/" + q.ID
			b.add(qid, "BrownfieldQuestion", "known", map[string]any{"scopeId": q.ScopeID, "blocking": q.Blocking, "alternativeCount": len(q.Alternatives), "claimIds": q.ClaimIDs}, it.Proposal.Digest)
			b.edge(iid, qid, "recordsQuestion", it.Proposal.Digest, "validated proposal question; prompt omitted")
		}
		if it.Resolution != nil {
			rid := iid + "/resolution/" + it.Resolution.Digest
			b.add(rid, "BrownfieldResolution", "known", map[string]any{"digest": it.Resolution.Digest, "actor": it.Resolution.Actor, "authorityClaim": it.Resolution.AuthorityClaim, "callerAccepted": true, "authenticated": false, "decisionReferencePresent": it.Resolution.DecisionReference != ""}, it.Resolution.Digest)
			b.edge(iid, rid, "hasResolution", it.Resolution.Digest, "validated caller-submitted resolution; no authentication claim")
		}
		b.out.SourceBindings = append(b.out.SourceBindings, SourceBinding{Kind: "brownfield-iteration", RecordID: s.ID + "/" + it.ID, Digest: iterationDigest(it), Schema: s.APIVersion, Source: "live-operational-record", ModelDigest: s.Target.ModelDigest, Revision: s.Target.Revision})
	}
}

func (b *factBuilder) addHistory(briefings []projectbriefing.Briefing, events []projectbriefing.Event, digest string, store projectbriefing.Store) {
	root := "record/history/" + b.scope.ManagerID
	b.out.SourceBindings = append(b.out.SourceBindings, SourceBinding{Kind: "briefing-history", RecordID: b.scope.ManagerID, Digest: digest, Schema: projectbriefing.APIVersion, Source: "selected-model-history", ModelDigest: b.modelDigest, Revision: b.projectRevision})
	eventIDs := map[string]bool{}
	for _, event := range events {
		if !b.scope.WholeProject && !contains(event.AffectedManagers, b.scope.ManagerID) {
			continue
		}
		eventIDs[event.ID] = true
	}
	for _, brief := range briefings {
		id := root + "/briefing/" + brief.ID
		visibleEventIDs := []string{}
		for _, eventID := range brief.EventIDs {
			if eventIDs[eventID] {
				visibleEventIDs = append(visibleEventIDs, eventID)
				b.edge(id, root+"/event/"+eventID, "containsEvent", digest, "validated briefing event ID matched an event admitted to this selected Manager history")
			}
		}
		b.add(id, "Briefing", "known", map[string]any{"revision": brief.Revision, "modelDigest": brief.ModelDigest, "eventIds": visibleEventIDs, "summary": brief.Summary}, digest)
	}
	for _, event := range events {
		if !eventIDs[event.ID] {
			continue
		}
		id := root + "/event/" + event.ID
		resolution := projectbriefing.EventResolutionStatus(store, event.ID)
		resolutionStatus := resolution.Status
		if resolutionStatus != "resolved" {
			resolutionStatus = "unresolved"
		}
		props := map[string]any{"digest": event.Digest, "change": event.Change, "category": event.Category, "severity": event.Severity, "decisionReference": event.Provenance.DecisionReference, "actor": event.Provenance.Actor, "authority": event.Provenance.Authority, "authenticated": false, "resolutionStatus": resolutionStatus}
		if b.scope.WholeProject || b.defs[event.DefinitionID.Key()] {
			props["definitionId"] = event.DefinitionID.Key()
		}
		b.add(id, "ModelHistoryEvent", "known", props, digest)
		definitionID := event.DefinitionID.Key()
		if b.currentDefs[definitionID] && (b.scope.WholeProject || b.defs[definitionID]) {
			b.edge(id, definitionID, "changedDefinition", event.Digest, "validated history identity matched a current source-model definition admitted to scope")
		}
		if resolution.Status == "resolved" && resolution.Resolution != nil {
			r := *resolution.Resolution
			rid := id + "/resolution"
			props := map[string]any{"digest": r.Digest, "modelRevision": r.ModelRevision, "modelDigest": r.ModelDigest, "planDigest": r.Evidence.PlanDigest, "candidateDigest": r.Evidence.CandidateDigest, "verificationDigest": r.Evidence.VerificationDigest, "applyDigest": r.Evidence.ApplyDigest, "recordedFullVerifyPassed": r.Evidence.FullVerifyPassed, "authenticated": false}
			selectedRun := b.selectedRunResolutionEvidence(r.Evidence)
			if b.scope.WholeProject {
				props["runId"] = r.Evidence.RunID
			} else if selectedRun {
				props["runId"] = r.Evidence.RunID
			}
			if b.scope.WholeProject {
				props["candidateId"] = r.Evidence.CandidateID
				props["coveredManagerIds"] = r.Evidence.CoveredManagerIDs
			}
			b.add(rid, "BriefingResolution", "known", props, r.Digest)
			b.edge(id, rid, "hasResolution", r.Digest, "validated briefing resolution; claims preserved without authentication")
			if selectedRun {
				b.edge(rid, "record/run/"+r.Evidence.RunID, "referencesRun", r.Digest, "resolution run, plan, candidate, verification, and derived Apply-content digests matched explicitly selected validated records; Apply remains unauthenticated")
			} else if r.Evidence.RunID != "" {
				b.unknown("briefing-run-binding", r.Evidence.RunID, "resolution run evidence does not match explicitly selected validated records")
			}
		}
	}
}

func (b *factBuilder) selectedRunResolutionEvidence(e projectbriefing.VerifiedResolutionEvidence) bool {
	if e.RunID == "" || !b.runCaptureConsistent[e.RunID] || e.PlanDigest == "" || e.CandidateDigest == "" || e.VerificationDigest == "" || e.ApplyDigest == "" {
		return false
	}
	matched := map[string]bool{}
	for _, binding := range b.out.SourceBindings {
		if binding.RecordID != e.RunID {
			continue
		}
		switch binding.Kind {
		case "run":
			matched["run"] = binding.Digest != ""
		case "plan":
			matched["plan"] = binding.Digest == e.PlanDigest
		case "candidate":
			matched["candidate"] = binding.Digest == e.CandidateDigest
		case "verification":
			matched["verification"] = binding.Digest == e.VerificationDigest
		case "apply":
			matched["apply"] = binding.Digest == e.ApplyDigest
		}
	}
	return matched["run"] && matched["plan"] && matched["candidate"] && matched["verification"] && matched["apply"]
}

func stateFor(s string) string {
	if s == "unknown" {
		return projectknowledge.FactUnknown
	}
	if s == "partial" {
		return projectknowledge.FactPartial
	}
	return projectknowledge.FactKnown
}
func stringSet(values []string) map[string]bool {
	out := map[string]bool{}
	for _, v := range values {
		out[v] = true
	}
	return out
}
func sortedUnique(values []string) []string {
	out := append([]string{}, values...)
	sort.Strings(out)
	n := 0
	for _, v := range out {
		if n == 0 || out[n-1] != v {
			out[n] = v
			n++
		}
	}
	return out[:n]
}
func filteredPaths(paths []string, visible map[string]bool, whole bool) []string {
	out := []string{}
	for _, p := range paths {
		if whole || visible[p] {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	return out
}
func filteredDefinitions(ids []string, visible map[string]bool, whole bool) []string {
	out := []string{}
	for _, id := range ids {
		if whole || visible[id] {
			out = append(out, id)
		}
	}
	sort.Strings(out)
	return out
}
func mapKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
func pathFactID(run, candidate, p string) string {
	h := sha256.Sum256([]byte(run + "\x00" + candidate + "\x00" + p))
	return "record/path/" + hex.EncodeToString(h[:])
}
func contains(values []string, value string) bool {
	for _, v := range values {
		if v == value {
			return true
		}
	}
	return false
}
func ownedTaskManager(p projectrun.PlanRecord, owner, managerID string) bool {
	for _, t := range p.Managers {
		if t.ManagerID == managerID && (owner == t.ManagerID || owner == t.ID || contains(t.Checks, owner)) {
			return true
		}
	}
	return false
}
func isMissing(err error) bool {
	s := strings.ToLower(err.Error())
	return strings.Contains(s, "does not exist") || strings.Contains(s, "not found") || strings.Contains(s, "no such file") || strings.Contains(s, "cannot find the path")
}
func digestJSON(v any) (string, error) {
	b, e := json.Marshal(v)
	if e != nil {
		return "", e
	}
	h := sha256.Sum256(b)
	return "sha256:" + hex.EncodeToString(h[:]), nil
}

func (b *factBuilder) explorationVisible(r projectexplore.Record) bool {
	if b.scope.WholeProject {
		return true
	}
	for _, scope := range r.Scopes {
		for _, manager := range scope.ManagerIDs {
			if manager == b.scope.ManagerID {
				return true
			}
		}
	}
	return false
}
func (b *factBuilder) brownfieldVisible(s projectadoption.BrownfieldSession) bool {
	if b.scope.WholeProject {
		return true
	}
	for _, iteration := range s.Iterations {
		if iteration.ManagerID == b.scope.ManagerID {
			return true
		}
	}
	return false
}
func discoveryPath(s projectadoption.BrownfieldSession, evidenceID string) string {
	for _, e := range s.Source.Evidence {
		if e.ID == evidenceID {
			return e.Path
		}
	}
	return ""
}
func iterationDigest(v projectadoption.ReverseIteration) string {
	d, _ := digestJSON(v)
	return d
}
