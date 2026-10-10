package githooks

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/core"
)

func fixtureInput() Input {
	const api = "example.test/v1"
	definition := core.Definition{
		APIVersion: api, Kind: "Build", Metadata: core.Metadata{Namespace: "delivery", Name: "local"},
		Purpose: "A locally verified build.", Spec: map[string]any{},
	}
	policy := core.Definition{
		APIVersion: "markitect.foundation/v1", Kind: "ProjectionPolicy",
		Metadata: core.Metadata{Namespace: "delivery", Name: "build-to-hooks"},
		Purpose:  "Run required local checks.", Spec: map[string]any{
			"sourceKind":       map[string]any{"apiVersion": api, "kind": "Build"},
			"targetTechnology": "githooks",
			"guidance":         "Keep the canonical-workflow-check active.\n$(do not execute guidance)",
		},
	}
	return Input{
		Definitions: []core.Definition{definition},
		Schemas: []core.Schema{{APIVersion: api, Purpose: "Delivery contracts.", Kinds: map[string]core.Kind{
			"Build": {Purpose: "A build verification target."},
		}}},
		Policies:     []core.Definition{policy},
		TargetPrefix: ".githooks/",
		AllowedRoots: []string{".githooks"},
		RequiredChecks: []NamedCheck{{Name: "canonical-workflow-check", Argv: []string{
			"git", "diff", "--cached", "--check", "--", "file with spaces.txt",
		}}},
		RequestDigest:     "sha256:" + strings.Repeat("1", 64),
		InventoryComplete: true,
	}
}

func TestProposeRendersExecutableHookWithExactCheckAndInertPolicyProvenance(t *testing.T) {
	input := fixtureInput()
	input.Definitions[0].Purpose = "Review architecture first.\n$(do not execute canonical prose)"
	got := Propose(input)
	if got.Decision != DecisionWork || len(got.Files) != 1 {
		t.Fatalf("proposal = %#v; want one work file", got)
	}
	file := got.Files[0]
	if file.Path != ".githooks/pre-commit" || file.Mode != "100755" || file.Digest != digest(file.Content) || file.RequestDigest != input.RequestDigest {
		t.Fatalf("candidate binding = %#v", file)
	}
	text := string(file.Content)
	for _, want := range []string{
		"#!/bin/sh",
		"ProjectionPolicy:",
		"# canonical-definition:",
		"Review architecture first.\\n$(do not execute canonical prose)",
		"\"targetTechnology\":\"githooks\"",
		"\"guidance\":\"Keep the canonical-workflow-check active.\\n$(do not execute guidance)\"",
		"# Required Project check: {\"Name\":\"canonical-workflow-check\",\"Argv\":[\"git\",\"diff\",\"--cached\",\"--check\",\"--\",\"file with spaces.txt\"]}",
		"'git' 'diff' '--cached' '--check' '--' 'file with spaces.txt'",
		"set -eu",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("hook missing %q:\n%s", want, text)
		}
	}
	if strings.Contains(text, "\n$(do not execute guidance)") || strings.Contains(text, "\n$(do not execute canonical prose)") {
		t.Fatalf("guidance was emitted outside escaped provenance: %s", text)
	}
}

func TestProposeEscalatesWhenPolicyChecksInventoryOrOwnershipAreInsufficient(t *testing.T) {
	tests := []struct {
		name string
		edit func(*Input)
		code string
	}{
		{"no policy", func(i *Input) { i.Policies = nil }, "policy.invalid"},
		{"no required checks", func(i *Input) { i.RequiredChecks = nil }, "checks.invalid"},
		{"empty argv", func(i *Input) { i.RequiredChecks[0].Argv = nil }, "checks.invalid"},
		{"unbounded inventory", func(i *Input) { i.InventoryComplete = false }, "target.inventory.incomplete"},
		{"unowned existing target", func(i *Input) {
			i.ObservedArtifacts = []ArtifactObservation{{Path: ".githooks/pre-commit", Bytes: []byte("manual"), Mode: "100755"}}
		}, "artifact.unowned"},
		{"target outside allowed root", func(i *Input) { i.AllowedRoots = []string{"scripts"} }, "target.invalid"},
		{"policy for wrong target", func(i *Input) { i.Policies[0].Spec["targetTechnology"] = "markdown" }, "policy.invalid"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			input := fixtureInput()
			test.edit(&input)
			got := Propose(input)
			if got.Decision != DecisionEscalate || len(got.Files) != 0 || len(got.Escalations) == 0 || got.Reasons[0] != test.code {
				t.Fatalf("proposal = %#v; want escalation %s", got, test.code)
			}
		})
	}
}

