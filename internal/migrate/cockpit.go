package migrate

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

const cockpitProjectPath = "markitect.yaml"

type cockpitOwner struct {
	name     string
	path     string
	imports  []string
	rules    []core.Ref
	parent   *cockpitOwner
	customer string
	project  string
	depth    int
}

// Cockpit plans co-located Markitect resources for the Cockpit documentation
// hierarchy. It preserves the existing provider ownership by omitting targets.
// The function is pure: it reads only the supplied immutable snapshot and
// returns relative output paths without writing files.
func Cockpit(snapshot *source.Snapshot) (map[string][]byte, error) {
	if snapshot == nil {
		return nil, fmt.Errorf("Cockpit migration snapshot is nil")
	}
	definitions, err := collectCockpitDefinitions(snapshot)
	if err != nil {
		return nil, err
	}
	owners, err := cockpitOwners(snapshot, definitions)
	if err != nil {
		return nil, err
	}
	if err := validateCockpitOwners(owners); err != nil {
		return nil, err
	}
	project, err := cockpitProject(definitions, owners)
	if err != nil {
		return nil, err
	}
	outputs := make(map[string][]byte, len(definitions)+1)
	outputNames := map[string]string{}
	snapshotNames := make(map[string]string, len(snapshot.Files))
	for name := range snapshot.Files {
		folded := strings.ToLower(name)
		if previous, exists := snapshotNames[folded]; exists && previous != name {
			return nil, fmt.Errorf("snapshot paths %q and %q collide case-insensitively", previous, name)
		}
		snapshotNames[folded] = name
	}
	projectBytes, err := format.Encode(project)
	if err != nil {
		return nil, fmt.Errorf("encode %s: %w", cockpitProjectPath, err)
	}
	outputs[cockpitProjectPath] = projectBytes
	outputNames[strings.ToLower(cockpitProjectPath)] = cockpitProjectPath
	for _, definition := range definitions {
		if !validCockpitSourcePath(definition.sourcePath) || !validCockpitSourcePath(definition.resource.Path) {
			return nil, fmt.Errorf("%s: migration paths must be clean repository-relative paths under docs/", definition.sourcePath)
		}
		folded := strings.ToLower(definition.resource.Path)
		if previous, exists := outputNames[folded]; exists {
			return nil, fmt.Errorf("migration output collision between %s and %s", previous, definition.resource.Path)
		}
		outputNames[folded] = definition.resource.Path
		if previous, exists := snapshotNames[folded]; exists {
			return nil, fmt.Errorf("migration output %s collides with existing snapshot path %s", definition.resource.Path, previous)
		}
		if _, exists := snapshot.Files[definition.resource.Path]; exists {
			return nil, fmt.Errorf("migration output %s already exists; refusing to overwrite an existing source", definition.resource.Path)
		}
		data, err := format.Encode(definition.resource)
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", definition.resource.Path, err)
		}
		outputs[definition.resource.Path] = data
	}
	if _, exists := snapshot.Files[cockpitProjectPath]; exists {
		return nil, fmt.Errorf("migration output %s already exists; refusing to overwrite an existing source", cockpitProjectPath)
	}
	if previous, exists := snapshotNames[strings.ToLower(cockpitProjectPath)]; exists {
		return nil, fmt.Errorf("migration output %s collides with existing snapshot path %s", cockpitProjectPath, previous)
	}
	return outputs, nil
}

func validCockpitSourcePath(value string) bool {
	return strings.HasPrefix(value, "docs/") && !strings.Contains(value, "\\") && path.Clean(value) == value && value != "docs"
}

func validateCockpitOwners(owners []cockpitOwner) error {
	names, paths := map[string]string{}, map[string]string{}
	for _, owner := range owners {
		name := strings.ToLower(owner.name)
		if previous, exists := names[name]; exists {
			return fmt.Errorf("Cockpit owner areas %q and %q collide case-insensitively", previous, owner.name)
		}
		names[name] = owner.name
		ownerPath := strings.ToLower(owner.path)
		if previous, exists := paths[ownerPath]; exists {
			return fmt.Errorf("Cockpit owner paths %q and %q collide case-insensitively", previous, owner.path)
		}
		paths[ownerPath] = owner.path
	}
	return nil
}

