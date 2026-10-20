package apply

import (
	"bytes"
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"go.yaml.in/yaml/v3"
)

const executionPath = ".markitect/execution.yaml"

func decodeExecution(checked model.Result, raw RawInputs) executionDecoding {
	r := executionDecoding{status: model.Passed}
	d := executionDecoder{result: &r}
	if checked.Status != model.Passed {
		d.issue(model.NotCompleted, "execution.model_not_passed", executionField{}, "Execution decoding requires a passed bound model stage.")
		return r
	}
	if len(raw.Execution) == 0 {
		d.issue(model.NotCompleted, "execution.source_uncaptured", executionField{}, "The project execution source has not been captured.")
		return r
	}
	if len(raw.Execution) != 1 {
		d.issue(model.Failed, "execution.source_count", executionField{}, "Exactly one project execution source is permitted.")
		return r
	}
	d.source = raw.Execution[0]
	if d.source.RepositoryPath.String() != executionPath {
		d.issue(model.Failed, "execution.source_path", executionField{}, "Project execution must be bound to .markitect/execution.yaml.")
		return r
	}
	if !sameCensusFile(raw.Census, executionPath, d.source.File) {
		d.issue(model.NotCompleted, "execution.source_mismatch", executionField{}, "Execution source is not bound to the actual Census bytes and mode.")
		return r
	}
	var err error
	d.topology, err = organization.BuildTopology(checked)
	if err != nil {
		d.issue(model.Failed, "execution.model_invalid", executionField{}, err.Error())
		return r
	}
	d.definitions = make(map[model.Identity]bool)
	for _, result := range checked.Models {
		for _, definition := range result.Definitions {
			d.definitions[definition.Identity] = true
		}
	}
	decoder := yaml.NewDecoder(bytes.NewReader(d.source.File.Bytes()))
	var document, extra yaml.Node
	if err := decoder.Decode(&document); err != nil {
		d.issue(model.Failed, "execution.yaml", executionField{}, err.Error())
		return r
	}
	if err := decoder.Decode(&extra); err != io.EOF {
		d.issue(model.Failed, "execution.documents", executionField{node: &extra}, "Execution must contain exactly one YAML document.")
	}
	if len(document.Content) != 1 {
		d.issue(model.Failed, "execution.document", executionField{}, "Execution requires a mapping document.")
		return r
	}
	root := executionField{node: document.Content[0]}
	d.restricted(root)
	fields := d.object(root, "apiVersion", "strategy", "budget", "profiles", "instructions", "packages", "layout", "scope", "checks")
	r.authored.APIVersion, _ = d.text(d.required(fields, root, "apiVersion"), false)
	if r.authored.APIVersion.Source.File != "" && r.authored.APIVersion.Value != "markitect.execution/v1" {
		d.issue(model.Failed, "execution.version", fields["apiVersion"], "Unsupported execution apiVersion.")
	}
	r.authored.Strategy, _ = d.text(d.required(fields, root, "strategy"), false)
	if r.authored.Strategy.Source.File != "" && r.authored.Strategy.Value != "shared-workspace-v1" {
		d.issue(model.Failed, "execution.strategy", fields["strategy"], "Unsupported execution strategy.")
	}
	r.authored.Budget = d.budget(d.required(fields, root, "budget"))
	r.authored.Defaults.Execute, r.authored.Defaults.Verify = d.roles(fields, root, true)
	if field, found := fields["packages"]; found {
		r.authored.Packages = d.packages(field)
	}
	if field, found := fields["layout"]; found {
		r.authored.Layout = d.layout(field, checked)
	}
	if field, found := fields["scope"]; found {
		r.authored.Scope = d.scope(field)
	}
	if field, found := fields["checks"]; found {
		r.authored.Checks = d.checks(field)
	}
	slices.SortFunc(r.diagnostics, func(a, b ExecutionDiagnostic) int {
		if n := strings.Compare(a.Source.Field, b.Source.Field); n != 0 {
			return n
		}
		if n := strings.Compare(a.Code, b.Code); n != 0 {
			return n
		}
		return strings.Compare(a.Message, b.Message)
	})
	return r
}

func (d *executionDecoder) localPackage(field executionField, name string) {
	if _, found := d.topology.Manager(name); !found {
		d.issue(model.Failed, "execution.package", field, fmt.Sprintf("Unknown or imported Package %q.", name))
	}
}

func (d *executionDecoder) identity(field executionField) (organization.Setting[model.Identity], bool) {
	text, ok := d.text(field, false)
	if !ok {
		return organization.Setting[model.Identity]{}, false
	}
	i := strings.LastIndexByte(text.Value, '/')
	if i <= 0 || i == len(text.Value)-1 {
		d.issue(model.Failed, "execution.definition_address", field, "Expected an exact full Package/ID address.")
		return organization.Setting[model.Identity]{}, false
	}
	identity := model.Identity{Package: text.Value[:i], ID: text.Value[i+1:]}
	if !d.definitions[identity] {
		d.issue(model.Failed, "execution.definition_missing", field, "Definition is absent from the complete checked model set.")
		return organization.Setting[model.Identity]{}, false
	}
	return organization.Setting[model.Identity]{Value: identity, Source: text.Source}, true
}