func TestProposeEscalatesPortableCaseAliasesBeforeWorkOrNoOp(t *testing.T) {
	for _, observedPath := range []string{
		".githooks/Pre-Commit",
		".GitHooks/pre-commit",
		".GitHooks/other-hook",
	} {
		t.Run(observedPath, func(t *testing.T) {
			input := fixtureInput()
			input.ObservedArtifacts = []ArtifactObservation{{
				Path: observedPath, Bytes: []byte("manual"), Mode: "100755",
			}}
			got := Propose(input)
			if got.Decision != DecisionEscalate || got.Reasons[0] != "target.inventory.invalid" {
				t.Fatalf("case alias proposal = %#v; want escalation", got)
			}
			if len(got.Escalations) != 1 ||
				(!strings.Contains(got.Escalations[0].Message, "alias") && !strings.Contains(got.Escalations[0].Message, "overlap")) {
				t.Fatalf("case alias was not identified before materialization: %#v", got.Escalations)
			}
		})
	}
}

func TestPortableRepositoryPathsRejectGitAndWindowsAliases(t *testing.T) {
	for _, value := range []string{
		".git", ".GIT/config", "folder/.Git/HEAD", "CON.txt", "aux", "COM1.log", "LPT9.ext",
		"trailing.", "trailing ", "nul\\x00byte", "has:stream", "bad?name", "back\\\\slash",
		string([]byte{0xff}),
	} {
		if validRepoPath(value, false) {
			t.Errorf("unsafe portable path %q was accepted", value)
		}
	}
	for _, prefix := range []string{
		".git/", ".GIT", "hooks/CON.txt", "hooks/com9.ext", "hooks/trailing.",
		"hooks/trailing ", "hooks/nul\\x00byte", "hooks/a|b",
	} {
		if _, _, err := safeTargetPath(prefix, []string{"."}); err == nil {
			t.Errorf("unsafe target prefix %q was accepted", prefix)
		}
	}
	if _, _, err := safeTargetPath(".githooks/", []string{".githooks"}); err != nil {
		t.Fatalf("canonical Git Hooks prefix was rejected: %v", err)
	}
	longPrefix := strings.Repeat("a", 1024) + "/"
	if _, _, err := safeTargetPath(longPrefix, []string{"."}); err == nil {
		t.Fatal("overlong slash-terminated target prefix was accepted")
	}
}

func TestProposeNoOpRequiresExactCompleteOwnershipAndVerification(t *testing.T) {
	input := fixtureInput()
	first := Propose(input)
	if first.Decision != DecisionWork || len(first.Files) != 1 {
		t.Fatalf("initial proposal = %#v", first)
	}
	file := first.Files[0]
	binding := ArtifactBinding{Path: file.Path, Digest: file.Digest, Mode: "100755"}
	input.Previous = &PriorProjection{RequestDigest: input.RequestDigest, Complete: true, Artifacts: []ArtifactBinding{binding}}
	input.ObservedArtifacts = []ArtifactObservation{{Path: file.Path, Bytes: file.Content, Mode: "100755"}}
	input.Verification = &VerificationBinding{Passed: true, RequestDigest: input.RequestDigest, Artifacts: []ArtifactBinding{binding}}
	got := Propose(input)
	if got.Decision != DecisionNoOp || got.EvidenceRefreshRequired || len(got.Files) != 1 || !bytes.Equal(got.Files[0].Content, file.Content) {
		t.Fatalf("verified current projection = %#v", got)
	}

	input.Verification = nil
	got = Propose(input)
	if got.Decision != DecisionNoOp || !got.EvidenceRefreshRequired {
		t.Fatalf("current bytes without current verification = %#v", got)
	}

	input.ObservedArtifacts[0].Mode = "100644"
	got = Propose(input)
	if got.Decision != DecisionWork {
		t.Fatalf("owned mode drift = %#v; want repair work", got)
	}
}

