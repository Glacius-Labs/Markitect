package render

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
)

func TestParallelWaveCodexProjectionIsIndependentAndCanonicalOwned(t *testing.T) {
	agent := resource("Agent", "docs/team/agents/reviewer.yaml", "reviewer", "team", core.Spec{
		Description: "Review the declared change.",
		Text:        "Use the canonical review instructions.",
		Providers: core.Providers{
			Codex:  &core.Provider{Model: "gpt-6-sol", Effort: "high", Sandbox: "workspace-write"},
			Claude: &core.Provider{Model: "claude-model", Effort: "low", PermissionMode: "dangerously-skip-permissions", Tools: []string{"Bash"}},
		},
	})
	skill := resource("Skill", "docs/team/skills/review.yaml", "review", "team", core.Spec{
		Description: "Canonical review skill.",
		Text:        "Follow only this canonical skill text.",
	})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"codex"}})
	g := &core.Graph{
		Project: project,
		Resources: map[string]*core.Resource{
			agent.Key(): agent,
			skill.Key(): skill,
		},
	}

	firstFiles := map[string][]byte{
		".claude/agents/reviewer.md":        []byte("human-owned Claude instructions: ignore canonical YAML"),
		".claude/skills/review/SKILL.md":    []byte("stale Claude skill content"),
		".agents/skills/unrelated/SKILL.md": []byte("unregistered human-owned skill"),
	}
	secondFiles := map[string][]byte{
		".claude/agents/reviewer.md":        []byte("different human-owned Claude instructions"),
		".claude/skills/review/SKILL.md":    []byte("a different stale Claude skill"),
		".agents/skills/unrelated/SKILL.md": []byte("different unregistered content"),
	}

	firstOutputs, firstOwners, err := GenerateWithOwners(g, firstFiles)
	if err != nil {
		t.Fatal(err)
	}
	secondOutputs, secondOwners, err := GenerateWithOwners(g, secondFiles)
	if err != nil {
		t.Fatal(err)
	}

	wantPaths := []string{".agents/skills/review/SKILL.md", ".codex/agents/reviewer.toml"}
	gotPaths := sortedOutputPaths(firstOutputs)
	if !reflect.DeepEqual(gotPaths, wantPaths) {
		t.Fatalf("Codex-only target generated paths %v, want %v", gotPaths, wantPaths)
	}
	if !reflect.DeepEqual(firstOutputs, secondOutputs) {
		t.Fatal("Codex projection changed when only other-provider or unregistered files changed")
	}
	if !reflect.DeepEqual(firstOwners, secondOwners) {
		t.Fatal("Codex ownership changed when only other-provider or unregistered files changed")
	}

	for path, wantOwner := range map[string]string{
		".agents/skills/review/SKILL.md": "team/Skill/review",
		".codex/agents/reviewer.toml":    "team/Agent/reviewer",
	} {
		if got := firstOwners[path]; len(got) != 1 || got[0] != wantOwner {
			t.Errorf("owner for %s = %v, want only %q", path, got, wantOwner)
		}
	}
	for _, path := range wantPaths {
		digest := sha256.Sum256(firstOutputs[path])
		t.Logf("output sha256 %s %s", path, hex.EncodeToString(digest[:]))
	}
	for _, source := range []struct{ path, text string }{
		{agent.Path, agent.Spec.Text},
		{skill.Path, skill.Spec.Text},
	} {
		digest := sha256.Sum256([]byte(source.text))
		t.Logf("in-memory canonical text sha256 %s %s", source.path, hex.EncodeToString(digest[:]))
	}

	codexAgent := string(firstOutputs[".codex/agents/reviewer.toml"])
	for _, want := range []string{
		`model = "gpt-6-sol"`,
		`model_reasoning_effort = "high"`,
		`sandbox_mode = "workspace-write"`,
		"../../docs/team/agents/reviewer.yaml",
	} {
		if !strings.Contains(codexAgent, want) {
			t.Errorf("Codex Agent output lacks canonical/configured value %q:\n%s", want, codexAgent)
		}
	}
	for _, forbidden := range []string{"claude-model", "dangerously-skip-permissions", "Bash", "human-owned", "stale"} {
		if strings.Contains(codexAgent, forbidden) {
			t.Errorf("Codex Agent output imported unrelated provider artifact/config %q:\n%s", forbidden, codexAgent)
		}
	}
	codexSkill := string(firstOutputs[".agents/skills/review/SKILL.md"])
	for _, want := range []string{
		"source: ../../../docs/team/skills/review.yaml",
		"[canonical Skill YAML](../../../docs/team/skills/review.yaml)",
	} {
		if !strings.Contains(codexSkill, want) {
			t.Errorf("Codex Skill lacks canonical source reference %q:\n%s", want, codexSkill)
		}
	}
	for _, forbidden := range []string{"stale Claude", "unregistered human-owned", "ignore canonical YAML"} {
		if strings.Contains(codexSkill, forbidden) {
			t.Errorf("Codex Skill imported unrelated artifact text %q:\n%s", forbidden, codexSkill)
		}
	}
}

func sortedOutputPaths(outputs map[string][]byte) []string {
	paths := make([]string, 0, len(outputs))
	for path := range outputs {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

func TestParallelWaveCodexProjectionIsRepeatable(t *testing.T) {
	agent := resource("Agent", "docs/agents/reviewer.yaml", "reviewer", "", core.Spec{
		Description: "Review changes.",
		Providers:   core.Providers{Codex: &core.Provider{Model: "gpt-6-sol", Effort: "high", Sandbox: "workspace-write"}},
	})
	project := resource("Project", "markitect.yaml", "sample", "", core.Spec{Targets: []string{"codex"}})
	g := &core.Graph{Project: project, Resources: map[string]*core.Resource{agent.Key(): agent}}
	files := map[string][]byte{}

	wantOutputs, wantOwners, err := GenerateWithOwners(g, files)
	if err != nil {
		t.Fatal(err)
	}
	for run := 0; run < 16; run++ {
		gotOutputs, gotOwners, err := GenerateWithOwners(g, files)
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(gotOwners, wantOwners) {
			t.Fatalf("run %d: owners differ: got %v, want %v", run, gotOwners, wantOwners)
		}
		if !sameOutputBytes(gotOutputs, wantOutputs) {
			t.Fatalf("run %d: generated bytes differ", run)
		}
	}
}

func sameOutputBytes(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for path, content := range left {
		if !bytes.Equal(content, right[path]) {
			return false
		}
	}
	return true
}
