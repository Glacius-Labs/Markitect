package projectrun

import (
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
)

const (
	OperationApply     = "apply"
	OperationCleanup   = "cleanup"
	OperationReconcile = "reconcile"
)

// StrictnessConfig contains a project minimum and additive Manager-specific
// requirements. It changes review depth and evidence requested, never the
// accepted model or mandatory project checks.
type StrictnessConfig struct {
	Default  StrictnessProfile            `json:"default" yaml:"default"`
	Managers map[string]StrictnessProfile `json:"managers,omitempty" yaml:"managers,omitempty"`
}

// StrictnessProfile describes additional evidence requested from a Manager.
// Counterexamples is an additive count on top of the project default.
type StrictnessProfile struct {
	Evidence        []string `json:"evidence,omitempty" yaml:"evidence,omitempty"`
	Counterexamples int      `json:"counterexamples,omitempty" yaml:"counterexamples,omitempty"`
}

// NormalizeOperation supplies the historical targeted-implementation
// behavior when a caller omits the operation.
func NormalizeOperation(operation string) (string, error) {
	operation = strings.TrimSpace(operation)
	if operation == "" {
		return OperationApply, nil
	}
	switch operation {
	case OperationApply, OperationCleanup, OperationReconcile:
		return operation, nil
	default:
		return "", fmt.Errorf("unsupported project operation %q", operation)
	}
}

// OperationGuidance returns the operation-specific mandate appended to each
// Manager phase context. The caller still supplies ownership and phase rules.
func OperationGuidance(operation string) string {
	switch operation {
	case OperationCleanup:
		return "Cleanup mandate: improve the existing realization within the accepted semantic model. Do not change business rules, decisions, public promises, authority, inventory, or required checks. You may make a bounded quality improvement or return a justified no-op when none is warranted. Report concrete evidence for each improvement and preserve all required artifacts."
	case OperationReconcile:
		return "Reconcile mandate: compare this Manager's complete responsibility against the entire accepted model, including obligations and required artifacts absent from the known change impact. Implement missing or divergent realizations. Conforming areas may return a reasoned no-op. Preserve the accepted model and escalate unknown scope or requirements rather than treating them as unaffected."
	default:
		return "Apply mandate: implement the bounded requested change within this Manager's assigned responsibility and the accepted model."
	}
}

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

func validateStrictnessManagers(config *StrictnessConfig, managers []projectmodel.Manager) error {
	if config == nil {
		return nil
	}
	known := make(map[string]bool, len(managers))
	for _, manager := range managers {
		known[manager.ID] = true
	}
	for managerID := range config.Managers {
		if !known[managerID] {
			return fmt.Errorf("strictness profile references Manager absent from the accepted model: %s", managerID)
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
