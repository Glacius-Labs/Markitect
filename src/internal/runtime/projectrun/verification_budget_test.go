package projectrun

import (
	"context"
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/src/internal/runtime/projectexecutor"
	"io"
	"os"
	"os/exec"
	"sort"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/runtime/agentexec"
)

const verifierProcessHelperEnv = "MARKITECT_PROJECTRUN_VERIFY_HELPER"
const unallowlistedEnvironmentSentinel = "MARKITECT_PROJECTRUN_UNALLOWLISTED_SENTINEL"

func TestProjectRunEmptyEnvironmentHelperProcess(t *testing.T) {
	for _, arg := range os.Args {
		if arg == "-test.run=^TestProjectRunEmptyEnvironmentHelperProcess$" {
			if os.Getenv(unallowlistedEnvironmentSentinel) != "" {
				t.Fatal("unallowlisted environment sentinel reached child process")
			}
			return
		}
	}
}

func TestExplicitEnvironmentWithUnsetAllowlistDoesNotInherit(t *testing.T) {
	t.Setenv(unallowlistedEnvironmentSentinel, "must-not-reach-child")
	cmd := exec.Command(os.Args[0], "-test.run=^TestProjectRunEmptyEnvironmentHelperProcess$")
	cmd.Env = explicitEnvironment([]string{"MARKITECT_PROJECTRUN_NOT_SET"})
	if cmd.Env == nil {
		t.Fatal("unset allowlist produced nil environment, which means inherit-all")
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("empty allowlisted child inherited a caller variable: %v\n%s", err, output)
	}
}

// TestProjectRunVerifierHelperProcess is invoked as a real agentexec child by
// TestFailedVerifierAttemptIsDurableAndCannotReplayBudget. Its failed response
// is still protocol-valid, so the projectrun verifier's role/outcome validation
// runs without contacting a provider.
func TestProjectRunVerifierHelperProcess(t *testing.T) {
	if os.Getenv(verifierProcessHelperEnv) != "failed-verifier" {
		return
	}
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		t.Fatal(err)
	}
	var invocation agentexec.Invocation
	if err := json.Unmarshal(data, &invocation); err != nil {
		t.Fatal(err)
	}
	refs := append(append([]string(nil), invocation.Request.ScopeIDs...), invocation.Request.PolicyIDs...)
	sort.Strings(refs)
	unique := refs[:0]
	for _, ref := range refs {
		if len(unique) == 0 || unique[len(unique)-1] != ref {
			unique = append(unique, ref)
		}
	}
	inTokens, outTokens := int64(1_000_000), int64(0)
	response := agentexec.Response{
		APIVersion: invocation.APIVersion, RunID: invocation.RunID, Nonce: invocation.Nonce,
		Role: agentexec.RoleVerifier, InputDigest: invocation.InputDigest, Outcome: agentexec.OutcomeFailed,
		CandidateFiles: []agentexec.CandidateFile{}, EvidenceRefs: unique,
		VerifierObservations: []agentexec.Observation{{Subject: "check-1", Outcome: "failed", Detail: "fixture verifier rejected the result"}},
		Uncertainty:          []string{"fixture failure"},
		Usage:                &agentexec.Usage{Source: "provider-reported", InputTokens: &inTokens, OutputTokens: &outTokens},
	}
	if err := json.NewEncoder(os.Stdout).Encode(response); err != nil {
		t.Fatal(err)
	}
	os.Exit(0)
}

type countingProcessInvoker struct {
	calls int
}

func (i *countingProcessInvoker) Run(ctx context.Context, config agentexec.Config, request agentexec.Request, options agentexec.RunOptions) (agentexec.RunResult, error) {
	i.calls++
	return (projectexecutor.ProcessInvoker{}).Run(ctx, config, request, options)
}

func (i *countingProcessInvoker) Fingerprint(config agentexec.Config) (string, error) {
	return (projectexecutor.ProcessInvoker{}).Fingerprint(config)
}

func git(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", args...)
	command.Dir = root
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
	return string(output)
}
