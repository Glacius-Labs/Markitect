package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"errors"
	"fmt"
	"time"

	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

// planSemantics is the checked result of applying method policy to one fixed
// plan basis. It is not durable until PlanExpecting binds readiness and digest.
type planSemantics struct {
	ID               string
	FinalProject     *projectvalue.Project
	EditPlan         *projectvalue.EditPlan
	InitialCandidate candidateData
	BriefingDigests  map[string]string
	Managers         []ManagerTask
	Strictness       map[string]projectconfig.StrictnessProfile
	Checks           []CheckPlan
	Findings         []string
	Blockers         []string
	RuntimeDigest    string
}

func preparePlanSemantics(host Host, basis planBasis) (planSemantics, error) {
	result := planSemantics{}
	request, project, working := basis.Request, basis.Project, basis.Working
	runtime := basis.Runtime
	finalProject := project
	var editPlan *projectvalue.EditPlan
	initialCandidate, err := emptyCandidate()
	if err != nil {
		return result, err
	}
	if request.ModelEdit != nil {
		if err := validateModelEditPaths(*request.ModelEdit); err != nil {
			return result, err
		}
		candidate, editErr := host.PlanEdit(project, *request.ModelEdit)
		if editErr != nil {
			return result, fmt.Errorf("validate requested model edit: %w", editErr)
		}
		if candidate.BaseDigest == "" || candidate.CandidateDigest == "" || !startableReport(candidate.Report) || hasErrorFinding(candidate.Report.Findings) {
			return result, fmt.Errorf("validated model edit is missing fixed digests or has structural findings")
		}
		editPlan = &candidate
		initialCandidate, err = modelEditCandidate(editPlan, project.Snapshot)
		if err != nil {
			return result, err
		}
		candidateSnapshot, snapshotErr := snapshotWithCandidate(project.Snapshot, initialCandidate)
		if snapshotErr != nil {
			return result, snapshotErr
		}
		validated, validationErr := host.FromSnapshot(basis.Root, candidateSnapshot)
		if validationErr != nil {
			return result, fmt.Errorf("recompile model edit candidate: %w", validationErr)
		}
		if validated == nil || validated.Snapshot == nil || validated.Report.Digest != candidate.Report.Digest {
			return result, fmt.Errorf("model edit candidate did not reproduce its planned report")
		}
		finalProject = validated
	}
	if !startableReport(finalProject.Report) || hasErrorFinding(finalProject.Report.Findings) {
		return result, fmt.Errorf("candidate model analysis did not succeed")
	}
	fullCoverage := finalProject.Config.CoverageMode == "full"
	if fullCoverage && request.ModelEdit != nil {
		return result, fmt.Errorf("full coverage requires accepting and committing the model separately before implementation; draft model edits are not an accepted execution basis")
	}
	if fullCoverage {
		for _, candidateBasis := range []*projectvalue.Project{project, working, finalProject} {
			if candidateBasis.Coverage == nil || !candidateBasis.Coverage.Accounted {
				return result, fmt.Errorf("whole-repository coverage is unaccounted; classify unknown paths before planning implementation")
			}
		}
	}
	if basis.Operation != OperationApply && !fullCoverage {
		return result, fmt.Errorf("%s requires project coverageMode full", basis.Operation)
	}
	briefingDigests := map[string]string{}
	if request.ModelEdit == nil {
		briefingDigests, err = briefingBindingsIn(host, basis.Root, basis.BriefingView, project)
		if err != nil {
			return result, err
		}
	} else {
		// A proposed, not-yet-accepted model has no valid accepted briefing
		// history. Keep the draft plan explicitly unbound until it is committed.
		for _, manager := range finalProject.Report.Managers {
			briefingDigests[manager.ID] = ""
		}
	}
	if basis.ChangeBase != nil && fullCoverage && request.ModelEdit == nil && basis.ChangeBase.Report.ModelDigest != project.Report.ModelDigest {
		for _, manager := range project.Report.Managers {
			if briefingDigests[manager.ID] == "" {
				return result, fmt.Errorf("full coverage after a semantic model change requires an accepted briefing for Manager %s", manager.ID)
			}
		}
	}
	if err := validateStrictnessManagers(runtime.Strictness, finalProject.Report.Managers); err != nil {
		return result, err
	}
	id, err := newID()
	if err != nil {
		return result, err
	}
	managerTasks, selected, findings, err := planManagers(finalProject.Report, finalProject.Snapshot.Files, request, editPlan, basis.ChangeImpact, runtime.Limits)
	if err != nil {
		return result, err
	}
	strictness := make(map[string]projectconfig.StrictnessProfile, len(managerTasks))
	profileManagers := make([]string, 0, len(managerTasks))
	for _, task := range managerTasks {
		profileManagers = append(profileManagers, task.ManagerID)
	}
	if fullCoverage {
		for _, manager := range finalProject.Report.Managers {
			profileManagers = append(profileManagers, manager.ID)
		}
	}
	for _, managerID := range uniqueSorted(profileManagers) {
		profile, profileErr := projectconfig.ResolveStrictness(runtime, managerID)
		if profileErr != nil {
			return result, fmt.Errorf("resolve strictness for Manager %s: %w", managerID, profileErr)
		}
		strictness[managerID] = profile
	}
	if len(managerTasks) == 0 {
		return result, fmt.Errorf("plan has no responsible Manager")
	}
	if err := checkDepth(managerTasks, runtime.Limits.MaxDepth); err != nil {
		return result, err
	}
	var forcedChecks []string
	if basis.ChangeImpact != nil {
		forcedChecks = basis.ChangeImpact.Checks
	}
	checkSelection := selected
	if fullCoverage {
		checkSelection = make(map[string]bool, len(finalProject.Report.Managers))
		for _, manager := range finalProject.Report.Managers {
			checkSelection[manager.ID] = true
		}
	}
	checkPlans, checkFindings := planChecksWithImpact(finalProject.Report, checkSelection, forcedChecks)
	findings = append(findings, checkFindings...)
	checkPlans, executableFindings, executableErr := bindCheckExecutables(checkPlans, runtime)
	if executableErr != nil {
		return result, executableErr
	}
	findings = append(findings, executableFindings...)
	checkFindings = append(checkFindings, executableFindings...)
	minimumStarts := len(managerTasks) + len(checkPlans)
	for _, task := range managerTasks {
		if len(activeChildren(managerTasks, task.ManagerID)) > 0 {
			minimumStarts++
		}
	}
	if fullCoverage {
		minimumStarts += len(finalProject.Report.Managers)
		if runtime.Review == nil {
			return result, fmt.Errorf("full coverage requires a configured reviewer for every Manager")
		}
		for _, manager := range finalProject.Report.Managers {
			if _, ok := runtime.Review.Agents[manager.ID]; !ok {
				return result, fmt.Errorf("full coverage has no configured reviewer for Manager %s", manager.ID)
			}
		}
	}
	if runtime.Verifier != nil {
		minimumStarts++
	}
	if runtime.Review != nil {
		minimumStarts += reviewPhaseStarts(finalProject, managerTasks)
	}
	if minimumStarts > runtime.Limits.MaxStarts {
		return result, fmt.Errorf("plan requires at least %d manager, check, and verifier starts; configured limit is %d", minimumStarts, runtime.Limits.MaxStarts)
	}
	for managerID, agent := range runtime.Agents {
		if _, configErr := agent.AgentConfig(); configErr != nil {
			return result, fmt.Errorf("invalid runtime agent %s: %w", managerID, configErr)
		}
	}
	for _, task := range managerTasks {
		if _, ok := runtime.Agents[task.ManagerID]; !ok {
			return result, fmt.Errorf("runtime agent for active Manager %s is not configured", task.ManagerID)
		}
	}
	if runtime.Verifier != nil {
		if _, configErr := runtime.Verifier.AgentConfig(); configErr != nil {
			return result, fmt.Errorf("invalid runtime verifier: %w", configErr)
		}
	}
	if runtime.Review != nil {
		for managerID, agent := range runtime.Review.Agents {
			if _, configErr := agent.AgentConfig(); configErr != nil {
				return result, fmt.Errorf("invalid runtime reviewer %s: %w", managerID, configErr)
			}
		}
		for _, task := range managerTasks {
			if _, ok := runtime.Review.Agents[task.ManagerID]; !ok {
				return result, fmt.Errorf("runtime reviewer for active Manager %s is not configured", task.ManagerID)
			}
		}
	}
	if fullCoverage {
		for _, manager := range finalProject.Report.Managers {
			if _, ok := runtime.Review.Agents[manager.ID]; !ok {
				return result, fmt.Errorf("runtime reviewer for full-coverage Manager %s is not configured", manager.ID)
			}
		}
	}
	runtimeDigest, err := methodPolicyDigest(runtime)
	if err != nil {
		return result, err
	}
	return planSemantics{
		ID: id, FinalProject: finalProject, EditPlan: editPlan, InitialCandidate: initialCandidate,
		BriefingDigests: briefingDigests, Managers: managerTasks, Strictness: strictness,
		Checks: checkPlans, Findings: findings, Blockers: checkFindings, RuntimeDigest: runtimeDigest,
	}, nil
}

