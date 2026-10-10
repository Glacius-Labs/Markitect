package cli

import (
	"bytes"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host"
	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
	core "github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/kernel"
)

type findCLIResult struct {
	Version        string           `yaml:"version"`
	ToolDigest     string           `yaml:"toolDigest"`
	Revision       string           `yaml:"revision"`
	Provisional    bool             `yaml:"provisional"`
	SnapshotDigest string           `yaml:"snapshotDigest"`
	Result         []host.FindMatch `yaml:"result"`
}

type explainCLIResult struct {
	Version        string             `yaml:"version"`
	ToolDigest     string             `yaml:"toolDigest"`
	Revision       string             `yaml:"revision"`
	Provisional    bool               `yaml:"provisional"`
	SnapshotDigest string             `yaml:"snapshotDigest"`
	Result         host.ExplainResult `yaml:"result"`
}

func TestFindIsDeterministicAndAppliesExactFilters(t *testing.T) {
	repo := newCLIRepo(t, false)
	args := []string{"find", "--repo", repo.root, "--revision", repo.base, "--query", "POLICY", "--kind", "Rule", "--namespace", cliNamespace}
	code, output, stderr := invoke(args...)
	if code != 0 {
		t.Fatalf("find exit=%d stderr=%s output=%s", code, stderr, output)
	}
	envelope := decodeYAML[findCLIResult](t, output)
	if envelope.Revision != repo.base || envelope.Provisional || envelope.Version != version || envelope.ToolDigest == "" || envelope.SnapshotDigest == "" {
		t.Fatalf("find omitted fixed snapshot provenance: %#v", envelope)
	}
	matches := envelope.Result
	if len(matches) != 1 {
		t.Fatalf("find returned %d matches: %#v", len(matches), matches)
	}
	wantKey := cliNamespace + "/Rule/policy"
	if matches[0].Key != wantKey || matches[0].Kind != "Rule" || matches[0].Namespace != cliNamespace || matches[0].Path != "docs/general/rules/policy.yaml" {
		t.Fatalf("find returned wrong filtered match: %#v", matches[0])
	}
	code2, output2, stderr2 := invoke(args...)
	if code2 != 0 || stderr2 != "" || !bytes.Equal([]byte(output), []byte(output2)) {
		t.Fatalf("repeated find was not deterministic: first=(%d,%q,%q), second=(%d,%q,%q)", code, output, stderr, code2, output2, stderr2)
	}
}

func TestFindReadsRequestedImmutableRevisionAfterWorkingEdit(t *testing.T) {
	repo := newCLIRepo(t, false)
	updated := authoring.Resource{Core: authoring.Core{APIVersion: core.APIVersion, Kind: "Rule", Metadata: core.Metadata{Name: "policy", Namespace: cliNamespace}}, Spec: authoring.Spec{Text: "Only the changed revision contains current-only token."}}
	data, err := authoring.Encode(updated)
	if err != nil {
		t.Fatal(err)
	}
	writeRepoFile(t, repo.root, "docs/general/rules/policy.yaml", data)

	code, output, stderr := invoke("find", "--repo", repo.root, "--revision", repo.base, "--query", "current-only")
	if code != 0 {
		t.Fatalf("fixed find exit=%d stderr=%s output=%s", code, stderr, output)
	}
	if result := decodeYAML[findCLIResult](t, output); result.Revision != repo.base || result.Provisional || result.SnapshotDigest == "" || len(result.Result) != 0 {
		t.Fatalf("fixed revision saw a working-tree-only edit or omitted provenance: %#v", result)
	}
	code, output, stderr = invoke("find", "--repo", repo.root, "--query", "current-only")
	if code != 0 {
		t.Fatalf("working find exit=%d stderr=%s output=%s", code, stderr, output)
	}
	working := decodeYAML[findCLIResult](t, output)
	matches := working.Result
	if !working.Provisional || working.Version != version || working.ToolDigest == "" || working.SnapshotDigest == "" {
		t.Fatalf("working find omitted provisional snapshot provenance: %#v", working)
	}
	if len(matches) != 1 || matches[0].Key != cliNamespace+"/Rule/policy" {
		t.Fatalf("working revision did not see its edit: %#v", matches)
	}
}

func TestExplainResolvesExactNamespacedAndProjectIdentity(t *testing.T) {
	repo := newCLIRepo(t, false)
	code, output, stderr := invoke("explain", "--repo", repo.root, "--revision", repo.base, "--kind", "Rule", "--name", "policy", "--namespace", cliNamespace)
	if code != 0 {
		t.Fatalf("explain Rule exit=%d stderr=%s output=%s", code, stderr, output)
	}
	ruleEnvelope := decodeYAML[explainCLIResult](t, output)
	if ruleEnvelope.Revision != repo.base || ruleEnvelope.Provisional || ruleEnvelope.SnapshotDigest == "" {
		t.Fatalf("Rule explanation omitted fixed snapshot provenance: %#v", ruleEnvelope)
	}
	rule := ruleEnvelope.Result
	if rule.Key != cliNamespace+"/Rule/policy" || len(rule.Incoming) != 1 || rule.Incoming[0].From != cliNamespace+"/Skill/entry" {
		t.Fatalf("Rule explanation omitted exact identity or incoming reference: %#v", rule)
	}
	code, output, stderr = invoke("explain", "--repo", repo.root, "--revision", repo.base, "--kind", "Project", "--name", "sample-project")
	if code != 0 {
		t.Fatalf("explain Project exit=%d stderr=%s output=%s", code, stderr, output)
	}
	projectEnvelope := decodeYAML[explainCLIResult](t, output)
	project := projectEnvelope.Result
	if project.Key != "/Project/sample-project" || project.Kind != "Project" || project.Namespace != "" {
		t.Fatalf("Project explanation has wrong identity: %#v", project)
	}
}

func TestFindAndExplainRejectInvalidInputAndUnknownIdentity(t *testing.T) {
	repo := newCLIRepo(t, false)
	for _, args := range [][]string{
		{"find", "--repo", repo.root, "--revision", repo.base, "--query", "policy", "unexpected"},
		{"find", "--repo", repo.root, "--revision", repo.base, "--base", repo.base},
		{"explain", "--repo", repo.root, "--revision", repo.base, "--kind", "Rule", "--name", "policy"},
		{"explain", "--repo", repo.root, "--revision", repo.base, "--kind", "Project", "--name", "sample-project", "--namespace", cliNamespace},
	} {
		code, _, _ := invoke(args...)
		if code != 2 {
			t.Errorf("invoke(%v) = %d, want invalid-input exit 2", args, code)
		}
	}
	code, _, _ := invoke("explain", "--repo", repo.root, "--revision", repo.base, "--kind", "Skill", "--name", "absent", "--namespace", cliNamespace)
	if code != 2 {
		t.Fatalf("unknown exact identity exit=%d, want 2", code)
	}
}
