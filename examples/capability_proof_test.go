package examples

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/canonical"
	"github.com/Glacius-Labs/Markitect/internal/host/cli"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"go.yaml.in/yaml/v3"
)

const proofUnrelatedPath = "unrelated/Billing/review-note.txt"

// Passing this test reproduces a negative capability result, not C5 success.
// Keep the frozen v1 protocol and historical observations when changing it.
func TestCapabilityProofGlobalAcquisitionStopWitness(t *testing.T) {
	root := filepath.Join(t.TempDir(), "repo")
	if err := os.MkdirAll(root, 0755); err != nil {
		t.Fatal(err)
	}
	copyCanonicalProjectionFixture(t, root)
	// The new Git repository must not normalize bytes differently by platform.
	writeFile(t, root, ".gitattributes", []byte("* -text\n"))
	unrelated := []byte("Capability proof v1: unrelated Billing note, never a selected canonical input or projection target.\n")
	writeFile(t, root, proofUnrelatedPath, unrelated)
	writeFile(t, root, "src/Commerce/Existing.cs", []byte("// Unverified operational specimen; no business correctness is claimed.\n"))
	writeFile(t, root, "docs/represented/index.md", []byte("Unverified operational specimen.\n"))
	proofGit(t, root, "init", "--template=", "--object-format=sha1", "--initial-branch=codex/capability-proof")
	proofGit(t, root, "config", "core.autocrlf", "false")
	baseRevision := proofCommit(t, root, "Freeze capability proof seed")

	// This is an actual canonical edit, with unchanged module packages/pins.
	policyPath := "examples/canonical-projection/definitions/create-order.projection-policy.yaml"
	policy, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(policyPath)))
	if err != nil {
		t.Fatal(err)
	}
	marker := []byte("Guides the representation of commerce use cases in the selected .NET target.")
	replacement := []byte("Guides idempotent representation of commerce use cases in the selected .NET target.")
	if bytes.Count(policy, marker) != 1 {
		t.Fatal("frozen policy mutation no longer matches")
	}
	writeFile(t, root, policyPath, bytes.Replace(policy, marker, replacement, 1))
	candidateRevision := proofCommit(t, root, "Require idempotent use-case representation")
	base := proofLoad(t, root, baseRevision)
	candidate := proofLoad(t, root, candidateRevision)
	observed, err := source.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	// Both Git snapshots and the working tree actually contain unrelated bytes.
	acquisitions := []map[string]any{}
	for i, s := range []*snapshot.Snapshot{base.Snapshot, candidate.Snapshot, observed} {
		if !bytes.Equal(s.Files[proofUnrelatedPath], unrelated) {
			t.Fatal("unrelated acquisition witness no longer holds")
		}
		count := 0
		paths := []string{}
		for name, data := range s.Files {
			count += len(data)
			paths = append(paths, name)
		}
		sort.Strings(paths)
		acquisitions = append(acquisitions, map[string]any{
			"stage":    []string{"base", "candidate", "working-target"}[i],
			"revision": s.ID, "snapshotDigest": s.Digest(),
			"materializedFiles": len(paths), "materializedBytes": count,
			"paths": paths, "unrelatedBytesAcquired": len(unrelated),
		})
	}
	active := proofActiveRecords(t, base)
	plan, err := host.PlanCanonicalReconciliation(base, candidate, observed, active)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Status != "planned" || len(plan.Work) != 1 || len(plan.NoApplicableWork) != 1 ||
		len(plan.EvidenceRefreshRequired) != 1 || len(plan.Impact.ScopeAffectedProjections) != 1 ||
		len(plan.Impact.ConservativeInvalidatedProjections) != 2 {
		t.Fatalf("unexpected scoped/conservative result: %+v", plan)
	}
	again, err := host.PlanCanonicalReconciliation(base, candidate, observed, active)
	if err != nil || again.Digest != plan.Digest {
		t.Fatal("non-deterministic reconciliation")
	}

	activePath := filepath.Join(t.TempDir(), "active-records.json")
	data, err := json.Marshal(active)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(activePath, data, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{"canonical", "--repo", root, "--config", canonicalProjectionConfigPath,
		"--action", "reconcile-plan", "--base", baseRevision, "--revision", candidateRevision,
		"--evidence", activePath}
	var output, diagnostics bytes.Buffer
	code := cli.Run(args, &output, &diagnostics)
	if code != 0 {
		t.Fatalf("CLI positive control=%d: %s", code, diagnostics.String())
	}
	var wire map[string]any
	if err := yaml.Unmarshal(output.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["plan"].(map[string]any)["digest"] != plan.Digest {
		t.Fatal("CLI and observed public pipeline differ")
	}
	after, err := source.Load(root, "")
	if err != nil || after.Digest() != observed.Digest() {
		t.Fatal("read-only CLI changed repository content")
	}

	// Remove one unique unrelated loose blob in this temp repository only.
	// Git ls-tree size preflight can fail before cat-file opens the missing blob.
	// This control proves whole-tree dependence; the positive snapshots separately
	// prove the unrelated content was acquired. It is not a blob-open syscall trace.
	oid := strings.TrimSpace(proofGit(t, root, "rev-parse", candidateRevision+":"+proofUnrelatedPath))
	for _, name := range candidate.Config.Definitions {
		if strings.TrimSpace(proofGit(t, root, "rev-parse", candidateRevision+":"+name)) == oid {
			t.Fatal("negative control blob aliases selected canonical bytes")
		}
	}
	objectPath := filepath.Join(root, ".git", "objects", oid[:2], oid[2:])
	relative, err := filepath.Rel(root, objectPath)
	if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		t.Fatal("refusing object mutation outside temporary fixture")
	}
	for _, directory := range []string{filepath.Join(root, ".git"), filepath.Join(root, ".git", "objects"), filepath.Dir(objectPath)} {
		info, err := os.Lstat(directory)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			t.Fatal("refusing aliased temporary Git object directory")
		}
	}
	blob, err := os.ReadFile(objectPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(objectPath); err != nil {
		t.Fatal(err)
	}
	output.Reset()
	diagnostics.Reset()
	negativeCode := cli.Run(args, &output, &diagnostics)
	negativeDiagnostic := strings.TrimSpace(diagnostics.String())
	for _, name := range candidate.Config.Definitions {
		proofGit(t, root, "show", candidateRevision+":"+name)
	}
	if err := os.WriteFile(objectPath, blob, 0444); err != nil {
		t.Fatal(err)
	}
	if negativeCode != 2 || !strings.Contains(negativeDiagnostic, proofUnrelatedPath) {
		t.Fatalf("negative control did not expose unrelated acquisition dependency: %d %s", negativeCode, negativeDiagnostic)
	}
	// Selected blobs stayed readable, and restoration returns the unchanged plan.
	for _, name := range candidate.Config.Definitions {
		proofGit(t, root, "show", candidateRevision+":"+name)
	}
	output.Reset()
	diagnostics.Reset()
	restoredCode := cli.Run(args, &output, &diagnostics)
	if restoredCode != 0 {
		t.Fatalf("restoration=%d: %s", restoredCode, diagnostics.String())
	}
	if err := yaml.Unmarshal(output.Bytes(), &wire); err != nil {
		t.Fatal(err)
	}
	if wire["plan"].(map[string]any)["digest"] != plan.Digest {
		t.Fatal("restored plan identity changed")
	}
	final, err := source.Load(root, "")
	if err != nil || final.Digest() != observed.Digest() {
		t.Fatal("fixture content changed by observation")
	}

	modulePins := append([]canonical.Pin(nil), candidate.Pins...)
	sort.Slice(modulePins, func(i, j int) bool { return modulePins[i].Name < modulePins[j].Name })
	observation := map[string]any{
		"protocol": "capability-proof/v1", "claim": "C5", "result": "FAIL",
		"productBaseline": "75031b8eca8161b0a247414741828bfb2e1d9069",
		"stopCondition":   "ordinary-reconcile-acquires-full-snapshots",
		"baseRevision":    baseRevision, "candidateRevision": candidateRevision,
		"baseModelDigest": base.Model.Digest, "candidateModelDigest": candidate.Model.Digest,
		"modulePins": modulePins, "definitionCount": len(candidate.Model.Definitions),
		"changedDefinitions":                 plan.Impact.ChangedDefinitions,
		"affectedDefinitions":                plan.Impact.AffectedDefinitions,
		"projectionBindingsConsidered":       len(candidate.Config.ProjectionBindings),
		"scopeAffectedProjections":           plan.Impact.ScopeAffectedProjections,
		"conservativeInvalidatedProjections": plan.Impact.ConservativeInvalidatedProjections,
		"workNodes":                          len(plan.Work), "evidenceRefreshRequired": plan.EvidenceRefreshRequired,
		"noApplicableWork": plan.NoApplicableWork, "planDigest": plan.Digest,
		"acquisitions": acquisitions, "derivedPlan": plan,
		"positiveCLIExit": code, "missingUnrelatedBlobCLIExit": negativeCode,
		"missingUnrelatedBlobDiagnostic": negativeDiagnostic, "negativeControlLimit": "missing-object refusal may occur at full-tree metadata preflight, not blob opening", "restoredCLIExit": restoredCode,
		"readOnlyContentDigest": final.Digest(), "artifactWritesByReconcile": 0,
		"parentChecksRun": 0, "executorRuns": 0, "verifierRuns": 0,
		"tokens": nil, "humanAttentionMinutes": nil, "syscallCounts": nil,
		"independentlyTimedAgentReads": nil,
		"limits":                       "two existing projections; third projection and end-to-end scoped execution NOT RUN at stop gate; supplied records are unverified; snapshot byte counts are not IO operation counts",
	}
	encoded, err := json.Marshal(observation)
	if err != nil {
		t.Fatal(err)
	}
	t.Log("CAPABILITY_OBSERVATION " + string(encoded))
}

