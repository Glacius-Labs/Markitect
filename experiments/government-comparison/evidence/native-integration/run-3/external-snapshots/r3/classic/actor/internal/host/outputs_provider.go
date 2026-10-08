package host

import "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"

// Provider findings belong to Agent Rules. Host supplies one fixed model,
// explicit source configuration and the same snapshot used by other modules.
func checkProviderAdapterInputs(p *Project) []core.Diagnostic { return checkAgentRules(p) }
