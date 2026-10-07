package execution

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// AreaReport is provenance for a private candidate, never acceptance or a vote.
type AreaReport struct {
	Area       core.DefinitionIdentity `json:"area"`
	NodeDigest string                  `json:"nodeDigest"`
	Status     string                  `json:"status"`
	Error      string                  `json:"error,omitempty"`
	ReportPath string                  `json:"reportPath"`
	Attempts   []AreaAttempt           `json:"attempts"`
}

type AreaAttempt struct {
	Attempt        int                 `json:"attempt"`
	InputDigest    string              `json:"inputDigest"`
	Repair         string              `json:"repair,omitempty"`
	Workspace      string              `json:"workspace,omitempty"`
	CandidateID    string              `json:"candidateId,omitempty"`
	SnapshotDigest string              `json:"snapshotDigest,omitempty"`
	Children       []AreaReport        `json:"children"`
	ExecutorRunID  string              `json:"executorRunId,omitempty"`
	ReviewerRunID  string              `json:"reviewerRunId,omitempty"`
	Checks         []host.GateResult   `json:"checks"`
	Review         *agentexec.Response `json:"review,omitempty"`
	Status         string              `json:"status"`
	Error          string              `json:"error,omitempty"`
}

type recursiveEngine struct {
	session    *actorSession
	model      government.Model
	order      government.Order
	delegation government.DelegationPlan
}

func validateAreaAssignments(tree government.DelegationNode, rt Runtime) error {
	expected := map[string]bool{}
	var visit func(government.DelegationNode)
	visit = func(n government.DelegationNode) {
		if n.Area.Key() != tree.Area.Key() {
			expected[n.Area.Key()] = true
		}
		for _, child := range n.Children {
			visit(child)
		}
	}
	visit(tree)
	if len(rt.Recursion.Areas) != len(expected) {
		return errors.New("recursive runtime must assign exactly every non-root participating Area")
	}
	for _, configured := range rt.Recursion.Areas {
		if !expected[configured.Area.Key()] {
			return errors.New("recursive runtime contains an unrelated or root Area assignment")
		}
		delete(expected, configured.Area.Key())
	}
	return nil
}

func (e *recursiveEngine) config(area core.DefinitionIdentity) (RunnerSpec, RunnerSpec, []authoring.Check) {
	executor, verifier, checks, ok := areaRuntime(e.session.runtime, area, e.delegation.Root.Area)
	if !ok {
		panic("validated recursive Area assignment missing")
	}
	return executor, verifier, checks
}

