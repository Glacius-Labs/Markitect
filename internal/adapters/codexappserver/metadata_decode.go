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
	engine, imageID, agentd, schema string
	instructions                    []apply.SourceLocation
}

func (p *preparation) decode(root *yaml.Node) (metadataConfig, bool) {
	var c metadataConfig
	start := len(p.out.Diagnostics)
	fields := p.object(root, "", "profileVersion", "engineExecutable", "imageID", "agentdVersion", "recordedCodexSchemaDigest", "instructions")
	v := fields["profileVersion"]
	if v == nil || v.Kind != yaml.ScalarNode || v.Tag != "!!int" || v.Value != "2" {
		p.issue(model.Failed, "version", "/profileVersion", "Only integer profileVersion 2 is supported.")
	}
	c.engine = p.text(fields["engineExecutable"], "/engineExecutable")
	if !filepath.IsAbs(c.engine) || filepath.Clean(c.engine) != c.engine {
		p.issue(model.Failed, "engine_path", "/engineExecutable", "Engine executable requires an explicit canonical native absolute path.")
	}
	c.imageID = p.text(fields["imageID"], "/imageID")
	if !sha256Literal(c.imageID) {
		p.issue(model.Failed, "image_identity", "/imageID", "Select an immutable sha256 image ID; tags and aliases are unsupported.")
	}
	c.agentd = p.text(fields["agentdVersion"], "/agentdVersion")
	c.schema = p.text(fields["recordedCodexSchemaDigest"], "/recordedCodexSchemaDigest")
	if !sha256Literal(c.schema) {
		p.issue(model.Failed, "recorded_schema", "/recordedCodexSchemaDigest", "Select the expected build-recorded sha256 schema digest.")
	}
	n := fields["instructions"]
	if n == nil || n.Kind != yaml.SequenceNode {
		p.issue(model.Failed, "instructions", "/instructions", "An explicit ordered captured instruction list is required.")
	} else {
		for i, child := range n.Content {
			field := fmt.Sprintf("/instructions/%d", i)
			entry := p.object(child, field, "boundary", "path")
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
	if n == nil || n.Kind != yaml.MappingNode || n.Tag != "!!map" || len(n.Content)%2 != 0 {
		p.issue(model.Failed, "object", field, "Expected a closed mapping.")
		return result
	}
	for i := 0; i < len(n.Content); i += 2 {
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
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag != "!!str" || !literal(n.Value) {
		p.issue(model.Failed, "literal", field, "Expected a bounded nonempty literal without controls or surrounding whitespace.")
		return ""
	}
	return n.Value
}

func literal(s string) bool {
	return len(s) > 0 && len(s) <= 4096 && utf8.ValidString(s) && strings.TrimSpace(s) == s && !strings.ContainsFunc(s, unicode.IsControl)
}

func sha256Literal(s string) bool {
	if len(s) != 71 || !strings.HasPrefix(s, "sha256:") {
		return false
	}
	for _, c := range s[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
