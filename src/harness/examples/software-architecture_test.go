package examples

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring/contentpackage"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

const (
	softwareArchitectureAPI = "architecture.markitect.org/v1alpha1"
	createOrderKey          = "engineering/architecture.markitect.org/v1alpha1/UseCase/create-order"
	getOrderKey             = "engineering/architecture.markitect.org/v1alpha1/UseCase/get-order"
	issueInvoiceKey         = "engineering/architecture.markitect.org/v1alpha1/UseCase/issue-invoice"
	checkAvailabilityKey    = "engineering/architecture.markitect.org/v1alpha1/UseCase/check-availability"
	ordersModuleKey         = "engineering/architecture.markitect.org/v1alpha1/Module/orders"
	billingModuleKey        = "engineering/architecture.markitect.org/v1alpha1/Module/billing"
	inventoryModuleKey      = "engineering/architecture.markitect.org/v1alpha1/Module/inventory"
	commandValidatorRule    = "selected-commands-have-validators"
	featureOwnershipRule    = "selected-feature-ownership-matches-module"
)

func softwareArchitectureRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(harnessRepositoryRoot(t), "examples", "software-architecture")
}

func copySoftwareArchitecture(t *testing.T, target string) {
	t.Helper()
	source := softwareArchitectureRoot(t)
	if err := filepath.WalkDir(source, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0755)
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0644)
	}); err != nil {
		t.Fatal(err)
	}
}

func loadSoftwareArchitecture(t *testing.T, root, revision string) *host.Project {
	t.Helper()
	project, err := host.Load(root, revision)
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func gitSoftwareArchitecture(t *testing.T, root string, args ...string) string {
	t.Helper()
	command := exec.Command("git", append([]string{"-C", root}, args...)...)
	output, err := command.CombinedOutput()
	if err != nil {
		t.Fatalf("git %s: %v\n%s", strings.Join(args, " "), err, output)
	}
	return strings.TrimSpace(string(output))
}

func initSoftwareArchitectureGit(t *testing.T, root string) string {
	t.Helper()
	gitSoftwareArchitecture(t, root, "init", "-b", "codex/software-architecture-test")
	gitSoftwareArchitecture(t, root, "config", "user.name", "Software Architecture Fixture")
	gitSoftwareArchitecture(t, root, "config", "user.email", "fixture@example.invalid")
	return commitSoftwareArchitecture(t, root, "Freeze software architecture fixture")
}

func commitSoftwareArchitecture(t *testing.T, root, message string) string {
	t.Helper()
	gitSoftwareArchitecture(t, root, "add", "--all")
	gitSoftwareArchitecture(t, root, "commit", "--allow-empty", "-m", message)
	return gitSoftwareArchitecture(t, root, "rev-parse", "HEAD")
}

func readSoftwareArchitectureTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := map[string][]byte{}
	if err := filepath.WalkDir(root, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		relative, err := filepath.Rel(root, name)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(name)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = data
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	return files
}

func installSoftwareArchitectureV2(t *testing.T, root string) []byte {
	t.Helper()
	files := readSoftwareArchitectureTree(t, filepath.Join(softwareArchitectureRoot(t), "architecture-package-v2"))
	return installSoftwareArchitecturePackage(t, root, files, "2.0.0", "fixture:software-architecture-package-v2")
}

func installSoftwareArchitectureV1(t *testing.T, root string) []byte {
	t.Helper()
	files := readSoftwareArchitectureTree(t, filepath.Join(softwareArchitectureRoot(t), "architecture-package-v1"))
	return installSoftwareArchitecturePackage(t, root, files, "1.0.0", "fixture:software-architecture-package-v1")
}

func installSoftwareArchitectureV1_1(t *testing.T, root string) []byte {
	t.Helper()
	files := readSoftwareArchitectureTree(t, filepath.Join(softwareArchitectureRoot(t), "architecture-package-v1.1.0"))
	return installSoftwareArchitecturePackage(t, root, files, "1.1.0", "fixture:software-architecture-package-1.1.0")
}

func installSoftwareArchitectureV2_1(t *testing.T, root string) []byte {
	t.Helper()
	files := readSoftwareArchitectureTree(t, filepath.Join(softwareArchitectureRoot(t), "architecture-package-v2.1.0"))
	return installSoftwareArchitecturePackage(t, root, files, "2.1.0", "fixture:software-architecture-package-2.1.0")
}

func installSoftwareArchitecturePackage(t *testing.T, root string, files map[string][]byte, version, source string) []byte {
	t.Helper()
	archive, err := contentpackage.Build(files)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(archive, mustBuildSoftwarePackage(t, files)) {
		t.Fatal("rebuilding the package from identical inputs was not deterministic")
	}
	archivePath := filepath.Join(root, ".markitect", "packages", "software-architecture-"+version+".zip")
	if err := os.WriteFile(archivePath, archive, 0644); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(archive)
	projectPath := filepath.Join(root, "markitect.yaml")
	project, err := authoring.Parse(projectPath, mustReadSoftwareFile(t, projectPath))
	if err != nil {
		t.Fatal(err)
	}
	if len(project.Spec.Packages) != 1 {
		t.Fatalf("expected one exact architecture package pin, got %#v", project.Spec.Packages)
	}
	pin := project.Spec.Packages[0]
	pin.Version = version
	pin.Source = source
	pin.Archive = ".markitect/packages/software-architecture-" + version + ".zip"
	pin.SHA256 = hex.EncodeToString(digest[:])
	project.Spec.Packages[0] = pin
	encoded, err := authoring.Encode(*project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(projectPath, encoded, 0644); err != nil {
		t.Fatal(err)
	}
	return archive
}

func mustBuildSoftwarePackage(t *testing.T, files map[string][]byte) []byte {
	t.Helper()
	archive, err := contentpackage.Build(files)
	if err != nil {
		t.Fatal(err)
	}
	return archive
}

func mustReadSoftwareFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func mutateSoftwareResource(t *testing.T, root string, project *host.Project, key string, mutate func(map[string]any)) {
	t.Helper()
	resource := project.Graph.Resources[key]
	if resource == nil {
		t.Fatalf("fixture resource %s is missing", key)
	}
	copy := *resource
	data := make(map[string]any, len(resource.Data))
	for field, value := range resource.Data {
		data[field] = value
	}
	copy.Data = data
	mutate(data)
	encoded, err := authoring.Encode(copy)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, filepath.FromSlash(resource.Path))
	if err := os.WriteFile(path, encoded, 0644); err != nil {
		t.Fatal(err)
	}
}

func updateSoftwareProject(t *testing.T, root string, project *host.Project, policyDate string, exceptions []core.PolicyException) {
	t.Helper()
	copy := *project.Graph.Project
	copy.Spec.PolicyDate = policyDate
	copy.Spec.PolicyExceptions = exceptions
	encoded, err := authoring.Encode(copy)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "markitect.yaml"), encoded, 0644); err != nil {
		t.Fatal(err)
	}
}

