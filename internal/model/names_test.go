package model

import (
	"bytes"
	"reflect"
	"slices"
	"testing"
)

func hasDiagnostic(result Result, code string) bool {
	return slices.ContainsFunc(result.Diagnostics, func(d Diagnostic) bool { return d.Code == code })
}

func TestUnicodeSourceNamesKeepTextAndPartialBoundary(t *testing.T) {
	input := sourceInput()
	input.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: ٢٠٢٦.١\nversion: '1.x'\nroot: model\ndependencies: {}")
	input.Root.Entries[1] = Entry{Path: "model/überweisung/文档/größe-٢.yaml", Kind: Regular,
		Bytes: []byte("id: größe-٢\nkind: 文档\nbody: \" u\\u0308 [[still-unchecked]] A\\0B \"\ntitle: \"u\\u0308\"\nspec: {文档: \"u\\u0308\"}\nimports: [{package: 文档, as: 文档}]")}
	original := bytes.Clone(input.Root.Entries[1].Bytes)
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("valid Unicode source spellings rejected: %+v", got)
	}
	if !bytes.Equal(original, input.Root.Entries[1].Bytes) {
		t.Fatal("compiler rewrote author bytes")
	}
	for _, model := range got.Models {
		if model.Status != NotCompleted || len(model.Definitions) != 0 || len(model.Canonical) != 0 || model.Digest != "" {
			t.Fatalf("name checks claimed a completed model: %+v", model)
		}
	}
	v, problems, completed := parseYAML(original, 20)
	if !completed || len(problems) != 0 {
		t.Fatalf("valid Unicode mapping failed: %+v", problems)
	}
	body, _ := lookup(v, "body")
	title, _ := lookup(v, "title")
	spec, _ := lookup(v, "spec")
	text, _ := lookup(spec, "文档")
	if body.text != " u\u0308 [[still-unchecked]] A\x00B " || title.text != "u\u0308" || text.text != "u\u0308" {
		t.Fatal("ordinary text was normalized or rewritten by name checks")
	}
}

func TestInvalidLexicalNames(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Input)
		code string
	}{
		{"decomposed ID", func(i *Input) {
			i.Root.Entries[1].Bytes = []byte("id: u\u0308ber\nkind: Example\nbody: text\nspec: {}")
		}, "name.definition"},
		{"dotted ID", func(i *Input) { i.Root.Entries[1].Bytes = []byte("id: item.part\nkind: Example\nbody: text\nspec: {}") }, "name.definition"},
		{"underscore ID", func(i *Input) { i.Root.Entries[1].Bytes = []byte("id: item_part\nkind: Example\nbody: text\nspec: {}") }, "name.definition"},
		{"emoji ID", func(i *Input) { i.Root.Entries[1].Bytes = []byte("id: 😀\nkind: Example\nbody: text\nspec: {}") }, "name.definition"},
		{"wrong alias shape", func(i *Input) {
			i.Root.Entries[1].Bytes = append(i.Root.Entries[1].Bytes, []byte("imports: [{package: valid, as: wrongAlias}]")...)
		}, "name.alias"},
		{"invalid reference", func(i *Input) {
			i.Root.Entries[1].Bytes = []byte("id: item\nkind: Core.part.more\nbody: text\nspec: {}")
		}, "name.reference"},
		{"nested field name", func(i *Input) {
			i.Root.Entries[1].Bytes = []byte("id: item\nkind: Example\nbody: text\nspec: {value: [{Wrong: text}]}")
		}, "name.field"},
		{"package uppercase", func(i *Input) { i.Root.Entries[1].Path = "model/Feature/item.yaml" }, "name.package"},
		{"package uppercase numeral", func(i *Input) { i.Root.Entries[1].Path = "model/Ⅰ/item.yaml" }, "name.package"},
		{"package digit start", func(i *Input) { i.Root.Entries[1].Path = "model/٢feature/item.yaml" }, "name.package"},
		{"decomposed filename", func(i *Input) { i.Root.Entries[1].Path = "model/feature/u\u0308ber.yaml" }, "name.filename"},
		{"filename dot", func(i *Input) { i.Root.Entries[1].Path = "model/feature/item.part.yaml" }, "name.filename"},
		{"empty invalid directory", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/empty_dir", Kind: Directory})
		}, "name.package"},
		{"root component hyphen", func(i *Input) {
			i.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: example\nroot: my-model\ndependencies: {}")
		}, "name.root_path"},
		{"model label uppercase", func(i *Input) {
			i.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: Example\nroot: model\ndependencies: {}")
		}, "name.model_label"},
		{"latest version", func(i *Input) {
			i.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: example\nversion: latest\nroot: model\ndependencies: {}")
		}, "name.version_label"},
		{"dependency label", func(i *Input) {
			i.Root.Entries[0].Bytes = []byte("formatVersion: 1\nid: example\nroot: model\ndependencies: {bad_id: '1.x'}")
		}, "name.dependency_label"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := sourceInput()
			tt.edit(&input)
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("expected %s: %+v", tt.code, got)
			}
		})
	}
}

