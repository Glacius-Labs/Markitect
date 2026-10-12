package projectrun

import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	projectconfig "github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
)

// Plan binds a requested project operation to one fixed model, inventory,
// runner and runtime configuration. It persists no source changes and starts
// no external agent.
func Plan(host Host, root, revision string, request PlanRequest) (PlanRecord, error) {
	return PlanExpecting(host, root, revision, request, "")
}

// PlanExpecting is Plan with a guard for persisting: when expectedPreview is
// set, an authorized plan is persisted only if its PlanPreviewDigest still
// equals the reviewed preview's; otherwise it returns ErrStale.
func PlanExpecting(host Host, root, revision string, request PlanRequest, expectedPreview string) (PlanRecord, error) {
	basis, err := loadPlanBasis(host, root, revision, request)
	if err != nil {
		return PlanRecord{}, err
	}
	semantics, err := preparePlanSemantics(host, basis)
	if err != nil {
		return PlanRecord{}, err
	}
	plan := planFromSemantics(root, basis, semantics)
	if err := bindExplorationReadiness(host, root, basis.Project, basis.Working.Snapshot, basis.Request, &plan); err != nil {
		return plan, err
	}
	plan.Digest, err = planDigest(plan)
	if err != nil {
		return plan, err
	}
	if !basis.Request.ExecuteAuthorized {
		return plan, nil
	}
	if expectedPreview != "" {
		preview, err := PlanPreviewDigest(plan)
		if err != nil {
			return plan, err
		}
		if preview != expectedPreview {
			return plan, ErrStale
		}
	}
	if err := persistAuthorizedPlan(root, semantics.ID, plan, semantics.InitialCandidate); err != nil {
		return plan, err
	}
	return plan, nil
}

func planManagers(report projectmodel.Report, baseFiles map[string][]byte, request PlanRequest, edit *projectvalue.EditPlan, changeImpact *projectmodel.ChangeImpact, limits projectconfig.Limits) ([]ManagerTask, map[string]bool, []string, error) {
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

func requireCleanSelectedBasis(fixed, working *snapshot.Snapshot) error {
	if fixed == nil || working == nil {
		return fmt.Errorf("fixed and working selected snapshots are required")
	}
	if fixed.Digest() != working.Digest() {
		return selectedBasisChangedError()
	}
	return nil
}

func requireCleanSelectedBasisAtRevision(root, revision string, fixed, working *snapshot.Snapshot) error {
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
	const maxPathspecArgumentBytes = 16 * 1024
	for start := 0; start < len(changes.Modified); {
		args := []string{"ls-files", "-v", "-z", "--"}
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
		flags, err := source.GitOutput(root, args...)
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
		start = end
	}
	// Compare captured changed bytes through Git's clean filters. On Windows, a
	// normal core.autocrlf checkout has CRLF worktree bytes while the fixed Git
	// tree contains LF. Hashing the captured bytes avoids consulting a second,
	// potentially different worktree read as `git diff` would.
	for _, path := range changes.Modified {
		if fixed.Modes[path] != working.Modes[path] {
			return selectedBasisChangedError()
		}
		fixedBytes, fixedOK := fixed.Files[path]
		workingBytes, workingOK := working.Files[path]
		if !fixedOK || !workingOK {
			return selectedBasisChangedError()
		}
		fixedAttributes, err := source.GitOutput(root, "check-attr", "--source", revision, "-z", "--all", "--", path)
		if err != nil {
			return fmt.Errorf("inspect fixed Git attributes for selected path %q: %w", path, err)
		}
		workingAttributes, err := source.GitOutput(root, "check-attr", "-z", "--all", "--", path)
		if err != nil {
			return fmt.Errorf("inspect working Git attributes for selected path %q: %w", path, err)
		}
		if !bytes.Equal(fixedAttributes, workingAttributes) {
			return fmt.Errorf("selected project input %q has Git attributes that differ from fixed revision %q", path, revision)
		}
		workingOID, err := source.GitOutputInput(root, workingBytes, "hash-object", "--path="+path, "--stdin")
		if err != nil {
			return fmt.Errorf("compare captured selected input %q through Git's clean filters: %w", path, err)
		}
		fixedOID, err := source.GitOutputInput(root, fixedBytes, "hash-object", "--no-filters", "--stdin")
		if err != nil {
			return fmt.Errorf("hash fixed selected input %q: %w", path, err)
		}
		if !bytes.Equal(bytes.TrimSpace(workingOID), bytes.TrimSpace(fixedOID)) {
			return selectedBasisChangedError()
		}
	}
	return nil
}

// RequireCleanSelectedBasisAtRevision verifies a captured selected working
// snapshot against the exact fixed project revision using Git's clean filters.
func RequireCleanSelectedBasisAtRevision(root, revision string, fixed, working *snapshot.Snapshot) error {
	return requireCleanSelectedBasisAtRevision(root, revision, fixed, working)
}

func selectedBasisChangedError() error {
	return fmt.Errorf("selected project inputs differ from fixed base revision; commit accepted selected changes before planning")
}

// validateChangeImpact re-derives the optional since-impact from both fixed
// project revisions. Its digests are also covered by the persisted plan digest.
func validateChangeImpact(host Host, root string, current *projectvalue.Project, plan PlanRecord) error {
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

// reviewPhaseStarts counts the reviewer starts a plan must allow: the work
// review of each Manager whose own scope requires one, and the integration
// review of every Manager with active children. An integration review is
// required whenever direct children deliver files (phaseReviewRequired), which
// planning cannot know yet, so the count is conservative: a tight start limit
// is rejected here instead of blocking the run.
func reviewPhaseStarts(project *projectvalue.Project, tasks []ManagerTask) int {
	starts := 0
	for _, task := range tasks {
		if reviewRequired(project, task) {
			starts++
		}
		if len(activeChildren(tasks, task.ManagerID)) > 0 {
			starts++
		}
	}
	return starts
}

func modelEditCandidate(plan *projectvalue.EditPlan, base *snapshot.Snapshot) (candidateData, error) {
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

func validateModelEditPaths(mutation projectvalue.Mutation) error {
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
