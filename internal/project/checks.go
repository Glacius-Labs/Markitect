package project

import (
	"path"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"go.yaml.in/yaml/v3"
)

// Check is one required check of execution.yaml: the one proof in a Run that
// does not come from an agent.
type Check struct {
	Name        string        `json:"name"`
	Command     string        `json:"command"`     // run by sh -c
	Directory   string        `json:"directory"`   // repository-relative, "." by default
	Timeout     time.Duration `json:"timeout"`     // greater than zero
	Environment string        `json:"environment"` // the container image the check runs in
}

// ParseChecks reads execution.yaml. It holds only the required checks.
func ParseChecks(file string, data []byte) ([]Check, model.Diagnostics) {
	var ds model.Diagnostics
	top, err := model.Mapping(data)
	if err != nil {
		ds.Add("execution.syntax", file, 0, "%v", err)
		return nil, ds
	}
	for key, node := range top {
		if key != "checks" {
			ds.Add("execution.syntax", file, node.Line, "unknown key %q: execution.yaml holds only checks", key)
		}
	}
	list := top["checks"]
	if list == nil {
		return nil, ds
	}
	if list.Kind != yaml.SequenceNode {
		ds.Add("execution.syntax", file, list.Line, "checks must be a list")
		return nil, ds
	}
	var checks []Check
	names := map[string]bool{}
	for _, item := range list.Content {
		var raw struct {
			Name        string `yaml:"name"`
			Command     string `yaml:"command"`
			Directory   string `yaml:"directory"`
			Timeout     string `yaml:"timeout"`
			Environment string `yaml:"environment"`
		}
		if err := item.Decode(&raw); err != nil || item.Kind != yaml.MappingNode {
			ds.Add("execution.syntax", file, item.Line, "each check is a mapping of name, command, directory, timeout and environment")
			continue
		}
		for i := 0; i+1 < len(item.Content); i += 2 {
			switch k := item.Content[i]; k.Value {
			case "name", "command", "directory", "timeout", "environment":
			default:
				ds.Add("execution.syntax", file, k.Line, "unknown check key %q", k.Value)
			}
		}
		c := Check{Name: raw.Name, Command: strings.TrimSpace(raw.Command), Directory: raw.Directory, Environment: strings.TrimSpace(raw.Environment)}
		if c.Directory == "" {
			c.Directory = "."
		}
		timeout, err := time.ParseDuration(raw.Timeout)
		switch {
		case !model.ValidSegment(c.Name):
			ds.Add("execution.syntax", file, item.Line, "a check needs a name of letters, digits, - and _")
		case names[c.Name]:
			ds.Add("execution.syntax", file, item.Line, "the check %s is named twice", c.Name)
		case c.Command == "":
			ds.Add("execution.syntax", file, item.Line, "check %s needs a command", c.Name)
		case c.Directory != path.Clean(c.Directory) || path.IsAbs(c.Directory) || strings.HasPrefix(c.Directory, ".."):
			ds.Add("execution.syntax", file, item.Line, "check %s: directory must be repository-relative", c.Name)
		case err != nil || timeout <= 0:
			ds.Add("execution.syntax", file, item.Line, "check %s needs a timeout such as 10m", c.Name)
		case c.Environment == "":
			ds.Add("execution.syntax", file, item.Line, "check %s needs an environment: the container image it runs in", c.Name)
		default:
			c.Timeout = timeout
			names[c.Name] = true
			checks = append(checks, c)
		}
	}
	return checks, ds
}
