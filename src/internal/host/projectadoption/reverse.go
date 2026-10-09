package projectadoption

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
)

type ManagerReverseEvidence struct {
	EvidenceID     string `json:"evidenceId"`
	Path           string `json:"path"`
	Basis          string `json:"basis"`
	Classification string `json:"classification"` // observed-implementation, documented-intent, submitted-runtime-record, or selected-evidence
	Digest         string `json:"digest"`
	Content        string `json:"content"`
}

type ManagerReverseEvidenceMetadata struct {
	EvidenceID     string `json:"evidenceId"`
	Path           string `json:"path"`
	Basis          string `json:"basis"`
	Classification string `json:"classification"`
	Digest         string `json:"digest"`
}

type ManagerReverseContract struct {
	Classification string                     `json:"classification"` // accepted-desired-contract
	Contract       DistillationTargetContract `json:"contract"`
}

type ManagerReverseProposalContract struct {
	Classification string                `json:"classification"` // manager-proposed-public-contract
	ManagerID      string                `json:"managerId"`
	ProposalDigest string                `json:"proposalDigest"`
	Contract       ManagerPublicContract `json:"contract"`
}

// ManagerReverseContext contains the Manager's own raw evidence, authorized
// delegation metadata, and only adjacent public contracts. Root inventory is
// metadata-only and never grants assignment authority by itself.
type ManagerReverseContext struct {
	APIVersion                string                           `json:"apiVersion"`
	SessionDigest             string                           `json:"sessionDigest"`
	DiscoveryDigest           string                           `json:"discoveryDigest"`
	TargetContextDigest       string                           `json:"targetContextDigest"`
	IterationID               string                           `json:"iterationId"`
	Manager                   DistillationTargetManager        `json:"manager"`
	ManagerOrigin             string                           `json:"managerOrigin"` // accepted-target or proposed-by-parent
	Evidence                  []ManagerReverseEvidence         `json:"evidence"`
	DelegationEvidence        []ManagerReverseEvidenceMetadata `json:"delegationEvidence"`
	DiscoveryInventory        []ManagerReverseEvidenceMetadata `json:"discoveryInventory,omitempty"`
	PublicNeighborContracts   []ManagerReverseContract         `json:"publicNeighborContracts"`
	ProposedNeighborContracts []ManagerReverseProposalContract `json:"proposedNeighborContracts"`
	Digest                    string                           `json:"digest"`
}

const ManagerReverseContextVersion = "markitect.example.org/manager-reverse-context/v1alpha2"

