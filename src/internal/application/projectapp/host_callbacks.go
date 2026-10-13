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

	"github.com/Glacius-Labs/Markitect/src/internal/application/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projectexplore"
	"github.com/Glacius-Labs/Markitect/src/internal/application/projecthistory"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectbriefing"
	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectrun"
)

func prepareBriefings(root, revision string, persist bool) (*projectbriefing.Store, error) {
	if persist {
		_, err := projecthistory.EnsureAcceptedHistory(root, revision)
		return nil, err
	}
	history, err := projecthistory.ReadAcceptedHistory(root, revision)
	if err != nil {
		return nil, err
	}
	return &history.State, nil
}

func briefingForManager(root, revision, modelDigest, managerID string, view *projectbriefing.Store) (projectrun.BriefingContext, error) {
	state := projectbriefing.Store{}
	if view != nil {
		state = *view
	} else {
		read, err := projecthistory.Read(root)
		if err != nil {
			return projectrun.BriefingContext{}, err
		}
		state = read.Accepted
	}
	if len(state.Briefings) == 0 {
		return projectrun.BriefingContext{Briefings: []projectbriefing.Briefing{}, Events: []projectbriefing.Event{}}, nil
	}
	briefings, events, binding, err := projecthistory.LoadForManagerFrom(root, state, modelDigest, managerID, revision)
	if err != nil {
		return projectrun.BriefingContext{}, fmt.Errorf("record accepted model briefing before work on this model: %w", err)
	}
	return projectrun.BriefingContext{Digest: binding, Briefings: briefings, Events: events}, nil
}

func loadDeliveryScope(root, explorationID, scopeID string) (projectrun.DeliveryScope, error) {
	record, err := projectexplore.Load(root, explorationID)
	if err != nil {
		return projectrun.DeliveryScope{}, err
	}
	for _, scope := range record.Scopes {
		if scope.ID == scopeID {
			return deliveryScope(record, scope), nil
		}
	}
	return projectrun.DeliveryScope{ExplorationID: record.ID, ScopeID: scopeID, Name: scopeID}, nil
}

func loadExplorationReadiness(host projectrun.Host, root, explorationID, scopeID string) (projectrun.DeliveryScope, projectvalue.Binding, projectrun.ExplorationReadiness, error) {
	record, scope, err := loadExplorationScope(root, explorationID, scopeID)
	if err != nil {
		return projectrun.DeliveryScope{}, projectvalue.Binding{}, projectrun.ExplorationReadiness{}, err
	}
	binding, err := projectrun.ExplorationBinding(host, root, "", projectrun.PlanRequest{
		ExplorationID: record.ID, ScopeID: scope.ID, Goal: scope.Goal, Operation: scope.Operation,
		Managers: append([]string(nil), scope.ManagerIDs...),
	})
	if err != nil {
		return projectrun.DeliveryScope{}, projectvalue.Binding{}, projectrun.ExplorationReadiness{}, err
	}
	readiness, err := projectexplore.EvaluateReadiness(record, scope.ID, binding)
	if err != nil {
		return projectrun.DeliveryScope{}, projectvalue.Binding{}, projectrun.ExplorationReadiness{}, err
	}
	return deliveryScope(record, scope), binding, projectrun.ExplorationReadiness{
		Ready: readiness.Ready, Digest: readiness.Digest, Blockers: append([]string{}, readiness.Blockers...),
	}, nil
}

func bindExploration(root string, binding projectvalue.Binding, request projectrun.PlanRequest, plan projectrun.PlanRecord) (projectrun.ExplorationAdmission, error) {
	if request.ExplorationID == "" || request.ScopeID == "" {
		return projectrun.ExplorationAdmission{}, errors.New("guided implementation requires a named exploration scope")
	}
	record, _, err := loadExplorationScope(root, request.ExplorationID, request.ScopeID)
	if err != nil {
		return projectrun.ExplorationAdmission{}, err
	}
	readiness, err := projectexplore.EvaluateReadiness(record, request.ScopeID, binding)
	if err != nil {
		return projectrun.ExplorationAdmission{}, err
	}
	if !readiness.Ready {
		return projectrun.ExplorationAdmission{}, fmt.Errorf("scope is not ready: %s", strings.Join(readiness.Blockers, "; "))
	}
	return projectrun.ExplorationAdmission{Binding: binding, ExplorationDigest: record.Digest, ReadinessDigest: readiness.Digest}, nil
}

