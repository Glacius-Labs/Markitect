package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/host/records"
)

func TestCanonicalControllerVerifyApplyResultOptionBoundary(t *testing.T) {
	const base = "0123456789abcdef0123456789abcdef01234567"
	const revision = "89abcdef0123456789abcdef0123456789abcdef"
	common := []string{
		"--action", "controller-verify",
		"--repo", ".",
		"--config", "canonical.yaml",
		"--runtime", "runtime.json",
	}
	parse := func(command string, args ...string) (commandOptions, int) {
		t.Helper()
		options, code, done := parseOptions(command, append([]string{command}, args...), mustCommandFlags(command), &bytes.Buffer{}, &bytes.Buffer{})
		if !done {
			return options, 0
		}
		return options, code
	}

	applyResult := append(append([]string{}, common...), "--apply-result", "applied.json")
	if options, code := parse("canonical", applyResult...); code != 0 || options.applyResult != "applied.json" {
		t.Fatalf("saved Apply result options rejected: code=%d options=%#v", code, options)
	}
	legacy := append(append([]string{}, common...), "--base", base, "--revision", revision)
	if options, code := parse("canonical", legacy...); code != 0 || options.base != base || options.revision != revision {
		t.Fatalf("legacy explicit-reference verify options regressed: code=%d options=%#v", code, options)
	}

	for _, test := range []struct {
		name    string
		command string
		args    []string
	}{
		{"apply result path required", "canonical", append(append([]string{}, common...), "--apply-result=")},
		{"empty base still conflicts", "canonical", append(append([]string{}, applyResult...), "--base=")},
		{"empty revision still conflicts", "canonical", append(append([]string{}, applyResult...), "--revision=")},
		{"full base conflicts", "canonical", append(append([]string{}, applyResult...), "--base", base)},
		{"full revision conflicts", "canonical", append(append([]string{}, applyResult...), "--revision", revision)},
		{"controller proposal rejects flag", "canonical", append(replaceArg(common, "controller-verify", "controller-propose"), "--apply-result=")},
		{"canonical model rejects flag", "canonical", []string{"--action", "model", "--apply-result="}},
		{"other command rejects flag", "check", []string{"--apply-result="}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, code := parse(test.command, test.args...); code != 2 {
				t.Fatalf("invalid apply-result options returned code %d", code)
			}
		})
	}
}

func TestCanonicalControllerVerifyUsesSavedApplyResultRevisionBindings(t *testing.T) {
	temp := t.TempDir()
	runtimePath := writeCanonicalControllerApplyResultTestRuntime(t, temp)
	sourceRevision := strings.Repeat("a", 40)
	evidenceRevision := strings.Repeat("b", 40)
	apply := canonicalControllerApplyResultForTest(t, sourceRevision, evidenceRevision)
	applyBytes, err := json.Marshal(apply)
	if err != nil {
		t.Fatal(err)
	}
	applyResultPath := filepath.Join(temp, "applied.json")
	if err := os.WriteFile(applyResultPath, applyBytes, 0600); err != nil {
		t.Fatal(err)
	}
	args := []string{
		"canonical", "--action", "controller-verify", "--repo", filepath.Join(temp, "missing-repository"),
		"--config", "canonical.yaml", "--runtime", runtimePath, "--apply-result", applyResultPath,
	}
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code == 0 {
		t.Fatalf("verification of a missing repository unexpectedly succeeded: stdout=%s stderr=%s", out.String(), errout.String())
	}
	var verification host.CanonicalControllerVerification
	if err := json.Unmarshal(out.Bytes(), &verification); err != nil {
		t.Fatalf("decode CLI verification report: %v; output=%s; stderr=%s", err, out.String(), errout.String())
	}
	if verification.SourceRevision != sourceRevision || verification.EvidenceRevision != evidenceRevision {
		t.Fatalf("Verify received revisions %q/%q, want saved Apply report values %q/%q", verification.SourceRevision, verification.EvidenceRevision, sourceRevision, evidenceRevision)
	}
}

func TestCanonicalControllerVerifyDecodesApplyResultBeforeRepositoryActivity(t *testing.T) {
	temp := t.TempDir()
	runtimePath := writeCanonicalControllerApplyResultTestRuntime(t, temp)
	applyResultPath := filepath.Join(temp, "apply.json")
	if err := os.WriteFile(applyResultPath, []byte(`{"status":"materialized-unverified"`), 0600); err != nil {
		t.Fatal(err)
	}
	missingRoot := filepath.Join(temp, "missing-repository")
	args := []string{
		"canonical", "--action", "controller-verify", "--repo", missingRoot,
		"--config", "canonical.yaml", "--runtime", runtimePath, "--apply-result", applyResultPath,
	}
	var out, errout bytes.Buffer
	if code := Run(args, &out, &errout); code == 0 {
		t.Fatalf("malformed saved Apply result unexpectedly succeeded: stdout=%s stderr=%s", out.String(), errout.String())
	}
	if !strings.Contains(errout.String(), "decode saved controller apply result JSON") {
		t.Fatalf("CLI did not reject the saved Apply result before repository processing: stdout=%s stderr=%s", out.String(), errout.String())
	}
}

func writeCanonicalControllerApplyResultTestRuntime(t *testing.T, directory string) string {
	t.Helper()
	runner := host.CanonicalRunnerConfig{
		Command: "provider-must-not-run", Args: []string{}, Model: "test-model", ModelOptions: json.RawMessage("null"),
		ProviderVersion: "test", TimeoutSeconds: 1, MaxStdoutBytes: 1024, MaxStderrBytes: 1024,
	}
	config := host.CanonicalControllerConfig{
		APIVersion: host.CanonicalControllerAPIVersion, RecordStore: filepath.Join(directory, "records"),
		PrivateLogs: filepath.Join(directory, "logs"), CheckInputs: []string{}, Executor: runner, Verifier: runner,
	}
	runtimeBytes, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	runtimePath := filepath.Join(directory, "runtime.json")
	if err := os.WriteFile(runtimePath, runtimeBytes, 0600); err != nil {
		t.Fatal(err)
	}
	return runtimePath
}

func canonicalControllerApplyResultForTest(t *testing.T, sourceRevision, evidenceRevision string) host.CanonicalControllerApply {
	t.Helper()
	const sha = "sha256:" + "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"
	record, err := records.NewProjectionRecord(records.ProjectionRecord{
		Revision: sourceRevision, ModelDigest: sha, PlanDigest: sha, InputSnapshotDigest: sha, RequestDigest: sha,
		Module:       records.ModuleIdentity{Name: "test-module", Version: "1", Digest: sha},
		ProjectionID: "test/v1:Projection:orders", Projector: records.ProjectorIdentity{ID: "test", Version: "1"},
		ScopeIDs: []string{"test/v1:Scope:orders"}, PolicyIDs: []string{"test/v1:Rule:orders"},
		Artifacts: []records.Artifact{{Path: "docs/orders.md", Digest: sha, Mode: "100644", Change: records.ChangeCreated}},
		State:     records.StateMaterializedUnverified,
	})
	if err != nil {
		t.Fatalf("create Apply receipt record: %v", err)
	}
	return host.CanonicalControllerApply{
		Status: records.StateMaterializedUnverified, SourceRevision: sourceRevision,
		RunDigest: sha, LedgerHead: sha, EvidenceRevision: evidenceRevision, Records: []records.ProjectionRecord{record},
		EvidenceRefreshRequired: []string{"test/v1:Projection:retained-parent"},
	}
}
