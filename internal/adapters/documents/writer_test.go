package documents

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/guardedwrite"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestMain(m *testing.M) { testkit.Main(m) }

func TestWriterRechecksTargetAndRetainsObservedPartialEffects(t *testing.T) {
	for _, tc := range []string{"pass", "changed target", "refused plan", "postwrite failure"} {
		t.Run(tc, func(t *testing.T) {
			fixture := testkit.NewRepo(t)
			fixture.Git("checkout", "-b", "codex/document-writer-fixture")
			root := fixture.Dir
			const path = "view.md"
			full := filepath.Join(root, path)
			if err := os.WriteFile(full, []byte("old"), 0644); err != nil {
				t.Fatal(err)
			}
			fixture.Commit("reviewed documentation target")
			w := Writer{}
			target, err := w.Observe(context.Background(), root, path)
			if err != nil {
				t.Fatal(err)
			}
			if tc == "changed target" {
				if err := os.WriteFile(full, []byte("new competing bytes"), 0644); err != nil {
					t.Fatal(err)
				}
			}
			validate := func() error {
				if tc == "refused plan" {
					return errors.New("source changed")
				}
				return nil
			}
			post := func() error {
				if tc == "postwrite failure" {
					return errors.New("source changed after effect")
				}
				return nil
			}
			result, err := w.Write(context.Background(), root, target, []byte("generated"), validate, post)
			if (err == nil) != (tc == "pass") {
				t.Fatalf("%s: result=%+v err=%v", tc, result, err)
			}
			expected := []byte("old")
			var paths []string
			switch tc {
			case "pass", "postwrite failure":
				expected = []byte("generated")
				paths = []string{path}
			case "changed target":
				expected = []byte("new competing bytes")
			}
			got, readErr := os.ReadFile(full)
			if readErr != nil || !bytes.Equal(got, expected) || !slices.Equal(result.CompletedPaths, paths) {
				t.Fatalf("bytes=%q paths=%v err=%v", got, result.CompletedPaths, readErr)
			}
		})
	}
}

func TestOversizedDocumentAndGuardedReadAreBounded(t *testing.T) {
	root := t.TempDir()
	full := filepath.Join(root, "large")
	file, err := os.Create(full)
	if err != nil {
		t.Fatal(err)
	}
	if err := file.Truncate((64 << 20) + 1); err != nil {
		file.Close()
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := (Writer{}).Observe(context.Background(), root, "large"); err == nil {
		t.Fatal("oversized target accepted")
	}
	anchored, err := guardedwrite.OpenRoot(root)
	if err != nil {
		t.Fatal(err)
	}
	defer anchored.Close()
	if _, err := anchored.ReadFile("large"); err == nil {
		t.Fatal("guarded read allocated an oversized race-grown target")
	}
}
