package core

import (
	"encoding/hex"
	"fmt"
	"path"
	"regexp"
	"strings"
)

var packageNamePattern = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*[a-z0-9]$|^[a-z0-9]$`)

func (g *Graph) validatePackagePins() {
	pins := map[string]PackagePin{}
	archives := map[string]string{}
	for _, pin := range g.Project.Spec.Packages {
		if !packageNamePattern.MatchString(pin.Name) || len(pin.Name) > 63 {
			g.diag(g.Project, "package.name", fmt.Sprintf("package name %q must be a DNS label of at most 63 characters", pin.Name))
		}
		if !isExactSemVer(pin.Version) {
			g.diag(g.Project, "package.version", fmt.Sprintf("package %q version %q must be an exact semantic version", pin.Name, pin.Version))
		}
		if strings.TrimSpace(pin.Source) == "" || strings.ContainsAny(pin.Source, "\x00\r\n") {
			g.diag(g.Project, "package.source", fmt.Sprintf("package %q source must be a nonempty provenance coordinate", pin.Name))
		}
		if err := validatePackageArchive(pin.Archive); err != nil {
			g.diag(g.Project, "package.archive", fmt.Sprintf("package %q archive path is unsafe: %v", pin.Name, err))
		}
		folded := strings.ToLower(pin.Archive)
		if previous, ok := archives[folded]; ok {
			g.diag(g.Project, "package.archive-duplicate", fmt.Sprintf("package archive path %q is duplicated or case-collides with package %q", pin.Archive, previous))
		} else {
			archives[folded] = pin.Name
		}
		if len(pin.SHA256) != 64 || strings.ToLower(pin.SHA256) != pin.SHA256 {
			g.diag(g.Project, "package.digest", fmt.Sprintf("package %q sha256 must be 64 lowercase hexadecimal characters", pin.Name))
		} else if _, err := hex.DecodeString(pin.SHA256); err != nil {
			g.diag(g.Project, "package.digest", fmt.Sprintf("package %q sha256 must be 64 lowercase hexadecimal characters", pin.Name))
		}
		if previous, ok := pins[pin.Name]; ok {
			g.diag(g.Project, "package.duplicate-pin", fmt.Sprintf("package %q has more than one pin (%s and %s)", pin.Name, previous.Version, pin.Version))
			continue
		}
		pins[pin.Name] = pin
		manifest := g.Packages[pin.Name]
		if manifest == nil {
			g.diag(g.Project, "package.missing", fmt.Sprintf("pinned package %q has no verified Package manifest", pin.Name))
			continue
		}
		if manifest.Spec.Version != pin.Version {
			g.diag(manifest, "package.version-mismatch", fmt.Sprintf("package manifest version %q does not match pinned version %q", manifest.Spec.Version, pin.Version))
		}
	}
	for name, manifest := range g.Packages {
		if _, ok := pins[name]; !ok {
			g.diag(manifest, "package.unpinned", fmt.Sprintf("Package manifest %q is not selected by Project.spec.packages", name))
		}
		if !isExactSemVer(manifest.Spec.Version) {
			g.diag(manifest, "package.version", fmt.Sprintf("Package %q version %q must be an exact semantic version", name, manifest.Spec.Version))
		}
	}
	for _, resource := range g.Resources {
		if resource.Kind == "Project" || resource.Kind == "Package" {
			continue
		}
		if resource.Package == "" {
			continue
		}
		if _, ok := pins[resource.Package]; !ok {
			g.diag(resource, "package.unpinned-resource", fmt.Sprintf("resource origin package %q is not pinned by the Project", resource.Package))
		}
		if g.Packages[resource.Package] == nil {
			g.diag(resource, "package.manifest", fmt.Sprintf("resource origin package %q has no Package manifest", resource.Package))
		}
	}
}

func validatePackageArchive(value string) error {
	if value == "" || strings.ContainsAny(value, "\\:\x00") || strings.ContainsAny(value, "*?[]{}") || path.IsAbs(value) || path.Clean(value) != value || value == "." || strings.HasPrefix(value, "../") || strings.HasSuffix(value, "/") {
		return fmt.Errorf("path must be a normalized repository-relative ZIP file path")
	}
	for _, part := range strings.Split(value, "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("path contains an unsafe component")
		}
	}
	if path.Ext(value) != ".zip" {
		return fmt.Errorf("path must end in .zip")
	}
	return nil
}

func isExactSemVer(value string) bool {
	if value == "" || strings.ContainsAny(value, " \t\r\n\x00") {
		return false
	}
	coreAndPre := value
	if strings.Contains(value, "+") {
		parts := strings.SplitN(value, "+", 2)
		if len(parts) != 2 || strings.Contains(parts[1], "+") || !validSemVerIdentifiers(parts[1], false) {
			return false
		}
		coreAndPre = parts[0]
	}
	core := coreAndPre
	if strings.Contains(coreAndPre, "-") {
		parts := strings.SplitN(coreAndPre, "-", 2)
		if len(parts) != 2 || !validSemVerIdentifiers(parts[1], true) {
			return false
		}
		core = parts[0]
	}
	numbers := strings.Split(core, ".")
	if len(numbers) != 3 {
		return false
	}
	for _, number := range numbers {
		if number == "" || (len(number) > 1 && number[0] == '0') {
			return false
		}
		for _, char := range number {
			if char < '0' || char > '9' {
				return false
			}
		}
	}
	return true
}

func validSemVerIdentifiers(value string, prerelease bool) bool {
	if value == "" {
		return false
	}
	for _, identifier := range strings.Split(value, ".") {
		if identifier == "" {
			return false
		}
		numeric := true
		for _, char := range identifier {
			if (char < '0' || char > '9') && (char < 'A' || char > 'Z') && (char < 'a' || char > 'z') && char != '-' {
				return false
			}
			if char < '0' || char > '9' {
				numeric = false
			}
		}
		if prerelease && numeric && len(identifier) > 1 && identifier[0] == '0' {
			return false
		}
	}
	return true
}

// IsExported reports whether resource is an explicit package export.
func (g *Graph) IsExported(resource *Resource) bool {
	if g == nil || resource == nil || resource.Package == "" || resource.Kind == "Package" {
		return false
	}
	manifest := g.Packages[resource.Package]
	if manifest == nil {
		return false
	}
	for _, ref := range manifest.Spec.Exports {
		if ref.Package == "" && ref.GraphKey(resource.Package, "", "") == resource.GraphKey() {
			return true
		}
	}
	return false
}

func (g *Graph) validatePackageManifests() {
	for _, name := range sortedKeys(g.Packages) {
		manifest := g.Packages[name]
		if manifest.Metadata.Namespace != "" {
			g.diag(manifest, "package.namespace", "Package metadata must not have a namespace")
		}
		if manifest.Package != name || manifest.Metadata.Name != name {
			g.diag(manifest, "package.identity", "Package runtime origin and metadata.name must match")
		}
		areas := map[string]bool{}
		for _, area := range manifest.Spec.Areas {
			if area.Name == "" || area.Path == "" {
				g.diag(manifest, "area.invalid", "package area name and path are required")
				continue
			}
			if areas[area.Name] {
				g.diag(manifest, "area.duplicate", fmt.Sprintf("duplicate package area %q", area.Name))
			}
			areas[area.Name] = true
		}
		for _, area := range manifest.Spec.Areas {
			for _, imported := range area.Imports {
				if !areas[imported] {
					g.diag(manifest, "area.import", fmt.Sprintf("package area %q imports unknown package namespace %q", area.Name, imported))
				}
			}
		}
		exports := map[string]bool{}
		for _, ref := range manifest.Spec.Exports {
			if ref.Package != "" || ref.Namespace == "" || ref.Kind == "" || ref.Name == "" {
				g.diag(manifest, "package.export-qualified", "package exports must specify kind, namespace, and name without a package")
				continue
			}
			key := ref.GraphKey(name, "", "")
			if exports[key] {
				g.diag(manifest, "package.export-duplicate", fmt.Sprintf("package export %s is duplicated", key))
			}
			exports[key] = true
			resource := g.Resources[key]
			if resource == nil || resource.Kind == "Project" || resource.Kind == "Package" {
				g.diag(manifest, "package.export-missing", fmt.Sprintf("package export %s does not resolve to a content resource", key))
			}
		}
	}
}

func (g *Graph) projectPackagePin(name string) (PackagePin, bool) {
	if g.Project == nil {
		return PackagePin{}, false
	}
	for _, pin := range g.Project.Spec.Packages {
		if pin.Name == name {
			return pin, true
		}
	}
	return PackagePin{}, false
}

func (g *Graph) packageHasBinding(contractKey string) bool {
	for _, manifest := range g.Packages {
		for _, binding := range manifest.Spec.Bindings {
			if binding.Contract.GraphKey(manifest.Package, "", "") == contractKey {
				return true
			}
		}
	}
	return false
}