func softwareResult(results []core.PolicyResult, constraint, subject string) (core.PolicyResult, bool) {
	for _, result := range results {
		if result.APIVersion == softwareArchitectureAPI && result.Constraint == constraint && result.Subject == subject {
			return result, true
		}
	}
	return core.PolicyResult{}, false
}

func hasSoftwareDiagnostic(project *host.Project, code string) bool {
	for _, diagnostic := range project.Diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func assertSoftwareHas(t *testing.T, values []string, want string) {
	t.Helper()
	for _, value := range values {
		if value == want {
			return
		}
	}
	t.Errorf("%q not found in %#v", want, values)
}

func TestSoftwareArchitectureV1PackageDrivesConsumerContextAndPolicy(t *testing.T) {
	root := softwareArchitectureRoot(t)
	project := loadSoftwareArchitecture(t, root, "")
	if len(project.Diagnostics) != 0 {
		t.Fatalf("v1 consumer should compile without policy failures: %#v", project.Diagnostics)
	}
	if got := project.Graph.Project.Spec.Packages[0].Version; got != "1.1.0" {
		t.Fatalf("consumer should target the same-target baseline package 1.1.0, got %s", got)
	}

	for _, key := range []string{createOrderKey, issueInvoiceKey} {
		if result, ok := softwareResult(project.Graph.Core.PolicyResults, "selected-command-labels-identify-commands", key); !ok || result.Status != core.PolicyPassed {
			t.Errorf("selected Command intent assertion missing or failed for %s: %#v", key, result)
		}
		if result, ok := softwareResult(project.Graph.Core.PolicyResults, featureOwnershipRule, key); !ok || result.Status != core.PolicyPassed {
			t.Errorf("selected Feature ownership assertion missing or failed for %s: %#v", key, result)
		}
	}
	if _, ok := softwareResult(project.Graph.Core.PolicyResults, featureOwnershipRule, getOrderKey); ok {
		t.Fatal("unlabeled Query unexpectedly entered the selected Feature ownership cohort")
	}
	if _, ok := softwareResult(project.Graph.Core.PolicyResults, "selected-command-labels-identify-commands", getOrderKey); ok {
		t.Fatal("unlabeled Query unexpectedly entered the selected Command cohort")
	}

	compiled, err := host.CompileContext(project, "engineering/Skill/implement-order", "0.11.0")
	if err != nil {
		t.Fatal(err)
	}
	resources := map[string]bool{}
	domains := 0
	for _, input := range compiled.Inputs {
		if input.Resource != nil {
			resources[input.Resource.GraphKey()] = true
		}
		if input.Role == "domain" && input.PackageVersion == "1.1.0" {
			domains++
			if !strings.Contains(input.Text, "selected-command-labels-identify-commands") || !strings.Contains(input.Text, "validation: required") || !strings.Contains(input.Text, featureOwnershipRule) || !strings.Contains(input.Text, "same-target") {
				t.Error("context did not retain the exact selected policy definition")
			}
		}
	}
	if domains != 1 {
		t.Fatalf("context should contain exactly the explicitly selected v1 Domain, got %d", domains)
	}
	for _, key := range []string{
		"engineering/Skill/implement-order", createOrderKey, ordersModuleKey,
		"engineering/architecture.markitect.org/v1alpha1/Product/commerce",
		"engineering/architecture.markitect.org/v1alpha1/Core/platform-core",
		"engineering/architecture.markitect.org/v1alpha1/Common/shared-kernel",
		"engineering/architecture.markitect.org/v1alpha1/Handler/create-order-handler",
		"software-architecture::architecture/Workflow/vertical-slice",
	} {
		if !resources[key] {
			t.Errorf("bounded implement-order context omitted %s", key)
		}
	}
	for _, key := range []string{issueInvoiceKey, checkAvailabilityKey, billingModuleKey, inventoryModuleKey} {
		if resources[key] {
			t.Errorf("Orders context unexpectedly pulled unrelated resource %s", key)
		}
	}
	if len(compiled.PolicyResults) != 3 {
		t.Fatalf("context should include the Module boundary and selected UseCase policy results in its subject closure, got %#v", compiled.PolicyResults)
	}
	for _, expected := range []struct{ constraint, subject string }{
		{"module-dependencies-stay-at-shared-boundaries", ordersModuleKey},
		{"selected-command-labels-identify-commands", createOrderKey},
		{featureOwnershipRule, createOrderKey},
	} {
		if _, ok := softwareResult(compiled.PolicyResults, expected.constraint, expected.subject); !ok {
			t.Errorf("context omitted exact policy result %s for %s", expected.constraint, expected.subject)
		}
	}
}

func TestSoftwareArchitectureV11SameTargetRuleAppearsInGeneratedViews(t *testing.T) {
	project := loadSoftwareArchitecture(t, softwareArchitectureRoot(t), "")
	views, err := host.GenerateOutputs(project)
	if err != nil {
		t.Fatal(err)
	}
	domainView := string(views["docs/markitect/_domains/architecture.markitect.org.v1alpha1.domain.md"])
	for _, want := range []string{featureOwnershipRule, "feature-ownership: required", "same-target", "belongsToModule", "realizesFeature"} {
		if !strings.Contains(domainView, want) {
			t.Errorf("generated Domain view omitted same-target policy detail %q", want)
		}
	}
	useCaseView := string(views["docs/markitect/engineering/usecase-create-order.usecase.architecture.markitect.org.v1alpha1.md"])
	if !strings.Contains(useCaseView, featureOwnershipRule) || !strings.Contains(useCaseView, "**PASSED**") {
		t.Errorf("generated UseCase view omitted its selected same-target result: %s", useCaseView)
	}
}

func TestSoftwareArchitecturePackageSourcesAndArchiveAreReproducible(t *testing.T) {
	for _, version := range []string{"v1", "v1.1.0", "v2", "v2.1.0"} {
		t.Run(version, func(t *testing.T) {
			root := filepath.Join(softwareArchitectureRoot(t), "architecture-package-"+version)
			files := readSoftwareArchitectureTree(t, root)
			domain, err := authoring.ParseDomain("domains/software.yaml", files["domains/software.yaml"])
			if err != nil {
				t.Fatal(err)
			}
			registry := authoring.NewRegistry()
			if err := registry.AddDomain(domain); err != nil {
				t.Fatal(err)
			}
			canonicalDomain, err := authoring.EncodeDomain(domain)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(canonicalDomain, files["domains/software.yaml"]) {
				t.Errorf("%s Domain source is not canonical", version)
			}
			for _, member := range []string{"markitect-package.yaml", ".markitect/areas/architecture/vertical-slice.workflow.yaml"} {
				resource, err := authoring.ParseWithRegistry(member, files[member], registry)
				if err != nil {
					t.Fatalf("parse %s: %v", member, err)
				}
				canonical, err := authoring.Encode(*resource)
				if err != nil {
					t.Fatal(err)
				}
				if !bytes.Equal(canonical, files[member]) {
					t.Errorf("%s/%s is not canonical", version, member)
				}
			}
			first, err := contentpackage.Build(files)
			if err != nil {
				t.Fatal(err)
			}
			second, err := contentpackage.Build(files)
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(first, second) {
				t.Fatal("package builder output changed for identical source bytes")
			}
			if version == "v1" || version == "v1.1.0" {
				archiveVersion := strings.TrimPrefix(version, "v")
				if version == "v1" {
					archiveVersion = "1.0.0"
				}
				checkedIn := mustReadSoftwareFile(t, filepath.Join(softwareArchitectureRoot(t), ".markitect", "packages", "software-architecture-"+archiveVersion+".zip"))
				if !bytes.Equal(first, checkedIn) {
					t.Fatalf("checked-in %s archive does not match the deterministic package builder output", version)
				}
			}
		})
	}
}

func TestSoftwareArchitectureV2PackagePolicyLifecycleUsesFixedSnapshots(t *testing.T) {
	root := t.TempDir()
	copySoftwareArchitecture(t, root)
	installSoftwareArchitectureV1(t, root)
	rev1 := initSoftwareArchitectureGit(t, root)
	v1 := loadSoftwareArchitecture(t, root, rev1)
	if v1.Snapshot.Provisional || len(v1.Diagnostics) != 0 {
		t.Fatalf("frozen v1 consumer is not clean: provisional=%t diagnostics=%#v", v1.Snapshot.Provisional, v1.Diagnostics)
	}

	v2Archive := installSoftwareArchitectureV2(t, root)
	rev2 := commitSoftwareArchitecture(t, root, "Select software architecture package v2")
	v2 := loadSoftwareArchitecture(t, root, rev2)
	if v2.Snapshot.Provisional || v2.Graph.Project.Spec.Packages[0].Version != "2.0.0" {
		t.Fatalf("v2 pin was not captured as an exact fixed snapshot: %#v", v2.Graph.Project.Spec.Packages)
	}
	if got := sha256.Sum256(v2Archive); hex.EncodeToString(got[:]) != v2.Graph.Project.Spec.Packages[0].SHA256 {
		t.Fatalf("v2 archive digest does not match exact selected pin")
	}
	oldDomain, ok := v1.Graph.Registry.Domain(softwareArchitectureAPI)
	if !ok {
		t.Fatal("v1 Domain is not registered under the exact API version")
	}
	newDomain, ok := v2.Graph.Registry.Domain(softwareArchitectureAPI)
	if !ok {
		t.Fatal("v2 Domain is not registered under the exact API version")
	}
	if oldDomain.APIVersion != newDomain.APIVersion || !reflect.DeepEqual(oldDomain.Kinds, newDomain.Kinds) || !reflect.DeepEqual(oldDomain.Relations, newDomain.Relations) {
		t.Fatal("v2 policy-only package update changed the active schema/relations or API version")
	}
	if len(newDomain.Constraints) != len(oldDomain.Constraints)+1 {
		t.Fatalf("v2 should add exactly one policy assertion, v1=%d v2=%d", len(oldDomain.Constraints), len(newDomain.Constraints))
	}
	if !reflect.DeepEqual(newDomain.Constraints[:len(oldDomain.Constraints)], oldDomain.Constraints) {
		t.Fatal("v2 changed an existing v1 assertion instead of appending one policy rule")
	}
	if got := v1.Graph.Project.Spec.Packages[0].SHA256; got == v2.Graph.Project.Spec.Packages[0].SHA256 {
		t.Fatal("v1 and v2 exact package pins unexpectedly share an archive digest")
	}

	for _, key := range []string{createOrderKey, issueInvoiceKey} {
		result, ok := softwareResult(v2.Graph.Core.PolicyResults, commandValidatorRule, key)
		if !ok || result.Status != core.PolicyFailed {
			t.Errorf("v2 should fail selected Command %s without a Validator: %#v", key, result)
		}
	}
	for _, key := range []string{getOrderKey, checkAvailabilityKey} {
		if _, ok := softwareResult(v2.Graph.Core.PolicyResults, commandValidatorRule, key); ok {
			t.Errorf("unlabeled Query %s unexpectedly received the command-only validator policy", key)
		}
	}
	impact := host.Changes(v1, v2)
	assertSoftwareHas(t, impact.Affected, createOrderKey)
	assertSoftwareHas(t, impact.Affected, issueInvoiceKey)
	assertSoftwareHas(t, impact.Affected, inventoryModuleKey)

	createFailure, ok := softwareResult(v2.Graph.Core.PolicyResults, commandValidatorRule, createOrderKey)
	if !ok {
		t.Fatal("create-order failure has no source-bound PolicyResult")
	}
	exception := core.PolicyException{
		Name: "create-order-validator-transition", APIVersion: softwareArchitectureAPI,
		Constraint: commandValidatorRule, Subject: createFailure.Subject,
		ConstraintDigest: createFailure.ConstraintDigest, SubjectDigest: createFailure.SubjectDigest,
		Rationale: "The validator implementation is scheduled in the next slice.",
		Owner:     "architecture-owner", Decision: "Approve one exact-source transition through the frozen review date.",
		ExpiresOn: "2026-11-01",
	}
	updateSoftwareProject(t, root, v2, "2026-10-02", []core.PolicyException{exception})
	rev3 := commitSoftwareArchitecture(t, root, "Record one bounded validator exception")
	waived := loadSoftwareArchitecture(t, root, rev3)
	if !hasSoftwareDiagnostic(waived, "constraint."+commandValidatorRule) {
		t.Fatal("waiving one subject incorrectly hid the other selected Command failure")
	}
	waivedResult, ok := softwareResult(waived.Graph.Core.PolicyResults, commandValidatorRule, createOrderKey)
	if !ok || waivedResult.Status != core.PolicyWaived || waivedResult.ExceptionName != exception.Name {
		t.Fatalf("exact exception did not yield an explicit waived PolicyResult: %#v", waivedResult)
	}
	stillFailed, ok := softwareResult(waived.Graph.Core.PolicyResults, commandValidatorRule, issueInvoiceKey)
	if !ok || stillFailed.Status != core.PolicyFailed {
		t.Fatalf("second selected Command must remain failed: %#v", stillFailed)
	}

	mutateSoftwareResource(t, root, waived, issueInvoiceKey, func(data map[string]any) {
		data["validators"] = []any{map[string]any{"kind": "Validator", "name": "invoice-validation", "namespace": "engineering"}}
	})
	writeSoftwareResource(t, root, authoring.Resource{Core: authoring.Core{APIVersion: softwareArchitectureAPI, Kind: "Validator", Metadata: core.Metadata{Name: "invoice-validation", Namespace: "engineering"}, Data: map[string]any{"summary": "Validates invoice command input."}}})
	rev4 := commitSoftwareArchitecture(t, root, "Implement the second selected Command Validator")
	cleared := loadSoftwareArchitecture(t, root, rev4)
	if len(cleared.Diagnostics) != 0 {
		t.Fatalf("the remaining failure should be cleared while the exact exception remains: %#v", cleared.Diagnostics)
	}
	context, err := host.CompileContext(cleared, "engineering/Skill/implement-order", "0.11.0")
	if err != nil {
		t.Fatal(err)
	}
	contextWaiver, ok := softwareResult(context.PolicyResults, commandValidatorRule, createOrderKey)
	if !ok || contextWaiver.Status != core.PolicyWaived || contextWaiver.ExceptionName != exception.Name || contextWaiver.Rationale != exception.Rationale || contextWaiver.Owner != exception.Owner || contextWaiver.Decision != exception.Decision || contextWaiver.PolicyDate != "2026-10-02" || contextWaiver.ExpiresOn != exception.ExpiresOn {
		t.Fatalf("agent context omitted the bound waiver decision: %#v", contextWaiver)
	}
	views, err := host.GenerateOutputs(cleared)
	if err != nil {
		t.Fatal(err)
	}
	contract := string(views["docs/markitect/_domains/architecture.markitect.org.v1alpha1.domain.md"])
	for _, value := range []string{"selected-commands-have-validators", "**WAIVED**", exception.Name, exception.Rationale, exception.Owner, exception.Decision, exception.ExpiresOn, "2026-10-02"} {
		if !strings.Contains(contract, value) {
			t.Errorf("rendered Domain view omitted policy/waiver evidence %q", value)
		}
	}
	useCaseView := string(views["docs/markitect/engineering/usecase-create-order.usecase.architecture.markitect.org.v1alpha1.md"])
	for _, value := range []string{"**WAIVED**", exception.Name, exception.Rationale, exception.Owner, exception.Decision, exception.ExpiresOn, "2026-10-02"} {
		if !strings.Contains(useCaseView, value) {
			t.Errorf("rendered UseCase view omitted the matching waiver evidence %q", value)
		}
	}

	mutateSoftwareResource(t, root, cleared, createOrderKey, func(data map[string]any) {
		data["validators"] = []any{map[string]any{"kind": "Validator", "name": "order-validation", "namespace": "engineering"}}
	})
	writeSoftwareResource(t, root, authoring.Resource{Core: authoring.Core{APIVersion: softwareArchitectureAPI, Kind: "Validator", Metadata: core.Metadata{Name: "order-validation", Namespace: "engineering"}, Data: map[string]any{"summary": "Validates order command input."}}})
	updateSoftwareProject(t, root, cleared, "", nil)
	rev5 := commitSoftwareArchitecture(t, root, "Implement final Validator and remove exception")
	passed := loadSoftwareArchitecture(t, root, rev5)
	if len(passed.Diagnostics) != 0 {
		t.Fatalf("adding Validators and removing the exception should pass v2: %#v", passed.Diagnostics)
	}
	for _, key := range []string{createOrderKey, issueInvoiceKey} {
		result, ok := softwareResult(passed.Graph.Core.PolicyResults, commandValidatorRule, key)
		if !ok || result.Status != core.PolicyPassed {
			t.Errorf("selected Command %s should pass after Validator implementation: %#v", key, result)
		}
	}
}

func TestSoftwareArchitectureStructuralAndPolicyFailuresRemainDistinct(t *testing.T) {
	t.Run("module dependency target is policy failure", func(t *testing.T) {
		root := t.TempDir()
		copySoftwareArchitecture(t, root)
		before := loadSoftwareArchitecture(t, root, "")
		mutateSoftwareResource(t, root, before, ordersModuleKey, func(data map[string]any) {
			data["dependsOn"] = []any{map[string]any{"kind": "Module", "name": "billing", "namespace": "engineering"}}
		})
		changed := loadSoftwareArchitecture(t, root, "")
		if !hasSoftwareDiagnostic(changed, "constraint.module-dependencies-stay-at-shared-boundaries") {
			t.Fatalf("direct Module dependency escaped the central shared-boundary policy: %#v", changed.Diagnostics)
		}
		result, ok := softwareResult(changed.Graph.Core.PolicyResults, "module-dependencies-stay-at-shared-boundaries", ordersModuleKey)
		if !ok || result.Status != core.PolicyFailed {
			t.Fatalf("forbidden Module target did not yield per-subject policy result: %#v", result)
		}
	})

	for _, test := range []struct {
		name   string
		key    string
		mutate func(map[string]any)
	}{
		{name: "missing module", key: createOrderKey, mutate: func(data map[string]any) { delete(data, "module") }},
		{name: "missing Handler", key: createOrderKey, mutate: func(data map[string]any) { delete(data, "handler") }},
		{name: "invalid Command Query enum", key: createOrderKey, mutate: func(data map[string]any) { data["intent"] = "Transaction" }},
		{name: "wrong Handler kind", key: createOrderKey, mutate: func(data map[string]any) {
			data["handler"] = map[string]any{"kind": "Module", "name": "orders", "namespace": "engineering"}
		}},
		{name: "multiple Handlers in scalar field", key: createOrderKey, mutate: func(data map[string]any) {
			data["handler"] = []any{map[string]any{"kind": "Handler", "name": "create-order-handler", "namespace": "engineering"}, map[string]any{"kind": "Handler", "name": "get-order-handler", "namespace": "engineering"}}
		}},
		{name: "unmodeled lateral UseCase relation", key: createOrderKey, mutate: func(data map[string]any) {
			data["dependsOn"] = []any{map[string]any{"kind": "UseCase", "name": "issue-invoice", "namespace": "engineering"}}
		}},
		{name: "Core cannot declare outgoing references", key: "engineering/architecture.markitect.org/v1alpha1/Core/platform-core", mutate: func(data map[string]any) {
			data["dependsOn"] = []any{map[string]any{"kind": "Module", "name": "orders", "namespace": "engineering"}}
		}},
		{name: "extension field cannot point to unapproved type", key: createOrderKey, mutate: func(data map[string]any) {
			data["validators"] = []any{map[string]any{"kind": "Authorizer", "name": "order-write", "namespace": "engineering"}}
		}},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			copySoftwareArchitecture(t, root)
			before := loadSoftwareArchitecture(t, root, "")
			mutateSoftwareResource(t, root, before, test.key, test.mutate)
			changed := loadSoftwareArchitecture(t, root, "")
			if !hasSoftwareDiagnostic(changed, "parse") {
				t.Fatalf("invalid structural model was not rejected by the registered schema: %#v", changed.Diagnostics)
			}
			if test.key == createOrderKey {
				if _, ok := softwareResult(changed.Graph.Core.PolicyResults, "handledBy", createOrderKey); ok {
					t.Fatal("structural Handler failure was misrepresented as a waivable PolicyResult")
				}
			}
		})
	}
	t.Run("Feature relation is optional", func(t *testing.T) {
		root := t.TempDir()
		copySoftwareArchitecture(t, root)
		before := loadSoftwareArchitecture(t, root, "")
		mutateSoftwareResource(t, root, before, getOrderKey, func(data map[string]any) { delete(data, "feature") })
		changed := loadSoftwareArchitecture(t, root, "")
		if len(changed.Diagnostics) != 0 {
			t.Fatalf("optional Feature was treated as required: %#v", changed.Diagnostics)
		}
	})

	t.Run("selected label must agree with Command intent", func(t *testing.T) {
		root := t.TempDir()
		copySoftwareArchitecture(t, root)
		before := loadSoftwareArchitecture(t, root, "")
		setSoftwareResourceLabel(t, root, before, getOrderKey, "validation", "required")
		changed := loadSoftwareArchitecture(t, root, "")
		result, ok := softwareResult(changed.Graph.Core.PolicyResults, "selected-command-labels-identify-commands", getOrderKey)
		if !ok || result.Status != core.PolicyFailed || !hasSoftwareDiagnostic(changed, "constraint.selected-command-labels-identify-commands") {
			t.Fatalf("a Query opted into the Command cohort must fail its selected intent assertion: %#v", result)
		}
	})
}

