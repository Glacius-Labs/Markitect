package host

import (
	"fmt"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
)

const localProjectionAdapterType = "local-projection"
const localProjectionAdapterVersion = "v1alpha1"

// RegisteredProjection returns the exact Project-owned config and coverage
// paths selected by its single local-projection adapter. It does not read
// either file; callers bind the returned paths through their normal snapshot
// input pipeline.
func RegisteredProjection(p *Project) (configPath, coveragePath string, active bool, err error) {
	if p == nil {
		return "", "", false, fmt.Errorf("project is required")
	}
	var projectResource *authoring.Resource
	for _, resource := range p.Resources {
		if resource == nil || resource.Kind != "Project" {
			continue
		}
		if projectResource != nil {
			return "", "", false, fmt.Errorf("project contains multiple Project resources")
		}
		projectResource = resource
	}
	if projectResource == nil {
		return "", "", false, fmt.Errorf("project resource is missing")
	}
	var registration *authoring.AdapterConfig
	for i := range projectResource.Spec.Adapters {
		adapter := &projectResource.Spec.Adapters[i]
		if adapter.Type != localProjectionAdapterType {
			continue
		}
		if registration != nil {
			return "", "", false, fmt.Errorf("Project registers multiple local-projection adapters")
		}
		registration = adapter
	}
	if registration == nil {
		return "", "", false, nil
	}
	if registration.Version != localProjectionAdapterVersion {
		return "", "", false, fmt.Errorf("local-projection adapter version must be %s", localProjectionAdapterVersion)
	}
	if len(registration.Config) != 2 {
		return "", "", false, fmt.Errorf("local-projection adapter config must contain exactly contracts and coverage")
	}
	configPath, ok := registration.Config["contracts"].(string)
	if !ok {
		return "", "", false, fmt.Errorf("local-projection contracts config must be an exact path string")
	}
	coveragePath, ok = registration.Config["coverage"].(string)
	if !ok {
		return "", "", false, fmt.Errorf("local-projection coverage config must be an exact path string")
	}
	for _, item := range []struct{ field, path string }{{"contracts", configPath}, {"coverage", coveragePath}} {
		if err := validateProjectionRegistrationPath(item.path); err != nil {
			return "", "", false, fmt.Errorf("local-projection %s path: %w", item.field, err)
		}
	}
	if strings.EqualFold(configPath, coveragePath) || projectionPathOverlap(configPath, coveragePath) {
		return "", "", false, fmt.Errorf("local-projection contracts and coverage paths must be distinct and non-overlapping")
	}
	return configPath, coveragePath, true, nil
}

func validateProjectionRegistrationPath(value string) error {
	if value == "" || strings.TrimSpace(value) != value || strings.ContainsAny(value, "\\\\:*?[]{}<>|\"\x00") || path.IsAbs(value) || path.Clean(value) != value || value == "." || strings.HasSuffix(value, "/") {
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

func projectionPathOverlap(left, right string) bool {
	return projectionPathWithin(left, right) || projectionPathWithin(right, left)
}

func projectionPathWithin(parent, child string) bool {
	return len(child) > len(parent) && strings.EqualFold(child[:len(parent)], parent) && child[len(parent)] == '/'
}
