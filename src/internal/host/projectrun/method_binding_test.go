package projectrun

import (
	"path/filepath"
	"testing"

	"github.com/Glacius-Labs/Markitect/src/internal/host/agentexec"
	"github.com/Glacius-Labs/Markitect/src/internal/host/projectconfig"
)

func TestMethodPolicyDigestExcludesExecutorIdentityAndRetainsPolicy(t *testing.T) {
	base := validRuntime()
	baseAgent := base.Agents["commerce"]
	baseAgent.Command = `C:\agents\manager-a.exe`
	baseAgent.Args = []string{"--wrapper", "one"}
	baseAgent.RuntimeFiles = []agentexec.RuntimeFile{{Path: `C:\agents\wrapper.py`, Mode: "0644", Digest: "sha256:one"}}
	baseAgent.Environment = []string{"MARKITECT_MODE", "MARKITECT_REGION"}
	base.Agents["commerce"] = baseAgent
	base.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{"commerce": baseAgent}, MaxRounds: 1, MaxManagerRounds: 2}

	baseDigest, err := methodPolicyDigest(base)
	if err != nil {
		t.Fatal(err)
	}

	changedExecutor := base
	changedAgent := baseAgent
	changedAgent.Command = `D:\tools\manager-b.exe`
	changedAgent.Args = []string{"--different-wrapper", "two"}
	changedAgent.RuntimeFiles = []agentexec.RuntimeFile{{Path: `D:\tools\wrapper.py`, Mode: "0600", Digest: "sha256:two"}}
	changedAgent.Transport = projectconfig.TransportCodexAppServer
	changedAgent.Environment = []string{"MARKITECT_REGION", "MARKITECT_MODE"}
	changedExecutor.Agents = map[string]projectconfig.Agent{"commerce": changedAgent}
	changedExecutor.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{"commerce": changedAgent}, MaxRounds: 1, MaxManagerRounds: 2}
	changedDigest, err := methodPolicyDigest(changedExecutor)
	if err != nil {
		t.Fatal(err)
	}
	if changedDigest != baseDigest {
		t.Fatalf("executor-only changes altered method policy digest: got %s want %s", changedDigest, baseDigest)
	}

	for name, mutate := range map[string]func(*projectconfig.Runtime){
		"model": func(runtime *projectconfig.Runtime) {
			agent := runtime.Agents["commerce"]
			agent.Model = "different-model"
			runtime.Agents["commerce"] = agent
		},
		"role options": func(runtime *projectconfig.Runtime) {
			agent := runtime.Agents["commerce"]
			agent.ModelOptions = map[string]any{"temperature": 0.2}
			runtime.Agents["commerce"] = agent
		},
		"workspace authority": func(runtime *projectconfig.Runtime) {
			agent := runtime.Agents["commerce"]
			agent.WorkspaceMode = "scoped"
			agent.InstructionPaths = []string{"AGENTS.md"}
			runtime.Agents["commerce"] = agent
		},
		"environment policy": func(runtime *projectconfig.Runtime) {
			agent := runtime.Agents["commerce"]
			agent.Environment = append(agent.Environment, "MARKITECT_EXTRA")
			runtime.Agents["commerce"] = agent
		},
		"finite limits":     func(runtime *projectconfig.Runtime) { runtime.Limits.MaxStarts++ },
		"review strictness": func(runtime *projectconfig.Runtime) { runtime.Review.MaxRounds++ },
		"strictness profile": func(runtime *projectconfig.Runtime) {
			runtime.Strictness = &projectconfig.StrictnessConfig{Default: projectconfig.StrictnessProfile{Counterexamples: 2}}
		},
		"output bounds": func(runtime *projectconfig.Runtime) {
			agent := runtime.Agents["commerce"]
			agent.MaxStdoutBytes++
			runtime.Agents["commerce"] = agent
		},
	} {
		t.Run(name, func(t *testing.T) {
			changed := base
			changed.Agents = map[string]projectconfig.Agent{"commerce": baseAgent}
			changed.Review = &projectconfig.ReviewConfig{Agents: map[string]projectconfig.Agent{"commerce": baseAgent}, MaxRounds: 1, MaxManagerRounds: 2}
			mutate(&changed)
			got, err := methodPolicyDigest(changed)
			if err != nil {
				t.Fatal(err)
			}
			if got == baseDigest {
				t.Fatal("policy change did not alter method policy digest")
			}
		})
	}
}

