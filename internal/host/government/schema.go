// Package government implements the experimental Government Host contracts.
// Core remains a provider-neutral structural compiler; routing and authority
// interpretation live here and never confer authority from observed files.
package government

import "github.com/Glacius-Labs/Markitect/internal/core"

const APIVersion = "markitect.government/v1alpha1"
const SourceVersion = "markitect.government-source/v1alpha1"
const OrderVersion = "markitect.government-order/v1alpha1"

// Schema is the canonical structural contract, also exposed by the CLI. Full
// arbitrary Definition identities are values checked by Host, not polymorphic
// Core references. Every relation is its own purpose-bearing Definition.
func Schema() core.Schema {
	text := func(purpose string) core.Property {
		return core.Property{Purpose: purpose, Type: core.TypeString, MinCount: 1, MaxCount: 1}
	}
	ref := func(kind, purpose string, min, max int) core.Property {
		return core.Property{Purpose: purpose, Type: core.TypeReference, MinCount: min, MaxCount: max, Target: &core.KindIdentity{APIVersion: APIVersion, Kind: kind}}
	}
	id := func(purpose string, min, max int) core.Property {
		return core.Property{Purpose: purpose, Type: core.TypeObject, MinCount: min, MaxCount: max, Properties: map[string]core.Property{
			"apiVersion": text("Exact Schema version"), "kind": text("Exact Kind"), "namespace": text("Explicit namespace, including empty"), "name": text("Definition name"),
		}}
	}
	enum := func(purpose string, values ...string) core.Property {
		return core.Property{Purpose: purpose, Type: core.TypeEnum, MinCount: 1, MaxCount: 1, Values: values}
	}
	return core.Schema{APIVersion: APIVersion, Purpose: "Versioned responsibilities, prior authority and intended repository realization", Kinds: map[string]core.Kind{
		"Constitution": {Purpose: "Joint domain and organization authority explicitly selected by the trusted caller", Properties: map[string]core.Property{
			"root":              ref("Area", "Root integration responsibility", 1, 1),
			"cabinet":           ref("Ressort", "Frozen unanimous review electorate", 1, core.Unbounded),
			"approval":          enum("All selected Ressorts explicitly assent to the final material and evidence", "unanimous-explicit-assent"),
			"protectedSubjects": id("Exact domain identities that are root goals protected from delegated amendment", 0, core.Unbounded),
		}},
		"Area": {Purpose: "Recursive durable responsibility independent of technology", Properties: map[string]core.Property{
			"parent":       ref("Area", "Parent retains integration responsibility", 0, 1),
			"capabilities": ref("Capability", "Available technical skills confer no authority", 0, core.Unbounded),
		}},
		"Ressort": {Purpose: "Crosscutting reviewer, never an implicit file writer", Properties: map[string]core.Property{
			"mandate": ref("Mandate", "Pinned review authority", 1, 1),
		}},
		"Mandate": {Purpose: "Bounded prior delegation; child delegation must be a subset", Properties: map[string]core.Property{
			"area":    ref("Area", "Area holding the delegated responsibility", 1, 1),
			"parent":  ref("Mandate", "Prior higher delegation", 0, 1),
			"scope":   id("Explicit model subjects whose changes are delegated", 1, core.Unbounded),
			"actions": {Purpose: "Explicit allowed decision classes", Type: core.TypeEnum, MinCount: 1, MaxCount: core.Unbounded, Values: []string{"implement", "review", "amend-model"}},
		}},
		"Responsibility": {Purpose: "One accountable Area per subject, separate from realization and review", Properties: map[string]core.Property{
			"subject": id("Exact arbitrary Definition identity", 1, 1), "area": ref("Area", "Accountable Area", 1, 1),
		}},
		"Realization": {Purpose: "Many-to-many intended subject-to-file relation, without observed bytes", Properties: map[string]core.Property{
			"subject": id("Exact subject being realized", 1, 1), "artifact": ref("Artifact", "Intended concrete artifact", 1, 1),
			"role": text("Contribution of this file to the subject"),
		}},
		"Artifact": {Purpose: "Intended exact repository path and exclusive writer", Properties: map[string]core.Property{
			"path":   text("Exact normalized repository-relative path"),
			"class":  enum("Normative role, never inferred from file contents", "canonical", "realization", "generated", "foreign"),
			"writer": ref("Area", "Unique writer for managed material", 0, 1),
		}},
		"Capability": {Purpose: "Pinned tool or skill orthogonal to responsibility", Properties: map[string]core.Property{
			"tool": text("Explicit tool identity"), "version": text("Exact tool version"), "digest": text("Immutable tool or skill bytes digest"),
		}},
	}}
}
