// Package execution implements the bounded Government execution lifecycle. It
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
	"github.com/Glacius-Labs/Markitect/internal/host/government"
	"github.com/Glacius-Labs/Markitect/internal/host/government/inventory"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

type Options struct {
	Repo, ConfigPath, OrderPath string
	Runtime                     Runtime
}
type ActorRecord struct {
	Sequence   int                 `json:"sequence"`
	StartedAt  time.Time           `json:"startedAt"`
	FinishedAt time.Time           `json:"finishedAt"`
	Phase      string              `json:"phase"`
	SlotID     string              `json:"slotId"`
	Scopes     []string            `json:"scopes"`
	Result     agentexec.RunResult `json:"result"`
	Error      string              `json:"error,omitempty"`
}
type Report struct {
	Delegation        *government.DelegationPlan     `json:"delegation,omitempty"`
	RootArea          *AreaReport                    `json:"rootArea,omitempty"`
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
		"Recursive execution is opt-in and bounded; amendment activation, queue and complete recovery remain unavailable.",
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
		return report, errors.New("execution only implements under unchanged prior authority; amendments require G4")
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
	if rt.Recursion == nil && (len(report.Plan.Work) != 1 || len(report.Plan.Work[0].Paths) == 0) {
		return report, errors.New("G2 requires exactly one nonempty Writer Area; recursive execution remains G3")
	}
	maxReviews := 128
	if rt.Recursion != nil {
		maxReviews = maxRuntimeAreas + 1
	}
	if len(report.Plan.IntegrationReviews) == 0 || len(report.Plan.IntegrationReviews) > maxReviews {
		return report, errors.New("integration review scopes exceed the configured execution boundary")
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
	for _, spec := range configuredRunners(rt) {
		pin, err := fingerprintRunner(spec)
		if err != nil {
			return report, err
		}
		frozenRunners[spec.SlotID] = pin
	}
	session := &actorSession{ctx: ctx, runtime: rt, report: &report, runDir: runDir, temporary: temp, frozen: frozenRunners, constitution: model.Constitution.Key()}
	if rt.Recursion != nil {
		session.semaphore = make(chan struct{}, rt.Recursion.Parallelism)
	}
	selectedPaths := []string{}
	for _, work := range report.Plan.Work {
		for _, path := range work.Paths {
			if path == opts.ConfigPath || path == opts.OrderPath {
				return report, errors.New("implementation order cannot write constitutional or order input")
			}
			selectedPaths = append(selectedPaths, path)
		}
	}
	subjects := identityKeys(report.Plan.Affected)
	invoke := func(phase string, spec RunnerSpec, scopes []string, s *snapshot.Snapshot, extra map[string]any) (agentexec.RunResult, error) {
		contextValue := map[string]any{"phase": phase, "runId": report.RunID, "order": order, "plan": report.Plan, "priorModel": model.Canonical, "cabinet": report.Cabinet, "allowedPaths": selectedPaths, "inputPaths": selectedPaths, "workspace": workspace}
		if rt.Recursion != nil {
			contextValue["area"] = model.Root
			contextValue["rootArea"] = report.RootArea
			if report.RootArea != nil && len(report.RootArea.Attempts) > 0 {
				latest := report.RootArea.Attempts[len(report.RootArea.Attempts)-1]
				contextValue["childReports"] = latest.Children
				contextValue["attempt"] = latest.Attempt
				contextValue["repair"] = latest.Repair
			}
			contextValue["delegationDigest"] = report.Delegation.Digest
			contextValue["allowedPaths"] = report.Delegation.Root.Work.Paths
		}
		for k, v := range extra {
			contextValue[k] = v
		}
		return session.invoke(phase, spec, scopes, s, workspace, selectedPaths, contextValue)
	}
	var candidate *snapshot.Snapshot
	if rt.Recursion != nil {
		delegation := government.BuildDelegationPlan(model, report.Plan, rt.Recursion.Limits)
		report.Delegation = &delegation
		if delegation.Status == "blocked" {
			report.Status = "blocked"
			return report, errors.New("recursive prior-authority delegation is blocked")
		}
		if delegation.EstimatedCalls+len(report.Cabinet)+1 > rt.Recursion.Limits.MaxCalls {
			return report, errors.New("invocation budget cannot cover initial tree, final Root review and frozen cabinet")
		}
		if err := validateNodeWireBounds(delegation.Root); err != nil {
			return report, err
		}
		if err := validateAreaAssignments(delegation.Root, rt); err != nil {
			return report, err
		}
		if err := persistJSON(filepath.Join(runDir, "delegation.json"), delegation); err != nil {
			return report, err
		}
		report.Stage = "recursive-execution"
		engine := recursiveEngine{session: session, model: model, order: order, delegation: delegation}
		var rootArea AreaReport
		candidate, rootArea, err = engine.node(delegation.Root, base, "")
		report.RootArea = &rootArea
		sort.Slice(report.Actors, func(i, j int) bool { return report.Actors[i].Sequence < report.Actors[j].Sequence })
		if err != nil {
			if rootArea.Status == "blocked" {
				report.Status = "blocked"
			}
			return report, err
		}
	} else {
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
		candidate, err = applyProposal(base, selectedPaths, executed.Response.CandidateFiles)
		if err != nil {
			return report, err
		}
	}
	report.ChangedPaths = changedPaths(base, candidate)
	if len(report.ChangedPaths) == 0 {
		return report, errors.New("executor proposal made no actual material change")
	}
	if err := writeChanges(workspace, base, candidate); err != nil {
		return report, err
	}
	report.CandidateCommit, report.CandidateTree, err = commitCandidate(ctx, repo, rt.ExpectedBase, report.RunID, runDir, report.ChangedPaths, candidate)
	if err != nil {
		return report, err
	}
	candidate.ID = report.CandidateCommit
	material, err := government.NewMaterialCandidate(government.MaterialCandidateInput{PriorConstitutionDigest: model.Digest, BaseRevision: rt.ExpectedBase, RepositoryTreeDigest: government.Digest(struct{ Tree, Snapshot string }{report.CandidateTree, candidate.Digest()}), ModelDigest: model.Digest, PlanDigest: report.Plan.Digest, CheckDefinitionsDigest: checkDefinitionsDigest(rt), ToolPinsDigest: report.ToolPins, InventoryDigest: observation.Digest})
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
	report.Checks, err = freshChecks(ctx, candidate, rt.Checks)
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
	reviewAreas := report.Plan.IntegrationReviews
	if rt.Recursion != nil {
		reviewAreas = []core.DefinitionIdentity{model.Root}
	}
	for _, area := range reviewAreas {
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
	if report.RootArea != nil {
		resultDigests = append(resultDigests, government.Digest(report.RootArea), report.Delegation.Digest)
	}
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
