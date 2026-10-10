package authoring

import "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"

// NewRegistry builds the source frontend's vocabulary over an otherwise empty
// Core registry. Project, Package and Domain are opaque source envelope kinds;
// only the AI content vocabulary carries relations into the normalized model.
func NewRegistry() *core.Registry {
	r := core.NewRegistry()
	empty := core.KindDefinition{Properties: map[string]core.PropertyDefinition{}}
	for _, kind := range []string{"Project", "Package", "Domain"} {
		if err := r.RegisterKind(core.APIVersion, kind, empty); err != nil {
			panic(err)
		}
	}
	ref := core.PropertyDefinition{Type: "array", Items: &core.PropertyDefinition{Type: "ref", RefKind: "*"}}
	relations := map[string]core.RelationDefinition{
		"rules":      {Field: "rules", SourceKinds: []string{"Rule", "Text", "Contract", "Workflow", "Skill", "Agent"}, TargetKinds: []string{"Rule"}, Context: true, Invalidate: true},
		"uses":       {Field: "uses", SourceKinds: []string{"Workflow", "Skill", "Agent"}, TargetKinds: []string{"*"}, Context: true, Invalidate: true},
		"needs":      {Field: "needs", SourceKinds: []string{"Workflow", "Skill", "Agent"}, TargetKinds: []string{"Contract"}, Context: true, Invalidate: true},
		"implements": {Field: "implements", SourceKinds: []string{"Workflow", "Skill", "Agent"}, TargetKinds: []string{"Contract"}, Context: true, Invalidate: true},
	}
	kinds := map[string]core.KindDefinition{}
	for _, kind := range []string{"Text", "Rule", "Contract", "Workflow", "Skill", "Agent"} {
		props := map[string]core.PropertyDefinition{}
		for _, field := range []string{"rules", "uses", "needs", "implements"} {
			props[field] = ref
		}
		kinds[kind] = core.KindDefinition{Properties: props}
	}
	if err := r.AddDomain(core.DomainDefinition{APIVersion: core.APIVersion, Kinds: kinds, Relations: relations}); err != nil {
		panic("invalid Host authoring vocabulary: " + err.Error())
	}
	return r
}
