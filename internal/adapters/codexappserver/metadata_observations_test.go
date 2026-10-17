package codexappserver

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Count actual held-directory enumeration opens, rather than elapsed time or
// private cache entries. Increasing files in the same directories must not
// multiply scans of shared parents during either observation phase.
func TestMetadataDirectoryScansScaleWithDirectoriesNotFiles(t *testing.T) {
	for _, extraFiles := range []int{8, 768} {
		t.Run(fmt.Sprint(extraFiles), func(t *testing.T) {
			f := fixture(t)
			for i := 0; i < extraFiles; i++ {
				writeFixture(t, filepath.Join(f.stable, fmt.Sprintf("member-%04d.json", i)), "{}")
			}
			d := newMetadataDisk(context.Background())
			defer d.close()
			phase := 0
			scans := [2]map[string]int{{}, {}}
			d.beforeOpen = func(stage, path string) {
				if stage == "after-enumeration-open" {
					scans[phase][path]++
				}
			}
			inventory, complete, err := d.inventory(f.stable, []string{"schema.json"})
			if err != nil || !complete || len(inventory.Files) != extraFiles+2 {
				t.Fatalf("complete actual fixture inventory: %d %t %v", len(inventory.Files), complete, err)
			}
			phase = 1
			if err := d.verify(); err != nil {
				t.Fatal(err)
			}
			for phase, counts := range scans {
				for path, count := range counts {
					if count > 1 {
						t.Fatalf("phase %d enumerated shared directory %q %d times for %d files", phase, path, count, len(inventory.Files))
					}
				}
				for _, path := range []string{f.stable, filepath.Dir(f.stable), filepath.Dir(filepath.Dir(f.stable))} {
					if counts[path] != 1 {
						t.Fatalf("phase %d lacked fresh directory observation for %q: %d", phase, path, counts[path])
					}
				}
				t.Logf("files=%d phase=%d actual_directory_scans=%d", len(inventory.Files), phase, len(counts))
			}
		})
	}
}

func TestMetadataVerifyAllowsUnrelatedOrdinaryParentChanges(t *testing.T) {
	for _, change := range []string{"add", "remove"} {
		t.Run(change, func(t *testing.T) {
			f := fixture(t)
			unrelated := filepath.Join(filepath.Dir(f.stable), "unrelated-temp-owner-file")
			if change == "remove" {
				writeFixture(t, unrelated, "belongs to another caller")
			}
			d := newMetadataDisk(context.Background())
			defer d.close()
			if _, _, err := d.readAbsolute(f.exe, maxExecutableBytes, false); err != nil {
				t.Fatal(err)
			}
			if change == "add" {
				writeFixture(t, unrelated, "belongs to another caller")
			} else if err := os.Remove(unrelated); err != nil {
				t.Fatal(err)
			}
			if err := d.verify(); err != nil {
				t.Fatalf("unrelated ordinary parent mutation refused: %v", err)
			}
		})
	}
}

func TestMetadataVerifyRefusesRelevantFinalNamespaceChanges(t *testing.T) {
	for _, change := range []string{"file-case", "ancestor-case", "inventory-add", "inventory-remove", "inventory-case", "file-size", "file-replacement"} {
		t.Run(change, func(t *testing.T) {
			f := fixture(t)
			d := newMetadataDisk(context.Background())
			defer d.close()
			if _, _, err := d.readAbsolute(f.exe, maxExecutableBytes, false); err != nil {
				t.Fatal(err)
			}
			if _, complete, err := d.inventory(f.stable, []string{"schema.json"}); err != nil || !complete {
				t.Fatal("fixture inventory", complete, err)
			}
			if err := d.verify(); err != nil {
				t.Fatal("initial unchanged verification", err)
			}
			switch change {
			case "inventory-add":
				writeFixture(t, filepath.Join(f.stable, "new-member.json"), "{}")
			case "inventory-remove":
				if err := os.Remove(filepath.Join(f.stable, "unused.json")); err != nil {
					t.Fatal(err)
				}
			case "file-size":
				writeFixture(t, f.exe, "different bytes and size")
			case "file-replacement":
				if err := os.Rename(f.exe, f.exe+"-old"); err != nil {
					t.Fatal(err)
				}
				writeFixture(t, f.exe, "fixture executable\r\n")
			default:
				selected := f.exe
				if change == "ancestor-case" {
					selected = filepath.Dir(f.exe)
				} else if change == "inventory-case" {
					selected = filepath.Join(f.stable, "unused.json")
				}
				target := filepath.Join(filepath.Dir(selected), strings.ToUpper(filepath.Base(selected)))
				if err := os.Rename(selected, selected+"-temporary"); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(selected+"-temporary", target); err != nil {
					t.Fatal(err)
				}
			}
			if err := d.verify(); err == nil {
				t.Fatal("relevant final mutation passed")
			}
		})
	}
}
