package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/app"
	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/source"
	"go.yaml.in/yaml/v3"
)

const reviewConfigPath = "markitect-review.yaml"

type cliReviewConfig struct {
	Question      string `yaml:"question"`
	PromptVersion string `yaml:"promptVersion"`
	Model         string `yaml:"model"`
	Effort        string `yaml:"effort"`
	AllowReuse    bool   `yaml:"allowReuse"`
}

type cliReviewRecord struct {
	SchemaVersion  int             `yaml:"schemaVersion"`
	Entry          string          `yaml:"entry"`
	Revision       string          `yaml:"revision"`
	SnapshotDigest string          `yaml:"snapshotDigest"`
	ContextDigest  string          `yaml:"contextDigest"`
	ToolDigest     string          `yaml:"toolDigest"`
	Version        string          `yaml:"version"`
	Config         cliReviewConfig `yaml:"config"`
	Report         string          `yaml:"report"`
	ReportDigest   string          `yaml:"reportDigest"`
}

type cliReviewDecision struct {
	Status           string   `yaml:"status"`
	Reasons          []string `yaml:"reasons"`
	OriginalRevision string   `yaml:"originalRevision"`
	Candidate        string   `yaml:"candidate"`
	Entry            string   `yaml:"entry"`
	Report           string   `yaml:"report"`
}

func reviewConfigData(t *testing.T, question string, allowReuse bool) []byte {
	t.Helper()
	data, err := format.Encode(cliReviewConfig{
		Question:      question,
		PromptVersion: "review-prompt-v1",
		Model:         "gpt-6-sol",
		Effort:        "high",
		AllowReuse:    allowReuse,
	})
	if err != nil {
		t.Fatal(err)
	}
	return data
}

func newReviewCLIRepo(t *testing.T) cliRepo {
	t.Helper()
	repo := newCLIRepo(t, false)
	writeRepoFile(t, repo.root, reviewConfigPath, reviewConfigData(t, "Does this preserve the required policy?", true))
	git(t, repo.root, "add", reviewConfigPath)
	git(t, repo.root, "commit", "-m", "add immutable review config")
	repo.base = git(t, repo.root, "rev-parse", "HEAD")
	return repo
}

func reviewArgs(repo cliRepo, revision, config, report, evidence string) []string {
	args := []string{"review", "--repo", repo.root, "--revision", revision, "--namespace", cliNamespace, "--kind", "Skill", "--name", "entry", "--config", config}
	if report != "" {
		args = append(args, "--report", report)
	}
	if evidence != "" {
		args = append(args, "--evidence", evidence)
	}
	return args
}

