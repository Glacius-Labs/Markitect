package model

// ManifestInput supplies only the explicitly selected manifest's original bytes.
// ManifestPath is an exact relative path ending in model.yaml; Origin is opaque.
// Zero limit fields select DefaultLimits. SourceBytes counts these bytes alone;
// YAMLDepth bounds decoding. All negative Limits fields are invalid configuration.
type ManifestInput struct {
	Origin       string
	ManifestPath string
	Bytes        []byte
	Limits       Limits
}

// ManifestResult reports only manifest decoding and lexical language validity.
// Root is the exact declared path relative to the manifest directory, including
// '.', and is absent unless Status == Passed. A zero result provides no root.
// Source binds original bytes after byte admission; its Digest is empty if
// configuration or byte admission refused inspection. Diagnostics retain their
// original source coordinates and are deterministically ordered.
//
// Passed does not establish root existence, physical spelling or separation,
// redirect safety, a captured tree, dependency availability, a lock, complete
// compilation, acquisition authority or permission to apply anything.
type ManifestResult struct {
	Status      Status
	Source      Source
	Root        string
	Diagnostics []Diagnostic
}

// InspectManifest uses Compile's manifest decoder and lexical validation over
// explicitly supplied bytes. It reads no directory, dependency or lock and
// neither repairs nor changes input bytes. Resource refusal is NotCompleted;
// known manifest violations are Failed, including beside a decoding limit.
func InspectManifest(input ManifestInput) ManifestResult {
	r := ManifestResult{Status: NotCompleted, Source: Source{Origin: input.Origin, Path: input.ManifestPath}}
	if invalidLimits(input.Limits) {
		r.Diagnostics = []Diagnostic{{Code: "config.invalid_limit", Message: "Limits must be nonnegative; zero selects the default."}}
		return r
	}
	limits := effectiveLimits(input.Limits)
	c := sourceCompiler{model: SourceModel{Origin: input.Origin, ManifestPath: input.ManifestPath}, limits: limits}
	entry := Entry{Path: input.ManifestPath}
	admitted := int64(len(input.Bytes)) <= limits.SourceBytes
	if admitted {
		entry.Kind, entry.Bytes = Regular, input.Bytes
		r.Source = source(input.Origin, entry)
	}
	pathValid := c.manifestPathValid(entry)
	if !admitted {
		c.limited = true
		c.diagnostics = append(c.diagnostics, Diagnostic{Code: "limit.source_bytes", Message: "The supplied manifest exceeds the source-byte budget.", Source: r.Source})
	} else if pathValid {
		root, rootOK := c.readManifest(entry)
		if !c.failed && !c.limited && rootOK {
			r.Status, r.Root = Passed, root
		}
	}
	if c.failed {
		r.Status = Failed
	}
	r.Diagnostics = c.diagnostics
	sortDiagnostics(r.Diagnostics)
	return r
}

func (c *sourceCompiler) manifestPathValid(selected Entry) bool {
	if !validPath(c.model.ManifestPath) || basename(c.model.ManifestPath) != "model.yaml" {
		c.problem("input.manifest_path", "ManifestPath must explicitly select a relative model.yaml path.", selected, nil)
		return false
	}
	return true
}

// Keep lexical validation and known metadata in one place. Compile continues
// with its own root/tree/lock checks; inspection publishes only a valid Root.
// Preserve known manifest facts even when later Definition capture is refused.
func (c *sourceCompiler) readManifest(manifest Entry) (string, bool) {
	v, valid := c.decode(manifest)
	if !valid {
		return "", false
	}
	c.closed(v, []string{"formatVersion", "id", "version", "root", "dependencies"}, manifest)
	c.manifestNames(v, manifest)
	c.formatVersion(v, manifest)
	c.scope.modelID, _ = c.requiredText(v, "id", manifest)
	c.scope.manifest = manifest
	if version, present := lookup(v, "version"); present {
		c.nonemptyText(version, "version", manifest)
	}
	root, rootOK := c.requiredText(v, "root", manifest)
	c.dependencies(v, manifest)
	c.scope.version, c.scope.versionDeclared = lookup(v, "version")
	dependencies, _ := lookup(v, "dependencies")
	for _, dependency := range dependencies.fields {
		c.scope.requirements = append(c.scope.requirements, modelRequirement{id: dependency.Name, version: dependency.Value})
	}
	c.scope.manifestReady = !c.failed && !c.limited
	if rootOK && root != "." && !validPath(root) {
		c.problem("manifest.root_path", "root must be '.' or an exact relative directory path without dot components.", manifest, nil)
		return root, false
	}
	return root, rootOK
}