func (e *recursiveEngine) node(node government.DelegationNode, input *snapshot.Snapshot, inheritedRepair string) (candidate *snapshot.Snapshot, report AreaReport, runErr error) {
	s := e.session
	dir, err := government.CreateOperationalDirectory(s.runDir, "area-")
	if err != nil {
		return nil, report, err
	}
	report = AreaReport{Area: node.Area, NodeDigest: government.Digest(node), Status: "incomplete", ReportPath: filepath.Join(dir, "report.json"), Attempts: []AreaAttempt{}}
	defer func() {
		if runErr != nil {
			report.Error = runErr.Error()
			if len(report.Attempts) > 0 {
				last := &report.Attempts[len(report.Attempts)-1]
				if last.Error == "" {
					last.Error = runErr.Error()
				}
			}
		}
		if err := persistJSON(report.ReportPath, report); err != nil {
			runErr = errors.Join(runErr, err)
			report.Status = "incomplete"
		}
	}()
	executor, verifier, checks := e.config(node.Area)
	repair := inheritedRepair
	for attempt := 0; attempt <= s.runtime.Recursion.MaxRepairs; attempt++ {
		if err := s.ctx.Err(); err != nil {
			return nil, report, err
		}
		if attempt > 0 && s.control != nil {
			if err := s.control.ReserveRepair(s.report.RunID, node.Area.Key(), attempt); err != nil {
				return nil, report, err
			}
		}
		report.Attempts = append(report.Attempts, AreaAttempt{Attempt: attempt, InputDigest: input.Digest(), Repair: repair, Status: "incomplete", Children: []AreaReport{}, Checks: []host.GateResult{}})
		entry := &report.Attempts[len(report.Attempts)-1]
		// Each sibling gets a private branch rooted in this exact input. Nested
		// recursion uses the same algorithm; only actual process/check work takes
		// a semaphore, avoiding deadlock while a parent waits for descendants.
		type childResult struct {
			candidate *snapshot.Snapshot
			report    AreaReport
			err       error
		}
		results := make([]childResult, len(node.Children))
		done := make(chan int, len(node.Children))
		for index, child := range node.Children {
			go func(index int, child government.DelegationNode) {
				results[index].candidate, results[index].report, results[index].err = e.node(child, input, repair)
				done <- index
			}(index, child)
		}
		for range node.Children {
			<-done
		}
		assembled := cloneSnapshot(input)
		var childErr error
		for index, result := range results {
			entry.Children = append(entry.Children, result.report)
			if result.err != nil {
				childErr = errors.Join(childErr, result.err)
				continue
			}
			if err := mergeBranch(assembled, input, result.candidate, node.Children[index].Paths); err != nil {
				childErr = errors.Join(childErr, err)
			}
		}
		if childErr != nil {
			entry.Error = childErr.Error()
			return nil, report, childErr
		}
		workspace, err := os.MkdirTemp(s.temporary, "government-area-")
		if err != nil {
			return nil, report, err
		}
		entry.Workspace = workspace
		if err := source.Materialize(assembled, workspace); err != nil {
			return nil, report, err
		}
		contextValue := e.nodeContext(node, *entry, workspace, repair)
		if len(node.Work.Paths) > 0 {
			result, err := s.invoke("execute", executor, identityKeys(node.Work.Subjects), assembled, workspace, node.Paths, contextValue)
			entry.ExecutorRunID = result.Receipt.RunID
			if err != nil {
				entry.Error = err.Error()
				return nil, report, err
			}
			if result.Response.Outcome != agentexec.OutcomeProposed || len(result.Response.Uncertainty) > 0 {
				err = errors.New("Area executor did not propose certain material within its frozen task")
				entry.Error = err.Error()
				report.Status = "blocked"
				return nil, report, err
			}
			if err := confirmWorkspace(workspace, assembled); err != nil {
				return nil, report, err
			}
			before := assembled
			assembled, err = applyProposal(assembled, node.Work.Paths, result.Response.CandidateFiles)
			if err != nil {
				entry.Error = err.Error()
				return nil, report, err
			}
			if err := writeChanges(workspace, before, assembled); err != nil {
				return nil, report, err
			}
		}
		entry.SnapshotDigest = assembled.Digest()
		entry.CandidateID = government.Digest(struct {
			Node, Input, Snapshot, Runtime, Tool, Delegation string
			Attempt                                          int
		}{report.NodeDigest, entry.InputDigest, entry.SnapshotDigest, s.report.RuntimePin, s.report.ToolPins, e.delegation.Digest, attempt})
		if err := confirmWorkspace(workspace, assembled); err != nil {
			return nil, report, err
		}
		entry.Checks, err = e.checks(assembled, checks)
		checkErr := err
		var checkFailure *host.VerifyError
		if err != nil && (!errors.As(err, &checkFailure) || checkFailure.Kind != "gate-failure") {
			entry.Error = err.Error()
			return nil, report, err
		}
		{
			contextValue = e.nodeContext(node, *entry, workspace, repair)
			contextValue["phase"] = "review"
			contextValue["candidate"] = map[string]string{"id": entry.CandidateID, "snapshotDigest": entry.SnapshotDigest}
			contextValue["checks"] = entry.Checks
			result, invokeErr := s.invoke("review", verifier, append([]string{node.Area.Key()}, identityKeys(node.Subjects)...), assembled, workspace, node.Paths, contextValue)
			entry.ReviewerRunID = result.Receipt.RunID
			entry.Review = &result.Response
			if invokeErr != nil {
				entry.Error = invokeErr.Error()
				return nil, report, invokeErr
			}
			err = errors.Join(checkErr, requireReview(result.Response, append([]string{node.Area.Key()}, identityKeys(node.Subjects)...)))
		}
		if integrityErr := confirmWorkspace(workspace, assembled); integrityErr != nil {
			return nil, report, integrityErr
		}
		if err == nil {
			entry.Status = "passed-scoped"
			report.Status = "passed-scoped"
			return assembled, report, nil
		}
		entry.Status = "blocked"
		entry.Error = err.Error()
		if attempt == s.runtime.Recursion.MaxRepairs {
			report.Status = "blocked"
			return nil, report, fmt.Errorf("Area %s exhausted bounded repair: %w", node.Area.Key(), err)
		}
		// Re-delegate within the same frozen owner map. Earlier reviews are
		// retained as lineage only; every rerun gets new processes and checks.
		repair = governmentJSONFeedback(node.Area, *entry)
		input = assembled
	}
	panic("finite recursive attempt loop exhausted without result")
}

