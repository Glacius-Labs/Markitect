package execution

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

// AmendmentRound retains the complete decision trail for one fresh candidate.
// Earlier rounds are provenance only and never supply votes to a later round.
type AmendmentRound struct {
	Round                   int                             `json:"round"`
	Status                  string                          `json:"status"`
	ChangedPaths            []string                        `json:"changedPaths"`
	ChangedPathDigests      []CandidatePathDigest           `json:"changedPathDigests"`
	ProposedModelDigest     string                          `json:"proposedModelDigest,omitempty"`
	CandidateSnapshotDigest string                          `json:"candidateSnapshotDigest,omitempty"`
	ActorSequences          []int                           `json:"actorSequences"`
	ConfigDigest            string                          `json:"configDigest,omitempty"`
	CandidateCommit         string                          `json:"candidateCommit,omitempty"`
	CandidateTree           string                          `json:"candidateTree,omitempty"`
	Candidate               *government.MaterialCandidate   `json:"candidate,omitempty"`
	Assessment              *government.AmendmentAssessment `json:"assessment,omitempty"`
	Checks                  []host.GateResult               `json:"checks"`
	ReviewActorSequences    []int                           `json:"reviewActorSequences"`
	VoteActorSequences      []int                           `json:"voteActorSequences"`
	Evidence                *government.Evidence            `json:"evidence,omitempty"`
	Votes                   []government.RessortVote        `json:"votes"`
	Decision                *government.AcceptanceDecision  `json:"decision,omitempty"`
	Error                   string                          `json:"error,omitempty"`
}

type CandidatePathDigest struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Mode   string `json:"mode"`
}

// EscalationRecord describes a blocked decision and the authority that must
// resolve it. Actor text is retained only as evidence and never authenticates
// an Owner approval.
type EscalationRecord struct {
	ID                      string                        `json:"id"`
	Round                   int                           `json:"round"`
	Reason                  string                        `json:"reason"`
	PriorConstitution       string                        `json:"priorConstitution"`
	BaseRevision            string                        `json:"baseRevision"`
	OrderPath               string                        `json:"orderPath"`
	OrderDigest             string                        `json:"orderDigest"`
	ConfigPath              string                        `json:"configPath"`
	ConfigDigest            string                        `json:"configDigest,omitempty"`
	CandidateModelDigest    string                        `json:"candidateModelDigest,omitempty"`
	CandidateCommit         string                        `json:"candidateCommit,omitempty"`
	CandidateTree           string                        `json:"candidateTree,omitempty"`
	MaterialCandidateID     string                        `json:"materialCandidateId,omitempty"`
	EvidenceID              string                        `json:"evidenceId,omitempty"`
	ChangedSubjects         []core.DefinitionIdentity     `json:"changedSubjects"`
	ChangedPaths            []string                      `json:"changedPaths"`
	ChangedPathDigests      []CandidatePathDigest         `json:"changedPathDigests"`
	CandidateSnapshotDigest string                        `json:"candidateSnapshotDigest,omitempty"`
	Findings                []government.AmendmentFinding `json:"findings"`
	PlanFindings            []government.Finding          `json:"planFindings"`
	DelegationDigest        string                        `json:"delegationDigest,omitempty"`
	DelegationFindings      []government.Finding          `json:"delegationFindings,omitempty"`
	CheckDigests            []string                      `json:"checkDigests"`
	ActorSequences          []int                         `json:"actorSequences"`
	ReviewActorSequences    []int                         `json:"reviewActorSequences"`
	VoteActorSequences      []int                         `json:"voteActorSequences"`
	VoteIDs                 []string                      `json:"voteIds"`
	Uncertainty             []string                      `json:"uncertainty"`
	NextHigherArea          core.DefinitionIdentity       `json:"nextHigherArea,omitempty"`
	OwnerRequired           bool                          `json:"ownerRequired"`
	RequiredDecision        string                        `json:"requiredDecision"`
	PromotionAttempted      bool                          `json:"promotionAttempted"`
}

func amendmentFeedback(round AmendmentRound) string {
	return fmt.Sprintf("Prior candidate round %d was not accepted. Assessment=%s; checks=%s; evidence=%s; votes=%s; error=%s. Propose a fresh candidate under the unchanged prior order and authority.", round.Round, jsonText(round.Assessment), jsonText(round.Checks), jsonText(round.Evidence), jsonText(round.Votes), round.Error)
}

