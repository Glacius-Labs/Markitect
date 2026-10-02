package render

import (
	"net/url"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func renderCompanion(r *core.Resource, target string, g *core.Graph) ([]byte, error) {
	var b strings.Builder
	b.WriteString("<!-- " + Marker + "; source: " + r.Path + " -->\n")
	b.WriteString("# " + r.Kind + ": " + r.Metadata.Name + "\n\n")
	if r.Spec.Description != "" {
		b.WriteString("**Description:** " + r.Spec.Description + "\n\n")
	}
	if r.Spec.Text != "" {
		b.WriteString(r.Spec.Text)
		if !strings.HasSuffix(r.Spec.Text, "\n") {
			b.WriteByte('\n')
		}
	}
	links := dependencyLinks(r, target, g, true)
	if len(links) > 0 {
		b.WriteString("\n## Dependencies\n\n")
		for _, link := range links {
			writeDependency(&b, link)
		}
	}
	return []byte(b.String()), nil
}

type dependencyLink struct{ label, path, instruction string }

func dependencyLinks(r *core.Resource, from string, g *core.Graph, markdownViews bool) []dependencyLink {
	type typedRef struct {
		ref  core.Ref
		kind string
	}
	refs := make([]typedRef, 0, len(r.Spec.Rules)+len(r.Spec.Uses)+len(r.Spec.Needs)+len(r.Spec.Implements))
	for _, ref := range r.Spec.Rules {
		refs = append(refs, typedRef{ref, "Rule"})
	}
	for _, ref := range r.Spec.Uses {
		refs = append(refs, typedRef{ref, ""})
	}
	for _, ref := range r.Spec.Needs {
		refs = append(refs, typedRef{ref, "Contract"})
	}
	for _, ref := range r.Spec.Implements {
		refs = append(refs, typedRef{ref, "Contract"})
	}
	seen := map[string]bool{}
	links := make([]dependencyLink, 0, len(refs))
	for _, item := range refs {
		ref := item.ref
		key := ref.GraphKey(r.Package, r.Metadata.Namespace, item.kind)
		if seen[key] {
			continue
		}
		seen[key] = true
		dep := findResource(g, ref, r.Package, r.Metadata.Namespace, item.kind)
		if dep == nil || dep.Kind == "Project" || dep.Path == "" {
			continue
		}
		if dep.Package != "" {
			version := ""
			if g.Project != nil {
				for _, pin := range g.Project.Spec.Packages {
					if pin.Name == dep.Package {
						version = pin.Version
						break
					}
				}
			}
			label := "Package dependency: " + dep.GraphKey()
			if version != "" {
				label += " (version " + version + ")"
			}
			instruction := "Select this exported dependency with `markitect context --repo . --package " + dep.Package + " --namespace " + dep.Metadata.Namespace + " --kind " + dep.Kind + " --name " + dep.Metadata.Name + "`."
			links = append(links, dependencyLink{label: label, instruction: instruction})
			continue
		}
		path := dep.Path
		if markdownViews {
			if view, err := MarkdownViewPath(g, dep); err == nil {
				path = view
			}
		}
		links = append(links, dependencyLink{label: dep.Kind + ": " + dep.Metadata.Name, path: relative(from, path)})
	}
	sort.Slice(links, func(i, j int) bool { return links[i].label < links[j].label })
	return links
}

func renderSkill(r *core.Resource, target string, g *core.Graph) []byte {
	var b strings.Builder
	b.WriteString("---\nname: " + yamlQuote(r.Metadata.Name) + "\ndescription: " + yamlQuote(r.Spec.Description) + "\n---\n\n")
	b.WriteString("<!-- " + Marker + "; source: " + relative(target, r.Path) + " -->\n\n")
	b.WriteString("Read the [canonical Skill YAML](" + relative(target, r.Path) + ") before acting.\n")
	links := dependencyLinks(r, target, g, false)
	if len(links) > 0 {
		b.WriteString("\n## Dependencies\n\n")
		for _, link := range links {
			writeDependency(&b, link)
		}
	}
	return []byte(b.String())
}

func writeDependency(b *strings.Builder, link dependencyLink) {
	if link.instruction != "" {
		b.WriteString("- " + link.label + ". " + link.instruction + "\n")
		return
	}
	b.WriteString("- [" + link.label + "](" + link.path + ")\n")
}

func findResource(g *core.Graph, ref core.Ref, defaultPackage, namespace, kind string) *core.Resource {
	if ref.Kind != "" {
		kind = ref.Kind
	}
	if ref.Namespace != "" {
		namespace = ref.Namespace
	}
	return g.Resources[ref.GraphKey(defaultPackage, namespace, kind)]
}

func companionPath(source string) string {
	ext := path.Ext(source)
	if ext == "" {
		return source + ".md"
	}
	return strings.TrimSuffix(source, ext) + ".md"
}

func relative(fromFile, toFile string) string {
	return relativePath(path.Dir(fromFile), toFile)
}

func relativePath(fromDir, to string) string {
	// Implement POSIX relative paths without consulting the host OS.
	from := strings.Split(strings.Trim(fromDir, "/"), "/")
	toParts := strings.Split(strings.Trim(to, "/"), "/")
	if len(from) == 1 && from[0] == "" {
		from = nil
	}
	i := 0
	for i < len(from) && i < len(toParts) && from[i] == toParts[i] {
		i++
	}
	parts := make([]string, 0, len(from)-i+len(toParts)-i)
	for range from[i:] {
		parts = append(parts, "..")
	}
	parts = append(parts, toParts[i:]...)
	if len(parts) == 0 {
		return "."
	}
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}