func planFromSemantics(root string, basis planBasis, semantics planSemantics) PlanRecord {
	project, working, finalProject := basis.Project, basis.Working, semantics.FinalProject
	plan := PlanRecord{
		APIVersion: APIVersion, ID: semantics.ID, Status: StatusPlanned, Operation: basis.Operation, Goal: basis.Request.Goal,
		ExecuteAuthorized: basis.Request.ExecuteAuthorized, Root: root,
		BaseRevision: project.Revision, TargetBranch: basis.TargetBranch, TargetHead: basis.TargetHead, BaseSnapshot: project.Snapshot.Digest(),
		WorkingSnapshot:  working.Snapshot.Digest(),
		RepositoryDigest: basis.RepositoryDigest, BaseProjectDigest: project.Digest, WorkingProjectDigest: working.Digest,
		BaseModelDigest: project.Report.ModelDigest,
		ModelDigest:     finalProject.Report.ModelDigest, ReportDigest: finalProject.Report.Digest,
		RuntimeDigest: semantics.RuntimeDigest, RuntimeBindingVersion: methodRuntimeBindingVersion, PlannedAt: time.Now().UTC(),
		Managers: semantics.Managers, Strictness: semantics.Strictness, BriefingDigests: semantics.BriefingDigests,
		Checks: semantics.Checks, ModelEdit: semantics.EditPlan, InitialCandidateID: semantics.InitialCandidate.ID,
		Findings: uniqueSorted(semantics.Findings), Blockers: uniqueSorted(semantics.Blockers),
	}
	if basis.ChangeBase != nil {
		plan.ChangeBaseRevision = basis.ChangeBase.Revision
		plan.ChangeBaseSnapshot = basis.ChangeBase.Snapshot.Digest()
		plan.ChangeBaseProjectDigest = basis.ChangeBase.Digest
		plan.ChangeBaseModelDigest = basis.ChangeBase.Report.ModelDigest
		plan.ChangeBaseReportDigest = basis.ChangeBase.Report.Digest
		plan.ChangeImpact = basis.ChangeImpact
		plan.ChangeImpactDigest = basis.ChangeImpact.Digest
	}
	return plan
}

func persistAuthorizedPlan(root, id string, plan PlanRecord, initialCandidate candidateData) error {
	store, err := newRunStore(root)
	if err != nil {
		return err
	}
	unlock, err := store.lock()
	if err != nil {
		return err
	}
	return persistPlanWithLease(store, id, plan, initialCandidate, unlock)
}

func persistPlanWithLease(store *runStore, id string, plan PlanRecord, initialCandidate candidateData, release func() error) (resultErr error) {
	defer func() { resultErr = errors.Join(resultErr, release()) }()
	dir, err := store.createRun(id)
	if err != nil {
		return err
	}
	if err := store.writeCandidate(dir, initialCandidate); err != nil {
		return err
	}
	return store.writePlan(plan)
}