// BuildManagerReverseContext creates a bounded prompt artifact. It does not
// invoke an agent. A root receives the full selected-evidence inventory as
// metadata only; non-root Managers see only their own and delegated metadata.
func BuildManagerReverseContext(session BrownfieldSession, iterationID string) (ManagerReverseContext, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return ManagerReverseContext{}, err
	}
	iteration, ok := findIteration(session, iterationID)
	if !ok {
		return ManagerReverseContext{}, errors.New("unknown reverse iteration")
	}
	manager, accepted := targetManager(session.TargetContext, iteration.ManagerID)
	managerOrigin := "accepted-target"
	if !accepted {
		parent, exists := findIteration(session, iteration.ParentIterationID)
		candidate, proposed := proposedManager(parent, iteration.ManagerID)
		if !exists || !proposed {
			return ManagerReverseContext{}, errors.New("iteration Manager is absent from accepted and proposed hierarchy")
		}
		owns := []string{}
		for _, id := range candidate.EvidenceIDs {
			for _, evidence := range session.Source.Evidence {
				if evidence.ID == id {
					owns = append(owns, evidence.Path)
				}
			}
		}
		manager = DistillationTargetManager{ID: candidate.ID, Name: candidate.Name, Namespace: candidate.ID, Purpose: candidate.Purpose, Parent: candidate.ParentID, Owns: owns}
		managerOrigin = "proposed-by-parent"
	}
	result := ManagerReverseContext{APIVersion: ManagerReverseContextVersion, SessionDigest: session.Digest, DiscoveryDigest: session.Source.Digest,
		TargetContextDigest: iteration.TargetContextDigest, IterationID: iteration.ID, Manager: manager, ManagerOrigin: managerOrigin,
		Evidence: []ManagerReverseEvidence{}, DelegationEvidence: []ManagerReverseEvidenceMetadata{}, DiscoveryInventory: []ManagerReverseEvidenceMetadata{},
		PublicNeighborContracts: []ManagerReverseContract{}, ProposedNeighborContracts: []ManagerReverseProposalContract{}}
	selected := map[string]bool{}
	for _, id := range iteration.EvidenceIDs {
		selected[id] = true
	}
	delegated := map[string]bool{}
	for _, id := range iteration.DelegationEvidenceIDs {
		delegated[id] = true
	}
	for _, evidence := range session.Source.Evidence {
		classification := evidenceClassification(evidence.Basis)
		if selected[evidence.ID] {
			result.Evidence = append(result.Evidence, ManagerReverseEvidence{EvidenceID: evidence.ID, Path: evidence.Path, Basis: evidence.Basis,
				Classification: classification, Digest: evidence.Digest, Content: evidence.Content})
		}
		if delegated[evidence.ID] {
			result.DelegationEvidence = append(result.DelegationEvidence, ManagerReverseEvidenceMetadata{EvidenceID: evidence.ID, Path: evidence.Path, Basis: evidence.Basis,
				Classification: classification, Digest: evidence.Digest})
		}
		if iteration.ParentIterationID == "" {
			result.DiscoveryInventory = append(result.DiscoveryInventory, ManagerReverseEvidenceMetadata{EvidenceID: evidence.ID, Path: evidence.Path, Basis: evidence.Basis,
				Classification: classification, Digest: evidence.Digest})
		}
	}
	acceptedAncestor := manager.ID
	for {
		if _, ok := targetManager(session.TargetContext, acceptedAncestor); ok {
			break
		}
		candidate, ok := anyProposedManager(session, acceptedAncestor)
		if !ok || candidate.ParentID == "" {
			acceptedAncestor = session.TargetContext.RootManagerID
			break
		}
		acceptedAncestor = candidate.ParentID
	}
	if _, ok := targetManager(session.TargetContext, acceptedAncestor); !ok {
		acceptedAncestor = session.TargetContext.RootManagerID
	}
	allowedOwners := adjacentManagerIDs(session.TargetContext, acceptedAncestor)
	for _, contract := range session.TargetContext.PublicContracts {
		if allowedOwners[contract.Owner] {
			result.PublicNeighborContracts = append(result.PublicNeighborContracts, ManagerReverseContract{Classification: "accepted-desired-contract", Contract: contract})
		}
	}
	allowedProposed := proposedNeighborIDs(session, iteration, manager)
	for _, prior := range session.Iterations {
		if !allowedProposed[prior.ManagerID] || prior.Proposal == nil {
			continue
		}
		for _, contract := range prior.Proposal.PublicContracts {
			result.ProposedNeighborContracts = append(result.ProposedNeighborContracts, ManagerReverseProposalContract{Classification: "manager-proposed-public-contract", ManagerID: prior.ManagerID, ProposalDigest: prior.Proposal.Digest, Contract: contract})
		}
	}
	sort.Slice(result.ProposedNeighborContracts, func(i, j int) bool {
		if result.ProposedNeighborContracts[i].ManagerID != result.ProposedNeighborContracts[j].ManagerID {
			return result.ProposedNeighborContracts[i].ManagerID < result.ProposedNeighborContracts[j].ManagerID
		}
		if result.ProposedNeighborContracts[i].ProposalDigest != result.ProposedNeighborContracts[j].ProposalDigest {
			return result.ProposedNeighborContracts[i].ProposalDigest < result.ProposedNeighborContracts[j].ProposalDigest
		}
		return result.ProposedNeighborContracts[i].Contract.Contract.ID < result.ProposedNeighborContracts[j].Contract.Contract.ID
	})
	copy := result
	copy.Digest = ""
	result.Digest = digestValue(copy)
	return result, nil
}

func proposedNeighborIDs(session BrownfieldSession, iteration ReverseIteration, manager DistillationTargetManager) map[string]bool {
	allowed := map[string]bool{manager.ID: true}
	parentID := manager.Parent
	if iteration.ParentIterationID != "" {
		if parent, ok := findIteration(session, iteration.ParentIterationID); ok {
			parentID = parent.ManagerID
			allowed[parentID] = true
			for _, item := range parent.Proposal.Hierarchy {
				allowed[item.ID] = true
			}
		}
	}
	if parentID != "" {
		allowed[parentID] = true
	}
	for _, candidate := range session.TargetContext.Managers {
		if candidate.Parent == parentID {
			allowed[candidate.ID] = true
		}
	}
	for _, item := range session.Iterations {
		if item.ManagerID == manager.ID && item.Proposal != nil {
			for _, child := range item.Proposal.Hierarchy {
				allowed[child.ID] = true
			}
		}
	}
	return allowed
}

func adjacentManagerIDs(context DistillationTargetContext, managerID string) map[string]bool {
	managers := map[string]DistillationTargetManager{}
	for _, manager := range context.Managers {
		managers[manager.ID] = manager
	}
	current := managers[managerID]
	allowed := map[string]bool{managerID: true}
	if current.Parent != "" {
		allowed[current.Parent] = true
	}
	for _, candidate := range context.Managers {
		if candidate.Parent == managerID {
			allowed[candidate.ID] = true
		}
		if current.Parent != "" && candidate.Parent == current.Parent {
			allowed[candidate.ID] = true
		}
	}
	return allowed
}

func proposedManager(iteration ReverseIteration, id string) (ProposedManager, bool) {
	if iteration.Proposal == nil {
		return ProposedManager{}, false
	}
	for _, candidate := range iteration.Proposal.Hierarchy {
		if candidate.ID == id {
			return candidate, true
		}
	}
	return ProposedManager{}, false
}
func anyProposedManager(session BrownfieldSession, id string) (ProposedManager, bool) {
	for _, iteration := range session.Iterations {
		if candidate, ok := proposedManager(iteration, id); ok {
			return candidate, true
		}
	}
	return ProposedManager{}, false
}

