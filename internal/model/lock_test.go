package model

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
)

const core10Digest = "sha256:b912d62b4e7ecdb181291f1dafa712e8a0f407ad35d336ac2c3420a523ec3de4"

func withLock(source SourceModel, text string, present bool) SourceModel {
	path := joinPath(parent(source.ManifestPath), "model-lock.yaml")
	entries := make([]Entry, 0, len(source.Entries)+1)
	for _, entry := range source.Entries {
		if entry.Path != path {
			entries = append(entries, entry)
		}
	}
	if present {
		entries = append(entries, Entry{Path: path, Kind: Regular, Bytes: []byte(text)})
	}
	source.Entries = entries
	return source
}

func coreLockText() string {
	return "formatVersion: 1\nmodels:\n  markitect-core: {version: '1.0', digest: '" + core10Digest + "'}\n"
}

func requireNoPublicContent(t *testing.T, model ModelResult) {
	t.Helper()
	if model.Status == Passed || model.Packages != nil || len(model.Definitions) != 0 || len(model.Canonical) != 0 || model.Digest != "" {
		t.Fatalf("invalid/incomplete scope exposed public content: id=%s status=%s digest=%s", model.ID, model.Status, model.Digest)
	}
}

func requireChecked(t *testing.T, input Input) Result {
	t.Helper()
	got := Check(input)
	if got.Status != Passed {
		t.Fatalf("valid input failed: status=%s diagnostics=%+v", got.Status, got.Diagnostics)
	}
	for _, model := range got.Models {
		if model.Status != Passed || len(model.Canonical) == 0 || !validContentDigest(model.Digest) {
			t.Fatalf("valid scope withheld canonical content: %s %s", model.ID, model.Status)
		}
		for _, phase := range model.Phases {
			if phase.Status != Passed {
				t.Fatalf("incomplete phase in passed scope: %+v", phase)
			}
		}
	}
	return got
}

func TestLockClosedFormatAndExactContentBinding(t *testing.T) {
	good := coreLockText()
	for _, tt := range []struct{ name, text, code string }{
		{"missing external", "formatVersion: 1\nmodels: {}", "lock.missing_model"},
		{"missing field", "models: {}", "lock.required"},
		{"extra top", good + "path: elsewhere\n", "lock.field"},
		{"wrong case field", strings.Replace(good, "models:", "Models:", 1), "lock.field"},
		{"wrong format", strings.Replace(good, "formatVersion: 1", "formatVersion: 2", 1), "lock.format"},
		{"boolean format", strings.Replace(good, "formatVersion: 1", "formatVersion: true", 1), "lock.format"},
		{"table not mapping", "formatVersion: 1\nmodels: []", "lock.models"},
		{"binding not mapping", "formatVersion: 1\nmodels: {markitect-core: '1.0'}", "lock.object"},
		{"missing digest", "formatVersion: 1\nmodels: {markitect-core: {version: '1.0'}}", "lock.required"},
		{"missing version", "formatVersion: 1\nmodels: {markitect-core: {digest: '" + core10Digest + "'}}", "lock.required"},
		{"extra binding", strings.Replace(good, "version:", "source: path, version:", 1), "lock.field"},
		{"numeric version", strings.Replace(good, "'1.0'", "1", 1), "lock.version"},
		{"nonexact version", strings.Replace(good, "'1.0'", "'1.0.0'", 1), "lock.version_mismatch"},
		{"numeric digest", strings.Replace(good, "'"+core10Digest+"'", "42", 1), "lock.digest"},
		{"wrong digest", strings.Replace(good, core10Digest, "sha256:"+strings.Repeat("f", 64), 1), "lock.digest_mismatch"},
		{"uppercase digest", strings.Replace(good, core10Digest, strings.ToUpper(core10Digest), 1), "lock.digest"},
		{"short digest", strings.Replace(good, core10Digest, core10Digest[:70], 1), "lock.digest"},
		{"long digest", strings.Replace(good, core10Digest, core10Digest+"0", 1), "lock.digest"},
		{"wrong id spelling", strings.Replace(good, "markitect-core:", "Markitect-core:", 1), "lock.model_id"},
		{"own entry", good + "  project: {version: '1.0', digest: '" + core10Digest + "'}\n", "lock.extra"},
		{"unreachable extra", good + "  other: {version: '1.0', digest: '" + core10Digest + "'}\n", "lock.extra"},
		{"duplicate id", good + "  markitect-core: {version: '1.0', digest: '" + core10Digest + "'}\n", ""},
		{"unicode collision", good + "  straße: {version: '1.0', digest: '" + core10Digest + "'}\n  strasse: {version: '1.0', digest: '" + core10Digest + "'}\n", ""},
		{"duplicate format", good + "formatVersion: 1\n", ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := singleValueInput("string", "0", "1", "", "{}")
			input.Root = withLock(input.Root, tt.text, true)
			original := bytes.Clone(input.Root.Entries[len(input.Root.Entries)-1].Bytes)
			got := Check(input)
			if got.Status != Failed || tt.code != "" && !hasDiagnostic(got, tt.code) {
				t.Fatalf("want %s: status=%s diagnostics=%+v", tt.code, got.Status, got.Diagnostics)
			}
			requireNoPublicContent(t, got.Models[0])
			if !bytes.Equal(original, input.Root.Entries[len(input.Root.Entries)-1].Bytes) {
				t.Fatal("lock check rewrote author bytes")
			}
		})
	}
	for _, version := range []string{"latest", "*", "^1.0", ">=1.0", "a_b", "1..0", "u\u0308ber", "Release"} {
		input := singleValueInput("string", "0", "1", "", "{}")
		input.Root = withLock(input.Root, strings.Replace(good, "'1.0'", "'"+version+"'", 1), true)
		got := Check(input)
		if got.Status != Failed {
			t.Fatalf("invalid lock version accepted: %q", version)
		}
	}
}

