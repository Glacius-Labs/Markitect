package project

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
func (c *fileChecker) modelInputs(source ModelSource, limits model.Limits) {
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
	declared, ok := c.modelRoot(source, limits)
	if !ok {
		return
	}
	root := source.RepositoryArea
	if !declared.IsRoot() {
		root, _ = snapshot.DirectoryAt(repositoryPath(source.RepositoryArea, declared.String()))
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

func (c *fileChecker) sourceInputs(limits model.Limits) {
	c.modelInputs(c.raw.Local, limits)
	for _, dependency := range c.raw.Dependencies {
		c.modelInputs(dependency, limits)
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
	for _, group := range [][]SourceFile{c.raw.Execution, c.raw.Registry, c.raw.Ignore, c.raw.RuntimeProfiles, c.raw.Instructions} {
		for _, source := range group {
			if source.RepositoryPath.String() != "" {
				c.protectFile(source.RepositoryPath, realization.MarkitectInput, "Explicitly captured Markitect input.")
			}
		}
	}
}

// Protection follows the actual authored root, not contradictory acquisition
// metadata. Reuse the language inspector; do not parse a second manifest schema.
func (c *fileChecker) modelRoot(source ModelSource, limits model.Limits) (snapshot.DirectoryBoundary, bool) {
	manifest, found := source.Files.Lookup(source.Selection.ManifestPath)
	if !found {
		c.issue(model.NotCompleted, "files.manifest_uncaptured", repositoryPath(source.RepositoryArea, source.Selection.ManifestPath.String()), "", nil, "The actual model manifest is unavailable for protection.")
		return snapshot.DirectoryBoundary{}, false
	}
	result := model.InspectManifest(model.ManifestInput{Origin: source.Source.Origin, ManifestPath: source.Selection.ManifestPath.String(), Bytes: manifest.Bytes(), Limits: limits})
	if result.Status != model.Passed {
		c.issue(result.Status, "files.manifest_not_passed", repositoryPath(source.RepositoryArea, source.Selection.ManifestPath.String()), "/root", nil, "The actual manifest root could not be inspected.")
		return snapshot.DirectoryBoundary{}, false
	}
	name := result.Root
	prefix := ""
	if slash := strings.LastIndexByte(source.Selection.ManifestPath.String(), '/'); slash >= 0 {
		prefix = source.Selection.ManifestPath.String()[:slash]
	}
	if name == "." {
		name = prefix
	} else if prefix != "" {
		name = prefix + "/" + name
	}
	declared := snapshot.RootDirectory()
	if name != "" {
		path, _ := snapshot.NewPath(name)
		declared, _ = snapshot.DirectoryAt(path)
	}
	if declared != source.Root {
		c.issue(model.Failed, "files.model_root_mismatch", repositoryPath(source.RepositoryArea, source.Selection.ManifestPath.String()), "/root", nil, "Captured model root metadata contradicts the actual manifest declaration.")
	}
	return declared, true
}
