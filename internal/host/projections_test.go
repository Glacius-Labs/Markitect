package host

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/modules/projections"
)

func representationFixture(t *testing.T) (string, *Project) {
	t.Helper()
	root := t.TempDir()
	files := map[string][]byte{
		"markitect.yaml":           []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Project\nmetadata:\n  name: proof\nspec:\n  targets: [markdown]\n  areas:\n    - name: proof\n      path: intent\n  checks:\n    - name: compiler-available\n      run: [go, version]\n"),
		"intent/rule.yaml":         []byte("apiVersion: markitect.example.org/v1alpha1\nkind: Rule\nmetadata:\n  name: desired\n  namespace: proof\nspec:\n  text: Keep declared intent.\n"),
		"markitect-artifacts.yaml": []byte("apiVersion: markitect.example.org/artifact-coverage/v1alpha1\nkind: ArtifactCoverage\nspec:\n  roots: [markitect.yaml, intent, projections.config, markitect-artifacts.yaml, docs/markitect, implementation.py]\n  tooling:\n    - path: markitect-artifacts.yaml\n      owner: proof-accounting\n"),
	}
	p, err := Parse(&snapshot.Snapshot{Provisional: true, Files: files})
	if err != nil {
		t.Fatal(err)
	}
	_, owners, err := GenerateOutputsWithOwners(p)
	if err != nil {
		t.Fatal(err)
	}
	config := projections.Config{APIVersion: "markitect.example.org/projections/v1alpha1", Version: "1"}
	sourceKeys := []string{}
	model, err := CompileModel(p)
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range model.Resources {
		sourceKeys = append(sourceKeys, r.Identity.Key)
	}
	deterministic := projections.Contract{ID: "views", Sources: sourceKeys, Representation: "documentation", Materializer: projections.Materializer{Name: "markitect-render", Version: "v1alpha1", Mode: "deterministic"}}
	for name := range owners {
		deterministic.Targets = append(deterministic.Targets, projections.TargetPath{Path: name})
	}
	config.Contracts = []projections.Contract{deterministic, {ID: "implementation", Sources: []string{"proof/Rule/desired"}, Representation: "implementation", Materializer: projections.Materializer{Name: "coding-agent", Version: "proof-1", Mode: "ai"}, Targets: []projections.TargetPath{{Path: "implementation.py"}}, VerificationChecks: []string{"compiler-available"}, Freedom: []string{"private helper names"}}}
	encoded, err := YAML(config)
	if err != nil {
		t.Fatal(err)
	}
	files["projections.config"] = encoded
	for name, data := range files {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatal(err)
		}
	}
	p, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) > 0 {
		t.Fatal(p.Diagnostics)
	}
	return root, p
}

func TestRepresentationApplyRejectsStaleAndForeignCandidate(t *testing.T) {
	root, p := representationFixture(t)
	plan, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	candidate := Materialization{APIVersion: MaterializationVersion, PlanDigest: plan.PlanDigest, Files: map[string]string{"intent/rule.yaml": "hostile"}}
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate)); err == nil {
		t.Fatal("foreign candidate path accepted")
	}
	candidate.Files = map[string]string{"implementation.py": "print('candidate')\n"}
	candidate.PlanDigest = "stale"
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate)); err == nil {
		t.Fatal("stale candidate accepted")
	}
	candidate.PlanDigest = plan.PlanDigest
	if err := os.WriteFile(filepath.Join(root, "intent/rule.yaml"), []byte("changed"), 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate)); err == nil {
		t.Fatal("concurrently changed intent accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "implementation.py")); !os.IsNotExist(err) {
		t.Fatalf("failure wrote target: %v", err)
	}
}

func TestRepresentationsConvergeOnlyWithFixedEvidence(t *testing.T) {
	root, p := representationFixture(t)
	first, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil || !yamlEqual(first, repeat) {
		t.Fatalf("non-deterministic plan: %v", err)
	}
	candidate := Materialization{APIVersion: MaterializationVersion, PlanDigest: first.PlanDigest, Files: map[string]string{"implementation.py": "print('candidate')\n"}}
	applied, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", first, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate))
	if err != nil {
		t.Fatal(err)
	}
	if len(applied.Written) == 0 || applied.Status != "materialized-unverified" {
		t.Fatal(applied)
	}
	p, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	observed, err := ObserveRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if observed.Status != "incomplete" {
		t.Fatalf("AI presence falsely considered verified: %s", observed.Status)
	}
	if _, err := VerifyRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool"); err == nil {
		t.Fatal("provisional verification accepted")
	}
	// This unit check proves orchestration only. 'go version' is intentionally
	// not behavioral evidence; the greenfield example uses independent tests.
	p.Snapshot.Provisional = false
	p.Snapshot.ID = strings.Repeat("a", 40)
	verified, err := VerifyRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if verified.Status != "converged" || verified.Plan.Status != verified.Status {
		t.Fatalf("fixed declared evidence did not converge: %+v", verified)
	}
}

