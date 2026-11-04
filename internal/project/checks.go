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
	Name      string        `json:"name"`
	Command   string        `json:"command"`   // run by sh -c
	Directory string        `json:"directory"` // repository-relative, "." by default
	Timeout   time.Duration `json:"timeout"`   // greater than zero
}

// Execution is what execution.yaml holds: the required checks, and the AI model
// and effort a Run's sessions use, empty for the tool's default.
type Execution struct {
	Checks []Check
	Model  string
	Effort string
}

// ParseExecution reads execution.yaml.
func ParseExecution(file string, data []byte) (Execution, model.Diagnostics) {
	var ds model.Diagnostics
	var out Execution
	top, err := model.Mapping(data)
	if err != nil {
		ds.Add("execution.syntax", file, 0, "%v", err)
		return out, ds
	}
	for key, node := range top {
		switch key {
		case "checks":
		case "model":
			out.Model = strings.TrimSpace(node.Value)
		case "effort":
			out.Effort = strings.TrimSpace(node.Value)
		default:
			ds.Add("execution.syntax", file, node.Line, "unknown key %q: execution.yaml holds only checks, model and effort", key)
		}
	}
	list := top["checks"]
	if list == nil {
		return out, ds
	}
	if list.Kind != yaml.SequenceNode {
		ds.Add("execution.syntax", file, list.Line, "checks must be a list")
		return out, ds
	}
	names := map[string]bool{}
	for _, item := range list.Content {
		var raw struct {
			Name      string `yaml:"name"`
			Command   string `yaml:"command"`
			Directory string `yaml:"directory"`
			Timeout   string `yaml:"timeout"`
		}
		if err := item.Decode(&raw); err != nil || item.Kind != yaml.MappingNode {
			ds.Add("execution.syntax", file, item.Line, "each check is a mapping of name, command, directory and timeout")
			continue
		}
		for i := 0; i+1 < len(item.Content); i += 2 {
			switch k := item.Content[i]; k.Value {
			case "name", "command", "directory", "timeout":
			default:
				ds.Add("execution.syntax", file, k.Line, "unknown check key %q", k.Value)
			}
		}
		c := Check{Name: raw.Name, Command: strings.TrimSpace(raw.Command), Directory: raw.Directory}
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
		default:
			c.Timeout = timeout
			names[c.Name] = true
			out.Checks = append(out.Checks, c)
		}
	}
	return out, ds
}
