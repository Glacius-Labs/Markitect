package codexappserver

import (
	"context"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestDirectoryOpenSeamRejectsSameBytesAtReplacementDirectory(t *testing.T) {
	f := fixture(t)
	d := newMetadataDisk(context.Background())
	defer d.close()
	swapped := false
	d.beforeOpen = func(stage, path string) {
		if stage != "before-directory-open" || path != f.stable || swapped {
			return
		}
		swapped = true
		if err := os.Rename(f.stable, f.stable+"-old"); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, filepath.Join(f.stable, "schema.json"), `{"type":"object"}`)
	}
	if _, err := d.root(f.stable); err == nil || !swapped {
		t.Fatal("directory replacement crossed held-handle admission", err)
	}
}

func TestOpenSeamsRefuseCaseOnlyNamespaceDrift(t *testing.T) {
	for _, which := range []string{"file", "ancestor", "enumeration"} {
		t.Run(which, func(t *testing.T) {
			f := fixture(t)
			d := newMetadataDisk(context.Background())
			defer d.close()
			if which == "enumeration" {
				if _, err := d.root(f.stable); err != nil {
					t.Fatal(err)
				}
			}
			swapped := false
			d.beforeOpen = func(stage, path string) {
				wantStage, wantPath := "before-file-open", f.exe
				if which == "enumeration" {
					wantStage, wantPath = "before-enumeration-open", f.stable
				}
				if swapped || stage != wantStage || path != wantPath {
					return
				}
				swapped = true
				selected := path
				if which == "ancestor" {
					selected = filepath.Dir(path)
				}
				target := filepath.Join(filepath.Dir(selected), strings.ToUpper(filepath.Base(selected)))
				if err := os.Rename(selected, selected+"-temporary"); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(selected+"-temporary", target); err != nil {
					t.Fatal(err)
				}
			}
			var err error
			if which == "enumeration" {
				_, err = d.names(f.stable)
			} else {
				_, _, err = d.readAbsolute(f.exe, maxExecutableBytes, false)
			}
			if err == nil {
				err = d.verify()
			}
			if err == nil || !swapped {
				t.Fatal("case-only namespace drift passed", err)
			}
		})
	}
}

func TestEnumerationSeamCannotReadReplacementDirectoryNames(t *testing.T) {
	f := fixture(t)
	d := newMetadataDisk(context.Background())
	defer d.close()
	if _, err := d.root(f.stable); err != nil {
		t.Fatal(err)
	}
	swapped := false
	d.beforeOpen = func(stage, path string) {
		if stage != "before-enumeration-open" || path != f.stable || swapped {
			return
		}
		swapped = true
		if err := os.Rename(f.stable, f.stable+"-old"); err != nil {
			t.Fatal(err)
		}
		writeFixture(t, filepath.Join(f.stable, "foreign-name.json"), "{}")
	}
	names, err := d.names(f.stable)
	if err == nil || !swapped || slices.Contains(names, "foreign-name.json") {
		t.Fatal("replacement directory was enumerated", names, err)
	}
}

func TestEnumerationAlwaysUsesRetainedDirectoryHandle(t *testing.T) {
	f := fixture(t)
	d := newMetadataDisk(context.Background())
	defer d.close()
	if _, err := d.root(f.stable); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		names, err := d.names(f.stable)
		if err != nil || !slices.Equal(names, []string{"schema.json", "unused.json"}) {
			t.Fatal("held directory enumeration cursor/state", names, err)
		}
	}
}
