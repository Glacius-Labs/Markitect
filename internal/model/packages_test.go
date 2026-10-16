package model

import (
	"bytes"
	"reflect"
	"slices"
	"strings"
	"testing"
)

func requirePackages(t *testing.T, model ModelResult, root string, names ...string) {
	t.Helper()
	want := make([]Package, 0, len(names))
	for _, name := range names {
		want = append(want, Package{Name: name, Source: Source{Origin: model.Origin, Path: joinPath(root, name)}})
	}
	if model.Status != Passed || !slices.Equal(model.Packages, want) {
		t.Fatalf("package inventory differs: status=%s got=%+v want=%+v", model.Status, model.Packages, want)
	}
}

func TestPackagesOwnDirectoriesAndExactProvenance(t *testing.T) {
	input := singleValueInput("string", "0", "1", "", "{}")
	baseline := requireCompiled(t, input)
	input.Root.Origin = "opaque:local/source"
	input.Root.Entries = append(input.Root.Entries,
		Entry{Path: "model/empty/leaf", Kind: Directory},
		Entry{Path: "model/feature", Kind: Directory},
		Entry{Path: "model/notes/child/readme.md", Kind: Regular, Bytes: []byte("not a Definition")},
		Entry{Path: "model/überweisung", Kind: Directory},
		Entry{Path: "model/文档", Kind: Directory},
		Entry{Path: "outside/unrelated", Kind: Directory},
	)
	// The same empty prefix in another model remains separate and does not
	// change that dependency's locked content or open any import binding.
	input.Dependencies[0].Entries = append(input.Dependencies[0].Entries, Entry{Path: "model/empty/external", Kind: Directory})
	before := cloneModelInput(input)
	got := requireCompiled(t, input)
	requirePackages(t, got.Models[0], "model", "empty", "empty/leaf", "feature", "notes", "notes/child", "überweisung", "文档")
	requirePackages(t, got.Models[1], "model", "empty", "empty/external", "markitect", "markitect/core")
	for index, model := range got.Models {
		if !bytes.Equal(model.Canonical, baseline.Models[index].Canonical) || model.Digest != baseline.Models[index].Digest {
			t.Fatalf("directory provenance changed canonical content: %s", model.ID)
		}
	}
	if !reflect.DeepEqual(input, before) {
		t.Fatal("Compile changed acquired input")
	}
	slices.Reverse(input.Root.Entries)
	slices.Reverse(input.Dependencies[0].Entries)
	if reversed := requireCompiled(t, input); !reflect.DeepEqual(got, reversed) {
		t.Fatal("source enumeration changed package inventory or compilation result")
	}
}

func TestPackagesSelectedRootAndAcquisitionBoundary(t *testing.T) {
	for _, tt := range []struct{ name, prefix, root string }{
		{"root dot", "", "."},
		{"nested manifest and root", "captured/", "models/world"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := singleValueInput("string", "0", "1", "", "{}")
			baseline := requireCompiled(t, input).Models[0]
			rootPath := tt.prefix
			if tt.root != "." {
				rootPath += tt.root
			}
			input.Root.ManifestPath = tt.prefix + "model.yaml"
			for index := range input.Root.Entries {
				entry := &input.Root.Entries[index]
				if strings.HasPrefix(entry.Path, "model/") {
					entry.Path = joinPath(strings.TrimSuffix(rootPath, "/"), strings.TrimPrefix(entry.Path, "model/"))
				} else {
					entry.Path = tt.prefix + entry.Path
				}
				if entry.Path == input.Root.ManifestPath {
					entry.Bytes = bytes.Replace(entry.Bytes, []byte("root: model"), []byte("root: "+tt.root), 1)
				}
			}
			input.Root.Entries = append(input.Root.Entries, Entry{Path: joinPath(strings.TrimSuffix(rootPath, "/"), "empty/leaf"), Kind: Directory})
			got := requireCompiled(t, input).Models[0]
			requirePackages(t, got, strings.TrimSuffix(rootPath, "/"), "empty", "empty/leaf", "feature")
			if !bytes.Equal(got.Canonical, baseline.Canonical) || got.Digest != baseline.Digest {
				t.Fatal("selected root or manifest location changed canonical content")
			}
		})
	}
}

