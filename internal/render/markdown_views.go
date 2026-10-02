package render

import (
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

const markdownViewsRoot = "docs/markitect"

// MarkdownViewPath returns the projection path for a local typed resource.
// Callers use it only when the Project explicitly selects Markdown, so an
// unselected projection path remains an ordinary project file.
func MarkdownViewPath(g *core.Graph, resource *core.Resource) (string, error) {
	if g == nil || resource == nil {
		return "", fmt.Errorf("graph and resource are required")
	}
	if resource.Package != "" || resource.Kind == "Project" || resource.Kind == "Package" {
		return "", fmt.Errorf("resource %s has no local Markdown view", resource.GraphKey())
	}
	area, ok := g.ResourceAreas[resource.GraphKey()]
	if !ok {
		return "", fmt.Errorf("resource %s has no owning Area", resource.GraphKey())
	}
	if area.Name == "" || area.Name == "." || area.Name == ".." || strings.ContainsAny(area.Name, `/\\:`) || strings.ContainsRune(area.Name, 0) {
		return "", fmt.Errorf("Area name %q is not a safe path segment", area.Name)
	}
	root := strings.TrimSuffix(area.Path, "/")
	if root == "" || (resource.Path != root && !strings.HasPrefix(resource.Path, root+"/")) {
		return "", fmt.Errorf("resource %q is outside its owning Area %q", resource.Path, area.Path)
	}
	relative := strings.TrimPrefix(strings.TrimPrefix(resource.Path, root), "/")
	if relative == "" || path.IsAbs(relative) || path.Clean(relative) != relative || strings.HasPrefix(relative, "../") {
		return "", fmt.Errorf("resource path %q is not normalized within Area %q", resource.Path, area.Path)
	}
	ext := path.Ext(relative)
	stem := strings.TrimSuffix(relative, ext)
	suffix := "." + strings.ToLower(resource.Kind)
	if resource.APIVersion != "" && resource.APIVersion != core.APIVersion {
		parts := strings.Split(resource.APIVersion, "/")
		if len(parts) == 2 {
			suffix += "." + strings.ReplaceAll(parts[0], "/", ".") + "." + parts[1]
		}
	}
	if !strings.HasSuffix(strings.ToLower(stem), suffix) {
		stem += suffix
	}
	return path.Join(markdownViewsRoot, area.Name, path.Dir(stem), path.Base(stem)+".md"), nil
}

// MarkdownViewPaths returns all generated resource view paths when the
// Project explicitly selects the Markdown target. Packages are immutable
// inputs and never produce local views.
func MarkdownViewPaths(g *core.Graph) (map[string]*core.Resource, error) {
	paths := map[string]*core.Resource{}
	if g == nil || g.Project == nil || !hasTarget(g.Project.Spec.Targets, "markdown") {
		return paths, nil
	}
	resources := make([]*core.Resource, 0, len(g.Resources))
	for _, resource := range g.Resources {
		if resource != nil {
			resources = append(resources, resource)
		}
	}
	sort.Slice(resources, func(i, j int) bool {
		if resources[i].Path != resources[j].Path {
			return resources[i].Path < resources[j].Path
		}
		return resources[i].GraphKey() < resources[j].GraphKey()
	})
	folded := map[string]string{}
	for _, resource := range resources {
		if resource == nil || resource.Package != "" || resource.Kind == "Project" || resource.Kind == "Package" {
			continue
		}
		output, err := MarkdownViewPath(g, resource)
		if err != nil {
			return nil, err
		}
		if previous, ok := folded[foldPath(output)]; ok {
			return nil, fmt.Errorf("Markdown view path collision: %q and %q", previous, output)
		}
		folded[foldPath(output)] = output
		paths[output] = resource
	}
	return paths, nil
}

func hasTarget(targets []string, target string) bool {
	for _, item := range targets {
		if item == target {
			return true
		}
	}
	return false
}

func addMarkdownNavigation(outputs map[string][]byte, owners map[string]map[string]bool, views map[string]*core.Resource, domains map[string]core.DomainDefinition, projectKey string) error {
	directFiles := map[string][]string{}
	childDirectories := map[string]map[string]bool{}
	directories := map[string]bool{markdownViewsRoot: true}
	for view := range views {
		dir := path.Dir(view)
		directFiles[dir] = append(directFiles[dir], path.Base(view))
		for dir != markdownViewsRoot && strings.HasPrefix(dir, markdownViewsRoot+"/") {
			parent := path.Dir(dir)
			childDirectories[parent] = ensureStringSet(childDirectories[parent])
			childDirectories[parent][path.Base(dir)] = true
			directories[dir] = true
			dir = parent
		}
	}
	if len(domains) > 0 {
		directories[domainMarkdownRoot] = true
		childDirectories[markdownViewsRoot] = ensureStringSet(childDirectories[markdownViewsRoot])
		childDirectories[markdownViewsRoot][path.Base(domainMarkdownRoot)] = true
		for view := range domains {
			directFiles[domainMarkdownRoot] = append(directFiles[domainMarkdownRoot], path.Base(view))
		}
	}
	dirs := make([]string, 0, len(directories))
	for dir := range directories {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)
	for _, dir := range dirs {
		file := path.Join(dir, "README.md")
		if dir == domainMarkdownRoot {
			if err := addOwned(outputs, owners, file, renderDomainIndex(domains, file), projectKey); err != nil {
				return err
			}
			continue
		}
		var b strings.Builder
		b.WriteString("<!-- " + Marker + "; source: markitect.yaml -->\n# Markitect views\n\n")
		files := directFiles[dir]
		sort.Strings(files)
		for _, name := range files {
			view := path.Join(dir, name)
			if domain, ok := domains[view]; ok {
				label := "Domain contract: " + domain.Name + " (" + domain.APIVersion + ")"
				b.WriteString("- [" + markdownText(label) + "](" + relative(file, view) + ")\n")
				continue
			}
			resource := views[view]
			label := resource.Kind + ": " + resource.Metadata.Name
			b.WriteString("- [" + label + "](" + relative(file, view) + ")\n")
		}
		children := make([]string, 0, len(childDirectories[dir]))
		for child := range childDirectories[dir] {
			children = append(children, child)
		}
		sort.Strings(children)
		for _, child := range children {
			childReadme := path.Join(dir, child, "README.md")
			b.WriteString("- [" + child + "](" + relative(file, childReadme) + ")\n")
		}
		if err := addOwned(outputs, owners, file, []byte(b.String()), projectKey); err != nil {
			return err
		}
	}
	return nil
}

func ensureStringSet(values map[string]bool) map[string]bool {
	if values == nil {
		return map[string]bool{}
	}
	return values
}
