package format

import (
	"strings"

	"go.yaml.in/yaml/v3"
)

func validateRefs(file string, n *yaml.Node, field string) error {
	if err := requireSequence(file, n, field); err != nil {
		return err
	}
	for _, item := range n.Content {
		if err := requireMapping(file, item, field+" reference"); err != nil {
			return err
		}
		if err := checkKeys(file, item, set("kind", "name", "namespace", "package")); err != nil {
			return err
		}
		if err := requireFields(file, item, "name"); err != nil {
			return err
		}
		for _, key := range []string{"kind", "name", "namespace", "package"} {
			if v := child(item, key); v != nil {
				if err := checkScalar(file, v, "string"); err != nil {
					return err
				}
				if (key == "name" || key == "namespace") && !validName(v.Value) {
					return diagnostic(file, v.Line, "reference %s must be a DNS label of at most 63 characters", key)
				}
			}
		}
		if field == "exports" {
			if child(item, "kind") == nil || child(item, "namespace") == nil || child(item, "package") != nil {
				return diagnostic(file, item.Line, "package exports require kind and namespace and must not name another package")
			}
		}
		if field == "uses" && child(item, "kind") == nil {
			return diagnostic(file, item.Line, "uses references require kind")
		}
		if field == "rules" && child(item, "kind") != nil && child(item, "kind").Value != "Rule" {
			return diagnostic(file, child(item, "kind").Line, "rules references must have kind Rule")
		}
	}
	return nil
}

func validateStrings(file string, n *yaml.Node, field string, unique bool) error {
	if err := requireSequence(file, n, field); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, item := range n.Content {
		if err := checkScalar(file, item, "string"); err != nil {
			return err
		}
		if unique && strings.TrimSpace(item.Value) == "" {
			return diagnostic(file, item.Line, "%s entries must not be empty", field)
		}
		if unique && seen[item.Value] {
			return diagnostic(file, item.Line, "%s entries must be unique", field)
		}
		seen[item.Value] = true
	}
	return nil
}
func uniqueNonemptyStrings(file string, n *yaml.Node, field string) error {
	return validateStrings(file, n, field, true)
}

func validateProviders(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "providers"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("codex", "claude")); err != nil {
		return err
	}
	for _, provider := range []string{"codex", "claude"} {
		p := child(n, provider)
		if p == nil {
			continue
		}
		if err := requireMapping(file, p, provider+" provider"); err != nil {
			return err
		}
		allowed := set(allowedProviderFields(provider)...)
		if err := checkKeys(file, p, allowed); err != nil {
			return err
		}
		for _, key := range []string{"model", "effort", "sandbox", "permissionMode"} {
			if v := child(p, key); v != nil {
				if _, ok := allowed[key]; ok {
					if err := checkScalar(file, v, "string"); err != nil {
						return err
					}
				}
			}
		}
		for _, field := range []string{"tools", "disallowedTools"} {
			if v := child(p, field); v != nil {
				if provider != "claude" {
					return diagnostic(file, v.Line, "%s is only valid for claude", field)
				}
				if err := validateStrings(file, v, "provider "+field, true); err != nil {
					return err
				}
			}
		}
		if v := child(p, "maxTurns"); v != nil {
			if provider != "claude" {
				return diagnostic(file, v.Line, "maxTurns is only valid for claude")
			}
			if err := checkScalar(file, v, "integer"); err != nil {
				return err
			}
			var turns int
			if err := v.Decode(&turns); err != nil {
				return diagnostic(file, v.Line, "maxTurns must be an integer")
			}
			if turns < 0 {
				return diagnostic(file, v.Line, "maxTurns must not be negative")
			}
		}
	}
	return nil
}

func allowedProviderFields(provider string) []string {
	if provider == "codex" {
		return []string{"model", "effort", "sandbox"}
	}
	return []string{"model", "effort", "permissionMode", "tools", "disallowedTools", "maxTurns"}
}