// BeginReverseIteration appends a fixed, evidence-selected Manager work item.
// The Manager identity must belong to the accepted target tree and a child
// iteration can only be routed beneath its declared parent.
func BeginReverseIteration(sourceRoot string, target *projectwork.Project, session BrownfieldSession, request ReverseIterationRequest) (BrownfieldSession, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return BrownfieldSession{}, err
	}
	if _, err := RefreshDiscovery(sourceRoot, session.Source); err != nil {
		return BrownfieldSession{}, fmt.Errorf("source basis is stale: %w", err)
	}
	if target == nil || target.Digest != session.Target.ProjectDigest || target.Revision != session.Target.Revision || !sameFilesystemPath(target.Root, session.Target.Root) {
		return BrownfieldSession{}, errors.New("reverse iteration target differs from the session's fixed target basis")
	}
	if !validID(request.ID) || strings.TrimSpace(request.ManagerID) == "" || strings.TrimSpace(request.Purpose) == "" || strings.TrimSpace(request.Review) == "" {
		return BrownfieldSession{}, errors.New("reverse iteration requires stable ID, accepted Manager, purpose, and review reference")
	}
	if _, err := sortedUniqueStrings(request.EvidenceIDs); err != nil || len(request.EvidenceIDs) == 0 {
		return BrownfieldSession{}, errors.New("reverse iteration requires unique selected evidence IDs")
	}
	if err := validateEvidencePools(request.EvidenceIDs, request.DelegationEvidenceIDs, session.Source); err != nil {
		return BrownfieldSession{}, err
	}
	manager, accepted := targetManager(session.TargetContext, request.ManagerID)
	if !accepted && request.ParentIterationID == "" {
		return BrownfieldSession{}, fmt.Errorf("root Manager %q is not in the accepted target hierarchy", request.ManagerID)
	}
	if request.ParentIterationID == "" {
		if request.ManagerID != session.TargetContext.RootManagerID {
			return BrownfieldSession{}, errors.New("a root reverse iteration must use the accepted root Manager")
		}
		priorRoot := ""
		for i := len(session.Iterations) - 1; i >= 0; i-- {
			if session.Iterations[i].ParentIterationID == "" {
				priorRoot = session.Iterations[i].ID
				break
			}
		}
		if priorRoot == "" && request.SupersedesIterationID != "" || priorRoot != "" && request.SupersedesIterationID != priorRoot {
			return BrownfieldSession{}, errors.New("a repeated root iteration must explicitly supersede the latest root pass")
		}
		if priorRoot != "" {
			prior, _ := findIteration(session, priorRoot)
			if prior.Integration == nil {
				return BrownfieldSession{}, errors.New("root Manager must integrate its current children before starting another reverse pass")
			}
		}
	} else {
		if request.SupersedesIterationID != "" {
			return BrownfieldSession{}, errors.New("child iterations cannot supersede a root pass")
		}
		parent, exists := findIteration(session, request.ParentIterationID)
		if !exists || parent.Proposal == nil {
			return BrownfieldSession{}, errors.New("child reverse iteration requires a parent Manager proposal that established its responsibility")
		}
		assignment, proposed := proposedManager(parent, request.ManagerID)
		if !proposed || assignment.ParentID != parent.ManagerID || !sameStrings(assignment.EvidenceIDs, request.EvidenceIDs) || !sameStrings(assignment.DelegationEvidenceIDs, request.DelegationEvidenceIDs) {
			return BrownfieldSession{}, errors.New("child Manager and exact evidence assignment must be explicitly listed by its parent")
		}
		available := iterationAvailableEvidence(parent)
		for _, id := range append(append([]string(nil), request.EvidenceIDs...), request.DelegationEvidenceIDs...) {
			if !available[id] {
				return BrownfieldSession{}, fmt.Errorf("child Manager evidence %q is outside the parent's own and delegated evidence", id)
			}
		}
		if accepted {
			if manager.Parent != parent.ManagerID || manager.Name != assignment.Name || manager.Purpose != assignment.Purpose {
				return BrownfieldSession{}, errors.New("accepted child Manager assignment differs from the fixed target hierarchy")
			}
		} else if !validID(request.ManagerID) {
			return BrownfieldSession{}, fmt.Errorf("proposed child Manager ID %q is invalid", request.ManagerID)
		}
	}
	for _, item := range session.Iterations {
		if request.ParentIterationID != "" && item.ParentIterationID == request.ParentIterationID && item.ManagerID == request.ManagerID {
			return BrownfieldSession{}, fmt.Errorf("parent iteration %q already has a child iteration for Manager %q", request.ParentIterationID, request.ManagerID)
		}
		if item.ID == request.ID {
			return BrownfieldSession{}, fmt.Errorf("reverse iteration %q already exists", request.ID)
		}
	}
	request.EvidenceIDs, _ = sortedUniqueStrings(request.EvidenceIDs)
	request.DelegationEvidenceIDs, _ = sortedUniqueStrings(request.DelegationEvidenceIDs)
	if request.DelegationEvidenceIDs == nil {
		request.DelegationEvidenceIDs = []string{}
	}
	result := cloneSession(session)
	result.Iterations = append(result.Iterations, ReverseIteration{ID: request.ID, ParentIterationID: request.ParentIterationID, SupersedesIterationID: request.SupersedesIterationID, ManagerID: request.ManagerID,
		EvidenceIDs: request.EvidenceIDs, DelegationEvidenceIDs: request.DelegationEvidenceIDs,
		Purpose: request.Purpose, Review: request.Review, TargetContextDigest: session.TargetContext.Digest})
	sealSession(&result)
	return result, nil
}

