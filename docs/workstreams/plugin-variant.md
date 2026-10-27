# Plugin variant (lane 2)

The plugin variant runs Markitect's Apply method as a native Claude Code plugin instead of the host runtime. It keeps the canonical model, the compiler, the registry and file accounting. It replaces the host's Run and Apply runtime with Claude Code workflow agents in git worktrees, hooks and a host-side publication guard. Its question is whether a much thinner runtime keeps the host variant's delivered quality and loses its runtime failures ([comparison report](../validation/v1-comparison-report-20261026.md)). It is an evaluation arm (arm P), not a product path: V1 ships Codex only ([D18](../concepts/v1-concept.md#23-provider-scope-d18)).

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. Its lead is the session "Lane 2: Plugin variant lead", which may start sub-sessions for the lane's work. This page holds only the lane's scope, a short state and its open questions; the [arm P README](https://github.com/Glacius-Labs/Markitect/blob/variant/plugin/tools/comparison/claude-plugin/README.md) on the variant branch owns the design, its deviations from concept 1.0, the cheap-check results and the tests.

## Scope

- **Owns:**
  - The long-lived branch `variant/plugin`, which holds only variant code under `tools/comparison/`: the plugin (`claude-plugin/plugin/`: command, workflow, Manager and verifier agents, hooks, host steps), its runner `armp.py`, the workflow simulator and their tests. The branch merges `main` regularly.
  - The plugin arm's dispatches and their results.
- **Shared, lands on `main` through normal PRs:**
  - `markitect tree`, the model-derived Manager tree the workflow walks ([PR #802](https://github.com/Glacius-Labs/Markitect/pull/802)). The compiler lane is informed.
- **Uses, does not own:** the compiler, the model and the Libraries; the evaluation suite, whose [common arm interface](https://github.com/Glacius-Labs/Markitect/pull/793) the plugin arm implements.

## State

- **Built and tested offline.** The plugin, its runner and 69 offline tests of the comparison suite pass, including the workflow through scripted agents and the host's publication guard. An independent review's findings are fixed. No Apply has run.
- **Cheap checks, 27 October:** in Claude Code 2.1.309 with Haiku, the Workflow tool runs headless; `SubagentStop` and `PreToolUse` fire for workflow agents with the agent's own worktree; parallel worktree agents branch and commit; 20 parallel agents returned; the record hook reads a verifier's answer and checked-out commit from its own transcript.
- **Arm interface:** agreed with lane 4 in [PR #793](https://github.com/Glacius-Labs/Markitect/pull/793): the suite builds each chain step's start, the arm writes `run.json` with `reported` instead of an outcome, `armLimits` holds P's own limits, and every limit says whether it is enforced. `armp.py` does not implement it yet.
- **Provider:** P runs only in Claude Code. The owner decided on 27 October that a separate Claude suite holds P, M through the K1 Claude adapter and K as one Claude Code session.
- **Sub-sessions:** none running.

## Next

1. **Implement the arm interface:** `armp.py run --task TASK.json --out NEW_DIR` writing `run.json`, once lane 4's schemas and run checker are on `main`. Retire `armp.py input` and the `analyze.py --arms` change, which the shared ledger replaces.
2. **Library smoke chain:** S1 to S4 on the Library from arm M's inputs, one attempt each, Sonnet 5.5 at effort high, 2,700 s and 150 agent calls per Apply. The [dispatch](https://github.com/Glacius-Labs/Markitect/blob/variant/plugin/tools/comparison/claude-plugin/smoke-dispatch.md) is prepared and waits for the owner's approval. It is feasibility evidence for one chain, not a comparison.
3. **Then, in the Claude suite:** Library repetitions, then the Makerspace chains, the decisive test because both native arms conformed on the Library.

## Open questions

1. **Approving the smoke dispatch.** Should the Library smoke chain run? It uses the owner's Claude subscription; its token use is unknown and recorded per run. The recommendation is yes, before the interface work, because it shows whether the plugin completes an Apply at all.
2. **Run on the host or in the Linux runner.** P's agents run as host processes in git worktrees, so on Windows the suite's CPU, memory and process limits are not enforced and a run is separation, not a sandbox. Running P inside the Linux runner would make the limits binding and match [DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution), at the cost of installing Claude Code and the login in the runner image. The recommendation is the host for the smoke chain and the runner before any comparison suite.
3. **What a good result would change.** If P matches M's delivered quality with fewer runtime failures, the host runtime packages (about 12,000 to 14,000 lines) become candidates for replacement, but only for Claude. Whether that question is worth asking before the Claude suite has results is the owner's call.
