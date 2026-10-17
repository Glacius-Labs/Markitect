//go:build linux

package codexappserver

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"syscall"
	"testing"
	"time"
)

// A broken blocking-open implementation is confined to a test child. The
// parent owns a finite deadline and waits for termination; no stuck goroutine.
func TestLinuxOpenSwapCasesCompleteWithinFiniteChild(t *testing.T) {
	for _, scenario := range []string{"file-fifo", "file-symlink", "directory-fifo", "ancestor-symlink", "transient-directory-symlink"} {
		t.Run(scenario, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()
			command := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestLinuxOpenSwapChild$", "-test.timeout=8s")
			command.Env = append(os.Environ(), "MARKITECT_CODEX_METADATA_OPEN_SWAP="+scenario)
			output, err := command.CombinedOutput()
			if err != nil {
				t.Fatalf("finite native-open regression %s: %v %s", scenario, err, output)
			}
		})
	}
}

func TestLinuxCollidingSiblingAddedAfterAdmissionIsRefused(t *testing.T) {
	for _, which := range []string{"file", "ancestor"} {
		t.Run(which, func(t *testing.T) {
			f := fixture(t)
			d := newMetadataDisk(context.Background())
			defer d.close()
			added := false
			d.beforeOpen = func(stage, path string) {
				if added || stage != "before-file-open" || path != f.exe {
					return
				}
				added = true
				selected := path
				if which == "ancestor" {
					selected = filepath.Dir(path)
				}
				alias := filepath.Join(filepath.Dir(selected), strings.ToUpper(filepath.Base(selected)))
				if which == "file" {
					writeFixture(t, alias, "colliding sibling")
				} else {
					if err := os.Mkdir(alias, 0700); err != nil {
						t.Fatal(err)
					}
				}
			}
			_, _, err := d.readAbsolute(f.exe, maxExecutableBytes, false)
			if err == nil {
				err = d.verify()
			}
			if err == nil || !added {
				t.Fatal("late colliding sibling was accepted", err)
			}
		})
	}
}

func TestLinuxOpenSwapChild(t *testing.T) {
	scenario := os.Getenv("MARKITECT_CODEX_METADATA_OPEN_SWAP")
	if scenario == "" {
		return
	}
	f := fixture(t)
	d := newMetadataDisk(context.Background())
	defer d.close()
	swapped := false
	if scenario == "ancestor-symlink" || scenario == "transient-directory-symlink" {
		if _, err := d.root(f.stable); err != nil {
			t.Fatal(err)
		}
	}
	d.beforeOpen = func(stage, path string) {
		if scenario == "transient-directory-symlink" && stage == "after-enumeration-open" && path == f.stable && swapped {
			if err := os.Remove(f.stable); err != nil {
				t.Fatal(err)
			}
			if err := os.Rename(f.stable+"-old", f.stable); err != nil {
				t.Fatal(err)
			}
			return
		}
		wantStage, wantPath := "before-file-open", f.exe
		if scenario == "directory-fifo" {
			wantStage, wantPath = "before-directory-open", f.stable
		}
		if scenario == "ancestor-symlink" || scenario == "transient-directory-symlink" {
			wantStage, wantPath = "before-enumeration-open", f.stable
		}
		if swapped || stage != wantStage || path != wantPath {
			return
		}
		swapped = true
		if err := os.Rename(path, path+"-old"); err != nil {
			t.Fatal(err)
		}
		switch scenario {
		case "file-fifo", "directory-fifo":
			if err := syscall.Mkfifo(path, 0600); err != nil {
				t.Fatal(err)
			}
		case "file-symlink":
			if err := os.Symlink(path+"-old", path); err != nil {
				t.Fatal(err)
			}
		default:
			foreign := f.stable + "-foreign"
			writeFixture(t, filepath.Join(foreign, "foreign-name.json"), "{}")
			if err := os.Symlink(foreign, path); err != nil {
				t.Fatal(err)
			}
		}
	}
	if scenario == "ancestor-symlink" || scenario == "transient-directory-symlink" {
		names, err := d.names(f.stable)
		if !swapped || slices.Contains(names, "foreign-name.json") {
			t.Fatal("foreign directory was enumerated", names, err)
		}
		if scenario == "transient-directory-symlink" {
			if err != nil || !slices.Equal(names, []string{"schema.json", "unused.json"}) {
				t.Fatal("transient path swap redirected enumeration", names, err)
			}
		} else if err == nil {
			t.Fatal("ancestor symlink passed")
		}
	} else if scenario == "directory-fifo" {
		if _, err := d.root(f.stable); err == nil || !swapped {
			t.Fatal("directory FIFO swap passed", err)
		}
	} else {
		if _, _, err := d.readAbsolute(f.exe, maxExecutableBytes, false); err == nil || !swapped {
			t.Fatal("unsafe leaf swap passed", err)
		}
	}
}