// RecordManagerProposal attaches one externally supplied proposal. It checks
// that every cited evidence item was explicitly selected for this Manager.
func RecordManagerProposal(session BrownfieldSession, iterationID string, proposal ManagerProposal) (BrownfieldSession, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return BrownfieldSession{}, err
	}
	result := cloneSession(session)
	index := iterationIndex(result, iterationID)
	if index < 0 {
		return BrownfieldSession{}, errors.New("unknown reverse iteration")
	}
	iteration := result.Iterations[index]
	if iteration.Proposal != nil {
		return BrownfieldSession{}, errors.New("Manager proposal is immutable once recorded")
	}
	if err := validateManagerProposal(proposal, iteration, session); err != nil {
		return BrownfieldSession{}, err
	}
	copy := proposal
	copy.Digest = ""
	proposal.Digest = digestValue(copy)
	result.Iterations[index].Proposal = &proposal
	setScopeStatesFromReport(&result, proposal.Report, "modeled")
	sealSession(&result)
	return result, nil
}

// IntegrateManagerProposal binds child public proposal digests into a parent
// report. Conflicts remain explicit until an owner resolution records them.
func IntegrateManagerProposal(session BrownfieldSession, iterationID, managerID string, integration ManagerIntegration) (BrownfieldSession, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return BrownfieldSession{}, err
	}
	result := cloneSession(session)
	index := iterationIndex(result, iterationID)
	if index < 0 {
		return BrownfieldSession{}, errors.New("unknown reverse iteration")
	}
	iteration := result.Iterations[index]
	if iteration.ManagerID != managerID || iteration.Proposal == nil || iteration.Integration != nil {
		return BrownfieldSession{}, errors.New("integration must belong to the proposal's Manager and be recorded once")
	}
	if err := validateManagerIntegration(integration, iteration, result); err != nil {
		return BrownfieldSession{}, err
	}
	copy := integration
	copy.Digest = ""
	integration.Digest = digestValue(copy)
	result.Iterations[index].Integration = &integration
	for _, conflict := range integration.Conflicts {
		if conflict.Disposition == "unresolved" {
			setOneScopeStatus(&result, conflict.ScopeID, "unresolved", conflict.Description)
		}
	}
	sealSession(&result)
	return result, nil
}

func RecordSessionResolution(session BrownfieldSession, iterationID string, resolution Resolution) (BrownfieldSession, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return BrownfieldSession{}, err
	}
	result := cloneSession(session)
	index := iterationIndex(result, iterationID)
	if index < 0 || result.Iterations[index].Integration == nil {
		return BrownfieldSession{}, errors.New("resolution requires an integrated iteration")
	}
	if result.Iterations[index].Resolution != nil {
		return BrownfieldSession{}, errors.New("iteration resolution is immutable once recorded")
	}
	if err := ValidateResolution(result.Source, result.Iterations[index].Integration.Report, resolution); err != nil {
		return BrownfieldSession{}, err
	}
	if resolution.TargetBasis != result.Target.ProjectDigest {
		return BrownfieldSession{}, errors.New("resolution must bind the session target basis")
	}
	for _, conflict := range result.Iterations[index].Integration.Conflicts {
		if conflict.Disposition != "unresolved" {
			continue
		}
		adopted := false
		for _, scope := range resolution.Scopes {
			if scope.ScopeID == conflict.ScopeID && scope.Status == "adopt" {
				adopted = true
			}
		}
		if adopted {
			answered := false
			for _, answer := range resolution.Questions {
				if answer.QuestionID == conflict.QuestionID && answer.Disposition == "answer" {
					answered = true
				}
			}
			if !answered {
				return BrownfieldSession{}, fmt.Errorf("scope %q cannot be adopted while conflict %q lacks an explicit owner answer", conflict.ScopeID, conflict.ID)
			}
		}
	}
	result.Iterations[index].Resolution = &resolution
	for _, scope := range resolution.Scopes {
		if scope.Status == "adopt" {
			setOneScopeStatus(&result, scope.ScopeID, "modeled", "Owner resolved this scope; model adoption is pending")
		} else {
			setOneScopeStatus(&result, scope.ScopeID, "transitional", "Owner resolution deferred this scope")
		}
	}
	sealSession(&result)
	return result, nil
}

