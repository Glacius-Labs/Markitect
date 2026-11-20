package cli

import (
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/state"
)

// settingsDocument is the JSON document of settings: whether the project has
// a settings file, what check finds in it, and its values. A value that is
// not of its form is left out.
type settingsDocument struct {
	header
	Present       bool              `json:"present"`
	Errors        []diagnostic      `json:"errors"`
	Models        map[string]string `json:"models"`
	CounterChecks string            `json:"counter_checks"`
	Tests         string            `json:"tests"`
	Cadence       map[string]string `json:"cadence"`
	WriteTable    []writeRuleJSON   `json:"write_table"`
	CI            []string          `json:"ci"`
}

type writeRuleJSON struct {
	Role    string `json:"role"`
	Pattern string `json:"pattern"`
}

// settings prints the settings of the project, so that a runner reads them
// without reading Markdown. It exits 1 while check finds errors in them.
func settings(o options, stdout io.Writer) (int, error) {
	w, err := load(o)
	if err != nil {
		return ExitFailed, err
	}
	f, err := factoryOf(w)
	if err != nil {
		return ExitFailed, err
	}
	s := f.Settings
	doc := settingsDocument{header: newHeader("settings"), Present: s != nil, Errors: []diagnostic{},
		Models: map[string]string{}, Cadence: map[string]string{}, WriteTable: []writeRuleJSON{}, CI: state.DefaultCI}
	if s != nil {
		doc.Models, doc.Cadence, doc.CI = s.Models, s.Cadence, nonNil(s.CI)
		doc.CounterChecks, doc.Tests = s.CounterChecks, s.Tests
		for _, r := range s.WriteTable {
			doc.WriteTable = append(doc.WriteTable, writeRuleJSON{r.Role, r.Pattern})
		}
	}
	var b strings.Builder
	for _, d := range f.Diagnostics {
		if d.Path == state.SettingsFile {
			doc.Errors = append(doc.Errors, diagnostic{File: d.Path, Line: d.Line, Code: d.Code, Message: d.Message})
			b.WriteString(d.String() + "\n")
		}
	}
	if s == nil {
		b.WriteString("no settings: " + state.SettingsFile + " does not exist\n")
	}
	pairs := func(title string, m map[string]string) {
		b.WriteString(title + ":")
		if len(m) == 0 {
			b.WriteString(" (none)\n")
			return
		}
		b.WriteString("\n")
		keys := make([]string, 0, len(m))
		for k := range m {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s: %s\n", k, m[k])
		}
	}
	or := func(v string) string {
		if v == "" {
			return "(none)"
		}
		return v
	}
	pairs("Models", doc.Models)
	fmt.Fprintf(&b, "Counter-checks: %s\nTests: %s\n", or(doc.CounterChecks), or(doc.Tests))
	pairs("Cadence", doc.Cadence)
	b.WriteString("Write table:")
	if len(doc.WriteTable) == 0 {
		b.WriteString(" (none)")
	}
	b.WriteString("\n")
	for _, r := range doc.WriteTable {
		fmt.Fprintf(&b, "  %s: %s\n", r.Role, r.Pattern)
	}
	fmt.Fprintf(&b, "CI: %s\n", or(strings.Join(doc.CI, ", ")))
	code, err := output(o, stdout, doc, b.String())
	if err == nil && len(doc.Errors) > 0 {
		code = ExitFound
	}
	return code, err
}