func TestEntryLockFailureDoesNotInvalidateIndependentDependency(t *testing.T) {
	for _, text := range []string{"", strings.Replace(coreLockText(), core10Digest, "sha256:"+strings.Repeat("a", 64), 1), "invalid: ["} {
		input := singleValueInput("string", "0", "1", "", "{}")
		input.Root = withLock(input.Root, text, text != "")
		got := Check(input)
		if got.Status != Failed {
			t.Fatalf("invalid root lock accepted: %+v", got.Diagnostics)
		}
		requireNoPublicContent(t, got.Models[0])
		if got.Models[1].Status != Passed || got.Models[1].Digest != core10Digest || len(got.Models[1].Definitions) != 9 || phaseStatus(got.Models[1], "lock") != Passed {
			t.Fatalf("consumer lock poisoned independent Core: status=%s digest=%s diagnostics=%+v", got.Models[1].Status, got.Models[1].Digest, got.Diagnostics)
		}
	}
	input := singleValueInput("string", "0", "1", "", "{}")
	input.Root.Entries[3].Bytes = []byte("id: [")
	got := Check(input)
	if got.Status != Failed || got.Models[1].Status != Passed || got.Models[1].Digest != core10Digest {
		t.Fatal("consumer YAML failure poisoned independently valid dependency")
	}
}

func TestDependencySidecarsNeverReplaceTheEntryLock(t *testing.T) {
	input := singleValueInput("string", "0", "1", "", "{}")
	input.Dependencies[0] = withLock(input.Dependencies[0], "formatVersion: 999\nmodels: {arbitrary: {version: '1.0', digest: wrong}}", true)
	requireChecked(t, input)
	input.Root = withLock(input.Root, "", false)
	got := Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "lock.missing") {
		t.Fatal("dependency sidecar replaced missing root lock")
	}
	input.Root = withLock(input.Root, coreLockText(), true)
	input.Dependencies[0] = withLock(input.Dependencies[0], "value: [", true)
	got = Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "yaml.syntax") {
		t.Fatal("ignored dependency sidecar swallowed a known loader error")
	}
	requireNoPublicContent(t, got.Models[0])
	requireNoPublicContent(t, got.Models[1])
}

func TestNoExternalDependenciesStillRequireExplicitEmptyLock(t *testing.T) {
	input := Input{Root: completeCoreSource()}
	requireChecked(t, input)
	input.Root = withLock(input.Root, "", false)
	got := Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "lock.missing") {
		t.Fatal("empty lock was silently generated")
	}
	requireNoPublicContent(t, got.Models[0])
}

func TestLockKnownFormatFailureSurvivesContentRefusal(t *testing.T) {
	input := singleValueInput("string", "0", "1", "", "{}")
	input.Limits.BodyBytes = 4
	input.Root = withLock(input.Root, "formatVersion: 2\nmodels: {}", true)
	got := Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "lock.format") || !hasDiagnostic(got, "limit.body_bytes") {
		t.Fatalf("known lock error erased by content refusal: %+v", got.Diagnostics)
	}
	requireNoPublicContent(t, got.Models[0])
}