func PlanSessionAdoption(sourceRoot string, target *projectwork.Project, session BrownfieldSession, iterationID, schemaDigest, buildDigest string) (AdoptionPlan, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return AdoptionPlan{}, err
	}
	activeRoot, ok := activeRootIteration(session)
	if !ok || iterationID != activeRoot.ID {
		return AdoptionPlan{}, errors.New("adoption plan must use the latest active root iteration; superseded or child iterations cannot be adopted")
	}
	iteration, exists := findIteration(session, iterationID)
	if !exists || iteration.Integration == nil || iteration.Resolution == nil {
		return AdoptionPlan{}, errors.New("adoption plan requires an integrated and owner-resolved iteration")
	}
	if target == nil || target.Digest != session.Target.ProjectDigest {
		return AdoptionPlan{}, errors.New("adoption target differs from the session's fixed target basis")
	}
	return PlanAdoption(sourceRoot, target, session.Source, iteration.Integration.Report, *iteration.Resolution, schemaDigest, buildDigest)
}

// recordSessionAdoption is package-private so external callers cannot turn a
// self-consistent plan and receipt into a durable claim without guarded Apply.
func recordSessionAdoption(sourceRoot string, session BrownfieldSession, iterationID string, plan AdoptionPlan, receipt AdoptionReceipt) (BrownfieldSession, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return BrownfieldSession{}, err
	}
	if _, err := RefreshDiscovery(sourceRoot, session.Source); err != nil {
		return BrownfieldSession{}, fmt.Errorf("source basis is stale: %w", err)
	}
	iteration, exists := findIteration(session, iterationID)
	if !exists || iteration.Resolution == nil || iteration.Integration == nil {
		return BrownfieldSession{}, errors.New("adoption receipt requires an owner-resolved iteration")
	}
	if err := ValidateAdoptionPlan(plan); err != nil {
		return BrownfieldSession{}, err
	}
	if plan.DiscoveryDigest != session.Source.Digest || plan.DistillationDigest != iteration.Integration.Report.Digest || plan.ResolutionDigest != iteration.Resolution.Digest || plan.TargetBasis != session.Target.ProjectDigest {
		return BrownfieldSession{}, errors.New("adoption plan does not bind the selected session stages")
	}
	if receipt.DiscoveryDigest != plan.DiscoveryDigest || receipt.DistillationDigest != plan.DistillationDigest || receipt.ResolutionDigest != plan.ResolutionDigest || receipt.PriorTargetBasis != plan.TargetBasis || receipt.CandidateDigest != plan.Edit.CandidateDigest {
		return BrownfieldSession{}, errors.New("adoption receipt does not match the applied model-only plan")
	}
	if !sameStrings(receipt.Adopted, plan.AdoptedScopes) || !sameStrings(receipt.Deferred, plan.DeferredScopes) {
		return BrownfieldSession{}, errors.New("adoption receipt scope outcome differs from its plan")
	}
	result := cloneSession(session)
	for _, prior := range result.Adoptions {
		if prior.IterationID == iterationID {
			return BrownfieldSession{}, errors.New("adoption receipt is immutable once recorded")
		}
	}
	result.Adoptions = append(result.Adoptions, SessionAdoption{IterationID: iterationID, Plan: plan, Receipt: receipt})
	for _, scopeID := range receipt.Adopted {
		setOneScopeStatus(&result, scopeID, "adopted", "Guarded model adoption was applied")
	}
	for _, scopeID := range receipt.Deferred {
		setOneScopeStatus(&result, scopeID, "transitional", "Owner deferred this scope")
	}
	sealSession(&result)
	return result, nil
}

func validateManagerProposal(proposal ManagerProposal, iteration ReverseIteration, session BrownfieldSession) error {
	if proposal.ManagerID != iteration.ManagerID || proposal.EvidenceIDs == nil || proposal.Hierarchy == nil || proposal.PublicContracts == nil || !sameStrings(proposal.EvidenceIDs, iteration.EvidenceIDs) {
		return errors.New("Manager proposal identity or selected evidence differs from its iteration")
	}
	if err := ValidateDistillation(session.Source, proposal.Report); err != nil {
		return fmt.Errorf("Manager proposal report: %w", err)
	}
	if err := validateSessionReportTarget(proposal.Report, session); err != nil {
		return err
	}
	selected := map[string]bool{}
	for _, id := range iteration.EvidenceIDs {
		selected[id] = true
	}
	for _, claim := range proposal.Report.Claims {
		for _, ref := range claim.Evidence {
			if !selected[ref.EvidenceID] {
				return fmt.Errorf("Manager proposal claim %q cites evidence outside its selected assignment", claim.ID)
			}
		}
	}
	for _, term := range proposal.Report.Terms {
		for _, occurrence := range term.Occurrences {
			if !selected[occurrence.EvidenceID] {
				return fmt.Errorf("Manager proposal term %q cites evidence outside its selected assignment", term.ID)
			}
		}
	}
	seenManagers := map[string]bool{}
	for _, child := range proposal.Hierarchy {
		if strings.TrimSpace(child.ID) == "" || seenManagers[strings.ToLower(child.ID)] || strings.TrimSpace(child.Name) == "" || strings.TrimSpace(child.Purpose) == "" || child.ParentID != proposal.ManagerID || child.EvidenceIDs == nil || len(child.EvidenceIDs) == 0 {
			return fmt.Errorf("invalid proposed child Manager %q", child.ID)
		}
		seenManagers[strings.ToLower(child.ID)] = true
		if err := validateEvidencePools(child.EvidenceIDs, child.DelegationEvidenceIDs, session.Source); err != nil {
			return fmt.Errorf("proposed Manager %q: %w", child.ID, err)
		}
		available := iterationAvailableEvidence(iteration)
		for _, evidenceID := range append(append([]string(nil), child.EvidenceIDs...), child.DelegationEvidenceIDs...) {
			if !available[evidenceID] {
				return fmt.Errorf("proposed Manager %q assignment exceeds its parent own and delegation evidence: evidence %q is not authorized", child.ID, evidenceID)
			}
		}
	}
	for _, contract := range proposal.PublicContracts {
		if err := validateProposedPublicContract(contract, proposal.ManagerID, proposal.Report); err != nil {
			return err
		}
	}
	copy := proposal
	copy.Digest = ""
	if proposal.Digest != "" && proposal.Digest != digestValue(copy) {
		return errors.New("Manager proposal digest mismatch")
	}
	return nil
}

