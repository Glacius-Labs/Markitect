package app

import (
	"strings"
	"testing"
)

func TestRunContextDigestTracksSelectedTaskAndSourceBytes(t *testing.T) {
	snapshot := fixtureFiles(t, "fixed-run-snapshot", "Policy source.", projectNS)
	snapshot.Files["work-items/17.md"] = []byte("Work item A\n")
	snapshot.Files["src/OrderService.cs"] = []byte("class OrderService {}\n")
	snapshot.Files["src/OrderRules.cs"] = []byte("class OrderRules {}\n")
	manifestOne := []byte("version: " + RunManifestVersion + "\nentry: sample/Skill/entry\ntask:\n  id: work-item-17\n  path: work-items/17.md\nsources:\n  - path: src/OrderService.cs\n    reason: primary implementation\n")
	manifestTwo := []byte("version: " + RunManifestVersion + "\nentry: sample/Skill/entry\ntask:\n  id: work-item-17\n  path: work-items/17.md\nsources:\n  - path: src/OrderRules.cs\n    reason: related implementation\n")
	snapshot.Files["run-one.yaml"] = manifestOne
	snapshot.Files["run-two.yaml"] = manifestTwo
	p, err := Parse(snapshot)
	if err != nil || len(p.Diagnostics) > 0 {
		t.Fatalf("parse fixed fixture: err=%v diagnostics=%#v", err, p.Diagnostics)
	}
	first, err := CompileRunContext(p, "run-one.yaml", manifestOne, "0.5.0", "tool")
	if err != nil {
		t.Fatal(err)
	}
	second, err := CompileRunContext(p, "run-two.yaml", manifestTwo, "0.5.0", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if first.Digest == second.Digest || first.Run.SelectionHash == second.Run.SelectionHash {
		t.Fatal("context digest ignored an explicitly changed source selection")
	}
	for _, ctx := range []*Context{first, second} {
		for _, input := range ctx.Inputs {
			if input.Role == "task" || input.Role == "source" {
				if input.Status != "included" || input.Hash == "" || input.Text == "" {
					t.Errorf("selected run input lacks captured bytes/hash: %#v", input)
				}
			}
		}
	}
	snapshot.Files["src/OrderService.cs"] = []byte("class OrderService { void Change() {} }\n")
	p, err = Parse(snapshot)
	if err != nil || len(p.Diagnostics) > 0 {
		t.Fatalf("parse changed fixture: err=%v diagnostics=%#v", err, p.Diagnostics)
	}
	changed, err := CompileRunContext(p, "run-one.yaml", manifestOne, "0.5.0", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if changed.Digest == first.Digest {
		t.Fatal("context digest ignored changed selected source bytes")
	}
}

func TestParseRunManifestRequiresUniqueNormalizedPaths(t *testing.T) {
	valid := "version: " + RunManifestVersion + "\nentry: sample/Skill/entry\ntask:\n  id: work-item-17\n  path: work-items/17.md\nsources:\n  - path: src/a.cs\n    reason: selected implementation\n"
	if _, err := ParseRunManifest([]byte(valid)); err != nil {
		t.Fatalf("valid run manifest rejected: %v", err)
	}
	missingID := strings.Replace(valid, "  id: work-item-17\n", "", 1)
	if _, err := ParseRunManifest([]byte(missingID)); err == nil {
		t.Fatal("manifest without a concrete task id was accepted")
	}
	duplicate := strings.Replace(valid, "path: src/a.cs", "path: work-items/17.md", 1)
	if _, err := ParseRunManifest([]byte(duplicate)); err == nil {
		t.Fatal("duplicate task/source path was accepted")
	}
	traversal := strings.Replace(valid, "src/a.cs", "../outside.cs", 1)
	if _, err := ParseRunManifest([]byte(traversal)); err == nil {
		t.Fatal("path traversal was accepted")
	}
}

func TestRunContextMarksBinarySelectionIncompleteAndRejectsManifestMismatch(t *testing.T) {
	snapshot := fixtureFiles(t, "fixed-run-snapshot", "Policy source.", projectNS)
	manifestBytes := []byte("version: " + RunManifestVersion + "\nentry: sample/Skill/entry\ntask:\n  id: work-item-17\n  path: work-items/17.md\nsources:\n  - path: src/binary.cs\n    reason: selected implementation\n")
	snapshot.Files["context-run.yaml"] = manifestBytes
	snapshot.Files["work-items/17.md"] = []byte("Acceptance criteria: preserve idempotency.\n")
	snapshot.Files["src/binary.cs"] = []byte{'c', 'l', 'a', 's', 's', 0, 'X'}
	p, err := Parse(snapshot)
	if err != nil || len(p.Diagnostics) > 0 {
		t.Fatalf("parse fixed fixture: err=%v diagnostics=%#v", err, p.Diagnostics)
	}
	ctx, err := CompileRunContext(p, "context-run.yaml", manifestBytes, "0.5.0", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if ctx.Complete || ctx.Status != "incomplete" {
		t.Fatalf("binary selected input was accepted: %#v", ctx)
	}
	for _, input := range ctx.Inputs {
		if input.Path == "src/binary.cs" {
			if input.Status != "invalid-text" || input.Text != "" || input.Hash == "" {
				t.Fatalf("binary source report = %#v", input)
			}
			break
		}
	}
	if _, err := CompileRunContext(p, "context-run.yaml", []byte("different manifest"), "0.5.0", "tool"); err == nil {
		t.Fatal("compiler accepted manifest bytes outside the fixed snapshot")
	}
}