func TestPackagesPassedEmptyInventoryIsAuthoritative(t *testing.T) {
	input := coreModelInput()
	input.Root.Entries = append(input.Root.Entries, Entry{Path: "model", Kind: Directory})
	baseline := requireCompiled(t, input).Models[0]
	requirePackages(t, baseline, "model")
	if len(baseline.Definitions) != 0 || len(baseline.Canonical) == 0 || baseline.Digest == "" {
		t.Fatal("empty local model did not publish its complete canonical result")
	}
	input.Root.Entries = append(input.Root.Entries, Entry{Path: "model/empty/leaf", Kind: Directory})
	got := requireCompiled(t, input).Models[0]
	requirePackages(t, got, "model", "empty", "empty/leaf")
	if len(got.Definitions) != 0 || !bytes.Equal(got.Canonical, baseline.Canonical) || got.Digest != baseline.Digest {
		t.Fatal("empty local packages changed Definitions or canonical content")
	}
}

func TestPackagesWithheldForFailedModels(t *testing.T) {
	for _, tt := range []struct {
		name, code string
		change     func(*Input)
	}{
		{"invalid empty folder", "name.package", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/empty_dir", Kind: Directory})
		}},
		{"non NFC folder", "name.package", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/u\u0308berweisung", Kind: Directory})
		}},
		{"empty prefix collision", "name.package_collision", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/straße/empty", Kind: Directory}, Entry{Path: "model/strasse/other", Kind: Directory})
		}},
		{"redirect", "source.redirect", func(i *Input) { i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/redirect", Kind: Redirect}) }},
		{"ambiguous input", "input.duplicate_path", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/empty/leaf", Kind: Directory})
		}},
		{"non directory parent", "input.nondirectory_parent", func(i *Input) { i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/empty", Kind: Regular}) }},
		{"missing root", "manifest.root_missing", func(i *Input) {
			i.Root.Entries[0].Bytes = bytes.Replace(i.Root.Entries[0].Bytes, []byte("root: model"), []byte("root: absent"), 1)
		}},
		{"bad Definition", "yaml.syntax", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/broken/item.yaml", Kind: Regular, Bytes: []byte("[")})
		}},
		{"missing lock", "lock.missing", func(i *Input) { i.Root = withLock(i.Root, "", false) }},
		{"known failure beside budget", "name.package", func(i *Input) {
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/empty_dir", Kind: Directory})
			i.Limits.BodyBytes = 4
		}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := singleValueInput("string", "0", "1", "", "{}")
			input.Root.Entries = append(input.Root.Entries, Entry{Path: "model/empty/leaf", Kind: Directory})
			tt.change(&input)
			got := Compile(input)
			if got.Status != Failed || got.Models[0].Status != Failed || !hasDiagnostic(got, tt.code) {
				t.Fatalf("expected known %s failure: status=%s diagnostics=%+v", tt.code, got.Status, got.Diagnostics)
			}
			requireNoPublicContent(t, got.Models[0])
			if tt.name == "bad Definition" || tt.name == "missing lock" {
				// An independently valid dependency still publishes its own
				// complete inventory beside a failed consumer.
				requirePackages(t, got.Models[1], "model", "markitect", "markitect/core")
			}
		})
	}
}

func TestPackagesWithheldForIncompleteModels(t *testing.T) {
	for _, tt := range []struct {
		name, code string
		change     func(*Input)
	}{
		{"source admission", "limit.source_bytes", func(i *Input) { i.Limits.SourceBytes = 1 }},
		{"entry admission", "limit.entries", func(i *Input) { i.Limits.Entries = 1 }},
		{"model admission", "limit.models", func(i *Input) { i.Limits.Models = 1 }},
		{"manifest budget", "limit.yaml_depth", func(i *Input) {
			i.Limits.YAMLDepth = 4
			i.Root.Entries[0].Bytes = append(i.Root.Entries[0].Bytes, []byte("\nextra: {a: {b: {c: {d: value}}}}\n")...)
		}},
		{"Definition budget", "limit.yaml_depth", func(i *Input) {
			i.Limits.YAMLDepth = 4
			i.Root.Entries = append(i.Root.Entries, Entry{Path: "model/pending/item.yaml", Kind: Regular, Bytes: []byte("spec: {a: {b: {c: {d: value}}}}")})
		}},
		{"body budget", "limit.body_bytes", func(i *Input) { i.Limits.BodyBytes = 4 }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := singleValueInput("string", "0", "1", "", "{}")
			input.Root.Entries = append(input.Root.Entries, Entry{Path: "model/empty/leaf", Kind: Directory})
			tt.change(&input)
			got := Compile(input)
			if got.Status != NotCompleted || !hasDiagnostic(got, tt.code) {
				t.Fatalf("expected incomplete %s proof: status=%s diagnostics=%+v", tt.code, got.Status, got.Diagnostics)
			}
			if len(got.Models) != 0 {
				if got.Models[0].Status != NotCompleted {
					t.Fatalf("incomplete local model became %s", got.Models[0].Status)
				}
				requireNoPublicContent(t, got.Models[0])
			}
		})
	}
}