func amendmentFindingsText(findings []government.AmendmentFinding) string {
	if len(findings) == 0 {
		return "candidate is outside the active prior authority"
	}
	parts := make([]string, 0, len(findings))
	for _, finding := range findings {
		route := "higher prior Area or trusted Owner"
		if finding.EscalateToOwner {
			route = "trusted Owner"
		} else if finding.EscalateTo.Name != "" {
			route = finding.EscalateTo.Key()
		}
		parts = append(parts, finding.Code+" ("+finding.Subject+"): "+finding.Detail+"; escalation="+route)
	}
	return strings.Join(parts, "; ")
}

func appendAmendmentEscalation(report *Report, in amendmentInputs, round AmendmentRound, reason string) error {
	findings := []government.AmendmentFinding{}
	changed := []core.DefinitionIdentity{}
	if round.Assessment != nil {
		findings = append(findings, round.Assessment.Findings...)
		changed = append(changed, round.Assessment.Subjects...)
	}
	if len(findings) == 0 {
		findings = append(findings, government.AmendmentFinding{Code: "amendment.execution-blocked", Subject: in.model.Constitution.Key(), Detail: reason, EscalateTo: in.model.Root})
	}
	higher, ownerRequired := amendmentEscalationRoute(in.model, findings)
	checkDigests := []string{}
	if len(round.Checks) > 0 {
		checkDigests = append(checkDigests, government.Digest(round.Checks))
	}
	voteIDs := []string{}
	for _, vote := range round.Votes {
		voteIDs = append(voteIDs, vote.ID)
	}
	materialID, evidenceID, modelDigest, commit, tree, configDigest := "", "", "", "", "", ""
	if round.Candidate != nil {
		materialID = round.Candidate.ID
		modelDigest = round.Candidate.Input.ModelDigest
	}
	if modelDigest == "" {
		modelDigest = round.ProposedModelDigest
	}
	if round.Evidence != nil {
		evidenceID = round.Evidence.ID
	}
	if round.CandidateCommit != "" {
		commit, tree = round.CandidateCommit, round.CandidateTree
	}
	configDigest = round.ConfigDigest
	delegationDigest := ""
	delegationFindings := []government.Finding{}
	if report.Delegation != nil {
		delegationDigest = report.Delegation.Digest
		delegationFindings = append(delegationFindings, report.Delegation.Findings...)
	}
	uncertainty := []string{"model quality and semantic sufficiency are not established by this procedure"}
	for _, sequence := range round.ActorSequences {
		for _, actor := range report.Actors {
			if actor.Sequence != sequence {
				continue
			}
			for _, detail := range actor.Result.Response.Uncertainty {
				uncertainty = append(uncertainty, fmt.Sprintf("actor %d (%s): %s", sequence, actor.Phase, detail))
			}
		}
	}
	entry := EscalationRecord{ID: government.Digest(struct {
		Run       string
		Round     int
		Reason    string
		Candidate string
	}{report.RunID, round.Round, reason, materialID}), Round: round.Round, Reason: reason, PriorConstitution: in.model.Digest, BaseRevision: in.session.runtime.ExpectedBase, OrderPath: in.opts.OrderPath, OrderDigest: government.Digest(in.order), ConfigPath: in.opts.ConfigPath, ConfigDigest: configDigest, CandidateModelDigest: modelDigest, CandidateCommit: commit, CandidateTree: tree, CandidateSnapshotDigest: round.CandidateSnapshotDigest, MaterialCandidateID: materialID, EvidenceID: evidenceID, ChangedSubjects: changed, ChangedPaths: append([]string(nil), round.ChangedPaths...), ChangedPathDigests: append([]CandidatePathDigest(nil), round.ChangedPathDigests...), Findings: findings, DelegationDigest: delegationDigest, DelegationFindings: delegationFindings, ActorSequences: append([]int(nil), round.ActorSequences...), CheckDigests: checkDigests, ReviewActorSequences: append([]int(nil), round.ReviewActorSequences...), VoteActorSequences: append([]int(nil), round.VoteActorSequences...), VoteIDs: voteIDs, Uncertainty: uncertainty, NextHigherArea: higher, OwnerRequired: ownerRequired, RequiredDecision: "resolve the named prior-law scope or authorize a new rule through the trusted Owner channel; then issue a fresh order and rerun all checks, reviews, and cabinet votes", PromotionAttempted: false}
	report.Escalations = append(report.Escalations, entry)
	if err := persistJSON(filepath.Join(filepath.Dir(report.ReportPath), fmt.Sprintf("escalation-%02d.json", round.Round)), entry); err != nil {
		return fmt.Errorf("persist amendment escalation: %w", err)
	}
	return nil
}

