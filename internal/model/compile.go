package model

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
)

// Compile examines only the supplied bytes. The current acquisition slice
// validates source structure, YAML scalar forms, manifests and Definition
// envelopes. Names/Unicode, dependency binding, Core/contracts, resolved values,
// body links and locks are not yet implemented, so no model can pass or receive
// canonical bytes. Known violations remain failed even with incomplete phases.
func Compile(input Input) Result {
	r := Result{Status: NotCompleted, Coordinates: NotCompleted}
	r.Limits = effectiveLimits(input.Limits)
	if input.Limits.SourceBytes < 0 || input.Limits.Entries < 0 || input.Limits.YAMLDepth < 0 || input.Limits.Models < 0 {
		r.Diagnostics = []Diagnostic{{Code: "config.invalid_limit", Message: "Limits must be nonnegative; zero selects the default."}}
		return r
	}
	// Include the local model and check before allocating the combined slice.
	if len(input.Dependencies) >= r.Limits.Models {
		r.Diagnostics = []Diagnostic{{Code: "limit.models", Message: "The supplied input exceeds the model-count budget."}}
		return r
	}
	models := append([]SourceModel{input.Root}, input.Dependencies...)
	// Admission counts every supplied byte/entry. Subtraction avoids overflow
	// even when the caller configures a budget near the integer maximum.
	remainingBytes, remainingEntries := r.Limits.SourceBytes, r.Limits.Entries
	for _, m := range models {
		if len(m.Entries) > remainingEntries {
			r.Diagnostics = append(r.Diagnostics, Diagnostic{Code: "limit.entries", Message: "The supplied input exceeds the entry budget."})
			return r
		}
		remainingEntries -= len(m.Entries)
		for _, e := range m.Entries {
			if int64(len(e.Bytes)) > remainingBytes {
				r.Diagnostics = append(r.Diagnostics, Diagnostic{Code: "limit.source_bytes", Message: "The supplied input exceeds the source-byte budget."})
				return r
			}
			remainingBytes -= int64(len(e.Bytes))
		}
	}
	var scopes []sourceScope
	for index, m := range models {
		modelResult, diagnostics, scope := compileSource(m, r.Limits)
		scopes = append(scopes, scope)
		modelResult.Local = index == 0
		r.Models = append(r.Models, modelResult)
		r.Diagnostics = append(r.Diagnostics, diagnostics...)
		if modelResult.Status == Failed {
			r.Status = Failed
		}
	}
	for _, finding := range checkExactScopeIdentities(scopes) {
		r.Diagnostics = append(r.Diagnostics, finding.diagnostic)
		r.Models[finding.scope].Status = Failed
		r.Models[finding.scope].Phases[2].Status = Failed
		r.Status = Failed
	}
	sort.Slice(r.Models, func(i, j int) bool {
		a, b := r.Models[i], r.Models[j]
		if a.Local != b.Local {
			return a.Local
		}
		if a.Origin != b.Origin {
			return a.Origin < b.Origin
		}
		if a.ID != b.ID {
			return a.ID < b.ID
		}
		if a.Manifest.Path != b.Manifest.Path {
			return a.Manifest.Path < b.Manifest.Path
		}
		if a.Manifest.Digest != b.Manifest.Digest {
			return a.Manifest.Digest < b.Manifest.Digest
		}
		return a.Status < b.Status
	})
	sortDiagnostics(r.Diagnostics)
	return r
}

func effectiveLimits(l Limits) Limits {
	d := DefaultLimits()
	if l.SourceBytes != 0 {
		d.SourceBytes = l.SourceBytes
	}
	if l.Entries != 0 {
		d.Entries = l.Entries
	}
	if l.YAMLDepth != 0 {
		d.YAMLDepth = l.YAMLDepth
	}
	if l.Models != 0 {
		d.Models = l.Models
	}
	return d
}

type sourceCompiler struct {
	model        SourceModel
	limits       Limits
	diagnostics  []Diagnostic
	failed       bool
	limited      bool
	sourceFailed bool
	yamlFailed   bool
	namesFailed  bool
	rootScanned  bool
	scope        sourceScope
}

