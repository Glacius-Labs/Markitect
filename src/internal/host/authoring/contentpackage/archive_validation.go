package contentpackage

import (
	"fmt"
	"path"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func parseDomainDefinitions(manifest *authoring.Resource, files map[string][]byte) (*core.Registry, map[string]core.DomainDefinition, error) {
	registry := authoring.NewRegistry()
	definitions := make(map[string]core.DomainDefinition, len(manifest.Spec.Domains))
	for _, domainPath := range manifest.Spec.Domains {
		data, exists := files[domainPath]
		if !exists {
			return nil, nil, fmt.Errorf("declared package domain %q is missing", domainPath)
		}
		definition, err := authoring.ParseDomain(domainPath, data)
		if err != nil {
			return nil, nil, fmt.Errorf("parse package domain %q: %w", domainPath, err)
		}
		if err := registry.AddDomain(definition); err != nil {
			return nil, nil, fmt.Errorf("register package domain %q: %w", domainPath, err)
		}
		definitions[domainPath] = definition
	}
	return registry, definitions, nil
}

func validateManifest(manifest *authoring.Resource) error {
	if manifest.Spec.Version == "" {
		return fmt.Errorf("package manifest requires spec.version")
	}
	if hasPackageRefs(manifest) {
		return fmt.Errorf("package manifest cannot contain transitive package references")
	}
	if len(manifest.Spec.Checks) != 0 || len(manifest.Spec.Targets) != 0 || len(manifest.Spec.RuleAdapters) != 0 || len(manifest.Spec.Packages) != 0 {
		return fmt.Errorf("package manifest must not declare checks, targets, ruleAdapters, or nested packages")
	}
	areas := map[string]bool{}
	areaPaths := make([]string, 0, len(manifest.Spec.Areas))
	for _, area := range manifest.Spec.Areas {
		areas[area.Name] = true
		areaPaths = append(areaPaths, area.Path)
	}
	for _, area := range manifest.Spec.Areas {
		for _, imported := range area.Imports {
			if !areas[imported] {
				return fmt.Errorf("package area %q imports unknown package-local area %q", area.Name, imported)
			}
		}
	}
	domainPaths := make(map[string]string, len(manifest.Spec.Domains))
	for _, domainPath := range manifest.Spec.Domains {
		if err := validatePath(domainPath); err != nil {
			return fmt.Errorf("invalid package domain path %q: %w", domainPath, err)
		}
		if !isYAML(domainPath) {
			return fmt.Errorf("package domain path %q must be a YAML file", domainPath)
		}
		folded := strings.ToLower(domainPath)
		if previous, exists := domainPaths[folded]; exists {
			return fmt.Errorf("package domain paths %q and %q are duplicated or collide by case", previous, domainPath)
		}
		domainPaths[folded] = domainPath
		if domainPath == ManifestName {
			return fmt.Errorf("package manifest cannot also be a domain definition")
		}
		if inAnyAreaFolded(domainPath, areaPaths) {
			return fmt.Errorf("package domain %q must not be inside a content area", domainPath)
		}
	}
	for _, binding := range manifest.Spec.Bindings {
		if binding.Contract.Package != "" || binding.Implementation.Package != "" {
			return fmt.Errorf("package bindings cannot reference another package")
		}
	}
	for _, export := range manifest.Spec.Exports {
		if export.Package != "" {
			return fmt.Errorf("package exports cannot reference another package")
		}
	}
	return nil
}

func inAnyAreaFolded(filePath string, roots []string) bool {
	folded := strings.ToLower(filePath)
	for _, root := range roots {
		root = strings.ToLower(root)
		if folded == root || strings.HasPrefix(folded, root+"/") {
			return true
		}
	}
	return false
}

func declaredInputs(resources []*authoring.Resource) map[string]bool {
	inputs := make(map[string]bool)
	for _, resource := range resources {
		for _, filePath := range resource.Spec.Files {
			inputs[filePath] = true
		}
	}
	return inputs
}

func hasPackageRefs(resource *authoring.Resource) bool {
	check := func(ref core.Ref) bool { return ref.Package != "" }
	for _, refs := range [][]core.Ref{resource.Spec.Rules, resource.Spec.Uses, resource.Spec.Needs, resource.Spec.Implements} {
		for _, ref := range refs {
			if check(ref) {
				return true
			}
		}
	}
	for _, area := range resource.Spec.Areas {
		for _, ref := range area.Rules {
			if check(ref) {
				return true
			}
		}
	}
	for _, binding := range resource.Spec.Bindings {
		if check(binding.Contract) || check(binding.Implementation) {
			return true
		}
	}
	for _, refs := range resource.Spec.RuleAdapters {
		for _, ref := range refs {
			if check(ref) {
				return true
			}
		}
	}
	return false
}

func inAnyArea(filePath string, areas []string) bool {
	for _, root := range areas {
		if filePath == root || strings.HasPrefix(filePath, root+"/") {
			return true
		}
	}
	return false
}

func isYAML(filePath string) bool {
	extension := strings.ToLower(path.Ext(filePath))
	return extension == ".yaml" || extension == ".yml"
}

func validatePath(filePath string) error {
	if filePath == "" || !utf8.ValidString(filePath) || strings.Contains(filePath, "\\") || strings.HasPrefix(filePath, "/") || path.Clean(filePath) != filePath {
		return fmt.Errorf("path must be a normalized relative POSIX path")
	}
	for _, segment := range strings.Split(filePath, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return fmt.Errorf("path has an unsafe segment")
		}
		for _, ch := range segment {
			if ch < 0x20 || strings.ContainsRune(`<>:"|?*`, ch) {
				return fmt.Errorf("path contains a non-portable character")
			}
		}
		base := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
		if isWindowsDeviceName(base) {
			return fmt.Errorf("path contains a reserved Windows device name")
		}
	}
	return nil
}

func isWindowsDeviceName(base string) bool {
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return true
	}
	return false
}
