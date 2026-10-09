package projectcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/host/projectadoption"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectwork"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

const maxResolutionChoicesBytes = 1 << 20

type resolutionChoices struct {
	Actor             string                               `json:"actor"`
	AuthorityClaim    string                               `json:"authorityClaim"`
	DecisionReference string                               `json:"decisionReference"`
	Questions         []projectadoption.QuestionResolution `json:"questions"`
	Scopes            []projectadoption.ScopeResolution    `json:"scopes"`
}

// decodeResolutionChoices accepts only human decisions. All binding fields,
// including Authenticated and every source/target digest, are Host-derived.
func decodeResolutionChoices(data []byte) (resolutionChoices, error) {
	var choices resolutionChoices
	if len(data) == 0 || len(data) > maxResolutionChoicesBytes || !utf8.Valid(data) {
		return choices, fmt.Errorf("resolution choices must be valid UTF-8 between 1 and %d bytes", maxResolutionChoicesBytes)
	}
	if err := validateResolutionChoiceJSON(data); err != nil {
		return choices, err
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&choices); err != nil {
		return resolutionChoices{}, fmt.Errorf("decode resolution choices: %w", err)
	}
	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return resolutionChoices{}, errors.New("resolution choices must contain exactly one JSON object")
		}
		return resolutionChoices{}, fmt.Errorf("invalid trailing resolution choices: %w", err)
	}
	if strings.TrimSpace(choices.Actor) == "" || strings.TrimSpace(choices.AuthorityClaim) == "" || strings.TrimSpace(choices.DecisionReference) == "" || choices.Questions == nil || choices.Scopes == nil {
		return resolutionChoices{}, errors.New("choices must include actor, authorityClaim, decisionReference, questions, and scopes; use [] when no questions apply")
	}
	return choices, nil
}

func buildResolution(discovery projectadoption.Discovery, report projectadoption.Distillation, target *projectwork.Project, choices resolutionChoices) (projectadoption.Resolution, error) {
	var result projectadoption.Resolution
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

func validateResolutionChoiceJSON(data []byte) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	root := map[string]string{
		"actor": "scalar", "authorityClaim": "scalar", "decisionReference": "scalar",
		"questions": "questions", "scopes": "scopes",
	}
	first, err := decoder.Token()
	if err != nil {
		return fmt.Errorf("invalid choices JSON: %w", err)
	}
	if err := scanChoiceValue(decoder, first, "root", root); err != nil {
		return fmt.Errorf("invalid closed resolution choices: %w", err)
	}
	if _, err := decoder.Token(); err != io.EOF {
		if err == nil {
			return errors.New("resolution choices must contain exactly one JSON object")
		}
		return fmt.Errorf("invalid trailing choices JSON: %w", err)
	}
	return nil
}

func scanChoiceValue(decoder *json.Decoder, first json.Token, kind string, fields map[string]string) error {
	delimiter, isDelimiter := first.(json.Delim)
	switch kind {
	case "root", "question", "scope":
		if !isDelimiter || delimiter != '{' {
			return fmt.Errorf("%s must be a JSON object", kind)
		}
		seen := map[string]bool{}
		for decoder.More() {
			keyToken, err := decoder.Token()
			if err != nil {
				return err
			}
			key, ok := keyToken.(string)
			if !ok {
				return errors.New("JSON object key must be a string")
			}
			if seen[key] {
				return fmt.Errorf("duplicate JSON field %q", key)
			}
			for prior := range seen {
				if strings.EqualFold(prior, key) {
					return fmt.Errorf("case-aliased JSON fields %q and %q", prior, key)
				}
			}
			childKind, ok := fields[key]
			if !ok {
				return fmt.Errorf("unknown or incorrectly cased JSON field %q", key)
			}
			seen[key] = true
			child, err := decoder.Token()
			if err != nil {
				return err
			}
			var childFields map[string]string
			switch childKind {
			case "questions":
				childFields = map[string]string{"questionId": "scalar", "scopeId": "scalar", "disposition": "scalar", "answer": "scalar", "reason": "scalar"}
			case "scopes":
				childFields = map[string]string{"scopeId": "scalar", "status": "scalar", "reason": "scalar"}
			}
			if err := scanChoiceValue(decoder, child, childKind, childFields); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	case "questions", "scopes":
		if !isDelimiter || delimiter != '[' {
			return fmt.Errorf("%s must be a JSON array", kind)
		}
		itemKind := "question"
		itemFields := map[string]string{"questionId": "scalar", "scopeId": "scalar", "disposition": "scalar", "answer": "scalar", "reason": "scalar"}
		if kind == "scopes" {
			itemKind = "scope"
			itemFields = map[string]string{"scopeId": "scalar", "status": "scalar", "reason": "scalar"}
		}
		for decoder.More() {
			item, err := decoder.Token()
			if err != nil {
				return err
			}
			if err := scanChoiceValue(decoder, item, itemKind, itemFields); err != nil {
				return err
			}
		}
		_, err := decoder.Token()
		return err
	default:
		if isDelimiter {
			return errors.New("decision fields must be scalar values")
		}
		return nil
	}
}
