// Package execution implements the bounded Government G2 vertical slice. It
// consumes trusted runtime configuration; model outputs never select authority.
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
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

type Options struct {
	Repo, ConfigPath, OrderPath string
	Runtime                     Runtime
}
type ActorRecord struct {
	Phase  string              `json:"phase"`
	SlotID string              `json:"slotId"`
	Scopes []string            `json:"scopes"`
	Result agentexec.RunResult `json:"result"`
	Error  string              `json:"error,omitempty"`
}
type Report struct {
	APIVersion        string                         `json:"apiVersion"`
	RunID             string                         `json:"runId"`
	Status            string                         `json:"status"`
	Stage             string                         `json:"stage"`
	Error             string                         `json:"error,omitempty"`
	ReportPath        string                         `json:"reportPath,omitempty"`
	ActiveRef         string                         `json:"activeRef"`
	BaseRevision      string                         `json:"baseRevision"`
	PriorConstitution string                         `json:"priorConstitution"`
	RuntimePin        string                         `json:"runtimePin"`
	ToolPins          string                         `json:"toolPins"`
	TimeoutSeconds    int                            `json:"timeoutSeconds"`
	Workspace         string                         `json:"workspace,omitempty"`
	Plan              government.Plan                `json:"plan"`
	Cabinet           []government.CabinetMember     `json:"cabinet"`
	ChangedPaths      []string                       `json:"changedPaths"`
	CandidateCommit   string                         `json:"candidateCommit,omitempty"`
	CandidateTree     string                         `json:"candidateTree,omitempty"`
	Candidate         *government.MaterialCandidate  `json:"candidate,omitempty"`
	Checks            []host.GateResult              `json:"checks"`
	Actors            []ActorRecord                  `json:"actors"`
	Evidence          *government.Evidence           `json:"evidence,omitempty"`
	Votes             []government.RessortVote       `json:"votes"`
	Decision          *government.AcceptanceDecision `json:"decision,omitempty"`
	Promotion         *government.PromotionResult    `json:"promotion,omitempty"`
	Limits            []string                       `json:"limits"`
}

