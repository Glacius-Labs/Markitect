package apply

import (
	"bytes"
	"io"
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
	f := d.object(root, "apiVersion", "strategy", "defaults", "acceptance")
	version := d.required(f, root, "apiVersion")
	strategy := d.required(f, root, "strategy")
	versionName, versionOK := d.text(version, false)
	strategyName, strategyOK := d.text(strategy, false)
	if versionOK && versionName != "markitect.strategy-defaults/v1" {
		r.issue(model.NotCompleted, "strategy_defaults.profile_unknown", version, "This product defaults version is not supported.")
		return r
	}
	if strategyOK && strategyName != "recursive-apply-v1" {
		r.issue(model.NotCompleted, "strategy_defaults.profile_unknown", strategy, "This product defaults strategy is not supported.")
		return r
	}
	defaults := d.required(f, root, "defaults")
	fields := d.object(defaults, "reviewRounds", "instructions", "batchNoChangeReviews")
	// Reuse existing positive-integer/list readers on the actual product bytes.
	// Their temporary relative file coordinate is replaced by the product locator;
	// content identity, field pointers and original positions remain unchanged.
	asset, _ := snapshot.NewPath("strategy_defaults.yaml")
	entry, _ := snapshot.NewEntry(asset, snapshot.Regular, r.input.Bytes)
	typed := executionDecoding{status: model.Passed}
	reader := executionDecoder{result: &typed, source: SourceFile{File: entry, RepositoryPath: asset}}
	if value, ok := reader.count(d.required(fields, defaults, "reviewRounds")); ok {
		value.Source.File = input.Source
		r.rounds = value
	}
	instructions := d.required(fields, defaults, "instructions")
	manager, reviewer := reader.roles(map[string]executionField{"instructions": instructions}, defaults, false)
	for _, role := range []struct {
		name   string
		value  *organization.Setting[[]string]
		target *organization.Setting[[]string]
	}{{"manager", manager.Instructions, &r.manager}, {"reviewer", reviewer.Instructions, &r.reviewer}} {
		if role.value == nil {
			r.issue(model.Failed, "strategy_defaults.role_instructions", configChild(instructions, role.name, nil), "Product defaults require a valid explicit instruction list for each role.")
		} else {
			value := *role.value
			value.Source.File = input.Source
			*role.target = value
		}
	}
	batch := d.required(fields, defaults, "batchNoChangeReviews")
	if batch.node != nil {
		if batch.node.Kind != yaml.ScalarNode || batch.node.Tag != "!!bool" {
			r.issue(model.Failed, "strategy_defaults.boolean", batch, "Expected a literal boolean.")
		} else {
			var value bool
			if err := batch.node.Decode(&value); err != nil {
				r.issue(model.Failed, "strategy_defaults.boolean", batch, err.Error())
			} else {
				source := r.source
				source.Field = batch.pointer
				r.batch = organization.Setting[bool]{Value: value, Source: source}
			}
		}
	}
	acceptance := d.required(f, root, "acceptance")
	accepted := d.object(acceptance, "ownerApproval")
	if value, ok := reader.strings(d.required(accepted, acceptance, "ownerApproval")); ok {
		value.Source.File = input.Source
		r.approval = value
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
