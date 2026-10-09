package projectrun

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// Plan binds a requested project operation to one fixed model, inventory,
// runner and runtime configuration. It persists no source changes and starts
// no external agent.
func Plan(host Host, root, revision string, request PlanRequest) (PlanRecord, error) {
	var plan PlanRecord
	var err error
	operation, err := NormalizeOperation(request.Operation)
	if err != nil {
		return plan, err
	}
	request.Operation = operation
	if operation != OperationApply && (len(request.Managers) > 0 || strings.TrimSpace(request.SinceRevision) != "" || request.ModelEdit != nil) {
		return plan, fmt.Errorf("%s does not accept manager, since-revision, or model-edit narrowing; prepare and accept any model changes separately", operation)
	}
	if host.Load == nil || host.FromSnapshot == nil || host.PlanEdit == nil {
		return plan, fmt.Errorf("project Host frontend is incomplete")
	}
	if strings.TrimSpace(request.Goal) == "" {
		return plan, fmt.Errorf("a bounded project goal is required")
	}
	if request.BaseRevision != "" {
		revision = request.BaseRevision
	}
	if strings.TrimSpace(revision) == "" {
		revision, err = resolveGitHead(root)
		if err != nil {
			return plan, fmt.Errorf("resolve fixed default base revision: %w", err)
		}
	}
	targetHead, err := resolveGitHead(root)
	if err != nil {
		return plan, fmt.Errorf("capture current target HEAD: %w", err)
	}
	targetBranch, err := resolveGitBranch(root)
	if err != nil {
		return plan, fmt.Errorf("capture current target branch: %w", err)
	}
	project, err := host.Load(root, revision)
	if err != nil {
		return plan, fmt.Errorf("load selected project revision: %w", err)
	}
	if project == nil || project.Snapshot == nil {
		return plan, fmt.Errorf("project frontend returned no fixed snapshot")
	}
	var changeBase *Project
	var changeImpact *projectmodel.ChangeImpact
	if strings.TrimSpace(request.SinceRevision) != "" {
		changeBase, err = host.Load(root, request.SinceRevision)
		if err != nil {
			return plan, fmt.Errorf("load project change baseline %q: %w", request.SinceRevision, err)
		}
		if changeBase == nil || changeBase.Snapshot == nil {
			return plan, fmt.Errorf("project frontend returned no change-baseline snapshot")
		}
		impact := projectmodel.Impact(changeBase.Report, project.Report)
		if impact.Digest == "" || impact.BaseDigest != changeBase.Report.Digest || impact.CandidateDigest != project.Report.Digest {
			return plan, fmt.Errorf("project change impact did not bind both selected reports")
		}
		changeImpact = &impact
	}
	if project.Snapshot.Provisional {
		return plan, fmt.Errorf("project run requires a non-provisional snapshot")
	}
	if (project.Config.WorkflowMode == "guided" || request.ExplorationID != "") && request.ModelEdit == nil {
		if _, err := projectbriefing.EnsureAcceptedHistory(root, project.Revision); err != nil {
			return plan, fmt.Errorf("capture accepted model history: %w", err)
		}
	}
	working, err := host.Load(root, "")
	if err != nil {
		return plan, fmt.Errorf("capture current project working inputs: %w", err)
	}
	if working == nil || working.Snapshot == nil {
		return plan, fmt.Errorf("project runtime requires a selected working-input snapshot")
	}
	if err := requireCleanSelectedBasisAtRevision(root, project.Revision, project.Snapshot, working.Snapshot); err != nil {
		return plan, err
	}
	repository, err := source.IdentifyGit(root)
	if err != nil {
		return plan, fmt.Errorf("identify project repository: %w", err)
	}
	currentHead, headErr := resolveGitHead(root)
	currentBranch, branchErr := resolveGitBranch(root)
	if headErr != nil || branchErr != nil || currentHead != targetHead || currentBranch != targetBranch {
		return plan, ErrStale
	}
	if !startableReport(project.Report) {
		return plan, fmt.Errorf("project model analysis did not succeed: %s", project.Report.Status)
	}
	if hasErrorFinding(project.Report.Findings) {
		return plan, fmt.Errorf("project model has structural error findings")
	}
	runtime, err := LoadRuntime(root)
	if err != nil {
		return plan, err
	}
	finalProject := project
	var editPlan *EditPlan
	initialCandidate, err := emptyCandidate()
	if err != nil {
		return plan, err
	}
	if request.ModelEdit != nil {
		if err := validateModelEditPaths(*request.ModelEdit); err != nil {
			return plan, err
		}
		candidate, editErr := host.PlanEdit(project, *request.ModelEdit)
		if editErr != nil {
			return plan, fmt.Errorf("validate requested model edit: %w", editErr)
		}
		if candidate.BaseDigest == "" || candidate.CandidateDigest == "" || !startableReport(candidate.Report) || hasErrorFinding(candidate.Report.Findings) {
			return plan, fmt.Errorf("validated model edit is missing fixed digests or has structural findings")
		}
		editPlan = &candidate
		initialCandidate, err = modelEditCandidate(editPlan, project.Snapshot)
		if err != nil {
			return plan, err
		}
		candidateSnapshot, snapshotErr := snapshotWithCandidate(project.Snapshot, initialCandidate)
		if snapshotErr != nil {
			return plan, snapshotErr
		}
		validated, validationErr := host.FromSnapshot(root, candidateSnapshot)
		if validationErr != nil {
			return plan, fmt.Errorf("recompile model edit candidate: %w", validationErr)
		}
		if validated == nil || validated.Snapshot == nil || validated.Report.Digest != candidate.Report.Digest {
			return plan, fmt.Errorf("model edit candidate did not reproduce its planned report")
		}
		finalProject = validated
	}
	if !startableReport(finalProject.Report) || hasErrorFinding(finalProject.Report.Findings) {
		return plan, fmt.Errorf("candidate model analysis did not succeed")
	}
	fullCoverage := finalProject.Config.CoverageMode == "full"
	if fullCoverage && request.ModelEdit != nil {
		return plan, fmt.Errorf("full coverage requires accepting and committing the model separately before implementation; draft model edits are not an accepted execution basis")
	}
	if fullCoverage {
		for _, basis := range []*Project{project, working, finalProject} {
			if basis.Coverage == nil || !basis.Coverage.Accounted {
				return plan, fmt.Errorf("whole-repository coverage is unaccounted; classify unknown paths before planning implementation")
			}
		}
	}
	if operation != OperationApply && !fullCoverage {
		return plan, fmt.Errorf("%s requires project coverageMode full", operation)
	}
	briefingDigests := map[string]string{}
	if request.ModelEdit == nil {
		briefingDigests, err = briefingBindings(root, project)
		if err != nil {
			return plan, err
		}
	} else {
		// A proposed, not-yet-accepted model has no valid accepted briefing
		// history. Keep the draft plan explicitly unbound until it is committed.
		for _, manager := range finalProject.Report.Managers {
			briefingDigests[manager.ID] = ""
		}
	}
	if changeBase != nil && fullCoverage && request.ModelEdit == nil && changeBase.Report.ModelDigest != project.Report.ModelDigest {
		for _, manager := range project.Report.Managers {
			if briefingDigests[manager.ID] == "" {
				return plan, fmt.Errorf("full coverage after a semantic model change requires an accepted briefing for Manager %s", manager.ID)
			}
		}
	}
	if err := validateStrictnessManagers(runtime.Strictness, finalProject.Report.Managers); err != nil {
		return plan, err
	}
	id, err := newID()
	if err != nil {
		return plan, err
	}
	managerTasks, selected, findings, err := planManagers(finalProject.Report, finalProject.Snapshot.Files, request, editPlan, changeImpact, runtime.Limits)
	if err != nil {
		return plan, err
	}
	strictness := make(map[string]StrictnessProfile, len(managerTasks))
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
		profile, profileErr := ResolveStrictness(runtime, managerID)
		if profileErr != nil {
			return plan, fmt.Errorf("resolve strictness for Manager %s: %w", managerID, profileErr)
		}
		strictness[managerID] = profile
	}
	if len(managerTasks) == 0 {
		return plan, fmt.Errorf("plan has no responsible Manager")
	}
	if err := checkDepth(managerTasks, runtime.Limits.MaxDepth); err != nil {
		return plan, err
	}
	var forcedChecks []string
	if changeImpact != nil {
		forcedChecks = changeImpact.Checks
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
		return plan, executableErr
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
			return plan, fmt.Errorf("full coverage requires a configured reviewer for every Manager")
		}
		for _, manager := range finalProject.Report.Managers {
			if _, ok := runtime.Review.Agents[manager.ID]; !ok {
				return plan, fmt.Errorf("full coverage has no configured reviewer for Manager %s", manager.ID)
			}
		}
	}
	if runtime.Verifier != nil {
		minimumStarts++
	}
	if runtime.Review != nil {
		phaseCount := 0
		for _, task := range managerTasks {
			if !reviewRequired(finalProject, task) {
				continue
			}
			phaseCount++
			if len(activeChildren(managerTasks, task.ManagerID)) > 0 {
				phaseCount++
			}
		}
		minimumStarts += phaseCount
	}
	if minimumStarts > runtime.Limits.MaxStarts {
		return plan, fmt.Errorf("plan requires at least %d manager, check, and verifier starts; configured limit is %d", minimumStarts, runtime.Limits.MaxStarts)
	}
	fingerprints := map[string]string{}
	for id, agent := range runtime.Agents {
		config, configErr := agent.AgentConfig()
		if configErr != nil {
			return plan, fmt.Errorf("invalid runtime agent %s: %w", id, configErr)
		}
		fingerprint, fingerprintErr := planRuntimeFingerprint(config)
		if fingerprintErr != nil {
			return plan, fmt.Errorf("fingerprint runtime agent %s: %w", id, fingerprintErr)
		}
		fingerprints[id] = fingerprint
	}
	for _, task := range managerTasks {
		if _, ok := fingerprints[task.ManagerID]; !ok {
			return plan, fmt.Errorf("runtime agent for active Manager %s is not configured", task.ManagerID)
		}
	}
	if runtime.Verifier != nil {
		config, configErr := runtime.Verifier.AgentConfig()
		if configErr != nil {
			return plan, fmt.Errorf("invalid runtime verifier: %w", configErr)
		}
		fingerprint, fingerprintErr := planRuntimeFingerprint(config)
		if fingerprintErr != nil {
			return plan, fmt.Errorf("fingerprint runtime verifier: %w", fingerprintErr)
		}
		fingerprints["$verifier"] = fingerprint
	}
	if runtime.Review != nil {
		for managerID, agent := range runtime.Review.Agents {
			config, configErr := agent.AgentConfig()
			if configErr != nil {
				return plan, fmt.Errorf("invalid runtime reviewer %s: %w", managerID, configErr)
			}
			fingerprint, fingerprintErr := planRuntimeFingerprint(config)
			if fingerprintErr != nil {
				return plan, fmt.Errorf("fingerprint runtime reviewer %s: %w", managerID, fingerprintErr)
			}
			fingerprints["$reviewer:"+managerID] = fingerprint
		}
		for _, task := range managerTasks {
			if _, ok := runtime.Review.Agents[task.ManagerID]; !ok {
				return plan, fmt.Errorf("runtime reviewer for active Manager %s is not configured", task.ManagerID)
			}
		}
	}
	if fullCoverage {
		for _, manager := range finalProject.Report.Managers {
			if _, ok := fingerprints["$reviewer:"+manager.ID]; !ok {
				return plan, fmt.Errorf("runtime reviewer for full-coverage Manager %s is not fingerprinted", manager.ID)
			}
		}
	}
	runtimeDigest, err := digest(struct {
		Runtime      Runtime           `json:"runtime"`
		Fingerprints map[string]string `json:"fingerprints"`
	}{Runtime: runtime, Fingerprints: fingerprints})
	if err != nil {
		return plan, err
	}
	plan = PlanRecord{
		APIVersion: APIVersion, ID: id, Status: StatusPlanned, Operation: operation, Goal: request.Goal,
		ExecuteAuthorized: request.ExecuteAuthorized, Root: root,
		BaseRevision: project.Revision, TargetBranch: targetBranch, TargetHead: targetHead, BaseSnapshot: project.Snapshot.Digest(),
		WorkingSnapshot:  working.Snapshot.Digest(),
		RepositoryDigest: repository.Digest, BaseProjectDigest: project.Digest, WorkingProjectDigest: working.Digest,
		BaseModelDigest: project.Report.ModelDigest,
		ModelDigest:     finalProject.Report.ModelDigest, ReportDigest: finalProject.Report.Digest,
		RuntimeDigest: runtimeDigest, PlannedAt: time.Now().UTC(), Managers: managerTasks, Strictness: strictness, BriefingDigests: briefingDigests,
		Checks: checkPlans, ModelEdit: editPlan, InitialCandidateID: initialCandidate.ID,
		RuntimeAgents: fingerprints, Findings: uniqueSorted(findings), Blockers: uniqueSorted(checkFindings),
	}
	if changeBase != nil {
		plan.ChangeBaseRevision = changeBase.Revision
		plan.ChangeBaseSnapshot = changeBase.Snapshot.Digest()
		plan.ChangeBaseProjectDigest = changeBase.Digest
		plan.ChangeBaseModelDigest = changeBase.Report.ModelDigest
		plan.ChangeBaseReportDigest = changeBase.Report.Digest
		plan.ChangeImpact = changeImpact
		plan.ChangeImpactDigest = changeImpact.Digest
	}
	if err := bindExplorationReadiness(root, project, request, &plan); err != nil {
		return plan, err
	}
	plan.Digest, err = planDigest(plan)
	if err != nil {
		return plan, err
	}
	if !request.ExecuteAuthorized {
		return plan, nil
	}
	store, err := newRunStore(root)
	if err != nil {
		return plan, err
	}
	unlock, err := store.lock()
	if err != nil {
		return plan, err
	}
	defer unlock()
	dir, err := store.createRun(id)
	if err != nil {
		return plan, err
	}
	if err := store.writeCandidate(dir, initialCandidate); err != nil {
		return plan, err
	}
	if err := store.writePlan(plan); err != nil {
		return plan, err
	}
	return plan, nil
}