func compileSource(m SourceModel, limits Limits) (ModelResult, []Diagnostic, sourceScope) {
	c := sourceCompiler{model: m, limits: limits, scope: sourceScope{origin: m.Origin}}
	r := ModelResult{Origin: m.Origin, Status: NotCompleted}
	r.Phases = []Phase{
		{Name: "source_structure", Status: NotCompleted},
		{Name: "yaml_and_envelopes", Status: NotCompleted},
		{Name: "names_and_unicode", Status: NotCompleted},
		{Name: "dependencies", Status: NotCompleted},
		{Name: "core_and_contracts", Status: NotCompleted},
		{Name: "values_and_references", Status: NotCompleted},
		{Name: "body_links", Status: NotCompleted},
		{Name: "lock", Status: NotCompleted},
		{Name: "canonical", Status: NotCompleted},
	}
	entries := c.indexEntries()
	manifest, ok := entries[m.ManifestPath]
	r.Manifest = source(m.Origin, manifest)
	r.Manifest.Path = m.ManifestPath
	if !validPath(m.ManifestPath) || basename(m.ManifestPath) != "model.yaml" {
		c.problem("input.manifest_path", "ManifestPath must explicitly select a relative model.yaml path.", Entry{Path: m.ManifestPath}, nil)
	} else if !ok || manifest.Kind != Regular {
		c.problem("manifest.missing", "The selected manifest must be a supplied regular file.", Entry{Path: m.ManifestPath}, nil)
	} else {
		v, valid := c.decode(manifest)
		if valid {
			c.closed(v, []string{"formatVersion", "id", "version", "root", "dependencies"}, manifest)
			c.formatVersion(v, manifest)
			r.ID, _ = c.requiredText(v, "id", manifest)
			c.scope.modelID, c.scope.manifest = r.ID, manifest
			if version, present := lookup(v, "version"); present {
				c.nonemptyText(version, "version", manifest)
			}
			root, rootOK := c.requiredText(v, "root", manifest)
			c.dependencies(v, manifest)
			if rootOK {
				if root != "." && !validPath(root) {
					c.problem("manifest.root_path", "root must be '.' or an exact relative directory path without dot components.", manifest, nil)
				} else {
					rootPath := joinPath(parent(m.ManifestPath), root)
					if !directoryExists(entries, rootPath) {
						c.problem("manifest.root_missing", "The declared root must exist as a directory in the supplied tree.", manifest, nil)
					} else {
						c.definitions(entries, rootPath)
						c.rootScanned = true
					}
				}
			}
		}
	}
	if c.failed {
		r.Status = Failed
	}
	if c.namesFailed {
		r.Phases[2].Status = Failed
	}
	if c.sourceFailed {
		r.Phases[0].Status = Failed
	} else if c.rootScanned {
		r.Phases[0].Status = Passed
	}
	if c.yamlFailed {
		r.Phases[1].Status = Failed
	} else if c.rootScanned && !c.limited && !c.sourceFailed {
		r.Phases[1].Status = Passed
	}
	return r, c.diagnostics, c.scope
}

