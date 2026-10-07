package execution

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

func validateAmendmentConfigWriter(model government.Model, plan government.Plan, order government.Order, configPath string) error {
	if plan.Action != "amend-model" || plan.ModelPath != configPath {
		return errors.New("amend-model plan does not bind the exact trusted ConfigPath as ModelPath")
	}
	writer, ok := priorConfigWriter(model, configPath)
	if !ok {
		return errors.New("amend-model ConfigPath requires one prior canonical Artifact with an explicit Writer")
	}
	workOwnsConfig := false
	for _, work := range plan.Work {
		if work.Area.Key() != writer.Key() {
			continue
		}
		if amendmentHasString(work.Paths, configPath) {
			workOwnsConfig = true
		}
	}
	if !workOwnsConfig {
		return errors.New("amend-model ConfigPath is not assigned to its prior canonical Writer Area")
	}
	for _, requested := range order.Subjects {
		if priorMandateForSubject(model, writer, requested, "amend-model") == "" {
			return fmt.Errorf("amend-model ConfigPath Writer %s lacks prior amend-model authority for requested subject %s", writer.Key(), requested.Key())
		}
	}
	return nil
}

func priorConfigWriter(model government.Model, configPath string) (core.DefinitionIdentity, bool) {
	for _, definition := range model.Canonical.Definitions {
		if definition.APIVersion == government.APIVersion && definition.Kind == "Artifact" && fmt.Sprint(definition.Spec["path"]) == configPath {
			writer := identityValue(definition.Spec["writer"])
			return writer, writer.Name != ""
		}
	}
	return core.DefinitionIdentity{}, false
}

func priorMandateForSubject(model government.Model, area, subject core.DefinitionIdentity, action string) string {
	for _, definition := range model.Canonical.Definitions {
		if definition.APIVersion != government.APIVersion || definition.Kind != "Mandate" || identityValue(definition.Spec["area"]).Key() != area.Key() || !amendmentHasString(amendmentStrings(definition.Spec["actions"]), action) {
			continue
		}
		for _, scoped := range amendmentIdentities(definition.Spec["scope"]) {
			if scoped.Key() == subject.Key() {
				return definition.Identity().Key()
			}
		}
	}
	return ""
}

func amendmentMandateApplies(model government.Model, area core.DefinitionIdentity, subject string) bool {
	if subject == "" {
		return false
	}
	for _, definition := range model.Canonical.Definitions {
		if definition.APIVersion == government.APIVersion && definition.Kind == "Mandate" {
			if identityValue(definition.Spec["area"]).Key() == area.Key() && amendmentHasString(amendmentStrings(definition.Spec["actions"]), "amend-model") {
				for _, scoped := range amendmentIdentities(definition.Spec["scope"]) {
					if scoped.Key() == subject {
						return true
					}
				}
			}
		}
	}
	return false
}

func amendmentHasString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func amendmentStrings(value any) []string {
	data, _ := json.Marshal(value)
	var out []string
	_ = json.Unmarshal(data, &out)
	return out
}
func amendmentIdentities(value any) []core.DefinitionIdentity {
	data, _ := json.Marshal(value)
	var out []core.DefinitionIdentity
	_ = json.Unmarshal(data, &out)
	return out
}
