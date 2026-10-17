package model

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"reflect"
	"slices"
	"strings"
	"testing"
)

const inspectionYAML = "formatVersion: 1\nid: project\nroot: model\ndependencies: {}\n"

func inspectionInput(text string) ManifestInput {
	return ManifestInput{Origin: "captured-manifest", ManifestPath: "config/model.yaml", Bytes: []byte(text)}
}

func manifestDiagnostic(result ManifestResult, code string) bool {
	return slices.ContainsFunc(result.Diagnostics, func(d Diagnostic) bool { return d.Code == code })
}

func TestInspectManifestPublishesOnlyLexicallyValidRoot(t *testing.T) {
	input := ManifestInput{Origin: "captured-manifest", ManifestPath: "config/model.yaml", Bytes: []byte("formatVersion: 1\nid: project\nroot: überweisung/文档\ndependencies: {}\n")}
	got := InspectManifest(input)
	if got.Status != Passed || got.Root != "überweisung/文档" || got.Source.Origin != input.Origin || got.Source.Path != input.ManifestPath || got.Source.Digest == "" || len(got.Diagnostics) != 0 {
		t.Fatalf("valid manifest inspection failed: %+v", got)
	}
	input.Bytes = []byte("formatVersion: 1\nid: project\nroot: ../world\ndependencies: {}\n")
	got = InspectManifest(input)
	if got.Status != Failed || got.Root != "" || len(got.Diagnostics) == 0 {
		t.Fatalf("invalid manifest exposed a usable root: %+v", got)
	}
}

func TestInspectManifestExactRootsAndOriginalBytes(t *testing.T) {
	for _, root := range []string{".", "model", "docs/model", "überweisung/文档"} {
		t.Run(root, func(t *testing.T) {
			input := inspectionInput("%YAML 1.2\r\n---\r\n# original source\r\nformatVersion: 1\r\nid: ٢٠٢٦.١\r\nversion: '1.x'\r\nroot: " + root + "\r\ndependencies: {文档: '٢.١', vocabulary: x}\r\n")
			original := bytes.Clone(input.Bytes)
			got := InspectManifest(input)
			wantSource := Source{Origin: input.Origin, Path: input.ManifestPath, Digest: fmt.Sprintf("sha256:%x", sha256.Sum256(original))}
			if got.Status != Passed || got.Root != root || got.Source != wantSource || len(got.Diagnostics) != 0 || !bytes.Equal(input.Bytes, original) {
				t.Fatalf("manifest spelling/source bytes changed: %+v", got)
			}
			if repeated := InspectManifest(input); !reflect.DeepEqual(got, repeated) {
				t.Fatal("same manifest did not produce the same inspection")
			}
		})
	}
}