func validateExploration(root string, plan projectrun.PlanRecord) error {
	if plan.ExplorationID == "" && plan.ScopeID == "" && plan.ExplorationBinding == nil {
		return nil
	}
	if plan.ExplorationID == "" || plan.ScopeID == "" || plan.ExplorationBinding == nil {
		return errors.New("incomplete exploration plan binding")
	}
	record, _, err := loadExplorationScope(root, plan.ExplorationID, plan.ScopeID)
	if err != nil {
		return err
	}
	if record.Digest != plan.ExplorationDigest {
		return fmt.Errorf("%w: exploration changed after planning", projectrun.ErrStale)
	}
	readiness, err := projectexplore.EvaluateReadiness(record, plan.ScopeID, *plan.ExplorationBinding)
	if err != nil {
		return err
	}
	if !readiness.Ready || readiness.Digest != plan.ReadinessDigest {
		return fmt.Errorf("%w: exploration readiness changed", projectrun.ErrStale)
	}
	return nil
}

func explorationCapturePaths(plan projectrun.PlanRecord) ([]string, error) {
	if plan.ExplorationID == "" {
		return []string{}, nil
	}
	path, err := projectexplore.RecordPath(plan.ExplorationID)
	if err != nil {
		return nil, err
	}
	return []string{path}, nil
}

func completeExploration(root string, completion projectrun.ExplorationCompletion) error {
	plan := completion.Plan
	if plan.ExplorationID == "" {
		return nil
	}
	if plan.ExplorationBinding == nil {
		return errors.New("exploration completion has no fixed binding")
	}
	bindingDigest, err := projectvalue.BindingDigest(*plan.ExplorationBinding)
	if err != nil {
		return err
	}
	structureDigest, err := projectvalue.StructureDigest(*plan.ExplorationBinding)
	if err != nil {
		return err
	}
	applyDigest, err := applicationEvidenceDigest(completion.Apply)
	if err != nil {
		return err
	}
	_, err = projectexplore.Complete(root, plan.ExplorationID, plan.ScopeID, *plan.ExplorationBinding, projectexplore.ApplyReceipt{
		ScopeID: plan.ScopeID, BindingDigest: bindingDigest, StructureDigest: structureDigest, Status: projectrun.StatusApplied,
		RunID: plan.ID, PlanID: plan.ID, PlanDigest: plan.Digest, CandidateID: completion.CandidateID, CandidateDigest: completion.CandidateDigest,
		VerificationID: completion.Verification.Digest, VerificationDigest: completion.Verification.Digest, VerificationStatus: "passed", ApplyID: applyDigest, ApplyDigest: applyDigest, AppliedAt: completion.Apply.AppliedAt,
	})
	return err
}

func resolveVerifiedBriefing(root string, evidence projectrun.DeliveredResolution) error {
	read, err := projecthistory.Read(root)
	if err != nil {
		return err
	}
	state := read.Accepted
	if state.History == nil {
		return nil
	}
	eventIDs := make([]string, 0)
	for _, bundle := range state.Briefings {
		for _, event := range bundle.Events {
			if projectbriefing.EventResolutionStatus(read.StatusView, event.ID).Status == projectbriefing.StatusResolved {
				continue
			}
			if _, err := source.GitOutput(root, "merge-base", "--is-ancestor", bundle.Revision, evidence.Plan.BaseRevision); err == nil {
				eventIDs = append(eventIDs, event.ID)
			}
		}
	}
	if len(eventIDs) == 0 {
		return nil
	}
	full := evidence.Verification.ManagerVerification
	if full == nil {
		return errors.New("resolution evidence requires full Manager verification")
	}
	managers := make([]string, 0, len(full.Managers))
	for _, manager := range full.Managers {
		if manager.Status != "passed" {
			return errors.New("incomplete full Manager evidence cannot resolve events")
		}
		managers = append(managers, manager.ManagerID)
	}
	applyDigest, err := applicationEvidenceDigest(evidence.Apply)
	if err != nil {
		return err
	}
	delivered := make([]projectbriefing.DeliveredFile, 0, len(evidence.Apply.Written))
	for _, path := range evidence.Apply.Written {
		file, ok := evidence.Files[path]
		if !ok {
			return fmt.Errorf("applied path %s is not in the verified candidate", path)
		}
		item, err := projecthistory.NewDeliveredFile(root, path, file.Mode, file.Content, file.Delete)
		if err != nil {
			return err
		}
		delivered = append(delivered, item)
	}
	_, err = projecthistory.ResolveVerified(root, evidence.Plan.BaseRevision, evidence.Plan.ModelDigest, projectbriefing.VerifiedResolutionEvidence{
		EventIDs: sortedUnique(eventIDs), RunID: evidence.Plan.ID, PlanDigest: evidence.Plan.Digest, CandidateID: evidence.CandidateID, CandidateDigest: evidence.CandidateDigest,
		VerificationDigest: evidence.Verification.Digest, ApplyDigest: applyDigest, FullVerifyPassed: true, CoveredManagerIDs: sortedUnique(managers),
		EvidenceRefs: sortedUnique([]string{"run:" + evidence.Plan.ID, "full-verify:" + full.Digest, "verify:" + evidence.Verification.Digest, "apply:" + applyDigest}), Delivered: delivered,
	}, read.PersistedDigest)
	return err
}

