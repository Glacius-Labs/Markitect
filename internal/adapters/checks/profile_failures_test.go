package checks

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestProcessProfileRejectsClosedFormatVersionsTypesAndPins(t *testing.T) {
	bytes := []byte("ordinary file bytes")
	executable := filepath.Join(testkit.TempDir(t), "check.exe")
	if err := os.WriteFile(executable, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	valid := processFields(executable, bytes)
	for _, tc := range []struct{ name, from, to, code string }{
		{"version absent", "profileVersion: 1\n", "", "process.field_missing"},
		{"wrong version", "profileVersion: 1", "profileVersion: 2", "process.version"},
		{"version string", "profileVersion: 1", "profileVersion: '1'", "process.version"},
		{"version negative", "profileVersion: 1", "profileVersion: -1", "process.version"},
		{"version overflow", "profileVersion: 1", "profileVersion: 4294967296", "process.version"},
		{"version mapping", "profileVersion: 1", "profileVersion: {}", "process.version"},
		{"empty executable", "executable:", "unused:", "process.unknown_field"},
		{"missing executable", "executable:", "other:", "process.field_missing"},
		{"executable type", "profileVersion: 1", "executable: true\nprofileVersion: 1", ""},
		{"unknown args", "profileVersion: 1", "args: [test]\nprofileVersion: 1", "process.unknown_field"},
		{"unknown env", "profileVersion: 1", "env: {}\nprofileVersion: 1", "process.unknown_field"},
		{"unknown timeout", "profileVersion: 1", "timeout: 2m\nprofileVersion: 1", "process.unknown_field"},
		{"relative path", "executable:", "executable:", "process.path"},
		{"path search", "executable:", "executable:", "process.path"},
		{"missing pin", "sha256:", "pin:", "process.field_missing"},
		{"short pin", "sha256:", "sha256:", "process.pin"},
		{"invalid pin", "sha256:", "sha256:", "process.pin"},
		{"pin type", "sha256:", "sha256:", "process.string"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			inner := strings.Replace(valid, tc.from, tc.to, 1)
			switch tc.name {
			case "empty executable":
				inner = "profileVersion: 1\nexecutable: ''\nsha256: '" + strings.Repeat("0", 64) + "'\n"
				tc.code = "process.string"
			case "executable type":
				inner = "profileVersion: 1\nexecutable: true\nsha256: '" + strings.Repeat("0", 64) + "'\n"
				tc.code = "process.string"
			case "relative path", "path search":
				inner = processFields("check.exe", bytes)
			case "short pin":
				inner = "profileVersion: 1\nexecutable: 'not-read'\nsha256: '123'\n"
			case "invalid pin":
				inner = "profileVersion: 1\nexecutable: 'not-read'\nsha256: '" + strings.Repeat("z", 64) + "'\n"
			case "pin type":
				inner = "profileVersion: 1\nexecutable: 'not-read'\nsha256: true\n"
			}
			input := processInput(t, inner)
			calls := 0
			o, err := (MetadataPreparer{observe: func(string) { calls++ }}).PrepareCheckProfile(context.Background(), input)
			if err != nil || o.Status != model.Failed || len(o.Files) != 0 || calls != 0 || o.InputBinding != input.Binding() {
				t.Fatalf("bad profile reached filesystem: %v %+v", err, o)
			}
			found := false
			for _, diagnostic := range o.Diagnostics {
				if diagnostic.Code == tc.code {
					found = true
				}
				if diagnostic.Source.ContentID != input.Document().File.ContentID() || !strings.HasPrefix(diagnostic.Source.Field, input.Field()) {
					t.Fatal("format error lost actual Host source")
				}
			}
			if !found {
				t.Fatalf("missing %s diagnostic: %+v", tc.code, o.Diagnostics)
			}
		})
	}
}

func TestProcessProfilePinMismatchRetainsActualFileAndUppercasePinMatches(t *testing.T) {
	bytes := []byte("actual complete bytes")
	executable := filepath.Join(testkit.TempDir(t), "check.exe")
	if err := os.WriteFile(executable, bytes, 0700); err != nil {
		t.Fatal(err)
	}
	input := processInput(t, processFields(executable, []byte("wrong bytes")))
	o, err := (MetadataPreparer{}).PrepareCheckProfile(context.Background(), input)
	if err != nil || o.Status != model.Failed || len(o.Files) != 1 || o.Files[0].Size != int64(len(bytes)) || o.Files[0].ContentID.String() != fmtDigest(bytes) {
		t.Fatalf("pin mismatch erased actual observation: %v %+v", err, o)
	}
	input = processInput(t, strings.Replace(processFields(executable, bytes), fmtDigest(bytes), strings.ToUpper(fmtDigest(bytes)), 1))
	o, err = (MetadataPreparer{}).PrepareCheckProfile(context.Background(), input)
	if err != nil || o.Status != model.Passed {
		t.Fatalf("explicit uppercase hex pin refused: %v %+v", err, o)
	}
}

func fmtDigest(bytes []byte) string { return fmt.Sprintf("%x", sha256.Sum256(bytes)) }