func governmentJSONFeedback(area core.DefinitionIdentity, entry AreaAttempt) string {
	// A stable structured value serialized as prose in the existing agent context.
	return fmt.Sprintf("Parent/Area %s rejected candidate %s: %s; checks=%s; review=%s", area.Key(), entry.CandidateID, entry.Error, jsonText(entry.Checks), jsonText(entry.Review))
}

func (e *recursiveEngine) nodeContext(node government.DelegationNode, entry AreaAttempt, workspace, repair string) map[string]any {
	definitions := contextDefinitions(e.model, node)
	order := e.order
	order.Subjects = node.Subjects
	order.Paths = node.Work.Paths
	order.UnknownScope = false
	return map[string]any{"phase": "execute", "runId": e.session.report.RunID, "order": order, "area": node.Area, "attempt": entry.Attempt, "repair": repair, "allowedPaths": node.Work.Paths, "inputPaths": node.Paths, "subjects": node.Subjects, "work": node.Work, "definitions": definitions, "delegationDigest": e.delegation.Digest, "nodeDigest": government.Digest(node), "inputDigest": entry.InputDigest, "childReports": entry.Children, "workspace": workspace, "limits": map[string]any{"delegation": e.session.runtime.Recursion.Limits, "parallelism": e.session.runtime.Recursion.Parallelism, "maxRepairs": e.session.runtime.Recursion.MaxRepairs}}
}

func (e *recursiveEngine) checks(candidate *snapshot.Snapshot, checks []authoring.Check) ([]host.GateResult, error) {
	s := e.session
	select {
	case s.semaphore <- struct{}{}:
		defer func() { <-s.semaphore }()
	case <-s.ctx.Done():
		return nil, s.ctx.Err()
	}
	if pin, err := toolPins(s.runtime); err != nil || pin != s.report.ToolPins {
		return nil, errors.Join(errors.New("runtime/tool pins changed before Area checks"), err)
	}
	return freshChecks(s.ctx, candidate, checks, s.control)
}

