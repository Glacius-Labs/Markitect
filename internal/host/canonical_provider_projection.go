package host

import (
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/modules/agentrules"
	"sort"
	"strings"
)

// Translate canonical, exact selected policies to module-local mapping data.
// Related Definitions remain read-only context, never projected ownership.
func canonicalProviderProjectionInput(model core.Model, request canonical.ProjectionRequest) (agentrules.Input, error) {
	context, err := BuildCanonicalAgentContext(model, request, 2)
	if err != nil {
		return agentrules.Input{}, err
	}
	input := agentrules.Input{Provider: agentrules.Provider(request.Projector.Target), Schemas: context.Schemas, Definitions: request.Definitions, RelatedDefinitions: context.RelatedDefinitions, TargetPrefix: request.TargetPrefix, AllowedRoots: append([]string(nil), request.Projector.AllowedRoots...), RequestDigest: request.RequestDigest}
	for i, root := range input.AllowedRoots {
		input.AllowedRoots[i] = strings.TrimSuffix(root, "/")
	}
	policy := agentrules.ProjectionPolicy{ID: request.Projection.Identity().Key(), Provider: input.Provider}
	kinds := map[string]bool{}
	for _, def := range request.Definitions {
		policy.DefinitionIDs = append(policy.DefinitionIDs, def.Identity())
		kinds[def.APIVersion+"/"+def.Kind] = true
	}
	policies := append([]core.Definition(nil), request.Policies...)
	sort.Slice(policies, func(i, j int) bool { return policies[i].Identity().Key() < policies[j].Identity().Key() })
	guidance := []string{}
	for _, p := range policies {
		source, ok := p.Spec["sourceKind"].(map[string]any)
		if !ok {
			return input, fmt.Errorf("policy %s has no normalized Kind reference", p.Identity().Key())
		}
		api, _ := source["apiVersion"].(string)
		kind, _ := source["kind"].(string)
		target, _ := p.Spec["targetTechnology"].(string)
		text, _ := p.Spec["guidance"].(string)
		if target != string(input.Provider) || strings.TrimSpace(text) == "" {
			return input, fmt.Errorf("policy %s has missing or mismatched provider guidance", p.Identity().Key())
		}
		delete(kinds, api+"/"+kind)
		guidance = append(guidance, p.Identity().Key()+" ("+api+"/"+kind+"):\n"+text)
	}
	if len(kinds) > 0 {
		missing := []string{}
		for kind := range kinds {
			missing = append(missing, kind)
		}
		sort.Strings(missing)
		return input, fmt.Errorf("provider guidance missing for selected Kinds: %s", strings.Join(missing, ", "))
	}
	policy.Guidance = strings.Join(guidance, "\n\n")
	input.Policies = []agentrules.ProjectionPolicy{policy}
	return input, nil
}
