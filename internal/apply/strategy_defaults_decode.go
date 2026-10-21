package apply

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

func decodeStrategyDefaults(input StrategyDefaultsInput) strategyDefaults {
	r := newStrategyDefaults(input)
	if input.Bytes == nil {
		r.issue(model.NotCompleted, "strategy_defaults.unavailable", configField{}, "The actual product defaults document is unavailable.")
		return r
	}
	if input.Source != strategyDefaultsSource {
		r.issue(model.NotCompleted, "strategy_defaults.source_unknown", configField{}, "This product defaults source is not supported.")
		return r
	}
	d := configYAML{name: "strategy defaults", title: "Strategy defaults", issue: func(code string, field configField, message string) {
		r.issue(model.Failed, "strategy_defaults."+code, field, message)
	}}
	decoder := yaml.NewDecoder(bytes.NewReader(r.input.Bytes))
	var document, extra yaml.Node
	if err := decoder.Decode(&document); err != nil {
		r.issue(model.Failed, "strategy_defaults.yaml", configField{}, err.Error())
		return r
	}
	if err := decoder.Decode(&extra); err != io.EOF {
		r.issue(model.Failed, "strategy_defaults.documents", configField{node: &extra}, "Product defaults require exactly one document.")
	}
	if len(document.Content) != 1 {
		r.issue(model.Failed, "strategy_defaults.document", configField{}, "Product defaults require a mapping document.")
		return r
	}
	root := configField{node: document.Content[0]}
	d.restricted(root)
	f := d.object(root, "apiVersion", "strategies", "defaults")
	version := d.required(f, root, "apiVersion")
	served := d.required(f, root, "strategies")
	versionName, versionOK := d.text(version, false)
	if versionOK && versionName != "markitect.strategy-defaults/v1" {
		r.issue(model.NotCompleted, "strategy_defaults.profile_unknown", version, "This product defaults version is not supported.")
		return r
	}
	// The product defaults serve every strategy this build implements and no
	// other.
	if names, ok := servedStrategies(d, served); ok && !slices.Equal(names, strategies) {
		r.issue(model.NotCompleted, "strategy_defaults.profile_unknown", served, "These product defaults do not serve exactly the strategies of this build: direct and staged.")
		return r
	}
	defaults := d.required(f, root, "defaults")
	if defaults.node == nil {
		return r
	}
	fields := d.object(defaults, "instructions", "team")
	// Reuse the existing list reader on the actual product bytes. Its temporary
	// relative file coordinate is replaced by the product locator; content
	// identity, field pointers and original positions remain unchanged.
	asset, _ := snapshot.NewPath("strategy_defaults.yaml")
	entry, _ := snapshot.NewEntry(asset, snapshot.Regular, r.input.Bytes)
	typed := executionDecoding{status: model.Passed}
	reader := executionDecoder{result: &typed, source: SourceFile{File: entry, RepositoryPath: asset}}
	instructions := d.required(fields, defaults, "instructions")
	execute, verify := reader.roles(map[string]executionField{"instructions": instructions}, defaults, false)
	for _, role := range []struct {
		name   string
		value  *organization.Setting[[]string]
		target *organization.Setting[[]string]
	}{{"execute", execute.Instructions, &r.execute}, {"verify", verify.Instructions, &r.verify}} {
		if role.value == nil {
			r.issue(model.Failed, "strategy_defaults.role_instructions", configChild(instructions, role.name, nil), "Product defaults require a valid explicit instruction list for each role.")
		} else {
			value := *role.value
			value.Source.File = input.Source
			*role.target = value
		}
	}
	if team := reader.team(d.required(fields, defaults, "team")); team != nil {
		r.team = *productSources(team, input.Source)
	}
	for _, diagnostic := range typed.diagnostics {
		source := diagnostic.Source
		source.File = input.Source
		r.diagnostics = append(r.diagnostics, StrategyDefaultsDiagnostic{Code: "strategy_defaults." + strings.TrimPrefix(diagnostic.Code, "execution."), Source: source, Position: diagnostic.Position, Message: diagnostic.Message})
	}
	if typed.Status() == model.Failed || r.status != model.Failed && typed.Status() == model.NotCompleted {
		r.status = typed.Status()
	}
	return r
}

// servedStrategies reads the sorted list of strategies the product defaults
// serve.
func servedStrategies(d configYAML, field configField) ([]Strategy, bool) {
	if field.node == nil {
		return nil, false
	}
	if field.node.Kind != yaml.SequenceNode {
		d.issue("list", field, "Expected an explicit list.")
		return nil, false
	}
	var names []Strategy
	for i, node := range field.node.Content {
		name, ok := d.text(configChild(field, fmt.Sprint(i), node), false)
		if !ok {
			return nil, false
		}
		names = append(names, Strategy(name))
	}
	slices.Sort(names)
	return names, true
}

// productSources attributes a team decoded from the product document to the
// product locator.
func productSources(team *ExecutionTeam, source string) *ExecutionTeam {
	team.MaxHelpers.Source.File = source
	for i := range team.Helpers {
		h := &team.Helpers[i]
		h.Role.Source.File, h.Writes.Source.File = source, source
		if h.Profile != nil {
			h.Profile.Source.File = source
		}
		if h.Instructions != nil {
			h.Instructions.Source.File = source
		}
	}
	return team
}
