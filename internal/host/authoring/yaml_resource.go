package authoring

import (
	"fmt"
	"path"
	"regexp"
	"strings"

	"go.yaml.in/yaml/v3"
)

var adapterExecutable = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)

func validateAdapters(file string, n *yaml.Node) error {
	if err := requireSequence(file, n, "adapters"); err != nil {
		return err
	}
	seen := map[string]bool{}
	localProjectionCount := 0
	for _, item := range n.Content {
		if err := requireMapping(file, item, "adapter"); err != nil {
			return err
		}
		if err := checkKeys(file, item, set("name", "type", "version", "config")); err != nil {
			return err
		}
		if err := requireFields(file, item, "name", "type", "version"); err != nil {
			return err
		}
		values := map[string]string{}
		for _, field := range []string{"name", "type", "version"} {
			node := child(item, field)
			if err := checkScalar(file, node, "string"); err != nil {
				return err
			}
			if strings.TrimSpace(node.Value) == "" {
				return diagnostic(file, node.Line, "adapter %s must not be empty", field)
			}
			values[field] = node.Value
		}
		if !validName(values["name"]) {
			return diagnostic(file, child(item, "name").Line, "adapter name must be a DNS label of at most 63 characters")
		}
		if seen[values["name"]] {
			return diagnostic(file, child(item, "name").Line, "adapter name %q is duplicated", values["name"])
		}
		seen[values["name"]] = true
		config := child(item, "config")
		switch values["type"] {
		case "command":
			if config != nil {
				if err := validateAdapterConfig(file, config); err != nil {
					return err
				}
			}
		case "local-projection":
			localProjectionCount++
			if localProjectionCount > 1 {
				return diagnostic(file, child(item, "type").Line, "Project may register at most one local-projection adapter")
			}
			if values["version"] != "v1alpha1" {
				return diagnostic(file, child(item, "version").Line, "local-projection adapter version must be v1alpha1")
			}
			if config == nil {
				return diagnostic(file, item.Line, "local-projection adapter requires config")
			}
			if err := validateLocalProjectionConfig(file, config); err != nil {
				return err
			}
		default:
			return diagnostic(file, child(item, "type").Line, "unsupported adapter type %q", values["type"])
		}
	}
	return nil
}

func validateLocalProjectionConfig(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "local-projection config"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("contracts", "coverage")); err != nil {
		return err
	}
	if err := requireFields(file, n, "contracts", "coverage"); err != nil {
		return err
	}
	paths := make([]string, 0, 2)
	for _, field := range []string{"contracts", "coverage"} {
		value := child(n, field)
		if err := checkScalar(file, value, "string"); err != nil {
			return err
		}
		if strings.TrimSpace(value.Value) != value.Value || value.Value == "" {
			return diagnostic(file, value.Line, "local-projection %s path must be a nonempty exact path", field)
		}
		if err := validateLocalProjectionPath(value.Value); err != nil {
			return diagnostic(file, value.Line, "local-projection %s path: %v", field, err)
		}
		paths = append(paths, value.Value)
	}
	if strings.EqualFold(paths[0], paths[1]) || pathWithin(paths[0], paths[1]) || pathWithin(paths[1], paths[0]) {
		return diagnostic(file, n.Line, "local-projection contracts and coverage paths must be distinct and non-overlapping")
	}
	return nil
}

func validateLocalProjectionPath(value string) error {
	if strings.ContainsAny(value, "\\\\:*?[]{}<>|\"\x00") || path.IsAbs(value) || path.Clean(value) != value || value == "." || strings.HasSuffix(value, "/") {
		return fmt.Errorf("must be a normalized relative POSIX YAML file path")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." || strings.TrimSpace(part) != part || strings.HasSuffix(part, ".") || strings.EqualFold(part, ".git") {
			return fmt.Errorf("contains an unsafe path component")
		}
		base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
		if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" || (len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9') {
			return fmt.Errorf("contains a Windows-reserved path component")
		}
		for _, r := range part {
			if r < 32 {
				return fmt.Errorf("contains a control character")
			}
		}
	}
	return nil
}

func pathWithin(parent, child string) bool {
	return len(child) > len(parent) && strings.EqualFold(child[:len(parent)], parent) && child[len(parent)] == '/'
}

