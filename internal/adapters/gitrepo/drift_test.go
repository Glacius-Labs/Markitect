package gitrepo

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/apply"
	"github.com/Glacius-Labs/Markitect/internal/testkit"
)

func TestConfirmationDetectsSameSizeRewriteWithRestoredTimestamp(t *testing.T) {
	for _, kind := range []string{"ignored-census", "local-input", "external-input", "external-model"} {
		t.Run(kind, func(t *testing.T) {
			repo, selection, _ := libraryFixture(t)
			target := filepath.Join(repo.Dir, "ignored", "data.bin")
			repo.Write("ignored/data.bin", "original bytes")
			switch kind {
			case "local-input":
				target = filepath.Join(repo.Dir, "config", "execution.data")
			case "external-input":
				dir := testkit.TempDir(t)
				target = filepath.Join(dir, "profile.data")
				if err := os.WriteFile(target, []byte("external bytes"), 0o644); err != nil {
					t.Fatal(err)
				}
				selection.RuntimeProfiles = []apply.SourceLocation{{Boundary: dir, Path: fixturePath(t, "profile.data")}}
			case "external-model":
				dir := testkit.TempDir(t)
				copyBoundary(t, selection.Dependencies[0].Boundary, dir)
				selection.Dependencies[0].Boundary = dir
				target = filepath.Join(dir, "model", "software", "rule.yaml")
			}
			r := adapter(t)
			mutated := false
			r.observe = func(event, p string) {
				if event != "before-confirm" || mutated {
					return
				}
				mutated = true
				info, err := os.Stat(target)
				if err != nil {
					t.Fatal(err)
				}
				data, err := os.ReadFile(target)
				if err != nil {
					t.Fatal(err)
				}
				data[len(data)-1] ^= 1
				if err := os.WriteFile(target, data, info.Mode().Perm()); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(target, info.ModTime(), info.ModTime()); err != nil {
					t.Fatal(err)
				}
			}
			result, err := r.CapturePreview(context.Background(), apply.PreviewRequest{Selection: selection})
			if !mutated || !errors.Is(err, ErrChanged) || result.Head == "" || len(result.Raw.Local.Source.Entries) == 0 {
				t.Fatalf("restored timestamp hid drift: %v mutated=%v", err, mutated)
			}
		})
	}
}

func TestExternalModeAndCensusMembershipChangesAreDetected(t *testing.T) {
	for _, kind := range []string{"new-ignored-file", "external-mode"} {
		t.Run(kind, func(t *testing.T) {
			if kind == "external-mode" && runtime.GOOS == "windows" {
				t.Skip("Windows has no portable executable mode bit")
			}
			repo, selection, _ := libraryFixture(t)
			dir := testkit.TempDir(t)
			target := filepath.Join(dir, "instruction.txt")
			if err := os.WriteFile(target, []byte("original instructions"), 0o644); err != nil {
				t.Fatal(err)
			}
			selection.Instructions = []apply.SourceLocation{{Boundary: dir, Path: fixturePath(t, "instruction.txt")}}
			r := adapter(t)
			changed := false
			r.observe = func(event, p string) {
				if event != "before-confirm" || changed {
					return
				}
				changed = true
				if kind == "new-ignored-file" {
					repo.Write("ignored/new.bin", "new census member")
				} else if err := os.Chmod(target, 0o755); err != nil {
					t.Fatal(err)
				}
			}
			result, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
			if !changed || !errors.Is(err, ErrChanged) || result.Guard != "" {
				t.Fatalf("membership/mode drift escaped: %v", err)
			}
		})
	}
}

func TestFileIdentityChangeAndCancellationPreserveKnownInputs(t *testing.T) {
	for _, kind := range []string{"file-replacement", "cancel-after-roots", "head-change"} {
		t.Run(kind, func(t *testing.T) {
			repo, selection, head := libraryFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			r := adapter(t)
			changed := false
			r.observe = func(event, p string) {
				if changed {
					return
				}
				if kind == "cancel-after-roots" && event == "roots-validated" {
					changed = true
					cancel()
				}
				if kind == "head-change" && event == "before-confirm" {
					changed = true
					repo.Commit("New HEAD during observation")
				}
				if kind == "file-replacement" && event == "before-file-open" && strings.HasSuffix(filepath.ToSlash(p), "model/lending/loan.yaml") {
					changed = true
					info, err := os.Stat(p)
					if err != nil {
						t.Fatal(err)
					}
					data, err := os.ReadFile(p)
					if err != nil {
						t.Fatal(err)
					}
					if err := os.Rename(p, p+".old"); err != nil {
						t.Fatal(err)
					}
					if err := os.WriteFile(p, data, 0o644); err != nil {
						t.Fatal(err)
					}
					if err := os.Chtimes(p, info.ModTime(), info.ModTime()); err != nil {
						t.Fatal(err)
					}
					if err := os.Remove(p + ".old"); err != nil {
						t.Fatal(err)
					}
				}
			}
			result, err := r.CapturePreview(ctx, apply.PreviewRequest{Selection: selection})
			wanted := ErrChanged
			if kind == "cancel-after-roots" {
				wanted = context.Canceled
			}
			if !changed || !errors.Is(err, wanted) || result.Head != head || len(result.Raw.Local.Files.Entries()) == 0 {
				t.Fatalf("lost identity/cancellation facts: %v changed=%v head=%s", err, changed, result.Head)
			}
		})
	}
}

func TestAdmissionLimitsAndAlreadyCanceledContext(t *testing.T) {
	for _, limits := range []Limits{{Entries: -1}, {FileBytes: -1}, {TotalBytes: -1}, {Timeout: -1}, {FileBytes: 2, TotalBytes: 1}} {
		if _, err := New(limits); !errors.Is(err, ErrLimit) {
			t.Fatal("invalid acquisition budget accepted")
		}
	}
	_, selection, head := libraryFixture(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	result, err := adapter(t).Capture(ctx, apply.CaptureRequest{Selection: selection})
	if !errors.Is(err, context.Canceled) || result.Guard != "" || result.Head != "" {
		t.Fatal("canceled call performed acquisition")
	}
	for _, limits := range []Limits{{Entries: 2}, {FileBytes: 1024, TotalBytes: 2048}} {
		r, err := New(limits)
		if err != nil {
			t.Fatal(err)
		}
		result, err := r.Capture(context.Background(), apply.CaptureRequest{Selection: selection})
		if !errors.Is(err, ErrLimit) || result.Head != head || result.Guard != "" {
			t.Fatalf("admission lost observations: %v head=%s", err, result.Head)
		}
	}
}
