package model

import (
	"fmt"
	"sort"
	"strings"
)

// The checks here do not infer a Definition's category from the spelling of
// kind or id. Category-specific ID/Property rules use resolved Core types in
// the contract phase. The lexical union here already rules
// out punctuation, non-NFC names and invalid XID spellings for every category.
func (c *sourceChecker) definitionNames(v Value, e Entry) {
	if id, present := lookup(v, "id"); present && id.kind == StringValue && !validLocalName(id.text) {
		c.problem("name.definition", "id must be an NFC local name in a v1 Definition naming category.", e, positionPointer(id.position))
	}
	if kind, present := lookup(v, "kind"); present && kind.kind == StringValue && !validAuthorReference(kind.text) {
		c.problem("name.reference", "kind must use an NFC local, Package-qualified or alias-qualified reference.", e, positionPointer(kind.position))
	}
	if spec, present := lookup(v, "spec"); present && spec.kind == ObjectValue {
		c.objectFieldNames(spec, e)
	}
	imports, _ := lookup(v, "imports")
	packages, aliases := map[string]string{}, map[string]string{}
	for _, item := range imports.items {
		if pkg, present := lookup(item, "package"); present && pkg.kind == StringValue {
			if !validPackageName(pkg.text) {
				c.problem("name.import_package", "An imported Package must use an NFC lowercase Package path.", e, positionPointer(pkg.position))
			}
			key := nameKey(pkg.text)
			if first, duplicate := packages[key]; duplicate {
				c.problem("name.import_duplicate", fmt.Sprintf("Package import %q collides with %q.", pkg.text, first), e, positionPointer(pkg.position))
			} else {
				packages[key] = pkg.text
			}
		}
		if alias, present := lookup(item, "as"); present && alias.kind == StringValue {
			if !validAlias(alias.text) {
				c.problem("name.alias", "An import alias must be an NFC PascalCase name.", e, positionPointer(alias.position))
			}
			key := nameKey(alias.text)
			if first, duplicate := aliases[key]; duplicate {
				c.problem("name.alias_collision", fmt.Sprintf("Import alias %q collides with %q.", alias.text, first), e, positionPointer(alias.position))
			} else {
				aliases[key] = alias.text
			}
		}
	}
}

func validAuthorReference(s string) bool {
	if !isNFC(s) {
		return false
	}
	if strings.Contains(s, "/") {
		separator := strings.LastIndexByte(s, '/')
		return validPackageName(s[:separator]) && validLocalName(s[separator+1:])
	}
	if strings.Contains(s, ".") {
		alias, id, _ := strings.Cut(s, ".")
		return validAlias(alias) && validLocalName(id)
	}
	return validLocalName(s)
}

func (c *sourceChecker) objectFieldNames(v Value, e Entry) {
	switch v.kind {
	case ObjectValue:
		for _, field := range v.fields {
			if !validFieldName(field.Name) {
				c.problem("name.field", fmt.Sprintf("Field %q must be an NFC camelCase name.", field.Name), e, nil)
			}
			c.objectFieldNames(field.Value, e)
		}
	case ListValue:
		for _, value := range v.items {
			c.objectFieldNames(value, e)
		}
	}
}

func (c *sourceChecker) manifestNames(v Value, e Entry) {
	if id, present := lookup(v, "id"); present && id.kind == StringValue && !validModelLabel(id.text, false) {
		c.problem("name.model_label", "Model ID must use NFC lowercase Unicode label sections.", e, positionPointer(id.position))
	}
	if version, present := lookup(v, "version"); present && version.kind == StringValue && !validModelLabel(version.text, true) {
		c.problem("name.version_label", "Version must be an exact NFC lowercase label other than latest.", e, positionPointer(version.position))
	}
	if root, present := lookup(v, "root"); present && root.kind == StringValue && root.text != "." && !validPackageName(root.text) {
		c.problem("name.root_path", "root must be '.' or an NFC lowercase directory path.", e, positionPointer(root.position))
	}
	dependencies, _ := lookup(v, "dependencies")
	for _, dependency := range dependencies.fields {
		if !validModelLabel(dependency.Name, false) {
			c.problem("name.dependency_label", fmt.Sprintf("Dependency ID %q must use NFC lowercase Unicode label sections.", dependency.Name), e, nil)
		}
		version := dependency.Value
		if version.kind == StringValue && !validModelLabel(version.text, true) {
			c.problem("name.version_label", "Dependency version must be an exact NFC lowercase label other than latest.", e, positionPointer(version.position))
		}
	}
}

func (c *sourceChecker) treeNames(entries map[string]Entry, root string) {
	directories := map[string]Entry{}
	var files []Entry
	lockPath := joinPath(parent(c.model.ManifestPath), "model-lock.yaml")
	for p, entry := range entries {
		if !within(p, root) || p == root {
			continue
		}
		if entry.Kind == Directory {
			directories[p] = entry
		}
		for directory := parent(p); directory != root && directory != ""; directory = parent(directory) {
			directories[directory] = Entry{Path: directory, Kind: Directory}
		}
		if entry.Kind == Regular && p != c.model.ManifestPath && p != lockPath && strings.HasSuffix(strings.ToLower(p), ".yaml") {
			files = append(files, entry)
		}
	}
	paths := make([]string, 0, len(directories))
	for p := range directories {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	seenPackages := map[string]string{}
	for _, p := range paths {
		pkg := relativePackage(p, root)
		c.scope.packages = append(c.scope.packages, Package{Name: pkg, Source: source(c.model.Origin, directories[p])})
		if !validPackageName(pkg) {
			c.problem("name.package", "Package folders must use NFC lowercase XID names.", directories[p], nil)
		}
		key := nameKey(pkg)
		if first, duplicate := seenPackages[key]; duplicate {
			c.problem("name.package_collision", fmt.Sprintf("Package spelling %q collides with %q.", pkg, first), directories[p], nil)
		} else {
			seenPackages[key] = pkg
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	seenFiles := map[Identity]string{}
	for _, file := range files {
		filename := basename(file.Path)
		if !validFilename(filename) {
			c.problem("name.filename", "Definition filenames must use NFC lowercase kebab-case and exact .yaml.", file, nil)
		}
		key := Identity{Package: nameKey(parent(file.Path)), ID: nameKey(filename)}
		if first, duplicate := seenFiles[key]; duplicate {
			c.problem("name.filename_collision", fmt.Sprintf("Definition filename %q collides with %q.", file.Path, first), file, nil)
		} else {
			seenFiles[key] = file.Path
		}
	}
}

func relativePackage(p, root string) string {
	if root == "" {
		return p
	}
	return p[len(root)+1:]
}