func TestInspectManifestRejectsLexicalAndRestrictedYAMLViolations(t *testing.T) {
	for _, tt := range []struct{ name, text, code string }{
		{"missing format", strings.Replace(inspectionYAML, "formatVersion: 1\n", "", 1), "manifest.format"},
		{"quoted format", strings.Replace(inspectionYAML, "formatVersion: 1", "formatVersion: '1'", 1), "manifest.format"},
		{"wrong format", strings.Replace(inspectionYAML, "formatVersion: 1", "formatVersion: 2", 1), "manifest.format"},
		{"missing ID", strings.Replace(inspectionYAML, "id: project\n", "", 1), "field.required"},
		{"numeric ID", strings.Replace(inspectionYAML, "id: project", "id: 42", 1), "field.nonempty_string"},
		{"empty ID", strings.Replace(inspectionYAML, "id: project", "id: ''", 1), "field.nonempty_string"},
		{"wrong ID case", strings.Replace(inspectionYAML, "id: project", "id: Project", 1), "name.model_label"},
		{"non NFC ID", strings.Replace(inspectionYAML, "id: project", "id: u\u0308ber", 1), "name.model_label"},
		{"missing root", strings.Replace(inspectionYAML, "root: model\n", "", 1), "field.required"},
		{"numeric root", strings.Replace(inspectionYAML, "root: model", "root: 42", 1), "field.nonempty_string"},
		{"missing dependencies", strings.Replace(inspectionYAML, "dependencies: {}\n", "", 1), "manifest.dependencies"},
		{"dependencies list", strings.Replace(inspectionYAML, "dependencies: {}", "dependencies: []", 1), "manifest.dependencies"},
		{"numeric dependency version", strings.Replace(inspectionYAML, "dependencies: {}", "dependencies: {vocabulary: 1}", 1), "field.nonempty_string"},
		{"dependency ID", strings.Replace(inspectionYAML, "dependencies: {}", "dependencies: {invalid_id: x}", 1), "name.dependency_label"},
		{"latest dependency", strings.Replace(inspectionYAML, "dependencies: {}", "dependencies: {vocabulary: latest}", 1), "name.version_label"},
		{"dependency label collision", strings.Replace(inspectionYAML, "dependencies: {}", "dependencies: {straße: x, strasse: x}", 1), "name.key_collision"},
		{"numeric version", inspectionYAML + "version: 1\n", "field.nonempty_string"},
		{"latest version", inspectionYAML + "version: latest\n", "name.version_label"},
		{"version range", inspectionYAML + "version: '^1.0'\n", "name.version_label"},
		{"unknown field", inspectionYAML + "imports: []\n", "field.unknown"},
		{"wrong field case", strings.Replace(inspectionYAML, "root:", "Root:", 1), "field.unknown"},
		{"duplicate key", inspectionYAML + "id: other\n", "yaml.duplicate_key"},
		{"null", strings.Replace(inspectionYAML, "root: model", "root: null", 1), "yaml.null"},
		{"anchor", strings.Replace(inspectionYAML, "root: model", "root: &path model", 1), "yaml.anchor_alias"},
		{"alias", strings.Replace(inspectionYAML, "root: model", "root: *path", 1), "yaml.syntax"},
		{"tag", strings.Replace(inspectionYAML, "root: model", "root: !!str model", 1), "yaml.explicit_tag"},
		{"YAML 1.1 directive", "%YAML 1.1\n---\n" + inspectionYAML, "yaml.directive"},
		{"multiple documents", inspectionYAML + "---\nid: other\n", "yaml.documents"},
		{"sequence root", "[]", "yaml.root"},
		{"empty document", "", "yaml.syntax"},
		{"bad syntax", "[", "yaml.syntax"},
		{"bad UTF8", string([]byte{0xff}), "yaml.utf8"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			input := inspectionInput(tt.text)
			original := bytes.Clone(input.Bytes)
			got := InspectManifest(input)
			if got.Status != Failed || got.Root != "" || !manifestDiagnostic(got, tt.code) || !bytes.Equal(input.Bytes, original) {
				t.Fatalf("expected %s with no root: %+v", tt.code, got)
			}
			// These are the exact same lexical diagnostics Compile emits before
			// performing its additional root/tree/lock/complete-v1 checks.
			_, compiledDiagnostics, _ := compileSource(SourceModel{Origin: input.Origin, ManifestPath: input.ManifestPath, Entries: []Entry{
				{Path: input.ManifestPath, Kind: Regular, Bytes: input.Bytes},
				{Path: "config/model", Kind: Directory},
			}}, DefaultLimits())
			for _, diagnostic := range got.Diagnostics {
				if diagnostic.Source != got.Source || !slices.ContainsFunc(compiledDiagnostics, func(d Diagnostic) bool { return reflect.DeepEqual(d, diagnostic) }) {
					t.Fatalf("inspection/Compile diagnostic source or meaning diverged: %+v / %+v", diagnostic, compiledDiagnostics)
				}
			}
		})
	}
}

func TestInspectManifestExactPathGrammar(t *testing.T) {
	for _, path := range []string{"", "model.yml", "Model.yaml", "config/MODEL.YAML", "./model.yaml", "../model.yaml", "config//model.yaml", "config\\model.yaml", "/model.yaml", "C:/model.yaml", "bad\x00/model.yaml"} {
		input := inspectionInput(inspectionYAML)
		input.ManifestPath = path
		got := InspectManifest(input)
		if got.Status != Failed || got.Root != "" || !manifestDiagnostic(got, "input.manifest_path") {
			t.Fatalf("manifest selection %q accepted: %+v", path, got)
		}
		for _, diagnostic := range got.Diagnostics {
			if diagnostic.Source != got.Source {
				t.Fatalf("known original bytes lost from path diagnostic: %+v / %+v", diagnostic.Source, got.Source)
			}
		}
	}
	for _, root := range []string{"", "./model", "../model", "model/", "docs//model", "model\\child", "/model", "C:/model", "Model", "model-part", "model_dir", "1model", "u\u0308ber"} {
		input := inspectionInput(strings.Replace(inspectionYAML, "root: model", "root: "+fmt.Sprintf("%q", root), 1))
		got := InspectManifest(input)
		if got.Status != Failed || got.Root != "" || len(got.Diagnostics) == 0 {
			t.Fatalf("root %q accepted: %+v", root, got)
		}
	}
}

