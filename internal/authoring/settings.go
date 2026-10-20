package authoring

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"slices"
	"strconv"

	"go.yaml.in/yaml/v3"

	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

// ExecutionPath is the project's execution settings document.
const ExecutionPath = ".markitect/execution.yaml"

// Settings are the parts of the execution settings the cockpit edits: the
// budget, the role profiles, the per-Package profile overrides and the team.
// Everything else of the document (ownership, top-level and per-Package
// instructions, scope, checks) is kept as it is. Head is the committed source
// the settings were read from. Packages lists the local Packages of the model.
type Settings struct {
	Head      string            `json:"head"`
	Budget    Budget            `json:"budget"`
	Profiles  Profiles          `json:"profiles"`
	Overrides []PackageProfiles `json:"overrides"`
	Team      *Team             `json:"team"`
	Packages  []string          `json:"packages"`
	Text      string            `json:"text"`
}

// Budget bounds a run. Durations keep the document's notation, such as 25m.
type Budget struct {
	ExecutionTime string `json:"executionTime" yaml:"executionTime"`
	TurnTimeout   string `json:"turnTimeout" yaml:"turnTimeout"`
	AgentTurns    int    `json:"agentTurns" yaml:"agentTurns"`
	Passes        int    `json:"passes" yaml:"passes"`
}

// Profiles names the Runtime profiles of the Execute and Verify roles.
type Profiles struct {
	Execute string `json:"execute" yaml:"execute"`
	Verify  string `json:"verify" yaml:"verify"`
}

// PackageProfiles overrides the profiles for one Package; an empty name keeps
// the default.
type PackageProfiles struct {
	Package string `json:"package"`
	Execute string `json:"execute"`
	Verify  string `json:"verify"`
}

// Team replaces the product's default team: at most MaxHelpers active helpers
// with the given roles.
type Team struct {
	MaxHelpers int      `json:"maxHelpers" yaml:"maxHelpers"`
	Helpers    []Helper `json:"helpers" yaml:"helpers"`
}

// Helper is one helper role of a Manager's team.
type Helper struct {
	Role         string   `json:"role" yaml:"role"`
	Profile      string   `json:"profile,omitempty" yaml:"profile"`
	Writes       bool     `json:"writes" yaml:"writes"`
	Instructions []string `json:"instructions,omitempty" yaml:"instructions"`
}

// ErrMoved reports settings edited from a committed source that is no longer
// the head: applying them could revert what was committed since.
var ErrMoved = errors.New("the committed source changed since the settings were read; read them again")

// ErrNoExecution reports a project without execution settings.
var ErrNoExecution = errors.New("the project has no " + ExecutionPath)

// Settings reads the execution settings of the committed source.
func (s *Service) Settings(ctx context.Context) (Settings, error) {
	st, err := s.stand(ctx)
	if err != nil {
		return Settings{}, err
	}
	content, err := lookup(st.tree, ExecutionPath)
	if err != nil {
		return Settings{}, err
	}
	settings, err := readSettings(content)
	if err != nil {
		return Settings{}, err
	}
	settings.Head = st.head
	if selections, complete := st.attempt.Contexts().Selections(); complete {
		for _, sel := range selections {
			if sel.Package() != "." {
				settings.Packages = append(settings.Packages, sel.Package())
			}
		}
	}
	return settings, nil
}

// ProposeSettings makes a proposal that changes only the execution settings
// document to the given settings, from the capture of the head they were read
// from when it is recent. Settings read from another head are refused.
func (s *Service) ProposeSettings(ctx context.Context, edit Settings) (Proposal, error) {
	st, err := s.standFrom(ctx, edit.Head)
	if err != nil {
		return Proposal{}, err
	}
	if edit.Head != st.head {
		return Proposal{}, fmt.Errorf("%w (read at %.12s, now %.12s)", ErrMoved, edit.Head, st.head)
	}
	content, err := lookup(st.tree, ExecutionPath)
	if err != nil {
		return Proposal{}, err
	}
	changed, err := editSettings(content, edit)
	if err != nil {
		return Proposal{}, err
	}
	tree, err := replace(st.tree, ExecutionPath, changed)
	if err != nil {
		return Proposal{}, err
	}
	return s.propose(st, "configuration", "Execution settings", tree, func(path string) bool { return path == ExecutionPath })
}