func validateManagerIntegration(integration ManagerIntegration, iteration ReverseIteration, session BrownfieldSession) error {
	if integration.ManagerID != iteration.ManagerID || integration.ChildProposalDigests == nil || integration.ChildContracts == nil || integration.Conflicts == nil {
		return errors.New("Manager integration requires identity, explicit child digests/contracts, and conflict list")
	}
	if err := ValidateDistillation(session.Source, integration.Report); err != nil {
		return fmt.Errorf("integrated report: %w", err)
	}
	allowedEvidence := map[string]bool{}
	allowedForwardedRefs := map[evidenceRefKey]bool{}
	for _, id := range iteration.EvidenceIDs {
		allowedEvidence[id] = true
	}
	for _, child := range session.Iterations {
		if child.ParentIterationID == iteration.ID && child.Proposal != nil {
			finalReport := child.Proposal.Report
			if child.Integration != nil {
				finalReport = child.Integration.Report
			}
			for _, claim := range finalReport.Claims {
				for _, ref := range claim.Evidence {
					allowedForwardedRefs[evidenceRefIdentity(ref.EvidenceID, ref.StartLine, ref.EndLine, ref.Excerpt)] = true
					allowedEvidence[ref.EvidenceID] = true
				}
			}
			for _, term := range finalReport.Terms {
				for _, ref := range term.Occurrences {
					allowedForwardedRefs[evidenceRefIdentity(ref.EvidenceID, ref.StartLine, ref.EndLine, ref.Excerpt)] = true
					allowedEvidence[ref.EvidenceID] = true
				}
			}
		}
	}
	for _, claim := range integration.Report.Claims {
		for _, ref := range claim.Evidence {
			if !allowedEvidence[ref.EvidenceID] || (!containsString(iteration.EvidenceIDs, ref.EvidenceID) && !allowedForwardedRefs[evidenceRefIdentity(ref.EvidenceID, ref.StartLine, ref.EndLine, ref.Excerpt)]) {
				return fmt.Errorf("integrated claim %q cites evidence outside parent raw evidence or exact direct-child report occurrences", claim.ID)
			}
		}
	}
	for _, term := range integration.Report.Terms {
		for _, occurrence := range term.Occurrences {
			if !allowedEvidence[occurrence.EvidenceID] || (!containsString(iteration.EvidenceIDs, occurrence.EvidenceID) && !allowedForwardedRefs[evidenceRefIdentity(occurrence.EvidenceID, occurrence.StartLine, occurrence.EndLine, occurrence.Excerpt)]) {
				return fmt.Errorf("integrated term %q cites evidence outside parent raw evidence or exact direct-child report occurrences", term.ID)
			}
		}
	}
	if err := validateSessionReportTarget(integration.Report, session); err != nil {
		return err
	}
	children := []string{}
	expectedContracts := []IntegratedChildContracts{}
	expectedIntegrations := []ChildIntegrationDigest{}
	for _, item := range session.Iterations {
		if item.ParentIterationID == iteration.ID && item.Proposal != nil {
			if len(item.Proposal.Hierarchy) > 0 && item.Integration == nil {
				return fmt.Errorf("parent integration cannot accept non-leaf child Manager %q before its own integration", item.ManagerID)
			}
			children = append(children, item.Proposal.Digest)
			expectedContracts = append(expectedContracts, IntegratedChildContracts{ManagerID: item.ManagerID, ProposalDigest: item.Proposal.Digest, Contracts: append([]ManagerPublicContract{}, item.Proposal.PublicContracts...)})
			if item.Integration != nil {
				expectedIntegrations = append(expectedIntegrations, ChildIntegrationDigest{ManagerID: item.ManagerID, ProposalDigest: item.Proposal.Digest,
					IntegrationDigest: item.Integration.Digest, ReportDigest: item.Integration.Report.Digest})
			}
		}
	}
	sort.Strings(children)
	provided := append([]string(nil), integration.ChildProposalDigests...)
	sort.Strings(provided)
	if len(children) != len(provided) {
		return errors.New("parent integration must bind every completed direct-child proposal")
	}
	for i := range children {
		if !validDigest(provided[i]) || children[i] != provided[i] {
			return errors.New("parent integration child proposal digests do not match completed children")
		}
	}
	if len(integration.ChildContracts) != len(expectedContracts) {
		return errors.New("parent integration must include every direct child's explicit public contract list")
	}
	byProposal := map[string]IntegratedChildContracts{}
	for _, childContracts := range integration.ChildContracts {
		if childContracts.Contracts == nil || !validDigest(childContracts.ProposalDigest) {
			return errors.New("integrated child contracts require exact proposal digest and explicit contract list")
		}
		if _, duplicate := byProposal[childContracts.ProposalDigest]; duplicate {
			return errors.New("parent integration repeats a child proposal contract list")
		}
		byProposal[childContracts.ProposalDigest] = childContracts
	}
	for _, expected := range expectedContracts {
		provided, ok := byProposal[expected.ProposalDigest]
		if !ok || provided.ManagerID != expected.ManagerID || !sameContracts(provided.Contracts, expected.Contracts) {
			return fmt.Errorf("parent integration does not preserve Manager %q public contracts under its exact proposal digest", expected.ManagerID)
		}
	}
	sort.Slice(expectedIntegrations, func(i, j int) bool { return expectedIntegrations[i].ManagerID < expectedIntegrations[j].ManagerID })
	providedIntegrations := append([]ChildIntegrationDigest(nil), integration.ChildIntegrationDigests...)
	sort.Slice(providedIntegrations, func(i, j int) bool { return providedIntegrations[i].ManagerID < providedIntegrations[j].ManagerID })
	if len(providedIntegrations) != len(expectedIntegrations) {
		return errors.New("parent integration must bind every direct child integration digest; an empty legacy list is valid only when no child integration exists")
	}
	for i, expected := range expectedIntegrations {
		provided := providedIntegrations[i]
		if provided.ManagerID != expected.ManagerID || provided.ProposalDigest != expected.ProposalDigest || provided.IntegrationDigest != expected.IntegrationDigest || provided.ReportDigest != expected.ReportDigest {
			return fmt.Errorf("parent integration child integration binding differs for Manager %q", expected.ManagerID)
		}
	}
	for _, conflict := range integration.Conflicts {
		if !validID(conflict.ID) || !validScopeID(integration.Report, conflict.ScopeID) || !validQuestionInScope(integration.Report, conflict.QuestionID, conflict.ScopeID) || strings.TrimSpace(conflict.Description) == "" || conflict.EvidenceIDs == nil || strings.TrimSpace(conflict.Reason) == "" {
			return fmt.Errorf("invalid integration conflict %q", conflict.ID)
		}
		if conflict.Disposition != "unresolved" {
			return errors.New("Manager integration may report conflicts only as unresolved; owner decisions are separate")
		}
		if err := validateEvidenceSubset(conflict.EvidenceIDs, session.Source); err != nil {
			return err
		}
		for _, id := range conflict.EvidenceIDs {
			if !allowedEvidence[id] {
				return fmt.Errorf("conflict %q cites evidence outside parent and child assignments", conflict.ID)
			}
		}
	}
	copy := integration
	copy.Digest = ""
	if integration.Digest != "" && integration.Digest != digestValue(copy) {
		return errors.New("Manager integration digest mismatch")
	}
	return nil
}