func freshChecks(ctx context.Context, candidate *snapshot.Snapshot, checks []authoring.Check, controls ...Control) (results []host.GateResult, err error) {
	for _, check := range checks {
		for _, control := range controls {
			if control != nil {
				if err := control.Fence(); err != nil {
					return results, err
				}
			}
		}
		seconds := authoring.DefaultCheckTimeoutSeconds
		if check.TimeoutSeconds != nil {
			seconds = *check.TimeoutSeconds
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < time.Duration(seconds)*time.Second+3*time.Second {
			return results, errors.New("remaining run budget cannot cover the next technical check cap")
		}
		if err := ctx.Err(); err != nil {
			return results, err
		}
		gates, err := host.VerifySnapshotChecks(candidate, []authoring.Check{check})
		results = append(results, gates...)
		if err != nil {
			return results, err
		}
		if err := ctx.Err(); err != nil {
			return results, err
		}
	}
	return results, nil
}

func identityKeys(ids []core.DefinitionIdentity) []string {
	keys := make([]string, 0, len(ids))
	for _, id := range ids {
		keys = append(keys, id.Key())
	}
	return keys
}

func cloneSnapshot(input *snapshot.Snapshot) *snapshot.Snapshot {
	out := &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, data := range input.Files {
		out.Files[path] = append([]byte{}, data...)
		out.Modes[path] = input.Modes[path]
	}
	return out
}

func applyProposal(input *snapshot.Snapshot, paths []string, files []agentexec.CandidateFile) (*snapshot.Snapshot, error) {
	out := cloneSnapshot(input)
	allowed := map[string]bool{}
	for _, path := range paths {
		allowed[path] = true
	}
	for _, file := range files {
		if !allowed[file.Path] {
			return nil, fmt.Errorf("executor proposed path outside frozen Writer scope: %s", file.Path)
		}
		out.Files[file.Path] = []byte(file.Content)
		out.Modes[file.Path] = snapshot.RegularMode
		if file.Mode == "0755" {
			out.Modes[file.Path] = snapshot.ExecutableMode
		}
	}
	return out, nil
}

func mergeBranch(assembled, input, branch *snapshot.Snapshot, paths []string) error {
	allowed := map[string]bool{}
	for _, path := range paths {
		allowed[path] = true
	}
	for _, path := range changedPaths(input, branch) {
		if !allowed[path] {
			return fmt.Errorf("child material changed non-owned path %s", path)
		}
		if _, exists := branch.Files[path]; !exists {
			return fmt.Errorf("child deletion is unsupported: %s", path)
		}
		if government.BytesDigest(assembled.Files[path]) != government.BytesDigest(input.Files[path]) || assembled.Modes[path] != input.Modes[path] {
			return fmt.Errorf("child integration writer collision at %s", path)
		}
		assembled.Files[path] = append([]byte{}, branch.Files[path]...)
		assembled.Modes[path] = branch.Modes[path]
	}
	return nil
}

func changedPaths(before, after *snapshot.Snapshot) []string {
	paths := map[string]bool{}
	for path := range before.Files {
		paths[path] = true
	}
	for path := range after.Files {
		paths[path] = true
	}
	changed := []string{}
	for path := range paths {
		if government.BytesDigest(before.Files[path]) != government.BytesDigest(after.Files[path]) || before.Modes[path] != after.Modes[path] {
			changed = append(changed, path)
		}
	}
	sort.Strings(changed)
	return changed
}

func writeChanges(workspace string, before, after *snapshot.Snapshot) error {
	for _, path := range changedPaths(before, after) {
		if !government.SafePath(path) || strings.Contains(path, "\\") {
			return errors.New("unsafe candidate material path")
		}
		dest := filepath.Join(workspace, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		mode := os.FileMode(0644)
		if after.Modes[path] == snapshot.ExecutableMode {
			mode = 0755
		}
		if err := os.WriteFile(dest, after.Files[path], mode); err != nil {
			return err
		}
		if err := os.Chmod(dest, mode); err != nil {
			return err
		}
	}
	return confirmWorkspace(workspace, after)
}

func jsonText(v any) string { data, _ := json.Marshal(v); return string(data) }
func checkDefinitionsDigest(rt Runtime) string {
	if rt.Recursion == nil {
		return government.Digest(rt.Checks)
	}
	return government.Digest(struct {
		Root  []authoring.Check
		Areas []AreaRunner
	}{rt.Checks, rt.Recursion.Areas})
}

// The relevant subtree and its prior authority chain are context, not extra
// permissions. Sibling runtime configurations and transcripts are not supplied.
func contextDefinitions(model government.Model, node government.DelegationNode) []core.Definition {
	ids := map[string]bool{model.Constitution.Key(): true}
	var selectNode func(government.DelegationNode)
	selectNode = func(n government.DelegationNode) {
		ids[n.Area.Key()] = true
		for _, id := range append(append([]core.DefinitionIdentity{}, n.Subjects...), n.Work.Mandates...) {
			ids[id.Key()] = true
		}
		for _, child := range n.Children {
			selectNode(child)
		}
	}
	selectNode(node)
	paths := map[string]bool{}
	for _, path := range node.Paths {
		paths[path] = true
	}
	// Include the purpose-bearing, many-to-many coverage records, not merely
	// parallel lists of subject IDs and file names.
	for _, d := range model.Canonical.Definitions {
		if d.APIVersion != government.APIVersion {
			continue
		}
		if d.Kind == "Artifact" && paths[fmt.Sprint(d.Spec["path"])] {
			ids[d.Identity().Key()] = true
		}
	}
	for _, d := range model.Canonical.Definitions {
		if d.APIVersion != government.APIVersion {
			continue
		}
		if d.Kind == "Responsibility" && ids[identityValue(d.Spec["subject"]).Key()] {
			ids[d.Identity().Key()] = true
			ids[identityValue(d.Spec["area"]).Key()] = true
		}
		if d.Kind == "Realization" && ids[identityValue(d.Spec["subject"]).Key()] && ids[identityValue(d.Spec["artifact"]).Key()] {
			ids[d.Identity().Key()] = true
		}
	}
	// Retain the explicit higher Area/Mandate records that legitimate this slice.
	for changed := true; changed; {
		changed = false
		for _, d := range model.Canonical.Definitions {
			if !ids[d.Identity().Key()] || d.APIVersion != government.APIVersion {
				continue
			}
			if d.Kind != "Mandate" && d.Kind != "Area" {
				continue
			}
			for _, name := range []string{"parent", "area"} {
				id := identityValue(d.Spec[name])
				if id.Name != "" && !ids[id.Key()] {
					ids[id.Key()] = true
					changed = true
				}
			}
		}
	}
	definitions := []core.Definition{}
	for _, d := range model.Canonical.Definitions {
		if ids[d.Identity().Key()] {
			definitions = append(definitions, d)
		}
	}
	return definitions
}

// The existing agent wire contract is finite too; reject a known oversized
// root/subtree before spending any actor budget on its descendants.
func validateNodeWireBounds(node government.DelegationNode) error {
	if len(node.Subjects)+1 > agentexec.MaxVerifierObservations || len(node.Paths) > 128 {
		return fmt.Errorf("Area %s exceeds agent request subject/artifact bounds", node.Area.Key())
	}
	for _, child := range node.Children {
		if err := validateNodeWireBounds(child); err != nil {
			return err
		}
	}
	return nil
}