func TestLockKnownManifestFactsSurviveDefinitionInventoryRefusal(t *testing.T) {
	for _, tt := range []struct{ text, code string }{
		{strings.Replace(coreLockText(), "'1.0'", "'1.1'", 1), "lock.version_mismatch"},
		{"formatVersion: 1\nmodels: {}", "lock.missing_model"},
		{coreLockText() + "  other: {version: '1.0', digest: '" + core10Digest + "'}\n", "lock.extra"},
		{coreLockText() + "  project: {version: '1.0', digest: '" + core10Digest + "'}\n", "lock.extra"},
		{coreLockText(), ""},
	} {
		input := singleValueInput("string", "0", "1", "", "{}")
		input.Limits.YAMLDepth = 4
		input.Dependencies[0].Entries = append(input.Dependencies[0].Entries, Entry{Path: "model/unavailable/pending.yaml", Kind: Regular, Bytes: []byte("spec: {a: {b: {c: {d: value}}}}")})
		input.Root = withLock(input.Root, tt.text, true)
		got := Check(input)
		if !hasDiagnostic(got, "limit.yaml_depth") {
			t.Fatal("fixture did not refuse definition inventory")
		}
		if tt.code == "" {
			if got.Status != NotCompleted {
				t.Fatalf("unknown digest became a language failure: %+v", got.Diagnostics)
			}
		} else if got.Status != Failed || !hasDiagnostic(got, tt.code) {
			t.Fatalf("known %s erased by definition limit: status=%s diagnostics=%+v", tt.code, got.Status, got.Diagnostics)
		}
		requireNoPublicContent(t, got.Models[0])
	}
}

func TestLockKnownDigestFailureSurvivesAnotherUnknownManifest(t *testing.T) {
	input := singleValueInput("string", "0", "1", "", "{}")
	input.Limits.YAMLDepth = 4
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{markitect-core: '1.0'}"), []byte("{markitect-core: '1.0', mystery: '1.0'}"), 1)
	input.Dependencies = append(input.Dependencies, SourceModel{Origin: "mystery", ManifestPath: "model.yaml", Entries: []Entry{{Path: "model.yaml", Kind: Regular, Bytes: []byte("a: {b: {c: {d: {e: value}}}}")}}})
	text := strings.Replace(coreLockText(), core10Digest, "sha256:"+strings.Repeat("f", 64), 1) + "  mystery: {version: '1.0', digest: 'sha256:" + strings.Repeat("1", 64) + "'}\n"
	input.Root = withLock(input.Root, text, true)
	got := Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "lock.digest_mismatch") || !hasDiagnostic(got, "limit.yaml_depth") || hasDiagnostic(got, "dependency.missing") {
		t.Fatalf("known digest failure lost beside unknown metadata: %+v", got.Diagnostics)
	}
	requireNoPublicContent(t, got.Models[0])
}

func TestLockUnknownClosureDoesNotCompareUnrelatedSuppliedRows(t *testing.T) {
	for _, id := range []string{"straße", "strasse"} {
		input := singleValueInput("string", "0", "1", "", "{}")
		input.Limits.YAMLDepth = 4
		input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{markitect-core: '1.0'}"), []byte("{markitect-core: '1.0', mystery: '1.0'}"), 1)
		unrelated := bindingSource("straße", "1.0", "{markitect-core: '1.0'}", contractDefinition("other/record.yaml", "Record", "Core.Kind", "{}"))
		input.Dependencies = append(input.Dependencies, unrelated, SourceModel{Origin: "mystery", ManifestPath: "model.yaml", Entries: []Entry{{Path: "model.yaml", Kind: Regular, Bytes: []byte("a: {b: {c: {d: {e: value}}}}")}}})
		text := coreLockText() + "  mystery: {version: '1.0', digest: 'sha256:" + strings.Repeat("1", 64) + "'}\n  " + id + ": {version: '9.9', digest: 'sha256:" + strings.Repeat("f", 64) + "'}\n"
		input.Root = withLock(input.Root, text, true)
		got := Check(input)
		if got.Status != NotCompleted || !hasDiagnostic(got, "limit.yaml_depth") || hasDiagnostic(got, "lock.extra") || hasDiagnostic(got, "lock.model_spelling") || hasDiagnostic(got, "lock.version_mismatch") || hasDiagnostic(got, "lock.digest_mismatch") {
			t.Fatalf("unknown closure invented a mismatch against unrelated %q: %+v", id, got.Diagnostics)
		}
		requireNoPublicContent(t, got.Models[0])
	}
}