func TestProposeIsDeterministicAndRejectsUnselectedPolicyKinds(t *testing.T) {
	input := fixtureInput()
	first := Propose(input)
	input.Definitions[0].Source.Path = "source/build.yaml"
	input.Policies[0].Source.Path = "source/policy.yaml"
	second := Propose(input)
	if !reflect.DeepEqual(first, second) {
		t.Fatalf("projection depends on source provenance: first=%#v second=%#v", first, second)
	}
	input.Policies[0].Spec["sourceKind"] = map[string]any{"apiVersion": "other/v1", "kind": "Other"}
	if got := Propose(input); got.Decision != DecisionEscalate {
		t.Fatalf("out-of-scope policy was accepted: %#v", got)
	}
}

func TestGitRunsPassingAndFailingHookChecksWithExactArgv(t *testing.T) {
	git, err := exec.LookPath("git")
	if err != nil {
		t.Skip("git is unavailable; deterministic hook rendering remains covered")
	}
	if output, err := exec.Command(git, "--version").CombinedOutput(); err != nil {
		t.Skipf("git cannot run hooks in this environment: %v: %s", err, output)
	}

	root := t.TempDir()
	runGit := func(args ...string) ([]byte, error) {
		t.Helper()
		command := exec.Command(git, args...)
		command.Dir = root
		command.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1", "GIT_TERMINAL_PROMPT=0")
		return command.CombinedOutput()
	}
	for _, args := range [][]string{
		{"init", "-q"},
		{"config", "user.name", "Projection Test"},
		{"config", "user.email", "projection@example.invalid"},
	} {
		if output, err := runGit(args...); err != nil {
			t.Fatalf("git %v failed: %v: %s", args, err, output)
		}
	}
	hookDir := filepath.Join(root, ".githooks")
	if err := os.MkdirAll(hookDir, 0o755); err != nil {
		t.Fatal(err)
	}
	input := fixtureInput()
	file := Propose(input).Files[0]
	hookPath := filepath.Join(root, filepath.FromSlash(file.Path))
	if err := os.WriteFile(hookPath, file.Content, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(hookPath, 0o755); err != nil {
		t.Fatal(err)
	}

	const sourcePath = "file with spaces.txt"
	path := filepath.Join(root, sourcePath)
	if err := os.WriteFile(path, []byte("bad trailing space \n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := runGit("add", "--", sourcePath); err != nil {
		t.Fatalf("git add failed: %v: %s", err, output)
	}
	hookConfig := []string{"-c", "core.hooksPath=" + hookDir}
	output, err := runGit(append(hookConfig, "commit", "-m", "expected hook failure")...)
	if err == nil || !strings.Contains(string(output), "trailing whitespace") {
		t.Fatalf("failing check did not block commit: err=%v output=%s", err, output)
	}
	if _, err := os.Stat(filepath.Join(root, ".git", "HEAD")); err != nil {
		t.Fatalf("git repository was not initialized: %v", err)
	}

	if err := os.WriteFile(path, []byte("clean line\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if output, err := runGit("add", "--", sourcePath); err != nil {
		t.Fatalf("git add clean file failed: %v: %s", err, output)
	}
	output, err = runGit(append(hookConfig, "commit", "-m", "expected hook success")...)
	if err != nil {
		t.Fatalf("passing exact check argv did not allow commit: %v: %s", err, output)
	}
}

func TestRenderIsCandidateOnlyAndProposeStillRefusesUnknownOwnership(t *testing.T) {
	input := fixtureInput()
	input.InventoryComplete = false
	input.ObservedArtifacts = []ArtifactObservation{{Path: ".githooks/pre-commit", Bytes: []byte("unowned"), Mode: ArtifactMode}}
	if got := Render(input); got.Decision != DecisionWork || len(got.Files) != 1 {
		t.Fatalf("pure rendering = %#v", got)
	}
	input.InventoryComplete = true
	if got := Propose(input); got.Decision != DecisionEscalate || got.Reasons[0] != "artifact.unowned" {
		t.Fatalf("rendering bypassed ownership: %#v", got)
	}
}

func TestDirectoryRootConventionPreservesAliasRefusal(t *testing.T) {
	target, _, err := safeTargetPath(".githooks/", []string{".githooks/"})
	if err != nil || target != ".githooks/pre-commit" {
		t.Fatalf("directory root: target=%s err=%v", target, err)
	}
	for _, roots := range [][]string{{".githooks//"}, {".githooks", ".githooks/"}, {".githooks/../other"}} {
		if _, _, err := safeTargetPath(".githooks", roots); err == nil {
			t.Fatalf("unsafe or duplicate root accepted: %v", roots)
		}
	}
}
