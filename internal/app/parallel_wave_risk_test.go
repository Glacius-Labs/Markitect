package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
	"go.yaml.in/yaml/v3"
)

const (
	parallelWaveAdapterEnv   = "MARKITECT_PARALLEL_WAVE_ADAPTER_HELPER"
	parallelWaveStateEnv     = "MARKITECT_PARALLEL_WAVE_ADAPTER_STATE"
	parallelWaveApplyMarkEnv = "MARKITECT_PARALLEL_WAVE_ADAPTER_APPLIED"
	parallelWaveAdapterName  = "markitect-parallel-wave-adapter"
	parallelWaveToolVersion  = "test-tool-v1"
	parallelWaveToolDigest   = "sha256:test-tool"
)

// The copied test binary acts as a deterministic command adapter in child
// processes. Exiting during init keeps the go test runner from writing text to
// the adapter's YAML response stream.
func init() {
	if os.Getenv(parallelWaveAdapterEnv) == "1" {
		parallelWaveAdapterProcess()
	}
}

func parallelWaveAdapterProcess() {
	var request AdapterRequest
	if err := yaml.NewDecoder(os.Stdin).Decode(&request); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	result := AdapterResult{
		APIVersion:  AdapterResultVersion,
		Adapter:     request.Adapter.Name,
		Action:      request.Action,
		Status:      "complete",
		ModelDigest: request.Model.ModelDigest,
		Target:      request.Adapter.Target,
	}
	switch request.Action {
	case "observe":
		state, err := os.ReadFile(os.Getenv(parallelWaveStateEnv))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		result.Observed = map[string]any{"generation": strings.TrimSpace(string(state))}
	case "plan":
		generation := ""
		if request.Observation != nil {
			generation, _ = request.Observation.Observed["generation"].(string)
		}
		result.Operations = []AdapterOperation{{
			ID: "write-generation", Action: "set", Target: request.Adapter.Target,
			Desired: map[string]any{"generation": generation},
		}}
	case "apply":
		if err := os.WriteFile(os.Getenv(parallelWaveApplyMarkEnv), []byte("called"), 0600); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
	case "verify":
	default:
		fmt.Fprintf(os.Stderr, "unsupported action %q", request.Action)
		os.Exit(2)
	}
	if err := yaml.NewEncoder(os.Stdout).Encode(result); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	os.Exit(0)
}

func TestParallelWaveCommandAdapterRejectsDriftAndAlteredPlanBeforeApply(t *testing.T) {
	t.Run("external observation changed after planning", func(t *testing.T) {
		p, statePath, applyMark := parallelWaveCommandAdapterProject(t)
		plan, err := PlanCommandAdapter(p, parallelWaveAdapterName, parallelWaveToolVersion, parallelWaveToolDigest)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(statePath, []byte("generation-2"), 0600); err != nil {
			t.Fatal(err)
		}
		if _, err = ApplyCommandAdapter(p, parallelWaveAdapterName, parallelWaveToolVersion, parallelWaveToolDigest, plan); err == nil || !strings.Contains(err.Error(), "external state changed since plan") {
			t.Fatalf("Apply error = %v, want stale external-state rejection", err)
		}
		parallelWaveAssertApplyNotCalled(t, applyMark)
	})

	t.Run("saved operations were altered", func(t *testing.T) {
		p, _, applyMark := parallelWaveCommandAdapterProject(t)
		plan, err := PlanCommandAdapter(p, parallelWaveAdapterName, parallelWaveToolVersion, parallelWaveToolDigest)
		if err != nil {
			t.Fatal(err)
		}
		plan.Result.Operations[0].Desired["generation"] = "forged"
		if _, err = ApplyCommandAdapter(p, parallelWaveAdapterName, parallelWaveToolVersion, parallelWaveToolDigest, plan); err == nil || !strings.Contains(err.Error(), "saved adapter operations differ") {
			t.Fatalf("Apply error = %v, want altered-operation rejection", err)
		}
		parallelWaveAssertApplyNotCalled(t, applyMark)
	})

	t.Run("adapter identity changed", func(t *testing.T) {
		p, _, applyMark := parallelWaveCommandAdapterProject(t)
		plan, err := PlanCommandAdapter(p, parallelWaveAdapterName, parallelWaveToolVersion, parallelWaveToolDigest)
		if err != nil {
			t.Fatal(err)
		}
		plan.Adapter.Target = "other-target"
		if _, err = ApplyCommandAdapter(p, parallelWaveAdapterName, parallelWaveToolVersion, parallelWaveToolDigest, plan); err == nil || !strings.Contains(err.Error(), "adapter identity") {
			t.Fatalf("Apply error = %v, want adapter-identity rejection", err)
		}
		parallelWaveAssertApplyNotCalled(t, applyMark)
	})
}