// planRuntimeFingerprint mirrors the default CLI invoker used by Project Run.
// Process agents retain the existing transport-neutral fingerprint contract;
// native agents additionally bind the App Server protocol and stable Host
// helper tool specification advertised by NewTransportInvoker's default
// options. Per-run callbacks and journal paths are intentionally not included.
func planRuntimeFingerprint(config agentexec.Config) (string, error) {
	if config.Transport == TransportCodexAppServer {
		return NewTransportInvoker(codexappserver.Options{}).Fingerprint(config)
	}
	return agentexec.Fingerprint(config)
}

func planManagers(report projectmodel.Report, baseFiles map[string][]byte, request PlanRequest, edit *EditPlan, changeImpact *projectmodel.ChangeImpact, limits Limits) ([]ManagerTask, map[string]bool, []string, error) {
	operation, err := NormalizeOperation(request.Operation)
	if err != nil {
		return nil, nil, nil, err
	}
	if operation != OperationApply && (len(request.Managers) > 0 || changeImpact != nil || edit != nil || request.ModelEdit != nil || strings.TrimSpace(request.SinceRevision) != "") {
		return nil, nil, nil, fmt.Errorf("%s requires full Manager selection without change-impact or model-edit narrowing", operation)
	}
	managerByID := make(map[string]projectmodel.Manager, len(report.Managers))
	children := map[string][]string{}
	for _, manager := range report.Managers {
		managerByID[manager.ID] = manager
		children[manager.Parent] = append(children[manager.Parent], manager.ID)
	}
	for parent := range children {
		sort.Strings(children[parent])
	}
	targets := map[string]bool{}
	for _, id := range request.Managers {
		if _, ok := managerByID[id]; !ok {
			return nil, nil, nil, fmt.Errorf("requested Manager %q was not found", id)
		}
		targets[id] = true
	}
	findings := append([]string(nil), report.Unknown...)
	includeImpactManagers := func(managerIDs []string, source string) bool {
		unknownManager := false
		for _, id := range managerIDs {
			if _, ok := managerByID[id]; !ok {
				unknownManager = true
				findings = append(findings, source+" references a Manager absent from the current model: "+id)
				continue
			}
			targets[id] = true
		}
		return unknownManager
	}
	broadImpact := false
	if edit != nil {
		broadImpact = includeImpactManagers(edit.Impact.Managers, "model edit impact") || broadImpact
		if len(edit.Impact.Unknown) > 0 {
			findings = append(findings, edit.Impact.Unknown...)
			broadImpact = true
		}
	}
	if changeImpact != nil {
		broadImpact = includeImpactManagers(changeImpact.Managers, "since impact") || broadImpact
		if len(changeImpact.Unknown) > 0 {
			findings = append(findings, changeImpact.Unknown...)
			broadImpact = true
		}
	}
	if broadImpact {
		for _, manager := range report.Managers {
			targets[manager.ID] = true
		}
		findings = append(findings, "change impact has unresolved scope; all current Managers are included")
	}
	if operation == OperationCleanup || operation == OperationReconcile {
		for _, manager := range report.Managers {
			targets[manager.ID] = true
		}
	} else if len(targets) == 0 {
		// A natural-language goal does not establish that any Manager is
		// unaffected. Without explicit routing or a model delta, preserve the
		// complete declared responsibility tree.
		for _, manager := range report.Managers {
			targets[manager.ID] = true
		}
	}
	for _, artifact := range report.Artifacts {
		if !artifact.Required {
			continue
		}
		missing := false
		if len(artifact.Paths) == 0 {
			missing = true
			findings = append(findings, "required artifact "+artifact.ID+" has no declared realization path")
		} else {
			for _, expected := range artifact.Paths {
				if !pathSatisfied(expected, baseFiles) {
					missing = true
					findings = append(findings, "required artifact "+artifact.ID+" still requires "+expected)
				}
			}
		}
		if missing && artifact.Owner != "" {
			if _, ok := managerByID[artifact.Owner]; ok {
				targets[artifact.Owner] = true
			}
		}
	}
	if len(report.Unknown) > 0 {
		// Unknown inventory or scope broadens review; it is never converted into
		// an empty impact set.
		for _, manager := range report.Managers {
			targets[manager.ID] = true
		}
	}
	// Typed cross-Manager Statement references are explicit work dependencies.
	// Include each referenced owner and the ancestors needed to route and
	// integrate its work before freezing the selected task tree. Prose does not
	// add edges or broaden this selection.
	queue := make([]string, 0, len(targets))
	for id := range targets {
		queue = append(queue, id)
	}
	sort.Strings(queue)
	statementByID := make(map[string]projectmodel.Statement, len(report.Statements))
	statementsByOwner := make(map[string][]projectmodel.Statement)
	for _, statement := range report.Statements {
		statementByID[statement.ID] = statement
		statementsByOwner[statement.Owner] = append(statementsByOwner[statement.Owner], statement)
	}
	for owner := range statementsByOwner {
		sort.Slice(statementsByOwner[owner], func(i, j int) bool {
			return statementsByOwner[owner][i].ID < statementsByOwner[owner][j].ID
		})
	}
	processedManagers := map[string]bool{}
	for len(queue) > 0 {
		id := queue[0]
		queue = queue[1:]
		if processedManagers[id] {
			continue
		}
		manager, exists := managerByID[id]
		if !exists {
			return nil, nil, nil, fmt.Errorf("selected Manager %q was not found", id)
		}
		processedManagers[id] = true
		if manager.Parent != "" && !targets[manager.Parent] {
			if _, parentExists := managerByID[manager.Parent]; !parentExists {
				return nil, nil, nil, fmt.Errorf("Manager %s has missing parent %s", id, manager.Parent)
			}
			targets[manager.Parent] = true
			queue = append(queue, manager.Parent)
		}
		for _, statement := range statementsByOwner[id] {
			for _, relation := range []struct {
				name string
				ids  []string
			}{{"requires", statement.Requires}, {"uses", statement.Uses}} {
				refs := append([]string(nil), relation.ids...)
				sort.Strings(refs)
				for _, ref := range refs {
					target, found := statementByID[ref]
					if !found {
						return nil, nil, nil, fmt.Errorf("selected Manager %s statement %s %s missing statement %s", id, statement.ID, relation.name, ref)
					}
					if _, ownerExists := managerByID[target.Owner]; !ownerExists {
						return nil, nil, nil, fmt.Errorf("selected Manager %s statement %s %s statement %s with missing owner %s", id, statement.ID, relation.name, ref, target.Owner)
					}
					if !targets[target.Owner] {
						targets[target.Owner] = true
						queue = append(queue, target.Owner)
					}
				}
			}
		}
	}
	selected := map[string]bool{}
	for id := range targets {
		for current := id; current != ""; current = managerByID[current].Parent {
			if selected[current] {
				break
			}
			selected[current] = true
		}
	}
	if len(selected) == 0 {
		return nil, nil, nil, fmt.Errorf("project model contains no runnable managers")
	}
	selectedTasks := make([]ManagerTask, 0, len(selected))
	for id := range selected {
		manager := managerByID[id]
		selectedTasks = append(selectedTasks, ManagerTask{ManagerID: id, ParentTask: manager.Parent})
	}
	if _, err := managerDependencies(report, selectedTasks); err != nil {
		return nil, nil, nil, err
	}
	rootCount := 0
	for _, id := range children[""] {
		if selected[id] {
			rootCount++
		}
	}
	if rootCount != 1 {
		return nil, nil, nil, fmt.Errorf("bounded project execution requires one selected top-level Manager to integrate the task tree")
	}
	ordered := make([]string, 0, len(selected))
	for id := range selected {
		ordered = append(ordered, id)
	}
	sort.Slice(ordered, func(i, j int) bool {
		di, dj := managerDepth(managerByID, ordered[i]), managerDepth(managerByID, ordered[j])
		if di != dj {
			return di < dj
		}
		return ordered[i] < ordered[j]
	})
	tasks := make([]ManagerTask, 0, len(ordered))
	for _, id := range ordered {
		manager := managerByID[id]
		depth := managerDepth(managerByID, id)
		if depth > limits.MaxDepth {
			return nil, nil, nil, fmt.Errorf("Manager %s exceeds configured maximum depth %d", id, limits.MaxDepth)
		}
		task := ManagerTask{ID: id, ManagerID: id, ParentTask: manager.Parent, Depth: depth,
			Goal: request.Goal, Owns: append([]string(nil), manager.Owns...), State: "queued"}
		for _, statement := range report.Statements {
			if statement.Owner == id {
				task.Statements = append(task.Statements, statement.ID)
			}
		}
		for _, artifact := range report.Artifacts {
			if artifact.Owner == id {
				task.Artifacts = append(task.Artifacts, artifact.ID)
			}
		}
		for _, check := range report.Checks {
			if check.Owner == id {
				task.Checks = append(task.Checks, check.ID)
			}
		}
		tasks = append(tasks, task)
	}
	return tasks, selected, findings, nil
}

