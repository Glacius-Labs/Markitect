package execution

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

type amendmentInputs struct {
	opts          Options
	repo          string
	base          *snapshot.Snapshot
	source        government.Source
	model         government.Model
	order         government.Order
	workspace     string
	runDir        string
	report        *Report
	session       *actorSession
	selectedPaths []string
	invoke        func(string, RunnerSpec, []string, *snapshot.Snapshot, map[string]any) (agentexec.RunResult, error)
}

func runAmendment(ctx context.Context, in amendmentInputs) (Report, error) {
	r := in.report
	if in.session.runtime.Amendment == nil {
		return *r, errors.New("amend-model order requires explicit bounded amendment runtime configuration")
	}
	configPath := in.opts.ConfigPath
	if len(in.source.Observation.Roots) == 0 {
		return *r, errors.New("amend-model requires a complete prior observation boundary")
	}
	if err := validateAmendmentConfigWriter(in.model, r.Plan, in.order, configPath); err != nil {
		err = errors.Join(err, appendAmendmentEscalation(r, in, AmendmentRound{Round: 1, Status: "blocked", Error: err.Error()}, err.Error()))
		r.Status = "blocked"
		return *r, err
	}
	if in.session.runtime.Recursion != nil {
		delegation := government.BuildDelegationPlan(in.model, r.Plan, in.session.runtime.Recursion.Limits)
		r.Delegation = &delegation
		if delegation.Status == "blocked" {
			reason := "recursive prior-authority delegation is blocked for amendment: " + planFindingsText(delegation.Findings)
			r.Status, r.Stage = "blocked", "amendment-delegation"
			return *r, errors.Join(errors.New(reason), appendDelegationAmendmentEscalation(r, in, delegation, reason))
		}
		if err := validateNodeWireBounds(delegation.Root); err != nil {
			return *r, err
		}
		if err := validateAreaAssignments(delegation.Root, in.session.runtime); err != nil {
			return *r, err
		}
		if err := persistJSON(filepath.Join(filepath.Dir(r.ReportPath), "delegation.json"), delegation); err != nil {
			return *r, err
		}
		maximumRounds := in.session.runtime.Amendment.MaxRepairs + 1
		minimumCalls := maximumRounds * (delegation.EstimatedCalls + len(r.Cabinet) + 1)
		if minimumCalls > in.session.runtime.Recursion.Limits.MaxCalls {
			return *r, errors.New("invocation budget cannot cover bounded amendment rounds, final Root review and frozen cabinet")
		}
	}
	if err := persistJSON(filepath.Join(filepath.Dir(r.ReportPath), "amendment-input.json"), map[string]any{
		"priorConstitution": in.model.Digest,
		"baseRevision":      in.session.runtime.ExpectedBase,
		"order":             in.order,
		"orderDigest":       government.Digest(in.order),
		"plan":              r.Plan,
		"configPath":        configPath,
		"configDigest":      government.BytesDigest(in.base.Files[configPath]),
	}); err != nil {
		return *r, err
	}
	var repairFeedback string
	repairInput := in.base
	maxRounds := in.session.runtime.Amendment.MaxRepairs + 1
roundLoop:
	for index := 1; index <= maxRounds; index++ {
		if err := ctx.Err(); err != nil {
			return *r, err
		}
		round := AmendmentRound{Round: index, Status: "executing", Checks: []host.GateResult{}, ReviewActorSequences: []int{}, Votes: []government.RessortVote{}}
		r.AmendmentRounds = append(r.AmendmentRounds, round)
		roundIndex := len(r.AmendmentRounds) - 1
		actorStart := len(r.Actors)
		var candidate *snapshot.Snapshot
		r.Candidate, r.Evidence, r.Decision = nil, nil, nil
		r.CandidateCommit, r.CandidateTree = "", ""
		r.AmendmentAssessment, r.ProposedModelDigest = nil, ""
		r.Votes = []government.RessortVote{}
		persistRevision := 0
		persistRound := func() error {
			persistRevision++
			path := filepath.Join(filepath.Dir(r.ReportPath), fmt.Sprintf("amendment-round-%02d-state-%02d.json", index, persistRevision))
			return persistJSON(path, r.AmendmentRounds[roundIndex])
		}
		setFailure := func(err error, hard bool) (Report, error) {
			current := &r.AmendmentRounds[roundIndex]
			current.Status = "blocked"
			current.Error = err.Error()
			if persistErr := persistRound(); persistErr != nil {
				err = errors.Join(err, persistErr)
			}
			if hard || index == maxRounds {
				err = errors.Join(err, appendAmendmentEscalation(r, in, *current, err.Error()))
				r.Status = "blocked"
				return *r, err
			}
			if candidate != nil {
				repairInput = candidate
			}
			repairFeedback = amendmentFeedback(*current)
			return Report{}, nil
		}
		if err := resetAmendmentWorkspace(in.workspace, in.session.runtime.TemporaryDirectory, repairInput); err != nil {
			return *r, err
		}
		if err := persistRound(); err != nil {
			return *r, err
		}
		contextExtra := map[string]any{"amendmentRound": index, "priorModel": in.model.Canonical, "priorModelDigest": in.model.Digest, "repairFeedback": repairFeedback}
		if in.session.runtime.Recursion != nil {
			engine := recursiveEngine{session: in.session, model: in.model, order: in.order, delegation: *r.Delegation}
			var area AreaReport
			var err error
			candidate, area, err = engine.node(r.Delegation.Root, repairInput, repairFeedback)
			r.RootArea = &area
			appendActorSequences(&r.AmendmentRounds[roundIndex], r.Actors[actorStart:])
			if err != nil {
				_, resultErr := setFailure(err, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
		} else {
			r.Stage = "amendment-execute"
			executed, err := in.invoke("execute", in.session.runtime.Executor, identityKeys(r.Plan.Affected), repairInput, contextExtra)
			appendActorSequences(&r.AmendmentRounds[roundIndex], r.Actors[actorStart:])
			if err != nil {
				_, resultErr := setFailure(err, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
			if executed.Response.Outcome != agentexec.OutcomeProposed {
				err := errors.New("amendment executor did not propose material")
				_, resultErr := setFailure(err, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
			if err := confirmWorkspace(in.workspace, repairInput); err != nil {
				_, resultErr := setFailure(err, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
			candidate, err = applyProposal(repairInput, in.selectedPaths, executed.Response.CandidateFiles)
			if err != nil {
				_, resultErr := setFailure(err, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
		}
		current := &r.AmendmentRounds[roundIndex]
		current.ChangedPaths = changedPaths(in.base, candidate)
		current.CandidateSnapshotDigest = candidate.Digest()
		current.ChangedPathDigests = candidatePathDigests(candidate, current.ChangedPaths)
		r.ChangedPaths = current.ChangedPaths
		if !amendmentHasString(current.ChangedPaths, configPath) {
			err := errors.New("amendment proposal did not change the canonical Government source ConfigPath")
			_, resultErr := setFailure(err, false)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		var proposed government.Source
		configBytes := candidate.Files[configPath]
		if err := government.Decode(configBytes, &proposed); err != nil {
			_, resultErr := setFailure(fmt.Errorf("proposed Government source is invalid: %w", err), false)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		pinSourceProvenance(&proposed, configPath, configBytes)
		proposedModel := government.Compile(proposed)
		assessment := government.AssessAmendment(in.model, proposed, in.order, r.Plan)
		if writer, ok := priorConfigWriter(in.model, configPath); ok {
			for _, subject := range assessment.Subjects {
				if priorMandateForSubject(in.model, writer, subject, "amend-model") == "" {
					assessment.Findings = append(assessment.Findings, government.AmendmentFinding{Code: "amendment.config-writer-authority", Subject: subject.Key(), Detail: "the prior ConfigPath Writer has no amend-model mandate covering this changed subject", EscalateToOwner: true})
				}
			}
		}
		if len(assessment.Findings) > 0 {
			assessment.Status = "blocked-escalation-required"
			assessment.Digest = government.Digest(struct {
				PriorDigest string
				Subjects    []core.DefinitionIdentity
				Findings    []government.AmendmentFinding
			}{assessment.Digest, assessment.Subjects, assessment.Findings})
		}
		if proposedModel.Digest == in.model.Digest && len(proposedModel.Findings) == 0 {
			assessment.Findings = append(assessment.Findings, government.AmendmentFinding{Code: "amendment.no-substantive-model-change", Subject: in.model.Constitution.Key(), Detail: "candidate changes no canonical Government model semantics", EscalateTo: in.model.Root})
			assessment.Status = "blocked-escalation-required"
			assessment.Digest = government.Digest(struct {
				Assessment string
				Findings   []government.AmendmentFinding
			}{assessment.Digest, assessment.Findings})
		}
		current.ConfigDigest = government.BytesDigest(configBytes)
		current.ProposedModelDigest = proposedModel.Digest
		current.Assessment = &assessment
		current.Status = assessment.Status
		r.AmendmentAssessment = &assessment
		r.ProposedModelDigest = proposedModel.Digest
		if err := persistRound(); err != nil {
			return *r, err
		}
		if proposedModel.Digest == in.model.Digest && len(proposedModel.Findings) == 0 {
			err := errors.New("amendment candidate changes no canonical Government model semantics")
			_, resultErr := setFailure(err, false)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		if assessment.Status != "eligible-for-fresh-review" {
			err := errors.New("amendment authority assessment blocked: " + amendmentFindingsText(assessment.Findings))
			current.Error = err.Error()
			err = errors.Join(err, persistRound())
			err = errors.Join(err, appendAmendmentEscalation(r, in, *current, err.Error()))
			r.Status = "blocked"
			return *r, err
		}
		if err := writeChanges(in.workspace, in.base, candidate); err != nil {
			_, resultErr := setFailure(err, true)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		commit, tree, err := commitCandidate(ctx, in.repo, in.session.runtime.ExpectedBase, fmt.Sprintf("%s-round-%02d", r.RunID, index), filepath.Dir(r.ReportPath), current.ChangedPaths, candidate)
		if err != nil {
			_, resultErr := setFailure(err, true)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		candidate.ID = commit
		current.CandidateCommit, current.CandidateTree = commit, tree
		material, err := government.NewMaterialCandidate(government.MaterialCandidateInput{
			PriorConstitutionDigest: in.model.Digest,
			BaseRevision:            in.session.runtime.ExpectedBase,
			RepositoryTreeDigest: government.Digest(struct {
				Tree, Snapshot string
				Round          int
			}{tree, candidate.Digest(), index}),
			ModelDigest:            proposedModel.Digest,
			PlanDigest:             r.Plan.Digest,
			CheckDefinitionsDigest: checkDefinitionsDigest(in.session.runtime),
			ToolPinsDigest:         r.ToolPins,
			InventoryDigest:        r.Plan.InventoryDigest,
		})
		if err != nil {
			_, resultErr := setFailure(err, true)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		current.Candidate = &material
		r.Candidate, r.CandidateCommit, r.CandidateTree = &material, commit, tree
		if err := persistRound(); err != nil {
			return *r, err
		}
		if err := persistJSON(filepath.Join(filepath.Dir(r.ReportPath), fmt.Sprintf("amendment-source-%02d.json", index)), proposed); err != nil {
			return *r, err
		}
		if err := persistJSON(filepath.Join(filepath.Dir(r.ReportPath), fmt.Sprintf("amendment-material-%02d.json", index)), material); err != nil {
			return *r, err
		}
		if err := confirmWorkspace(in.workspace, candidate); err != nil {
			return *r, err
		}

		r.Stage = "amendment-technical-checks"
		if pins, err := toolPins(in.session.runtime); err != nil || pins != r.ToolPins {
			return *r, errors.Join(errors.New("runtime/tool pins changed before amendment checks"), err)
		}
		checks, checkErr := freshChecks(ctx, candidate, in.session.runtime.Checks)
		current.Checks, r.Checks = checks, checks
		if err := persistJSON(filepath.Join(filepath.Dir(r.ReportPath), fmt.Sprintf("amendment-checks-%02d.json", index)), checks); err != nil {
			return *r, err
		}
		var ordinaryCheckFailure *host.VerifyError
		if checkErr != nil && (!errors.As(checkErr, &ordinaryCheckFailure) || ordinaryCheckFailure.Kind != "gate-failure") {
			_, resultErr := setFailure(checkErr, true)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		if err := confirmWorkspace(in.workspace, candidate); err != nil {
			return *r, err
		}

		r.Stage = "amendment-independent-review"
		reviewAreas := r.Plan.IntegrationReviews
		if in.session.runtime.Recursion != nil {
			reviewAreas = []core.DefinitionIdentity{in.model.Root}
		}
		subjectScopes := identityKeys(r.Plan.Affected)
		reviewErrors := []string{}
		for _, area := range reviewAreas {
			scopes := append([]string{area.Key()}, subjectScopes...)
			extra := map[string]any{"candidate": material, "candidateCommit": commit, "candidateTree": tree, "checks": checks, "reviewArea": area, "priorModel": in.model.Canonical, "priorModelDigest": in.model.Digest, "proposedModel": proposedModel.Canonical, "proposedModelDigest": proposedModel.Digest, "amendmentAssessment": assessment}
			actorCount := len(r.Actors)
			result, invokeErr := in.invoke("review", in.session.runtime.Verifier, scopes, candidate, extra)
			if len(r.Actors) > actorCount {
				current.ReviewActorSequences = append(current.ReviewActorSequences, r.Actors[len(r.Actors)-1].Sequence)
				appendActorSequences(current, r.Actors[len(r.Actors)-1:])
			}
			if invokeErr != nil {
				_, resultErr := setFailure(invokeErr, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
			if err := requireReview(result.Response, scopes); err != nil {
				reviewErrors = append(reviewErrors, err.Error())
			}
		}
		if err := confirmWorkspace(in.workspace, candidate); err != nil {
			return *r, err
		}
		resultDigests := []string{government.Digest(checks), government.Digest(assessment)}
		for _, sequence := range current.ActorSequences {
			for _, actor := range r.Actors {
				if actor.Sequence == sequence {
					resultDigests = append(resultDigests, government.Digest(actor))
					break
				}
			}
		}
		if r.RootArea != nil {
			resultDigests = append(resultDigests, government.Digest(r.RootArea), r.Delegation.Digest)
		}
		lineageDigests := []string{government.Digest(r.Plan)}
		for _, previous := range r.AmendmentRounds[:roundIndex] {
			lineageDigests = append(lineageDigests, government.Digest(previous))
		}
		evidence, err := government.NewEvidence(material, uint64(index), resultDigests, lineageDigests)
		if err != nil {
			return *r, err
		}
		current.Evidence, r.Evidence = &evidence, &evidence
		if err := persistRound(); err != nil {
			return *r, err
		}
		if err := persistJSON(filepath.Join(filepath.Dir(r.ReportPath), fmt.Sprintf("amendment-evidence-%02d.json", index)), evidence); err != nil {
			return *r, err
		}
		if checkErr != nil || len(reviewErrors) > 0 {
			reason := errors.Join(checkErr, errors.New(strings.Join(reviewErrors, "; ")))
			_, resultErr := setFailure(reason, false)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}

		r.Stage = "amendment-ressort-votes"
		r.Votes = []government.RessortVote{}
		current.Votes = []government.RessortVote{}
		for _, member := range r.Cabinet {
			spec := ressortSpec(in.session.runtime, member.Ressort)
			actorCount := len(r.Actors)
			result, invokeErr := in.invoke("vote", spec, []string{member.Ressort.Key()}, candidate, map[string]any{"candidate": material, "candidateCommit": commit, "candidateTree": tree, "evidence": evidence, "checks": checks, "member": member, "priorModel": in.model.Canonical, "proposedModel": proposedModel.Canonical, "amendmentAssessment": assessment})
			if len(r.Actors) > actorCount {
				sequence := r.Actors[len(r.Actors)-1].Sequence
				current.VoteActorSequences = append(current.VoteActorSequences, sequence)
				appendActorSequences(current, r.Actors[len(r.Actors)-1:])
			}
			if invokeErr != nil {
				_, resultErr := setFailure(invokeErr, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
			vote, voteErr := validatedVote(material, evidence, member, result)
			if voteErr != nil {
				_, resultErr := setFailure(voteErr, true)
				if resultErr == nil {
					continue roundLoop
				}
				return *r, resultErr
			}
			current.Votes = append(current.Votes, vote)
			r.Votes = append(r.Votes, vote)
			if err := persistRound(); err != nil {
				return *r, err
			}
		}
		objection := false
		for _, vote := range current.Votes {
			if vote.Outcome == government.VoteObjection {
				objection = true
			}
		}
		if objection {
			err := errors.New("frozen prior cabinet explicitly objected to amendment candidate")
			_, resultErr := setFailure(err, false)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		decision, err := government.NewAcceptanceDecision(material, evidence, in.model.Digest, r.Cabinet, current.Votes)
		if err != nil {
			_, resultErr := setFailure(err, true)
			if resultErr == nil {
				continue roundLoop
			}
			return *r, resultErr
		}
		current.Decision, r.Decision = &decision, &decision
		if err := persistJSON(filepath.Join(filepath.Dir(r.ReportPath), fmt.Sprintf("amendment-decision-%02d.json", index)), decision); err != nil {
			return *r, err
		}
		current.Status = "accepted-scoped"
		if err := persistRound(); err != nil {
			return *r, err
		}
		r.Stage = "amendment-promote"
		if err := ctx.Err(); err != nil {
			return *r, err
		}
		if err := confirmWorkspace(in.workspace, candidate); err != nil {
			return *r, err
		}
		finalPins, err := toolPins(in.session.runtime)
		if err != nil || finalPins != r.ToolPins {
			return *r, errors.Join(errors.New("runtime/tool pins changed after amendment material binding"), err)
		}
		promoted, err := government.Promote(ctx, government.PromotionRequest{Repo: in.repo, ActiveRef: in.session.runtime.ActiveRef, ExpectedOld: in.session.runtime.ExpectedBase, NewCommit: commit, ExpectedTreeID: tree, MaterialCandidateID: material.ID, EvidenceID: evidence.ID, DecisionID: decision.ID, StateDirectory: filepath.Dir(r.ReportPath), IdempotencyKey: r.RunID})
		r.Promotion = &promoted
		if err != nil {
			return *r, err
		}
		if promoted.Status != "promoted" {
			return *r, errors.New("amendment promotion did not establish the new active commit")
		}
		r.Stage, r.Status = "complete", "accepted-scoped"
		return *r, nil
	}
	return *r, errors.New("bounded amendment rounds ended without a terminal report")
}

func pinSourceProvenance(source *government.Source, path string, data []byte) {
	digest := government.BytesDigest(data)
	for i := range source.Schemas {
		source.Schemas[i].Source.Path, source.Schemas[i].Source.Digest = path, digest
	}
	for i := range source.Definitions {
		source.Definitions[i].Source.Path, source.Definitions[i].Source.Digest = path, digest
	}
}

func resetAmendmentWorkspace(workspace, temporary string, material *snapshot.Snapshot) error {
	workspaceAbs, err := filepath.Abs(workspace)
	if err != nil {
		return err
	}
	temporaryAbs, err := filepath.Abs(temporary)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(temporaryAbs, workspaceAbs)
	if err != nil || relative == "." || filepath.IsAbs(relative) || strings.Contains(relative, string(filepath.Separator)) || strings.HasPrefix(relative, "..") || !strings.HasPrefix(filepath.Base(workspaceAbs), "government-candidate-") {
		return errors.New("amendment workspace is outside its generated temporary directory")
	}
	info, err := os.Lstat(workspaceAbs)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return errors.New("amendment workspace is not a generated regular directory")
	}
	workspaceParent, err := filepath.EvalSymlinks(filepath.Dir(workspaceAbs))
	if err != nil {
		return err
	}
	temporaryReal, err := filepath.EvalSymlinks(temporaryAbs)
	if err != nil {
		return err
	}
	parentInfo, parentErr := os.Stat(workspaceParent)
	temporaryInfo, temporaryErr := os.Stat(temporaryReal)
	if parentErr != nil || temporaryErr != nil || !os.SameFile(parentInfo, temporaryInfo) {
		return errors.New("amendment workspace parent differs from configured temporary directory")
	}
	if err := os.RemoveAll(workspaceAbs); err != nil {
		return err
	}
	if err := os.Mkdir(workspaceAbs, 0700); err != nil {
		return err
	}
	empty := &snapshot.Snapshot{Files: map[string][]byte{}, Modes: map[string]string{}}
	return writeChanges(workspaceAbs, empty, material)
}
