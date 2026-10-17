package codexappserver

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestClosedMetadataFormatRejectsUnsupportedFieldsAndAliases(t *testing.T) {
	f := fixture(t)
	for _, tc := range []struct{ name, from, to string }{
		{"version", "profileVersion: 1", "profileVersion: 2"},
		{"version string", "profileVersion: 1", "profileVersion: '1'"},
		{"unknown pin", metadataPinName, "other-pin"},
		{"runtime field", "instructions: []", "instructions: []\nmodel: chosen"},
		{"config passthrough", "instructions: []", "instructions: []\nconfig: {}"},
		{"missing list", "instructions: []", ""},
		{"wrong list", "instructions: []", "instructions: {}"},
		{"unknown schema field", "schemas: {", "schemas: {unknown: other, "},
		{"relative executable", f.exe, "program"},
		{"dot alias", f.exe, filepath.Dir(f.exe) + string(filepath.Separator) + "." + string(filepath.Separator) + filepath.Base(f.exe)},
		{"trailing separator", f.stable, f.stable + string(filepath.Separator)},
		{"same inventories", f.experimental, f.stable},
		{"nested inventories", f.experimental, filepath.Join(f.stable, "nested")},
		{"instruction extra", "instructions: []", "instructions: [{boundary: '.', path: x, unknown: true}]"},
		{"instruction path", "instructions: []", "instructions: [{boundary: '.', path: ../x}]"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			input := profileInput(t, strings.Replace(f.inline(), tc.from, tc.to, 1), nil)
			out, err := prepareMetadata(context.Background(), input, f.pin)
			if err != nil || out.Status != model.Failed || len(out.Files) != 0 || len(out.Inventories) != 0 {
				t.Fatalf("closed decoder: %v %+v", err, out)
			}
			for _, diagnostic := range out.Diagnostics {
				if diagnostic.Source.ContentID != input.Document().File.ContentID() || !strings.HasPrefix(diagnostic.Source.Field, input.Field()) {
					t.Fatal("source binding lost")
				}
			}
		})
	}
}

func TestChangedMissingAndUnexpectedFilesRetainActualEvidence(t *testing.T) {
	for _, tc := range []struct {
		name   string
		want   model.Status
		change func(*testing.T, metadataFixture)
	}{
		{"changed executable", model.Failed, func(t *testing.T, f metadataFixture) { writeFixture(t, f.exe, "changed executable") }},
		{"missing executable", model.NotCompleted, func(t *testing.T, f metadataFixture) {
			if err := os.Remove(f.exe); err != nil {
				t.Fatal(err)
			}
		}},
		{"added schema", model.Failed, func(t *testing.T, f metadataFixture) { writeFixture(t, filepath.Join(f.stable, "extra.json"), "{}") }},
		{"extra empty directory", model.Failed, func(t *testing.T, f metadataFixture) {
			if err := os.Mkdir(filepath.Join(f.stable, "extra-empty"), 0700); err != nil {
				t.Fatal(err)
			}
		}},
		{"deleted unused schema", model.Failed, func(t *testing.T, f metadataFixture) {
			if err := os.Remove(filepath.Join(f.stable, "unused.json")); err != nil {
				t.Fatal(err)
			}
		}},
		{"deleted selected schema", model.Failed, func(t *testing.T, f metadataFixture) {
			if err := os.Remove(filepath.Join(f.stable, "schema.json")); err != nil {
				t.Fatal(err)
			}
		}},
		{"invalid selected JSON", model.Failed, func(t *testing.T, f metadataFixture) {
			writeFixture(t, filepath.Join(f.stable, "schema.json"), "not JSON")
		}},
		{"missing inventory", model.NotCompleted, func(t *testing.T, f metadataFixture) {
			if err := os.Rename(f.stable, f.stable+"-moved"); err != nil {
				t.Fatal(err)
			}
		}},
		{"executable directory", model.Failed, func(t *testing.T, f metadataFixture) {
			if err := os.Remove(f.exe); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(f.exe, 0700); err != nil {
				t.Fatal(err)
			}
		}},
		{"schema byte bound", model.NotCompleted, func(t *testing.T, f metadataFixture) {
			writeFixture(t, filepath.Join(f.stable, "unused.json"), strings.Repeat("x", int(maxSchemaFileBytes+1)))
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := fixture(t)
			input := profileInput(t, f.inline(), nil)
			tc.change(t, f)
			out, _ := prepareMetadata(context.Background(), input, f.pin)
			if out.Status != tc.want || len(out.Diagnostics) == 0 || len(out.Inventories) != 2 {
				t.Fatalf("failure evidence: %+v", out)
			}
			if len(out.Inventories[1].Files) != 2 {
				t.Fatal("independent known inventory was discarded")
			}
			if tc.name == "added schema" && len(out.Inventories[0].Files) != 3 {
				t.Fatal("unexpected member was skipped")
			}
		})
	}
}