func requireCleanSelectedBasis(fixed, working *Snapshot) error {
	if fixed == nil || working == nil {
		return fmt.Errorf("fixed and working selected snapshots are required")
	}
	if fixed.Digest() != working.Digest() {
		return selectedBasisChangedError()
	}
	return nil
}

func requireCleanSelectedBasisAtRevision(root, revision string, fixed, working *Snapshot) error {
	if fixed == nil || working == nil {
		return fmt.Errorf("fixed and working selected snapshots are required")
	}
	if fixed.Digest() == working.Digest() {
		return nil
	}
	changes := snapshot.Compare(fixed, working)
	if len(changes.Added) != 0 || len(changes.Removed) != 0 {
		return selectedBasisChangedError()
	}
	// Compare changed bytes through Git's clean filters. On Windows, a normal
	// core.autocrlf checkout has CRLF worktree bytes while the fixed Git tree
	// contains LF; those bytes are the same selected source after Git's normal
	// conversion. Git still reports substantive edits as differences.
	const maxPathspecArgumentBytes = 16 * 1024
	for start := 0; start < len(changes.Modified); {
		args := []string{"diff", "--no-ext-diff", "--quiet", revision, "--"}
		size := 0
		end := start
		for end < len(changes.Modified) {
			pathspec := ":(literal)" + changes.Modified[end]
			if end > start && size+len(pathspec) > maxPathspecArgumentBytes {
				break
			}
			args = append(args, pathspec)
			size += len(pathspec)
			end++
		}
		if end == start {
			return fmt.Errorf("selected path exceeds Git comparison argument limit")
		}
		flagArgs := append([]string{"ls-files", "-v", "-z", "--"}, args[5:]...)
		flags, err := source.GitOutput(root, flagArgs...)
		if err != nil {
			return fmt.Errorf("inspect selected path index flags: %w", err)
		}
		for _, record := range bytes.Split(flags, []byte{0}) {
			if len(record) < 3 || record[1] != ' ' {
				continue
			}
			tag := record[0]
			if (tag >= 'a' && tag <= 'z') || tag == 'S' {
				return fmt.Errorf("selected project input %q has an assume-unchanged or skip-worktree Git index flag; clear the flag before planning", string(record[2:]))
			}
		}
		if _, err := source.GitOutput(root, args...); err != nil {
			var exitErr *exec.ExitError
			if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
				return selectedBasisChangedError()
			}
			return fmt.Errorf("compare selected project inputs with Git's clean filters: %w", err)
		}
		start = end
	}
	return nil
}

