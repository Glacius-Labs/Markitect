# Plugin variant (lane 2)

The plugin variant runs Markitect's Apply method as a native Claude Code plugin instead of the host runtime. It keeps the canonical model, the compiler, the registry and file accounting. It replaces the host's Run and Apply runtime with Claude Code workflow agents in git worktrees, hooks and a host-side publication guard. Its question is whether a much thinner runtime keeps the host variant's delivered quality and loses its runtime failures ([comparison report](../validation/v1-comparison-report-20261026.md)). It is an evaluation variant (arm P), not a product path: V1 ships Codex only ([D18](../concepts/v1-concept.md#23-provider-scope-d18)).

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. Its lead is the session "Lane 2: Plugin variant lead", which may start sub-sessions for the lane's work. This page holds only the lane's scope, a short state and its open questions. The [arm P README](https://github.com/Glacius-Labs/Markitect/blob/variant/plugin/tools/comparison/claude-plugin/README.md) on the variant branch owns the design, its deviations from concept 1.0, the cheap checks and the tests.

## Scope

- **Owns:**
  - The long-lived branch `variant/plugin`, which holds only variant code under `tools/comparison/` and merges `main` regularly. The code is the plugin (`claude-plugin/plugin/`: command, workflow, Manager and verifier agents, hooks, host steps), its runner `armp.py`, the workflow simulator and their tests.
  - The plugin variant's dispatches and their results.
- **Shared, on `main`:** `markitect tree`, the model-derived Manager tree the workflow walks ([PR #802](https://github.com/Glacius-Labs/Markitect/pull/802)). The compiler topic keeps its output shape stable or agrees changes with this lane.
- **Uses, does not own:** the compiler, the model and the Libraries, and the evaluation suite with its [variant interface](https://github.com/Glacius-Labs/Markitect/pull/793).

## State

- **Built and tested offline.** The plugin, its runner and the 69 offline tests of the comparison suite pass. These include the workflow through scripted agents and the publication guard. An independent review's findings are fixed. One-off cheap checks in Claude Code with Haiku showed that the workflow, hooks, worktree agents and host records work headless; no Apply has run.
- **Evaluation interface:** the suite treats every variant as a black box: one command `VARIANT TASK_DIR OUT_DIR [STATE_DIR]` that writes `result.bundle` and may write `run.json` ([PR #793](https://github.com/Glacius-Labs/Markitect/pull/793)). `armp.py` does not implement it yet; it takes arm M's bundles directly.
- **Provider:** P runs only in Claude Code, because the workflow, hooks and plugin agents are Claude Code features. The suite records the provider as part of a variant's configuration; its current runs use Claude.
- **Sub-sessions:** none running.

## Next

1. **Library smoke chain:** S1 to S4 on the Library from arm M's inputs, one attempt each, Sonnet 5.5 at effort high, 2,700 s and 150 agent calls per Apply. The [dispatch](https://github.com/Glacius-Labs/Markitect/blob/variant/plugin/tools/comparison/claude-plugin/smoke-dispatch.md) is prepared and waits for the owner's approval. It is feasibility evidence for one chain, not a comparison.
2. **The variant command:** wrap `armp.py` as `VARIANT TASK_DIR OUT_DIR [STATE_DIR]`. It reads `model/`, `start.bundle` and `limits.json`, writes `result.bundle` and a `run.json` with provider, model, tokens, status and cause, and keeps each run's records in `OUT_DIR`. Then `armp.py input` and the `analyze.py --arms` change go, because the suite builds chain starts and reports. This waits for the suite runner on `main`.
3. **Then, in the suite:** Library repetitions, then the Makerspace chains. These are the decisive test, because both native arms conformed on the Library.

## Open questions

1. **Approving the smoke dispatch.** Should the Library smoke chain run? It uses the owner's Claude subscription; its token use is unknown and recorded per run. The recommendation is yes, now, because it shows whether the plugin completes an Apply at all, and the variant command is blocked until the suite runner lands anyway.
2. **Host or Linux runner.** P's agents run as host processes in git worktrees. Its own deadline kills the process tree, but CPU, memory and processes are not bounded, and a run is separation, not a sandbox. Running P inside the Linux runner would bound them and match [DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution), at the cost of Claude Code and the login in the runner image. The recommendation is the host for the smoke chain and the runner before the suite's comparison runs.
3. **What a good result would change.** If P matches the host variant's delivered quality with fewer runtime failures, the host runtime packages (about 12,000 to 14,000 lines) become candidates for replacement. The recommendation is to decide nothing about the runtime before P has suite results on the Makerspace chains.