func TestUnicodeCollisionScopes(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Input)
		code string
	}{
		{"definitions", func(i *Input) {
			i.Root.Entries[1].Bytes = []byte("id: Straße\nkind: Example\nbody: text\nspec: {}")
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/feature/other.yaml", Kind: Regular, Bytes: []byte("id: strasse\nkind: Example\nbody: text\nspec: {}")})
		}, "name.definition_collision"},
		{"filenames", func(i *Input) {
			i.Root.Entries[1].Path = "model/feature/straße.yaml"
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/feature/strasse.yaml", Kind: Regular, Bytes: []byte("id: other\nkind: Example\nbody: text\nspec: {}")})
		}, "name.filename_collision"},
		{"empty folders", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/straße/empty", Kind: Directory}, Entry{Path: "model/strasse/other", Kind: Directory})
		}, "name.package_collision"},
		{"import aliases", func(i *Input) {
			i.Root.Entries[1].Bytes = append(i.Root.Entries[1].Bytes, []byte("imports: [{package: first, as: Straße}, {package: second, as: Strasse}]")...)
		}, "name.alias_collision"},
		{"package imports", func(i *Input) {
			i.Root.Entries[1].Bytes = append(i.Root.Entries[1].Bytes, []byte("imports: [{package: straße}, {package: strasse}]")...)
		}, "name.import_duplicate"},
		{"model IDs", func(i *Input) {
			i.Dependencies = []SourceModel{dependencySource("straße", "first", "a", "a"), dependencySource("strasse", "second", "b", "b")}
		}, "name.model_collision"},
		{"field keys", func(i *Input) {
			i.Root.Entries[1].Bytes = []byte("id: item\nkind: Example\nbody: text\nspec: {straße: a, strasse: b}")
		}, "name.key_collision"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := sourceInput()
			tt.edit(&input)
			got := Compile(input)
			if got.Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("expected %s: %+v", tt.code, got)
			}
			slices.Reverse(input.Root.Entries)
			slices.Reverse(input.Dependencies)
			if permuted := Compile(input); !reflect.DeepEqual(got, permuted) {
				t.Fatalf("input order changed Unicode collision diagnostics: %+v / %+v", got, permuted)
			}
		})
	}
}

func TestGlobalUnicodePackagePrefixesExcludeEmptyTrees(t *testing.T) {
	input := sourceInput()
	input.Root.Entries[1].Path = "model/straße/feature/item.yaml"
	dependency := dependencySource("vocabulary", "dependency", "vocabulary", "item")
	dependency.Entries = append(dependency.Entries, Entry{Path: "model/strasse/empty", Kind: Directory})
	input.Dependencies = []SourceModel{dependency}
	if got := Compile(input); got.Status != NotCompleted || len(got.Diagnostics) != 0 {
		t.Fatalf("empty foreign tree bound a global Package name: %+v", got)
	}
	input.Dependencies[0].Entries = append(input.Dependencies[0].Entries, Entry{
		Path: "model/strasse/empty/another.yaml", Kind: Regular,
		Bytes: []byte("id: another\nkind: Example\nbody: text\nspec: {}"),
	})
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "name.package_collision") || got.Models[0].Status != Failed || got.Models[1].Status != Failed {
		t.Fatalf("populated foreign tree did not bind its colliding prefix: %+v", got)
	}
}

func TestKeyNFCAndCasefoldDoNotSilentlySelectAnExactSubset(t *testing.T) {
	input := sourceInput()
	input.Root.Entries[1].Bytes = []byte("id: item\nID: second\nkind: Example\nbody: text\nspec: {u\u0308: text}")
	got := Compile(input)
	if got.Status != Failed || !hasDiagnostic(got, "name.key_collision") || !hasDiagnostic(got, "name.key_nfc") {
		t.Fatalf("colliding/non-NFC keys were corrected or ignored: %+v", got)
	}
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Code == "name.key_collision" && (diagnostic.Position == nil || diagnostic.Position.Line != 2 || diagnostic.Position.Column != 1) {
			t.Fatalf("key collision did not retain its actual token location: %+v", diagnostic)
		}
	}
}

func TestKindSpellingDoesNotGrantMetaCategoryPrivileges(t *testing.T) {
	input := sourceInput()
	input.Root.Entries[1].Bytes = []byte("id: camelName\nkind: Kind\nbody: Category must follow resolution.\nspec: {}")
	got := Compile(input)
	if got.Status != NotCompleted || len(got.Diagnostics) != 0 || got.Models[0].Phases[2].Status != NotCompleted {
		t.Fatalf("written kind was used as a resolved meta-identity: %+v", got)
	}
}

func TestDuplicateCandidateProvenanceDoesNotMakeReportOrderUnstable(t *testing.T) {
	input := sourceInput()
	first := dependencySource("same-model", "same-origin", "a", "first")
	second := dependencySource("same-model", "same-origin", "b", "second")
	second.Entries[1].Bytes = append(second.Entries[1].Bytes, []byte("\nunknown: value\n")...)
	input.Dependencies = []SourceModel{first, second}
	got := Compile(input)
	slices.Reverse(input.Dependencies)
	if reordered := Compile(input); !reflect.DeepEqual(got, reordered) {
		t.Fatalf("duplicate candidate provenance hid distinct phase reports: %+v / %+v", got, reordered)
	}
}