func loadExplorationScope(root, explorationID, scopeID string) (projectexplore.Record, projectexplore.Scope, error) {
	record, err := projectexplore.Load(root, explorationID)
	if err != nil {
		return projectexplore.Record{}, projectexplore.Scope{}, err
	}
	for _, scope := range record.Scopes {
		if scope.ID == scopeID {
			return record, scope, nil
		}
	}
	return projectexplore.Record{}, projectexplore.Scope{}, fmt.Errorf("exploration %s has no scope %q", record.ID, scopeID)
}

func deliveryScope(record projectexplore.Record, scope projectexplore.Scope) projectrun.DeliveryScope {
	return projectrun.DeliveryScope{
		ExplorationID: record.ID, ScopeID: scope.ID, Name: scope.Name, Goal: scope.Goal,
		Operation: scope.Operation, ManagerIDs: append([]string{}, scope.ManagerIDs...),
	}
}

func applicationEvidenceDigest(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode delivery evidence: %w", err)
	}
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:]), nil
}

func sortedUnique(values []string) []string {
	result := append([]string(nil), values...)
	sort.Strings(result)
	if len(result) == 0 {
		return []string{}
	}
	out := result[:1]
	for _, value := range result[1:] {
		if value != out[len(out)-1] {
			out = append(out, value)
		}
	}
	return out
}

// Prepared facts are rebuilt from the current Session, accepted target and
// target Runtime at each finite Runtime boundary. Supplied paths/context are
// never treated as Application authority.
func (o Operations) brownfieldStageHost(sourceRoot string, operation BrownfieldRunOperation, invoker projectadoption.ManagerRunInvoker) projectrun.Host {
	host := o.Host
	host.ValidateManagerStage = func(binding projectrun.ManagerStageValidationBinding) error {
		runtime, err := projectconfig.LoadRuntime(operation.Root)
		if err != nil {
			return err
		}
		agent, err := projectconfig.ReadOnlyAgent(runtime, operation.Request.AgentManagerID)
		if err != nil {
			return err
		}
		config, err := agent.AgentConfig()
		if err != nil {
			return err
		}
		limits := projectadoption.ManagerRunLimits{MaxStarts: runtime.Limits.MaxStarts, MaxRetries: runtime.Limits.MaxRetries, MaxDuration: time.Duration(runtime.Limits.MaxDuration), MaxCostMicros: runtime.Limits.MaxCostMicros, InputPriceMicrosPerMillion: agent.Pricing.InputMicrosPerMillion, OutputPriceMicrosPerMillion: agent.Pricing.OutputMicrosPerMillion, MaxTimeout: config.Timeout, MaxStdoutBytes: config.MaxStdoutBytes, MaxStderrBytes: config.MaxStderrBytes}
		_, spec, _, _, err := o.prepareBrownfieldRuntimeStage(sourceRoot, operation, runtime, agent, config, limits, binding.SessionDigest, invoker)
		if err != nil {
			return err
		}
		if brownfieldStageValueDigest(spec.Validation) != brownfieldStageValueDigest(binding) || spec.Validation.SourceRoot != binding.SourceRoot || spec.Validation.TargetRoot != binding.TargetRoot {
			return errors.New("prepared Brownfield facts changed before Runtime acceptance")
		}
		return nil
	}
	return host
}
