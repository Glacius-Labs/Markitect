package projectrun

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/modules/projectmodel"
)

// Plan binds a requested project operation to one fixed model, inventory,
// runner and runtime configuration. It persists no source changes and starts
// no external agent.
func Plan(host Host, root, revision string, request PlanRequest) (PlanRecord, error) {
	var plan PlanRecord
	var err error
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
	if project.Snapshot.Provisional {
		return plan, fmt.Errorf("project run requires a non-provisional snapshot")
	}
	working, err := host.Load(root, "")
	if err != nil {
		return plan, fmt.Errorf("capture current project working inputs: %w", err)
	}
	if working == nil || working.Snapshot == nil {
		return plan, fmt.Errorf("project runtime requires a selected working-input snapshot")
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
	id, err := newID()
	if err != nil {
		return plan, err
	}
	managerTasks, selected, findings, err := planManagers(finalProject.Report, project.Snapshot.Files, request, editPlan, runtime.Limits)
	if err != nil {
		return plan, err
	}
	if len(managerTasks) == 0 {
		return plan, fmt.Errorf("plan has no responsible Manager")
	}
	if err := checkDepth(managerTasks, runtime.Limits.MaxDepth); err != nil {
		return plan, err
	}
	checkPlans, checkFindings := planChecks(finalProject.Report, selected)
	findings = append(findings, checkFindings...)
	checkPlans, executableFindings, executableErr := bindCheckExecutables(checkPlans, runtime)
	if executableErr != nil {
		return plan, executableErr
	}
	findings = append(findings, executableFindings...)
	checkFindings = append(checkFindings, executableFindings...)
	minimumStarts := len(managerTasks)*2 + len(checkPlans)
	if runtime.Verifier != nil {
		minimumStarts++
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
		fingerprint, fingerprintErr := agentexec.Fingerprint(config)
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
		fingerprint, fingerprintErr := agentexec.Fingerprint(config)
		if fingerprintErr != nil {
			return plan, fmt.Errorf("fingerprint runtime verifier: %w", fingerprintErr)
		}
		fingerprints["$verifier"] = fingerprint
	}
	runtimeDigest, err := digest(struct {
		Runtime      Runtime           `json:"runtime"`
		Fingerprints map[string]string `json:"fingerprints"`
	}{Runtime: runtime, Fingerprints: fingerprints})
	if err != nil {
		return plan, err
	}
	plan = PlanRecord{
		APIVersion: APIVersion, ID: id, Status: StatusPlanned, Goal: request.Goal,
		ExecuteAuthorized: request.ExecuteAuthorized, Root: root,
		BaseRevision: project.Revision, TargetBranch: targetBranch, TargetHead: targetHead, BaseSnapshot: project.Snapshot.Digest(),
		WorkingSnapshot:  working.Snapshot.Digest(),
		RepositoryDigest: repository.Digest, BaseProjectDigest: project.Digest, WorkingProjectDigest: working.Digest,
		BaseModelDigest: project.Report.ModelDigest,
		ModelDigest:     finalProject.Report.ModelDigest, ReportDigest: finalProject.Report.Digest,
		RuntimeDigest: runtimeDigest, PlannedAt: time.Now().UTC(), Managers: managerTasks,
		Checks: checkPlans, ModelEdit: editPlan, InitialCandidateID: initialCandidate.ID,
		RuntimeAgents: fingerprints, Findings: uniqueSorted(findings), Blockers: uniqueSorted(checkFindings),
	}
	plan.Digest, err = planDigest(plan)
	if err != nil {
		return plan, err
	}
	if !request.ExecuteAuthorized {
		return plan, nil
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

func planManagers(report projectmodel.Report, baseFiles map[string][]byte, request PlanRequest, edit *EditPlan, limits Limits) ([]ManagerTask, map[string]bool, []string, error) {
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
	if len(targets) == 0 && edit != nil {
		for _, id := range edit.Impact.Managers {
			if _, ok := managerByID[id]; ok {
				targets[id] = true
			}
		}
	}
	findings := append([]string(nil), report.Unknown...)
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
	if len(targets) == 0 {
		for _, id := range children[""] {
			targets[id] = true
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
	required := map[string]bool{}
	var findings []string
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
