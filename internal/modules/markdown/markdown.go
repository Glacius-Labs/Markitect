package markdown

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

// Input is a Host-bounded semantic projection request. TargetPath is the exact
// artifact path selected by canonical projection intent.
type Input struct {
	Definitions       []core.Definition
	Schemas           []core.Schema
	Policies          []core.Definition
	TargetPath        string
	TargetPrefix      string
	AllowedRoots      []string
	RequestDigest     string
	CanonicalAffected bool
	InventoryComplete bool
	Previous          *PriorProjection
	ObservedArtifacts []ArtifactObservation
	Verification      *VerificationBinding
}

// Result contains proposed bytes and structural diagnostics. It does not apply
// the artifact or claim semantic acceptance.
type Result struct {
	Files       map[string][]byte
	Diagnostics []core.Diagnostic
}

// Render builds one generic Markdown representation for the selected scope.
// It presents Kind and Property purposes, contracts, and normalized Definition
// values without assigning ontology-specific meaning to any Kind.
func Render(input Input) Result {
	result := Result{Files: map[string][]byte{}}
	if strings.TrimSpace(input.TargetPath) == "" {
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Code: "markdown.target.empty", Message: "an exact target path is required"})
		return result
	}
	if len(input.Definitions) == 0 {
		result.Diagnostics = append(result.Diagnostics, core.Diagnostic{Code: "markdown.scope.empty", Message: "the selected projection scope contains no Definitions"})
		return result
	}

	kinds := make(map[string]core.Kind)
	schemaPurposes := make(map[string]string)
	for _, schema := range input.Schemas {
		schemaPurposes[schema.APIVersion] = schema.Purpose
		for name, kind := range schema.Kinds {
			kinds[(core.KindIdentity{APIVersion: schema.APIVersion, Kind: name}).Key()] = kind
		}
	}
	definitions := append([]core.Definition(nil), input.Definitions...)
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Identity().Key() < definitions[j].Identity().Key()
	})

	var out bytes.Buffer
	out.WriteString("# Canonical projection\n")
	for _, definition := range definitions {
		identity := definition.Identity()
		kind, found := kinds[(core.KindIdentity{APIVersion: definition.APIVersion, Kind: definition.Kind}).Key()]
		if !found {
			result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
				Code: "markdown.kind-missing", Identity: identity.Key(),
				Message: "the explicitly supplied Schemas do not contain this Definition Kind",
				Source:  definition.Source,
			})
		}
		out.WriteString("\n## ")
		out.WriteString(definition.Metadata.Name)
		out.WriteString("\n\n")
		fmt.Fprintf(&out, "Kind: %s (%s)  \nNamespace: %s\n\n", definition.Kind, definition.APIVersion, definition.Metadata.Namespace)
		out.WriteString("Purpose: ")
		out.WriteString(definition.Purpose)
		out.WriteString("\n")
		if found {
			out.WriteString("\nSchema purpose: ")
			out.WriteString(schemaPurposes[definition.APIVersion])
			out.WriteString("\nKind purpose: ")
			out.WriteString(kind.Purpose)
			out.WriteString("\n")
			writePropertyContracts(&out, kind.Properties, "")
		}
		if len(definition.Spec) > 0 {
			encoded, err := json.MarshalIndent(definition.Spec, "", "  ")
			if err != nil {
				result.Diagnostics = append(result.Diagnostics, core.Diagnostic{
					Code: "markdown.value.encode", Identity: identity.Key(),
					Message: "Definition values could not be represented as JSON",
					Source:  definition.Source,
				})
				return result
			}
			out.WriteString("\n### Values\n\n")
			out.WriteString("~~~json\n")
			out.Write(encoded)
			out.WriteString("\n~~~\n")
		}
	}
	if len(input.Policies) > 0 {
		policies := append([]core.Definition(nil), input.Policies...)
		sort.Slice(policies, func(i, j int) bool { return policies[i].Identity().Key() < policies[j].Identity().Key() })
		out.WriteString("\n## Projection policies\n")
		for _, policy := range policies {
			out.WriteString("\n### ")
			out.WriteString(policy.Metadata.Name)
			out.WriteString("\n\n")
			out.WriteString(policy.Purpose)
			if encoded, err := json.MarshalIndent(policy.Spec, "", "  "); err == nil {
				out.WriteString("\n\n~~~json\n")
				out.Write(encoded)
				out.WriteString("\n~~~\n")
			}
		}
	}
	result.Files[input.TargetPath] = []byte(out.String())
	return result
}

func writePropertyContracts(out *bytes.Buffer, properties map[string]core.Property, prefix string) {
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return
	}
	out.WriteString("\n### Properties\n\n")
	for _, name := range names {
		property := properties[name]
		fullName := name
		if prefix != "" {
			fullName = prefix + "." + name
		}
		fmt.Fprintf(out, "- `%s` (%s, %d..%s): %s\n", fullName, property.Type, property.MinCount, maxCount(property.MaxCount), property.Purpose)
		if property.Type == core.TypeObject {
			writeNestedProperties(out, property.Properties, fullName)
		}
	}
}

func writeNestedProperties(out *bytes.Buffer, properties map[string]core.Property, prefix string) {
	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		property := properties[name]
		fullName := prefix + "." + name
		fmt.Fprintf(out, "  - `%s` (%s, %d..%s): %s\n", fullName, property.Type, property.MinCount, maxCount(property.MaxCount), property.Purpose)
		if property.Type == core.TypeObject {
			writeNestedProperties(out, property.Properties, fullName)
		}
	}
}

func maxCount(count int) string {
	if count == core.Unbounded {
		return "unbounded"
	}
	return fmt.Sprint(count)
}