func TestInspectManifestLimitsWithholdRootAndRetainKnownErrors(t *testing.T) {
	for _, limits := range []Limits{{SourceBytes: -1}, {Entries: -1}, {Models: -1}, {YAMLDepth: -1}, {BodyBytes: -1}, {MarkdownNesting: -1}} {
		input := inspectionInput(inspectionYAML)
		input.Limits = limits
		got := InspectManifest(input)
		if got.Status != NotCompleted || got.Root != "" || got.Source.Digest != "" || !manifestDiagnostic(got, "config.invalid_limit") {
			t.Fatalf("invalid tool configuration claimed manifest validity: %+v", got)
		}
	}
	input := inspectionInput(inspectionYAML)
	input.Limits.SourceBytes = int64(len(input.Bytes))
	input.Limits.Entries, input.Limits.Models, input.Limits.BodyBytes, input.Limits.MarkdownNesting = 1, 1, 1, 1
	if got := InspectManifest(input); got.Status != Passed {
		t.Fatalf("exact byte boundary/unrelated budgets refused one manifest: %+v", got)
	}
	input.Limits.SourceBytes--
	if got := InspectManifest(input); got.Status != NotCompleted || got.Root != "" || got.Source.Digest != "" || !manifestDiagnostic(got, "limit.source_bytes") {
		t.Fatalf("byte refusal became validity or hashed unadmitted bytes: %+v", got)
	}
	input.ManifestPath = "wrong.yaml"
	got := InspectManifest(input)
	if got.Status != Failed || got.Root != "" || got.Source.Digest != "" || !manifestDiagnostic(got, "input.manifest_path") || !manifestDiagnostic(got, "limit.source_bytes") {
		t.Fatalf("byte refusal erased independently known selection failure: %+v", got)
	}
	for _, diagnostic := range got.Diagnostics {
		if diagnostic.Source != got.Source {
			t.Fatalf("byte-refused diagnostic invented a source digest: %+v", diagnostic.Source)
		}
	}
	input = inspectionInput(inspectionYAML + "extra: {a: {b: {c: {d: value}}}}\n")
	input.Limits.YAMLDepth = 3
	if got := InspectManifest(input); got.Status != NotCompleted || got.Root != "" || !manifestDiagnostic(got, "limit.yaml_depth") {
		t.Fatalf("incomplete YAML became manifest validity: %+v", got)
	}
	input.Bytes = append([]byte("bad: null\n"), input.Bytes...)
	if got := InspectManifest(input); got.Status != Failed || got.Root != "" || !manifestDiagnostic(got, "yaml.null") || !manifestDiagnostic(got, "limit.yaml_depth") {
		t.Fatalf("known YAML violation was hidden by depth refusal: %+v", got)
	}
}

func TestInspectManifestPreservesDiagnosticCoordinates(t *testing.T) {
	input := inspectionInput("# provenance\r\nformatVersion: 1\r\nid: project\r\nroot: 7\r\ndependencies: {}\r\n")
	got := InspectManifest(input)
	if got.Status != Failed || len(got.Diagnostics) != 1 || got.Diagnostics[0].Code != "field.nonempty_string" || got.Diagnostics[0].Source != got.Source || got.Diagnostics[0].Position == nil || *got.Diagnostics[0].Position != (Position{Line: 4, Column: 7}) {
		t.Fatalf("original manifest coordinate was lost: %+v", got)
	}
}

func TestInspectManifestDoesNotProveTreeLockOrCompleteCompilation(t *testing.T) {
	input := inspectionInput(strings.Replace(inspectionYAML, "root: model", "root: unavailable", 1))
	if got := InspectManifest(input); got.Status != Passed || got.Root != "unavailable" {
		t.Fatalf("lexical inspection attempted to acquire root: %+v", got)
	}
	compiled := Compile(Input{Root: SourceModel{Origin: input.Origin, ManifestPath: input.ManifestPath, Entries: []Entry{{Path: input.ManifestPath, Kind: Regular, Bytes: input.Bytes}}}})
	if compiled.Status != Failed || !hasDiagnostic(compiled, "manifest.root_missing") {
		t.Fatalf("Compile stopped checking actual supplied root: %+v", compiled)
	}
	complete := singleValueInput("string", "0", "1", "", "{}")
	complete.Root = withLock(complete.Root, "", false)
	manifest := complete.Root.Entries[0]
	if got := InspectManifest(ManifestInput{Origin: complete.Root.Origin, ManifestPath: manifest.Path, Bytes: manifest.Bytes}); got.Status != Passed {
		t.Fatalf("manifest inspector attempted the separate lock obligation: %+v", got)
	}
	compiled = Compile(complete)
	if compiled.Status != Failed || !hasDiagnostic(compiled, "lock.missing") {
		t.Fatalf("Compile lost its lock obligation after manifest refactor: %+v", compiled)
	}
	if zero := (ManifestResult{}); zero.Status == Passed || zero.Root != "" {
		t.Fatal("zero manifest result claims a usable root")
	}
	if got := InspectManifest(ManifestInput{}); got.Status == Passed || got.Root != "" {
		t.Fatalf("zero manifest input claims a usable root: %+v", got)
	}
}
