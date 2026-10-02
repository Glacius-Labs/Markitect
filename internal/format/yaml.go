// Package format reads and writes Markitect YAML resources.
package format

import (
	"bytes"
	"io"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

const maxResourceSize = 2 << 20

var dnsLabel = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]*[a-z0-9])?$`)
var sha256Pattern = regexp.MustCompile(`^[0-9a-f]{64}$`)

var specFields = map[string][]string{
	"Text":     {"text", "files", "assertions"},
	"Rule":     {"text", "check", "files", "assertions"},
	"Contract": {"text", "kind", "input", "output", "files", "assertions"},
	"Workflow": {"text", "rules", "uses", "needs", "implements", "input", "output", "files", "assertions"},
	"Skill":    {"text", "description", "rules", "uses", "needs", "implements", "input", "output", "files", "assertions"},
	"Agent":    {"text", "description", "rules", "uses", "needs", "implements", "input", "output", "providers", "files", "assertions"},
	"Project":  {"targets", "areas", "bindings", "checks", "documentation", "ruleAdapters", "providerAdapters", "consistency", "packages", "domains", "adapters"},
	"Package":  {"version", "areas", "exports", "bindings", "domains"},
}

// AllowedSpecFields returns the accepted spec fields for a resource kind.
// Unknown kinds return nil. The returned slice is independent of the format package.
func AllowedSpecFields(kind string) []string {
	fields := specFields[kind]
	return append([]string(nil), fields...)
}

// Parse decodes exactly one strict Markitect YAML resource from data.
func Parse(filePath string, data []byte) (*core.Resource, error) {
	return ParseWithRegistry(filePath, data, core.NewRegistry())
}

// ParseWithRegistry decodes one strict resource using the project's loaded domain vocabulary.
func ParseWithRegistry(filePath string, data []byte, registry *core.Registry) (*core.Resource, error) {
	if registry == nil {
		registry = core.NewRegistry()
	}
	if len(data) > maxResourceSize {
		return nil, diagnostic(filePath, 1, "resource exceeds the 2 MiB limit")
	}
	dec := yaml.NewDecoder(bytes.NewReader(data))
	var doc yaml.Node
	if err := dec.Decode(&doc); err != nil {
		return nil, diagnostic(filePath, yamlErrorLine(err), "invalid YAML: %v", err)
	}
	if len(doc.Content) == 0 || doc.Content[0].Kind == 0 || isNull(doc.Content[0]) {
		return nil, diagnostic(filePath, maxInt(1, doc.Line), "empty YAML document")
	}
	if err := inspectNode(filePath, &doc); err != nil {
		return nil, err
	}
	var extra yaml.Node
	err := dec.Decode(&extra)
	if err == nil {
		line := extra.Line
		if line == 0 {
			line = doc.Line + 1
		}
		return nil, diagnostic(filePath, line, "multiple YAML documents are not allowed")
	}
	if err != io.EOF {
		return nil, diagnostic(filePath, yamlErrorLine(err), "invalid YAML: %v", err)
	}

	root := doc.Content[0]
	if err := requireMapping(filePath, root, "resource"); err != nil {
		return nil, err
	}
	if err := checkKeys(filePath, root, set("apiVersion", "kind", "metadata", "spec")); err != nil {
		return nil, err
	}
	if err := requireFields(filePath, root, "apiVersion", "kind", "metadata", "spec"); err != nil {
		return nil, err
	}
	if err := checkScalar(filePath, child(root, "apiVersion"), "string"); err != nil {
		return nil, err
	}
	if err := checkScalar(filePath, child(root, "kind"), "string"); err != nil {
		return nil, err
	}
	kind := child(root, "kind").Value
	apiVersion := child(root, "apiVersion").Value
	if kind == "Domain" && apiVersion == core.APIVersion {
		d, err := ParseDomain(filePath, data)
		if err != nil {
			return nil, err
		}
		return &core.Resource{APIVersion: core.APIVersion, Kind: "Domain", Metadata: core.Metadata{Name: d.Name}, Data: map[string]any{}, Path: filePath, Line: root.Line}, nil
	}
	if !registry.IsKnownKind(apiVersion, kind) {
		return nil, diagnostic(filePath, child(root, "kind").Line, "unsupported resource kind %q for apiVersion %q", kind, apiVersion)
	}
	if err := validateMetadata(filePath, child(root, "metadata"), kind); err != nil {
		return nil, err
	}
	if apiVersion == core.APIVersion {
		if err := validateSpec(filePath, child(root, "spec"), kind); err != nil {
			return nil, err
		}
	} else if err := validateGenericSpec(filePath, child(root, "spec"), registry, apiVersion, kind); err != nil {
		return nil, err
	}
	var r core.Resource
	if err := child(root, "metadata").Decode(&r.Metadata); err != nil {
		return nil, diagnostic(filePath, child(root, "metadata").Line, "invalid metadata: %v", err)
	}
	r.APIVersion = apiVersion
	r.Kind = kind
	var spec map[string]any
	if err := child(root, "spec").Decode(&spec); err != nil {
		return nil, diagnostic(filePath, child(root, "spec").Line, "invalid spec: %v", err)
	}
	if err := r.SetData(spec); err != nil {
		return nil, diagnostic(filePath, child(root, "spec").Line, "invalid spec: %v", err)
	}
	if apiVersion != core.APIVersion {
		definition, _ := registry.Lookup(apiVersion, kind)
		if definition.InputsField != "" {
			if values, ok := spec[definition.InputsField].([]any); ok {
				for _, value := range values {
					if path, ok := value.(string); ok {
						r.Spec.Files = append(r.Spec.Files, path)
					}
				}
			}
		}
		if text, ok := spec["text"].(string); ok {
			r.Spec.Text = text
		}
		if description, ok := spec["description"].(string); ok {
			r.Spec.Description = description
		}
	}
	r.Path = filePath
	r.Line = root.Line
	return &r, nil
}

// Encode serializes a Go value as YAML.
func Encode(value any) ([]byte, error) { return yaml.Marshal(value) }

func validateMetadata(file string, n *yaml.Node, kind string) error {
	if err := requireMapping(file, n, "metadata"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("name", "namespace", "labels")); err != nil {
		return err
	}
	if err := requireFields(file, n, "name"); err != nil {
		return err
	}
	if (kind == "Project" || kind == "Package" || kind == "Domain") && child(n, "namespace") != nil {
		return diagnostic(file, child(n, "namespace").Line, "%s metadata must not have a namespace", kind)
	}
	if labels := child(n, "labels"); labels != nil {
		if err := requireMapping(file, labels, "metadata labels"); err != nil {
			return err
		}
		for i := 0; i+1 < len(labels.Content); i += 2 {
			key, value := labels.Content[i], labels.Content[i+1]
			if err := checkScalar(file, key, "string"); err != nil {
				return err
			}
			if err := checkScalar(file, value, "string"); err != nil {
				return err
			}
		}
	}
	for _, field := range []string{"name", "namespace"} {
		v := child(n, field)
		if v == nil {
			continue
		}
		if err := checkScalar(file, v, "string"); err != nil {
			return err
		}
		if !validName(v.Value) {
			return diagnostic(file, v.Line, "%s must be a DNS label of at most 63 characters", field)
		}
	}
	return nil
}

func validateSpec(file string, n *yaml.Node, kind string) error {
	if err := requireMapping(file, n, "spec"); err != nil {
		return err
	}
	allowed := AllowedSpecFields(kind)
	keys := make(map[string]struct{}, len(allowed))
	for _, k := range allowed {
		keys[k] = struct{}{}
	}
	if err := checkKeys(file, n, keys); err != nil {
		return err
	}
	if kind != "Project" {
		if kind == "Package" {
			if err := requireFields(file, n, "version", "areas", "exports"); err != nil {
				return err
			}
			if err := checkScalar(file, child(n, "version"), "string"); err != nil {
				return err
			}
			if strings.TrimSpace(child(n, "version").Value) == "" {
				return diagnostic(file, child(n, "version").Line, "Package spec version must not be empty")
			}
		}
		if kind == "Package" {
			// Package manifests describe a graph boundary and have no renderable prose.
		} else {
			text := child(n, "text")
			if text == nil {
				return diagnostic(file, n.Line, "%s spec requires nonempty text", kind)
			}
			if err := checkScalar(file, text, "string"); err != nil {
				return err
			}
			if strings.TrimSpace(text.Value) == "" {
				return diagnostic(file, text.Line, "%s spec text must not be empty", kind)
			}
		}
	}
	if d := child(n, "description"); d != nil {
		if err := checkScalar(file, d, "string"); err != nil {
			return err
		}
	}
	if d := child(n, "check"); d != nil {
		if err := checkScalar(file, d, "string"); err != nil {
			return err
		}
	}
	if d := child(n, "kind"); d != nil {
		if err := checkScalar(file, d, "string"); err != nil {
			return err
		}
		if kind == "Contract" && !oneOf(d.Value, "Agent", "Workflow", "Skill") {
			return diagnostic(file, d.Line, "Contract kind must be Agent, Workflow, or Skill")
		}
	}
	for _, field := range []string{"input", "output", "targets"} {
		if d := child(n, field); d != nil {
			if err := validateStrings(file, d, field, field == "targets"); err != nil {
				return err
			}
		}
	}
	if d := child(n, "files"); d != nil {
		if kind == "Project" {
			return diagnostic(file, d.Line, "Project spec must not declare files")
		}
		if err := validateStrings(file, d, "files", true); err != nil {
			return err
		}
	}
	for _, field := range []string{"rules", "uses", "needs", "implements", "exports"} {
		if d := child(n, field); d != nil {
			if err := validateRefs(file, d, field); err != nil {
				return err
			}
		}
	}
	if d := child(n, "packages"); d != nil {
		if kind != "Project" {
			return diagnostic(file, d.Line, "packages are only allowed on Project resources")
		}
		if err := validatePackagePins(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "domains"); d != nil {
		if kind != "Project" && kind != "Package" {
			return diagnostic(file, d.Line, "domains are only allowed on Project or Package resources")
		}
		if err := validateDomainPaths(file, d, kind == "Project"); err != nil {
			return err
		}
	}
	if d := child(n, "adapters"); d != nil {
		if kind != "Project" {
			return diagnostic(file, d.Line, "adapters are only allowed on Project resources")
		}
		if err := validateAdapters(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "documentation"); d != nil {
		if err := validateDocumentation(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "exports"); d != nil && kind != "Package" {
		return diagnostic(file, d.Line, "exports are only allowed on Package resources")
	}
	if d := child(n, "version"); d != nil && kind != "Package" {
		return diagnostic(file, d.Line, "version is only allowed on Package resources")
	}
	if kind == "Contract" {
		for _, field := range []string{"input", "output"} {
			if d := child(n, field); d != nil {
				if err := uniqueNonemptyStrings(file, d, field); err != nil {
					return err
				}
			}
		}
	}
	if d := child(n, "providers"); d != nil {
		if err := validateProviders(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "areas"); d != nil {
		if err := validateAreas(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "bindings"); d != nil {
		if err := validateBindings(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "checks"); d != nil {
		if err := validateChecks(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "ruleAdapters"); d != nil {
		if err := validateRuleAdapters(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "providerAdapters"); d != nil {
		if err := validateProviderAdapters(file, d); err != nil {
			return err
		}
	}
	if d := child(n, "consistency"); d != nil {
		if err := requireMapping(file, d, "consistency"); err != nil {
			return err
		}
		if err := checkKeys(file, d, set("functionalPredicates")); err != nil {
			return err
		}
		if v := child(d, "functionalPredicates"); v != nil {
			if err := validateStrings(file, v, "functionalPredicates", true); err != nil {
				return err
			}
		}
	}
	if d := child(n, "assertions"); d != nil {
		if err := requireSequence(file, d, "assertions"); err != nil {
			return err
		}
		for _, a := range d.Content {
			if err := requireMapping(file, a, "assertion"); err != nil {
				return err
			}
			if err := checkKeys(file, a, set("subject", "predicate", "value", "source", "quote")); err != nil {
				return err
			}
			if err := requireFields(file, a, "subject", "predicate", "value", "source", "quote"); err != nil {
				return err
			}
			for _, field := range []string{"subject", "predicate", "value", "source", "quote"} {
				v := child(a, field)
				if err := checkScalar(file, v, "string"); err != nil {
					return err
				}
				if strings.TrimSpace(v.Value) == "" {
					return diagnostic(file, v.Line, "assertion %s must not be empty", field)
				}
			}
		}
	}
	return nil
}

func validateProviderAdapters(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "providerAdapters"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("ruleSources", "agentContract", "roleRegister", "inlineAgentText", "strictInventory", "retiredSkills", "retiredAgents")); err != nil {
		return err
	}
	for _, field := range []string{"agentContract", "roleRegister"} {
		if v := child(n, field); v != nil {
			if err := checkScalar(file, v, "string"); err != nil {
				return err
			}
		}
	}
	for _, field := range []string{"inlineAgentText", "strictInventory"} {
		if v := child(n, field); v != nil {
			if err := checkScalar(file, v, "boolean"); err != nil {
				return err
			}
		}
	}
	for _, field := range []string{"retiredSkills", "retiredAgents"} {
		if v := child(n, field); v != nil {
			if err := validateStrings(file, v, field, true); err != nil {
				return err
			}
		}
	}
	if rules := child(n, "ruleSources"); rules != nil {
		if err := requireMapping(file, rules, "ruleSources"); err != nil {
			return err
		}
		for i := 0; i < len(rules.Content); i += 2 {
			if err := checkScalar(file, rules.Content[i], "string"); err != nil {
				return err
			}
			if err := validateStrings(file, rules.Content[i+1], "ruleSources", true); err != nil {
				return err
			}
		}
	}
	return nil
}