func TestLockWrongSpellingSelfRequirementIsNeverExternal(t *testing.T) {
	input := singleValueInput("string", "0", "1", "", "{}")
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("id: project"), []byte("id: straße"), 1)
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{markitect-core: '1.0'}"), []byte("{markitect-core: '1.0', strasse: '1.0'}"), 1)
	got := Check(input)
	if got.Status != Failed || !hasDiagnostic(got, "dependency.spelling") || hasDiagnostic(got, "lock.missing_model") {
		t.Fatalf("local-root equivalent became a missing external lock model: %+v", got.Diagnostics)
	}
}

func TestRootLockBindsTransitiveDiamondExactlyOnce(t *testing.T) {
	shared := bindingSource("shared", "1.0", "{markitect-core: '1.0'}", contractDefinition("shared/record.yaml", "Record", "Core.Kind", "{}"))
	alpha := bindingSource("alpha", "1.0", "{markitect-core: '1.0', shared: '1.0'}", contractDefinition("alpha/record.yaml", "Record", "Core.Kind", "{}"))
	beta := bindingSource("beta", "1.0", "{shared: '1.0', markitect-core: '1.0'}", contractDefinition("beta/record.yaml", "Record", "Core.Kind", "{}"))
	core := completeCoreSource()
	digests := map[string]string{"markitect-core": core10Digest}
	for _, model := range []SourceModel{shared, alpha, beta} {
		dependencies := []SourceModel{core}
		if model.Origin != "shared" {
			dependencies = append(dependencies, shared)
		}
		digests[model.Origin] = privateCanonical(t, Input{Root: model, Dependencies: dependencies}).digest
	}
	input := coreModelInput(contractDefinition("feature/record.yaml", "Record", "Core.Kind", "{}"))
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{markitect-core: '1.0'}"), []byte("{alpha: '1.0', beta: '1.0'}"), 1)
	// The root uses Core directly, so that exact direct requirement remains.
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{alpha: '1.0', beta: '1.0'}"), []byte("{alpha: '1.0', beta: '1.0', markitect-core: '1.0'}"), 1)
	input.Dependencies = []SourceModel{beta, core, alpha, shared}
	text := "formatVersion: 1\nmodels:\n"
	for _, id := range []string{"beta", "shared", "markitect-core", "alpha"} {
		text += fmt.Sprintf("  %s: {version: '1.0', digest: '%s'}\n", id, digests[id])
	}
	input.Root = withLock(input.Root, text, true)
	requireChecked(t, input)
	input.Root = withLock(input.Root, strings.Replace(text, fmt.Sprintf("  shared: {version: '1.0', digest: '%s'}\n", digests["shared"]), "", 1), true)
	if got := Check(input); got.Status != Failed || !hasDiagnostic(got, "lock.missing_model") {
		t.Fatalf("transitive model omitted from root table: %+v", got.Diagnostics)
	}
}

func TestLockExactUnicodeAndLiteralVersionLabels(t *testing.T) {
	for _, version := range []string{"x", "1.x", "release.x", "٢٠٢٦.١"} {
		input := singleValueInput("string", "0", "1", "", "{}")
		input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("markitect-core: '1.0'"), []byte("markitect-core: '"+version+"'"), 1)
		input.Dependencies[0].Entries[0].Bytes = bytes.Replace(input.Dependencies[0].Entries[0].Bytes, []byte("version: '1.0'"), []byte("version: '"+version+"'"), 1)
		// Derive the expected version-only change from the independent fixed
		// Core byte vector, not from the product canonicalizer.
		expected := strings.Replace(canonicalCoreVector, `"version":"1.0.0"`, `"version":"`+version+`"`, 1)
		digest := fmt.Sprintf("sha256:%x", sha256.Sum256([]byte(expected)))
		input.Root = withLock(input.Root, "formatVersion: 1\nmodels:\n  markitect-core: {version: '"+version+"', digest: '"+digest+"'}", true)
		requireChecked(t, input)
	}
	input := singleValueInput("string", "0", "1", "", "{}")
	model := bindingSource("文档", "1.0", "{markitect-core: '1.0'}", contractDefinition("文档/record.yaml", "Record", "Core.Kind", "{}"))
	digest := privateCanonical(t, Input{Root: model, Dependencies: []SourceModel{completeCoreSource()}}).digest
	input.Root.Entries[0].Bytes = bytes.Replace(input.Root.Entries[0].Bytes, []byte("{markitect-core: '1.0'}"), []byte("{markitect-core: '1.0', 文档: '1.0'}"), 1)
	input.Dependencies = append(input.Dependencies, model)
	input.Root = withLock(input.Root, coreLockText()+"  文档: {version: '1.0', digest: '"+digest+"'}", true)
	requireChecked(t, input)
}