func (c *sourceCompiler) indexEntries() map[string]Entry {
	entries := make(map[string]Entry, len(c.model.Entries))
	ordered := append([]Entry(nil), c.model.Entries...)
	sort.Slice(ordered, func(i, j int) bool {
		a, b := ordered[i], ordered[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Kind != b.Kind {
			return a.Kind < b.Kind
		}
		return string(a.Bytes) < string(b.Bytes)
	})
	for _, e := range ordered {
		if !validPath(e.Path) {
			c.problem("input.path", "Entry paths must be exact relative '/' paths without empty or dot components.", e, nil)
			continue
		}
		if _, exists := entries[e.Path]; exists {
			c.problem("input.duplicate_path", "A source path was supplied more than once.", e, nil)
			continue
		}
		entries[e.Path] = e
		if e.Kind != Regular && e.Kind != Directory && e.Kind != Redirect {
			c.problem("input.entry_kind", "Unknown acquired entry kind.", e, nil)
		}
		if e.Kind != Regular && len(e.Bytes) != 0 {
			c.problem("input.entry_bytes", "Only regular files may carry bytes.", e, nil)
		}
	}
	for _, e := range ordered {
		for p := parent(e.Path); p != ""; p = parent(p) {
			if ancestor, exists := entries[p]; exists && ancestor.Kind != Directory {
				c.problem("input.nondirectory_parent", "A supplied entry has a non-directory ancestor.", e, nil)
				break
			}
		}
	}
	return entries
}

func (c *sourceCompiler) definitions(entries map[string]Entry, root string) {
	paths := make([]string, 0, len(entries))
	for p := range entries {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	lockPath := joinPath(parent(c.model.ManifestPath), "model-lock.yaml")
	// An explicitly supplied lock is YAML-checked but not yet semantically bound.
	if lock, present := entries[lockPath]; present {
		if lock.Kind != Regular {
			c.problem("lock.not_regular", "The selected lock must be a regular file.", lock, nil)
		} else {
			c.decode(lock)
		}
	}
	for _, p := range paths {
		e := entries[p]
		if !within(p, root) {
			continue
		}
		if e.Kind == Redirect {
			c.problem("source.redirect", "Redirects in a model root are forbidden.", e, nil)
			continue
		}
		if e.Kind != Regular || p == c.model.ManifestPath || p == lockPath {
			continue
		}
		lower := strings.ToLower(p)
		if strings.HasSuffix(lower, ".yml") || (strings.HasSuffix(lower, ".yaml") && !strings.HasSuffix(p, ".yaml")) {
			c.problem("definition.extension", "Definition files must use the exact .yaml extension.", e, nil)
			continue
		}
		if !strings.HasSuffix(p, ".yaml") {
			continue
		}
		if parent(p) == root {
			c.problem("definition.root_package", "The model root is not a Package; Definitions need a named subdirectory.", e, nil)
		}
		v, valid := c.decode(e)
		if valid {
			c.envelope(v, e)
			c.collectDefinition(v, e, root)
		}
	}
	c.checkExactIdentities()
}

func (c *sourceCompiler) decode(e Entry) (Value, bool) {
	v, problems, completed := parseYAML(e.Bytes, c.limits.YAMLDepth)
	valid := completed && len(problems) == 0
	for _, p := range problems {
		if strings.HasPrefix(p.code, "limit.") {
			c.limited = true
			c.diagnostics = append(c.diagnostics, Diagnostic{Code: p.code, Message: p.message, Source: source(c.model.Origin, e), Position: positionPointer(p.position)})
		} else {
			c.problem(p.code, p.message, e, positionPointer(p.position))
		}
	}
	if !completed {
		c.limited = true
	}
	return v, valid
}

func (c *sourceCompiler) envelope(v Value, e Entry) {
	c.closed(v, []string{"imports", "id", "kind", "body", "spec", "title"}, e)
	c.requiredText(v, "id", e)
	c.requiredText(v, "kind", e)
	if body, ok := c.requiredText(v, "body", e); ok && whitespaceOnly(body) {
		c.problem("definition.empty_body", "body must contain a character outside Unicode 15.1 White_Space.", e, nil)
	}
	if title, present := lookup(v, "title"); present && title.kind != StringValue {
		c.problem("definition.title", "title must be a string when present.", e, positionPointer(title.position))
	}
	if spec, present := lookup(v, "spec"); !present || spec.kind != ObjectValue {
		c.problem("definition.spec", "spec is required and must be a mapping.", e, nil)
	}
	if imports, present := lookup(v, "imports"); present {
		if imports.kind != ListValue {
			c.problem("definition.imports", "imports must be a list.", e, positionPointer(imports.position))
			return
		}
		for _, item := range imports.items {
			if item.kind != ObjectValue {
				c.problem("import.object", "Each import must be a mapping.", e, positionPointer(item.position))
				continue
			}
			c.closed(item, []string{"package", "as"}, e)
			c.requiredText(item, "package", e)
			if alias, present := lookup(item, "as"); present {
				c.nonemptyText(alias, "as", e)
			}
		}
	}
}

func (c *sourceCompiler) formatVersion(v Value, e Entry) {
	f, present := lookup(v, "formatVersion")
	if !present || f.kind != IntegerValue || f.text != "1" {
		c.problem("manifest.format", "formatVersion must be the integer 1.", e, nil)
	}
}

func (c *sourceCompiler) dependencies(v Value, e Entry) {
	d, present := lookup(v, "dependencies")
	if !present || d.kind != ObjectValue {
		c.problem("manifest.dependencies", "dependencies is required and must be a mapping.", e, nil)
		return
	}
	for _, field := range d.fields {
		c.nonemptyText(field.Value, "dependency version", e)
	}
}

func (c *sourceCompiler) closed(v Value, allowed []string, e Entry) {
	for _, field := range v.fields {
		known := false
		for _, key := range allowed {
			known = known || field.Name == key
		}
		if !known {
			c.problem("field.unknown", fmt.Sprintf("Unknown field %q.", field.Name), e, positionPointer(field.Value.position))
		}
	}
}

func (c *sourceCompiler) requiredText(v Value, key string, e Entry) (string, bool) {
	f, present := lookup(v, key)
	if !present {
		c.problem("field.required", fmt.Sprintf("Required string field %q is missing.", key), e, nil)
		return "", false
	}
	return c.nonemptyText(f, key, e)
}

func (c *sourceCompiler) nonemptyText(v Value, field string, e Entry) (string, bool) {
	if v.kind != StringValue || v.text == "" {
		c.problem("field.nonempty_string", fmt.Sprintf("%s must be a nonempty string.", field), e, positionPointer(v.position))
		return "", false
	}
	return v.text, true
}

func (c *sourceCompiler) problem(code, message string, e Entry, pos *Position) {
	c.failed = true
	if strings.HasPrefix(code, "name.") {
		c.namesFailed = true
	} else if strings.HasPrefix(code, "input.") || strings.HasPrefix(code, "source.") || strings.HasPrefix(code, "manifest.root") || code == "manifest.missing" || code == "definition.extension" || code == "definition.root_package" || code == "lock.not_regular" {
		c.sourceFailed = true
	} else {
		c.yamlFailed = true
	}
	c.diagnostics = append(c.diagnostics, Diagnostic{Code: code, Message: message, Source: source(c.model.Origin, e), Position: pos})
}

func source(origin string, e Entry) Source {
	digest := ""
	if e.Kind == Regular {
		digest = fmt.Sprintf("sha256:%x", sha256.Sum256(e.Bytes))
	}
	return Source{Origin: origin, Path: e.Path, Digest: digest}
}

func positionPointer(p Position) *Position {
	if p.Line <= 0 || p.Column <= 0 {
		return nil
	}
	return &p
}

func lookup(v Value, name string) (Value, bool) {
	for _, f := range v.fields {
		if f.Name == name {
			return f.Value, true
		}
	}
	return Value{}, false
}

// These checks are path structure only; Unicode/name grammar is a separate,
// explicitly incomplete content phase. Never clean a path into valid spelling.
func validPath(p string) bool {
	if p == "" || strings.ContainsAny(p, "\\:\x00") {
		return false
	}
	for _, part := range strings.Split(p, "/") {
		if part == "" || part == "." || part == ".." {
			return false
		}
	}
	return true
}

func parent(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[:i]
	}
	return ""
}

func basename(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

func joinPath(p, child string) string {
	if child == "." {
		return p
	}
	if p == "" {
		return child
	}
	return p + "/" + child
}

func within(p, root string) bool {
	return root == "" || p == root || strings.HasPrefix(p, root+"/")
}

func directoryExists(entries map[string]Entry, p string) bool {
	if p == "" {
		return true // The supplied acquisition boundary exists by construction.
	}
	if entry, present := entries[p]; present {
		return entry.Kind == Directory
	}
	for path := range entries {
		if strings.HasPrefix(path, p+"/") {
			return true
		}
	}
	return false
}

func whitespaceOnly(s string) bool {
	for _, r := range s {
		// Unicode 15.1.0 PropList.txt White_Space, independent of Go's tables.
		if !(r >= 0x09 && r <= 0x0d || r == 0x20 || r == 0x85 || r == 0xa0 || r == 0x1680 || r >= 0x2000 && r <= 0x200a || r == 0x2028 || r == 0x2029 || r == 0x202f || r == 0x205f || r == 0x3000) {
			return false
		}
	}
	return true
}

func sortDiagnostics(d []Diagnostic) {
	sort.Slice(d, func(i, j int) bool {
		a, b := d[i], d[j]
		if a.Source.Origin != b.Source.Origin {
			return a.Source.Origin < b.Source.Origin
		}
		if a.Source.Path != b.Source.Path {
			return a.Source.Path < b.Source.Path
		}
		ap, bp := Position{}, Position{}
		if a.Position != nil {
			ap = *a.Position
		}
		if b.Position != nil {
			bp = *b.Position
		}
		if ap.Line != bp.Line {
			return ap.Line < bp.Line
		}
		if ap.Column != bp.Column {
			return ap.Column < bp.Column
		}
		if a.Code != b.Code {
			return a.Code < b.Code
		}
		if a.Message != b.Message {
			return a.Message < b.Message
		}
		return a.Source.Digest < b.Source.Digest
	})
}
