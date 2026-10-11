package projectonboarding

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/host/codexappserver"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectsetup"
)

// RuntimePins are the runtime versions and the standard profile that the
// generated guidance names. Values the runtime binds come from the packages
// that bind them, so the guidance cannot drift from what Markitect runs.
type RuntimePins struct {
	CodexCLI   string `json:"codexCli"`
	Model      string `json:"model"`
	Effort     string `json:"effort"`
	ClaudeCode string `json:"claudeCode"`
}

const (
	// standardModel is the standard runtime profile's model, as in
	// `config --provider codex --model gpt-6-luna`.
	standardModel = "gpt-6-luna"
	// claudeCodeVersion is the outer Claude Code version the guidance is
	// written for. Claude Code is never an inner worker, so no runtime binds it.
	claudeCodeVersion = "2.1.295"
)

// Pins returns the one source of the runtime pins in generated guidance.
func Pins() RuntimePins {
	return RuntimePins{
		CodexCLI:   strings.TrimPrefix(codexappserver.SupportedProviderVersion, "codex-cli "),
		Model:      standardModel,
		Effort:     projectsetup.DefaultCodexEffort,
		ClaudeCode: claudeCodeVersion,
	}
}

// innerRuntime names the default inner Manager and reviewer runtime.
func (p RuntimePins) innerRuntime() string {
	return fmt.Sprintf("Codex CLI %s App Server using `%s` at `%s`", p.CodexCLI, p.Model, p.Effort)
}
