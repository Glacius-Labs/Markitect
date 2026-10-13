package agentexec

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSnapshotEmptyInputRetainsExactFactsAndDetectsGrowth(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "empty.txt")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	original, err := os.Lstat(path)
	if err != nil {
		t.Fatal(err)
	}
	before, err := snapshotRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, fact := range before.files {
		if fact.path == path {
			found = true
			if fact.kind != "file" || fact.size != 0 || fact.digest != digest(nil) || fact.mode != uint32(original.Mode().Perm()) || fact.modTime != original.ModTime().UnixNano() {
				t.Fatalf("empty input fact was omitted or changed: %+v", fact)
			}
		}
	}
	if !found || before.totalBytes != 0 {
		t.Fatal("empty input was excluded from normal facts/counting")
	}
	if err := os.WriteFile(path, []byte("grown"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := snapshotEmptyInputFile(path, original); err == nil {
		t.Fatal("growth between original Lstat and open was accepted")
	}
	after, err := snapshotRoots([]string{root})
	if err != nil {
		t.Fatal(err)
	}
	if sameSnapshot(before, after) || after.totalBytes != 5 {
		t.Fatal("existing full input-change guard missed growth")
	}
}

func TestSnapshotEmptyInputRejectsReplacementAndSymlink(t *testing.T) {
	for _, change := range []string{"replacement", "symlink", "missing", "mtime"} {
		t.Run(change, func(t *testing.T) {
			root := t.TempDir()
			path := filepath.Join(root, "empty.txt")
			if err := os.WriteFile(path, nil, 0600); err != nil {
				t.Fatal(err)
			}
			original, err := os.Lstat(path)
			if err != nil {
				t.Fatal(err)
			}
			if !os.SameFile(original, original) {
				t.Fatal("original identity unavailable")
			}
			before, err := snapshotRoots([]string{root})
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "replacement":
				replacement := filepath.Join(root, "replacement.txt")
				if err := os.WriteFile(replacement, nil, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Rename(replacement, path); err != nil {
					t.Fatal(err)
				}
				_ = os.Chtimes(path, original.ModTime(), original.ModTime())
			case "symlink":
				other := filepath.Join(root, "other.txt")
				if err := os.WriteFile(other, nil, 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(other, path); err != nil {
					t.Skipf("symlink privilege unavailable: %v", err)
				}
			case "missing":
				if err := os.Remove(path); err != nil {
					t.Fatal(err)
				}
			case "mtime":
				if err := os.Chtimes(path, original.ModTime().Add(1_000_000_000), original.ModTime().Add(1_000_000_000)); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := snapshotEmptyInputFile(path, original); err == nil {
				t.Fatal("changed empty input was accepted")
			}
			after, err := snapshotRoots([]string{root})
			if err != nil {
				t.Fatal(err)
			}
			// Identity-only replacement is rejected by the helper, and growth/type/
			// deletion/time changes remain covered by the existing complete snapshot.
			if change != "replacement" && sameSnapshot(before, after) {
				t.Fatal("full input-change guard missed changed empty input")
			}
		})
	}
}
