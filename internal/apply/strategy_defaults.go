package apply

import (
	"crypto/sha256"
	_ "embed"
	"slices"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/organization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

const strategyDefaultsSource = "product:apply/defaults.yaml"

//go:embed strategy_defaults.yaml
var strategyDefaultsBytes []byte

// StrategyDefaultsInput supplies an actual product document, never an implicit
// set of fallback literals. Nil Bytes means unavailable; nonnil empty bytes are
// an observed invalid document. Source is an opaque logical product locator.
type StrategyDefaultsInput struct {
	Source string
	Bytes  []byte
}

// ProductStrategyDefaults returns owned original bytes from the versioned
// product document embedded in this build. No repository or host I/O occurs.
func ProductStrategyDefaults() StrategyDefaultsInput {
	return StrategyDefaultsInput{Source: strategyDefaultsSource, Bytes: slices.Clone(strategyDefaultsBytes)}
}

type StrategyDefaultsDiagnostic struct {
	Code     string
	Source   organization.SettingSource
	Position *model.Position
	Message  string
}

type strategyDefaults struct {
	status      model.Status
	input       StrategyDefaultsInput
	source      organization.SettingSource
	execute     organization.Setting[[]string]
	verify      organization.Setting[[]string]
	team        ExecutionTeam
	diagnostics []StrategyDefaultsDiagnostic
}

func (r *strategyDefaults) issue(status model.Status, code string, field configField, message string) {
	source := r.source
	source.Field = field.pointer
	var position *model.Position
	if field.node != nil && field.node.Line > 0 {
		position = &model.Position{Line: field.node.Line, Column: field.node.Column}
	}
	r.diagnostics = append(r.diagnostics, StrategyDefaultsDiagnostic{Code: code, Source: source, Position: position, Message: message})
	if status == model.Failed || r.status != model.Failed {
		r.status = status
	}
}

func newStrategyDefaults(input StrategyDefaultsInput) strategyDefaults {
	input.Bytes = slices.Clone(input.Bytes)
	r := strategyDefaults{status: model.Passed, input: input, source: organization.SettingSource{File: input.Source}}
	if input.Bytes != nil {
		r.source.ContentID = snapshot.ContentID(sha256.Sum256(input.Bytes))
	}
	return r
}