// Run creates one isolated material candidate, runs actual configured processes,
// then promotes only an explicitly assented final tree. Even aborted runs retain
// their report and any completed receipts outside the material tree.
func Run(ctx context.Context, opts Options) (report Report, runErr error) {
	rt := opts.Runtime
	report = Report{APIVersion: RuntimeVersion, Status: "incomplete", Stage: "prepare", ActiveRef: rt.ActiveRef, BaseRevision: rt.ExpectedBase, Limits: []string{
		"Cooperative processes share caller OS rights; separated processes and input audits are not an OS sandbox or proof of institutional independence.",
		"G2 executes one Writer Area; recursive child execution, amendment activation, queue and complete recovery remain unavailable.",
		"PromotionIntent and Git CAS are separate durable effects. No automatic rollback or atomic ledger/Git transaction is claimed.",
		"Accepted-scoped records configured process and check outcomes; it does not establish semantic sufficiency or human acceptance.",
		"Area review is read-only evidence assigned by the frozen plan, never a Ressort vote or acceptance/change/promotion authority; no per-Area review mandate is inferred from implement.",
	}}
	defer func() {
		if runErr != nil {
			report.Error = runErr.Error()
		}
		if report.ReportPath != "" {
			if err := persistJSON(report.ReportPath, report); err != nil {
				report.Status = "incomplete"
				runErr = errors.Join(runErr, fmt.Errorf("persist run report: %w", err))
				report.Error = runErr.Error()
			}
		}
	}()
	if ctx == nil {
		return report, errors.New("execution context is required")
	}
	if rt.TimeoutSeconds > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, time.Duration(rt.TimeoutSeconds)*time.Second)
		defer cancel()
	}
	report.TimeoutSeconds = rt.TimeoutSeconds
	// Validate the reporting boundary before creating any operational files.
	state, err := realDirectory(rt.StateDirectory)
	if err != nil {
		return report, err
	}
	repo, err := filepath.Abs(opts.Repo)
	if err != nil {
		return report, err
	}
	repo, err = realDirectory(repo)
	if err != nil {
		return report, err
	}
	common, err := gitOutput(ctx, repo, nil, nil, "rev-parse", "--path-format=absolute", "--git-common-dir")
	if err != nil {
		return report, err
	}
	common, err = realDirectory(common)
	if err != nil {
		return report, err
	}
	if directoriesOverlap(repo, state) || directoriesOverlap(common, state) {
		return report, errors.New("run state must be outside the repository")
	}
	runDir, err := government.CreateOperationalDirectory(state, "government-run-")
	if err != nil {
		return report, err
	}
	report.RunID = filepath.Base(runDir)
	report.ReportPath = filepath.Join(runDir, "report.json")
	if err := ValidateRuntime(rt); err != nil {
		return report, err
	}
	repo, err = realDirectory(repo)
	if err != nil {
		return report, err
	}
	temp, err := realDirectory(rt.TemporaryDirectory)
	if err != nil {
		return report, err
	}
	for _, external := range []string{state, temp} {
		if directoriesOverlap(repo, external) || directoriesOverlap(common, external) {
			return report, errors.New("run state and temporary roots must be external to source and Git metadata")
		}
	}
	active, err := gitOutput(ctx, repo, nil, nil, "rev-parse", "--verify", rt.ActiveRef+"^{commit}")
	if err != nil || active != rt.ExpectedBase {
		return report, errors.New("active revision differs from expected immutable base")
	}
	base, err := source.Load(repo, rt.ExpectedBase)
	if err != nil || base.ID != rt.ExpectedBase {
		return report, errors.Join(errors.New("cannot acquire exact active snapshot"), err)
	}
	if !government.SafePath(opts.ConfigPath) || !government.SafePath(opts.OrderPath) {
		return report, errors.New("config and order must be exact repository-relative paths")
	}
	var src government.Source
	configBytes := base.Files[opts.ConfigPath]
	if err := government.Decode(configBytes, &src); err != nil {
		return report, err
	}
	for i := range src.Schemas {
		src.Schemas[i].Source.Path = opts.ConfigPath
		src.Schemas[i].Source.Digest = government.BytesDigest(configBytes)
	}
	for i := range src.Definitions {
		src.Definitions[i].Source.Path = opts.ConfigPath
		src.Definitions[i].Source.Digest = government.BytesDigest(configBytes)
	}
	model := government.Compile(src)
	report.PriorConstitution = model.Digest
	if len(model.Findings) > 0 {
		return report, errors.New("active Government model is invalid")
	}
	var order government.Order
	if err := government.Decode(base.Files[opts.OrderPath], &order); err != nil {
		return report, err
	}
	if order.Action != "implement" {
		return report, errors.New("G2 only executes implementation under unchanged prior authority; amendments require G4")
	}
	workspace, err := os.MkdirTemp(temp, "government-candidate-")
	if err != nil {
		return report, err
	}
	report.Workspace = workspace // retained for audit, including on failure
	if err := source.Materialize(base, workspace); err != nil {
		return report, err
	}
	if err := confirmWorkspace(workspace, base); err != nil {
		return report, err
	}
	observation, err := inventory.Capture(workspace, src.Observation)
	if err != nil {
		return report, err
	}
	report.Plan = government.BuildPlan(model, order, observation)
	if report.Plan.Status == "blocked" {
		report.Status = "blocked"
		return report, errors.New("prior-authority plan is blocked")
	}
	if len(report.Plan.Work) != 1 || len(report.Plan.Work[0].Paths) == 0 {
		return report, errors.New("G2 requires exactly one nonempty Writer Area; recursive execution remains G3")
	}
	if len(report.Plan.IntegrationReviews) == 0 || len(report.Plan.IntegrationReviews) > 128 {
		return report, errors.New("G2 requires 1 through 128 explicit integration review scopes")
	}
	report.Cabinet, err = freezeCabinet(model, rt)
	if err != nil {
		return report, err
	}
	report.RuntimePin, err = FingerprintRuntime(rt)
	if err != nil {
		return report, err
	}
	report.ToolPins, err = toolPins(rt)
	if err != nil {
		return report, err
	}
	frozenRunners := map[string]runnerFingerprint{}
	for _, spec := range append([]RunnerSpec{rt.Executor, rt.Verifier}, ressortRunners(rt)...) {
		pin, err := fingerprintRunner(spec)
		if err != nil {
			return report, err
		}
		frozenRunners[spec.SlotID] = pin
	}
	allowed := map[string]bool{}
	for _, path := range report.Plan.Work[0].Paths {
		if path == opts.ConfigPath || path == opts.OrderPath {
			return report, errors.New("implementation order cannot write constitutional or order input")
		}
		allowed[path] = true
	}
	artifacts := func(s *snapshot.Snapshot) []agentexec.Artifact {
		out := []agentexec.Artifact{}
		for _, path := range sortedPaths(s.Files) {
			if allowed[path] {
				mode := "0644"
				if s.Modes[path] == snapshot.ExecutableMode {
					mode = "0755"
				}
				out = append(out, agentexec.Artifact{Path: path, Mode: mode, Digest: government.BytesDigest(s.Files[path]), Content: s.Files[path]})
			}
		}
		return out
	}
	subjects := []string{}
	for _, subject := range report.Plan.Affected {
		subjects = append(subjects, subject.Key())
	}
	invoke := func(phase string, spec RunnerSpec, scopes []string, s *snapshot.Snapshot, extra map[string]any) (agentexec.RunResult, error) {
		pin, err := fingerprintRunner(spec)
		if err != nil || pin != frozenRunners[spec.SlotID] {
			return agentexec.RunResult{}, errors.Join(errors.New("runner differs from frozen runtime before invocation"), err)
		}
		contextValue := map[string]any{"phase": phase, "runId": report.RunID, "order": order, "plan": report.Plan, "priorModel": model.Canonical, "cabinet": report.Cabinet, "allowedPaths": report.Plan.Work[0].Paths, "workspace": workspace}
		for k, v := range extra {
			contextValue[k] = v
		}
		contextBytes, _ := json.Marshal(contextValue)
		role := agentexec.RoleVerifier
		if phase == "execute" {
			role = agentexec.RoleExecutor
		}
		request := agentexec.Request{Role: role, SourceRevision: rt.ExpectedBase, ModelDigest: model.Digest, ModulePin: report.ToolPins, ProjectionID: "government/" + phase + "/" + spec.SlotID, ScopeIDs: scopes, PolicyIDs: []string{model.Constitution.Key()}, Context: contextBytes, Artifacts: artifacts(s)}
		result, err := Invoke(ctx, spec, request, workspace, runDir, temp)
		if err == nil && (result.Receipt.ConfigDigest != pin.ConfigDigest || result.Receipt.ExecutableDigest != pin.ExecutableDigest) {
			err = errors.New("actual runner receipt differs from frozen runtime")
		}
		record := ActorRecord{Phase: phase, SlotID: spec.SlotID, Scopes: scopes, Result: result}
		if err != nil {
			record.Error = err.Error()
		}
		report.Actors = append(report.Actors, record)
		persistErr := persistJSON(filepath.Join(runDir, fmt.Sprintf("actor-%02d.json", len(report.Actors))), record)
		return result, errors.Join(err, persistErr)
	}
	report.Stage = "execute"
	executed, err := invoke("execute", rt.Executor, subjects, base, nil)
	if err != nil {
		return report, err
	}
	if executed.Response.Outcome != agentexec.OutcomeProposed {
		report.Status = "blocked"
		return report, errors.New("executor did not propose material")
	}
	if err := confirmWorkspace(workspace, base); err != nil {
		return report, err
	}
	candidate := &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, data := range base.Files {
		candidate.Files[path] = append([]byte(nil), data...)
		candidate.Modes[path] = base.Modes[path]
	}
	for _, file := range executed.Response.CandidateFiles {
		if !allowed[file.Path] {
			return report, fmt.Errorf("executor proposed path outside frozen Writer scope: %s", file.Path)
		}
		candidate.Files[file.Path] = []byte(file.Content)
		candidate.Modes[file.Path] = snapshot.RegularMode
		if file.Mode == "0755" {
			candidate.Modes[file.Path] = snapshot.ExecutableMode
		}
	}
	for path, data := range candidate.Files {
		if government.BytesDigest(data) != government.BytesDigest(base.Files[path]) || candidate.Modes[path] != base.Modes[path] {
			report.ChangedPaths = append(report.ChangedPaths, path)
		}
	}
	sort.Strings(report.ChangedPaths)
	if len(report.ChangedPaths) == 0 {
		return report, errors.New("executor proposal made no actual material change")
	}
	for _, path := range report.ChangedPaths {
		dest := filepath.Join(workspace, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return report, err
		}
		mode := os.FileMode(0644)
		if candidate.Modes[path] == snapshot.ExecutableMode {
			mode = 0755
		}
		if err := os.WriteFile(dest, candidate.Files[path], mode); err != nil {
			return report, err
		}
		if err := os.Chmod(dest, mode); err != nil {
			return report, err
		}
	}
	if err := confirmWorkspace(workspace, candidate); err != nil {
		return report, err
	}
	report.CandidateCommit, report.CandidateTree, err = commitCandidate(ctx, repo, rt.ExpectedBase, report.RunID, runDir, report.ChangedPaths, candidate)
	if err != nil {
		return report, err
	}
	candidate.ID = report.CandidateCommit
	material, err := government.NewMaterialCandidate(government.MaterialCandidateInput{PriorConstitutionDigest: model.Digest, BaseRevision: rt.ExpectedBase, RepositoryTreeDigest: government.Digest(struct{ Tree, Snapshot string }{report.CandidateTree, candidate.Digest()}), ModelDigest: model.Digest, PlanDigest: report.Plan.Digest, CheckDefinitionsDigest: government.Digest(rt.Checks), ToolPinsDigest: report.ToolPins, InventoryDigest: observation.Digest})
	if err != nil {
		return report, err
	}
	report.Candidate = &material
	if err := persistJSON(filepath.Join(runDir, "material.json"), material); err != nil {
		return report, err
	}
	report.Stage = "technical-checks"
	if pin, err := toolPins(rt); err != nil || pin != report.ToolPins {
		return report, errors.Join(errors.New("runtime/tool pins changed before fresh technical checks"), err)
	}
	// The existing fixed-snapshot checker has its own per-command deadline.
	// Reserve its complete cap before each call, then reject any elapsed run
	// deadline before starting another stage. Snapshot IO remains cooperative.
	for _, check := range rt.Checks {
		seconds := authoring.DefaultCheckTimeoutSeconds
		if check.TimeoutSeconds != nil {
			seconds = *check.TimeoutSeconds
		}
		if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < time.Duration(seconds)*time.Second+3*time.Second {
			err = errors.New("remaining run budget cannot cover the next technical check cap")
			break
		}
		var gates []host.GateResult
		gates, err = host.VerifySnapshotChecks(candidate, []authoring.Check{check})
		report.Checks = append(report.Checks, gates...)
		if err != nil {
			break
		}
		if err = ctx.Err(); err != nil {
			break
		}
	}
	if persistErr := persistJSON(filepath.Join(runDir, "checks.json"), report.Checks); persistErr != nil {
		return report, persistErr
	}
	if err != nil {
		return report, err
	}
	if err := confirmWorkspace(workspace, candidate); err != nil {
		return report, err
	}
	report.Stage = "independent-review"
	for _, area := range report.Plan.IntegrationReviews {
		scopes := append([]string{area.Key()}, subjects...)
		result, err := invoke("review", rt.Verifier, scopes, candidate, map[string]any{"candidate": material, "candidateCommit": report.CandidateCommit, "checks": report.Checks, "reviewArea": area})
		if err != nil {
			return report, err
		}
		if err := requireReview(result.Response, scopes); err != nil {
			report.Status = "blocked"
			return report, err
		}
	}
	resultDigests := []string{government.Digest(report.Checks)}
	for _, actor := range report.Actors {
		resultDigests = append(resultDigests, government.Digest(actor))
	}
	evidence, err := government.NewEvidence(material, 1, resultDigests, nil)
	if err != nil {
		return report, err
	}
	report.Evidence = &evidence
	if err := persistJSON(filepath.Join(runDir, "evidence.json"), evidence); err != nil {
		return report, err
	}
	report.Stage = "ressort-votes"
	for _, member := range report.Cabinet {
		spec := ressortSpec(rt, member.Ressort)
		result, err := invoke("vote", spec, []string{member.Ressort.Key()}, candidate, map[string]any{"candidate": material, "candidateCommit": report.CandidateCommit, "evidence": evidence, "checks": report.Checks, "member": member})
		if err != nil {
			return report, err
		}
		vote, err := validatedVote(material, evidence, member, result)
		if err != nil {
			report.Status = "blocked"
			return report, err
		}
		report.Votes = append(report.Votes, vote)
	}
	decision, err := government.NewAcceptanceDecision(material, evidence, model.Digest, report.Cabinet, report.Votes)
	if err != nil {
		report.Status = "blocked"
		return report, err
	}
	report.Decision = &decision
	if err := persistJSON(filepath.Join(runDir, "decision.json"), decision); err != nil {
		return report, err
	}
	report.Stage = "promote"
	if err := ctx.Err(); err != nil {
		return report, err
	}
	if err := confirmWorkspace(workspace, candidate); err != nil {
		return report, err
	}
	finalPins, err := toolPins(rt)
	if err != nil || finalPins != report.ToolPins {
		return report, errors.Join(errors.New("runtime/tool pins changed after material binding"), err)
	}
	promoted, err := government.Promote(ctx, government.PromotionRequest{Repo: repo, ActiveRef: rt.ActiveRef, ExpectedOld: rt.ExpectedBase, NewCommit: report.CandidateCommit, ExpectedTreeID: report.CandidateTree, MaterialCandidateID: material.ID, EvidenceID: evidence.ID, DecisionID: decision.ID, StateDirectory: runDir, IdempotencyKey: report.RunID})
	report.Promotion = &promoted
	if err != nil {
		return report, err
	}
	if promoted.Status != "promoted" {
		return report, errors.New("promotion did not establish the new active commit")
	}
	report.Stage = "complete"
	report.Status = "accepted-scoped"
	return report, nil
}