func lookup(tree snapshot.Tree, name string) ([]byte, error) {
	path, err := snapshot.NewPath(name)
	if err != nil {
		return nil, err
	}
	entry, ok := tree.Lookup(path)
	if !ok {
		return nil, ErrNoExecution
	}
	return entry.Bytes(), nil
}

func readSettings(content []byte) (Settings, error) {
	var doc struct {
		Budget   Budget   `yaml:"budget"`
		Profiles Profiles `yaml:"profiles"`
		Packages map[string]struct {
			Profiles Profiles `yaml:"profiles"`
		} `yaml:"packages"`
		Team *Team `yaml:"team"`
	}
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return Settings{}, fmt.Errorf("%s is not readable YAML: %w", ExecutionPath, err)
	}
	settings := Settings{Budget: doc.Budget, Profiles: doc.Profiles, Team: doc.Team, Text: string(content), Overrides: []PackageProfiles{}}
	for name, p := range doc.Packages {
		if p.Profiles.Execute != "" || p.Profiles.Verify != "" {
			settings.Overrides = append(settings.Overrides, PackageProfiles{Package: name, Execute: p.Profiles.Execute, Verify: p.Profiles.Verify})
		}
	}
	slices.SortFunc(settings.Overrides, func(a, b PackageProfiles) int {
		if a.Package < b.Package {
			return -1
		}
		if a.Package > b.Package {
			return 1
		}
		return 0
	})
	return settings, nil
}

// editSettings changes the fields whose value differs from the document's in
// its node tree, so unchanged fields keep their text, comments and style and
// every other field stays as it was. A changed team is written anew.
func editSettings(content []byte, edit Settings) ([]byte, error) {
	current, err := readSettings(content)
	if err != nil {
		return nil, err
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(content, &doc); err != nil {
		return nil, fmt.Errorf("%s is not readable YAML: %w", ExecutionPath, err)
	}
	if len(doc.Content) != 1 || doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("%s is not a mapping", ExecutionPath)
	}
	root := doc.Content[0]
	for _, f := range []struct{ section, key, before, after, tag string }{
		{"budget", "executionTime", current.Budget.ExecutionTime, edit.Budget.ExecutionTime, "!!str"},
		{"budget", "turnTimeout", current.Budget.TurnTimeout, edit.Budget.TurnTimeout, "!!str"},
		{"budget", "agentTurns", strconv.Itoa(current.Budget.AgentTurns), strconv.Itoa(edit.Budget.AgentTurns), "!!int"},
		{"budget", "passes", strconv.Itoa(current.Budget.Passes), strconv.Itoa(edit.Budget.Passes), "!!int"},
		{"profiles", "execute", current.Profiles.Execute, edit.Profiles.Execute, "!!str"},
		{"profiles", "verify", current.Profiles.Verify, edit.Profiles.Verify, "!!str"},
	} {
		if f.before != f.after {
			setScalar(child(root, f.section), f.key, f.after, f.tag)
		}
	}
	editOverrides(root, current.Overrides, edit.Overrides)
	switch {
	case sameTeam(current.Team, edit.Team):
	case edit.Team == nil:
		remove(root, "team")
	default:
		set(root, "team", teamNode(*edit.Team))
	}
	var out bytes.Buffer
	enc := yaml.NewEncoder(&out)
	enc.SetIndent(2)
	if err := enc.Encode(&doc); err != nil {
		return nil, err
	}
	if err := enc.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// sameTeam says whether two teams declare the same helpers.
func sameTeam(a, b *Team) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if a.MaxHelpers != b.MaxHelpers || len(a.Helpers) != len(b.Helpers) {
		return false
	}
	for i, x := range a.Helpers {
		y := b.Helpers[i]
		if x.Role != y.Role || x.Profile != y.Profile || x.Writes != y.Writes || !slices.Equal(x.Instructions, y.Instructions) {
			return false
		}
	}
	return true
}

