package format

import (
	"fmt"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"go.yaml.in/yaml/v3"
)

func validateChecks(file string, n *yaml.Node) error {
	if err := requireSequence(file, n, "checks"); err != nil {
		return err
	}
	seen := map[string]bool{}
	for _, item := range n.Content {
		if err := requireMapping(file, item, "project check"); err != nil {
			return err
		}
		if err := checkKeys(file, item, set("name", "run")); err != nil {
			return err
		}
		if err := requireFields(file, item, "name", "run"); err != nil {
			return err
		}
		name := child(item, "name")
		if err := checkScalar(file, name, "string"); err != nil {
			return err
		}
		run := child(item, "run")
		if err := requireSequence(file, run, "check run"); err != nil {
			return err
		}
		check := core.Check{Name: name.Value, Run: make([]string, 0, len(run.Content))}
		for _, arg := range run.Content {
			if err := checkScalar(file, arg, "string"); err != nil {
				return err
			}
			check.Run = append(check.Run, arg.Value)
		}
		if err := core.ValidateCheck(check); err != nil {
			return diagnostic(file, item.Line, "invalid project check: %v", err)
		}
		if seen[check.Name] {
			return diagnostic(file, name.Line, "project check name %q is duplicated", check.Name)
		}
		seen[check.Name] = true
	}
	return nil
}

func validatePackagePins(file string, n *yaml.Node) error {
	if err := requireSequence(file, n, "packages"); err != nil {
		return err
	}
	seen := map[string]bool{}
	archives := map[string]bool{}
	for _, item := range n.Content {
		if err := requireMapping(file, item, "package pin"); err != nil {
			return err
		}
		if err := checkKeys(file, item, set("name", "version", "source", "archive", "sha256")); err != nil {
			return err
		}
		if err := requireFields(file, item, "name", "version", "source", "archive", "sha256"); err != nil {
			return err
		}
		values := make(map[string]*yaml.Node, 5)
		for _, field := range []string{"name", "version", "source", "archive", "sha256"} {
			value := child(item, field)
			if err := checkScalar(file, value, "string"); err != nil {
				return err
			}
			values[field] = value
		}
		name := values["name"]
		if !validName(name.Value) {
			return diagnostic(file, name.Line, "package name must be a DNS label of at most 63 characters")
		}
		if seen[name.Value] {
			return diagnostic(file, name.Line, "duplicate package pin %q", name.Value)
		}
		seen[name.Value] = true
		if strings.TrimSpace(values["version"].Value) == "" {
			return diagnostic(file, values["version"].Line, "package version must not be empty")
		}
		if strings.TrimSpace(values["source"].Value) == "" || strings.ContainsAny(values["source"].Value, "\x00\r\n") {
			return diagnostic(file, values["source"].Line, "package source must be a nonempty provenance coordinate without control characters")
		}
		archive := values["archive"]
		if err := validatePackageArchivePath(archive.Value); err != nil {
			return diagnostic(file, archive.Line, "unsafe package archive path: %v", err)
		}
		foldedArchive := strings.ToLower(archive.Value)
		if archives[foldedArchive] {
			return diagnostic(file, archive.Line, "package archive path %q is duplicated or case-colliding", archive.Value)
		}
		archives[foldedArchive] = true
		sha := values["sha256"]
		if !sha256Pattern.MatchString(sha.Value) {
			return diagnostic(file, sha.Line, "package sha256 must be 64 lowercase hexadecimal characters")
		}
	}
	return nil
}