func TestProcessProfileMissingNonregularOversizeAndCancellation(t *testing.T) {
	bytes := []byte("file bytes")
	directory := testkit.TempDir(t)
	executable := filepath.Join(directory, "check.exe")
	for _, tc := range []string{"missing", "directory", "oversize", "cancelled", "zero input"} {
		t.Run(tc, func(t *testing.T) {
			file := executable
			if tc == "directory" {
				file = directory
			}
			if tc == "oversize" {
				file = filepath.Join(directory, "oversize.exe")
				f, err := os.Create(file)
				if err != nil {
					t.Fatal(err)
				}
				err = f.Truncate(maxExecutableBytes + 1)
				closeErr := f.Close()
				if err != nil || closeErr != nil {
					t.Fatalf("oversize fixture: %v %v", err, closeErr)
				}
			}
			input := processInput(t, processFields(file, bytes))
			ctx := context.Background()
			if tc == "cancelled" {
				cancelled, cancel := context.WithCancel(ctx)
				cancel()
				ctx = cancelled
			}
			if tc == "zero input" {
				input = apply.ProfileInput{}
			}
			o, err := (MetadataPreparer{}).PrepareCheckProfile(ctx, input)
			want := model.NotCompleted
			if tc == "directory" || tc == "zero input" {
				want = model.Failed
			}
			if o.Status != want || len(o.Files) != 0 || len(o.Diagnostics) == 0 {
				t.Fatalf("incomplete/invalid metadata passed: %v %+v", err, o)
			}
			if tc == "missing" && !errors.Is(err, os.ErrNotExist) || tc == "oversize" && !errors.Is(err, ErrMetadataLimit) || tc == "cancelled" && !errors.Is(err, context.Canceled) || tc == "directory" && !errors.Is(err, ErrUnsafeExecutable) {
				t.Fatalf("original error lost: %s %v", tc, err)
			}
		})
	}
}

func TestProcessProfileKeepsPartialEvidenceAndDetectsRestoredTimestampRewrite(t *testing.T) {
	for _, tc := range []string{"rewrite", "cancel", "disappear"} {
		t.Run(tc, func(t *testing.T) {
			bytes := []byte("original bytes")
			executable := filepath.Join(testkit.TempDir(t), "check.exe")
			if err := os.WriteFile(executable, bytes, 0600); err != nil {
				t.Fatal(err)
			}
			info, err := os.Stat(executable)
			if err != nil {
				t.Fatal(err)
			}
			input := processInput(t, processFields(executable, bytes))
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			p := MetadataPreparer{observe: func(stage string) {
				if stage != "after-read" {
					return
				}
				switch tc {
				case "rewrite":
					if err := os.WriteFile(executable, []byte("modified bytes"), 0600); err != nil {
						t.Fatal(err)
					}
					if err := os.Chtimes(executable, info.ModTime(), info.ModTime()); err != nil {
						t.Fatal(err)
					}
				case "cancel":
					cancel()
				case "disappear":
					if err := os.Remove(executable); err != nil {
						t.Fatal(err)
					}
				}
			}}
			o, err := p.PrepareCheckProfile(ctx, input)
			if err == nil || o.Status != model.NotCompleted || len(o.Files) != 1 || o.Files[0].ContentID.String() != fmtDigest(bytes) || o.InputBinding != input.Binding() {
				t.Fatalf("partial evidence/stale refusal lost: %v %+v", err, o)
			}
			if tc == "rewrite" && !errors.Is(err, ErrExecutableChanged) || tc == "cancel" && !errors.Is(err, context.Canceled) {
				t.Fatalf("actual drift/cancellation error lost: %v", err)
			}
		})
	}
}

func TestProcessProfileResultsAndInputsDoNotAliasFutureObservations(t *testing.T) {
	bytes := []byte("read only file bytes")
	executable := filepath.Join(testkit.TempDir(t), "check.exe")
	if err := os.WriteFile(executable, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	input := processInput(t, processFields(executable, bytes))
	p := MetadataPreparer{}
	before, err := p.PrepareCheckProfile(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := p.PrepareCheckProfile(context.Background(), input)
	if err != nil {
		t.Fatal(err)
	}
	changed.Files[0].Size++
	node := input.Inline()
	node.Content[1].Value = "changed"
	after, err := p.PrepareCheckProfile(context.Background(), input)
	if err != nil || !reflect.DeepEqual(before, after) || reflect.DeepEqual(changed, after) {
		t.Fatal("results or opaque input alias another observation")
	}
}

func TestProcessProfileDoesNotNormalizeExpandOrGlobPaths(t *testing.T) {
	bytes := []byte("matching literal file")
	directory := testkit.TempDir(t)
	executable := filepath.Join(directory, "matching-file.exe")
	if err := os.WriteFile(executable, bytes, 0600); err != nil {
		t.Fatal(err)
	}
	for _, literal := range []string{
		directory + string(filepath.Separator) + "." + string(filepath.Separator) + "matching-file.exe",
		filepath.Join(directory, "matching-*.exe"),
		filepath.Join(directory, "$CHECK_PROFILE_EXECUTABLE"),
	} {
		input := processInput(t, processFields(literal, bytes))
		o, err := (MetadataPreparer{}).PrepareCheckProfile(context.Background(), input)
		if o.Status == model.Passed || len(o.Files) != 0 {
			t.Fatalf("nonliteral executable selection passed: %v %+v", err, o)
		}
	}
}
