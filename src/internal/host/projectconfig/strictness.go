package projectconfig

import (
	"fmt"
	"sort"
	"strings"
)

// ValidateStrictness checks profile shape independently of model/provider
// configuration. Empty profiles are the default normal level.
func ValidateStrictness(config *StrictnessConfig) error {
	if config == nil {
		return nil
	}
	if err := validateStrictnessProfile("default", config.Default); err != nil {
		return err
	}
	for managerID, profile := range config.Managers {
		if strings.TrimSpace(managerID) == "" {
			return fmt.Errorf("strictness Manager ID must not be empty")
		}
		if err := validateStrictnessProfile("Manager "+managerID, profile); err != nil {
			return err
		}
		if config.Default.Counterexamples+profile.Counterexamples > 20 {
			return fmt.Errorf("strictness Manager %s combined counterexamples must not exceed 20", managerID)
		}
		if len(unionEvidence(config.Default.Evidence, profile.Evidence)) > 32 {
			return fmt.Errorf("strictness Manager %s combined evidence items must not exceed 32", managerID)
		}
	}
	return nil
}

func validateStrictnessProfile(name string, profile StrictnessProfile) error {
	if profile.Counterexamples < 0 || profile.Counterexamples > 20 {
		return fmt.Errorf("strictness %s counterexamples must be between 0 and 20", name)
	}
	if len(profile.Evidence) > 32 {
		return fmt.Errorf("strictness %s may request at most 32 evidence items", name)
	}
	seen := map[string]bool{}
	for _, evidence := range profile.Evidence {
		evidence = strings.TrimSpace(evidence)
		if evidence == "" || len(evidence) > 256 || seen[evidence] {
			return fmt.Errorf("strictness %s has an empty, oversized, or duplicate evidence item", name)
		}
		seen[evidence] = true
	}
	return nil
}

// ResolveStrictness combines the project minimum with additive Manager
// requirements. Counterexample requests add; evidence requests form a sorted
// set union. Mandatory checks are planned separately and cannot be removed by
// this profile.
func ResolveStrictness(runtime Runtime, managerID string) (StrictnessProfile, error) {
	if strings.TrimSpace(managerID) == "" {
		return StrictnessProfile{}, fmt.Errorf("Manager ID is required to resolve strictness")
	}
	if err := ValidateStrictness(runtime.Strictness); err != nil {
		return StrictnessProfile{}, err
	}
	var base, local StrictnessProfile
	if runtime.Strictness != nil {
		base = runtime.Strictness.Default
		local = runtime.Strictness.Managers[managerID]
	}
	return StrictnessProfile{Evidence: unionEvidence(base.Evidence, local.Evidence), Counterexamples: base.Counterexamples + local.Counterexamples}, nil
}

func unionEvidence(profiles ...[]string) []string {
	evidence := map[string]bool{}
	for _, profile := range profiles {
		for _, item := range profile {
			evidence[strings.TrimSpace(item)] = true
		}
	}
	out := make([]string, 0, len(evidence))
	for item := range evidence {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}
