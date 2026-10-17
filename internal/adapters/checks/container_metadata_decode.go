package checks

import (
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/adapters/container"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"go.yaml.in/yaml/v3"
)

type containerConfig struct {
	engine  string
	profile ContainerProfile
}

func (p *containerPreparation) decode() (containerConfig, bool) {
	var config containerConfig
	fields := p.object(p.input.Inline(), "", "profileVersion", "engineExecutable", "imageID", "executable", "argumentPrefix", "env", "limits", "maxOutputBytes")
	var version uint32
	p.integer(fields["profileVersion"], "/profileVersion", &version)
	if version != 1 {
		p.issue(model.Failed, "version", "/profileVersion", "Only integer container profileVersion 1 is supported.")
	}
	config.engine = p.literal(fields["engineExecutable"], "/engineExecutable")
	if !filepath.IsAbs(config.engine) || filepath.Clean(config.engine) != config.engine {
		p.issue(model.Failed, "engine_path", "/engineExecutable", "Engine executable requires a clean absolute native path.")
	}
	profile := &config.profile
	profile.ImageID = p.literal(fields["imageID"], "/imageID")
	if !containerImageID(profile.ImageID) {
		p.issue(model.Failed, "image_identity", "/imageID", "An immutable lowercase sha256 engine image ID is required.")
	}
	profile.Executable = p.literal(fields["executable"], "/executable")
	if len(profile.Executable) > 256 || !strings.HasPrefix(profile.Executable, "/") || profile.Executable == "/" || path.Clean(profile.Executable) != profile.Executable || strings.Contains(profile.Executable, "\\") {
		p.issue(model.Failed, "executable_path", "/executable", "Executable requires a clean absolute POSIX image file path of at most 256 bytes.")
	}
	// Runner's work/home/tmp mounts and the container engine's standard system
	// mounts obscure these image paths. Their image bytes cannot bind execution.
	for _, mount := range []string{"/work", "/home/agent", "/tmp", "/dev", "/proc", "/sys"} {
		if profile.Executable == mount || strings.HasPrefix(profile.Executable, mount+"/") {
			p.issue(model.Failed, "executable_mount", "/executable", "Executable cannot be inside a runtime mount that obscures image bytes.")
		}
	}
	if slices.Contains([]string{"/etc/hosts", "/etc/hostname", "/etc/resolv.conf"}, profile.Executable) {
		p.issue(model.Failed, "executable_mount", "/executable", "Executable cannot name a container-engine managed runtime file.")
	}
	profile.ArgumentPrefix = p.list(fields["argumentPrefix"], "/argumentPrefix")
	profile.Env = p.list(fields["env"], "/env")
	keys := map[string]bool{}
	for i, entry := range profile.Env {
		key, _, ok := strings.Cut(entry, "=")
		if !ok || !containerEnvKey(key) || keys[key] {
			p.issue(model.Failed, "environment", fmt.Sprintf("/env/%d", i), "Environment requires unique literal KEY=value entries with portable variable names.")
		}
		keys[key] = true
	}
	limits := p.object(fields["limits"], "/limits", "milliCPUs", "memoryBytes", "processes")
	p.integer(limits["milliCPUs"], "/limits/milliCPUs", &profile.Limits.MilliCPUs)
	p.integer(limits["memoryBytes"], "/limits/memoryBytes", &profile.Limits.MemoryBytes)
	p.integer(limits["processes"], "/limits/processes", &profile.Limits.Processes)
	if profile.Limits.MilliCPUs <= 0 || profile.Limits.MemoryBytes <= 0 || profile.Limits.Processes <= 0 || profile.Limits.Processes > int64(int(^uint(0)>>1)) {
		p.issue(model.Failed, "limits", "/limits", "CPU, memory and process limits must be positive; process count must fit the runtime's integer.")
	}
	p.integer(fields["maxOutputBytes"], "/maxOutputBytes", &profile.MaxOutputBytes)
	if profile.MaxOutputBytes <= 0 || profile.MaxOutputBytes > container.MaxCommandOutputBytes {
		p.issue(model.Failed, "output_limit", "/maxOutputBytes", "Output limit must be positive and at most 64 MiB.")
	}
	return config, p.out.Status == model.Passed
}

func (p *containerPreparation) object(n *yaml.Node, field string, names ...string) map[string]*yaml.Node {
	result := map[string]*yaml.Node{}
	if n == nil || n.Kind != yaml.MappingNode || n.Tag != "!!map" || len(n.Content)%2 != 0 {
		p.issue(model.Failed, "object", field, "An explicit closed mapping is required.")
		return result
	}
	for i := 0; i < len(n.Content); i += 2 {
		key, value := n.Content[i], n.Content[i+1]
		if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || !slices.Contains(names, key.Value) || result[key.Value] != nil {
			p.issue(model.Failed, "field", field, "Unknown, duplicate or non-string mapping key.")
		}
		result[key.Value] = value
	}
	for _, name := range names {
		if result[name] == nil {
			p.issue(model.Failed, "required", field+"/"+name, "Required container/v1 field is absent.")
		}
	}
	return result
}

func (p *containerPreparation) integer(n *yaml.Node, field string, target any) {
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag != "!!int" || n.Decode(target) != nil {
		p.issue(model.Failed, "integer", field, "An explicit integer within the target range is required.")
	}
}

func (p *containerPreparation) literal(n *yaml.Node, field string) string {
	if n == nil || n.Kind != yaml.ScalarNode || n.Tag != "!!str" || n.Value == "" || len(n.Value) > 4096 || strings.TrimSpace(n.Value) != n.Value || !utf8.ValidString(n.Value) || strings.ContainsFunc(n.Value, unicode.IsControl) {
		p.issue(model.Failed, "literal", field, "An explicit nonempty bounded literal without controls or surrounding whitespace is required.")
		return ""
	}
	return n.Value
}

func (p *containerPreparation) list(n *yaml.Node, field string) []string {
	if n == nil || n.Kind != yaml.SequenceNode || n.Tag != "!!seq" {
		p.issue(model.Failed, "list", field, "An explicit ordered literal string list is required; an empty list is allowed.")
		return nil
	}
	values := make([]string, 0, len(n.Content))
	for i, child := range n.Content {
		if child.Kind != yaml.ScalarNode || child.Tag != "!!str" || !utf8.ValidString(child.Value) || strings.ContainsRune(child.Value, 0) {
			p.issue(model.Failed, "string", fmt.Sprintf("%s/%d", field, i), "Expected a literal string without NUL.")
			continue
		}
		values = append(values, child.Value)
	}
	return values
}

func containerImageID(value string) bool {
	if len(value) != 71 || !strings.HasPrefix(value, "sha256:") {
		return false
	}
	for _, c := range value[7:] {
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}

func containerEnvKey(value string) bool {
	if value == "" {
		return false
	}
	for i, c := range value {
		if !(c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z' || i > 0 && c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}
