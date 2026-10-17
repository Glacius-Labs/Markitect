package apply

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"go.yaml.in/yaml/v3"
)

const maxHostDocumentBytes = 4 << 20
const maxHostNodes = 100000
const maxHostDepth = 64
const maxHostProfiles = 1024

type hostEnvelope struct {
	status      model.Status
	profiles    []ProfileInput
	diagnostics []HostProfileDiagnostic
}

func (h *hostEnvelope) issue(file SourceFile, status model.Status, code string, f configField, message string) {
	var position *model.Position
	if f.node != nil && f.node.Line > 0 {
		position = &model.Position{Line: f.node.Line, Column: f.node.Column}
	}
	h.diagnostics = append(h.diagnostics, HostProfileDiagnostic{Code: "host." + code, Source: hostSource(file, f.pointer), Position: position, Message: message})
	if status == model.Failed || h.status != model.Failed {
		h.status = status
	}
}

func decodeHost(raw RawInputs) hostEnvelope {
	h := hostEnvelope{status: model.Passed}
	if len(raw.HostProfiles) != 1 {
		status := model.Failed
		if len(raw.HostProfiles) == 0 {
			status = model.NotCompleted
		}
		h.issue(SourceFile{}, status, "document_count", configField{}, "Exactly one explicitly captured Host document is required.")
		return h
	}
	file := raw.HostProfiles[0]
	data := file.File.Bytes()
	if len(data) > maxHostDocumentBytes {
		h.issue(file, model.NotCompleted, "byte_limit", configField{}, "Host document exceeds the finite preparation byte limit.")
		return h
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document yaml.Node
	if err := decoder.Decode(&document); err != nil || len(document.Content) != 1 {
		h.issue(file, model.Failed, "yaml", configField{}, "Expected one nonempty Host YAML document.")
		return h
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		h.issue(file, model.Failed, "documents", configField{}, "Host input must contain exactly one YAML document.")
		return h
	}
	root := configField{node: document.Content[0]}
	d := configYAML{name: "Host", title: "Host", issue: func(code string, f configField, message string) { h.issue(file, model.Failed, code, f, message) }}
	// Bound recursion before invoking configYAML's recursive restricted walk.
	nodes := 0
	var bounded func(*yaml.Node, int) bool
	bounded = func(n *yaml.Node, depth int) bool {
		nodes++
		if nodes > maxHostNodes || depth > maxHostDepth {
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
		h.issue(file, model.NotCompleted, "node_limit", root, "Host YAML exceeds finite node/depth limits.")
		return h
	}
	d.restricted(root)
	// object checks apply recursively to inline mappings too, without taking
	// ownership of the adapter's allowed fields or semantic version checks.
	var maps func(configField)
	maps = func(f configField) {
		if f.node.Style&yaml.TaggedStyle != 0 {
			d.issue("yaml_tag", f, "Host YAML cannot use explicit tags.")
		}
		if f.node.Kind == yaml.MappingNode {
			d.object(f)
			for i := 0; i+1 < len(f.node.Content); i += 2 {
				if f.node.Content[i].Style&yaml.TaggedStyle != 0 {
					d.issue("yaml_tag", configChild(f, f.node.Content[i].Value, f.node.Content[i]), "Host YAML cannot use explicit key tags.")
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
	if value, ok := d.text(version, false); ok && value != "markitect.host/v1" {
		d.issue("version", version, "Unsupported Host document version.")
	}
	for _, ns := range []struct {
		name     string
		sections []string
	}{{"agentProfiles", []string{"codexAppServer", "claudeCode"}}, {"checkProfiles", []string{"process", "container"}}} {
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
			// Exactly one adapter section selects the profile's adapter.
			wrapper := d.object(entry, ns.sections...)
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
	if len(h.profiles) > maxHostProfiles {
		h.issue(file, model.NotCompleted, "profile_limit", root, "Host profile count exceeds the finite preparation limit.")
	}
	// No partially decoded envelope is ever supplied to a port.
	if h.status != model.Passed {
		h.profiles = nil
	}
	return h
}
