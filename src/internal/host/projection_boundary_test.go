package host

import (
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/consumers/projections"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func registeredRepresentationFiles(t *testing.T) map[string][]byte {
	t.Helper()
	_, p := representationFixture(t)
	files := map[string][]byte{}
	for name, data := range p.Snapshot.Files {
		files[name] = append([]byte(nil), data...)
	}
	files["markitect.yaml"] = append(files["markitect.yaml"], []byte("  adapters:\n    - name: projections\n      type: local-projection\n      version: v1alpha1\n      config:\n        contracts: projections.config\n        coverage: markitect-artifacts.yaml\n")...)
	return files
}

func TestProjectionTargetsRemainOpaqueWithoutHidingCanonicalSources(t *testing.T) {
	files := registeredRepresentationFiles(t)
	config, err := projections.ParseConfig(files["projections.config"])
	if err != nil {
		t.Fatal(err)
	}
	config.Contracts[1].Targets = []projections.TargetPath{{Path: "intent/ci.yaml"}}
	files["projections.config"], err = YAML(config)
	if err != nil {
		t.Fatal(err)
	}
	files["intent/ci.yaml"] = []byte("name: checks\non: [push]\njobs: {}\n")
	p, err := Parse(&snapshot.Snapshot{Provisional: true, Files: files})
	if err != nil || len(p.Diagnostics) != 0 {
		t.Fatalf("opaque provider YAML rejected: %v, %#v", err, p)
	}
	if len(p.Resources) != 2 {
		t.Fatalf("opaque target adopted: %#v", p.Resources)
	}
	files["intent/ci.yaml"] = files["intent/rule.yaml"]
	if _, err := Parse(&snapshot.Snapshot{Provisional: true, Files: files}); err == nil || !strings.Contains(err.Error(), "canonical") {
		t.Fatalf("hidden canonical source accepted: %v", err)
	}
	delete(files, "projections.config")
	if _, err := Parse(&snapshot.Snapshot{Provisional: true, Files: files}); err == nil {
		t.Fatal("missing registered config accepted")
	}
}

func TestRepresentationCandidateCannotIntroduceCanonicalResources(t *testing.T) {
	root, p := representationFixture(t)
	plan, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	candidate := Materialization{APIVersion: MaterializationVersion, PlanDigest: plan.PlanDigest, Files: map[string]string{"implementation.py": string(p.Snapshot.Files["intent/rule.yaml"])}}
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate)); err == nil {
		t.Fatal("candidate silently introduced canonical resource")
	}
	if _, err := os.Stat(filepath.Join(root, "implementation.py")); !os.IsNotExist(err) {
		t.Fatalf("refusal wrote candidate: %v", err)
	}
}

func TestProjectionDomainSourcesAreIncludedWithoutInventingContextEdges(t *testing.T) {
	p, err := Load(filepath.Join("..", "..", "..", "examples", "projection-first"), "")
	if err != nil {
		t.Fatal(err)
	}
	config, err := projections.ParseConfig(p.Snapshot.Files["projections.config"])
	if err != nil {
		t.Fatal(err)
	}
	domainKey := "domain:dispatch.example.org/v1alpha1/dispatch"
	config.Contracts = []projections.Contract{{ID: "domain-only", Sources: []string{domainKey}, Representation: "contract", Materializer: projections.Materializer{Name: "agent", Version: "1", Mode: "ai"}, Targets: []projections.TargetPath{{Path: "contract.txt"}}, VerificationChecks: []string{"independent-contract"}}}
	p.Snapshot.Files["projections.config"], err = YAML(config)
	if err != nil {
		t.Fatal(err)
	}
	context, err := ProjectionContext(p, nil)
	if err != nil || len(context) != 1 || context[0].MatchingSourceKeys[0] != domainKey {
		t.Fatalf("Domain context provenance missing: %#v %v", context, err)
	}
	untouched, err := ProjectionImpact(p, nil, []string{"README.md"})
	if err != nil || len(untouched) != 0 {
		t.Fatalf("unrelated narrative broadened Domain fanout: %#v %v", untouched, err)
	}
	for _, paths := range [][]string{{"domains/dispatch.yaml"}, {"projections.config"}, {"markitect.yaml"}} {
		impact, err := ProjectionImpact(p, nil, paths)
		if err != nil || len(impact) != 1 {
			t.Fatalf("Domain/config input fanout missing for %v: %#v %v", paths, impact, err)
		}
	}
}

func TestRepresentationCheckCannotChangeExecutableMode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows regular-file permissions do not encode Unix executable mode")
	}
	dir := t.TempDir()
	file := filepath.Join(dir, "input")
	if err := os.WriteFile(file, []byte("fixed"), 0644); err != nil {
		t.Fatal(err)
	}
	p := &Project{Snapshot: &snapshot.Snapshot{Files: map[string][]byte{"input": []byte("fixed")}, Modes: map[string]string{"input": "100644"}}}
	if err := verifySnapshotFilesUnchanged(p, dir); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(file, 0755); err != nil {
		t.Fatal(err)
	}
	if err := verifySnapshotFilesUnchanged(p, dir); err == nil {
		t.Fatal("check changed reviewed executable mode")
	}
}
