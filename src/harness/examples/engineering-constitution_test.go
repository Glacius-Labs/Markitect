package examples

import (
	"crypto/sha256"
	"encoding/hex"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring/contentpackage"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

const (
	constitutionV1API     = "engineering.markitect.org/v1alpha1"
	constitutionV2API     = "engineering.markitect.org/v1beta1"
	constitutionUseCaseV1 = "engineering/engineering.markitect.org/v1alpha1/UseCase/create-order"
	constitutionUseCaseV2 = "engineering/engineering.markitect.org/v1beta1/UseCase/create-order"
)

func engineeringConstitutionRoot(t *testing.T) string {
	t.Helper()
	return filepath.Join(harnessRepositoryRoot(t), "examples", "engineering-constitution")
}

func copyEngineeringConstitution(t *testing.T, destination string) {
	t.Helper()
	source := engineeringConstitutionRoot(t)
	if err := filepath.WalkDir(source, func(name string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, name)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
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

func loadEngineeringConstitution(t *testing.T, root string) *host.Project {
	t.Helper()
	project, err := host.Load(root, "")
	if err != nil {
		t.Fatal(err)
	}
	return project
}

func TestEngineeringConstitutionV1PackageDrivesModelContextAndViews(t *testing.T) {
	root := engineeringConstitutionRoot(t)
	project := loadEngineeringConstitution(t, root)
	if len(project.Diagnostics) != 0 {
		t.Fatalf("v1 constitution should compile cleanly: %#v", project.Diagnostics)
	}
	if got := project.Graph.Project.Spec.Packages[0].Version; got != "1.0.0" {
		t.Fatalf("fixture package version = %q, want exact v1.0.0 pin", got)
	}
	model, err := host.CompileModel(project)
	if err != nil {
		t.Fatal(err)
	}
	modulePolicy := findConstitutionResult(model.PolicyResults, constitutionV1API, "module-depends-on-core-only", "engineering/engineering.markitect.org/v1alpha1/Module/orders")
	if modulePolicy.Status != core.PolicyPassed {
		t.Fatalf("canonical Module-to-Core policy result = %#v", modulePolicy)
	}

	entry := "engineering/Skill/add-order"
	compiled, err := host.CompileContext(project, entry, "0.11.0")
	if err != nil {
		t.Fatal(err)
	}
	resources, domains := map[string]bool{}, map[string]string{}
	for _, input := range compiled.Inputs {
		if input.Resource != nil {
			resources[input.Resource.GraphKey()] = true
		}
		if input.Role == "domain" {
			domains[input.Key] = input.Text
		}
	}
	for _, key := range []string{
		entry,
		"engineering/engineering.markitect.org/v1alpha1/UseCase/create-order",
		"engineering/engineering.markitect.org/v1alpha1/Module/orders",
		"engineering/engineering.markitect.org/v1alpha1/Core/platform-core",
		"engineering/engineering.markitect.org/v1alpha1/Handler/create-order-handler",
		"engineering-constitution::constitution/Workflow/vertical-slice",
	} {
		if !resources[key] {
			t.Errorf("Skill context omitted %s", key)
		}
	}
	if len(domains) != 1 {
		t.Fatalf("selected Package Domain definition absent from context: %#v", domains)
	}
	for _, value := range []string{"module-depends-on-core-only", "allowed-targets", "values:", "Core"} {
		if !strings.Contains(strings.Join(mapValues(domains), "\n"), value) {
			t.Errorf("context Domain source omitted policy value %q", value)
		}
	}
	if len(compiled.PolicyResults) == 0 {
		t.Fatal("agent context omitted model policy results")
	}

	outputs, err := host.GenerateOutputs(project)
	if err != nil {
		t.Fatal(err)
	}
	contract := string(outputs["docs/markitect/_domains/engineering.markitect.org.v1alpha1.domain.md"])
	for _, value := range []string{"module-depends-on-core-only", "allowed-targets", "Core", "Policy outcomes", "PASSED"} {
		if !strings.Contains(contract, value) {
			t.Errorf("generated Domain contract omitted %q: %s", value, contract)
		}
	}
	useCaseView := findOutputContaining(outputs, "## Applicable constraints")
	if !strings.Contains(useCaseView, "Policy outcomes") {
		t.Errorf("generated resource view omitted policy outcome: %s", useCaseView)
	}
}

func TestEngineeringConstitutionStructuralRelationsAndConflictFailClosed(t *testing.T) {
	for _, test := range []struct {
		name     string
		mutate   func(t *testing.T, root string, project *host.Project)
		wantCode string
		alsoCode string
	}{
		{
			name: "zero handlers",
			mutate: func(t *testing.T, root string, project *host.Project) {
				writeEngineeringResource(t, root, project, constitutionUseCaseV1, func(data map[string]any) { data["handlers"] = []any{} })
			},
			wantCode: "relation.min-targets",
		},
		{
			name: "two handlers",
			mutate: func(t *testing.T, root string, project *host.Project) {
				writeEngineeringResource(t, root, project, constitutionUseCaseV1, func(data map[string]any) {
					data["handlers"] = append(data["handlers"].([]any), map[string]any{"kind": "Handler", "name": "second-handler", "namespace": "engineering"})
				})
				second := authoring.Resource{Core: authoring.Core{APIVersion: constitutionV1API, Kind: "Handler", Metadata: core.Metadata{Name: "second-handler", Namespace: "engineering"}, Data: map[string]any{"summary": "A second structural handler."}}}
				writeEngineeringResourceFile(t, filepath.Join(root, "resources", "second-handler.yaml"), second)
			},
			wantCode: "relation.max-targets",
		},
		{
			name: "handler reference has forbidden target kind",
			mutate: func(t *testing.T, root string, project *host.Project) {
				writeEngineeringResource(t, root, project, constitutionUseCaseV1, func(data map[string]any) {
					data["handlers"] = []any{map[string]any{"kind": "Module", "name": "orders", "namespace": "engineering"}}
				})
			},
			wantCode: "parse",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			copyEngineeringConstitution(t, root)
			before := loadEngineeringConstitution(t, root)
			test.mutate(t, root, before)
			// A fabricated exception cannot waive structural relation/ref errors.
			before.Graph.Project.Spec.PolicyDate = "2026-10-02"
			before.Graph.Project.Spec.PolicyExceptions = []core.PolicyException{{
				Name: "not-a-waiver", APIVersion: constitutionV1API, Constraint: "handledBy",
				Subject: constitutionUseCaseV1, ConstraintDigest: "sha256:" + strings.Repeat("a", 64),
				SubjectDigest: "sha256:" + strings.Repeat("b", 64), Rationale: "Testing structural boundary.",
				Owner: "architecture-owner", Decision: "Attempted waiver.",
			}}
			writeEngineeringProject(t, root, before.Graph.Project)
			invalid := loadEngineeringConstitution(t, root)
			if !hasConstitutionDiagnostic(invalid.Diagnostics, test.wantCode) {
				t.Fatalf("wanted structural diagnostic %q, got %#v", test.wantCode, invalid.Diagnostics)
			}
			if hasConstitutionResult(invalid.Graph.Core.PolicyResults, "handledBy", constitutionUseCaseV1) {
				t.Fatal("structural relation failure was incorrectly converted to a policy result")
			}
			if !hasConstitutionDiagnostic(invalid.Diagnostics, "policy.exception.unknown") && !hasConstitutionDiagnostic(invalid.Diagnostics, "policy.exception.stale") {
				t.Fatalf("attempted waiver was not rejected: %#v", invalid.Diagnostics)
			}
		})
	}
}

func TestEngineeringConstitutionV2PackageMigrationAndWaiverLifecycle(t *testing.T) {
	v1 := loadEngineeringConstitution(t, engineeringConstitutionRoot(t))
	root := t.TempDir()
	copyEngineeringConstitution(t, root)
	v2 := migrateEngineeringConstitutionToV2(t, root)
	if len(v2.Diagnostics) == 0 || !hasConstitutionDiagnostic(v2.Diagnostics, "constraint.each-usecase-has-validator") {
		t.Fatalf("v2 should identify the newly affected UseCase: %#v", v2.Diagnostics)
	}
	missing := findConstitutionResult(v2.Graph.Core.PolicyResults, constitutionV2API, "each-usecase-has-validator", constitutionUseCaseV2)
	if missing.Status != core.PolicyFailed || !strings.Contains(missing.Message, "0 targets") {
		t.Fatalf("v2 validator policy did not return a per-subject failure: %#v", missing)
	}
	if len(v2.DomainInputs) != 1 || v2.DomainInputs[0].Package != "engineering-constitution" || v2.DomainInputs[0].Path != "domains/software.yaml" {
		t.Fatalf("selected Domain provenance = %#v, want the pinned engineering-constitution domain path", v2.DomainInputs)
	}
	if got := v2.Graph.Project.Spec.Packages[0].SHA256; got == v1.Graph.Project.Spec.Packages[0].SHA256 {
		t.Fatal("v1 and v2 package pins unexpectedly select identical archive digests")
	}
	impact := host.Changes(v1, v2)
	if !containsString(impact.Affected, constitutionUseCaseV2) {
		t.Fatalf("exact package migration did not conservatively affect the UseCase: %#v", impact)
	}

	writeEngineeringException(t, root, v2, missing, "validator-transition", "2026-10-02", "2026-10-30")
	waived := loadEngineeringConstitution(t, root)
	if len(waived.Diagnostics) != 0 {
		t.Fatalf("valid dated exception should leave no model diagnostics: %#v", waived.Diagnostics)
	}
	waiver := findConstitutionResult(waived.Graph.Core.PolicyResults, constitutionV2API, "each-usecase-has-validator", constitutionUseCaseV2)
	if waiver.Status != core.PolicyWaived || waiver.ExceptionName != "validator-transition" || waiver.Owner != "architecture-owner" || waiver.PolicyDate != "2026-10-02" || !strings.Contains(waiver.Decision, "temporary") {
		t.Fatalf("exception did not yield an explicit waived result: %#v", waiver)
	}

	compiled, err := host.CompileContext(waived, "engineering/Skill/add-order", "0.11.0")
	if err != nil {
		t.Fatal(err)
	}
	contextResult := findConstitutionResult(compiled.PolicyResults, constitutionV2API, "each-usecase-has-validator", constitutionUseCaseV2)
	if contextResult.Status != core.PolicyWaived || contextResult.Rationale != waiver.Rationale || contextResult.Decision != waiver.Decision {
		t.Fatalf("agent context did not receive the exact policy exception decision: %#v", contextResult)
	}
	var selectedDomain string
	for _, input := range compiled.Inputs {
		if input.Role == "domain" {
			selectedDomain = input.Text
			if input.PackageVersion != "2.0.0" {
				t.Errorf("context Domain is not bound to the v2 pin: %#v", input)
			}
		}
	}
	if !strings.Contains(selectedDomain, "each-usecase-has-validator") || !strings.Contains(selectedDomain, "min: 1") {
		t.Fatalf("agent context did not carry the exact v2 policy definition: %s", selectedDomain)
	}
	outputs, err := host.GenerateOutputs(waived)
	if err != nil {
		t.Fatal(err)
	}
	contract := string(outputs["docs/markitect/_domains/engineering.markitect.org.v1beta1.domain.md"])
	for _, value := range []string{"each-usecase-has-validator", "min: 1", "WAIVED", "validator-transition", "rationale:", "owner: architecture-owner", "decision:", "policyDate: 2026-10-02", "expiresOn: 2026-10-30"} {
		if !strings.Contains(contract, value) {
			t.Errorf("human Domain view omitted %q: %s", value, contract)
		}
	}

	t.Run("stale subject digest", func(t *testing.T) {
		staleRoot := t.TempDir()
		copyEngineeringConstitution(t, staleRoot)
		staleV2 := migrateEngineeringConstitutionToV2(t, staleRoot)
		staleResult := findConstitutionResult(staleV2.Graph.Core.PolicyResults, constitutionV2API, "each-usecase-has-validator", constitutionUseCaseV2)
		writeEngineeringException(t, staleRoot, staleV2, staleResult, "stale-validator", "2026-10-02", "2026-10-30")
		waivedV2 := loadEngineeringConstitution(t, staleRoot)
		writeEngineeringResource(t, staleRoot, waivedV2, constitutionUseCaseV2, func(data map[string]any) { data["summary"] = "A changed subject must invalidate its exception." })
		stale := loadEngineeringConstitution(t, staleRoot)
		if !hasConstitutionDiagnostic(stale.Diagnostics, "policy.exception.stale") {
			t.Fatalf("changed subject retained stale exception: %#v", stale.Diagnostics)
		}
	})

	t.Run("expired date", func(t *testing.T) {
		expiredRoot := t.TempDir()
		copyEngineeringConstitution(t, expiredRoot)
		expiredV2 := migrateEngineeringConstitutionToV2(t, expiredRoot)
		failed := findConstitutionResult(expiredV2.Graph.Core.PolicyResults, constitutionV2API, "each-usecase-has-validator", constitutionUseCaseV2)
		writeEngineeringException(t, expiredRoot, expiredV2, failed, "expired-validator", "2026-10-30", "2026-10-30")
		expired := loadEngineeringConstitution(t, expiredRoot)
		if !hasConstitutionDiagnostic(expired.Diagnostics, "policy.exception.expired") {
			t.Fatalf("exception expiry was not evaluated against pinned policyDate: %#v", expired.Diagnostics)
		}
	})

	t.Run("unknown constraint", func(t *testing.T) {
		unknownRoot := t.TempDir()
		copyEngineeringConstitution(t, unknownRoot)
		unknownV2 := migrateEngineeringConstitutionToV2(t, unknownRoot)
		failed := findConstitutionResult(unknownV2.Graph.Core.PolicyResults, constitutionV2API, "each-usecase-has-validator", constitutionUseCaseV2)
		exception := exceptionForConstitutionResult(failed, "unknown-validator", "2026-10-02", "2026-10-30")
		exception.Constraint = "no-such-policy"
		writeEngineeringProjectWithExceptions(t, unknownRoot, unknownV2.Graph.Project, []core.PolicyException{exception}, "2026-10-02")
		unknown := loadEngineeringConstitution(t, unknownRoot)
		if !hasConstitutionDiagnostic(unknown.Diagnostics, "policy.exception.unknown") {
			t.Fatalf("unknown exception target was accepted: %#v", unknown.Diagnostics)
		}
	})

	t.Run("unused exception", func(t *testing.T) {
		unusedRoot := t.TempDir()
		copyEngineeringConstitution(t, unusedRoot)
		unusedV2 := migrateEngineeringConstitutionToV2(t, unusedRoot)
		writeEngineeringResourceFile(t, filepath.Join(unusedRoot, "resources", "create-order-validator.yaml"), authoring.Resource{Core: authoring.Core{APIVersion: constitutionV2API, Kind: "Validator", Metadata: core.Metadata{Name: "create-order-validator", Namespace: "engineering"},
			Data: map[string]any{"summary": "Validates the order request."}},
		})
		writeEngineeringResource(t, unusedRoot, unusedV2, constitutionUseCaseV2, func(data map[string]any) {
			data["validators"] = []any{map[string]any{"kind": "Validator", "name": "create-order-validator", "namespace": "engineering"}}
		})
		passing := loadEngineeringConstitution(t, unusedRoot)
		passed := findConstitutionResult(passing.Graph.Core.PolicyResults, constitutionV2API, "each-usecase-has-validator", constitutionUseCaseV2)
		if passed.Status != core.PolicyPassed {
			t.Fatalf("migrated UseCase should satisfy v2: %#v", passed)
		}
		exception := exceptionForConstitutionResult(passed, "unneeded-validator", "2026-10-02", "2026-10-30")
		writeEngineeringProjectWithExceptions(t, unusedRoot, passing.Graph.Project, []core.PolicyException{exception}, "2026-10-02")
		unused := loadEngineeringConstitution(t, unusedRoot)
		if !hasConstitutionDiagnostic(unused.Diagnostics, "policy.exception.unneeded") {
			t.Fatalf("exception for a passing result was silently retained: %#v", unused.Diagnostics)
		}
	})
}

func TestEngineeringConstitutionConflictingComposedRulesRemainVisible(t *testing.T) {
	root := t.TempDir()
	copyEngineeringConstitution(t, root)
	_ = migrateEngineeringConstitutionToV2(t, root)
	packageFiles := readConstitutionTree(t, filepath.Join(engineeringConstitutionRoot(t), "constitution-package-v2"))
	domain, err := authoring.ParseDomain("domains/software.yaml", packageFiles["domains/software.yaml"])
	if err != nil {
		t.Fatal(err)
	}
	domain.Constraints = append(domain.Constraints, core.ConstraintDefinition{
		Name:   "conflicting-module-depends-on-modules-only",
		Select: core.ResourceSelector{Kind: "Module"},
		Assert: core.ConstraintAssertion{Op: "allowed-targets", Relation: "dependsOn", Values: []any{"Module"}},
	})
	packageFiles["domains/software.yaml"], err = authoring.EncodeDomain(domain)
	if err != nil {
		t.Fatal(err)
	}
	archive, err := contentpackage.Build(packageFiles)
	if err != nil {
		t.Fatal(err)
	}
	installEngineeringPackage(t, root, "2.0.0", archive)
	conflict := loadEngineeringConstitution(t, root)
	if !hasConstitutionDiagnostic(conflict.Diagnostics, "constraint.conflicting-module-depends-on-modules-only") {
		t.Fatalf("contradictory Domain rules were silently overridden: %#v", conflict.Diagnostics)
	}
	coreRule := findConstitutionResult(conflict.Graph.Core.PolicyResults, constitutionV2API, "module-depends-on-core-only", "engineering/engineering.markitect.org/v1beta1/Module/orders")
	moduleRule := findConstitutionResult(conflict.Graph.Core.PolicyResults, constitutionV2API, "conflicting-module-depends-on-modules-only", "engineering/engineering.markitect.org/v1beta1/Module/orders")
	if coreRule.Status != core.PolicyPassed || moduleRule.Status != core.PolicyFailed {
		t.Fatalf("composed conflict must retain both rule outcomes, got core=%#v modules=%#v", coreRule, moduleRule)
	}
}

func TestEngineeringConstitutionPackageSourcesAreCanonical(t *testing.T) {
	for _, version := range []string{"v1", "v2"} {
		t.Run(version, func(t *testing.T) {
			files := readConstitutionTree(t, filepath.Join(engineeringConstitutionRoot(t), "constitution-package-"+version))
			definition, err := authoring.ParseDomain("domains/software.yaml", files["domains/software.yaml"])
			if err != nil {
				t.Fatal(err)
			}
			registry := authoring.NewRegistry()
			if err := registry.AddDomain(definition); err != nil {
				t.Fatal(err)
			}
			for name, data := range files {
				if filepath.Ext(name) != ".yaml" && filepath.Ext(name) != ".yml" {
					continue
				}
				var normalized []byte
				var err error
				if name == "domains/software.yaml" {
					normalized, err = authoring.EncodeDomain(definition)
				} else {
					resource, parseErr := authoring.ParseWithRegistry(name, data, registry)
					if parseErr != nil {
						t.Fatalf("parse %s: %v", name, parseErr)
					}
					normalized, err = authoring.Encode(*resource)
				}
				if err != nil {
					t.Fatalf("normalize %s: %v", name, err)
				}
				if string(normalized) != string(data) {
					t.Errorf("%s is not in canonical format", name)
				}
			}
		})
	}
}

func migrateEngineeringConstitutionToV2(t *testing.T, root string) *host.Project {
	t.Helper()
	archive := buildEngineeringPackage(t, "constitution-package-v2")
	installEngineeringPackage(t, root, "2.0.0", archive)
	for _, name := range []string{"add-order.skill.yaml", "core.yaml", "create-order-handler.yaml", "create-order-usecase.yaml", "order-write-authorization.yaml", "orders-module.yaml"} {
		file := filepath.Join(root, "resources", name)
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		updated := strings.ReplaceAll(string(data), constitutionV1API, constitutionV2API)
		if updated == string(data) {
			t.Fatalf("resource migration did not update %s", name)
		}
		if err := os.WriteFile(file, []byte(updated), 0644); err != nil {
			t.Fatal(err)
		}
	}
	return loadEngineeringConstitution(t, root)
}

func buildEngineeringPackage(t *testing.T, packageSource string) []byte {
	t.Helper()
	files := readConstitutionTree(t, filepath.Join(engineeringConstitutionRoot(t), packageSource))
	archive, err := contentpackage.Build(files)
	if err != nil {
		t.Fatalf("build %s: %v", packageSource, err)
	}
	return archive
}

func readConstitutionTree(t *testing.T, root string) map[string][]byte {
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

func installEngineeringPackage(t *testing.T, root, version string, archive []byte) {
	t.Helper()
	digest := sha256.Sum256(archive)
	sha := hex.EncodeToString(digest[:])
	packageVersion := "1.0.0"
	if version == "2.0.0" {
		packageVersion = version
	}
	archivePath := filepath.Join(root, ".markitect", "packages", "engineering-constitution-"+packageVersion+".zip")
	if err := os.MkdirAll(filepath.Dir(archivePath), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, archive, 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(root, "markitect.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	project, err := authoring.Parse("markitect.yaml", data)
	if err != nil {
		t.Fatal(err)
	}
	project.Spec.Packages = []authoring.PackagePin{{Name: "engineering-constitution", Version: packageVersion,
		Source:  "fixture:engineering-constitution-package-v" + strings.TrimSuffix(packageVersion, ".0.0"),
		Archive: ".markitect/packages/engineering-constitution-" + packageVersion + ".zip", SHA256: sha}}
	encoded, err := authoring.Encode(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, encoded, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeEngineeringException(t *testing.T, root string, project *host.Project, failed core.PolicyResult, name, policyDate, expiresOn string) {
	t.Helper()
	exception := exceptionForConstitutionResult(failed, name, policyDate, expiresOn)
	writeEngineeringProjectWithExceptions(t, root, project.Graph.Project, []core.PolicyException{exception}, policyDate)
}

func exceptionForConstitutionResult(result core.PolicyResult, name, policyDate, expiresOn string) core.PolicyException {
	return core.PolicyException{
		Name: name, APIVersion: result.APIVersion, Constraint: result.Constraint, Subject: result.Subject,
		ConstraintDigest: result.ConstraintDigest, SubjectDigest: result.SubjectDigest,
		Rationale: "Keep the existing UseCase unblocked while its Validator slice is implemented.",
		Owner:     "architecture-owner", Decision: "Approved as a temporary, reviewable architecture exception.",
		ExpiresOn: expiresOn,
	}
}

func writeEngineeringProjectWithExceptions(t *testing.T, root string, project *authoring.Resource, exceptions []core.PolicyException, policyDate string) {
	t.Helper()
	updated := *project
	updated.Spec.PolicyDate = policyDate
	updated.Spec.PolicyExceptions = append([]core.PolicyException(nil), exceptions...)
	writeEngineeringProject(t, root, &updated)
}

func writeEngineeringProject(t *testing.T, root string, project *authoring.Resource) {
	t.Helper()
	data, err := authoring.Encode(project)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "markitect.yaml"), data, 0644); err != nil {
		t.Fatal(err)
	}
}

func writeEngineeringResource(t *testing.T, root string, project *host.Project, key string, mutate func(map[string]any)) {
	t.Helper()
	resource := project.Graph.Resources[key]
	if resource == nil {
		t.Fatalf("fixture resource %s is missing", key)
	}
	updated := *resource
	updated.Data = make(map[string]any, len(resource.Data))
	for key, value := range resource.Data {
		updated.Data[key] = value
	}
	mutate(updated.Data)
	writeEngineeringResourceFile(t, filepath.Join(root, filepath.FromSlash(resource.Path)), updated)
}

func writeEngineeringResourceFile(t *testing.T, file string, resource authoring.Resource) {
	t.Helper()
	data, err := authoring.Encode(&resource)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func findConstitutionResult(results []core.PolicyResult, apiVersion, constraint, subject string) core.PolicyResult {
	for _, result := range results {
		if result.APIVersion == apiVersion && result.Constraint == constraint && result.Subject == subject {
			return result
		}
	}
	return core.PolicyResult{}
}

func hasConstitutionResult(results []core.PolicyResult, constraint, subject string) bool {
	for _, result := range results {
		if result.Constraint == constraint && result.Subject == subject {
			return true
		}
	}
	return false
}

func hasConstitutionDiagnostic(diagnostics []core.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}

func findOutputContaining(outputs map[string][]byte, value string) string {
	for _, output := range outputs {
		if strings.Contains(string(output), value) {
			return string(output)
		}
	}
	return ""
}

func mapValues(values map[string]string) []string {
	result := make([]string, 0, len(values))
	for _, value := range values {
		result = append(result, value)
	}
	return result
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
