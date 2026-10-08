package capture

import (
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"strings"
	"testing"
)

func testHandoff() (Handoff, map[string][]byte) {
	data := []byte("A supplied architecture observation.\n")
	identity := RepositoryIdentity{Root: "/supplied/repository", GitDir: "/supplied/repository/.git", CommonDir: "/supplied/repository/.git", ObjectFormat: "sha1"}
	identity.Digest = ValueDigest(identity)
	s := &snapshot.Snapshot{Files: map[string][]byte{"docs/rule.md": data}, Modes: map[string]string{"docs/rule.md": snapshot.RegularMode}}
	h := Handoff{APIVersion: HandoffVersion, ID: "review-1", Purpose: "bounded review", Review: "owner supplied review", Privacy: Privacy{Constraints: "public fixture only"}, Retention: "delete after review", Repositories: []Repository{{ID: "repo", Identity: identity, Commit: strings.Repeat("a", 40), SnapshotDigest: s.Digest(), Files: []File{{Path: "docs/rule.md", Reason: "explicit architecture sample", Mode: snapshot.RegularMode, Digest: Hash(data)}}, Exclusions: []Exclusion{{Path: "private", Reason: "not selected"}}}}, Coverage: []Coverage{{ID: "question", Repository: "repo", Question: "Is ownership explicit?", State: "uninspected", Reason: "interpretation pending"}}}
	Seal(&h)
	return h, map[string][]byte{"repo/docs/rule.md": data}
}

func TestHandoffBindsSelectedBytesAndIndependentIdentity(t *testing.T) {
	h, b := testHandoff()
	if err := ValidateHandoff(h, b); err != nil {
		t.Fatal(err)
	}
	before := h.Digest
	Seal(&h)
	if h.Digest != before {
		t.Fatal("non-deterministic seal")
	}
	originalContent := h.Repositories[0].SnapshotDigest
	h.Repositories[0].Commit = strings.Repeat("b", 40)
	Seal(&h)
	if h.Digest == before || h.Repositories[0].SnapshotDigest != originalContent {
		t.Fatal("revision identity must change without changing content digest")
	}
	if err := ValidateHandoff(h, b); err != nil {
		t.Fatal(err)
	}
	b["repo/docs/rule.md"] = []byte("changed")
	if err := ValidateHandoff(h, b); err == nil {
		t.Fatal("changed selected bytes accepted")
	}
}
func TestHandoffPreflightAndOptionalMarkitectBindings(t *testing.T) {
	for _, test := range []struct {
		name   string
		mutate func(*Handoff)
	}{
		{"unsafe path", func(h *Handoff) { h.Repositories[0].Files[0].Path = "../escape"; Seal(h) }},
		{"duplicate identity", func(h *Handoff) { h.Repositories = append(h.Repositories, h.Repositories[0]); Seal(h) }},
		{"changed identity", func(h *Handoff) { h.Repositories[0].Identity.Root = "/other"; Seal(h) }},
		{"abbreviated commit", func(h *Handoff) { h.Repositories[0].Commit = "abc"; Seal(h) }},
		{"coverage state", func(h *Handoff) { h.Coverage[0].State = "complete"; Seal(h) }},
		{"excluded selection", func(h *Handoff) { h.Repositories[0].Exclusions[0].Path = "docs"; Seal(h) }},
		{"wrong mode", func(h *Handoff) { h.Repositories[0].Files[0].Mode = "120000"; Seal(h) }},
		{"parent alias", func(h *Handoff) {
			f := h.Repositories[0].Files[0]
			f.Path = "Docs/second.md"
			h.Repositories[0].Files = append(h.Repositories[0].Files, f)
			Seal(h)
		}},
		{"file directory alias", func(h *Handoff) {
			f := h.Repositories[0].Files[0]
			f.Path = "docs"
			h.Repositories[0].Files = append(h.Repositories[0].Files, f)
			Seal(h)
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			h, _ := testHandoff()
			test.mutate(&h)
			if err := ValidateManifest(h); err == nil {
				t.Fatal("invalid manifest accepted")
			}
		})
	}
	h, b := testHandoff()
	a := Artifact{Path: "docs/rule.md", Digest: h.Repositories[0].Files[0].Digest}
	h.Repositories[0].Markitect = &MarkitectEvidence{Manifest: a, Report: a, Context: a, SelectionDigest: strings.Repeat("c", 64), Version: "supplied", BuildDigest: strings.Repeat("d", 64)}
	Seal(&h)
	if err := ValidateHandoff(h, b); err != nil {
		t.Fatal(err)
	}
	h.Repositories[0].Markitect.Manifest.Digest = strings.Repeat("0", 64)
	Seal(&h)
	if err := ValidateManifest(h); err == nil {
		t.Fatal("optional Markitect binding not preflighted")
	}
	if err := ValidateHandoff(h, b); err == nil {
		t.Fatal("changed optional artifact accepted")
	}
}
func TestExactByteSetAndStrictRecord(t *testing.T) {
	h, b := testHandoff()
	b["repo/unselected"] = []byte("unselected")
	if err := ValidateHandoff(h, b); err == nil {
		t.Fatal("broadened supplied byte set accepted")
	}
	for _, p := range []string{"../x", "/x", "a\\b", "a/*", "a/[x]", ".git/config", "CON.md", "a/./b", "a:stream", "a. ", "README.md\n"} {
		if ExactPath(p) == nil {
			t.Fatalf("unsafe path accepted %q", p)
		}
	}
	for _, data := range []string{"id: x\nid: y\n", "id: &x hello\npurpose: *x\n", "id: x\n---\nid: y\n", "unknown: x\n"} {
		var s Scope
		if Decode([]byte(data), &s) == nil {
			t.Fatalf("unsafe/unknown record accepted %q", data)
		}
	}
}

func TestEmptyOptionalListsRoundTripWithoutDigestDrift(t *testing.T) {
	h, b := testHandoff()
	h.Coverage = nil
	h.Repositories[0].Exclusions = []Exclusion{}
	Seal(&h)
	data, err := Encode(h)
	if err != nil {
		t.Fatal(err)
	}
	var decoded Handoff
	if err := Decode(data, &decoded); err != nil {
		t.Fatal(err)
	}
	if err := ValidateHandoff(decoded, b); err != nil {
		t.Fatal(err)
	}
}