func TestRepresentationStrictPolicyAndStructureBoundary(t *testing.T) {
	_, p := representationFixture(t)
	p.Diagnostics = append(p.Diagnostics, core.Diagnostic{Code: "constraint", Message: "failed desired invariant"})
	if _, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool"); err == nil {
		t.Fatal("policy or structural failure accepted")
	}
}

func TestRepresentationRecordRefusals(t *testing.T) {
	for _, text := range []string{"apiVersion: wrong\nplanDigest: x\nfiles: {}\n", "apiVersion: markitect.example.org/materialization/v1alpha1\nplanDigest: x\nfiles: {}\n---\n{}\n", "apiVersion: markitect.example.org/materialization/v1alpha1\nplanDigest: x\nfiles: {}\nunknown: yes\n"} {
		path := filepath.Join(t.TempDir(), "candidate.yaml")
		if err := os.WriteFile(path, []byte(text), 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := ReadMaterialization(path); err == nil {
			t.Fatalf("invalid record accepted: %s", text)
		}
	}
}

func reviewedMaterialization(t *testing.T, candidate Materialization) *Materialization {
	t.Helper()
	data, err := YAML(candidate)
	if err != nil {
		t.Fatal(err)
	}
	name := filepath.Join(t.TempDir(), "candidate.yaml")
	if err := os.WriteFile(name, data, 0600); err != nil {
		t.Fatal(err)
	}
	value, err := ReadMaterialization(name)
	if err != nil {
		t.Fatal(err)
	}
	return &value
}
func materializationReviewDigest(t *testing.T, candidate Materialization) string {
	t.Helper()
	data, err := YAML(candidate)
	if err != nil {
		t.Fatal(err)
	}
	return hashBytes(data)
}
func TestRepresentationRequiresCompleteReviewedCandidate(t *testing.T) {
	root, p := representationFixture(t)
	plan, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, nil, ""); err == nil {
		t.Fatal("missing AI candidate accepted")
	}
	incomplete := Materialization{APIVersion: MaterializationVersion, PlanDigest: plan.PlanDigest, Files: map[string]string{}}
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewedMaterialization(t, incomplete), materializationReviewDigest(t, incomplete)); err == nil {
		t.Fatal("partial AI candidate accepted")
	}
	complete := Materialization{APIVersion: MaterializationVersion, PlanDigest: plan.PlanDigest, Files: map[string]string{"implementation.py": "print('candidate')\n"}}
	reviewed := reviewedMaterialization(t, complete)
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewed, "sha256:wrong"); err == nil {
		t.Fatal("unreviewed bytes accepted")
	}
	reviewed.Files["implementation.py"] = "print('substituted')\n"
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewed, materializationReviewDigest(t, complete)); err == nil {
		t.Fatal("object substitution after read accepted")
	}
	if _, err := os.Stat(filepath.Join(root, "implementation.py")); !os.IsNotExist(err) {
		t.Fatal("refused candidate wrote target")
	}
}
func TestRepresentationRejectsHostLockTarget(t *testing.T) {
	_, p := representationFixture(t)
	p.Snapshot.Files["projections.config"] = []byte(strings.ReplaceAll(string(p.Snapshot.Files["projections.config"]), "implementation.py", ".artifacts/markitect/write.lock"))
	if _, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool"); err == nil {
		t.Fatal("Host workspace target accepted")
	}
}