func collectCockpitDefinitions(snapshot *source.Snapshot) ([]definition, error) {
	paths := make([]string, 0)
	for file := range snapshot.Files {
		if _, ok := directMechanism(file); ok && !strings.EqualFold(path.Base(file), "README.md") {
			paths = append(paths, file)
		}
	}
	sort.Strings(paths)
	definitions := make([]definition, 0, len(paths))
	for _, file := range paths {
		kind, _ := directMechanism(file)
		data := snapshot.Files[file]
		name := strings.TrimSuffix(path.Base(file), path.Ext(file))
		description, body := "", normalizeLineEndings(data)
		providers := core.Providers{}
		if kind == "Skill" {
			metadata, content, err := parseSkill(file, data)
			if err != nil {
				return nil, err
			}
			if metadata.Name != name {
				return nil, fmt.Errorf("%s: metadata name %q must match filename %q", file, metadata.Name, name)
			}
			description, body = metadata.Description, normalizeLineEndings(content)
		} else if kind == "Agent" {
			metadata, content, err := parseAgent(file, data)
			if err != nil {
				return nil, err
			}
			if metadata.Name != name {
				return nil, fmt.Errorf("%s: metadata name %q must match filename %q", file, metadata.Name, name)
			}
			description, body, providers = metadata.Description, normalizeLineEndings(content), metadata.providers()
		}
		namespace, err := cockpitNamespace(file)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", file, err)
		}
		resource := &core.Resource{
			APIVersion: core.APIVersion,
			Kind:       kind,
			Metadata:   core.Metadata{Name: name, Namespace: namespace},
			Path:       strings.TrimSuffix(file, path.Ext(file)) + ".yaml",
			Spec:       core.Spec{Text: string(body), Description: description, Providers: providers},
		}
		definitions = append(definitions, definition{sourcePath: file, resource: resource})
	}
	if err := applyCockpitRelationships(definitions); err != nil {
		return nil, err
	}
	return definitions, nil
}

// applyCockpitRelationships applies a reviewed migration allowlist only when
// the canonical text still contains the named direct link. Other navigation
// remains prose and cannot silently become a dependency.
func applyCockpitRelationships(definitions []definition) error {
	type mapping struct {
		source string
		target string
		kind   string
	}
	reviewed := []mapping{
		{"docs/general/skills/author-mechanism.md", "docs/general/workflows/author-mechanism.md", "Workflow"},
		{"docs/general/skills/select-work-context.md", "docs/general/workflows/select-work-context.md", "Workflow"},
		{"docs/customers/septeo/projects/wz-assist/skills/wz-assist-work.md", "docs/customers/septeo/projects/wz-assist/workflows/delivery.md", "Workflow"},
		{"docs/general/agents/documentation-auditor.md", "docs/general/rules/documentation.md", "Rule"},
		{"docs/general/agents/documentation-auditor.md", "docs/general/rules/mechanisms.md", "Rule"},
	}
	targets := make(map[string]core.Ref, len(definitions))
	for _, item := range definitions {
		targets[item.sourcePath] = core.Ref{Kind: item.resource.Kind, Name: item.resource.Metadata.Name, Namespace: item.resource.Metadata.Namespace}
	}
	for _, item := range definitions {
		linked := map[string]bool{}
		for _, link := range markdownTargets.FindAllStringSubmatch(item.resource.Spec.Text, -1) {
			if targetPath, ok := normalizeCockpitTarget(item.sourcePath, link[1]); ok {
				linked[targetPath] = true
			}
		}
		for _, allow := range reviewed {
			if item.sourcePath != allow.source || !linked[allow.target] {
				continue
			}
			ref, exists := targets[allow.target]
			if !exists || ref.Kind != allow.kind {
				return fmt.Errorf("reviewed Cockpit relationship %s -> %s has no typed %s target", allow.source, allow.target, allow.kind)
			}
			if (item.resource.Kind == "Skill" && allow.kind == "Workflow") || (item.resource.Kind == "Agent" && allow.kind == "Rule") {
				if item.resource.Kind == "Skill" {
					item.resource.Spec.Uses = appendUniqueRef(item.resource.Spec.Uses, ref)
				} else {
					item.resource.Spec.Rules = appendUniqueRef(item.resource.Spec.Rules, ref)
				}
			}
		}
	}
	return nil
}

