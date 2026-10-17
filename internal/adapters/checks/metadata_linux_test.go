//go:build linux

package checks

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestLinuxProcessMetadataRejectsLeafAndAncestorSymlinksAndFIFO(t *testing.T) {
	for _, tc := range []string{"leaf", "dangling", "ancestor", "fifo"} {
		t.Run(tc, func(t *testing.T) {
			directory := testkit.TempDir(t)
			bytes := []byte("ordinary actual file")
			target := filepath.Join(directory, "target.exe")
			if err := os.WriteFile(target, bytes, 0700); err != nil {
				t.Fatal(err)
			}
			executable := filepath.Join(directory, "check.exe")
			switch tc {
			case "leaf":
				if err := os.Symlink(target, executable); err != nil {
					t.Fatal(err)
				}
			case "dangling":
				if err := os.Symlink(filepath.Join(directory, "missing.exe"), executable); err != nil {
					t.Fatal(err)
				}
			case "ancestor":
				actual := filepath.Join(directory, "actual")
				if err := os.Mkdir(actual, 0700); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(actual, "check.exe"), bytes, 0700); err != nil {
					t.Fatal(err)
				}
				link := filepath.Join(directory, "link")
				if err := os.Symlink(actual, link); err != nil {
					t.Fatal(err)
				}
				executable = filepath.Join(link, "check.exe")
			case "fifo":
				if err := syscall.Mkfifo(executable, 0600); err != nil {
					t.Fatal(err)
				}
			}
			o, err := (MetadataPreparer{}).PrepareCheckProfile(context.Background(), processInput(t, processFields(executable, bytes)))
			if o.Status != model.Failed || !errors.Is(err, ErrUnsafeExecutable) || len(o.Files) != 0 {
				t.Fatalf("redirect/special file accepted: %v %+v", err, o)
			}
		})
	}
}

func TestLinuxProcessMetadataRefusesReplacementWithoutBlockingOpen(t *testing.T) {
	for _, tc := range []string{"symlink", "fifo", "same bytes new inode"} {
		t.Run(tc, func(t *testing.T) {
			directory := testkit.TempDir(t)
			executable := filepath.Join(directory, "check.exe")
			bytes := []byte("original bytes")
			if err := os.WriteFile(executable, bytes, 0700); err != nil {
				t.Fatal(err)
			}
			p := MetadataPreparer{observe: func(stage string) {
				if stage != "before-open" {
					return
				}
				if err := os.Rename(executable, executable+".original"); err != nil {
					t.Fatal(err)
				}
				switch tc {
				case "symlink":
					if err := os.Symlink(executable+".original", executable); err != nil {
						t.Fatal(err)
					}
				case "fifo":
					if err := syscall.Mkfifo(executable, 0600); err != nil {
						t.Fatal(err)
					}
				case "same bytes new inode":
					if err := os.WriteFile(executable, bytes, 0700); err != nil {
						t.Fatal(err)
					}
				}
			}}
			o, err := p.PrepareCheckProfile(context.Background(), processInput(t, processFields(executable, bytes)))
			if err == nil || o.Status != model.NotCompleted || len(o.Files) != 0 {
				t.Fatalf("changed path accepted: %v %+v", err, o)
			}
		})
	}
}

func TestLinuxProcessMetadataDetectsAncestorReplacementAndObservesMode(t *testing.T) {
	directory := testkit.TempDir(t)
	parent := filepath.Join(directory, "bin")
	if err := os.Mkdir(parent, 0700); err != nil {
		t.Fatal(err)
	}
	executable := filepath.Join(parent, "check.exe")
	bytes := []byte("same file bytes")
	if err := os.WriteFile(executable, bytes, 0700); err != nil {
		t.Fatal(err)
	}
	input := processInput(t, processFields(executable, bytes))
	o, err := (MetadataPreparer{}).PrepareCheckProfile(context.Background(), input)
	if err != nil || o.Status != model.Passed || o.Files[0].Mode != snapshot.Executable {
		t.Fatalf("actual executable mode lost: %v %+v", err, o)
	}
	p := MetadataPreparer{observe: func(stage string) {
		if stage != "after-read" {
			return
		}
		if err := os.Rename(parent, parent+".original"); err != nil {
			t.Fatal(err)
		}
		if err := os.Mkdir(parent, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(executable, bytes, 0700); err != nil {
			t.Fatal(err)
		}
	}}
	o, err = p.PrepareCheckProfile(context.Background(), input)
	if err == nil || o.Status != model.NotCompleted || len(o.Files) != 1 {
		t.Fatalf("ancestor replacement lost partial refusal: %v %+v", err, o)
	}
}
