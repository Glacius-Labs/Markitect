package codexappserver

import (
	"fmt"
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

type metadataConfig struct {
	executable, stable, experimental string
	instructions                     []apply.SourceLocation
}

func (p *preparation) decode() (metadataConfig, bool) {
	var c metadataConfig
	start := len(p.out.Diagnostics)
	if p.input.Namespace() != "agentProfiles" || p.input.Binding() == (apply.ProfileBinding{}) {
		p.issue(model.Failed, "input", "", "A bound Agent profile input is required.")
		return c, false
	}
	root := p.input.Inline()
	fields := p.object(root, "", "profileVersion", "metadataPin", "executable", "schemas", "instructions")
	v := fields["profileVersion"]
	if v == nil || v.Kind != yaml.ScalarNode || v.Tag != "!!int" || v.Value != "1" {
		p.issue(model.Failed, "version", "/profileVersion", "Only integer profileVersion 1 is supported.")
	}
	if p.text(fields["metadataPin"], "/metadataPin") != metadataPinName {
		p.issue(model.Failed, "pin", "/metadataPin", "Only the fixed historical offline metadata pin is supported.")
	}
	c.executable = p.absolute(fields["executable"], "/executable")
	schemas := p.object(fields["schemas"], "/schemas", "stable", "experimental")
	c.stable = p.absolute(schemas["stable"], "/schemas/stable")
	c.experimental = p.absolute(schemas["experimental"], "/schemas/experimental")
	if c.stable != "" && c.experimental != "" && (overlaps(c.stable, c.experimental) || overlaps(c.experimental, c.stable)) {
		p.issue(model.Failed, "schema_boundaries", "/schemas", "Schema inventories must have distinct nonnested literal boundaries.")
	}
	n := fields["instructions"]
	if n == nil || n.Kind != yaml.SequenceNode {
		p.issue(model.Failed, "instructions", "/instructions", "An explicit ordered instruction list is required.")
	} else {
		for i, child := range n.Content {
			field := fmt.Sprintf("/instructions/%d", i)
			entry := p.object(child, field, "boundary", "path")
			// Instruction boundaries are opaque captured coordinates. Unlike our
			// executable/schema acquisition paths, this port never resolves them.
			boundary := p.text(entry["boundary"], field+"/boundary")
			path, err := snapshot.NewPath(p.text(entry["path"], field+"/path"))
			if err != nil {
				p.issue(model.Failed, "instruction_path", field+"/path", err.Error())
			}
			c.instructions = append(c.instructions, apply.SourceLocation{Boundary: boundary, Path: path})
		}
	}
	return c, len(p.out.Diagnostics) == start
}

func (p *preparation) object(n *yaml.Node, field string, names ...string) map[string]*yaml.Node {
	result := map[string]*yaml.Node{}
	if n == nil || n.Kind != yaml.MappingNode {
		p.issue(model.Failed, "object", field, "Expected a closed mapping.")
		return result
	}
	for i := 0; i+1 < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		allowed := false
		for _, name := range names {
			if key.Value == name {
				allowed = true
			}
		}
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || !allowed || result[key.Value] != nil {
			p.issue(model.Failed, "field", field, "Unknown, duplicate or non-string mapping key.")
		}
		result[key.Value] = value
	}
	for _, name := range names {
		if result[name] == nil {
			p.issue(model.Failed, "required", field+"/"+name, "Required field is absent.")
		}
	}
	return result
}

func (p *preparation) text(n *yaml.Node, field string) string {
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag != "!!str" || strings.TrimSpace(n.Value) == "" {
		p.issue(model.Failed, "string", field, "Expected an explicit nonempty string.")
		return ""
	}
	return n.Value
}

func literalAbsolute(s string) bool {
	if !utf8.ValidString(s) || !filepath.IsAbs(s) || filepath.Clean(s) != s || strings.HasPrefix(s, `\\`) {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}

func (p *preparation) absolute(n *yaml.Node, field string) string {
	s := p.text(n, field)
	if !literalAbsolute(s) {
		p.issue(model.Failed, "literal_path", field, "Expected an exact native absolute path without aliases, dot segments or redirects.")
		return ""
	}
	return s
}
