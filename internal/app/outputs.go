package app

import (
	"bytes"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

func Generated(data []byte) bool {
	return render.IsGenerated(data)
}

func CheckOutputs(p *Project) []core.Diagnostic {
	if len(p.Diagnostics) > 0 {
		return nil
	}
	outputs, err := render.Generate(p.Graph, p.Snapshot.Files)
	if err != nil {
		return []core.Diagnostic{{Code: "render", Message: err.Error()}}
	}
	var findings []core.Diagnostic
	findings = append(findings, CheckConsistency(p)...)
	findings = append(findings, checkProviderAdapterInputs(p)...)
	names := sortedFiles(outputs)
	for _, name := range names {
		data, ok := p.Snapshot.Files[name]
		if !ok {
			findings = append(findings, core.Diagnostic{Code: "missing-output", Path: name, Message: "generated file is missing; run markitect render --write"})
		} else if !bytes.Equal(normalize(data), normalize(outputs[name])) {
			findings = append(findings, core.Diagnostic{Code: "output-drift", Path: name, Message: "generated file differs from its canonical source"})
		}
	}
	for _, name := range sortedFiles(p.Snapshot.Files) {
		if (strings.HasSuffix(name, ".md") || strings.HasSuffix(name, ".toml")) && Generated(p.Snapshot.Files[name]) {
			if _, ok := outputs[name]; !ok {
				findings = append(findings, core.Diagnostic{Code: "stale-output", Path: name, Message: "previously generated file has no current source; inspect and remove in the same migration"})
			}
		}
	}
	findings = append(findings, checkProviderInventory(p, outputs)...)
	return findings
}

func normalize(data []byte) []byte { return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n")) }
