package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestExportProofsAreHashedEvidenceNotRescoring(t *testing.T) {
	arena := t.TempDir()
	run, task := "fixture", "01"
	for _, dir := range []string{"runs/" + run, "raw/" + run} {
		if err := os.MkdirAll(filepath.Join(arena, filepath.FromSlash(dir)), 0700); err != nil {
			t.Fatal(err)
		}
	}
	evaluation := []byte("task: '01'\nstatus: manual-review-required\nchecks:\n  - name: requires-review\n    status: manual\n")
	if err := os.WriteFile(filepath.Join(arena, "runs", run, task+".evaluation.yaml"), evaluation, 0600); err != nil {
		t.Fatal(err)
	}
	expected := map[string]string{}
	for _, phase := range []string{"final", "before-repair"} {
		ref := "raw/" + run + "/" + task + "." + phase + ".evaluation-export.yaml"
		path := filepath.Join(arena, filepath.FromSlash(ref))
		// A supplied claim cannot turn a manual result into passed or grant reads.
		if err := os.WriteFile(path, []byte("raw_evaluation: '/unreadable/outside/arena'\nstatus: passed\n"), 0600); err != nil {
			t.Fatal(err)
		}
		hash, err := hashFile(path)
		if err != nil {
			t.Fatal(err)
		}
		expected[ref] = "sha256:" + hash
	}
	n := nativeRecord{RunID: run, Project: "p", Arm: "a", Trial: 1, TaskID: task, ActorStatus: "completed"}
	got, err := analyzeTask(arena, n, taskCard{ID: task}, true)
	if err != nil {
		t.Fatal(err)
	}
	if got.EvaluationStatus != "manual" || got.CountedOutcome != "manual" {
		t.Fatalf("sidecar reinterpreted policy outcome: %#v", got)
	}
	for ref, hash := range expected {
		if got.RawReferenceSHA256[ref] != hash {
			t.Fatalf("proof missing: %s", ref)
		}
	}
	again, err := analyzeTask(arena, n, taskCard{ID: task}, true)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got.RawReferences, again.RawReferences) || !reflect.DeepEqual(got.RawReferenceSHA256, again.RawReferenceSHA256) {
		t.Fatal("provenance is not deterministic")
	}
}

func TestExportProofDirectoryIsRefused(t *testing.T) {
	arena := t.TempDir()
	if err := os.MkdirAll(filepath.Join(arena, "raw", "fixture", "01.final.evaluation-export.yaml"), 0700); err != nil {
		t.Fatal(err)
	}
	_, err := analyzeTask(arena, nativeRecord{RunID: "fixture", Project: "p", Arm: "a", TaskID: "01"}, taskCard{}, false)
	if err == nil || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("directory proof accepted: %v", err)
	}
}

func TestExportProofCannotAliasOutsideArena(t *testing.T) {
	arena, outside := t.TempDir(), t.TempDir()
	if err := os.MkdirAll(filepath.Join(arena, "raw", "fixture"), 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(outside, "proof.yaml")
	if err := os.WriteFile(target, []byte("outside"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(arena, "raw", "fixture", "01.final.evaluation-export.yaml")); err != nil {
		t.Skipf("symlink creation unavailable: %v", err)
	}
	if _, err := analyzeTask(arena, nativeRecord{RunID: "fixture", Project: "p", Arm: "a", TaskID: "01"}, taskCard{}, false); err == nil {
		t.Fatal("outside proof alias accepted")
	}
}
