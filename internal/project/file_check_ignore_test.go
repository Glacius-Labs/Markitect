package project_test

import (
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

func TestFileCheckExactIgnorePolicyAndObservedAbsence(t *testing.T) {
	for _, policy := range []string{"", emptyFilePolicy, editorFilePolicy} {
		selection, raw := fileCheckInputs(t)
		if policy == editorFilePolicy {
			censusFile(t, &raw, ".editor/library-session.json", snapshot.Regular, []byte("editor"))
		}
		if policy != "" {
			fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, policy, true)
		}
		result := checkedFiles(t, selection, raw)
		accounting, ran := result.Accounting()
		if result.Status() != model.Passed || !ran || accounting.Status() != realization.Accounted {
			t.Fatalf("valid exact policy failed: %+v %+v", result.Diagnostics(), accounting.Findings())
		}
		for _, file := range accounting.Files() {
			if file.Entry().Path().String() == ".editor/library-session.json" && (file.Class() != realization.IgnoredFile || file.Why() != "  Local editor layout.  ") {
				t.Fatal("exact ignore reason changed")
			}
		}
		if string(result.RegistryPreparation().Model().Inputs().Raw.Execution[0].File.Bytes()) != documentedExecution(t) {
			t.Fatal("execution source changed")
		}
	}
}

func TestFileCheckUncapturedPolicyKeepsActualAccountingIncomplete(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	censusFile(t, &raw, ".editor/library-session.json", snapshot.Regular, []byte("editor"))
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, editorFilePolicy, false)
	result := checkedFiles(t, selection, raw)
	accounting, ran := result.Accounting()
	if result.Status() != model.NotCompleted || !ran || accounting.Status() != realization.AccountingInvalid || !fileDiagnostic(result, "files.ignore_uncaptured") {
		t.Fatalf("missing policy became empty or known invalid: %s %+v", result.Status(), result.Diagnostics())
	}
	if classes, ok := result.Classifications(); ok || len(classes) != 0 {
		t.Fatal("partial classifications became usable")
	}
	if protections, ok := result.Protections(); ok || len(protections) != 0 {
		t.Fatal("partial protections became usable")
	}
	if len(accounting.Findings()) != 1 || accounting.Findings()[0].Code != "file.unexplained" {
		t.Fatalf("actual partial accounting lost: %+v", accounting.Findings())
	}
}

func TestFileCheckRejectsInvalidExactPolicy(t *testing.T) {
	for _, tc := range []struct{ name, from, to, code string }{
		{"version", "markitect.ignore/v1", "markitect.ignore/v2", "version"},
		{"legacy", "reason:", "why:", "unknown_key"},
		{"unknown", "files:", "directories: []\nfiles:", "unknown_key"},
		{"duplicate field", "files:", "files: []\nfiles:", "duplicate_key"},
		{"duplicate path", "    reason:", "    path: other\n    reason:", "duplicate_key"},
		{"null", "files:\n  - path: .editor/library-session.json\n    reason: '  Local editor layout.  '", "files: null", "yaml_feature"},
		{"implicit null", "'  Local editor layout.  '", "", "yaml_feature"},
		{"anchor", "files:", "files: &policy", "yaml_feature"},
		{"alias", "    reason: '  Local editor layout.  '", "    reason: &reason editor\n  - path: .editor/library-session.json\n    reason: *reason", "yaml_feature"},
		{"tag", "files:", "files: !policy", "yaml_feature"},
		{"URI tag", "apiVersion:", "!<tag:example.org,2026:policy>\napiVersion:", "yaml_feature"},
		{"multiple documents", "apiVersion:", "---\n{}\n---\napiVersion:", "documents"},
		{"nonstring path", ".editor/library-session.json", "9", "string"},
		{"blank reason", "'  Local editor layout.  '", "'  '", "string"},
		{"missing reason", "    reason: '  Local editor layout.  '\n", "", "field_missing"},
		{"directory", ".editor/library-session.json", ".editor/", "path"},
		{"implicit directory", ".editor/library-session.json", ".editor", "missing_file"},
		{"absolute", ".editor/library-session.json", "/editor.json", "path"},
		{"escape", ".editor/library-session.json", "../editor.json", "path"},
		{"normalization", ".editor/library-session.json", "./.editor/library-session.json", "path"},
		{"stale", ".editor/library-session.json", ".editor/deleted.json", "missing_file"},
		{"no glob expansion", ".editor/library-session.json", ".editor/*.json", "missing_file"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			selection, raw := fileCheckInputs(t)
			censusFile(t, &raw, ".editor/library-session.json", snapshot.Regular, []byte("editor"))
			if !strings.Contains(editorFilePolicy, tc.from) {
				t.Fatal("fixture mutation missing")
			}
			policy := strings.Replace(editorFilePolicy, tc.from, tc.to, 1)
			fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, policy, true)
			result := checkedFiles(t, selection, raw)
			if result.Status() != model.Failed || !fileDiagnostic(result, "files.ignore."+tc.code) {
				t.Fatalf("invalid policy passed: %s %+v", result.Status(), result.Diagnostics())
			}
			if string(result.RegistryPreparation().Model().Inputs().Raw.Ignore[0].File.Bytes()) != policy {
				t.Fatal("malformed source lost")
			}
		})
	}
}

