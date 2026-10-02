package render

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/markdownlinks"
)

// proseNavigation projects source-relative navigation without adding graph edges.
// The caller supplies fixed snapshot files so an ordinary file can take precedence
// over a former generated companion at exactly the same path.
type proseNavigation struct {
	files      map[string][]byte
	companions map[string][]*core.Resource
	sources    map[string]*core.Resource
	graph      *core.Graph
}

func newProseNavigation(g *core.Graph, resources []*core.Resource, files map[string][]byte) *proseNavigation {
	n := &proseNavigation{files: files, companions: map[string][]*core.Resource{}, sources: map[string]*core.Resource{}, graph: g}
	for _, r := range resources {
		if r.Kind != "Project" && r.Kind != "Package" && r.Package == "" {
			n.sources[r.Path] = r
			p := companionPath(r.Path)
			n.companions[p] = append(n.companions[p], r)
		}
	}
	return n
}

func (n *proseNavigation) rewrite(text string, r *core.Resource, output string, markdownView bool) (string, error) {
	rewritten, err := markdownlinks.Rewrite(text, func(destination string) (string, error) {
		// Drive-letter paths must not bypass portability checks as URL schemes.
		if len(destination) >= 3 && ((destination[0] >= 'a' && destination[0] <= 'z') || (destination[0] >= 'A' && destination[0] <= 'Z')) && destination[1] == ':' && (destination[2] == '/' || destination[2] == '\\') {
			return "", fmt.Errorf("Markdown destination %q is not a portable relative path", destination)
		}
		if strings.HasPrefix(destination, "/") || strings.HasPrefix(destination, "#") || strings.HasPrefix(destination, "?") || hasURLScheme(destination) {
			return destination, nil
		}
		u, err := url.Parse(destination)
		if err != nil {
			return "", fmt.Errorf("invalid Markdown destination %q: %w", destination, err)
		}
		// Absolute URLs, site-root paths and local anchors retain their meaning.
		if u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") || u.Path == "" {
			return destination, nil
		}
		if strings.ContainsAny(u.Path, "\\:\x00") {
			return "", fmt.Errorf("Markdown destination %q is not a portable relative path", destination)
		}
		sourcePath := path.Clean(path.Join(path.Dir(r.Path), u.Path))
		if sourcePath == ".." || strings.HasPrefix(sourcePath, "../") {
			return "", fmt.Errorf("Markdown destination %q escapes the repository", destination)
		}
		target := sourcePath
		if typed := n.sources[sourcePath]; typed != nil && markdownView {
			target, err = MarkdownViewPath(n.graph, typed)
			if err != nil {
				return "", err
			}
		}
		if aliases := n.companions[sourcePath]; len(aliases) != 0 {
			data, exists := n.files[sourcePath]
			if !exists || IsGenerated(data) {
				if len(aliases) != 1 {
					return "", fmt.Errorf("Markdown destination %q is an ambiguous companion alias", destination)
				}
				target = aliases[0].Path
				if markdownView {
					target, err = MarkdownViewPath(n.graph, aliases[0])
					if err != nil {
						return "", err
					}
				}
			}
		}
		encoded := relative(output, target)
		if strings.HasSuffix(u.Path, "/") && !strings.HasSuffix(encoded, "/") {
			encoded += "/"
		}
		u.Path, err = url.PathUnescape(encoded)
		if err != nil {
			return "", err
		}
		u.RawPath = encoded
		return u.String(), nil
	})
	if err != nil {
		return "", fmt.Errorf("resource %s prose navigation: %w", r.GraphKey(), err)
	}
	return rewritten, nil
}

func hasURLScheme(destination string) bool {
	end := strings.IndexByte(destination, ':')
	if end <= 0 {
		return false
	}
	for i := 0; i < end; i++ {
		c := destination[i]
		if c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' {
			continue
		}
		if i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.') {
			continue
		}
		return false
	}
	return true
}