func selectedBasisChangedError() error {
	return fmt.Errorf("selected project inputs differ from fixed base revision; commit accepted selected changes before planning")
}

// validateChangeImpact re-derives the optional since-impact from both fixed
// project revisions. Its digests are also covered by the persisted plan digest.
func validateChangeImpact(host Host, root string, current *Project, plan PlanRecord) error {
	if plan.ChangeImpact == nil {
		if plan.ChangeBaseRevision != "" || plan.ChangeBaseSnapshot != "" || plan.ChangeBaseProjectDigest != "" ||
			plan.ChangeBaseModelDigest != "" || plan.ChangeBaseReportDigest != "" || plan.ChangeImpactDigest != "" {
			return ErrStale
		}
		return nil
	}
	if current == nil || current.Snapshot == nil || plan.ChangeBaseRevision == "" || plan.ChangeBaseSnapshot == "" ||
		plan.ChangeBaseProjectDigest == "" || plan.ChangeBaseModelDigest == "" || plan.ChangeBaseReportDigest == "" ||
		plan.ChangeImpactDigest == "" {
		return ErrStale
	}
	baseline, err := host.Load(root, plan.ChangeBaseRevision)
	if err != nil {
		return fmt.Errorf("reload fixed change-impact baseline: %w", err)
	}
	if baseline == nil || baseline.Snapshot == nil || baseline.Revision != plan.ChangeBaseRevision ||
		baseline.Snapshot.Digest() != plan.ChangeBaseSnapshot || baseline.Digest != plan.ChangeBaseProjectDigest ||
		baseline.Report.ModelDigest != plan.ChangeBaseModelDigest || baseline.Report.Digest != plan.ChangeBaseReportDigest {
		return ErrStale
	}
	impact := projectmodel.Impact(baseline.Report, current.Report)
	if impact.BaseDigest != plan.ChangeBaseReportDigest || impact.CandidateDigest != current.Report.Digest ||
		impact.Digest != plan.ChangeImpactDigest || impact.Digest != plan.ChangeImpact.Digest {
		return ErrStale
	}
	return nil
}

