package projectadoption

import (
	"errors"
	"fmt"
	"sort"
)

const ManagerIntegrationContextVersion = "markitect.example.org/manager-integration-context/v1alpha1"

// ManagerIntegrationContext gives a parent only its own bounded context and
// the completed direct-child outputs it must integrate. It never serializes
// the Brownfield session or unassigned Discovery evidence.
type ManagerIntegrationContext struct {
	APIVersion     string                    `json:"apiVersion"`
	SessionDigest  string                    `json:"sessionDigest"`
	IterationID    string                    `json:"iterationId"`
	Parent         ManagerReverseContext     `json:"parent"`
	ParentProposal ManagerProposal           `json:"parentProposal"`
	Children       []ManagerIntegrationChild `json:"children"`
	Digest         string                    `json:"digest"`
}

// ManagerIntegrationChild contains a child's final report and public
// contracts. A non-leaf child contributes its integrated report and exact
// integration digest; a leaf contributes its proposal report.
type ManagerIntegrationChild struct {
	ManagerID         string                  `json:"managerId"`
	ProposalDigest    string                  `json:"proposalDigest"`
	IntegrationDigest string                  `json:"integrationDigest,omitempty"`
	ReportDigest      string                  `json:"reportDigest"`
	Report            Distillation            `json:"report"`
	PublicContracts   []ManagerPublicContract `json:"publicContracts"`
}

// ChildIntegrationDigest binds a parent's report to the final direct-child
// report after that child has integrated its own descendants.
type ChildIntegrationDigest struct {
	ManagerID         string `json:"managerId"`
	ProposalDigest    string `json:"proposalDigest"`
	IntegrationDigest string `json:"integrationDigest"`
	ReportDigest      string `json:"reportDigest"`
}

// BuildManagerIntegrationContext creates the bounded input for a Manager's
// integration phase. Every assigned direct child must have a proposal; a
// non-leaf child must also have integrated its own children first.
func BuildManagerIntegrationContext(session BrownfieldSession, parentIterationID string) (ManagerIntegrationContext, error) {
	if err := ValidateBrownfieldSession(session); err != nil {
		return ManagerIntegrationContext{}, err
	}
	active := activeIterationTree(session)
	parent, exists := findIteration(session, parentIterationID)
	if !exists || !active[parentIterationID] || parent.Proposal == nil {
		return ManagerIntegrationContext{}, errors.New("integration context requires an active parent iteration with a recorded proposal")
	}
	if err := validateIntegratedChildBranches(session, parent); err != nil {
		return ManagerIntegrationContext{}, err
	}
	parentContext, err := BuildManagerReverseContext(session, parentIterationID)
	if err != nil {
		return ManagerIntegrationContext{}, err
	}
	result := ManagerIntegrationContext{APIVersion: ManagerIntegrationContextVersion, SessionDigest: session.Digest, IterationID: parentIterationID,
		Parent: parentContext, ParentProposal: *parent.Proposal, Children: []ManagerIntegrationChild{}}
	assignments := append([]ProposedManager(nil), parent.Proposal.Hierarchy...)
	sort.Slice(assignments, func(i, j int) bool { return assignments[i].ID < assignments[j].ID })
	for _, assignment := range assignments {
		var child *ReverseIteration
		for i := range session.Iterations {
			candidate := &session.Iterations[i]
			if candidate.ParentIterationID == parentIterationID && candidate.ManagerID == assignment.ID {
				child = candidate
				break
			}
		}
		if child == nil || !active[child.ID] || child.Proposal == nil {
			return ManagerIntegrationContext{}, fmt.Errorf("parent Manager %q cannot integrate before assigned child %q has a proposal", parent.ManagerID, assignment.ID)
		}
		if len(child.Proposal.Hierarchy) > 0 && child.Integration == nil {
			return ManagerIntegrationContext{}, fmt.Errorf("parent Manager %q cannot integrate before non-leaf child %q integrates its own descendants", parent.ManagerID, child.ManagerID)
		}
		report := child.Proposal.Report
		integrationDigest := ""
		if child.Integration != nil {
			report = child.Integration.Report
			integrationDigest = child.Integration.Digest
		}
		result.Children = append(result.Children, ManagerIntegrationChild{ManagerID: child.ManagerID, ProposalDigest: child.Proposal.Digest,
			IntegrationDigest: integrationDigest, ReportDigest: report.Digest, Report: report, PublicContracts: append([]ManagerPublicContract{}, child.Proposal.PublicContracts...)})
	}
	copy := result
	copy.Digest = ""
	result.Digest = digestValue(copy)
	return result, nil
}

func validateIntegratedChildBranches(session BrownfieldSession, parent ReverseIteration) error {
	if parent.Proposal == nil {
		return fmt.Errorf("Manager iteration %q has no proposal", parent.ID)
	}
	for _, assignment := range parent.Proposal.Hierarchy {
		var child *ReverseIteration
		for i := range session.Iterations {
			candidate := &session.Iterations[i]
			if candidate.ParentIterationID == parent.ID && candidate.ManagerID == assignment.ID {
				child = candidate
				break
			}
		}
		if child == nil || child.Proposal == nil {
			return fmt.Errorf("Manager %q cannot integrate before assigned child %q has a proposal", parent.ManagerID, assignment.ID)
		}
		if len(child.Proposal.Hierarchy) > 0 {
			if child.Integration == nil {
				return fmt.Errorf("Manager %q cannot integrate before non-leaf child %q integrates its descendants", parent.ManagerID, child.ManagerID)
			}
			if err := validateIntegratedChildBranches(session, *child); err != nil {
				return err
			}
		}
	}
	return nil
}