func proofLoad(t *testing.T, root, revision string) *host.CanonicalSource {
	t.Helper()
	s, err := host.LoadCanonicalSource(root, revision, canonicalProjectionConfigPath, true)
	if err != nil || len(s.Diagnostics) != 0 {
		t.Fatalf("fixed canonical source: %v %+v", err, s)
	}
	return s
}

func proofActiveRecords(t *testing.T, base *host.CanonicalSource) []records.ProjectionRecord {
	t.Helper()
	active := []records.ProjectionRecord{}
	for _, binding := range base.Config.ProjectionBindings {
		targetPath := "src/Commerce/Existing.cs"
		if binding.Module == "markitect-markdown" {
			targetPath = "docs/represented/index.md"
		}
		targets := map[string][]byte{targetPath: base.Snapshot.Files[targetPath]}
		request, err := canonical.BindProjection(base.Model, base.Activation, base.Config.ProjectionBindings, binding.Projection, targets)
		if err != nil {
			t.Fatal(err)
		}
		facts := []records.ArtifactFact{{Path: targetPath, Digest: proofDigest(targets[targetPath]), Mode: snapshot.RegularMode}}
		targetDigest, err := records.TargetSnapshotDigest(facts)
		if err != nil {
			t.Fatal(err)
		}
		scopes, policies := []string{}, []string{}
		for _, d := range request.Definitions {
			scopes = append(scopes, d.Identity().Key())
		}
		for _, d := range request.Policies {
			policies = append(policies, d.Identity().Key())
		}
		record, err := records.NewProjectionRecord(records.ProjectionRecord{
			Revision: base.Model.Revision, ModelDigest: request.ModelDigest,
			PlanDigest:          proofDigest([]byte("supplied unverified specimen; no actual materialization claimed")),
			InputSnapshotDigest: "sha256:" + base.Snapshot.Digest(), RequestDigest: request.RequestDigest,
			Module:       records.ModuleIdentity{Name: request.ModulePin.Name, Version: request.ModulePin.Version, Digest: request.ModulePin.Digest},
			ProjectionID: binding.Projection.Key(), Projector: records.ProjectorIdentity{ID: request.Projector.ID, Version: request.Projector.Version},
			ScopeIDs: scopes, PolicyIDs: policies,
			Artifacts:            []records.Artifact{{Path: targetPath, Digest: facts[0].Digest, Mode: snapshot.RegularMode, Change: records.ChangeRetained}},
			TargetSnapshotDigest: targetDigest, State: records.StateMaterializedUnverified,
		})
		if err != nil {
			t.Fatal(err)
		}
		active = append(active, record)
	}
	return active
}

