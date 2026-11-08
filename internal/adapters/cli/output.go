package cli

import (
	"bytes"
	"encoding/json"

	"github.com/Glacius-Labs/Markitect/internal/model"
)

// FormatVersion is the version of every JSON document markitect prints with
// --json. A program reads format_version and kind, and so depends on the format,
// not on the wording of the text output (DEC-099).
const FormatVersion = 1

// envelope starts every JSON document.
type envelope struct {
	FormatVersion int    `json:"format_version"`
	Kind          string `json:"kind"`
}

func newEnvelope(kind string) envelope { return envelope{FormatVersion: FormatVersion, Kind: kind} }

// Severities of a diagnostic.
const (
	severityError   = "error"
	severityWarning = "warning"
	severityNote    = "note"
)

// diagnostic is one diagnostic in a JSON document. In text it is one line,
// file:line: code: message, as Go's tools print them.
type diagnostic struct {
	Severity string `json:"severity"`
	Code     string `json:"code"`
	Message  string `json:"message"`
	File     string `json:"file,omitempty"`
	Line     int    `json:"line,omitempty"`
	// Targets and Structure belong to vet.unlinked_mention: the Definitions named
	// without a link, and "against" or "new_dependency" when a link would close a
	// Package cycle or add a Package dependency.
	Targets   []string `json:"targets,omitempty"`
	Structure string   `json:"structure,omitempty"`
}

// diagnostics converts a list for a JSON document; never nil, so that the
// document holds an empty list rather than null.
func diagnostics(severity string, ds model.Diagnostics) []diagnostic {
	out := []diagnostic{}
	for _, d := range ds {
		out = append(out, diagnostic{Severity: severity, Code: d.Code, Message: d.Message, File: d.File, Line: d.Line})
	}
	return out
}

// printJSON prints a document, indented, without escaping <, > and &.
func (c CLI) printJSON(v any) error {
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	_, err := c.Stdout.Write(b.Bytes())
	return err
}
