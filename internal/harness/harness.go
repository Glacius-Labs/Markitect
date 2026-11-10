// Package harness is the configuration of the harness of the work (DEC-103):
// one file format, harness.yaml, at two levels. The user's
// ~/.markitect/harness.yaml is not committed and holds what applies to all of a
// user's projects; the project's .markitect/harness.yaml is committed with the
// project. Key by key, the project's value beats the user's, and the user's beats
// the built-in one, as git config has it.
//
// It holds the model and effort of each role's agent, the strategy of each skill
// and the custom workflows, and it renders the agents' front matter from them. It
// reads and writes bytes only: where the files lie is the caller's.
package harness

import (
	"bytes"
	"fmt"
	"regexp"
	"slices"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

// FileName is the name of the configuration file at either level.
const FileName = "harness.yaml"

// Origin says where a value of the configuration comes from.
type Origin int

// The levels, from the weakest to the strongest.
const (
	BuiltIn Origin = iota
	User
	Project
)

func (o Origin) String() string {
	switch o {
	case User:
		return "user"
	case Project:
		return "project"
	}
	return "built-in"
}

// Roles are the four roles whose agents the configuration sets (DEC-100), and
// Skills the skills that take a strategy.
var (
	Roles   = []string{"builder", "reviewer", "architect", "auditor"}
	Skills  = []string{"build", "review", "cleanup", "bughunt", "audit"}
	Efforts = []string{"low", "medium", "high", "xhigh", "max"}
)

// The built-in strategies. Any other strategy names a custom workflow.
const (
	// Auto has Claude estimate the work and choose sequential or parallel; the
	// skills choose sequential until the parallel strategy has proven itself
	// on a long list.
	Auto = "auto"
	// Sequential runs one agent of each role at a time, in the session.
	Sequential = "sequential"
	// Parallel runs the shipped workflow, ParallelWorkflow.
	Parallel = "parallel"
)

// Strategies are the built-in strategies.
var Strategies = []string{Auto, Sequential, Parallel}

// ParallelWorkflow is the name of the workflow the parallel strategy runs, which
// init installs as .claude/workflows/markitect-parallel.js.
const ParallelWorkflow = "markitect-parallel"

// ownPrefix starts the name of every file init and update ship and keep, so a
// custom workflow may not start with it.
const ownPrefix = "markitect-"

// Built-in values: the agents as the binary ships their files, and auto.
const (
	defaultStrategy = Auto
	strategyDefault = "strategy.default"
)

// agentDefaults are the built-in model and effort of each role. The roles that
// work on the spec think on a larger model, since the spec is little text that
// everything else is built from; the roles that work on the code build and
// verify from it on a smaller one.
var agentDefaults = map[string]struct{ model, effort string }{
	"builder":   {"sonnet", "high"},
	"reviewer":  {"sonnet", "high"},
	"architect": {"opus", "medium"},
	"auditor":   {"opus", "medium"},
}

var (
	// modelName is what a model may look like: an alias such as opus, a full
	// model ID, opus[1m] or inherit. Nothing else reaches an agent file.
	modelName = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._\[\]-]*$`)
	// workflowName is what the name of a custom workflow may look like.
	workflowName = regexp.MustCompile(`^[a-z0-9][a-z0-9-]*$`)
)

// builtIn returns the values no file sets.
func builtIn() map[string]string {
	values := map[string]string{strategyDefault: defaultStrategy}
	for _, r := range Roles {
		values["agents."+r+".model"] = agentDefaults[r].model
		values["agents."+r+".effort"] = agentDefaults[r].effort
	}
	return values
}

// keyForms say which keys there are, for an error about one that is not.
const keyForms = "agents.ROLE.model, agents.ROLE.effort (ROLE: builder, reviewer, architect or auditor), strategy.default, strategy.SKILL (SKILL: build, review, cleanup, bughunt or audit) and workflows.NAME"

// CheckKey says why a key is not one the configuration takes, or nil.
func CheckKey(key string) error {
	parts := strings.Split(key, ".")
	switch {
	case len(parts) == 3 && parts[0] == "agents" && slices.Contains(Roles, parts[1]) && (parts[2] == "model" || parts[2] == "effort"):
	case len(parts) == 2 && parts[0] == "strategy" && (parts[1] == "default" || slices.Contains(Skills, parts[1])):
	case len(parts) == 2 && parts[0] == "workflows":
		if !workflowName.MatchString(parts[1]) || slices.Contains(Strategies, parts[1]) || strings.HasPrefix(parts[1], ownPrefix) {
			return fmt.Errorf("%s: a workflow's name is lower-case letters, digits and hyphens, not auto, sequential or parallel, and does not start with %s, which init and update keep for the files they ship", key, ownPrefix)
		}
	default:
		return fmt.Errorf("unknown key %q; the keys are %s", key, keyForms)
	}
	return nil
}

// Check says why a key or its value is not one the configuration takes, or nil.
// The workflow a strategy names is checked against the whole configuration by
// Config.Problems.
func Check(key, value string) error {
	if err := CheckKey(key); err != nil {
		return err
	}
	parts := strings.Split(key, ".")
	switch {
	case parts[0] == "agents" && parts[2] == "model":
		if !modelName.MatchString(value) {
			return fmt.Errorf("%s: %q is no model name, such as sonnet, opus, haiku, a full model ID or inherit", key, value)
		}
	case parts[0] == "agents":
		if !slices.Contains(Efforts, value) {
			return fmt.Errorf("%s: %q is no effort; it is one of %s", key, value, strings.Join(Efforts, ", "))
		}
	case parts[0] == "strategy":
		if !slices.Contains(Strategies, value) && (!workflowName.MatchString(value) || strings.HasPrefix(value, ownPrefix)) {
			return fmt.Errorf("%s: %q is no strategy; it is auto, sequential, parallel or the name of a custom workflow, which does not start with %s", key, value, ownPrefix)
		}
	default: // workflows
		if strings.TrimSpace(value) == "" || strings.ContainsAny(value, "\r\n") {
			return fmt.Errorf("%s: the value is the path of the workflow's script, on one line", key)
		}
	}
	return nil
}

// File is one harness.yaml, kept as a YAML document so that a change keeps the
// owner's comments and order.
type File struct {
	doc *yaml.Node
	// head is the text of a file that holds only comments, which a YAML document
	// does not keep.
	head string
}

// Parse reads a harness.yaml and checks every key and value in it. An empty
// input is an empty file.
func Parse(data []byte) (File, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return File{}, err
	}
	f := File{doc: &doc}
	if doc.Kind == 0 {
		f.doc = &yaml.Node{Kind: yaml.DocumentNode}
		f.head = strings.TrimRight(string(data), "\r\n\t ")
	}
	if _, err := f.values(); err != nil {
		return File{}, err
	}
	return f, nil
}

// root returns the top mapping, made when create is set and there is none.
func (f *File) root(create bool) *yaml.Node {
	if f.doc == nil {
		if !create {
			return nil
		}
		f.doc = &yaml.Node{Kind: yaml.DocumentNode}
	}
	if len(f.doc.Content) == 0 {
		if !create {
			return nil
		}
		f.doc.Content = []*yaml.Node{{Kind: yaml.MappingNode, Tag: "!!map"}}
		if f.head != "" {
			f.doc.HeadComment = f.head
			f.head = ""
		}
	}
	return f.doc.Content[0]
}

// Values returns every key the file sets, with its value.
func (f File) Values() map[string]string {
	values, _ := f.values()
	return values
}

func (f File) values() (map[string]string, error) {
	values := map[string]string{}
	top := f.root(false)
	if top == nil {
		return values, nil
	}
	if top.Kind != yaml.MappingNode {
		return nil, fmt.Errorf("line %d: harness.yaml holds a mapping of agents, strategy and workflows", top.Line)
	}
	var walk func(prefix string, n *yaml.Node) error
	walk = func(prefix string, n *yaml.Node) error {
		for i := 0; i+1 < len(n.Content); i += 2 {
			k, v := n.Content[i], n.Content[i+1]
			key := k.Value
			if prefix != "" {
				key = prefix + "." + k.Value
			}
			switch v.Kind {
			case yaml.MappingNode:
				if err := walk(key, v); err != nil {
					return err
				}
				continue
			case yaml.ScalarNode:
			default:
				return fmt.Errorf("line %d: %s must be a single value", v.Line, key)
			}
			if err := Check(key, v.Value); err != nil {
				return fmt.Errorf("line %d: %w", k.Line, err)
			}
			if _, dup := values[key]; dup {
				return fmt.Errorf("line %d: %s is set twice", k.Line, key)
			}
			values[key] = v.Value
		}
		return nil
	}
	if err := walk("", top); err != nil {
		return nil, err
	}
	return values, nil
}

// Set sets a key to a value, making the mappings it lies in.
func (f *File) Set(key, value string) error {
	if err := Check(key, value); err != nil {
		return err
	}
	n := f.root(true)
	parts := strings.Split(key, ".")
	for i, part := range parts {
		last := i == len(parts)-1
		var found *yaml.Node
		for j := 0; j+1 < len(n.Content); j += 2 {
			if n.Content[j].Value == part {
				found = n.Content[j+1]
				break
			}
		}
		if last {
			if found == nil {
				found = &yaml.Node{Kind: yaml.ScalarNode}
				n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, found)
			}
			found.Kind, found.Tag, found.Value, found.Style, found.Content = yaml.ScalarNode, "!!str", value, 0, nil
			return nil
		}
		if found == nil || found.Kind != yaml.MappingNode {
			child := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
			if found == nil {
				n.Content = append(n.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: part}, child)
			} else {
				*found = *child
			}
			found = child
		}
		n = found
	}
	return nil
}

// Unset removes a key and the mappings it leaves empty, and says whether the file
// set it.
func (f *File) Unset(key string) bool {
	top := f.root(false)
	if top == nil {
		return false
	}
	removed, orphan := unset(top, strings.Split(key, "."))
	switch {
	case orphan == "":
	case len(top.Content) > 0:
		top.FootComment = joinComments(top.FootComment, orphan)
	default:
		f.doc.HeadComment = joinComments(f.doc.HeadComment, orphan)
	}
	return removed
}

// unset removes a key from a mapping, and the mappings it leaves empty. It
// returns whether it removed one, and the comments above the keys it removed that
// it found no key below to keep them on, so that the owner's notes stay.
func unset(n *yaml.Node, parts []string) (bool, string) {
	for j := 0; j+1 < len(n.Content); j += 2 {
		k, v := n.Content[j], n.Content[j+1]
		if k.Value != parts[0] {
			continue
		}
		orphan := ""
		if len(parts) == 1 {
			if v.Kind != yaml.ScalarNode {
				return false, ""
			}
		} else {
			if v.Kind != yaml.MappingNode {
				return false, ""
			}
			removed, below := unset(v, parts[1:])
			if !removed {
				return false, ""
			}
			if len(v.Content) > 0 {
				if below != "" {
					v.FootComment = joinComments(v.FootComment, below)
				}
				return true, ""
			}
			orphan = below
		}
		orphan = joinComments(k.HeadComment, orphan)
		n.Content = append(n.Content[:j], n.Content[j+2:]...)
		if orphan != "" && j < len(n.Content) {
			n.Content[j].HeadComment = joinComments(orphan, n.Content[j].HeadComment)
			orphan = ""
		}
		return true, orphan
	}
	return false, ""
}

func joinComments(a, b string) string {
	switch {
	case a == "":
		return b
	case b == "":
		return a
	}
	return a + "\n" + b
}

// Empty says whether the file sets nothing.
func (f File) Empty() bool { return len(f.Values()) == 0 }

// Bytes returns the file as text, with two spaces of indent.
func (f File) Bytes() ([]byte, error) {
	top := f.root(false)
	if top == nil || len(top.Content) == 0 {
		head := f.head
		if f.doc != nil && head == "" {
			head = f.doc.HeadComment
		}
		if head == "" {
			return nil, nil
		}
		return []byte(head + "\n"), nil
	}
	var b bytes.Buffer
	enc := yaml.NewEncoder(&b)
	enc.SetIndent(2)
	if err := enc.Encode(f.doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// New returns an empty file that starts with a comment, each line of which is
// written as given.
func New(comment string) File {
	return File{doc: &yaml.Node{Kind: yaml.DocumentNode}, head: strings.TrimRight(comment, "\n")}
}

// Value is one value of the configuration and where it comes from.
type Value struct {
	Key    string `json:"key"`
	Value  string `json:"value"`
	Origin Origin `json:"-"`
	// Via names the key a value is taken from when the key itself is set
	// nowhere: strategy.default for the strategy of a skill.
	Via string `json:"via,omitempty"`
}

// Config is the configuration in force: the project's values over the user's
// over the built-in ones.
type Config struct {
	byKey map[string]Value
}

// Resolve merges the user's file and the project's, key by key. Either may be
// the zero File.
func Resolve(user, project File) Config {
	c := Config{byKey: map[string]Value{}}
	for k, v := range builtIn() {
		c.byKey[k] = Value{Key: k, Value: v, Origin: BuiltIn}
	}
	for k, v := range user.Values() {
		c.byKey[k] = Value{Key: k, Value: v, Origin: User}
	}
	for k, v := range project.Values() {
		c.byKey[k] = Value{Key: k, Value: v, Origin: Project}
	}
	for _, s := range Skills {
		k := "strategy." + s
		if _, set := c.byKey[k]; !set {
			d := c.byKey[strategyDefault]
			c.byKey[k] = Value{Key: k, Value: d.Value, Origin: d.Origin, Via: strategyDefault}
		}
	}
	return c
}

// Get returns the value of a key in force.
func (c Config) Get(key string) (Value, bool) {
	v, ok := c.byKey[key]
	return v, ok
}

// Values returns every value in force: the agents by role, the strategies, the
// default first, then the workflows by name.
func (c Config) Values() []Value {
	var out []Value
	for _, r := range Roles {
		out = append(out, c.byKey["agents."+r+".model"], c.byKey["agents."+r+".effort"])
	}
	out = append(out, c.byKey[strategyDefault])
	for _, s := range Skills {
		out = append(out, c.byKey["strategy."+s])
	}
	var workflows []string
	for k := range c.byKey {
		if strings.HasPrefix(k, "workflows.") {
			workflows = append(workflows, k)
		}
	}
	sort.Strings(workflows)
	for _, k := range workflows {
		out = append(out, c.byKey[k])
	}
	return out
}

// Agent returns the model and effort of a role's agent.
func (c Config) Agent(role string) (model, effort string) {
	return c.byKey["agents."+role+".model"].Value, c.byKey["agents."+role+".effort"].Value
}

// Strategy returns the strategy of a skill: auto, sequential, parallel or the
// name of a custom workflow.
func (c Config) Strategy(skill string) string {
	return c.byKey["strategy."+skill].Value
}

// Workflow returns the path of a custom workflow's script.
func (c Config) Workflow(name string) (string, bool) {
	v, ok := c.byKey["workflows."+name]
	return v.Value, ok
}

// Problems are what keeps the configuration from working: a strategy that names
// a workflow no level defines.
func (c Config) Problems() []string {
	var out []string
	for _, k := range append([]string{strategyDefault}, prefixed("strategy.", Skills)...) {
		v := c.byKey[k]
		if v.Via != "" || slices.Contains(Strategies, v.Value) {
			continue
		}
		if _, ok := c.Workflow(v.Value); !ok {
			out = append(out, fmt.Sprintf("%s names the workflow %s, which no harness.yaml defines; set workflows.%s to the path of its script", k, v.Value, v.Value))
		}
	}
	return out
}

func prefixed(prefix string, names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = prefix + n
	}
	return out
}

// AgentPrefix is what the file of a role's agent starts with below .claude/agents/.
const AgentPrefix = ".claude/agents/markitect-"

// RoleOf returns the role whose agent a harness file is, such as builder for
// .claude/agents/markitect-builder.md.
func RoleOf(file string) (string, bool) {
	name, ok := strings.CutPrefix(file, AgentPrefix)
	if !ok {
		return "", false
	}
	role, ok := strings.CutSuffix(name, ".md")
	if !ok || !slices.Contains(Roles, role) {
		return "", false
	}
	return role, true
}

// RenderAgent returns the file of a role's agent with the model and effort the
// configuration gives the role, in place of the ones its front matter names.
func (c Config) RenderAgent(role string, data []byte) []byte {
	model, effort := c.Agent(role)
	lines := strings.SplitAfter(string(data), "\n")
	if len(lines) == 0 || strings.TrimRight(lines[0], "\r\n") != "---" {
		return data
	}
	for i := 1; i < len(lines); i++ {
		line := strings.TrimRight(lines[i], "\r\n")
		if line == "---" {
			break
		}
		switch {
		case strings.HasPrefix(line, "model:"):
			lines[i] = "model: " + model + "\n"
		case strings.HasPrefix(line, "effort:"):
			lines[i] = "effort: " + effort + "\n"
		}
	}
	return []byte(strings.Join(lines, ""))
}
