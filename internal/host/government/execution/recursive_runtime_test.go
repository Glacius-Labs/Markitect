package execution

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/host/government"
)

func TestRecursiveRuntimeValidatesBoundsIdentityAndGlobalSlots(t *testing.T) {
	runtime := testRuntime(t)
	runtime.Recursion = validRecursiveRuntime()
	if err := ValidateRuntime(runtime); err != nil {
		t.Fatal(err)
	}

	invalid := runtime
	copyRecursion := *runtime.Recursion
	invalid.Recursion = &copyRecursion
	invalid.Recursion.Limits.MaxDepth = maxRecursionDepth + 1
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "maxDepth") {
		t.Fatalf("excessive depth should fail clearly, got %v", err)
	}
	invalid = runtime
	copyRecursion = *runtime.Recursion
	invalid.Recursion = &copyRecursion
	invalid.Recursion.Parallelism = maxRecursionParallel + 1
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "parallelism") {
		t.Fatalf("excessive parallelism should fail clearly, got %v", err)
	}
	invalid = runtime
	copyRecursion = *runtime.Recursion
	copyRecursion.Areas = append([]AreaRunner(nil), runtime.Recursion.Areas...)
	copyRecursion.Areas[1].Executor.SlotID = runtime.Executor.SlotID
	invalid.Recursion = &copyRecursion
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "assigned more than once") {
		t.Fatalf("global slot reuse should fail, got %v", err)
	}
	invalid = runtime
	copyRecursion = *runtime.Recursion
	copyRecursion.Areas = append([]AreaRunner(nil), runtime.Recursion.Areas...)
	copyRecursion.Areas[1].Area = copyRecursion.Areas[0].Area
	invalid.Recursion = &copyRecursion
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "duplicates an Area") {
		t.Fatalf("duplicate Area should fail, got %v", err)
	}
	invalid = runtime
	copyRecursion = *runtime.Recursion
	copyRecursion.Areas = append([]AreaRunner(nil), runtime.Recursion.Areas...)
	copyRecursion.Areas[0].Checks = nil
	invalid.Recursion = &copyRecursion
	if err := ValidateRuntime(invalid); err == nil || !strings.Contains(err.Error(), "checks must contain") {
		t.Fatalf("Area without composition checks should fail, got %v", err)
	}
}

func TestRecursiveRuntimeDecodeRejectsNestedNullCheckTimeout(t *testing.T) {
	runtime := testRuntime(t)
	runtime.Recursion = validRecursiveRuntime()
	runtime.Recursion.Areas[0].Checks[0].TimeoutSeconds = intPointer(600)
	encoded, err := json.Marshal(runtime)
	if err != nil {
		t.Fatal(err)
	}
	withNull := strings.Replace(string(encoded), `"timeoutSeconds":600`, `"timeoutSeconds":null`, 1)
	if withNull == string(encoded) {
		t.Fatal("test did not inject explicit nested null timeout")
	}
	if _, err := DecodeRuntime([]byte(withNull)); err == nil || !strings.Contains(err.Error(), "recursion.areas[0].checks[0].timeoutSeconds") {
		t.Fatalf("nested null timeout should be rejected with path, got %v", err)
	}
}

func TestRecursiveRuntimeFingerprintBindsActorsLimitsAndChecks(t *testing.T) {
	runtime := testRuntime(t)
	runtime.Recursion = validRecursiveRuntime()
	first, err := FingerprintRuntime(runtime)
	if err != nil {
		t.Fatal(err)
	}
	permuted := runtime
	copyRecursion := *runtime.Recursion
	copyRecursion.Areas = append([]AreaRunner(nil), runtime.Recursion.Areas...)
	copyRecursion.Areas[0], copyRecursion.Areas[1] = copyRecursion.Areas[1], copyRecursion.Areas[0]
	permuted.Recursion = &copyRecursion
	permutedFingerprint, err := FingerprintRuntime(permuted)
	if err != nil || permutedFingerprint != first {
		t.Fatalf("Area ordering should not change the canonical fingerprint: %s %s %v", first, permutedFingerprint, err)
	}
	copyRecursion.Parallelism++
	permuted.Recursion = &copyRecursion
	changed, err := FingerprintRuntime(permuted)
	if err != nil || changed == first {
		t.Fatalf("parallelism change must change fingerprint: %s %s %v", first, changed, err)
	}
	copyRecursion = *runtime.Recursion
	copyRecursion.Areas = append([]AreaRunner(nil), runtime.Recursion.Areas...)
	copyRecursion.Areas[0].Executor.Model = "different-area-model"
	permuted.Recursion = &copyRecursion
	changed, err = FingerprintRuntime(permuted)
	if err != nil || changed == first {
		t.Fatalf("Area actor change must change fingerprint: %s %s %v", first, changed, err)
	}
	copyRecursion = *runtime.Recursion
	copyRecursion.Areas = append([]AreaRunner(nil), runtime.Recursion.Areas...)
	copyRecursion.Areas[0].Checks = append([]authoring.Check(nil), runtime.Recursion.Areas[0].Checks...)
	copyRecursion.Areas[0].Checks[0].Run = []string{"go", "test"}
	permuted.Recursion = &copyRecursion
	changed, err = FingerprintRuntime(permuted)
	if err != nil || changed == first {
		t.Fatalf("Area check arguments change must change fingerprint: %s %s %v", first, changed, err)
	}

	root := core.DefinitionIdentity{APIVersion: "markitect.government/v1alpha1", Kind: "Area", Name: "root"}
	node := runtime.Recursion.Areas[0].Area
	executor, verifier, checks, ok := areaRuntime(runtime, node, root)
	if !ok || executor.SlotID != "area-a-writer" || verifier.SlotID != "area-a-verifier" || len(checks) != 1 {
		t.Fatalf("areaRuntime did not return exact area configuration: %q %q %#v %v", executor.SlotID, verifier.SlotID, checks, ok)
	}
	if got := configuredRunners(runtime); len(got) != 7 || got[0].SlotID != runtime.Executor.SlotID || got[1].SlotID != runtime.Verifier.SlotID {
		t.Fatalf("configuredRunners should include all global slots deterministically: %#v", got)
	}
}

func validRecursiveRuntime() *RecursiveRuntime {
	checks := []authoring.Check{{Name: "tests", Run: []string{"go"}}}
	return &RecursiveRuntime{
		Limits: government.DelegationLimits{MaxDepth: 4, MaxFanout: 3, MaxCalls: 64}, Parallelism: 2, MaxRepairs: 1,
		Areas: []AreaRunner{
			{Area: areaIdentity("area-a"), Executor: runnerSpec("area-a-writer"), Verifier: runnerSpec("area-a-verifier"), Checks: append([]authoring.Check(nil), checks...)},
			{Area: areaIdentity("area-b"), Executor: runnerSpec("area-b-writer"), Verifier: runnerSpec("area-b-verifier"), Checks: append([]authoring.Check(nil), checks...)},
		},
	}
}

func areaIdentity(name string) core.DefinitionIdentity {
	return core.DefinitionIdentity{APIVersion: "markitect.government/v1alpha1", Kind: "Area", Name: name}
}
