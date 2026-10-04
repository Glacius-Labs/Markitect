package examples

import (
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/core"
)

func newSoftwareArchitectureV11(t *testing.T) (string, *host.Project) {
	t.Helper()
	root := t.TempDir()
	copySoftwareArchitecture(t, root)
	installSoftwareArchitectureV1_1(t, root)
	project := loadSoftwareArchitecture(t, root, "")
	if len(project.Diagnostics) != 0 {
		t.Fatalf("v1.1.0 fixture should be clean: %#v", project.Diagnostics)
	}
	return root, project
}

func TestSoftwareArchitectureSameTargetFeatureOwnership(t *testing.T) {
	t.Run("selected equal paths pass", func(t *testing.T) {
		_, project := newSoftwareArchitectureV11(t)
		for _, key := range []string{createOrderKey, issueInvoiceKey} {
			result, ok := softwareResult(project.Graph.PolicyResults, featureOwnershipRule, key)
			if !ok || result.Status != core.PolicyPassed || result.Comparison == nil {
				t.Fatalf("selected feature ownership should compare two resolved paths for %s: %#v", key, result)
			}
			if result.Comparison.Left.Target != result.Comparison.Right.Target {
				t.Errorf("equal Module paths resolved differently for %s: %#v", key, result.Comparison)
			}
		}
	})

	t.Run("UseCase points to Feature owned by another Module", func(t *testing.T) {
		root, project := newSoftwareArchitectureV11(t)
		mutateSoftwareResource(t, root, project, createOrderKey, func(data map[string]any) {
			data["feature"] = map[string]any{"kind": "Feature", "name": "invoicing", "namespace": "engineering"}
		})
		changed := loadSoftwareArchitecture(t, root, "")
		assertSoftwareOwnershipFailure(t, changed, createOrderKey)
	})

	t.Run("intermediate Feature points to another Module", func(t *testing.T) {
		root, project := newSoftwareArchitectureV11(t)
		mutateSoftwareResource(t, root, project, "engineering/architecture.markitect.org/v1alpha1/Feature/order-lifecycle", func(data map[string]any) {
			data["module"] = map[string]any{"kind": "Module", "name": "billing", "namespace": "engineering"}
		})
		changed := loadSoftwareArchitecture(t, root, "")
		assertSoftwareOwnershipFailure(t, changed, createOrderKey)
	})

	t.Run("selected missing Feature is structural and non-waivable", func(t *testing.T) {
		root, project := newSoftwareArchitectureV11(t)
		baseline, ok := softwareResult(project.Graph.PolicyResults, featureOwnershipRule, createOrderKey)
		if !ok || baseline.Status != core.PolicyPassed {
			t.Fatalf("expected a valid baseline comparison before the structural mutation: %#v", baseline)
		}
		mutateSoftwareResource(t, root, project, createOrderKey, func(data map[string]any) { delete(data, "feature") })
		updateSoftwareProject(t, root, project, "2026-10-02", []core.PolicyException{{
			Name: "cannot-waive-missing-feature-path", APIVersion: softwareArchitectureAPI,
			Constraint: featureOwnershipRule, Subject: createOrderKey,
			ConstraintDigest: baseline.ConstraintDigest, SubjectDigest: "sha256:" + strings.Repeat("0", 64),
			Rationale: "Probe structural path handling.", Owner: "architecture-owner",
			Decision: "A malformed path must remain structural.", ExpiresOn: "2026-11-01",
		}})
		changed := loadSoftwareArchitecture(t, root, "")
		if !hasSoftwareDiagnostic(changed, "constraint.path") || !hasSoftwareDiagnostic(changed, "policy.exception.not-waivable") {
			t.Fatalf("missing selected Feature must be a non-waivable structural path error: %#v", changed.Diagnostics)
		}
		if _, ok := softwareResult(changed.Graph.PolicyResults, featureOwnershipRule, createOrderKey); ok {
			t.Fatal("invalid same-target path incorrectly produced a PolicyResult")
		}
	})

	t.Run("unselected featureless or mismatched UseCase is outside coverage", func(t *testing.T) {
		root, project := newSoftwareArchitectureV11(t)
		mutateSoftwareResource(t, root, project, getOrderKey, func(data map[string]any) { delete(data, "feature") })
		changed := loadSoftwareArchitecture(t, root, "")
		if len(changed.Diagnostics) != 0 {
			t.Fatalf("feature remains optional outside the explicit cohort: %#v", changed.Diagnostics)
		}
		if _, ok := softwareResult(changed.Graph.PolicyResults, featureOwnershipRule, getOrderKey); ok {
			t.Fatal("unselected featureless UseCase unexpectedly received an equality result")
		}

		mutateSoftwareResource(t, root, changed, getOrderKey, func(data map[string]any) {
			data["feature"] = map[string]any{"kind": "Feature", "name": "invoicing", "namespace": "engineering"}
		})
		mismatched := loadSoftwareArchitecture(t, root, "")
		if len(mismatched.Diagnostics) != 0 {
			t.Fatalf("unlabeled owner mismatch must remain explicitly outside policy coverage: %#v", mismatched.Diagnostics)
		}
		if _, ok := softwareResult(mismatched.Graph.PolicyResults, featureOwnershipRule, getOrderKey); ok {
			t.Fatal("unselected mismatched UseCase unexpectedly received an equality result")
		}
	})

	t.Run("waiver is exact, becomes stale, and repair clears it", func(t *testing.T) {
		root, project := newSoftwareArchitectureV11(t)
		mutateSoftwareResource(t, root, project, createOrderKey, func(data map[string]any) {
			data["feature"] = map[string]any{"kind": "Feature", "name": "invoicing", "namespace": "engineering"}
		})
		failed := loadSoftwareArchitecture(t, root, "")
		result, ok := softwareResult(failed.Graph.PolicyResults, featureOwnershipRule, createOrderKey)
		if !ok || result.Status != core.PolicyFailed {
			t.Fatalf("owner mismatch should be a policy failure eligible for an exact exception: %#v", result)
		}
		exception := core.PolicyException{
			Name: "create-order-feature-transition", APIVersion: softwareArchitectureAPI,
			Constraint: featureOwnershipRule, Subject: result.Subject,
			ConstraintDigest: result.ConstraintDigest, SubjectDigest: result.SubjectDigest,
			Rationale: "The Feature ownership correction is scheduled in the next slice.",
			Owner:     "architecture-owner", Decision: "Approve this exact mismatch through the review date.", ExpiresOn: "2026-11-01",
		}
		updateSoftwareProject(t, root, failed, "2026-10-02", []core.PolicyException{exception})
		waived := loadSoftwareArchitecture(t, root, "")
		waivedResult, ok := softwareResult(waived.Graph.PolicyResults, featureOwnershipRule, createOrderKey)
		if !ok || waivedResult.Status != core.PolicyWaived || waivedResult.ExceptionName != exception.Name {
			t.Fatalf("exact policy exception should remain visible as waived: %#v", waivedResult)
		}

		mutateSoftwareResource(t, root, waived, createOrderKey, func(data map[string]any) {
			data["summary"] = "Changed after the equality exception was bound to this subject."
		})
		stale := loadSoftwareArchitecture(t, root, "")
		if !hasSoftwareDiagnostic(stale, "policy.exception.stale") {
			t.Fatalf("changed selected subject should stale its exact waiver: %#v", stale.Diagnostics)
		}
		staleResult, ok := softwareResult(stale.Graph.PolicyResults, featureOwnershipRule, createOrderKey)
		if !ok || staleResult.Status != core.PolicyFailed {
			t.Fatalf("stale exception must leave the ownership finding failed: %#v", staleResult)
		}

		mutateSoftwareResource(t, root, stale, createOrderKey, func(data map[string]any) {
			data["feature"] = map[string]any{"kind": "Feature", "name": "order-lifecycle", "namespace": "engineering"}
		})
		updateSoftwareProject(t, root, stale, "", nil)
		repaired := loadSoftwareArchitecture(t, root, "")
		if len(repaired.Diagnostics) != 0 {
			t.Fatalf("corrected ownership and removed exception should leave a clean project: %#v", repaired.Diagnostics)
		}
		repairedResult, ok := softwareResult(repaired.Graph.PolicyResults, featureOwnershipRule, createOrderKey)
		if !ok || repairedResult.Status != core.PolicyPassed {
			t.Fatalf("repaired selected paths should pass without a waiver: %#v", repairedResult)
		}
	})
}

