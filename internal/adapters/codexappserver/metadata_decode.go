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
	version                         int
	credentialRef                   string
	maxWireBytes                    int64
	session                         SessionProfile
}

func (p *preparation) decode(root *yaml.Node) (metadataConfig, bool) {
	var c metadataConfig
	start := len(p.out.Diagnostics)
	names := []string{"profileVersion", "engineExecutable", "imageID", "agentdVersion", "recordedCodexSchemaDigest", "instructions"}
	// Select the closed field set without interpreting any execution setting in v2.
	if root != nil && root.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(root.Content); i += 2 {
			if root.Content[i].Value == "profileVersion" && root.Content[i+1].Tag == "!!int" && root.Content[i+1].Value == "3" {
				c.version = 3
			}
		}
	}
	if c.version == 3 {
		names = append(names, "credentialRef", "maxWireBytes", "session")
	}
	fields := p.object(root, "", names...)
	v := fields["profileVersion"]
	if v == nil || v.Kind != yaml.ScalarNode || v.Tag != "!!int" || (v.Value != "2" && v.Value != "3") {
		p.issue(model.Failed, "version", "/profileVersion", "Only integer profileVersion 2 (metadata) or 3 (execution settings) is supported.")
	} else if v.Value == "2" {
		c.version = 2
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
	if c.version == 3 {
		c.credentialRef = p.text(fields["credentialRef"], "/credentialRef")
		p.scalar(fields["maxWireBytes"], "/maxWireBytes", &c.maxWireBytes, "!!int")
		if c.maxWireBytes <= 0 || c.maxWireBytes > maxProtocolMessage {
			p.issue(model.Failed, "wire_limit", "/maxWireBytes", "Wire limit must be positive and at most 1 MiB.")
		}
		c.session = p.decodeSession(fields["session"])
	}
	return c, len(p.out.Diagnostics) == start
}

func (p *preparation) decodeSession(n *yaml.Node) SessionProfile {
	fields := p.object(n, "/session", "version", "model", "effort", "baseInstructions", "developerInstructions", "store", "cpus", "memoryBytes", "pids", "tmpBytes", "network")
	var s SessionProfile
	p.scalar(fields["version"], "/session/version", &s.Version, "!!int")
	s.Model = p.text(fields["model"], "/session/model")
	s.Effort = p.text(fields["effort"], "/session/effort")
	s.Store = p.text(fields["store"], "/session/store")
	p.scalar(fields["baseInstructions"], "/session/baseInstructions", &s.BaseInstructions, "!!str")
	p.scalar(fields["developerInstructions"], "/session/developerInstructions", &s.DeveloperInstructions, "!!str")
	p.scalar(fields["network"], "/session/network", &s.Network, "!!str")
	p.scalar(fields["cpus"], "/session/cpus", &s.CPUs, "!!int", "!!float")
	p.scalar(fields["memoryBytes"], "/session/memoryBytes", &s.MemoryBytes, "!!int")
	p.scalar(fields["pids"], "/session/pids", &s.Pids, "!!int")
	p.scalar(fields["tmpBytes"], "/session/tmpBytes", &s.TmpBytes, "!!int")
	if _, err := NewSessionProfile(s); err != nil {
		p.issue(model.Failed, "session", "/session", "Session settings do not satisfy the Codex SessionProfile v1 contract.")
	}
	return s
}

func (p *preparation) scalar(n *yaml.Node, field string, target any, tags ...string) {
	if n != nil && n.Kind == yaml.ScalarNode {
		for _, tag := range tags {
			if n.Tag == tag && n.Decode(target) == nil {
				return
			}
		}
	}
	p.issue(model.Failed, "scalar", field, "Expected an explicit scalar with the required type and range.")
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