func proofCommit(t *testing.T, root, message string) string {
	t.Helper()
	proofGit(t, root, "add", "--all")
	proofGit(t, root, "commit", "-m", message)
	return strings.TrimSpace(proofGit(t, root, "rev-parse", "HEAD"))
}

func proofGit(t *testing.T, root string, args ...string) string {
	t.Helper()
	prefix := []string{"-C", root, "-c", "core.hooksPath=" + filepath.Join(root, ".git", "proof-empty-hooks"), "-c", "user.name=Capability Proof",
		"-c", "user.email=capability-proof@example.invalid", "-c", "commit.gpgsign=false", "-c", "core.autocrlf=false"}
	command := exec.Command("git", append(prefix, args...)...)
	command.Env = append(source.CleanGitEnv(), "GIT_AUTHOR_DATE=2000-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2000-01-01T00:00:00Z")
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v %s", args, err, output)
	}
	return string(output)
}

func proofDigest(data []byte) string {
	value := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(value[:])
}

// Retained raw evidence must remain byte-identical after Git checkout on either
// supported platform. This is integrity, not a semantic capability result.
func TestCapabilityProofEvidenceManifest(t *testing.T) {
	root := filepath.Join(filepath.Dir(canonicalProjectionFixtureRoot(t)), "..", "experiments", "capability-proof")
	data, err := os.ReadFile(filepath.Join(root, "file-manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Version int
		Files   []struct {
			Path   string
			SHA256 string
			Bytes  int
		}
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest.Version != 1 || len(manifest.Files) == 0 {
		t.Fatal("missing evidence manifest")
	}
	for _, entry := range manifest.Files {
		if filepath.IsAbs(entry.Path) || strings.Contains(entry.Path, "..") || strings.Contains(entry.Path, "\\") {
			t.Fatal("unsafe evidence path")
		}
		data, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(entry.Path)))
		if err != nil {
			t.Fatal(err)
		}
		if len(data) != entry.Bytes || strings.TrimPrefix(proofDigest(data), "sha256:") != entry.SHA256 {
			t.Fatalf("retained evidence bytes changed: %s", entry.Path)
		}
	}
}
