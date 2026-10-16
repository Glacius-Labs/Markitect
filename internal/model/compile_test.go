package model

import (
	"crypto/sha256"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func sourceInput() Input {
	return Input{Root: SourceModel{
		Origin:       "local-snapshot",
		ManifestPath: "model.yaml",
		Entries: []Entry{
			{Path: "model.yaml", Kind: Regular, Bytes: []byte("formatVersion: 1\nid: example\nroot: model\ndependencies: {}\n")},
			// A self-address isolates binding from the still-unimplemented Core
			// target-category check; this is not a positive full-v1 fixture.
			{Path: "model/feature/item.yaml", Kind: Regular, Bytes: []byte("id: item\nkind: item\nbody: Binding prose.\nspec: {}\n")},
		},
	}}
}

func TestPartialCompilerNeverClaimsConformance(t *testing.T) {
	input := sourceInput()
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 || got.Coordinates != NotCompleted {
		t.Fatalf("unexpected partial result: %+v", got)
	}
	if got.Limits != DefaultLimits() || len(got.Models) != 1 || got.Models[0].ID != "example" {
		t.Fatalf("missing scope/budget report: %+v", got)
	}
	m := got.Models[0]
	if m.Status != NotCompleted || len(m.Canonical) != 0 || m.Digest != "" || len(m.Definitions) != 0 {
		t.Fatalf("partial scope exposed validated IR or canonical evidence: %+v", m)
	}
	if m.Phases[0].Status != Passed || m.Phases[1].Status != Passed {
		t.Fatalf("implemented acquisition did not complete: %+v", m.Phases)
	}
	for _, p := range m.Phases[2:] {
		if p.Name == "dependencies" || p.Name == "imports_and_kinds" {
			if p.Status != Passed {
				t.Fatalf("completed binding phase was not reported: %+v", p)
			}
			continue
		}
		if p.Status != NotCompleted {
			t.Fatalf("unimplemented phase claimed completion: %+v", p)
		}
	}
	if (Result{}).Status == Passed || (Value{}).Kind() != "" {
		t.Fatal("zero values claim success")
	}
}

func TestSourceDiagnosticsBindExactBytes(t *testing.T) {
	input := sourceInput()
	e := &input.Root.Entries[1]
	e.Bytes = []byte("id: item\nkind: Example\nbody: '\u3000\u0085'\nspec: {}\n")
	got := Compile(input)
	if got.Status != Failed || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != "definition.empty_body" {
		t.Fatalf("expected the pinned White_Space failure: %+v", got)
	}
	d := got.Diagnostics[0]
	if d.Source.Origin != input.Root.Origin || d.Source.Path != e.Path || d.Source.Digest != fmt.Sprintf("sha256:%x", sha256.Sum256(e.Bytes)) {
		t.Fatalf("unbound source diagnostic: %+v", d)
	}
	for _, m := range got.Models {
		if len(m.Canonical) != 0 || m.Digest != "" {
			t.Fatal("invalid model received canonical evidence")
		}
	}
}

func TestSourceViolations(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Input)
		code string
	}{
		{"duplicate path", func(i *Input) { i.Root.Entries = append(i.Root.Entries, i.Root.Entries[1]) }, "input.duplicate_path"},
		{"dot path", func(i *Input) { i.Root.Entries[1].Path = "model/./feature/item.yaml" }, "input.path"},
		{"backslash", func(i *Input) { i.Root.Entries[1].Path = `model\feature\item.yaml` }, "input.path"},
		{"nondirectory parent", func(i *Input) { i.Root.Entries = append(i.Root.Entries, Entry{Path: "model", Kind: Regular}) }, "input.nondirectory_parent"},
		{"redirect", func(i *Input) { i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/link", Kind: Redirect}) }, "source.redirect"},
		{"unknown entry kind", func(i *Input) { i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/socket", Kind: "socket"}) }, "input.entry_kind"},
		{"nonregular bytes", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/empty", Kind: Directory, Bytes: []byte("x")})
		}, "input.entry_bytes"},
		{"root definition", func(i *Input) { i.Root.Entries[1].Path = "model/item.yaml" }, "definition.root_package"},
		{"yaml uppercase", func(i *Input) { i.Root.Entries[1].Path = "model/feature/item.YAML" }, "definition.extension"},
		{"yml", func(i *Input) { i.Root.Entries[1].Path = "model/feature/item.yml" }, "definition.extension"},
		{"manifest missing", func(i *Input) { i.Root.Entries = i.Root.Entries[1:] }, "manifest.missing"},
		{"manifest selection", func(i *Input) { i.Root.ManifestPath = "other.yaml" }, "input.manifest_path"},
		{"empty body", func(i *Input) { i.Root.Entries[1].Bytes = []byte("id: item\nkind: Example\nbody: ''\nspec: {}") }, "field.nonempty_string"},
		{"closed envelope", func(i *Input) { i.Root.Entries[1].Bytes = append(i.Root.Entries[1].Bytes, []byte("extra: value\n")...) }, "field.unknown"},
		{"invalid import", func(i *Input) {
			i.Root.Entries[1].Bytes = append(i.Root.Entries[1].Bytes, []byte("imports: [foreign]\n")...)
		}, "import.object"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := sourceInput()
			tt.edit(&input)
			got := Compile(input)
			if got.Status != Failed || !slices.ContainsFunc(got.Diagnostics, func(d Diagnostic) bool { return d.Code == tt.code }) {
				t.Fatalf("expected %s: %+v", tt.code, got)
			}
		})
	}
}

func TestManifestShape(t *testing.T) {
	for _, tt := range []struct{ yaml, code string }{
		{"formatVersion: '1'\nid: example\nroot: model\ndependencies: {}", "manifest.format"},
		{"formatVersion: 1\nid: 42\nroot: model\ndependencies: {}", "field.nonempty_string"},
		{"formatVersion: 1\nid: example\nroot: ./model\ndependencies: {}", "manifest.root_path"},
		{"formatVersion: 1\nid: example\nroot: absent\ndependencies: {}", "manifest.root_missing"},
		{"formatVersion: 1\nid: example\nroot: model\ndependencies: []", "manifest.dependencies"},
		{"formatVersion: 1\nid: example\nroot: model\ndependencies: {vocabulary: 1}", "field.nonempty_string"},
		{"formatVersion: 1\nid: example\nroot: model\ndependencies: {}\nversion: 1", "field.nonempty_string"},
	} {
		input := sourceInput()
		input.Root.Entries[0].Bytes = []byte(tt.yaml)
		got := Compile(input)
		if got.Status != Failed || !slices.ContainsFunc(got.Diagnostics, func(d Diagnostic) bool { return d.Code == tt.code }) {
			t.Fatalf("expected %s for %s: %+v", tt.code, tt.yaml, got)
		}
	}
}

func TestOnlyExplicitMetadataPathsAreExempt(t *testing.T) {
	input := sourceInput()
	input.Root.Entries = append(input.Root.Entries,
		Entry{Path: "model/feature/model.yaml", Kind: Regular, Bytes: []byte("formatVersion: 1")},
		Entry{Path: "model/feature/model-lock.yaml", Kind: Regular, Bytes: []byte("models: {}")},
		Entry{Path: "ignored/outside.yaml", Kind: Regular, Bytes: []byte("not: [valid")},
		Entry{Path: "model/feature/notes.md", Kind: Regular, Bytes: []byte("not: [valid")},
	)
	got := Compile(input)
	if got.Status != Failed {
		t.Fatalf("non-Definitions were treated as metadata: %+v", got)
	}
	for _, d := range got.Diagnostics {
		if strings.Contains(d.Source.Path, "outside") || strings.HasSuffix(d.Source.Path, ".md") {
			t.Fatalf("non-model content parsed: %+v", d)
		}
	}
}

func TestRootDotAndEmptyDirectories(t *testing.T) {
	input := sourceInput()
	input.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: example\nroot: .\ndependencies: {}")
	input.Root.Entries = append(input.Root.Entries, Entry{Path: "empty", Kind: Directory})
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("selected metadata must not create a root Package: %+v", got)
	}
	input.Root.Entries = []Entry{
		{Path: "model.yaml", Kind: Regular, Bytes: []byte("formatVersion: 1\nid: example\nroot: model\ndependencies: {}")},
		{Path: "model", Kind: Directory},
	}
	if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("explicit empty root rejected: %+v", got)
	}
}