// editOverrides sets the profile overrides of the listed Packages and removes
// those of every other Package; an entry whose overrides are unchanged is left
// as it is, and a Package entry left empty is removed.
func editOverrides(root *yaml.Node, current, overrides []PackageProfiles) {
	wanted := map[string]PackageProfiles{}
	for _, o := range overrides {
		if o.Package != "" && (o.Execute != "" || o.Verify != "") {
			wanted[o.Package] = o
		}
	}
	have := map[string]PackageProfiles{}
	for _, o := range current {
		have[o.Package] = o
	}
	packages := value(root, "packages")
	if packages == nil {
		if len(wanted) == 0 {
			return
		}
		packages = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		set(root, "packages", packages)
	}
	if packages.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i+1 < len(packages.Content); i += 2 {
		name, entry := packages.Content[i].Value, packages.Content[i+1]
		o := wanted[name]
		delete(wanted, name)
		if entry.Kind != yaml.MappingNode || o.Execute == have[name].Execute && o.Verify == have[name].Verify {
			continue
		}
		if o.Execute == "" && o.Verify == "" {
			remove(entry, "profiles")
		} else {
			setProfiles(entry, o)
		}
		if len(entry.Content) == 0 {
			packages.Content = append(packages.Content[:i], packages.Content[i+2:]...)
			i -= 2
		}
	}
	names := make([]string, 0, len(wanted))
	for name := range wanted {
		names = append(names, name)
	}
	slices.Sort(names)
	if len(names) > 0 {
		packages.Style = 0
	}
	for _, name := range names {
		entry := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
		setProfiles(entry, wanted[name])
		set(packages, name, entry)
	}
	if len(packages.Content) == 0 {
		packages.Style = yaml.FlowStyle
	}
}

// setProfiles sets one Package's profile overrides in place; a new profiles
// mapping is written in flow style.
func setProfiles(entry *yaml.Node, o PackageProfiles) {
	profiles := value(entry, "profiles")
	if profiles == nil || profiles.Kind != yaml.MappingNode {
		profiles = &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}
		set(entry, "profiles", profiles)
	}
	for _, role := range []struct{ key, value string }{{"execute", o.Execute}, {"verify", o.Verify}} {
		if role.value == "" {
			remove(profiles, role.key)
		} else {
			setScalar(profiles, role.key, role.value, "!!str")
		}
	}
}

func teamNode(t Team) *yaml.Node {
	team := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	setScalar(team, "maxHelpers", strconv.Itoa(t.MaxHelpers), "!!int")
	helpers := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq"}
	for _, h := range t.Helpers {
		node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map", Style: yaml.FlowStyle}
		setScalar(node, "role", h.Role, "!!str")
		if h.Profile != "" {
			setScalar(node, "profile", h.Profile, "!!str")
		}
		setScalar(node, "writes", strconv.FormatBool(h.Writes), "!!bool")
		if len(h.Instructions) > 0 {
			list := &yaml.Node{Kind: yaml.SequenceNode, Tag: "!!seq", Style: yaml.FlowStyle}
			for _, path := range h.Instructions {
				list.Content = append(list.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: path})
			}
			set(node, "instructions", list)
		}
		helpers.Content = append(helpers.Content, node)
	}
	set(team, "helpers", helpers)
	return team
}

// value returns the value node of a key in a mapping, or nil.
func value(mapping *yaml.Node, key string) *yaml.Node {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			return mapping.Content[i+1]
		}
	}
	return nil
}

// child returns the mapping under a key, creating it when absent.
func child(mapping *yaml.Node, key string) *yaml.Node {
	if v := value(mapping, key); v != nil && v.Kind == yaml.MappingNode {
		return v
	}
	node := &yaml.Node{Kind: yaml.MappingNode, Tag: "!!map"}
	set(mapping, key, node)
	return node
}

// set replaces the value of a key, keeping the key's comments, or appends it.
func set(mapping *yaml.Node, key string, node *yaml.Node) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			old := mapping.Content[i+1]
			node.LineComment, node.FootComment = old.LineComment, old.FootComment
			mapping.Content[i+1] = node
			return
		}
	}
	mapping.Content = append(mapping.Content, &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: key}, node)
}

// setScalar sets a scalar value in place, keeping its style and comments.
func setScalar(mapping *yaml.Node, key, text, tag string) {
	if v := value(mapping, key); v != nil && v.Kind == yaml.ScalarNode {
		v.Value, v.Tag = text, tag
		return
	}
	set(mapping, key, &yaml.Node{Kind: yaml.ScalarNode, Tag: tag, Value: text})
}

func remove(mapping *yaml.Node, key string) {
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		if mapping.Content[i].Value == key {
			mapping.Content = append(mapping.Content[:i], mapping.Content[i+2:]...)
			return
		}
	}
}