func TestInstructionsSelectCapturedCoordinatesWithoutFilesystemReads(t *testing.T) {
	f := fixture(t)
	boundary := filepath.Join(testkit.TempDir(t), "deliberately-absent-instruction-directory")
	location := apply.SourceLocation{Boundary: boundary, Path: testPath(t, "private.md")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("captured original\x00\r\n"))
	if err != nil {
		t.Fatal(err)
	}
	instructions := []apply.SourceFile{{Location: location, File: entry}}
	list := "instructions: [{boundary: '" + boundary + "', path: private.md}, {boundary: '" + boundary + "', path: private.md}]"
	inline := strings.Replace(f.inline(), "instructions: []", list, 1)
	input := profileInput(t, inline, instructions)
	out, err := prepareMetadata(context.Background(), input, f.pin)
	if err != nil || out.Status != model.Passed || !reflect.DeepEqual(out.Instructions, []apply.SourceLocation{location, location}) {
		t.Fatalf("captured ordered repeats: %v %+v", err, out)
	}
	uncaptured := profileInput(t, inline, nil)
	missing, err := prepareMetadata(context.Background(), uncaptured, f.pin)
	if err != nil || missing.Status != model.NotCompleted || !reflect.DeepEqual(missing.RequiredInstructions, []apply.SourceLocation{location, location}) || len(missing.Instructions) != 0 || len(missing.Files) != 1 {
		t.Fatalf("uncaptured instructions: %v %+v", err, missing)
	}
}

func TestOpaqueCapturedInstructionBoundaryIsNeverInterpretedAsPath(t *testing.T) {
	f := fixture(t)
	location := apply.SourceLocation{Boundary: "opaque:source-coordinate", Path: testPath(t, "instruction.md")}
	entry, err := snapshot.NewEntry(location.Path, snapshot.Regular, []byte("captured bytes"))
	if err != nil {
		t.Fatal(err)
	}
	inline := strings.Replace(f.inline(), "instructions: []", "instructions: [{boundary: 'opaque:source-coordinate', path: instruction.md}]", 1)
	input := profileInput(t, inline, []apply.SourceFile{{Location: location, File: entry}})
	out, err := prepareMetadata(context.Background(), input, f.pin)
	if err != nil || out.Status != model.Passed || !reflect.DeepEqual(out.Instructions, []apply.SourceLocation{location}) {
		t.Fatal("opaque captured coordinate was resolved", err, out)
	}
}

func TestInputAndOutputCopiesAndPrivateLocationsRemainBound(t *testing.T) {
	f := fixture(t)
	input := profileInput(t, f.inline(), nil)
	node := input.Inline()
	node.Content = nil
	out, err := prepareMetadata(context.Background(), input, f.pin)
	if err != nil || out.Status != model.Passed {
		t.Fatal(err, out)
	}
	if out.Files[0].Location.Boundary != filepath.Dir(f.exe) || out.Files[0].Location.Path.String() != filepath.Base(f.exe) || out.Inventories[0].Boundary != f.stable {
		t.Fatal("fabricated metadata locations")
	}
	out.Files[0].Size = 999
	out.Inventories[0].Files[0].Size = 999
	next, err := prepareMetadata(context.Background(), input, f.pin)
	if err != nil || next.Status != model.Passed || next.Files[0].Size == 999 || next.Inventories[0].Files[0].Size == 999 {
		t.Fatal("result or node mutation escaped")
	}
	other := fixture(t)
	otherInput := profileInput(t, other.inline(), nil)
	otherOut, err := prepareMetadata(context.Background(), otherInput, other.pin)
	if err != nil || otherOut.InputBinding == next.InputBinding || otherOut.Files[0].Location == next.Files[0].Location {
		t.Fatal("private coordinates/request binding collapsed")
	}
}

type finiteCancelContext struct {
	context.Context
	remaining int
}

func (c *finiteCancelContext) Err() error {
	c.remaining--
	if c.remaining <= 0 {
		return context.Canceled
	}
	return nil
}

func TestCancellationRetainsKnownMetadataWithoutFalseInvalidity(t *testing.T) {
	f := fixture(t)
	input := profileInput(t, f.inline(), nil)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	out, err := prepareMetadata(ctx, input, f.pin)
	if out.Status != model.NotCompleted || !errors.Is(err, context.Canceled) || len(out.Files) != 0 {
		t.Fatal("initial cancellation", err, out)
	}
	// A deterministic context cancellation at an acquisition checkpoint, no
	// timing assumptions or helper processes.
	partial, err := prepareMetadata(&finiteCancelContext{Context: context.Background(), remaining: 80}, input, f.pin)
	if partial.Status != model.NotCompleted || !errors.Is(err, context.Canceled) {
		t.Fatal("mid acquisition cancellation", err, partial)
	}
}

func TestNativeCaseAliasIsRefused(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("native case-insensitive path diagnostic")
	}
	f := fixture(t)
	alias := filepath.Join(filepath.Dir(f.exe), "PROGRAM")
	out, _ := prepareMetadata(context.Background(), profileInput(t, strings.Replace(f.inline(), f.exe, alias, 1), nil), f.pin)
	if out.Status != model.Failed || len(out.Files) != 0 {
		t.Fatal("case alias accepted", out)
	}
}

func TestSchemaInventoryCannotIgnoreNestedUnselectedMembers(t *testing.T) {
	f := fixture(t)
	writeFixture(t, filepath.Join(f.stable, "nested", "unselected.json"), "{}\n")
	out, _ := prepareMetadata(context.Background(), profileInput(t, f.inline(), nil), f.pin)
	if out.Status != model.Failed || len(out.Inventories[0].Files) != 3 {
		t.Fatal("nested addition omitted", out)
	}
}