func toolPins(rt Runtime) (string, error) {
	pin, err := FingerprintRuntime(rt)
	if err != nil {
		return "", err
	}
	_, git, err := fingerprintExecutable("git")
	if err != nil {
		return "", err
	}
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	host, err := digestFile(exe)
	if err != nil {
		return "", err
	}
	return government.Digest(struct{ Runtime, Git, Host string }{pin, git, host}), nil
}

func requireReview(response agentexec.Response, scopes []string) error {
	if response.Outcome != agentexec.OutcomePassed || len(response.Uncertainty) > 0 {
		return errors.New("independent review did not explicitly pass without unresolved uncertainty")
	}
	seen := map[string]bool{}
	for _, observation := range response.VerifierObservations {
		if observation.Outcome != agentexec.OutcomePassed || seen[observation.Subject] || strings.TrimSpace(observation.Detail) == "" {
			return errors.New("review has rejected, duplicate or unexplained observations")
		}
		seen[observation.Subject] = true
	}
	for _, scope := range scopes {
		if !seen[scope] {
			return fmt.Errorf("independent review is missing required scope %s", scope)
		}
	}
	return nil
}

func definition(m government.Model, id core.DefinitionIdentity) (core.Definition, error) {
	for _, d := range m.Canonical.Definitions {
		if d.Identity().Key() == id.Key() {
			return d, nil
		}
	}
	return core.Definition{}, errors.New("missing prior definition")
}
func identityValue(value any) core.DefinitionIdentity {
	data, _ := json.Marshal(value)
	var id core.DefinitionIdentity
	_ = json.Unmarshal(data, &id)
	return id
}
func ressortSpec(rt Runtime, id core.DefinitionIdentity) RunnerSpec {
	for _, r := range rt.Ressorts {
		if r.Ressort.Key() == id.Key() {
			return r.Runner
		}
	}
	return RunnerSpec{}
}
func ressortRunners(rt Runtime) []RunnerSpec {
	out := make([]RunnerSpec, 0, len(rt.Ressorts))
	for _, configured := range rt.Ressorts {
		out = append(out, configured.Runner)
	}
	return out
}
func freezeCabinet(m government.Model, rt Runtime) ([]government.CabinetMember, error) {
	if len(rt.Ressorts) != len(m.Cabinet) {
		return nil, errors.New("runtime requires exactly every prior-Constitution cabinet Ressort")
	}
	out := []government.CabinetMember{}
	for _, id := range m.Cabinet {
		d, err := definition(m, id)
		if err != nil {
			return nil, err
		}
		mandateID := identityValue(d.Spec["mandate"])
		mandate, err := definition(m, mandateID)
		if err != nil {
			return nil, err
		}
		spec := ressortSpec(rt, id)
		if spec.SlotID == "" {
			return nil, errors.New("missing configured prior cabinet slot")
		}
		out = append(out, government.CabinetMember{Ressort: id, PriorMandate: mandateID, MandateDigest: government.Digest(mandate), SlotID: spec.SlotID})
	}
	return out, nil
}

