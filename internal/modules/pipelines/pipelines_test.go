package pipelines

import (
	"reflect"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func pipelineModel() core.SemanticModel {
	return core.SemanticModel{
		APIVersion:       core.SemanticModelVersion,
		StructuralStatus: "passed",
		Snapshot:         core.ModelSnapshot{ID: "commit-2", Digest: strings.Repeat("a", 64)},
		ModelDigest:      strings.Repeat("b", 64),
		Resources:        []core.ModelResource{{Identity: core.ModelIdentity{Key: "development/Rule/ci-owner"}}},
	}
}

func TestCheckConfiguredGitHubAndAzureLiteralReferences(t *testing.T) {
	githubBytes := []byte("name: test\njobs:\n  build:\n    steps:\n      - run: go test ./...\n")
	azureBytes := []byte("steps:\n  - script: dotnet test\n")
	config := Config{APIVersion: ConfigVersion, Pipelines: []Pipeline{
		{
			Name: "github-tests", Provider: "github-actions", Path: ".github/workflows/tests.yml",
			Digest: digest(githubBytes), Owner: "development/Rule/ci-owner",
			ExpectedChecks: []ExpectedCheck{{Name: "go-tests", YAMLPath: "/jobs/build/steps/0/run"}},
		},
		{
			Name: "azure-tests", Provider: "azure-devops", Path: "azure-pipelines.yaml",
			Digest:         digest(azureBytes),
			Owner:          "development/Rule/ci-owner",
			ExpectedChecks: []ExpectedCheck{{Name: "dotnet-tests", YAMLPath: "/steps/0/script"}},
		},
	}}
	report, err := Check(Input{
		Model: pipelineModel(), Config: config,
		Artifacts: map[string][]byte{
			".github/workflows/tests.yml": githubBytes,
			"azure-pipelines.yaml":        azureBytes,
		},
		Ownership: map[string][]string{
			".github/workflows/tests.yml": {"development/Rule/ci-owner"},
			"azure-pipelines.yaml":        {"development/Rule/ci-owner"},
		},
		CheckFacts: []CheckFact{
			{Name: "go-tests", Reference: "go test ./..."},
			{Name: "dotnet-tests", Reference: "dotnet test"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusPassed || report.PipelinesChecked != 2 || report.CheckReferencesChecked != 2 || len(report.Findings) != 0 {
		t.Fatalf("exact configured references should pass: %#v", report)
	}
}

func TestCheckReportsStaleDigestOwnerAndLiteralMismatch(t *testing.T) {
	content := []byte("jobs:\n  test:\n    steps:\n      - run: go test ./...\n")
	config := Config{APIVersion: ConfigVersion, Pipelines: []Pipeline{{
		Name: "tests", Provider: "github-actions", Path: ".github/workflows/tests.yaml",
		Digest: strings.Repeat("c", 64), Owner: "development/Rule/expected-owner",
		ExpectedChecks: []ExpectedCheck{{Name: "go-tests", YAMLPath: "/jobs/test/steps/0/run"}},
	}}}
	report, err := Check(Input{
		Model: pipelineModel(), Config: config,
		Artifacts:  map[string][]byte{".github/workflows/tests.yaml": content},
		Ownership:  map[string][]string{".github/workflows/tests.yaml": {"development/Rule/other-owner"}},
		CheckFacts: []CheckFact{{Name: "go-tests", Reference: "go test ./... -race"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, code := range []string{"pipelines.digest.mismatch", "pipelines.owner.resource-missing", "pipelines.owner.link-missing", "pipelines.check-reference.mismatch"} {
		if !hasCode(report.Findings, code) {
			t.Errorf("missing %s in %#v", code, report.Findings)
		}
	}
}

func TestCheckReportsMissingArtifactAndExplicitCheckFact(t *testing.T) {
	config := Config{APIVersion: ConfigVersion, Pipelines: []Pipeline{{
		Name: "azure-build", Provider: "azure-devops", Path: "ci/azure.yml",
		Digest:         strings.Repeat("d", 64),
		Owner:          "development/Rule/ci-owner",
		ExpectedChecks: []ExpectedCheck{{Name: "build", YAMLPath: "/steps/0/script"}},
	}}}
	report, err := Check(Input{Model: pipelineModel(), Config: config})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(report.Findings, "pipelines.artifact.missing") {
		t.Fatalf("missing exact pipeline bytes were not reported: %#v", report.Findings)
	}
	if report.CheckReferencesChecked != 0 {
		t.Fatalf("missing file must not be parsed: %#v", report)
	}

	content := []byte("steps:\n  - script: dotnet test\n")
	report, err = Check(Input{
		Model: pipelineModel(), Config: config,
		Artifacts: map[string][]byte{"ci/azure.yml": content},
		Ownership: map[string][]string{"ci/azure.yml": {"development/Rule/ci-owner"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !hasCode(report.Findings, "pipelines.check-fact.missing") {
		t.Fatalf("unprovided check fact must be explicit: %#v", report.Findings)
	}
}

func TestDecodeConfigRejectsUnsupportedProviderAndPointer(t *testing.T) {
	for _, input := range []string{
		"apiVersion: " + ConfigVersion + "\npipelines:\n- name: other\n  provider: circleci\n  path: .circleci/config.yml\n  owner: owner\n",
		"apiVersion: " + ConfigVersion + "\npipelines:\n- name: tests\n  provider: github-actions\n  path: .github/workflows/test.yml\n  owner: owner\n  expectedChecks:\n  - name: test\n    yamlPath: jobs.test.steps.0.run\n",
	} {
		if _, err := DecodeConfig([]byte(input)); err == nil {
			t.Fatalf("invalid config accepted: %s", input)
		}
	}
}

func TestEmptyScopeIsNotConfigured(t *testing.T) {
	report, err := Check(Input{Model: pipelineModel(), Config: Config{APIVersion: ConfigVersion}})
	if err != nil {
		t.Fatal(err)
	}
	if report.Status != StatusNotConfigured || len(report.Findings) != 1 || report.Findings[0].Code != "pipelines.not-configured" {
		t.Fatalf("empty scope must not imply pipeline verification: %#v", report)
	}
}

func hasCode(findings []Finding, code string) bool {
	for _, finding := range findings {
		if finding.Code == code {
			return true
		}
	}
	return false
}

func TestFindingOrderIsStableAcrossPipelineConfigOrder(t *testing.T) {
	pipelines := []Pipeline{
		{Name: "second", Provider: "azure-devops", Path: "azure-second.yml", Digest: strings.Repeat("c", 64), Owner: "development/Rule/ci-owner"},
		{Name: "first", Provider: "github-actions", Path: ".github/workflows/first.yml", Digest: strings.Repeat("c", 64), Owner: "development/Rule/ci-owner"},
	}
	one, err := Check(Input{Model: pipelineModel(), Config: Config{APIVersion: ConfigVersion, Pipelines: pipelines}})
	if err != nil {
		t.Fatal(err)
	}
	two, err := Check(Input{Model: pipelineModel(), Config: Config{APIVersion: ConfigVersion, Pipelines: []Pipeline{pipelines[1], pipelines[0]}}})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(one, two) {
		t.Fatalf("report depends on input order:\n%#v\n%#v", one, two)
	}
}

func TestReportBindsNormalizedConfigWithoutChangingModel(t *testing.T) {
	model := pipelineModel()
	first := Config{APIVersion: ConfigVersion, Pipelines: []Pipeline{{
		Name: "tests", Provider: "github-actions", Path: ".github/workflows/ci.yaml",
		Digest: strings.Repeat("c", 64), Owner: "development/Rule/ci-owner",
		ExpectedChecks: []ExpectedCheck{{Name: "go-tests", YAMLPath: "/jobs/test/steps/0/run"}},
	}}}
	second := first
	second.Pipelines = append([]Pipeline(nil), first.Pipelines...)
	second.Pipelines[0].ExpectedChecks = append([]ExpectedCheck(nil), first.Pipelines[0].ExpectedChecks...)
	second.Pipelines[0].ExpectedChecks[0].YAMLPath = "/jobs/build/steps/0/run"
	one, err := Check(Input{Model: model, Config: first})
	if err != nil {
		t.Fatal(err)
	}
	two, err := Check(Input{Model: model, Config: second})
	if err != nil {
		t.Fatal(err)
	}
	if one.ModelDigest != two.ModelDigest || one.SnapshotDigest != two.SnapshotDigest {
		t.Fatal("test must hold the model snapshot fixed")
	}
	if one.ConfigDigest == "" || one.ConfigDigest == two.ConfigDigest {
		t.Fatalf("report must bind the changed module config: first=%q second=%q", one.ConfigDigest, two.ConfigDigest)
	}
}

func TestConfigRejectsPortableCaseFoldPathDuplicates(t *testing.T) {
	config := Config{APIVersion: ConfigVersion, Pipelines: []Pipeline{
		{Name: "upper", Provider: "github-actions", Path: ".github/workflows/CI.yaml", Digest: strings.Repeat("c", 64), Owner: "owner"},
		{Name: "lower", Provider: "github-actions", Path: ".github/workflows/ci.yaml", Digest: strings.Repeat("c", 64), Owner: "owner"},
	}}
	_, err := Check(Input{Model: pipelineModel(), Config: config})
	if err == nil || !strings.Contains(err.Error(), "portable case folding") {
		t.Fatalf("case-fold colliding pipeline paths must be rejected: %v", err)
	}
}
