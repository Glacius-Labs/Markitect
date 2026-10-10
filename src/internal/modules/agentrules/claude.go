package agentrules

import "strings"

func renderClaude(guidance, facts string) []byte {
	var b strings.Builder
	b.WriteString("# CLAUDE.md\n\n")
	b.WriteString("This file is a bounded projection of selected Markitect canonical facts. The supplied Schema, Kind, and Property purposes, Definition purposes and validated values, and ProjectionPolicy provide its full meaning; the identified canonical inputs remain authoritative if this file differs. Do not infer project rules from Kind names, types, or cardinality alone, and do not add unstated requirements.\n\n")
	b.WriteString("## Project-owned projection guidance\n\n")
	b.WriteString(guidance)
	b.WriteString("\n\n## Canonical facts and exact source identities\n\n")
	writeCodeBlock(&b, "json", facts)
	return []byte(b.String())
}
