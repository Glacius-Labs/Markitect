//go:build windows

package guardedwrite

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestGuardedWriteAcceptsShortAndLongRootSpellings(t *testing.T) {
	root, shortRoot := windowsTestPath(t)
	runGit(t, root, "init", "-b", "codex/short-root")
	for _, captureRoot := range []string{root, shortRoot} {
		for _, applyRoot := range []string{root, shortRoot} {
			t.Run(captureRoot+"/"+applyRoot, func(t *testing.T) {
				capture, err := CaptureFiles(captureRoot, []string{"result.txt"})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := Apply(applyRoot, capture, []Change{{Path: "result.txt", Bytes: []byte("updated\n"), Mode: 0644}}); err != nil {
					t.Fatalf("apply with equivalent root spelling: %v", err)
				}
				if got := string(mustRead(t, filepath.Join(root, "result.txt"))); got != "updated\n" {
					t.Fatalf("written bytes = %q", got)
				}
				if err := os.Remove(filepath.Join(root, "result.txt")); err != nil {
					t.Fatal(err)
				}
			})
		}
	}
}

// Lexical checks see only the requested spelling. Windows also resolves an
// existing entry through its 8.3 short name or another case, so such an alias
// would let a guarded write reach .git or .markitect.
func TestGuardedWriteRefusesWindowsAliasesOfExistingEntries(t *testing.T) {
	root := testRepo(t, "feature/aliases")
	for _, directory := range []string{".markitect", "docs/generated-protos"} {
		if err := os.MkdirAll(filepath.Join(root, filepath.FromSlash(directory)), 0755); err != nil {
			t.Fatal(err)
		}
	}
	hook := filepath.Join(root, ".git", "hooks", "pre-commit")
	runtimeFile := filepath.Join(root, ".markitect", "runtime.yaml")
	if _, err := CaptureFiles(root, []string{".git/hooks/pre-commit"}); err == nil {
		t.Fatal("CaptureFiles accepted the literal .git path")
	}
	writer, err := OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	// One check for the whole tree: a proven directory must not excuse
	// another spelling of it.
	stored := NewStoredNames(os.DirFS(root))
	for _, name := range []string{"README.md", "docs/generated-protos/schema.md", "notes~1/new.md"} {
		if err := stored.Require(name); err != nil {
			t.Errorf("StoredNames refused %q: %v", name, err)
		}
	}
	refuse := func(t *testing.T, aliases []string) {
		t.Helper()
		for _, alias := range aliases {
			capture, err := CaptureFiles(root, []string{alias})
			if err == nil {
				result, applyErr := Apply(root, capture, []Change{{Path: alias, Bytes: []byte("#!/bin/sh\necho alias\n"), Mode: 0755}})
				t.Errorf("CaptureFiles accepted alias %q; Apply completed %v, err %v", alias, result.CompletedPaths, applyErr)
			}
			// The anchored root refuses them too, also on paths that skip checkPath.
			if err := writer.AtomicWrite(alias, []byte("alias\n"), 0644); err == nil {
				t.Errorf("AtomicWrite accepted alias %q", alias)
			}
			if file, err := writer.CreateExclusive(alias, 0644); err == nil {
				file.Close()
				t.Errorf("CreateExclusive accepted alias %q", alias)
			}
			if err := stored.Require(alias); err == nil {
				t.Errorf("StoredNames accepted alias %q", alias)
			}
		}
	}
	// Every NTFS volume resolves case variants, so only the 8.3 cases skip.
	t.Run("case variants", func(t *testing.T) {
		refuse(t, []string{"Docs/generated-protos/schema.md", "readme.md"})
	})
	t.Run("8.3 short names", func(t *testing.T) {
		gitAlias := windowsShortLeaf(t, filepath.Join(root, ".git"))
		markitectAlias := windowsShortLeaf(t, filepath.Join(root, ".markitect"))
		protosAlias := windowsShortLeaf(t, filepath.Join(root, "docs", "generated-protos"))
		refuse(t, []string{gitAlias + "/hooks/pre-commit", markitectAlias + "/runtime.yaml", "docs/" + protosAlias + "/schema.md"})
	})
	for _, target := range []string{hook, runtimeFile, filepath.Join(root, "docs", "generated-protos", "schema.md")} {
		if _, err := os.Lstat(target); !os.IsNotExist(err) {
			t.Errorf("alias write reached %s: %v", target, err)
		}
	}
	if got := string(mustRead(t, filepath.Join(root, "README.md"))); got != "test consumer\n" {
		t.Errorf("alias write changed README.md to %q", got)
	}

	// New names alias nothing, so an 8.3-shaped new name stays writable.
	capture, err := CaptureFiles(root, []string{"README.md", "notes~1/new.md"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Apply(root, capture, []Change{{Path: "README.md", Bytes: []byte("updated\n"), Mode: 0644}, {Path: "notes~1/new.md", Bytes: []byte("new\n"), Mode: 0644}}); err != nil {
		t.Fatalf("apply exact and new names: %v", err)
	}
	if got := string(mustRead(t, filepath.Join(root, "notes~1", "new.md"))); got != "new\n" {
		t.Fatalf("new 8.3-shaped path bytes = %q", got)
	}
}

// windowsShortLeaf returns the 8.3 short name of path's final component and
// skips the calling subtest when the volume generates none.
func windowsShortLeaf(t *testing.T, path string) string {
	t.Helper()
	short, err := windowsShortPath(path)
	if err != nil {
		t.Fatalf("short name of %s: %v", path, err)
	}
	leaf := filepath.Base(short)
	if strings.EqualFold(leaf, filepath.Base(path)) {
		t.Skipf("volume generates no 8.3 short names for %s", path)
	}
	return leaf
}
