package projectrun

import (
	"fmt"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/projectmodel"
	"strings"
)

const (
	OperationApply     = "apply"
	OperationCleanup   = "cleanup"
	OperationReconcile = "reconcile"
)

// NormalizeOperation supplies the historical targeted-implementation behavior when omitted.
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

// OperationGuidance returns the operation-specific mandate appended to each Manager phase context.
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

// validateStrictnessManagers binds otherwise model-independent profiles to the accepted model.
func validateStrictnessManagers(config *projectconfig.StrictnessConfig, managers []projectmodel.Manager) error {
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
