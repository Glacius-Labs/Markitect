package host

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring/contentpackage"
	core "github.com/Glacius-Labs/Markitect/internal/host/compat/v0_13/kernel"
)

// DomainInput identifies one explicitly activated compiler definition. Imported
// content does not activate a domain or its policies by itself.
type DomainInput struct {
	APIVersion string `yaml:"apiVersion"`
	Name       string `yaml:"name"`
	Path       string `yaml:"path"`
	Package    string `yaml:"package,omitempty"`
}

func (p *Project) loadPackageInputs(config *authoring.Resource) error {
	var byteCount int64
	var fileCount int
	names := map[string]bool{}
	for _, pin := range config.Spec.Packages {
		if names[pin.Name] {
			return fmt.Errorf("package %q is pinned more than once", pin.Name)
		}
		names[pin.Name] = true
		data, exists := p.Snapshot.Files[pin.Archive]
		if !exists {
			return fmt.Errorf("package %q archive %q is missing from the selected snapshot", pin.Name, pin.Archive)
		}
		archive, err := contentpackage.Read(pin, data)
		if err != nil {
			return fmt.Errorf("package %q: %w", pin.Name, err)
		}
		for _, member := range archive.Files {
			byteCount += int64(len(member))
		}
		fileCount += len(archive.Files)
		if byteCount > 128<<20 {
			return fmt.Errorf("combined content packages exceed the 128 MiB limit")
		}
		if fileCount > 10_000 {
			return fmt.Errorf("combined content packages exceed the 10000 file limit")
		}
		p.PackageFiles[pin.Name] = archive.Files
		p.Resources = append(p.Resources, archive.Manifest)
		p.Resources = append(p.Resources, archive.Resources...)
	}
	return nil
}

func (p *Project) loadDomainInputs(config *authoring.Resource) (*core.Registry, error) {
	registry := authoring.NewRegistry()
	if len(config.Spec.Domains) > 64 {
		return nil, fmt.Errorf("project exceeds the 64 domain definition limit")
	}
	seen := map[string]bool{}
	var selectedBytes int
	for _, selected := range config.Spec.Domains {
		input := DomainInput{Path: selected}
		if strings.HasPrefix(selected, "package:") {
			origin, member, ok := strings.Cut(strings.TrimPrefix(selected, "package:"), "/")
			if !ok || origin == "" || p.PackageFiles[origin] == nil {
				return nil, fmt.Errorf("domain %q requires an explicitly pinned package", selected)
			}
			declared := false
			for _, resource := range p.Resources {
				if resource.Kind == "Package" && resource.Metadata.Name == origin {
					for _, name := range resource.Spec.Domains {
						if name == member {
							declared = true
						}
					}
				}
			}
			if !declared {
				return nil, fmt.Errorf("domain %q is not declared by its Package", selected)
			}
			input = DomainInput{Path: member, Package: origin}
		}
		if err := validateDomainPath(input.Path); err != nil {
			return nil, err
		}
		key := inputKey(input.Package, input.Path)
		if seen[key] {
			return nil, fmt.Errorf("domain definition %q is selected more than once", selected)
		}
		seen[key] = true
		data := p.fileBytes(input.Package, input.Path)
		if data == nil {
			return nil, fmt.Errorf("domain definition %q is missing from the selected snapshot", selected)
		}
		selectedBytes += len(data)
		if selectedBytes > 16<<20 {
			return nil, fmt.Errorf("selected domain definitions exceed the 16 MiB limit")
		}
		definition, err := authoring.ParseDomain(input.Path, data)
		if err != nil {
			return nil, fmt.Errorf("domain %q: %w", selected, err)
		}
		if err := registry.AddDomain(definition); err != nil {
			return nil, fmt.Errorf("domain %q: %w", selected, err)
		}
		input.APIVersion = definition.APIVersion
		input.Name = definition.Name
		p.DomainInputs = append(p.DomainInputs, input)
	}
	sort.Slice(p.DomainInputs, func(i, j int) bool {
		return inputKey(p.DomainInputs[i].Package, p.DomainInputs[i].Path) < inputKey(p.DomainInputs[j].Package, p.DomainInputs[j].Path)
	})
	return registry, nil
}

func validateDomainPath(name string) error {
	if name == "" || path.IsAbs(name) || path.Clean(name) != name || strings.ContainsAny(name, "\\:\x00") {
		return fmt.Errorf("unsafe domain definition path %q", name)
	}
	for _, part := range strings.Split(name, "/") {
		if part == "." || part == ".." || strings.EqualFold(part, ".git") {
			return fmt.Errorf("unsafe domain definition path %q", name)
		}
	}
	if path.Ext(name) != ".yaml" && path.Ext(name) != ".yml" {
		return fmt.Errorf("domain definition %q must be YAML", name)
	}
	return nil
}

func (p *Project) isDomainInput(origin, name string) bool {
	for _, input := range p.DomainInputs {
		if input.Package == origin && input.Path == name {
			return true
		}
	}
	return false
}
