package runtimeconfig

import (
	"bytes"
	"fmt"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"io"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"go.yaml.in/yaml/v3"
)

const maxRuntimeDocumentBytes = 4 << 20
const maxRuntimeNodes = 100000
const maxRuntimeDepth = 64
const maxRuntimeProfiles = 1024

type runtimeEnvelope struct {
	status      model.Status
	profiles    []ProfileInput
	diagnostics []RuntimeProfileDiagnostic
}

func (h *runtimeEnvelope) issue(file project.SourceFile, status model.Status, code string, f configField, message string) {
	var position *model.Position
	if f.node != nil && f.node.Line > 0 {
		position = &model.Position{Line: f.node.Line, Column: f.node.Column}
	}
	h.diagnostics = append(h.diagnostics, RuntimeProfileDiagnostic{Code: "runtime." + code, Source: runtimeSource(file, f.pointer), Position: position, Message: message})
	if status == model.Failed || h.status != model.Failed {
		h.status = status
	}
}

func decodeRuntime(raw project.RawInputs) runtimeEnvelope {
	h := runtimeEnvelope{status: model.Passed}
	if len(raw.RuntimeProfiles) != 1 {
		status := model.Failed
		if len(raw.RuntimeProfiles) == 0 {
			status = model.NotCompleted
		}
		h.issue(project.SourceFile{}, status, "document_count", configField{}, "Exactly one explicitly captured Runtime profile document is required.")
		return h
	}
	file := raw.RuntimeProfiles[0]
	data := file.File.Bytes()
	if len(data) > maxRuntimeDocumentBytes {
		h.issue(file, model.NotCompleted, "byte_limit", configField{}, "Runtime profile document exceeds the finite preparation byte limit.")
		return h
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil || len(document.Content) != 1 {
		h.issue(file, model.Failed, "yaml", configField{}, "Expected one nonempty Runtime profile YAML document.")
		return h
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		h.issue(file, model.Failed, "documents", configField{}, "Runtime profile input must contain exactly one YAML document.")
		return h
	}
	root := configField{node: document.Content[0]}
	d := configYAML{name: "Runtime profile", title: "Runtime profile", issue: func(code string, f configField, message string) { h.issue(file, model.Failed, code, f, message) }}
	// Bound recursion before invoking configYAML's recursive restricted walk.
	nodes := 0
	var bounded func(*yaml.Node, int) bool
	bounded = func(n *yaml.Node, depth int) bool {
		nodes++
		if nodes > maxRuntimeNodes || depth > maxRuntimeDepth {
			return false
		}
		for _, child := range n.Content {
			if !bounded(child, depth+1) {
				return false
			}
		}
		return true
	}
	if !bounded(root.node, 0) {
		h.issue(file, model.NotCompleted, "node_limit", root, "Runtime profile YAML exceeds finite node/depth limits.")
		return h
	}
	d.restricted(root)
	// object checks apply recursively to inline mappings too, without taking
	// ownership of the adapter's allowed fields or semantic version checks.
	var maps func(configField)
	maps = func(f configField) {
		if f.node.Style&yaml.TaggedStyle != 0 {
			d.issue("yaml_tag", f, "Runtime profile YAML cannot use explicit tags.")
		}
		if f.node.Kind == yaml.MappingNode {
			d.object(f)
			for i := 0; i+1 < len(f.node.Content); i += 2 {
				if f.node.Content[i].Style&yaml.TaggedStyle != 0 {
					d.issue("yaml_tag", configChild(f, f.node.Content[i].Value, f.node.Content[i]), "Runtime profile YAML cannot use explicit key tags.")
				}
				maps(configChild(f, f.node.Content[i].Value, f.node.Content[i+1]))
			}
		} else {
			for i, n := range f.node.Content {
				maps(configChild(f, fmt.Sprint(i), n))
			}
		}
	}
	maps(root)
	fields := d.object(root, "apiVersion", "agentProfiles", "checkProfiles")
	version := d.required(fields, root, "apiVersion")
	if value, ok := d.text(version, false); ok && value != "markitect.runtime/v1" {
		d.issue("version", version, "Unsupported Runtime profile document version.")
	}
	for _, ns := range []struct {
		name     string
		sections []string
	}{{"agentProfiles", []string{"codexAppServer", "claudeCode"}}, {"checkProfiles", []string{"container"}}} {
		catalog := d.required(fields, root, ns.name)
		entries := d.object(catalog)
		names := make([]string, 0, len(entries))
		for name := range entries {
			names = append(names, name)
		}
		slices.Sort(names)
		for _, name := range names {
			entry := entries[name]
			if strings.TrimSpace(name) == "" {
				d.issue("name", entry, "Profile names must be nonempty literal strings.")
			}
			// Exactly one known adapter section selects the profile's adapter.
			wrapper := d.object(entry, ns.sections...)
			for section := range wrapper {
				if !slices.Contains(ns.sections, section) {
					delete(wrapper, section)
				}
			}
			if len(wrapper) != 1 {
				d.issue("section", entry, "A profile selects exactly one adapter section: "+strings.Join(ns.sections, " or ")+".")
				continue
			}
			for section, inline := range wrapper {
				d.object(inline)
				h.profiles = append(h.profiles, ProfileInput{document: file, namespace: ns.name, name: name, section: section, field: inline.pointer, inline: inline.node, instructions: slices.Clone(raw.Instructions)})
			}
		}
	}
	if len(h.profiles) > maxRuntimeProfiles {
		h.issue(file, model.NotCompleted, "profile_limit", root, "Runtime profile count exceeds the finite preparation limit.")
	}
	// No partially decoded envelope is ever supplied to a port.
	if h.status != model.Passed {
		h.profiles = nil
	}
	return h
}
