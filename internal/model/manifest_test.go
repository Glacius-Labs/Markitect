package model

import "testing"

func TestInspectManifestPublishesOnlyLexicallyValidRoot(t *testing.T) {
	input := ManifestInput{Origin: "captured-manifest", ManifestPath: "config/model.yaml", Bytes: []byte("formatVersion: 1\nid: project\nroot: überweisung/文档\ndependencies: {}\n")}
	got := InspectManifest(input)
	if got.Status != Passed || got.Root != "überweisung/文档" || got.Source.Origin != input.Origin || got.Source.Path != input.ManifestPath || got.Source.Digest == "" || len(got.Diagnostics) != 0 {
		t.Fatalf("valid manifest inspection failed: %+v", got)
	}
	input.Bytes = []byte("formatVersion: 1\nid: project\nroot: ../world\ndependencies: {}\n")
	got = InspectManifest(input)
	if got.Status != Failed || got.Root != "" || len(got.Diagnostics) == 0 {
		t.Fatalf("invalid manifest exposed a usable root: %+v", got)
	}
}