func TestFileCheckLiteralWildcardFilenameIsNeverExpanded(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	censusFile(t, &raw, ".editor/*.json", snapshot.Regular, []byte("literal filename in supplied tree"))
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, strings.Replace(editorFilePolicy, ".editor/library-session.json", ".editor/*.json", 1), true)
	if result := checkedFiles(t, selection, raw); result.Status() != model.Passed {
		t.Fatalf("literal path was treated as a pattern: %+v", result.Diagnostics())
	}
}

func TestFileCheckRejectsDuplicateIgnoreRowsAndInputArtifactOverlaps(t *testing.T) {
	for _, target := range []string{".markitect/model.yaml", ".markitect/execution.yaml", ".markitect/ignore.yaml", ".markitect/registry/registry.yaml", "realization/lending.go"} {
		selection, raw := fileCheckInputs(t)
		policy := "apiVersion: markitect.ignore/v1\nfiles: [{path: " + target + ", reason: cannot-hide}]\n"
		fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, policy, true)
		result := checkedFiles(t, selection, raw)
		accounting, ran := result.Accounting()
		if result.Status() != model.Failed || !ran || accounting.Status() != realization.AccountingInvalid {
			t.Fatalf("overlap hidden: %q %+v", target, accounting.Findings())
		}
	}
	selection, raw := fileCheckInputs(t)
	censusFile(t, &raw, ".editor/library-session.json", snapshot.Regular, []byte("editor"))
	row := "  - path: .editor/library-session.json\n    reason: '  Local editor layout.  '\n"
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, editorFilePolicy+row, true)
	if result := checkedFiles(t, selection, raw); result.Status() != model.Failed || !fileDiagnostic(result, "files.ignore.duplicate_path") {
		t.Fatal("duplicate exact path accepted")
	}
}

func TestFileCheckDiagnosticOrderIncludesDistinctSourcePositions(t *testing.T) {
	selection, raw := fileCheckInputs(t)
	policy := "apiVersion: markitect.ignore/v1\nfiles: []\nfiles: []\nfiles: []\nfiles: []\n"
	fileCheckSource(t, &selection, &raw, "ignore", ".markitect/ignore.yaml", snapshot.Regular, policy, true)
	result := checkedFiles(t, selection, raw)
	var lines []int
	for _, diagnostic := range result.Diagnostics() {
		if diagnostic.Code == "files.ignore.duplicate_key" {
			if diagnostic.Source.Field != "/files" || diagnostic.Source.ContentID != raw.Ignore[0].File.ContentID() || diagnostic.Position == nil {
				t.Fatal("duplicate-key provenance missing")
			}
			lines = append(lines, diagnostic.Position.Line)
		}
	}
	if len(lines) != 3 || lines[0] != 3 || lines[1] != 4 || lines[2] != 5 {
		t.Fatalf("source positions not totally ordered: %v", lines)
	}
}