func normalizeCockpitTarget(sourcePath, raw string) (string, bool) {
	target := strings.TrimSpace(raw)
	if target == "" || strings.Contains(target, "://") || strings.HasPrefix(target, "mailto:") {
		return "", false
	}
	if cut, _, ok := strings.Cut(target, "#"); ok {
		target = cut
	}
	if cut, _, ok := strings.Cut(target, "?"); ok {
		target = cut
	}
	if path.Ext(target) != ".md" {
		return "", false
	}
	if strings.HasPrefix(target, "/") {
		target = strings.TrimPrefix(target, "/")
	} else if !strings.HasPrefix(target, "docs/") {
		target = path.Join(path.Dir(sourcePath), target)
	}
	target = path.Clean(target)
	return target, strings.HasPrefix(target, "docs/")
}

func appendUniqueRef(refs []core.Ref, ref core.Ref) []core.Ref {
	for _, existing := range refs {
		if existing == ref {
			return refs
		}
	}
	return append(refs, ref)
}

func cockpitNamespace(file string) (string, error) {
	switch {
	case strings.HasPrefix(file, "docs/general/"):
		return "cockpit-general", nil
	case strings.HasPrefix(file, "docs/consiliari/"):
		return "cockpit-consiliari", nil
	case strings.HasPrefix(file, "docs/customers/"):
		parts := strings.Split(file, "/")
		if len(parts) < 5 || parts[0] != "docs" || parts[1] != "customers" || parts[2] == "" {
			return "", fmt.Errorf("customer mechanism path has no customer owner")
		}
		customer := parts[2]
		if len(parts) >= 7 && parts[3] == "projects" && parts[4] != "" {
			return cockpitProjectAreaName(customer, parts[4]), nil
		}
		return cockpitCustomerAreaName(customer), nil
	default:
		return "", fmt.Errorf("mechanism is outside General, Consiliari, or a customer/project owner area")
	}
}

func cockpitOwners(snapshot *source.Snapshot, definitions []definition) ([]cockpitOwner, error) {
	rules := make(map[string][]core.Ref)
	for _, item := range definitions {
		if item.resource.Kind == "Rule" {
			rules[item.resource.Metadata.Namespace] = append(rules[item.resource.Metadata.Namespace], core.Ref{
				Kind: "Rule", Name: item.resource.Metadata.Name, Namespace: item.resource.Metadata.Namespace,
			})
		}
	}
	for namespace := range rules {
		sort.Slice(rules[namespace], func(i, j int) bool { return rules[namespace][i].Name < rules[namespace][j].Name })
	}
	owners := []cockpitOwner{
		{name: "cockpit", path: "docs"},
		{name: "cockpit-general", path: "docs/general", rules: rules["cockpit-general"]},
		{name: "cockpit-consiliari", path: "docs/consiliari", imports: []string{"cockpit-general"}},
		{name: "cockpit-customers", path: "docs/customers", imports: []string{"cockpit-general", "cockpit-consiliari"}},
	}
	owners[2].rules = appendRefs(rules["cockpit-general"], rules["cockpit-consiliari"])
	owners[3].rules = appendRefs(rules["cockpit-general"], rules["cockpit-consiliari"])

	customers := discoverCockpitCustomers(snapshot)
	for _, customer := range customers {
		customerOwner := cockpitOwner{
			name: cockpitCustomerAreaName(customer), path: "docs/customers/" + customer,
			imports:  []string{"cockpit-general", "cockpit-consiliari", "cockpit-customers"},
			customer: customer, rules: appendRefs(rules["cockpit-general"], rules["cockpit-consiliari"], rules[cockpitCustomerAreaName(customer)]), depth: 1,
		}
		owners = append(owners, customerOwner)
		for _, project := range discoverCockpitProjects(snapshot, customer) {
			projectNS := cockpitProjectAreaName(customer, project)
			projectOwner := cockpitOwner{
				name: projectNS, path: "docs/customers/" + customer + "/projects/" + project,
				imports:  []string{"cockpit-general", "cockpit-consiliari", customerOwner.name},
				customer: customer, project: project, depth: 2,
				rules: appendRefs(rules["cockpit-general"], rules["cockpit-consiliari"], rules[customerOwner.name], rules[projectNS]),
			}
			owners = append(owners, projectOwner)
		}
	}
	for _, item := range definitions {
		found := false
		for _, owner := range owners {
			if owner.name == item.resource.Metadata.Namespace {
				found = true
				break
			}
		}
		if !found {
			return nil, fmt.Errorf("%s: no explicit owner area for namespace %q; add its README router before migration", item.sourcePath, item.resource.Metadata.Namespace)
		}
	}
	return owners, nil
}