func TestParallelWaveCommandAdapterPlanReadRejectsAlteredObservation(t *testing.T) {
	p, _, _ := parallelWaveCommandAdapterProject(t)
	plan, err := PlanCommandAdapter(p, parallelWaveAdapterName, parallelWaveToolVersion, parallelWaveToolDigest)
	if err != nil {
		t.Fatal(err)
	}
	plan.Observation.Observed["generation"] = "forged"
	data, err := YAML(plan)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "adapter-plan.yaml")
	if err = os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	if _, err = ReadCommandAdapterPlan(path); err == nil || !strings.Contains(err.Error(), "captured observation") {
		t.Fatalf("read error = %v, want observation identity/digest rejection", err)
	}
}

func parallelWaveCommandAdapterProject(t *testing.T) (*Project, string, string) {
	t.Helper()
	root := t.TempDir()
	toolName := parallelWaveAdapterName
	if runtime.GOOS == "windows" {
		toolName += ".exe"
	}
	toolPath := filepath.Join(root, toolName)
	current, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	tool, err := os.ReadFile(current)
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(toolPath, tool, 0700); err != nil {
		t.Fatal(err)
	}
	pathValue := root + string(os.PathListSeparator) + os.Getenv("PATH")
	t.Setenv("PATH", pathValue)
	t.Setenv(parallelWaveAdapterEnv, "1")
	statePath := filepath.Join(root, "external-state")
	applyMark := filepath.Join(root, "apply-called")
	if err = os.WriteFile(statePath, []byte("generation-1"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv(parallelWaveStateEnv, statePath)
	t.Setenv(parallelWaveApplyMarkEnv, applyMark)

	project := projectResource(projectNS)
	project.Spec.Adapters = []core.AdapterConfig{{
		Name:    parallelWaveAdapterName,
		Type:    "command",
		Version: "fixture-v1",
		Config: map[string]any{
			"inputs":     []string{projectPath},
			"target":     "remote:test-target",
			"observe":    []string{parallelWaveAdapterName},
			"plan":       []string{parallelWaveAdapterName},
			"apply":      []string{parallelWaveAdapterName},
			"allowApply": true,
			"verify":     []string{parallelWaveAdapterName},
		},
	}}
	files := fixtureFiles(t, "fixed-snapshot", "Keep the owner source.", projectNS).Files
	files[projectPath] = encodeResource(t, project)
	modes := map[string]string{}
	for name := range files {
		modes[name] = "100644"
	}
	p, err := Parse(&snapshot.Snapshot{ID: "parallel-wave-adapter-snapshot", Files: files, Modes: modes})
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Diagnostics) != 0 {
		t.Fatalf("adapter fixture has diagnostics: %#v", p.Diagnostics)
	}
	return p, statePath, applyMark
}

func parallelWaveAssertApplyNotCalled(t *testing.T, marker string) {
	t.Helper()
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatalf("apply marker stat error = %v, want no apply invocation", err)
	}
}