type voteBody struct {
	Outcome             government.VoteOutcome `json:"outcome"`
	Reason              string                 `json:"reason"`
	MaterialCandidateID string                 `json:"materialCandidateId"`
	EvidenceID          string                 `json:"evidenceId"`
	Round               uint64                 `json:"round"`
}

func validatedVote(candidate government.MaterialCandidate, evidence government.Evidence, member government.CabinetMember, result agentexec.RunResult) (government.RessortVote, error) {
	response := result.Response
	if response.Outcome != agentexec.OutcomePassed || len(response.Uncertainty) > 0 || len(response.VerifierObservations) != 1 {
		return government.RessortVote{}, errors.New("Ressort did not provide one explicit final vote")
	}
	observation := response.VerifierObservations[0]
	if observation.Subject != "government-vote" || observation.Outcome != agentexec.OutcomePassed {
		return government.RessortVote{}, errors.New("Ressort vote observation is missing or rejected")
	}
	if err := rejectRuntimeDuplicateKeys([]byte(observation.Detail)); err != nil {
		return government.RessortVote{}, err
	}
	decoder := json.NewDecoder(strings.NewReader(observation.Detail))
	decoder.DisallowUnknownFields()
	var body voteBody
	if err := decoder.Decode(&body); err != nil {
		return government.RessortVote{}, err
	}
	if body.MaterialCandidateID != candidate.ID || body.EvidenceID != evidence.ID || body.Round != evidence.Round {
		return government.RessortVote{}, errors.New("Ressort vote binds stale or foreign candidate/evidence/round")
	}
	return government.NewRessortVote(candidate, evidence, member.PriorMandate, member.MandateDigest, member.Ressort, body.Outcome, body.Reason, government.VoteProvenance{RunID: result.Receipt.RunID, SlotID: member.SlotID})
}