func writeReviewRecord(t *testing.T, recordPath, data string) {
	t.Helper()
	if err := os.WriteFile(recordPath, []byte(data), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestReviewRecordsAndReusesTheSameFixedSnapshot(t *testing.T) {
	repo := newReviewCLIRepo(t)
	files := t.TempDir()
	reportPath := filepath.Join(files, "review.md")
	evidencePath := filepath.Join(files, "record.yaml")
	reviewText := "Reviewed the selected skill and its required Rule. No findings.\n"
	if err := os.WriteFile(reportPath, []byte(reviewText), 0644); err != nil {
		t.Fatal(err)
	}

	// A fixed revision must source this configuration from Git even while the
	// working copy has a different value.
	writeRepoFile(t, repo.root, reviewConfigPath, reviewConfigData(t, "Uncommitted question must be ignored.", false))
	args := reviewArgs(repo, repo.base, reviewConfigPath, reportPath, "")
	code, recordYAML, stderr := invoke(args...)
	if code != 0 {
		t.Fatalf("record review exit=%d stderr=%s output=%s", code, stderr, recordYAML)
	}
	var record cliReviewRecord
	if err := yaml.Unmarshal([]byte(recordYAML), &record); err != nil {
		t.Fatalf("decode review record: %v\n%s", err, recordYAML)
	}
	snapshot, err := source.Load(repo.root, repo.base)
	if err != nil {
		t.Fatal(err)
	}
	wantEntry := cliNamespace + "/Skill/entry"
	if record.Entry != wantEntry || record.Revision != repo.base || record.SnapshotDigest != snapshot.Digest() {
		t.Fatalf("record does not identify the fixed entry/snapshot: %#v", record)
	}
	if record.ContextDigest == "" || record.ToolDigest == "" || record.Report != reviewText || record.ReportDigest == "" {
		t.Fatalf("record is missing context, tool or report evidence: %#v", record)
	}
	if record.Config.Question != "Does this preserve the required policy?" || !record.Config.AllowReuse {
		t.Fatalf("record used mutable worktree configuration: %#v", record.Config)
	}
	writeReviewRecord(t, evidencePath, recordYAML)

	code, resultYAML, stderr := invoke(reviewArgs(repo, repo.base, reviewConfigPath, "", evidencePath)...)
	if code != 0 {
		t.Fatalf("same-snapshot evidence check exit=%d stderr=%s output=%s", code, stderr, resultYAML)
	}
	decision := decodeYAML[cliReviewDecision](t, resultYAML)
	if decision.Status != "reusable" || decision.Candidate != repo.base || decision.OriginalRevision != repo.base || decision.Report != reviewText {
		t.Fatalf("unexpected same-snapshot decision: %#v", decision)
	}
}

func TestReviewRequiresReviewWhenCandidateContextChanges(t *testing.T) {
	repo := newReviewCLIRepo(t)
	files := t.TempDir()
	reportPath := filepath.Join(files, "review.md")
	evidencePath := filepath.Join(files, "record.yaml")
	if err := os.WriteFile(reportPath, []byte("Previous review report.\n"), 0644); err != nil {
		t.Fatal(err)
	}
	code, recordYAML, stderr := invoke(reviewArgs(repo, repo.base, reviewConfigPath, reportPath, "")...)
	if code != 0 {
		t.Fatalf("record review exit=%d stderr=%s output=%s", code, stderr, recordYAML)
	}
	writeReviewRecord(t, evidencePath, recordYAML)

	base, err := app.Load(repo.root, repo.base)
	if err != nil {
		t.Fatal(err)
	}
	changedRule := core.Resource{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: cliNamespace}, Path: "docs/general/rules/policy.yaml", Spec: core.Spec{Text: "Changed policy after the original review."}}
	data, err := format.Encode(changedRule)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo.root, "docs/general/rules/policy.yaml", data)
	for _, resource := range base.Resources {
		if resource.Kind == "Rule" && resource.Metadata.Name == "policy" {
			resource.Spec = changedRule.Spec
		}
	}
	base.Graph = core.Build(base.Resources)
	generated, err := render.Generate(base.Graph, base.Snapshot.Files)
	if err != nil {
		t.Fatal(err)
	}
	for path, generatedData := range generated {
		writeRepoFile(t, repo.root, path, generatedData)
	}
	git(t, repo.root, "add", "docs/general/rules/policy.yaml", "docs/markitect/sample/rules/policy.rule.md")
	git(t, repo.root, "commit", "-m", "change reviewed dependency")
	candidate := git(t, repo.root, "rev-parse", "HEAD")
	code, resultYAML, stderr := invoke(reviewArgs(repo, candidate, reviewConfigPath, "", evidencePath)...)
	if code != 1 {
		t.Fatalf("changed-context review exit=%d stderr=%s output=%s, want review-required exit 1", code, stderr, resultYAML)
	}
	decision := decodeYAML[cliReviewDecision](t, resultYAML)
	if decision.Status != "review-required" || decision.OriginalRevision != repo.base || decision.Candidate != candidate || len(decision.Reasons) == 0 {
		t.Fatalf("unexpected changed-context decision: %#v", decision)
	}
}

func TestReviewRejectsInvalidInvocationAndConfig(t *testing.T) {
	repo := newReviewCLIRepo(t)
	files := t.TempDir()
	reportPath := filepath.Join(files, "review.md")
	evidencePath := filepath.Join(files, "record.yaml")
	if err := os.WriteFile(reportPath, []byte("Completed report."), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(evidencePath, []byte("not: [valid"), 0644); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name string
		args []string
	}{
		{"missing report and evidence", reviewArgs(repo, repo.base, reviewConfigPath, "", "")},
		{"both report and evidence", reviewArgs(repo, repo.base, reviewConfigPath, reportPath, evidencePath)},
		{"missing config in snapshot", reviewArgs(repo, repo.base, "missing-review.yaml", reportPath, "")},
		{"unsafe config path", reviewArgs(repo, repo.base, "../markitect-review.yaml", reportPath, "")},
		{"provisional working tree", reviewArgs(repo, "", reviewConfigPath, reportPath, "")},
		{"misapplied impact flag", append(reviewArgs(repo, repo.base, reviewConfigPath, reportPath, ""), "--base", repo.base)},
		{"malformed evidence", reviewArgs(repo, repo.base, reviewConfigPath, "", evidencePath)},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code, _, _ := invoke(tt.args...)
			if code != 2 {
				t.Fatalf("run(%v) = %d, want invalid-input exit 2", tt.args, code)
			}
		})
	}
}