func validatePackageArchivePath(value string) error {
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

func validateAreas(file string, n *yaml.Node) error {
	if err := requireSequence(file, n, "areas"); err != nil {
		return err
	}
	names, paths := map[string]bool{}, map[string]bool{}
	for _, a := range n.Content {
		if err := requireMapping(file, a, "area"); err != nil {
			return err
		}
		if err := checkKeys(file, a, set("name", "path", "imports", "rules")); err != nil {
			return err
		}
		if err := requireFields(file, a, "name", "path"); err != nil {
			return err
		}
		name, p := child(a, "name"), child(a, "path")
		for _, v := range []*yaml.Node{name, p} {
			if err := checkScalar(file, v, "string"); err != nil {
				return err
			}
		}
		if !validName(name.Value) {
			return diagnostic(file, name.Line, "area name must be a DNS label of at most 63 characters")
		}
		if names[name.Value] {
			return diagnostic(file, name.Line, "duplicate area name %q", name.Value)
		}
		names[name.Value] = true
		if p.Value == "" || path.IsAbs(p.Value) || path.Clean(p.Value) != p.Value || p.Value == "." || strings.HasPrefix(p.Value, "../") {
			return diagnostic(file, p.Line, "area path must be a normalized relative POSIX path without '..'")
		}
		for _, part := range strings.Split(p.Value, "/") {
			if part == ".." || part == "." || part == "" {
				return diagnostic(file, p.Line, "area path must be a normalized relative POSIX path without '..'")
			}
		}
		if paths[p.Value] {
			return diagnostic(file, p.Line, "duplicate area path %q", p.Value)
		}
		paths[p.Value] = true
		if x := child(a, "imports"); x != nil {
			if err := validateStrings(file, x, "area imports", true); err != nil {
				return err
			}
			for _, s := range x.Content {
				if !validName(s.Value) {
					return diagnostic(file, s.Line, "area import must be a DNS label")
				}
			}
		}
		if x := child(a, "rules"); x != nil {
			if err := validateRefs(file, x, "rules"); err != nil {
				return err
			}
		}
	}
	return nil
}

func validateBindings(file string, n *yaml.Node) error {
	if err := requireSequence(file, n, "bindings"); err != nil {
		return err
	}
	for _, b := range n.Content {
		if err := requireMapping(file, b, "binding"); err != nil {
			return err
		}
		if err := checkKeys(file, b, set("contract", "implementation")); err != nil {
			return err
		}
		if err := requireFields(file, b, "contract", "implementation"); err != nil {
			return err
		}
		for _, field := range []string{"contract", "implementation"} {
			r := child(b, field)
			if err := requireMapping(file, r, field); err != nil {
				return err
			}
			if err := checkKeys(file, r, set("kind", "name", "namespace", "package")); err != nil {
				return err
			}
			if err := requireFields(file, r, "name"); err != nil {
				return err
			}
			for _, key := range []string{"kind", "name", "namespace", "package"} {
				if x := child(r, key); x != nil {
					if err := checkScalar(file, x, "string"); err != nil {
						return err
					}
					if (key == "name" || key == "namespace" || key == "package") && !validName(x.Value) {
						return diagnostic(file, x.Line, "binding reference %s must be a DNS label", key)
					}
				}
			}
		}
		if child(child(b, "contract"), "kind") != nil && child(child(b, "contract"), "kind").Value != "Contract" {
			return diagnostic(file, child(child(b, "contract"), "kind").Line, "binding contract kind must be Contract")
		}
		if child(child(b, "implementation"), "kind") == nil {
			return diagnostic(file, child(b, "implementation").Line, "binding implementation requires kind")
		}
	}
	return nil
}

func validateRuleAdapters(file string, n *yaml.Node) error {
	if err := requireMapping(file, n, "ruleAdapters"); err != nil {
		return err
	}
	seen := map[string]bool{}
	for i := 0; i < len(n.Content); i += 2 {
		k, v := n.Content[i], n.Content[i+1]
		if err := checkScalar(file, k, "string"); err != nil {
			return err
		}
		if strings.TrimSpace(k.Value) == "" {
			return diagnostic(file, k.Line, "rule adapter names must not be empty")
		}
		if seen[k.Value] {
			return diagnostic(file, k.Line, "duplicate rule adapter %q", k.Value)
		}
		seen[k.Value] = true
		if err := validateRefs(file, v, "rules"); err != nil {
			return err
		}
	}
	return nil
}
