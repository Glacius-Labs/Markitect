package projectrun

import "github.com/Glacius-Labs/Markitect/internal/tooling/source"
import "github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectvalue"

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core/projectmodel/projectcoverage"
)

// Complete coverage is a closure requirement. Planning may admit a missing
// required artifact so Reconcile can create it; a final candidate may not.
func requireFullCoverage(project *projectvalue.Project) error {
	if project.Config.CoverageMode != "full" {
		return nil
	}
	if project.Coverage == nil || !project.Coverage.Conforming {
		return fmt.Errorf("whole-repository coverage is not conforming; classify every path and realize all required artifacts before closure")
	}
	return nil
}

// The Host owns its readable model view. It is added after implementation
// integration, then verified at the same immutable snapshot as every other file.
// It renders the closure compile, which may bind census files the candidate
// now models, so validateCandidateDocument sees the same view.
func completeCandidateDocument(host Host, root string, store *runStore, dir string, base *projectvalue.Project, candidate candidateData) (candidateData, error) {
	project, err := finalProjectForCandidate(host, root, base, candidate)
	if err != nil {
		return candidate, err
	}
	if project.Config.CoverageMode != "full" {
		return candidate, nil
	}
	if host.RenderDocument == nil {
		return candidate, fmt.Errorf("document rendering is not bound")
	}
	content, err := host.RenderDocument(project)
	if err != nil {
		return candidate, err
	}
	file := projectvalue.DocumentPath(project.Config)
	old, exists := project.Snapshot.Files[file]
	if exists && !projectvalue.IsGeneratedDocument(old) {
		return candidate, fmt.Errorf("refusing to replace non-generated documentation at %s", file)
	}
	if exists && bytes.Equal(old, []byte(content)) {
		return candidate, nil
	}
	id, err := newID()
	if err != nil {
		return candidate, err
	}
	files := make(map[string]File, len(candidate.Files)+1)
	for path, value := range candidate.Files {
		value.Content = append([]byte(nil), value.Content...)
		files[path] = value
	}
	files[file] = File{Path: file, Mode: "100644", Content: []byte(content)}
	completed := candidateData{ID: id, Parents: []string{candidate.ID}, Files: files}
	if err := store.writeCandidate(dir, completed); err != nil {
		return candidate, err
	}
	return store.readCandidate(dir, completed.ID)
}

func validateCandidateDocument(host Host, project *projectvalue.Project) error {
	if project.Config.CoverageMode != "full" {
		return nil
	}
	if host.RenderDocument == nil {
		return fmt.Errorf("document rendering is not bound")
	}
	content, err := host.RenderDocument(project)
	if err != nil {
		return err
	}
	file := projectvalue.DocumentPath(project.Config)
	if !bytes.Equal(project.Snapshot.Files[file], []byte(content)) {
		return fmt.Errorf("generated documentation is missing or stale at %s", file)
	}
	return nil
}

func validateIgnoredCandidatePaths(base *source.Input, candidate candidateData, config projectvalue.Config) error {
	ignored, err := ignoredWritePaths(config, base)
	if err != nil {
		return err
	}
	for _, file := range candidateDeltaPaths(base, candidate) {
		if pathIgnored(ignored, file) {
			return fmt.Errorf("candidate may not change explicitly ignored path %s", file)
		}
	}
	return nil
}

func ignoredWritePaths(config projectvalue.Config, input *source.Input) ([]string, error) {
	if config.CoverageMode != "full" {
		return []string{}, nil
	}
	if input == nil {
		return nil, fmt.Errorf("full coverage write policy requires its bound snapshot")
	}
	ignore, err := projectcoverage.DecodeIgnore(input.Files[projectcoverage.IgnorePath])
	if err != nil {
		return nil, err
	}
	paths := make([]string, 0, len(ignore.Entries))
	for _, entry := range ignore.Entries {
		paths = append(paths, entry.Path)
	}
	return paths, nil
}

func pathIgnored(ignored []string, file string) bool {
	for _, entry := range ignored {
		if file == strings.TrimSuffix(entry, "/") || strings.HasSuffix(entry, "/") && strings.HasPrefix(file, entry) {
			return true
		}
	}
	return false
}