func TestRequireCurrentRuntimeBindingFailsClosedForOldAndUnknownPlans(t *testing.T) {
	for _, version := range []int{0, 1, methodRuntimeBindingVersion + 1} {
		if err := requireCurrentRuntimeBinding(PlanRecord{RuntimeBindingVersion: version}); err != ErrStale {
			t.Errorf("binding version %d returned %v, want ErrStale", version, err)
		}
	}
	if err := requireCurrentRuntimeBinding(PlanRecord{RuntimeBindingVersion: methodRuntimeBindingVersion}); err != nil {
		t.Fatalf("current binding version rejected: %v", err)
	}
}

func TestPlanDoesNotResolveExecutorAndBindsOnlyMethodPolicy(t *testing.T) {
	root := makeProjectRunFixture(t)
	missingA := filepath.Join(t.TempDir(), "executor-a-that-does-not-exist")
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		for id, agent := range runtime.Agents {
			agent.Command = missingA
			agent.Args = []string{"--before"}
			runtime.Agents[id] = agent
		}
		if runtime.Verifier != nil {
			runtime.Verifier.Command = missingA
		}
		if runtime.Review != nil {
			for id, agent := range runtime.Review.Agents {
				agent.Command = missingA
				runtime.Review.Agents[id] = agent
			}
		}
	})

	revision := identityHead(t, root)
	project := processRuntimeProject(t, root, revision, false)
	host := processRuntimeHost(root, project)
	plan, err := Plan(host, root, revision, PlanRequest{Goal: "Plan with an unavailable executor."})
	if err != nil {
		t.Fatalf("Plan resolved or required an available executor: %v", err)
	}
	if plan.RuntimeBindingVersion != methodRuntimeBindingVersion {
		t.Fatalf("runtime binding version = %d, want %d", plan.RuntimeBindingVersion, methodRuntimeBindingVersion)
	}
	if len(plan.RuntimeAgents) != 0 {
		t.Fatalf("new plan persisted executor fingerprints: %v", plan.RuntimeAgents)
	}

	runtime, err := projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	if digest, err := methodPolicyDigest(runtime); err != nil || digest != plan.RuntimeDigest {
		t.Fatalf("method policy digest = %q, want plan digest %q, err=%v", digest, plan.RuntimeDigest, err)
	}

	// Moving to a different currently-missing executable does not stale the
	// plan; changing an actual role policy still does.
	missingB := filepath.Join(t.TempDir(), "executor-b-that-does-not-exist")
	updateE2ERuntime(t, root, func(runtime *projectconfig.Runtime) {
		for id, agent := range runtime.Agents {
			agent.Command = missingB
			agent.Args = []string{"--after"}
			runtime.Agents[id] = agent
		}
	})
	runtime, err = projectconfig.LoadRuntime(root)
	if err != nil {
		t.Fatal(err)
	}
	if digest, err := runtimeDigestWithInvoker(nil, runtime); err != nil || digest != plan.RuntimeDigest {
		t.Fatalf("executor change staled method policy: got %q want %q err=%v", digest, plan.RuntimeDigest, err)
	}
	changed := runtime.Agents[e2eManagerID("", "project-owner")]
	changed.Model = "changed-model"
	runtime.Agents[e2eManagerID("", "project-owner")] = changed
	if digest, err := methodPolicyDigest(runtime); err != nil || digest == plan.RuntimeDigest {
		t.Fatalf("actual method policy change did not stale digest: got %q want != %q err=%v", digest, plan.RuntimeDigest, err)
	}
}
