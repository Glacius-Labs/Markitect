package projectadoption

import (
	"fmt"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/host/projectcoverage"
)

// AssessReadiness reports source/model discussion status without treating
// partial adoption or transitional files as repository conformance.
func AssessReadiness(session BrownfieldSession, coverage *projectcoverage.Report) Readiness {
	result := Readiness{
		SessionDigest: session.Digest, SourceCurrent: false, TargetCurrent: false,
		Scopes: append([]ScopeStatus{}, session.Scopes...), UnresolvedConflicts: []SessionConflict{}, DeferredConflicts: []SessionConflict{},
		BlockingQuestions: []Question{}, Findings: []string{},
	}
	if ValidateBrownfieldSession(session) != nil {
		result.Findings = append(result.Findings, "session ledger is invalid")
		return result
	}
	active := activeIterationTree(session)
	for i := len(session.Iterations) - 1; i >= 0; i-- {
		if !active[session.Iterations[i].ID] {
			continue
		}
		integration := session.Iterations[i].Integration
		if integration == nil {
			continue
		}
		for _, conflict := range integration.Conflicts {
			if conflict.Disposition != "unresolved" {
				continue
			}
			decision := ""
			if session.Iterations[i].Resolution != nil {
				for _, scope := range session.Iterations[i].Resolution.Scopes {
					if scope.ScopeID == conflict.ScopeID {
						decision = scope.Status
					}
				}
			}
			switch decision {
			case "defer":
				result.DeferredConflicts = append(result.DeferredConflicts, conflict)
			case "adopt":
				answered := false
				if session.Iterations[i].Resolution != nil {
					for _, answer := range session.Iterations[i].Resolution.Questions {
						if answer.QuestionID == conflict.QuestionID && answer.Disposition == "answer" {
							answered = true
						}
					}
				}
				if !answered {
					result.UnresolvedConflicts = append(result.UnresolvedConflicts, conflict)
				}
			default:
				result.UnresolvedConflicts = append(result.UnresolvedConflicts, conflict)
			}
		}
		resolvedQuestions := map[string]bool{}
		if session.Iterations[i].Resolution != nil {
			for _, q := range session.Iterations[i].Resolution.Questions {
				if q.Disposition == "answer" {
					resolvedQuestions[q.QuestionID] = true
				}
			}
		}
		for _, question := range integration.Report.Questions {
			if question.Blocking != nil && *question.Blocking && !resolvedQuestions[question.ID] {
				result.BlockingQuestions = append(result.BlockingQuestions, question)
			}
		}
	}
	if coverage == nil {
		result.Findings = append(result.Findings, "complete repository coverage report is unavailable")
	} else {
		result.CoverageAccounted = coverage.Accounted
		result.CoverageConforming = coverage.Accounted && coverage.Conforming
		if !coverage.Accounted {
			result.Findings = append(result.Findings, "one or more repository paths remain unclassified")
		}
		if !coverage.Conforming {
			result.Findings = append(result.Findings, "repository coverage is not conforming")
		}
	}
	for _, scope := range result.Scopes {
		switch scope.Status {
		case "transitional":
			result.Findings = append(result.Findings, "scope "+scope.ScopeID+" is explicitly transitional")
		case "unresolved":
			result.Findings = append(result.Findings, "scope "+scope.ScopeID+" has unresolved evidence or ownership")
		case "observed":
			result.Findings = append(result.Findings, "scope "+scope.ScopeID+" has not completed reverse modeling")
		case "modeled":
			result.Findings = append(result.Findings, "scope "+scope.ScopeID+" has a proposal but no owner adoption")
		}
	}
	if len(result.UnresolvedConflicts) > 0 {
		result.Findings = append(result.Findings, "reverse-model conflicts remain unresolved")
	}
	if len(result.DeferredConflicts) > 0 {
		result.Findings = append(result.Findings, "reverse-model conflicts were deferred")
	}
	if len(result.BlockingQuestions) > 0 {
		result.Findings = append(result.Findings, "blocking reverse-model questions remain unanswered")
	}
	sort.Slice(result.Scopes, func(i, j int) bool { return result.Scopes[i].ScopeID < result.Scopes[j].ScopeID })
	sort.Slice(result.UnresolvedConflicts, func(i, j int) bool { return result.UnresolvedConflicts[i].ID < result.UnresolvedConflicts[j].ID })
	sort.Slice(result.DeferredConflicts, func(i, j int) bool { return result.DeferredConflicts[i].ID < result.DeferredConflicts[j].ID })
	sort.Slice(result.BlockingQuestions, func(i, j int) bool { return result.BlockingQuestions[i].ID < result.BlockingQuestions[j].ID })
	sort.Strings(result.Findings)
	finalizeReadiness(&result, session)
	return result
}