func validateProposedPublicContract(value ManagerPublicContract, owner string, report Distillation) error {
	contract := value.Contract
	if strings.TrimSpace(contract.ID) == "" || strings.TrimSpace(contract.Name) == "" || strings.TrimSpace(contract.Description) == "" || strings.TrimSpace(contract.Category) == "" || contract.Owner != owner || contract.Uses == nil || contract.Requires == nil || value.ClaimIDs == nil || len(value.ClaimIDs) == 0 {
		return fmt.Errorf("proposed public contract %q requires stable identity, description, category, exact owner, explicit references, and claim grounding", contract.ID)
	}
	claims := map[string]bool{}
	for _, claim := range report.Claims {
		claims[claim.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range value.ClaimIDs {
		if !claims[id] || seen[id] {
			return fmt.Errorf("proposed public contract %q has unknown or duplicate grounding claim %q", contract.ID, id)
		}
		seen[id] = true
	}
	return nil
}

func sameContracts(left, right []ManagerPublicContract) bool {
	if left == nil || right == nil || len(left) != len(right) {
		return false
	}
	copyLeft := append([]ManagerPublicContract(nil), left...)
	copyRight := append([]ManagerPublicContract(nil), right...)
	sort.Slice(copyLeft, func(i, j int) bool { return copyLeft[i].Contract.ID < copyLeft[j].Contract.ID })
	sort.Slice(copyRight, func(i, j int) bool { return copyRight[i].Contract.ID < copyRight[j].Contract.ID })
	for i := range copyLeft {
		if digestValue(copyLeft[i]) != digestValue(copyRight[i]) {
			return false
		}
	}
	return true
}

func validateSessionReportTarget(report Distillation, session BrownfieldSession) error {
	if report.TargetBasis != "" || report.TargetRevision != "" || report.TargetContextDigest != "" {
		if report.TargetBasis != session.Target.ProjectDigest || report.TargetRevision != session.Target.Revision || report.TargetContextDigest != session.TargetContext.Digest {
			return errors.New("Manager report target binding differs from the session target basis")
		}
	} else if report.Method == "agent-assisted" {
		return errors.New("agent-assisted Manager report must bind the fixed target context")
	}
	return nil
}

func validateEvidenceSubset(ids []string, discovery Discovery) error {
	if ids == nil {
		return errors.New("evidence ID list must be explicit")
	}
	known := map[string]bool{}
	for _, evidence := range discovery.Evidence {
		known[evidence.ID] = true
	}
	seen := map[string]bool{}
	for _, id := range ids {
		if !known[id] || seen[id] {
			return fmt.Errorf("unknown or duplicate evidence ID %q", id)
		}
		seen[id] = true
	}
	return nil
}

func validateEvidencePools(own, delegation []string, discovery Discovery) error {
	if own == nil || len(own) == 0 {
		return errors.New("own evidence IDs must be an explicit nonempty list")
	}
	if _, err := sortedUniqueStrings(own); err != nil {
		return fmt.Errorf("own evidence IDs: %w", err)
	}
	if err := validateEvidenceSubset(own, discovery); err != nil {
		return err
	}
	if delegation == nil {
		return errors.New("delegation evidence IDs must be an explicit list, which may be empty")
	}
	if _, err := sortedUniqueStrings(delegation); err != nil {
		return fmt.Errorf("delegation evidence IDs: %w", err)
	}
	if err := validateEvidenceSubset(delegation, discovery); err != nil {
		return err
	}
	owned := make(map[string]bool, len(own))
	for _, id := range own {
		owned[id] = true
	}
	for _, id := range delegation {
		if owned[id] {
			return fmt.Errorf("evidence %q cannot appear in both own and delegation pools", id)
		}
	}
	return nil
}

func iterationAvailableEvidence(iteration ReverseIteration) map[string]bool {
	available := make(map[string]bool, len(iteration.EvidenceIDs)+len(iteration.DelegationEvidenceIDs))
	for _, id := range iteration.EvidenceIDs {
		available[id] = true
	}
	for _, id := range iteration.DelegationEvidenceIDs {
		available[id] = true
	}
	return available
}

func evidenceClassification(basis string) string {
	switch basis {
	case "code", "test", "configuration":
		return "observed-implementation"
	case "documentation":
		return "documented-intent"
	case "runtime-record":
		return "submitted-runtime-record"
	default:
		return "selected-evidence"
	}
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

type evidenceRefKey struct {
	ID        string
	StartLine int
	EndLine   int
	Excerpt   string
}

func evidenceRefIdentity(id string, startLine, endLine int, excerpt string) evidenceRefKey {
	return evidenceRefKey{ID: id, StartLine: startLine, EndLine: endLine, Excerpt: excerpt}
}

func targetManager(context DistillationTargetContext, id string) (DistillationTargetManager, bool) {
	for _, manager := range context.Managers {
		if manager.ID == id {
			return manager, true
		}
	}
	return DistillationTargetManager{}, false
}
func findIteration(session BrownfieldSession, id string) (ReverseIteration, bool) {
	for _, item := range session.Iterations {
		if item.ID == id {
			return item, true
		}
	}
	return ReverseIteration{}, false
}
func iterationIndex(session BrownfieldSession, id string) int {
	for i := range session.Iterations {
		if session.Iterations[i].ID == id {
			return i
		}
	}
	return -1
}
func validScopeID(report Distillation, id string) bool {
	for _, scope := range report.Scopes {
		if scope.ID == id {
			return true
		}
	}
	return false
}
func validQuestionInScope(report Distillation, questionID, scopeID string) bool {
	for _, question := range report.Questions {
		if question.ID == questionID && question.ScopeID == scopeID {
			return true
		}
	}
	return false
}
func cloneSession(session BrownfieldSession) BrownfieldSession {
	data, _ := encodeClosedJSON(session)
	copy, _ := DecodeBrownfieldSession(data)
	return copy
}
func setScopeStatesFromReport(session *BrownfieldSession, report Distillation, status string) {
	for _, scope := range report.Scopes {
		setOneScopeStatus(session, scope.ID, status, "Reverse-model proposal recorded")
	}
}
func setOneScopeStatus(session *BrownfieldSession, id, status, reason string) {
	for i := range session.Scopes {
		if session.Scopes[i].ScopeID == id {
			session.Scopes[i].Status = status
			session.Scopes[i].Reason = reason
			return
		}
	}
	session.Scopes = append(session.Scopes, ScopeStatus{ScopeID: id, Status: status, Reason: reason})
	sort.Slice(session.Scopes, func(i, j int) bool { return session.Scopes[i].ScopeID < session.Scopes[j].ScopeID })
}
