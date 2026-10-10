package projectapp

import (
	"errors"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

// ResolutionChoices holds only human decisions. Every binding field of the
// sealed Resolution, including Authenticated and each source and target
// digest, is derived by the Host.
type ResolutionChoices struct {
	Actor             string                               `json:"actor"`
	AuthorityClaim    string                               `json:"authorityClaim"`
	DecisionReference string                               `json:"decisionReference"`
	Questions         []projectadoption.QuestionResolution `json:"questions"`
	Scopes            []projectadoption.ScopeResolution    `json:"scopes"`
}

func (c ResolutionChoices) validate() error {
	if strings.TrimSpace(c.Actor) == "" || strings.TrimSpace(c.AuthorityClaim) == "" || strings.TrimSpace(c.DecisionReference) == "" || c.Questions == nil || c.Scopes == nil {
		return errors.New("choices must include actor, authorityClaim, decisionReference, questions, and scopes; use [] when no questions apply")
	}
	return nil
}

// resolveFromChoices builds the sealed Resolution for one integrated
// iteration from the session's own discovery and integration report.
func resolveFromChoices(session projectadoption.BrownfieldSession, target *projectwork.Project, request BrownfieldResolveInput) (projectadoption.Resolution, error) {
	for _, iteration := range session.Iterations {
		if iteration.ID == request.IterationID {
			if iteration.Integration == nil {
				return projectadoption.Resolution{}, errors.New("resolution requires an integrated iteration")
			}
			return buildResolution(session.Source, iteration.Integration.Report, target, request.Choices)
		}
	}
	return projectadoption.Resolution{}, fmt.Errorf("unknown adoption iteration %q", request.IterationID)
}

// buildResolution seals a Resolution from human choices against the fixed
// discovery, distillation report and target project.
func buildResolution(discovery projectadoption.Discovery, report projectadoption.Distillation, target *projectwork.Project, choices ResolutionChoices) (projectadoption.Resolution, error) {
	var result projectadoption.Resolution
	if err := choices.validate(); err != nil {
		return result, err
	}
	if target == nil || target.Snapshot == nil || target.Provisional || target.Revision == "" {
		return result, errors.New("resolution requires a committed fixed target project")
	}
	if err := projectadoption.ValidateDistillationTarget(report, target); err != nil {
		return result, fmt.Errorf("distillation target binding does not match the fixed target project: %w", err)
	}
	actorValid := choices.Actor == projectwork.HumanActor
	for _, manager := range target.Report.Managers {
		if manager.ID == choices.Actor {
			actorValid = true
			break
		}
	}
	if !actorValid {
		return result, errors.New("actor must be user or an active Manager ID in the fixed target model")
	}
	schemaDigest, buildDigest, err := projectadoption.CurrentBindings(projectmodel.Schema())
	if err != nil {
		return result, fmt.Errorf("derive active schema/build bindings: %w", err)
	}
	authenticated := false
	result = projectadoption.Resolution{
		APIVersion:      projectadoption.ResolutionVersion,
		DiscoveryDigest: discovery.Digest, DistillationDigest: report.Digest,
		ProposalDigest: projectadoption.ProposalDigest(report.Proposal), TargetBasis: target.Digest,
		SchemaDigest: schemaDigest, BuildDigest: buildDigest,
		Actor: choices.Actor, AuthorityClaim: choices.AuthorityClaim, DecisionReference: choices.DecisionReference,
		Authenticated: &authenticated, Questions: append([]projectadoption.QuestionResolution{}, choices.Questions...),
		Scopes: append([]projectadoption.ScopeResolution{}, choices.Scopes...),
	}
	projectadoption.SealResolution(&result)
	if err := projectadoption.ValidateResolution(discovery, report, result); err != nil {
		return projectadoption.Resolution{}, fmt.Errorf("validate explicit resolution choices: %w", err)
	}
	return result, nil
}