func TestRepresentationChecksRejectMutatedOriginalInputs(t *testing.T) {
	_, p := representationFixture(t)
	p.Snapshot.Provisional = false
	p.Snapshot.Files["mutator.go"] = []byte(`package main
import "os"
func main(){if err:=os.WriteFile("intent/rule.yaml",[]byte("substituted"),0644);err!=nil{panic(err)}}
`)
	p.Snapshot.Modes["mutator.go"] = snapshot.RegularMode
	p.Graph.Project.Spec.Checks = []authoring.Check{{Name: "mutator", Run: []string{"go", "run", "mutator.go"}}}
	results, err := VerifyRepresentationChecks(p)
	if err == nil || !strings.Contains(err.Error(), "input-mutation") {
		t.Fatalf("mutating check accepted: %v", err)
	}
	if len(results) != 1 || results[0].ExitCode != -1 {
		t.Fatalf("mutation result %v", results)
	}
}
func TestRepresentationChecksDoNotInheritEarlierOutputs(t *testing.T) {
	_, p := representationFixture(t)
	p.Snapshot.Provisional = false
	p.Snapshot.Files["check.go"] = []byte(`package main
import "os"
func main(){if os.Args[1]=="write" {if err:=os.WriteFile("temporary-check-output",[]byte("temporary"),0644);err!=nil{panic(err)};return};if _,err:=os.Stat("temporary-check-output");!os.IsNotExist(err){panic("inherited prior output")}}
`)
	p.Snapshot.Modes["check.go"] = snapshot.RegularMode
	p.Graph.Project.Spec.Checks = []authoring.Check{{Name: "first", Run: []string{"go", "run", "check.go", "write"}}, {Name: "second", Run: []string{"go", "run", "check.go", "absent"}}}
	results, err := VerifyRepresentationChecks(p)
	if err != nil || len(results) != 2 {
		t.Fatalf("checks did not start at identical snapshots: %v %+v", err, results)
	}
}

func TestRepresentationFailureOutsideSelectedChecksStillRejectsAcceptance(t *testing.T) {
	root, p := representationFixture(t)
	plan, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	candidate := Materialization{APIVersion: MaterializationVersion, PlanDigest: plan.PlanDigest, Files: map[string]string{"implementation.py": "print('candidate')\n"}}
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", plan, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate)); err != nil {
		t.Fatal(err)
	}
	p, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	p.Graph.Project.Spec.Checks = append([]authoring.Check{{Name: "independent-precondition", Run: []string{"go", "version", "unsupported-argument"}}}, p.Graph.Project.Spec.Checks...)
	p.Snapshot.Provisional = false
	report, err := VerifyRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != "drift" || report.Plan.Status != report.Status || len(report.Checks) != 1 || report.Checks[0].ExitCode <= 0 {
		t.Fatalf("known failed gate was not drift: %#v", report)
	}
}

func TestRepresentationPlanCannotClaimConvergenceWithoutAccountingAndChecks(t *testing.T) {
	root, p := representationFixture(t)
	first, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	candidate := Materialization{APIVersion: MaterializationVersion, PlanDigest: first.PlanDigest, Files: map[string]string{"implementation.py": "print('candidate')\n"}}
	if _, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "tool", first, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate)); err != nil {
		t.Fatal(err)
	}
	p, err = Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	config, err := projections.ParseConfig(p.Snapshot.Files["projections.config"])
	if err != nil {
		t.Fatal(err)
	}
	var deterministic []projections.Contract
	for _, contract := range config.Contracts {
		if contract.Materializer.Mode == "deterministic" {
			deterministic = append(deterministic, contract)
		}
	}
	config.Contracts = deterministic
	p.Snapshot.Files["projections.config"], err = YAML(config)
	if err != nil {
		t.Fatal(err)
	}
	input, err := projectionInput(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	contractOnly, err := projections.Build(input)
	if err != nil || contractOnly.Status != "converged" {
		t.Fatalf("test must start from matched pure contracts: %v %#v", err, contractOnly)
	}
	plan, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "drift" {
		t.Fatalf("unmanaged implementation accepted in plan: %#v", plan)
	}
	found := false
	for _, d := range plan.Diagnostics {
		if d.Code == "projection.accounting.unmanaged" {
			found = true
		}
	}
	if !found {
		t.Fatal("accounting failure had no bound plan cause")
	}
	// Fixed-check execution must not conceal accounting drift in its nested plan.
	p.Snapshot.Provisional = false
	verified, err := VerifyRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil || verified.Status != "drift" || verified.Plan.Status != verified.Status {
		t.Fatalf("verify aggregate/plan accounting contradiction: %v %#v", err, verified)
	}
	found = false
	for _, d := range verified.Plan.Diagnostics {
		if d.Code == "projection.accounting.unmanaged" {
			found = true
		}
	}
	if !found {
		t.Fatal("verify nested plan omitted accounting cause")
	}
	p.Snapshot.Files["markitect-artifacts.yaml"] = append(p.Snapshot.Files["markitect-artifacts.yaml"], []byte("    - path: implementation.py\n      owner: explicitly-owned-tool\n")...)
	plan, err = PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "incomplete" || plan.Contracts[0].Status != "converged" {
		t.Fatalf("plan concealed missing check evidence or matching target state: %#v", plan)
	}
	again, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "tool")
	if err != nil || !yamlEqual(plan, again) {
		t.Fatalf("composed plan not deterministic: %v", err)
	}
}
