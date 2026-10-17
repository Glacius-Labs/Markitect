package apply

import (
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func (c *fileChecker) input(path snapshot.Path, class realization.FileClass, reason string) {
	if _, found := c.raw.Census.Lookup(path); !found {
		return
	}
	classification, err := realization.NewClassification(path, class, reason)
	if err != nil {
		c.issue(model.Failed, "files.classification", path, "", nil, err.Error())
		return
	}
	if old, found := c.inputs[path]; found {
		if old.Class() != class {
			c.result.classifications = append(c.result.classifications, classification)
			c.issue(model.Failed, "files.input_overlap", path, "", nil, "One observed file has incompatible input classifications.")
		}
		return
	}
	c.inputs[path] = classification
}

func (c *fileChecker) protectFile(path snapshot.Path, class realization.FileClass, reason string) {
	if !c.protections["file:"+path.String()] {
		protection, err := realization.ProtectFile(path, reason)
		if err != nil {
			c.issue(model.Failed, "files.protection", path, "", nil, err.Error())
			return
		}
		c.result.protections = append(c.result.protections, protection)
		c.protections["file:"+path.String()] = true
	}
	if class != "" {
		c.input(path, class, reason)
	}
}
func (c *fileChecker) protectDirectory(area snapshot.DirectoryBoundary, class realization.FileClass, reason string) {
	if !c.protections["directory:"+area.String()] {
		protection, err := realization.ProtectDirectory(area, reason)
		if err != nil {
			c.issue(model.Failed, "files.protection", snapshot.Path{}, "", nil, err.Error())
			return
		}
		c.result.protections = append(c.result.protections, protection)
		c.protections["directory:"+area.String()] = true
	}
	if class != "" {
		for _, entry := range c.raw.Census.Entries() {
			if area.Contains(entry.Path()) {
				c.input(entry.Path(), class, reason)
			}
		}
	}
}

// These joins use already validated relative coordinates, never filesystem
// normalization or interpretation of opaque acquisition/provenance locators.
func repositoryPath(area snapshot.DirectoryBoundary, relative string) snapshot.Path {
	if !area.IsRoot() {
		relative = area.String() + "/" + relative
	}
	path, _ := snapshot.NewPath(relative)
	return path
}
func (c *fileChecker) modelInputs(source ModelSource) {
	if source.RepositoryArea.String() == "" {
		return
	}
	manifest := source.Selection.ManifestPath.String()
	lock := "model-lock.yaml"
	if slash := strings.LastIndexByte(manifest, '/'); slash >= 0 {
		lock = manifest[:slash+1] + lock
	}
	c.protectFile(repositoryPath(source.RepositoryArea, manifest), realization.CanonicalInput, "Canonical model manifest.")
	c.protectFile(repositoryPath(source.RepositoryArea, lock), realization.CanonicalInput, "Canonical model lock.")
	root := source.RepositoryArea
	if !source.Root.IsRoot() {
		root, _ = snapshot.DirectoryAt(repositoryPath(source.RepositoryArea, source.Root.String()))
	}
	c.protectDirectory(root, realization.CanonicalInput, "Declared canonical model tree.")
	for _, entry := range c.raw.Census.Entries() {
		if !root.Contains(entry.Path()) {
			continue
		}
		relative := entry.Path().String()
		if !source.RepositoryArea.IsRoot() {
			relative = strings.TrimPrefix(relative, source.RepositoryArea.String()+"/")
		}
		path, _ := snapshot.NewPath(relative)
		if _, found := source.Files.Lookup(path); !found {
			c.issue(model.NotCompleted, "files.model_source_uncaptured", entry.Path(), "", nil, "An observed file in the declared model tree was not captured as model input.")
		}
	}
}

func (c *fileChecker) sourceInputs() {
	c.modelInputs(c.raw.Local)
	for _, dependency := range c.raw.Dependencies {
		c.modelInputs(dependency)
	}
	for _, name := range []string{executionPath, ignorePath} {
		path, _ := snapshot.NewPath(name)
		c.protectFile(path, realization.MarkitectInput, "Reserved project configuration input.")
	}
	for _, name := range []string{".markitect/registry", ".markitect/state", ".git"} {
		path, _ := snapshot.NewPath(name)
		area, _ := snapshot.DirectoryAt(path)
		class := realization.FileClass("")
		if name == ".markitect/registry" {
			class = realization.MarkitectInput
		}
		c.protectDirectory(area, class, "Reserved Markitect registry or control area.")
	}
	for _, group := range [][]SourceFile{c.raw.Execution, c.raw.Registry, c.raw.Ignore, c.raw.HostProfiles, c.raw.Instructions} {
		for _, source := range group {
			if source.RepositoryPath.String() != "" {
				c.protectFile(source.RepositoryPath, realization.MarkitectInput, "Explicitly captured Markitect input.")
			}
		}
	}
}