func TestSoftwareArchitectureConflictingDomainPoliciesRemainVisible(t *testing.T) {
	root := t.TempDir()
	copySoftwareArchitecture(t, root)
	files := readSoftwareArchitectureTree(t, filepath.Join(softwareArchitectureRoot(t), "architecture-package-v2"))
	domain, err := authoring.ParseDomain("domains/software.yaml", files["domains/software.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	domain.Constraints = append(domain.Constraints, core.ConstraintDefinition{
		Name:   "conflicting-module-dependency-allows-only-modules",
		Select: core.ResourceSelector{Kind: "Module"},
		Assert: core.ConstraintAssertion{Op: "allowed-targets", Relation: "dependsOn", Values: []any{"Module"}},
	})
	files["domains/software.yaml"], err = authoring.EncodeDomain(domain)
	if err != nil {
		t.Fatal(err)
	}
	installSoftwareArchitecturePackage(t, root, files, "2.0.0", "fixture:software-architecture-conflicting-package")
	project := loadSoftwareArchitecture(t, root, "")
	if !hasSoftwareDiagnostic(project, "constraint.conflicting-module-dependency-allows-only-modules") {
		t.Fatalf("contradictory policy was hidden or overwritten by declaration order: %#v", project.Diagnostics)
	}
	conflict, ok := softwareResult(project.Graph.Core.PolicyResults, "conflicting-module-dependency-allows-only-modules", ordersModuleKey)
	if !ok || conflict.Status != core.PolicyFailed {
		t.Fatalf("conflicting assertion did not remain a separate failed PolicyResult: %#v", conflict)
	}
}

