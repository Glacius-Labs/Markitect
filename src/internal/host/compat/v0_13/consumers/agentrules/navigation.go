package agentrules

import (
	"fmt"
	"net/url"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/consumers/agentrules/links"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

func rewriteAgentText(text string, resource core.ModelResource, output string, resources map[string]core.ModelResource, files map[string][]byte) (string, error) {
	sources := map[string]core.ModelResource{}
	companions := map[string][]core.ModelResource{}
	for _, item := range resources {
		if item.Identity.Package != "" || item.Identity.Kind == "Project" || item.Identity.Kind == "Package" || item.Source.Path == "" {
			continue
		}
		sources[item.Source.Path] = item
		companions[companionPath(item.Source.Path)] = append(companions[companionPath(item.Source.Path)], item)
	}
	rewritten, err := links.Rewrite(text, func(destination string) (string, error) {
		if len(destination) >= 3 && isASCIILetter(destination[0]) && destination[1] == ':' && (destination[2] == '/' || destination[2] == '\\') {
			return "", fmt.Errorf("Markdown destination %q is not a portable relative path", destination)
		}
		if strings.HasPrefix(destination, "/") || strings.HasPrefix(destination, "#") || strings.HasPrefix(destination, "?") || hasURLScheme(destination) {
			return destination, nil
		}
		u, err := url.Parse(destination)
		if err != nil {
			return "", fmt.Errorf("invalid Markdown destination %q: %w", destination, err)
		}
		if u.IsAbs() || u.Host != "" || strings.HasPrefix(u.Path, "/") || u.Path == "" {
			return destination, nil
		}
		if strings.ContainsAny(u.Path, "\\:\x00") {
			return "", fmt.Errorf("Markdown destination %q is not a portable relative path", destination)
		}
		sourcePath := path.Clean(path.Join(path.Dir(resource.Source.Path), u.Path))
		if sourcePath == ".." || strings.HasPrefix(sourcePath, "../") {
			return "", fmt.Errorf("Markdown destination %q escapes the repository", destination)
		}
		target := sourcePath
		if aliases := companions[sourcePath]; len(aliases) != 0 {
			data, exists := files[sourcePath]
			if !exists || IsGenerated(data) {
				if len(aliases) != 1 {
					return "", fmt.Errorf("Markdown destination %q is an ambiguous companion alias", destination)
				}
				target = aliases[0].Source.Path
			}
		} else if typed, exists := sources[sourcePath]; exists {
			target = typed.Source.Path
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
		return "", fmt.Errorf("resource %s prose navigation: %w", resource.Identity.Key, err)
	}
	return rewritten, nil
}

func companionPath(source string) string {
	ext := path.Ext(source)
	if ext == "" {
		return source + ".md"
	}
	return strings.TrimSuffix(source, ext) + ".md"
}

func hasURLScheme(value string) bool {
	end := strings.IndexByte(value, ':')
	if end <= 0 {
		return false
	}
	for i := 0; i < end; i++ {
		c := value[i]
		if isASCIILetter(c) {
			continue
		}
		if i > 0 && (c >= '0' && c <= '9' || c == '+' || c == '-' || c == '.') {
			continue
		}
		return false
	}
	return true
}

func isASCIILetter(value byte) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}