func amendmentEscalationRoute(model government.Model, findings []government.AmendmentFinding) (core.DefinitionIdentity, bool) {
	for _, finding := range findings {
		if finding.EscalateToOwner {
			return core.DefinitionIdentity{}, true
		}
	}
	for _, finding := range findings {
		if finding.EscalateTo.Name != "" && amendmentMandateApplies(model, finding.EscalateTo, finding.Subject) {
			return finding.EscalateTo, false
		}
	}
	return core.DefinitionIdentity{}, true
}

func blockAmendmentOrderPathWrite(report *Report, opts Options, runtime Runtime, model government.Model, order government.Order, session *actorSession, base *snapshot.Snapshot) (Report, error) {
	reason := "amend-model plan selected the immutable OrderPath as a writable candidate path"
	finding := government.AmendmentFinding{Code: "amendment.order-input", Subject: opts.OrderPath, Detail: "the issued order is immutable input and cannot be included in the amendment candidate", EscalateToOwner: true}
	assessment := government.AmendmentAssessment{
		Status:            "blocked-escalation-required",
		Digest:            government.Digest(struct{ Prior, Order, Path string }{model.Digest, government.Digest(order), opts.OrderPath}),
		PriorConstitution: model.Digest,
		Changed:           []core.DefinitionIdentity{},
		Subjects:          []core.DefinitionIdentity{},
		Findings:          []government.AmendmentFinding{finding},
	}
	configDigest := ""
	if base != nil {
		if data, ok := base.Files[opts.ConfigPath]; ok {
			configDigest = government.BytesDigest(data)
		}
	}
	round := AmendmentRound{Round: 0, Status: assessment.Status, ConfigDigest: configDigest, Assessment: &assessment, Checks: []host.GateResult{}, ActorSequences: []int{}, ReviewActorSequences: []int{}, VoteActorSequences: []int{}, Votes: []government.RessortVote{}, Error: reason}
	report.Status, report.Stage = "blocked", "amendment-authority"
	report.AmendmentAssessment = &assessment
	report.AmendmentRounds = append(report.AmendmentRounds, round)
	in := amendmentInputs{opts: opts, model: model, order: order, report: report, session: session}
	escalationErr := appendAmendmentEscalation(report, in, round, reason)
	return *report, errors.Join(errors.New(reason), escalationErr)
}

func appendDelegationAmendmentEscalation(report *Report, in amendmentInputs, delegation government.DelegationPlan, reason string) error {
	findings := make([]government.AmendmentFinding, 0, len(delegation.Findings))
	for _, priorFinding := range delegation.Findings {
		finding := government.AmendmentFinding{Code: priorFinding.Code, Subject: priorFinding.Subject, Detail: priorFinding.Detail, EscalateToOwner: true}
		if subject := priorIdentityByKey(in.model, priorFinding.Subject); subject.Name != "" && subject.APIVersion != government.APIVersion {
			if higher := priorEscalationTarget(in.model, subject); higher.Name != "" && priorMandateForSubject(in.model, higher, subject, "amend-model") != "" {
				finding.EscalateTo, finding.EscalateToOwner = higher, false
			}
		}
		findings = append(findings, finding)
	}
	if len(findings) == 0 {
		findings = append(findings, government.AmendmentFinding{Code: "amendment.delegation-blocked", Subject: in.model.Root.Key(), Detail: reason, EscalateToOwner: true})
	}
	assessment := government.AmendmentAssessment{
		Status: "blocked-escalation-required",
		Digest: government.Digest(struct {
			Prior, Plan, Delegation string
			Findings                []government.AmendmentFinding
		}{in.model.Digest, in.report.Plan.Digest, delegation.Digest, findings}),
		PriorConstitution: in.model.Digest,
		Changed:           []core.DefinitionIdentity{},
		Subjects:          []core.DefinitionIdentity{},
		Findings:          findings,
	}
	round := AmendmentRound{Round: 0, Status: assessment.Status, Assessment: &assessment, Checks: []host.GateResult{}, ActorSequences: []int{}, ReviewActorSequences: []int{}, VoteActorSequences: []int{}, Votes: []government.RessortVote{}, Error: reason}
	report.AmendmentAssessment = &assessment
	report.AmendmentRounds = append(report.AmendmentRounds, round)
	delegationErr := persistJSON(filepath.Join(filepath.Dir(report.ReportPath), "delegation.json"), delegation)
	escalationErr := appendAmendmentEscalation(report, in, round, reason)
	return errors.Join(delegationErr, escalationErr)
}