func TestReviewRejectsUnknownConfigFieldsFromSnapshot(t *testing.T) {
	repo := newReviewCLIRepo(t)
	badConfig := append(reviewConfigData(t, "Valid question.", true), []byte("unexpected: true\n")...)
	writeRepoFile(t, repo.root, "bad-review.yaml", badConfig)
	git(t, repo.root, "add", "bad-review.yaml")
	git(t, repo.root, "commit", "-m", "add malformed review config")
	revision := git(t, repo.root, "rev-parse", "HEAD")
	files := t.TempDir()
	reportPath := filepath.Join(files, "report.md")
	if err := os.WriteFile(reportPath, []byte("Review report."), 0644); err != nil {
		t.Fatal(err)
	}
	code, _, _ := invoke(reviewArgs(repo, revision, "bad-review.yaml", reportPath, "")...)
	if code != 2 {
		t.Fatalf("review with unknown config field returned %d, want 2", code)
	}
}

func TestReviewEvidenceMustBeValidYAMLAndEntryBound(t *testing.T) {
	repo := newReviewCLIRepo(t)
	files := t.TempDir()
	reportPath := filepath.Join(files, "report.md")
	if err := os.WriteFile(reportPath, []byte("Review report."), 0644); err != nil {
		t.Fatal(err)
	}
	code, recordYAML, stderr := invoke(reviewArgs(repo, repo.base, reviewConfigPath, reportPath, "")...)
	if code != 0 {
		t.Fatalf("record review exit=%d stderr=%s", code, stderr)
	}
	var evidence map[string]any
	if err := yaml.Unmarshal([]byte(recordYAML), &evidence); err != nil {
		t.Fatal(err)
	}
	evidence["entry"] = "sample/Skill/another"
	wrongEntry, err := yaml.Marshal(evidence)
	if err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(files, "wrong-entry.yaml")
	if err := os.WriteFile(evidencePath, wrongEntry, 0644); err != nil {
		t.Fatal(err)
	}
	code, _, _ = invoke(reviewArgs(repo, repo.base, reviewConfigPath, "", evidencePath)...)
	if code != 2 {
		t.Fatalf("review accepted evidence for another entry: exit %d", code)
	}
}

func TestReviewCLIRequiresFullGitIdentityForStoredEvidence(t *testing.T) {
	repo := newReviewCLIRepo(t)
	project, err := app.Load(repo.root, repo.base)
	if err != nil {
		t.Fatal(err)
	}
	config, err := app.DecodeReviewConfig(project.Snapshot.Files[reviewConfigPath])
	if err != nil {
		t.Fatal(err)
	}
	record, err := app.RecordReview(project, cliNamespace+"/Skill/entry", version, "test-tool", config, "Advisory fixture report.")
	if err != nil {
		t.Fatal(err)
	}
	evidencePath := filepath.Join(t.TempDir(), "record.yaml")
	for _, id := range []string{"HEAD", "deadbeef", "fixture:base", strings.Repeat("z", 40)} {
		t.Run(id, func(t *testing.T) {
			candidate := *record
			candidate.Revision = id
			data, err := app.YAML(candidate)
			if err != nil {
				t.Fatal(err)
			}
			writeReviewRecord(t, evidencePath, string(data))
			code, _, stderr := invoke(reviewArgs(repo, repo.base, reviewConfigPath, "", evidencePath)...)
			if code != 2 || !strings.Contains(stderr, "hexadecimal commit id") {
				t.Fatalf("non-Git evidence identity %q: code=%d stderr=%s", id, code, stderr)
			}
		})
	}
}