func managerDepth(managers map[string]projectmodel.Manager, id string) int {
	depth := 0
	seen := map[string]bool{}
	for current := id; current != ""; current = managers[current].Parent {
		if seen[current] {
			return 1 << 30
		}
		seen[current] = true
		depth++
	}
	return depth
}

func checkDepth(tasks []ManagerTask, maximum int) error {
	for _, task := range tasks {
		if task.Depth > maximum {
			return fmt.Errorf("Manager %s exceeds configured maximum depth %d", task.ManagerID, maximum)
		}
	}
	return nil
}

func planChecks(report projectmodel.Report, selected map[string]bool) ([]CheckPlan, []string) {
	return planChecksWithImpact(report, selected, nil)
}

func planChecksWithImpact(report projectmodel.Report, selected map[string]bool, impactChecks []string) ([]CheckPlan, []string) {
	required := map[string]bool{}
	var findings []string
	for _, id := range impactChecks {
		required[id] = true
	}
	for _, artifact := range report.Artifacts {
		if artifact.Required && selected[artifact.Owner] {
			if len(artifact.Checks) == 0 {
				findings = append(findings, "required artifact "+artifact.ID+" has no declared check")
			}
			for _, check := range artifact.Checks {
				required[check] = true
			}
		}
	}
	var checks []CheckPlan
	for _, check := range report.Checks {
		if !selected[check.Owner] && !required[check.ID] {
			continue
		}
		valid := len(check.Command) > 0 && !strings.ContainsAny(check.Command[0], `/\`) && filepath.Base(check.Command[0]) == check.Command[0]
		for _, arg := range check.Command {
			if strings.ContainsRune(arg, '\x00') {
				valid = false
			}
		}
		if !valid {
			findings = append(findings, "check "+check.ID+" has no runnable literal argv")
		}
		checks = append(checks, CheckPlan{ID: check.ID, Owner: check.Owner,
			Command: append([]string(nil), check.Command...), Required: required[check.ID] || selected[check.Owner]})
	}
	for id := range required {
		found := false
		for _, check := range checks {
			if check.ID == id {
				found = true
				break
			}
		}
		if !found {
			findings = append(findings, "required artifact refers to missing check "+id)
		}
	}
	sort.Slice(checks, func(i, j int) bool { return checks[i].ID < checks[j].ID })
	return checks, uniqueSorted(findings)
}

func pathSatisfied(expected string, files map[string][]byte) bool {
	if !safeRepoPath(strings.TrimSuffix(expected, "/")) && !strings.HasSuffix(expected, "/") {
		return false
	}
	if strings.HasSuffix(expected, "/") {
		for path := range files {
			if strings.HasPrefix(path, expected) {
				return true
			}
		}
		return false
	}
	_, ok := files[expected]
	return ok
}

func hasErrorFinding(findings []projectmodel.Finding) bool {
	for _, finding := range findings {
		if finding.Severity == "error" {
			return true
		}
	}
	return false
}

func startableReport(report projectmodel.Report) bool {
	return report.Status == "succeeded" || report.Status == "incomplete"
}

func uniqueSorted(values []string) []string {
	seen := map[string]bool{}
	var out []string
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			out = append(out, value)
		}
	}
	sort.Strings(out)
	return out
}

func emptyCandidate() (candidateData, error) {
	id, err := newID()
	if err != nil {
		return candidateData{}, err
	}
	return candidateData{ID: id, Files: map[string]File{}}, nil
}

func modelEditCandidate(plan *EditPlan, base *Snapshot) (candidateData, error) {
	data := candidateData{ID: planDigestID(plan.Digest), Files: map[string]File{}}
	for _, change := range plan.Mutation.Files {
		if !safeRepoPath(change.Path) || !strings.HasPrefix(change.Path, ".markitect/model/") {
			return candidateData{}, fmt.Errorf("model edit path %q is outside the validated .markitect/model scope", change.Path)
		}
		mode := "100644"
		if existing, ok := base.Modes[change.Path]; ok {
			mode = existing
		}
		data.Files[change.Path] = File{Path: change.Path, Mode: mode, Content: []byte(change.Content), Delete: change.Delete}
	}
	return data, nil
}

func validateModelEditPaths(mutation Mutation) error {
	for _, change := range mutation.Files {
		if !safeRepoPath(change.Path) || !strings.HasPrefix(change.Path, ".markitect/model/") {
			return fmt.Errorf("projectrun ModelEdit path %q must remain within .markitect/model; edit runtime configuration before planning", change.Path)
		}
	}
	return nil
}

func planDigestID(value string) string {
	if len(value) >= 32 {
		trimmed := strings.TrimPrefix(value, "sha256:")
		if len(trimmed) >= 32 {
			return trimmed[:32]
		}
	}
	value = strings.ToLower(strings.ReplaceAll(value, "-", ""))
	if validID(value) {
		return value
	}
	sum := sha256String(value)
	return sum[:32]
}

func sha256String(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
