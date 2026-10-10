//go:build !windows

package host

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestAnchoredExclusiveCreationReportsPublishedObjectsAfterParentSwap(t *testing.T) {
	for _, tc := range []struct {
		name    string
		leaf    string
		wantDir bool
		call    func(*writeRoot, string, func() error) error
	}{
		{
			name:    "mkdir",
			leaf:    "new-area",
			wantDir: true,
			call: func(root *writeRoot, path string, inject func() error) error {
				return root.mkdirWithHook(path, 0755, inject)
			},
		},
		{
			name: "exclusive-file",
			leaf: "new-file",
			call: func(root *writeRoot, path string, inject func() error) error {
				_, err := root.createExclusiveWithHook(path, 0644, inject)
				return err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			base, err := os.MkdirTemp(os.TempDir(), "markitect-exclusive-parent-swap-")
			if err != nil {
				t.Fatal(err)
			}
			defer os.RemoveAll(base)
			rootPath := filepath.Join(base, "repo")
			parent := filepath.Join(rootPath, "target")
			sibling := filepath.Join(rootPath, "sibling")
			if err := os.MkdirAll(parent, 0755); err != nil {
				t.Fatal(err)
			}
			if err := os.Mkdir(sibling, 0755); err != nil {
				t.Fatal(err)
			}
			sentinel := filepath.Join(sibling, "sentinel.txt")
			if err := os.WriteFile(sentinel, []byte("keep\n"), 0644); err != nil {
				t.Fatal(err)
			}
			root, err := openWriteRoot(rootPath)
			if err != nil {
				t.Fatal(err)
			}
			defer root.Close()
			original := filepath.Join(rootPath, "target-original")
			var injected bool
			err = tc.call(root, "target/"+tc.leaf, func() error {
				if err := os.Rename(parent, original); err != nil {
					return err
				}
				if err := os.Symlink(sibling, parent); err != nil {
					return err
				}
				injected = true
				return nil
			})
			if !injected {
				t.Skipf("parent symlink swap could not be injected: %v", err)
			}
			var published *publishedWriteError
			if !errors.As(err, &published) || published.Path != "target/"+tc.leaf {
				t.Fatalf("expected published-object error, got %v", err)
			}
			entries, err := os.ReadDir(sibling)
			if err != nil || len(entries) != 1 || entries[0].Name() != "sentinel.txt" {
				t.Fatalf("sibling received a created object: entries=%v error=%v", entries, err)
			}
			got, err := os.ReadFile(sentinel)
			if err != nil || string(got) != "keep\n" {
				t.Fatalf("sibling sentinel changed: bytes=%q error=%v", got, err)
			}
			info, err := os.Stat(filepath.Join(original, tc.leaf))
			if err != nil || info.IsDir() != tc.wantDir {
				t.Fatalf("published object missing or wrong type: info=%v error=%v", info, err)
			}
		})
	}
}