func appendActorSequences(round *AmendmentRound, records []ActorRecord) {
	seen := make(map[int]bool, len(round.ActorSequences))
	for _, sequence := range round.ActorSequences {
		seen[sequence] = true
	}
	for _, actor := range records {
		if !seen[actor.Sequence] {
			round.ActorSequences = append(round.ActorSequences, actor.Sequence)
			seen[actor.Sequence] = true
		}
	}
}

func candidatePathDigests(candidate *snapshot.Snapshot, paths []string) []CandidatePathDigest {
	result := make([]CandidatePathDigest, 0, len(paths))
	for _, path := range paths {
		result = append(result, CandidatePathDigest{Path: path, Digest: government.BytesDigest(candidate.Files[path]), Mode: candidate.Modes[path]})
	}
	return result
}

func appendPlanAmendmentEscalation(report *Report, opts Options, runtime Runtime, model government.Model, order government.Order) error {
	findings := make([]government.AmendmentFinding, 0, len(report.Plan.Findings))
	for _, finding := range report.Plan.Findings {
		escalateTo := core.DefinitionIdentity{}
		if subject := priorIdentityByKey(model, finding.Subject); subject.Name != "" {
			escalateTo = priorEscalationTarget(model, subject)
			if escalateTo.Name != "" && priorMandateForSubject(model, escalateTo, subject, "amend-model") == "" {
				escalateTo = core.DefinitionIdentity{}
			}
		}
		findings = append(findings, government.AmendmentFinding{Code: finding.Code, Subject: finding.Subject, Detail: finding.Detail, EscalateTo: escalateTo, EscalateToOwner: escalateTo.Name == ""})
	}
	nextArea, ownerRequired := amendmentEscalationRoute(model, findings)
	reason := planFindingsText(report.Plan.Findings)
	entry := EscalationRecord{ID: government.Digest(struct{ Run, Reason string }{report.RunID, reason}), Round: 0, Reason: reason, PriorConstitution: model.Digest, BaseRevision: runtime.ExpectedBase, OrderPath: opts.OrderPath, OrderDigest: government.Digest(order), ConfigPath: opts.ConfigPath, ChangedSubjects: []core.DefinitionIdentity{}, ChangedPaths: []string{}, Findings: findings, PlanFindings: append([]government.Finding(nil), report.Plan.Findings...), CheckDigests: []string{}, ActorSequences: []int{}, ReviewActorSequences: []int{}, VoteActorSequences: []int{}, VoteIDs: []string{}, Uncertainty: []string{"no candidate actor was invoked because the frozen prior-authority plan was blocked"}, NextHigherArea: nextArea, OwnerRequired: ownerRequired, RequiredDecision: "the named prior Area with an applicable amend-model mandate, or the trusted Owner, must resolve the concrete prior-plan findings and issue a new order", PromotionAttempted: false}
	report.Escalations = append(report.Escalations, entry)
	if err := persistJSON(filepath.Join(filepath.Dir(report.ReportPath), "escalation-plan.json"), entry); err != nil {
		return fmt.Errorf("persist amendment plan escalation: %w", err)
	}
	return nil
}

func priorIdentityByKey(model government.Model, key string) core.DefinitionIdentity {
	for _, definition := range model.Canonical.Definitions {
		if definition.Identity().Key() == key {
			return definition.Identity()
		}
	}
	return core.DefinitionIdentity{}
}

func priorEscalationTarget(model government.Model, subject core.DefinitionIdentity) core.DefinitionIdentity {
	var owner core.DefinitionIdentity
	var definitions []core.Definition
	for _, definition := range model.Canonical.Definitions {
		definitions = append(definitions, definition)
		if definition.APIVersion == government.APIVersion && definition.Kind == "Responsibility" && identityValue(definition.Spec["subject"]).Key() == subject.Key() {
			owner = identityValue(definition.Spec["area"])
		}
	}
	if owner.Name == "" {
		return core.DefinitionIdentity{}
	}
	byKey := make(map[string]core.Definition, len(definitions))
	for _, definition := range definitions {
		byKey[definition.Identity().Key()] = definition
	}
	parent := identityValue(byKey[owner.Key()].Spec["parent"])
	for parent.Name != "" {
		if priorMandateForSubject(model, parent, subject, "amend-model") != "" {
			return parent
		}
		parent = identityValue(byKey[parent.Key()].Spec["parent"])
	}
	return core.DefinitionIdentity{}
}

func planFindingsText(findings []government.Finding) string {
	parts := make([]string, 0, len(findings))
	for _, finding := range findings {
		parts = append(parts, finding.Code+" ("+finding.Subject+"): "+finding.Detail)
	}
	return strings.Join(parts, "; ")
}