func TestBudgetsDoNotBecomeLanguageFailures(t *testing.T) {
	for _, limits := range []Limits{{SourceBytes: 1}, {Entries: 1}, {YAMLDepth: 1}, {Entries: -1}, {Models: -1}} {
		input := sourceInput()
		input.Limits = limits
		got := Compile(input)
		if got.Status != NotCompleted || len(got.Diagnostics) == 0 {
			t.Fatalf("resource/config refusal misclassified: %+v", got)
		}
		for _, m := range got.Models {
			if m.Digest != "" || len(m.Canonical) != 0 {
				t.Fatal("resource-limited scope received canonical output")
			}
		}
	}
}

func TestModelCountIsBoundedBeforeCompilingEmptySources(t *testing.T) {
	input := sourceInput()
	input.Limits.Models = 2
	input.Dependencies = []SourceModel{{}, {}}
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Models) != 0 || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != "limit.models" || got.Limits.Models != 2 {
		t.Fatalf("model admission exceeded its reported budget: %+v", got)
	}
}

func TestPhasesKeepSourceErrorsSeparateFromYAMLErrors(t *testing.T) {
	input := sourceInput()
	input.Root.Entries = append(input.Root.Entries, Entry{Path: "model/feature/link", Kind: Redirect})
	got := Compile(input)
	if got.Status != Failed || got.Models[0].Phases[0].Status != Failed || got.Models[0].Phases[1].Status == Failed {
		t.Fatalf("redirect was attributed to YAML rather than source structure: %+v", got)
	}
	input = sourceInput()
	input.Root.Entries[1].Bytes = append(input.Root.Entries[1].Bytes, []byte("unknown: text\n")...)
	got = Compile(input)
	if got.Status != Failed || got.Models[0].Phases[0].Status != Passed || got.Models[0].Phases[1].Status != Failed {
		t.Fatalf("envelope error was attributed to source structure: %+v", got)
	}
}

