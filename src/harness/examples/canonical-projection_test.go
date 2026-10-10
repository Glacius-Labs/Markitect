package examples

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/host/records"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/modules/dotnet"
	"go.yaml.in/yaml/v3"
)

const canonicalProjectionConfigPath = "examples/canonical-projection/canonical.yaml"

func TestCanonicalProjectionBoundCandidateApplyVerifyAndRepair(t *testing.T) {
	fixtureRoot := canonicalProjectionFixtureRoot(t)
	tempRoot := t.TempDir()
	root := filepath.Join(tempRoot, "repo")
	absoluteTemp, err := filepath.Abs(tempRoot)
	if err != nil {
		t.Fatal(err)
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	absoluteFixture, err := filepath.Abs(fixtureRoot)
	if err != nil {
		t.Fatal(err)
	}
	relativeRoot, err := filepath.Rel(absoluteTemp, absoluteRoot)
	if err != nil || relativeRoot == "." || relativeRoot == ".." || strings.HasPrefix(relativeRoot, ".."+string(filepath.Separator)) || filepath.IsAbs(relativeRoot) || strings.EqualFold(absoluteRoot, absoluteFixture) {
		t.Fatalf("refusing to use the checked-in fixture as a mutable Git root: temp=%q root=%q fixture=%q err=%v", absoluteTemp, absoluteRoot, absoluteFixture, err)
	}
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	copyCanonicalProjectionFixture(t, root)
	writeFile(t, root, "scratch/unclaimed.txt", []byte("explicitly unknown inventory input\n"))
	writeFile(t, root, "docs/legacy/old.md", []byte("excluded from this slice\n"))
	check := authoring.Check{
		Name: "canonical-projection-fixture",
		Run:  []string{"go", "run", "examples/canonical-projection/evidence/check.go"},
	}

	preview, err := host.LoadCanonicalSource(root, "", canonicalProjectionConfigPath, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Pins) != len(preview.Previews) {
		t.Fatalf("checked-in fixture pins do not match current module digests: pins=%d modules=%d", len(preview.Pins), len(preview.Previews))
	}
	var config host.CanonicalSourceConfig
	configBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(canonicalProjectionConfigPath)))
	if err != nil {
		t.Fatal(err)
	}
	if err := yaml.Unmarshal(configBytes, &config); err != nil {
		t.Fatal(err)
	}
	if len(config.Modules) != len(preview.Previews) {
		t.Fatalf("preview/module count mismatch: %d != %d", len(config.Modules), len(preview.Previews))
	}
	for i, module := range preview.Previews {
		pin := module.Pin
		pin.Digest = module.Digest
		config.Modules[i].Pin = &pin
	}
	configBytes, err = yaml.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, root, canonicalProjectionConfigPath, configBytes)

	initCanonicalProjectionGit(t, root)
	sourceRevision := commitCanonicalProjection(t, root, "pin canonical projection fixture")
	fixed, err := host.LoadCanonicalSource(root, sourceRevision, canonicalProjectionConfigPath, true)
	if err != nil {
		t.Fatal(err)
	}
	if len(fixed.Diagnostics) != 0 || fixed.Model.Revision != fixed.Snapshot.ID || len(fixed.Activation.Projectors) != 2 {
		t.Fatalf("fixed pinned source did not compile/activate cleanly: diagnostics=%#v projectors=%d", fixed.Diagnostics, len(fixed.Activation.Projectors))
	}

	observed, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	if !observed.Provisional {
		t.Fatal("working target state must remain provisional")
	}
	for _, target := range []string{"docs/represented/index.md", "src/Commerce/CreateOrderHandler.cs", "src/Commerce/EffectAxis.cs", "src/Commerce/Commerce.csproj"} {
		if _, exists := observed.Files[target]; exists {
			t.Fatalf("installing projection Modules unexpectedly materialized %s", target)
		}
	}

	markdownID := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-markdown"}
	dotnetID := core.DefinitionIdentity{APIVersion: "markitect.foundation/v1", Kind: "Projection", Namespace: "commerce", Name: "application-dotnet"}

	markdownPrepared, err := host.PrepareCanonicalProjection(fixed, observed, markdownID, "example-tool/1", host.Hash([]byte("canonical-projection-example")), nil, check)
	if err != nil {
		t.Fatal(err)
	}
	if markdownPrepared.Plan == nil || len(markdownPrepared.Escalations) != 0 || len(markdownPrepared.Outputs) != 1 {
		t.Fatalf("explicit Markdown Projection did not prepare: %#v", markdownPrepared.Escalations)
	}
	markdownApply, err := host.ApplyCanonicalProjection(root, fixed, observed, markdownPrepared, markdownPrepared.CandidateDigest, true)
	if err != nil {
		t.Fatal(err)
	}
	if markdownApply.Status != records.StateMaterializedUnverified || markdownApply.Record == nil || len(markdownApply.Written) != 1 {
		t.Fatalf("Markdown Apply must produce materialized-unverified provenance: %#v", markdownApply)
	}
	persistProjectionRecord(t, root, *markdownApply.Record)

	observed, err = source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	missingCandidate, err := host.PrepareCanonicalProjection(fixed, observed, dotnetID, "example-tool/1", host.Hash([]byte("canonical-projection-example")), nil, check)
	if err != nil {
		t.Fatal(err)
	}
	if missingCandidate.Plan != nil || !hasEscalation(missingCandidate.Escalations, "projection.candidate-required") {
		t.Fatalf("installing .NET capability should require an explicit candidate: %#v", missingCandidate.Escalations)
	}
	if len(missingCandidate.Request.Definitions) != 3 {
		t.Fatalf(".NET Projection should bind exactly its three selected Definitions, got %d", len(missingCandidate.Request.Definitions))
	}

	withoutGuidance := dotnet.Evaluate(dotnet.Input{
		Definitions:    missingCandidate.Request.Definitions,
		Schemas:        missingCandidate.Request.Schemas,
		AllowedPaths:   []string{"src/Commerce/CreateOrderHandler.cs"},
		CandidateFiles: map[string][]byte{"src/Commerce/CreateOrderHandler.cs": []byte("class CreateOrderHandler {}")},
	})
	if len(withoutGuidance.Escalations) == 0 || len(withoutGuidance.Files) != 0 {
		t.Fatalf("missing per-Kind guidance must escalate with no candidate files: %#v", withoutGuidance)
	}
	withoutChecks, err := host.PrepareCanonicalProjection(fixed, observed, dotnetID, "example-tool/1", host.Hash([]byte("canonical-projection-example")), candidateEnvelope(t, missingCandidate.Request.RequestDigest))
	if err != nil {
		t.Fatal(err)
	}
	if withoutChecks.Plan != nil || !hasEscalation(withoutChecks.Escalations, "projection.checks-required") {
		t.Fatalf("candidate without an independent named check must escalate: %#v", withoutChecks.Escalations)
	}

	candidate := candidateEnvelope(t, missingCandidate.Request.RequestDigest)
	reviewed, err := host.PrepareCanonicalProjection(fixed, observed, dotnetID, "example-tool/1", host.Hash([]byte("canonical-projection-example")), candidate, check)
	if err != nil {
		t.Fatal(err)
	}
	if reviewed.Plan == nil || len(reviewed.Escalations) != 0 || len(reviewed.Outputs) != 3 {
		t.Fatalf("request-bound candidate did not prepare: escalations=%#v outputs=%v", reviewed.Escalations, sortedMapKeys(reviewed.Outputs))
	}
	if reviewed.CandidateDigest == "" || reviewed.Request.TargetPrefix != "src" {
		t.Fatalf("candidate is missing exact review bindings: digest=%q target=%q", reviewed.CandidateDigest, reviewed.Request.TargetPrefix)
	}

	if _, err := host.ApplyCanonicalProjection(root, fixed, observed, reviewed, "sha256:wrong-candidate", true); err == nil {
		t.Fatal("Apply accepted a digest different from the reviewed candidate")
	}
	staleSource := cloneSnapshot(observed)
	sourcePath := reviewed.Request.Definitions[0].Source.Path
	staleSource.Files[sourcePath] = append(staleSource.Files[sourcePath], []byte("# changed after fixed revision\n")...)
	if _, err := host.ApplyCanonicalProjection(root, fixed, staleSource, reviewed, reviewed.CandidateDigest, true); err == nil {
		t.Fatal("Apply accepted canonical source bytes changed after the fixed revision")
	}
	staleTarget := cloneSnapshot(observed)
	staleTarget.Files["src/Unexpected.cs"] = []byte("class Unexpected {}")
	staleTarget.Modes["src/Unexpected.cs"] = snapshot.RegularMode
	if _, err := host.ApplyCanonicalProjection(root, fixed, staleTarget, reviewed, reviewed.CandidateDigest, true); err == nil {
		t.Fatal("Apply accepted a candidate bound to an older target snapshot")
	}
	stalePlan := reviewed
	planCopy := *reviewed.Plan
	planCopy.PlanDigest = "sha256:stale"
	stalePlan.Plan = &planCopy
	if _, err := host.ApplyCanonicalProjection(root, fixed, observed, stalePlan, reviewed.CandidateDigest, true); err == nil {
		t.Fatal("Apply accepted a modified reviewed plan")
	}

	dotnetApply, err := host.ApplyCanonicalProjection(root, fixed, observed, reviewed, reviewed.CandidateDigest, true)
	if err != nil {
		t.Fatal(err)
	}
	if dotnetApply.Status != records.StateMaterializedUnverified || dotnetApply.Record == nil || len(dotnetApply.Written) != 3 {
		t.Fatalf(".NET Apply did not record the exact candidate artifacts: %#v", dotnetApply)
	}
	if len(dotnetApply.Record.ScopeIDs) != 3 || len(dotnetApply.Record.Artifacts) != 3 {
		t.Fatalf("three canonical Definitions should map to two C# files plus a project file: %#v", dotnetApply.Record)
	}
	persistProjectionRecord(t, root, *dotnetApply.Record)

	active := []records.ProjectionRecord{*markdownApply.Record, *dotnetApply.Record}
	if err := records.ValidateRecordReferences(active); err != nil {
		t.Fatalf("projection record references are invalid: %v", err)
	}
	materialized, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	index := buildExampleOwnershipIndex(t, fixed, active, materialized)
	reconciliation, err := host.PlanCanonicalReconciliation(fixed, fixed, materialized, active)
	if err != nil {
		t.Fatal(err)
	}
	if len(reconciliation.Work) != 0 || len(reconciliation.NoApplicableWork) != 2 || len(reconciliation.Escalations) != 0 {
		t.Fatalf("unchanged represented scopes should produce no work: %#v", reconciliation)
	}
	assertOwnershipStatus(t, index, "src/Commerce/CreateOrderHandler.cs", records.OwnershipManaged)
	assertOwnershipStatus(t, index, "src/Commerce/EffectAxis.cs", records.OwnershipManaged)
	assertOwnershipStatus(t, index, "docs/represented/index.md", records.OwnershipManaged)
	assertOwnershipStatus(t, index, "scratch/unclaimed.txt", records.OwnershipUnknown)
	assertOwnershipStatus(t, index, "docs/legacy/old.md", records.OwnershipExcluded)

	firstDotnetRecordPath := filepath.Join(root, ".markitect", "projections", "records", strings.TrimPrefix(dotnetApply.Record.ID, "sha256:")+".json")
	firstRecordBytes, err := os.ReadFile(firstDotnetRecordPath)
	if err != nil {
		t.Fatal(err)
	}
	initialCanonicalDigest := fixed.Model.Digest
	finalCommit := commitCanonicalProjection(t, root, "materialize verified projection fixture")
	verifySnapshot, err := source.Load(root, finalCommit)
	if err != nil {
		t.Fatal(err)
	}
	gates, err := host.VerifySnapshotChecks(verifySnapshot, []authoring.Check{check})
	if err != nil {
		t.Fatal(err)
	}
	if len(gates) != 1 || gates[0].ExitCode != 0 {
		t.Fatalf("fixed-snapshot verifier did not pass the declared project check: %#v", gates)
	}
	verifier := records.VerifierIdentity{ID: "canonical-fixture-checks", Version: "1", Digest: host.Hash([]byte("canonical-fixture-checker"))}
	verified, err := host.VerifyCanonicalProjection(fixed, verifySnapshot, *dotnetApply.Record, verifier)
	if err != nil {
		t.Fatalf("canonical projection verification should run the exact configured fixed check: %v", err)
	}
	withoutConfiguredCheck := *fixed
	withoutConfiguredCheck.Config.Checks = nil
	if _, err := host.VerifyCanonicalProjection(&withoutConfiguredCheck, verifySnapshot, *dotnetApply.Record, verifier); err == nil || !strings.Contains(err.Error(), "check") {
		t.Fatalf("canonical verification should escalate when the fixed source omits the configured Project check: %v", err)
	}
	boundChecks := len(verified.Result.Checks) == 1 && verified.Result.Checks[0].ID == check.Name && verified.Result.Checks[0].Outcome == records.CheckPassed
	bindings := []bool{verified.EvidenceRevision == verifySnapshot.ID, verified.EvidenceSnapshotDigest == "sha256:"+verifySnapshot.Digest(), verified.Result.Outcome == records.OutcomePassed, verified.Result.RecordID == dotnetApply.Record.ID, verified.Result.Revision == fixed.Snapshot.ID, verified.Result.ModelDigest == fixed.Model.Digest, verified.Result.TargetSnapshotDigest == dotnetApply.Record.TargetSnapshotDigest, verified.Result.Verifier == verifier, boundChecks}
	for _, bound := range bindings {
		if !bound {
			t.Fatalf("canonical verification result did not retain its bindings: checksBound=%t bindingChecks=%v report=%#v", boundChecks, bindings, verified)
		}
	}
	checkEvidenceDigest := verified.Result.Checks[0].Digest
	mutatedTarget := cloneSnapshot(verifySnapshot)
	mutatedTarget.Files["src/Commerce/EffectAxis.cs"] = append(mutatedTarget.Files["src/Commerce/EffectAxis.cs"], []byte("// unreviewed target drift\n")...)
	if _, err := host.VerifyCanonicalProjection(fixed, mutatedTarget, *dotnetApply.Record, verifier); err == nil || !strings.Contains(err.Error(), "drifted") {
		t.Fatalf("canonical verification should reject a changed target artifact against the supplied record: %v", err)
	}

	checkerPath := "examples/canonical-projection/evidence/check.go"
	checkerBytes, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(checkerPath)))
	if err != nil {
		t.Fatal(err)
	}
	checkerBytes = append(checkerBytes, []byte("\n// changed checker source for evidence binding\n")...)
	writeFile(t, root, checkerPath, checkerBytes)
	checkerRevision := commitCanonicalProjection(t, root, "change only checker source for evidence binding")
	checkerSnapshot, err := source.Load(root, checkerRevision)
	if err != nil {
		t.Fatal(err)
	}
	changedCheckEvidenceDigest := host.CanonicalCheckEvidenceDigest(check, checkerSnapshot)
	if changedCheckEvidenceDigest == checkEvidenceDigest {
		t.Fatal("checker source changes must change the complete-snapshot check evidence digest")
	}
	checkerVerified, err := host.VerifyCanonicalProjection(fixed, checkerSnapshot, *dotnetApply.Record, verifier)
	if err != nil {
		t.Fatalf("the changed checker source should be verified as its own immutable evidence snapshot: %v", err)
	}
	if checkerVerified.Result.Outcome != records.OutcomePassed || len(checkerVerified.Result.Checks) != 1 || checkerVerified.Result.Checks[0].Digest != changedCheckEvidenceDigest || checkerVerified.EvidenceRevision != checkerSnapshot.ID {
		t.Fatalf("updated checker evidence was not rebound to its exact snapshot: %#v", checkerVerified)
	}

	// The old broad enum-marker check would accept this candidate, while the
	// request-bound fixture check rejects it because the canonical boundary is absent.
	badCandidate := []byte("namespace Commerce.Domain;\npublic enum EffectAxis { Inventory, Payment }\n")
	for _, marker := range []string{"enum EffectAxis", "Inventory", "Payment"} {
		if !strings.Contains(string(badCandidate), marker) {
			t.Fatalf("negative candidate is missing old blind-check marker %q", marker)
		}
	}
	writeFile(t, root, "src/Commerce/EffectAxis.cs", badCandidate)
	negativeCommit := commitCanonicalProjection(t, root, "record a deliberately inadequate candidate")
	negativeSnapshot, err := source.Load(root, negativeCommit)
	if err != nil {
		t.Fatal(err)
	}
	negativeGates, err := host.VerifySnapshotChecks(negativeSnapshot, []authoring.Check{check})
	if err == nil || len(negativeGates) != 1 || negativeGates[0].ExitCode != 1 {
		t.Fatalf("independent fixed-snapshot check should reject the enum-only candidate: gates=%#v err=%v", negativeGates, err)
	}
	checkCommand := exec.Command("go", "run", "examples/canonical-projection/evidence/check.go")
	checkCommand.Dir = root
	checkOutput, checkErr := checkCommand.CombinedOutput()
	if checkErr == nil || !strings.Contains(string(checkOutput), "explicit application boundary") {
		t.Fatalf("bounded checker should explain why the enum-only candidate fails: output=%q err=%v", checkOutput, checkErr)
	}
	gitProjection(t, root, "checkout", "codex/canonical-projection-e2e")

	// Drift changes only a target artifact. Reusing the same fixed canonical
	// source and model digest demonstrates repair without rewriting intent.
	driftedContent := []byte("namespace Commerce.Domain;\npublic enum EffectAxis { Corrupted }\n")
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash("src/Commerce/EffectAxis.cs")), driftedContent, 0644); err != nil {
		t.Fatal(err)
	}
	drifted, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	driftIndex := buildExampleOwnershipIndex(t, fixed, active, drifted)
	assertOwnershipStatus(t, driftIndex, "src/Commerce/EffectAxis.cs", records.OwnershipDrift)
	driftPlan, err := host.PlanCanonicalReconciliation(fixed, fixed, drifted, active)
	if err != nil {
		t.Fatal(err)
	}
	if len(driftPlan.Work) != 1 || driftPlan.Work[0].ProjectionID != dotnetID.Key() || !containsString(driftPlan.Work[0].Reasons, "projection-drift") {
		t.Fatalf("drift should propose work only for the owning .NET Projection: %#v", driftPlan.Work)
	}
	if !containsString(driftPlan.NoApplicableWork, markdownID.Key()) {
		t.Fatalf("unmodified Markdown Projection should have no applicable work: %#v", driftPlan.NoApplicableWork)
	}
	if fixed.Model.Digest != initialCanonicalDigest {
		t.Fatal("target drift changed the fixed canonical Model")
	}

	driftBinding, err := host.PrepareCanonicalProjection(fixed, drifted, dotnetID, "example-tool/1", host.Hash([]byte("canonical-projection-example")), nil, check)
	if err != nil {
		t.Fatal(err)
	}
	repairCandidate := candidateEnvelope(t, driftBinding.Request.RequestDigest)
	repair, err := host.PrepareCanonicalProjection(fixed, drifted, dotnetID, "example-tool/1", host.Hash([]byte("canonical-projection-example")), repairCandidate, check)
	if err != nil {
		t.Fatal(err)
	}
	if repair.Plan == nil || len(repair.Escalations) != 0 || repair.Request.ModelDigest != initialCanonicalDigest {
		t.Fatalf("target repair did not bind the unchanged canonical Model: %#v", repair.Escalations)
	}
	repaired, err := host.ApplyCanonicalProjection(root, fixed, drifted, repair, repair.CandidateDigest, true)
	if err != nil {
		t.Fatal(err)
	}
	if repaired.Status != records.StateMaterializedUnverified || repaired.Record == nil {
		t.Fatalf("repair did not append new unverified provenance: %#v", repaired)
	}
	persistProjectionRecord(t, root, *repaired.Record)
	afterRepairRecord, err := os.ReadFile(firstDotnetRecordPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstRecordBytes, afterRepairRecord) {
		t.Fatal("repair rewrote the prior immutable ProjectionRecord")
	}

	active = []records.ProjectionRecord{*markdownApply.Record, *repaired.Record}
	repairedSnapshot, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	repairedIndex := buildExampleOwnershipIndex(t, fixed, active, repairedSnapshot)
	assertOwnershipStatus(t, repairedIndex, "src/Commerce/EffectAxis.cs", records.OwnershipManaged)
	for _, path := range []string{"examples/canonical-projection/canonical.yaml", "examples/canonical-projection/definitions/create-order.effect-axis.yaml"} {
		if !bytes.Equal(fixed.Snapshot.Files[path], repairedSnapshot.Files[path]) {
			t.Fatalf("target repair mutated canonical source %s", path)
		}
	}
	repairCommit := commitCanonicalProjection(t, root, "repair target drift without changing canonical intent")
	repairedVerifySnapshot, err := source.Load(root, repairCommit)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := host.VerifySnapshotChecks(repairedVerifySnapshot, []authoring.Check{check}); err != nil {
		t.Fatalf("independent fixed-snapshot check failed after repair: %v", err)
	}
}

func canonicalProjectionFixtureRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(harnessRepositoryRoot(t), "examples", "canonical-projection")
}

func copyCanonicalProjectionFixture(t *testing.T, root string) {
	t.Helper()
	sourceRoot := canonicalProjectionFixtureRoot(t)
	targetRoot := filepath.Join(root, "examples", "canonical-projection")
	if err := filepath.WalkDir(sourceRoot, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(sourceRoot, name)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
}

func initCanonicalProjectionGit(t *testing.T, root string) {
	t.Helper()
	gitProjection(t, root, "init", "--initial-branch=codex/canonical-projection-e2e")
	gitProjection(t, root, "config", "core.autocrlf", "false")
}

func commitCanonicalProjection(t *testing.T, root, message string) string {
	t.Helper()
	gitProjection(t, root, "add", "--all")
	if strings.TrimSpace(gitProjection(t, root, "status", "--porcelain")) == "" {
		return strings.TrimSpace(gitProjection(t, root, "rev-parse", "HEAD"))
	}
	recordDirectory := filepath.Join(root, ".markitect", "projections", "records")
	if info, err := os.Stat(recordDirectory); err == nil && info.IsDir() {
		gitProjection(t, root, "add", "--force", "--all", ".markitect/projections/records")
	}
	gitProjection(t, root, "-c", "user.name=Markitect Example", "-c", "user.email=markitect-example@example.invalid", "-c", "commit.gpgsign=false", "commit", "-m", message)
	return strings.TrimSpace(gitProjection(t, root, "rev-parse", "HEAD"))
}

func gitProjection(t *testing.T, root string, args ...string) string {
	t.Helper()
	commandArgs := append([]string{"-C", root}, args...)
	command := exec.Command("git", commandArgs...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}

func candidateEnvelope(t *testing.T, requestDigest string) []byte {
	t.Helper()
	candidate := host.CanonicalCandidate{
		RequestDigest: requestDigest,
		Files: []host.CanonicalCandidateFile{
			{Path: "src/Commerce/Commerce.csproj", Content: "<Project Sdk=\"Microsoft.NET.Sdk\">\n<PropertyGroup><TargetFramework>net8.0</TargetFramework></PropertyGroup>\n</Project>\n"},
			{Path: "src/Commerce/CreateOrderHandler.cs", Content: "namespace Commerce.Application;\npublic sealed class CreateOrderHandler { public void Handle() {} }\n"},
			{Path: "src/Commerce/EffectAxis.cs", Content: "namespace Commerce.Domain;\npublic sealed record EffectAxis { public string Boundary { get; init; } = \"application\"; }\n"},
		},
	}
	data, err := json.Marshal(candidate)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func persistProjectionRecord(t *testing.T, root string, record records.ProjectionRecord) {
	t.Helper()
	data, err := records.AppendPayload(record)
	if err != nil {
		t.Fatal(err)
	}
	directory := filepath.Join(root, ".markitect", "projections", "records")
	if err := os.MkdirAll(directory, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, strings.TrimPrefix(record.ID, "sha256:")+".json"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func buildExampleOwnershipIndex(t *testing.T, fixed *host.CanonicalSource, active []records.ProjectionRecord, observed *snapshot.Snapshot) records.OwnershipIndex {
	t.Helper()
	projected := map[string]bool{}
	for _, record := range active {
		for _, artifact := range record.Artifacts {
			projected[artifact.Path] = true
		}
	}
	canonical := map[string]bool{fixed.ConfigPath: true}
	for _, definition := range fixed.Config.Definitions {
		canonical[definition] = true
	}
	for _, module := range fixed.Config.Modules {
		canonical[module.Manifest] = true
		for _, file := range module.Files {
			canonical[file] = true
		}
	}
	facts := make([]records.ArtifactFact, 0, len(observed.Files))
	for path, content := range observed.Files {
		role, reason := records.RoleUnknown, ""
		switch {
		case projected[path]:
			role = records.RoleProjectionTarget
		case canonical[path]:
			role = records.RoleCanonicalSource
		case path == "docs/legacy/old.md":
			role, reason = records.RoleExcluded, "outside the canonical projection slice"
		case strings.HasPrefix(path, ".markitect/") || path == "examples/canonical-projection/evidence/check.go":
			role = records.RoleToolOwned
		}
		digest := sha256.Sum256(content)
		mode := observed.Modes[path]
		if mode == "" {
			mode = snapshot.RegularMode
		}
		facts = append(facts, records.ArtifactFact{
			Path: path, Role: role, Reason: reason,
			Digest: "sha256:" + hex.EncodeToString(digest[:]), Mode: mode,
		})
	}
	index, err := records.BuildOwnershipIndex(active, facts)
	if err != nil {
		t.Fatal(err)
	}
	return index
}

func assertOwnershipStatus(t *testing.T, index records.OwnershipIndex, path, expected string) {
	t.Helper()
	entry, ok := index.Artifacts[path]
	if !ok || entry.Status != expected {
		t.Fatalf("artifact %s status=%q exists=%v, want %q", path, entry.Status, ok, expected)
	}
}

func cloneSnapshot(input *snapshot.Snapshot) *snapshot.Snapshot {
	clone := &snapshot.Snapshot{ID: input.ID, Provisional: input.Provisional, Files: map[string][]byte{}, Modes: map[string]string{}}
	for path, content := range input.Files {
		clone.Files[path] = append([]byte(nil), content...)
	}
	for path, mode := range input.Modes {
		clone.Modes[path] = mode
	}
	return clone
}

func writeFile(t *testing.T, root, relative string, content []byte) {
	t.Helper()
	name := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(name), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(name, content, 0644); err != nil {
		t.Fatal(err)
	}
}

func hasEscalation(escalations []host.CanonicalProjectionEscalation, code string) bool {
	for _, escalation := range escalations {
		if escalation.Code == code {
			return true
		}
	}
	return false
}

func sortedMapKeys(values map[string][]byte) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	return keys
}

var _ = fmt.Sprintf
