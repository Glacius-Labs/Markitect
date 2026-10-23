package project

import (
	"bytes"
	"fmt"
	"io"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

const ignorePath = ".markitect/ignore.yaml"

func (c *fileChecker) ignore() {
	path, _ := snapshot.NewPath(ignorePath)
	_, present := c.raw.Census.Lookup(path)
	if len(c.raw.Ignore) == 0 {
		if present {
			c.issue(model.NotCompleted, "files.ignore_uncaptured", path, "", nil, "The observed ignore policy has no captured source; a new complete capture is required.")
		}
		return
	}
	if len(c.raw.Ignore) != 1 {
		c.issue(model.Failed, "files.ignore_source_count", path, "", nil, "At most one exact-file ignore policy may be captured.")
	}
	for _, source := range c.raw.Ignore {
		if source.RepositoryPath != path {
			c.issue(model.Failed, "files.ignore_source_path", source.RepositoryPath, "", nil, "Ignore policy must be bound to .markitect/ignore.yaml.")
			continue
		}
		if !sameCensusFile(c.raw.Census, ignorePath, source.File) {
			c.issue(model.NotCompleted, "files.ignore_source_mismatch", path, "", nil, "Ignore policy bytes/mode are not bound to Census.")
			continue
		}
		c.decodeIgnore(source)
	}
}

func (c *fileChecker) decodeIgnore(source SourceFile) {
	path := source.RepositoryPath
	invalid := func(code string, field configField, message string) {
		var position *model.Position
		if field.node != nil && field.node.Line > 0 {
			position = &model.Position{Line: field.node.Line, Column: field.node.Column}
		}
		c.issue(model.Failed, "files.ignore."+code, path, field.pointer, position, message)
	}
	d := configYAML{name: "ignore", title: "Ignore", issue: invalid}
	decoder := yaml.NewDecoder(bytes.NewReader(source.File.Bytes()))
	var document, extra yaml.Node
	if err := decoder.Decode(&document); err != nil {
		invalid("yaml", configField{}, err.Error())
		return
	}
	if err := decoder.Decode(&extra); err != io.EOF {
		invalid("documents", configField{node: &extra}, "Ignore policy requires exactly one YAML document.")
	}
	if len(document.Content) != 1 {
		invalid("document", configField{}, "Ignore policy requires a mapping document.")
		return
	}
	root := configField{node: document.Content[0]}
	d.restricted(root)
	f := d.object(root, "apiVersion", "files")
	version := d.required(f, root, "apiVersion")
	if value, ok := d.text(version, false); ok && value != "markitect.ignore/v1" {
		invalid("version", version, "Unsupported exact-file ignore policy version.")
	}
	files := d.required(f, root, "files")
	if files.node == nil {
		return
	}
	if files.node.Kind != yaml.SequenceNode {
		invalid("list", files, "Ignore files must be an explicit list, including [] when empty.")
		return
	}
	seen := make(map[snapshot.Path]bool)
	for i, node := range files.node.Content {
		row := configChild(files, fmt.Sprint(i), node)
		fields := d.object(row, "path", "reason")
		pathField := d.required(fields, row, "path")
		ref, pathOK := d.text(pathField, false)
		reason, reasonOK := d.text(d.required(fields, row, "reason"), false)
		if !pathOK || !reasonOK {
			continue
		}
		filePath, err := snapshot.NewPath(ref)
		if err != nil {
			invalid("path", pathField, err.Error())
			continue
		}
		if seen[filePath] {
			invalid("duplicate_path", pathField, "An exact file is listed more than once.")
		}
		seen[filePath] = true
		if _, found := c.raw.Census.Lookup(filePath); !found {
			invalid("missing_file", pathField, "Ignore path is not an exact observed regular file.")
		}
		classification, err := realization.NewClassification(filePath, realization.IgnoredFile, reason)
		if err != nil {
			invalid("classification", pathField, err.Error())
			continue
		}
		c.result.classifications = append(c.result.classifications, classification)
	}
}
