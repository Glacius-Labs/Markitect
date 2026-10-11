package host

import (
	"encoding/json"
	"github.com/Glacius-Labs/Markitect/src/internal/host/compat/v0_13/consumers/projections"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// Replays fixed candidate bytes without model calls. Independently owned checks
// are copied unchanged; agent-authored tests never supply the sole proof.
func TestProjectionGreenfieldCandidatesConvergeAgainstIndependentEvidence(t *testing.T) {
	if _, err := exec.LookPath("python"); err != nil {
		t.Skip("fixture requires Python 3 on PATH")
	}
	example := filepath.Join("..", "..", "..", "examples", "projection-first")
	snap, err := source.Load(example, "")
	if err != nil {
		t.Fatal(err)
	}
	config, err := projections.ParseConfig(snap.Files["projections.config"])
	if err != nil {
		t.Fatal(err)
	}
	targets := map[string]bool{}
	for _, contract := range config.Contracts {
		for _, target := range contract.Targets {
			targets[target.Path] = true
		}
	}
	for _, variant := range []string{"a", "b"} {
		t.Run(variant, func(t *testing.T) {
			root := t.TempDir()
			for name, data := range snap.Files {
				if targets[name] {
					continue
				}
				destination := filepath.Join(root, filepath.FromSlash(name))
				if err := os.MkdirAll(filepath.Dir(destination), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(destination, data, 0644); err != nil {
					t.Fatal(err)
				}
			}
			p, err := Load(root, "")
			if err != nil {
				t.Fatal(err)
			}
			first, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "fixed-tool")
			if err != nil {
				t.Fatal(err)
			}
			repeat, err := PlanRepresentations(p, "projections.config", "markitect-artifacts.yaml", "test", "fixed-tool")
			if err != nil || !yamlEqual(first, repeat) {
				t.Fatalf("non-deterministic seed plan: %v", err)
			}
			raw, err := os.ReadFile(filepath.Join("..", "..", "..", "experiments", "projection-first", "candidate-"+variant+".json"))
			if err != nil {
				t.Fatal(err)
			}
			var response struct {
				Files       map[string]string `json:"files"`
				Explanation string            `json:"explanation"`
			}
			if err := json.Unmarshal(raw, &response); err != nil {
				t.Fatal(err)
			}
			candidate := Materialization{APIVersion: MaterializationVersion, PlanDigest: first.PlanDigest, Files: response.Files}
			applied, err := ApplyRepresentations(root, p, "projections.config", "markitect-artifacts.yaml", "test", "fixed-tool", first, reviewedMaterialization(t, candidate), materializationReviewDigest(t, candidate))
			if err != nil {
				t.Fatal(err)
			}
			if applied.Status != "materialized-unverified" || len(applied.Written) != 20 {
				t.Fatalf("wrong greenfield materialization: %#v", applied)
			}
			git := func(args ...string) string {
				t.Helper()
				cmd := exec.Command("git", append([]string{"-C", root}, args...)...)
				cmd.Env = append(os.Environ(), "GIT_CONFIG_NOSYSTEM=1")
				out, err := cmd.CombinedOutput()
				if err != nil {
					t.Fatalf("git %v: %v %s", args, err, out)
				}
				return strings.TrimSpace(string(out))
			}
			git("init", "-b", "codex/projection-example")
			git("config", "user.name", "Synthetic proof")
			git("config", "user.email", "proof@example.invalid")
			git("config", "core.autocrlf", "false")
			git("add", ".")
			git("commit", "-m", "Materialized finite projection")
			fixed, err := Load(root, git("rev-parse", "HEAD"))
			if err != nil {
				t.Fatal(err)
			}
			verified, err := VerifyRepresentations(fixed, "projections.config", "markitect-artifacts.yaml", "test", "fixed-tool")
			if err != nil || verified.Status != "converged" {
				t.Fatalf("candidate failed independent evidence: %v %#v", err, verified)
			}
			if !strings.Contains(verified.VerificationBoundary, "not proved") {
				t.Fatal("fixed check results concealed assurance limits")
			}
			if len(verified.Checks) != 3 {
				t.Fatalf("not all declared independent/project checks ran: %#v", verified.Checks)
			}
			// A successful immutable Verify must not mutate the observed working tree.
			if dirty := git("status", "--porcelain"); dirty != "" {
				t.Fatalf("Verify mutated source: %s", dirty)
			}
			plan, err := PlanRepresentations(fixed, "projections.config", "markitect-artifacts.yaml", "test", "fixed-tool")
			if err != nil {
				t.Fatal(err)
			}
			if plan.Status == "converged" {
				t.Fatal("AI presence without check evidence claimed convergence")
			}
		})
	}
}