func TestSoftwareArchitectureCommandPolicyIsExplicitLabelScope(t *testing.T) {
	root := t.TempDir()
	copySoftwareArchitecture(t, root)
	installSoftwareArchitectureV2(t, root)
	before := loadSoftwareArchitecture(t, root, "")
	setSoftwareResourceLabel(t, root, before, createOrderKey, "validation", "")
	changed := loadSoftwareArchitecture(t, root, "")
	if _, ok := softwareResult(changed.Graph.Core.PolicyResults, commandValidatorRule, createOrderKey); ok {
		t.Fatal("unlabeled Command unexpectedly received the v2 command-only Validator finding")
	}
	if result, ok := softwareResult(changed.Graph.Core.PolicyResults, commandValidatorRule, issueInvoiceKey); !ok || result.Status != core.PolicyFailed {
		t.Fatalf("other labeled Command should remain in scope and failed: %#v", result)
	}
	if len(changed.Diagnostics) == 0 {
		t.Fatal("removing one label unexpectedly passed while another selected Command still lacks a Validator")
	}
}

func TestSoftwareArchitecturePolicyExceptionExpiryAndStalenessAreActionable(t *testing.T) {
	t.Run("expired exception remains failed", func(t *testing.T) {
		root := t.TempDir()
		copySoftwareArchitecture(t, root)
		installSoftwareArchitectureV2(t, root)
		v2 := loadSoftwareArchitecture(t, root, "")
		failure, ok := softwareResult(v2.Graph.Core.PolicyResults, commandValidatorRule, createOrderKey)
		if !ok || failure.Status != core.PolicyFailed {
			t.Fatalf("expected source-bound validator finding: %#v", failure)
		}
		exception := core.PolicyException{
			Name: "expired-create-order-validator-transition", APIVersion: softwareArchitectureAPI,
			Constraint: commandValidatorRule, Subject: failure.Subject,
			ConstraintDigest: failure.ConstraintDigest, SubjectDigest: failure.SubjectDigest,
			Rationale: "The validator implementation is scheduled in the next slice.",
			Owner:     "architecture-owner", Decision: "Approve one exact-source transition through the prior review date.",
			ExpiresOn: "2026-10-01",
		}
		updateSoftwareProject(t, root, v2, "2026-10-02", []core.PolicyException{exception})
		expired := loadSoftwareArchitecture(t, root, "")
		if !hasSoftwareDiagnostic(expired, "policy.exception.expired") {
			t.Fatalf("exception past its pinned expiry date should be actionable: %#v", expired.Diagnostics)
		}
		result, ok := softwareResult(expired.Graph.Core.PolicyResults, commandValidatorRule, createOrderKey)
		if !ok || result.Status != core.PolicyFailed {
			t.Fatalf("expired exception must not waive its source-bound failure: %#v", result)
		}
	})

	t.Run("changed subject makes exception stale", func(t *testing.T) {
		root := t.TempDir()
		copySoftwareArchitecture(t, root)
		installSoftwareArchitectureV2(t, root)
		v2 := loadSoftwareArchitecture(t, root, "")
		failure, ok := softwareResult(v2.Graph.Core.PolicyResults, commandValidatorRule, createOrderKey)
		if !ok || failure.Status != core.PolicyFailed {
			t.Fatalf("expected source-bound validator finding: %#v", failure)
		}
		exception := core.PolicyException{
			Name: "stale-create-order-validator-transition", APIVersion: softwareArchitectureAPI,
			Constraint: commandValidatorRule, Subject: failure.Subject,
			ConstraintDigest: failure.ConstraintDigest, SubjectDigest: failure.SubjectDigest,
			Rationale: "The validator implementation is scheduled in the next slice.",
			Owner:     "architecture-owner", Decision: "Approve one exact-source transition through the review date.",
			ExpiresOn: "2026-11-01",
		}
		updateSoftwareProject(t, root, v2, "2026-10-02", []core.PolicyException{exception})
		mutateSoftwareResource(t, root, v2, createOrderKey, func(data map[string]any) {
			data["summary"] = "Changed after the exception was bound to this subject."
		})
		stale := loadSoftwareArchitecture(t, root, "")
		if !hasSoftwareDiagnostic(stale, "policy.exception.stale") {
			t.Fatalf("subject digest change should make the exception actionable as stale: %#v", stale.Diagnostics)
		}
		result, ok := softwareResult(stale.Graph.Core.PolicyResults, commandValidatorRule, createOrderKey)
		if !ok || result.Status != core.PolicyFailed {
			t.Fatalf("stale exception must not waive its changed source finding: %#v", result)
		}
	})
}