func TestKnownErrorSurvivesLaterDepthRefusal(t *testing.T) {
	input := sourceInput()
	input.Limits.YAMLDepth = 3
	input.Root.Entries = append(input.Root.Entries,
		Entry{Path: "model/feature/a.yaml", Kind: Regular, Bytes: []byte("id: item\nkind: Example\nbody: text\nspec: {}\nextra: value")},
		Entry{Path: "model/feature/z.yaml", Kind: Regular, Bytes: []byte("id: deep\nkind: Example\nbody: text\nspec: {field: {nested: {deeper: x}}}")},
	)
	got := Compile(input)
	if got.Status != Failed || !slices.ContainsFunc(got.Diagnostics, func(d Diagnostic) bool { return strings.HasPrefix(d.Code, "limit.") }) || !slices.ContainsFunc(got.Diagnostics, func(d Diagnostic) bool { return d.Code == "field.unknown" }) {
		t.Fatalf("failed and incomplete evidence not retained together: %+v", got)
	}
}

func TestInputOrderingDoesNotChangeDiagnostics(t *testing.T) {
	input := sourceInput()
	input.Root.Entries = append(input.Root.Entries,
		Entry{Path: "model/feature/z.yaml", Kind: Regular, Bytes: []byte("id: z\nkind: Example\nbody: text\nspec: {}\nunknown: x")},
		Entry{Path: "model/feature/a.yaml", Kind: Regular, Bytes: []byte("id: a\nkind: Example\nbody: text\nspec: {}\nunknown: y")},
		Entry{Path: "model/feature/item.yaml", Kind: Regular, Bytes: []byte("id: duplicate\nkind: Example\nbody: text\nspec: {}")},
	)
	first := Compile(input)
	slices.Reverse(input.Root.Entries)
	second := Compile(input)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("source enumeration changed the report:\n%+v\n%+v", first, second)
	}
}

func TestTypedValuesDoNotCoerceOrExposeMutableCollections(t *testing.T) {
	v := Value{kind: ObjectValue, fields: []Field{{Name: "count", Value: Value{kind: IntegerValue, text: "999999999999999999999999999999999999"}}}}
	fields, ok := v.Fields()
	if !ok {
		t.Fatal("object inaccessible")
	}
	count, ok := fields[0].Value.Integer()
	if !ok || count != "999999999999999999999999999999999999" {
		t.Fatal("integer lost precision")
	}
	if _, ok := fields[0].Value.Text(); ok {
		t.Fatal("integer coerced to text")
	}
	fields[0].Name = "changed"
	unchanged, _ := v.Fields()
	if unchanged[0].Name != "count" {
		t.Fatal("caller mutated object through accessor")
	}
	list := Value{kind: ListValue, items: []Value{{kind: BooleanValue, boolean: true}}}
	items, _ := list.Items()
	items[0] = Value{}
	unchangedItems, _ := list.Items()
	if b, ok := unchangedItems[0].Boolean(); !ok || !b {
		t.Fatal("caller mutated list through accessor")
	}
	ref := Value{kind: ReferenceValue, reference: Identity{Package: "feature", ID: "item"}}
	if id, ok := ref.Reference(); !ok || id != (Identity{Package: "feature", ID: "item"}) {
		t.Fatal("reference identity was not retained")
	}
	number := Value{kind: NumberValue, decimal: Decimal{Coefficient: "1", Exponent: "9999999999999999999999"}}
	if d, ok := number.Number(); !ok || d.Exponent != "9999999999999999999999" {
		t.Fatal("decimal exponent lost precision")
	}
}

func TestDecodedTextRetainsNULAndUnicodeWithoutNormalization(t *testing.T) {
	v, problems, completed := parseYAML([]byte("body: \"[[api|A\\0B]] u\\u0308 <>&\\u2028\"\r\ntitle: ''\r\n"), 8)
	if !completed || len(problems) != 0 {
		t.Fatalf("text decoding failed: %+v", problems)
	}
	body, _ := lookup(v, "body")
	if text, ok := body.Text(); !ok || text != "[[api|A\x00B]] u\u0308 <>&\u2028" {
		t.Fatalf("decoded text was normalized or replaced: %q", text)
	}
	title, _ := lookup(v, "title")
	if text, ok := title.Text(); !ok || text != "" {
		t.Fatalf("empty quoted string was changed: %q", text)
	}
}