func finalizeReadiness(result *Readiness, session BrownfieldSession) {
	complete, reason := activeTreeComplete(session)
	if !complete {
		result.Findings = appendUnique(result.Findings, reason)
	}
	sort.Strings(result.Findings)
	result.Ready = result.SourceCurrent && result.TargetCurrent && complete && result.CoverageAccounted && result.CoverageConforming && len(result.Scopes) > 0 && len(result.UnresolvedConflicts) == 0 && len(result.DeferredConflicts) == 0 && len(result.BlockingQuestions) == 0
	for _, scope := range result.Scopes {
		if scope.Status != "adopted" {
			result.Ready = false
		}
	}
}

func activeTreeComplete(session BrownfieldSession) (bool, string) {
	root, ok := activeRootIteration(session)
	if !ok {
		return false, "active reverse-model tree has no root iteration"
	}
	if root.Proposal == nil {
		return false, "active root iteration has no Manager proposal"
	}
	if root.Integration == nil {
		return false, "active root iteration has no integrated Manager report"
	}
	if root.Resolution == nil {
		return false, "active root iteration has no owner resolution"
	}
	var visit func(ReverseIteration) (bool, string)
	visit = func(parent ReverseIteration) (bool, string) {
		if parent.Proposal == nil {
			return false, fmt.Sprintf("active Manager iteration %q has no proposal", parent.ID)
		}
		for _, assignment := range parent.Proposal.Hierarchy {
			var child *ReverseIteration
			for i := range session.Iterations {
				candidate := &session.Iterations[i]
				if candidate.ParentIterationID == parent.ID && candidate.ManagerID == assignment.ID {
					child = candidate
				}
			}
			if child == nil {
				return false, fmt.Sprintf("active Manager %q has no iteration for assigned child %q", parent.ManagerID, assignment.ID)
			}
			if child.Proposal == nil {
				return false, fmt.Sprintf("active child Manager %q has no proposal", child.ManagerID)
			}
			if len(child.Proposal.Hierarchy) > 0 && child.Integration == nil {
				return false, fmt.Sprintf("active non-leaf Manager %q has no integration", child.ManagerID)
			}
			if len(child.Proposal.Hierarchy) > 0 {
				if complete, reason := visit(*child); !complete {
					return false, reason
				}
			}
		}
		return true, ""
	}
	return visit(root)
}

func activeRootIteration(session BrownfieldSession) (ReverseIteration, bool) {
	for i := len(session.Iterations) - 1; i >= 0; i-- {
		if session.Iterations[i].ParentIterationID == "" {
			return session.Iterations[i], true
		}
	}
	return ReverseIteration{}, false
}

func appendUnique(values []string, value string) []string {
	for _, existing := range values {
		if existing == value {
			return values
		}
	}
	return append(values, value)
}

// activeIterationTree selects the latest root pass and the descendants it
// explicitly assigned. Earlier root passes and their children remain in the
// immutable ledger but no longer gate readiness after a clarified iteration.
func activeIterationTree(session BrownfieldSession) map[string]bool {
	active := map[string]bool{}
	rootID := ""
	for i := len(session.Iterations) - 1; i >= 0; i-- {
		if session.Iterations[i].ParentIterationID == "" {
			rootID = session.Iterations[i].ID
			break
		}
	}
	if rootID == "" {
		return active
	}
	active[rootID] = true
	for _, iteration := range session.Iterations {
		if iteration.ParentIterationID != "" && active[iteration.ParentIterationID] {
			active[iteration.ID] = true
		}
	}
	return active
}