func TestSoftwareArchitectureBoundedHandlerImpactAndReadOnlyReconcile(t *testing.T) {
	root := t.TempDir()
	copySoftwareArchitecture(t, root)
	baseRevision := initSoftwareArchitectureGit(t, root)
	base := loadSoftwareArchitecture(t, root, baseRevision)
	mutateSoftwareResource(t, root, base, "engineering/architecture.markitect.org/v1alpha1/Handler/create-order-handler", func(data map[string]any) {
		data["summary"] = "Coordinates the revised order acceptance flow."
	})
	changed := loadSoftwareArchitecture(t, root, "")
	impact := host.Changes(base, changed)
	assertSoftwareHas(t, impact.Affected, createOrderKey)
	assertSoftwareHas(t, impact.Affected, "engineering/Skill/implement-order")
	for _, unrelated := range []string{checkAvailabilityKey, inventoryModuleKey} {
		if containsSoftwareString(impact.Affected, unrelated) {
			t.Errorf("bounded Handler edit unexpectedly affected unrelated resource %s", unrelated)
		}
	}

	planRoot := t.TempDir()
	copySoftwareArchitecture(t, planRoot)
	generatedIndex := filepath.Join(planRoot, "docs", "markitect", "README.md")
	if err := os.Remove(generatedIndex); err != nil && !os.IsNotExist(err) {
		t.Fatal(err)
	}
	planProject := loadSoftwareArchitecture(t, planRoot, "")
	observation, err := host.ObserveProjection(planProject, "0.11.0", host.Hash([]byte("software-architecture-test-tool")))
	if err != nil {
		t.Fatal(err)
	}
	if len(observation.Drift) == 0 {
		t.Fatal("new fixture should expose missing or changed generated projections to read-only observe")
	}
	plan, err := host.PlanProjection(planProject, "0.11.0", host.Hash([]byte("software-architecture-test-tool")))
	if err != nil {
		t.Fatal(err)
	}
	if len(plan.Operations) == 0 {
		t.Fatal("projection plan did not produce concrete local operations")
	}
	projectFile := filepath.Join(planRoot, "markitect.yaml")
	projectBytes := mustReadSoftwareFile(t, projectFile)
	projectBytes = bytes.Replace(projectBytes, []byte("name: software-architecture-consumer"), []byte("name: changed-software-architecture-consumer"), 1)
	if err := os.WriteFile(projectFile, projectBytes, 0644); err != nil {
		t.Fatal(err)
	}
	changedProject := loadSoftwareArchitecture(t, planRoot, "")
	if err := host.ValidateProjectionPlan(changedProject, plan, "0.11.0", host.Hash([]byte("software-architecture-test-tool"))); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("source edit should invalidate a saved projection plan before any write, got %v", err)
	}
}

func containsSoftwareString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func setSoftwareResourceLabel(t *testing.T, root string, project *host.Project, key, label, value string) {
	t.Helper()
	resource := *project.Graph.Resources[key]
	resource.Metadata.Labels = map[string]string{}
	for name, current := range project.Graph.Resources[key].Metadata.Labels {
		resource.Metadata.Labels[name] = current
	}
	if value == "" {
		delete(resource.Metadata.Labels, label)
	} else {
		resource.Metadata.Labels[label] = value
	}
	if len(resource.Metadata.Labels) == 0 {
		resource.Metadata.Labels = nil
	}
	encoded, err := authoring.Encode(resource)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, filepath.FromSlash(resource.Path)), encoded, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeSoftwareResource(t *testing.T, root string, resource authoring.Resource) {
	t.Helper()
	encoded, err := authoring.Encode(resource)
	if err != nil {
		t.Fatal(err)
	}
	name := strings.ToLower(resource.Kind) + "-" + resource.Metadata.Name + ".yaml"
	if err := os.WriteFile(filepath.Join(root, "resources", name), encoded, 0644); err != nil {
		t.Fatal(err)
	}
}