func discoverCockpitCustomers(snapshot *source.Snapshot) []string {
	const prefix = "docs/customers/"
	set := map[string]bool{}
	for file := range snapshot.Files {
		if strings.HasPrefix(file, prefix) && strings.HasSuffix(strings.ToLower(file), "/readme.md") {
			parts := strings.Split(file, "/")
			if len(parts) == 4 && parts[2] != "" {
				set[parts[2]] = true
			}
		}
	}
	return cockpitSortedSet(set)
}

func discoverCockpitProjects(snapshot *source.Snapshot, customer string) []string {
	prefix := "docs/customers/" + customer + "/projects/"
	set := map[string]bool{}
	for file := range snapshot.Files {
		if strings.HasPrefix(file, prefix) && strings.HasSuffix(strings.ToLower(file), "/readme.md") {
			rest := strings.TrimPrefix(file, prefix)
			parts := strings.Split(rest, "/")
			if len(parts) == 2 && parts[0] != "" {
				set[parts[0]] = true
			}
		}
	}
	return cockpitSortedSet(set)
}

func cockpitSortedSet(set map[string]bool) []string {
	values := make([]string, 0, len(set))
	for value := range set {
		values = append(values, value)
	}
	sort.Strings(values)
	return values
}

func cockpitProject(definitions []definition, owners []cockpitOwner) (*core.Resource, error) {
	areas := make([]core.Area, 0, len(owners))
	for _, owner := range owners {
		area := core.Area{Name: owner.name, Path: owner.path, Imports: append([]string(nil), owner.imports...), Rules: append([]core.Ref(nil), owner.rules...)}
		areas = append(areas, area)
	}
	// Keep resources' namespaces and project paths unambiguously owned.
	for _, item := range definitions {
		if !strings.HasPrefix(item.resource.Path, "docs/") {
			return nil, fmt.Errorf("%s: generated resource path must remain under docs/", item.resource.Path)
		}
	}
	return &core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: "cockpit"},
		Path:       cockpitProjectPath,
		Spec:       core.Spec{Profile: "cockpit", Areas: areas},
	}, nil
}

func appendRefs(groups ...[]core.Ref) []core.Ref {
	seen := map[string]bool{}
	var refs []core.Ref
	for _, group := range groups {
		for _, ref := range group {
			key := ref.Namespace + "\x00" + ref.Kind + "\x00" + ref.Name
			if !seen[key] {
				seen[key] = true
				refs = append(refs, ref)
			}
		}
	}
	return refs
}

func cockpitCustomerAreaName(customer string) string { return "customer-" + strings.ToLower(customer) }
func cockpitProjectAreaName(customer, project string) string {
	return "project-" + strings.ToLower(customer) + "-" + strings.ToLower(project)
}