func validateAdapterConfig(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "adapter config"); err != nil {
		return err
	}
	if err := checkKeys(file, n, set("inputs", "parameters", "target", "observe", "plan", "verify", "apply", "allowApply", "timeoutSeconds", "outputLimitBytes")); err != nil {
		return err
	}
	if inputs := child(n, "inputs"); inputs != nil {
		if err := validateStrings(file, inputs, "adapter inputs", true); err != nil {
			return err
		}
		for _, item := range inputs.Content {
			if path.IsAbs(item.Value) || strings.ContainsAny(item.Value, "\\:") || path.Clean(item.Value) != item.Value || item.Value == "." {
				return diagnostic(file, item.Line, "adapter input path %q must be an exact relative POSIX path", item.Value)
			}
			for _, part := range strings.Split(item.Value, "/") {
				if part == ".." || part == "." {
					return diagnostic(file, item.Line, "adapter input path %q must not traverse directories", item.Value)
				}
			}
		}
	}
	for _, field := range []string{"observe", "plan", "verify", "apply"} {
		if command := child(n, field); command != nil {
			if err := validateAdapterCommand(file, command, field); err != nil {
				return err
			}
		}
	}
	allowApply := false
	if node := child(n, "allowApply"); node != nil {
		if err := checkScalar(file, node, "boolean"); err != nil {
			return err
		}
		if err := node.Decode(&allowApply); err != nil {
			return diagnostic(file, node.Line, "allowApply must be boolean")
		}
	}
	apply := child(n, "apply")
	if allowApply != (apply != nil) {
		return diagnostic(file, n.Line, "adapter apply command and allowApply: true must be configured together")
	}
	if allowApply {
		target := child(n, "target")
		if target == nil {
			return diagnostic(file, n.Line, "adapter target is required when apply is enabled")
		}
		if err := checkScalar(file, target, "string"); err != nil {
			return err
		}
		if strings.TrimSpace(target.Value) == "" {
			return diagnostic(file, target.Line, "adapter target must not be empty")
		}
	}
	if target := child(n, "target"); target != nil {
		if err := checkScalar(file, target, "string"); err != nil {
			return err
		}
		if strings.TrimSpace(target.Value) == "" {
			return diagnostic(file, target.Line, "adapter target must not be empty")
		}
	}
	for _, field := range []string{"timeoutSeconds", "outputLimitBytes"} {
		bounds := [2]int{1, 600}
		if field == "outputLimitBytes" {
			bounds = [2]int{1024, 10 * 1024 * 1024}
		}
		if value := child(n, field); value != nil {
			if err := checkScalar(file, value, "integer"); err != nil {
				return err
			}
			var number int
			if err := value.Decode(&number); err != nil || number < bounds[0] || number > bounds[1] {
				return diagnostic(file, value.Line, "adapter %s must be between %d and %d", field, bounds[0], bounds[1])
			}
		}
	}
	if params := child(n, "parameters"); params != nil {
		if err := requireMapping(file, params, "adapter parameters"); err != nil {
			return err
		}
	}
	return nil
}

func validateAdapterCommand(file string, n *yaml.Node, field string) error {
	if err := requireSequence(file, n, "adapter "+field); err != nil {
		return err
	}
	if len(n.Content) == 0 {
		return diagnostic(file, n.Line, "adapter %s command must not be empty", field)
	}
	for i, arg := range n.Content {
		if err := checkScalar(file, arg, "string"); err != nil {
			return err
		}
		if strings.TrimSpace(arg.Value) == "" {
			return diagnostic(file, arg.Line, "adapter %s arguments must not be empty", field)
		}
		if i == 0 && !adapterExecutable.MatchString(arg.Value) {
			return diagnostic(file, arg.Line, "adapter %s executable must be a bare executable name", field)
		}
	}
	return nil
}

func validateRefs(file string, n *yaml.Node, field string) error {
	if err := requireSequence(file, n, field); err != nil {
		return err
	}
	for _, item := range n.Content {
		if err := requireMapping(file, item, field+" reference"); err != nil {
			return err
		}
		if err := checkKeys(file, item, set("apiVersion", "kind", "name", "namespace", "package")); err != nil {
			return err
		}
		if err := requireFields(file, item, "name"); err != nil {
			return err
		}
		for _, key := range []string{"apiVersion", "kind", "name", "namespace", "package"} {
			if v := child(item, key); v != nil {
				if err := checkScalar(file, v, "string"); err != nil {
					return err
				}
				if key == "apiVersion" {
					parts := strings.Split(v.Value, "/")
					if len(parts) != 2 || parts[0] == "" || parts[1] == "" || strings.ContainsAny(v.Value, " \t\r\n") {
						return diagnostic(file, v.Line, "reference apiVersion must be a nonempty group/version")
					}
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