func TestSoftwareArchitectureV11ToV21VersionedPolicyLifecycle(t *testing.T) {
	root := t.TempDir()
	copySoftwareArchitecture(t, root)
	installSoftwareArchitectureV1_1(t, root)
	rev1 := initSoftwareArchitectureGit(t, root)
	v11 := loadSoftwareArchitecture(t, root, rev1)
	if v11.Snapshot.Provisional || len(v11.Diagnostics) != 0 || v11.Graph.Project.Spec.Packages[0].Version != "1.1.0" {
		t.Fatalf("frozen v1.1 baseline is not clean/exact: %#v %#v", v11.Snapshot, v11.Diagnostics)
	}
	for _, key := range []string{createOrderKey, issueInvoiceKey} {
		if result, ok := softwareResult(v11.Graph.PolicyResults, featureOwnershipRule, key); !ok || result.Status != core.PolicyPassed {
			t.Fatalf("v1.1 should enforce aligned Feature ownership for %s: %#v", key, result)
		}
	}

	archive := installSoftwareArchitectureV2_1(t, root)
	rev2 := commitSoftwareArchitecture(t, root, "Select software architecture package v2.1")
	v21 := loadSoftwareArchitecture(t, root, rev2)
	if v21.Snapshot.Provisional || v21.Graph.Project.Spec.Packages[0].Version != "2.1.0" {
		t.Fatalf("v2.1 pin was not captured in an exact fixed snapshot: %#v", v21.Graph.Project.Spec.Packages)
	}
	digestBytes := sha256.Sum256(archive)
	if digest := hex.EncodeToString(digestBytes[:]); digest != v21.Graph.Project.Spec.Packages[0].SHA256 {
		t.Fatal("v2.1 archive digest does not match selected exact pin")
	}
	oldDomain, _ := v11.Graph.Registry.Domain(softwareArchitectureAPI)
	newDomain, _ := v21.Graph.Registry.Domain(softwareArchitectureAPI)
	if !reflect.DeepEqual(newDomain.Kinds, oldDomain.Kinds) || !reflect.DeepEqual(newDomain.Relations, oldDomain.Relations) || len(newDomain.Constraints) != len(oldDomain.Constraints)+1 {
		t.Fatal("v2.1 should retain v1.1 schema/relations and append only the Validator policy")
	}
	for _, key := range []string{createOrderKey, issueInvoiceKey} {
		if result, ok := softwareResult(v21.Graph.PolicyResults, featureOwnershipRule, key); !ok || result.Status != core.PolicyPassed {
			t.Errorf("v2.1 should retain same-target passes for %s: %#v", key, result)
		}
		if result, ok := softwareResult(v21.Graph.PolicyResults, commandValidatorRule, key); !ok || result.Status != core.PolicyFailed {
			t.Errorf("v2.1 should add the Validator lifecycle failure for %s: %#v", key, result)
		}
	}
	impact := host.Changes(v11, v21)
	assertSoftwareHas(t, impact.Affected, createOrderKey)
	assertSoftwareHas(t, impact.Affected, issueInvoiceKey)

	for _, key := range []string{createOrderKey, issueInvoiceKey} {
		validatorName := map[string]string{createOrderKey: "order-validation", issueInvoiceKey: "invoice-validation"}[key]
		mutateSoftwareResource(t, root, v21, key, func(data map[string]any) {
			data["validators"] = []any{map[string]any{"kind": "Validator", "name": validatorName, "namespace": "engineering"}}
		})
		writeSoftwareResource(t, root, core.Resource{APIVersion: softwareArchitectureAPI, Kind: "Validator", Metadata: core.Metadata{Name: validatorName, Namespace: "engineering"}, Data: map[string]any{"summary": "Validates the selected command input."}})
	}
	rev3 := commitSoftwareArchitecture(t, root, "Complete v2.1 validator policy")
	passed := loadSoftwareArchitecture(t, root, rev3)
	if len(passed.Diagnostics) != 0 {
		t.Fatalf("implemented Validators should complete the v2.1 lifecycle cleanly: %#v", passed.Diagnostics)
	}
	for _, key := range []string{createOrderKey, issueInvoiceKey} {
		if result, ok := softwareResult(passed.Graph.PolicyResults, featureOwnershipRule, key); !ok || result.Status != core.PolicyPassed {
			t.Errorf("completed v2.1 lifecycle lost the equality pass for %s: %#v", key, result)
		}
		if result, ok := softwareResult(passed.Graph.PolicyResults, commandValidatorRule, key); !ok || result.Status != core.PolicyPassed {
			t.Errorf("completed v2.1 lifecycle did not pass Validator rule for %s: %#v", key, result)
		}
	}
}

func assertSoftwareOwnershipFailure(t *testing.T, project *host.Project, subject string) {
	t.Helper()
	if !hasSoftwareDiagnostic(project, "constraint."+featureOwnershipRule) {
		t.Fatalf("selected owner mismatch did not produce its policy diagnostic: %#v", project.Diagnostics)
	}
	result, ok := softwareResult(project.Graph.PolicyResults, featureOwnershipRule, subject)
	if !ok || result.Status != core.PolicyFailed || result.Comparison == nil || result.Comparison.Left.Target == result.Comparison.Right.Target {
		t.Fatalf("mismatched path targets did not produce a failed comparison: %#v", result)
	}
}
