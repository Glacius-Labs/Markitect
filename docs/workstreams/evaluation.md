# Evaluation suite (lane 4)

The evaluation suite answers one question: which way of developing a project performs better, and in which areas? It works in three steps:

1. **Task:** what we want to build.
2. **Variant:** a solution is plugged in and builds it.
3. **Evaluation:** the result is judged against fixed criteria.

The suite treats every variant as a black box. It does not care how a variant works, which agents, tools or provider it uses, or whether it runs on Claude or Codex. Every evaluation can run with any provider; the provider is part of the variant's configuration, and the suite only records it. Since Codex is out of tokens, the current runs use Claude.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026. Its lead is the "Lane 4: Evaluation suite" session.

## Scope

- **Owns:**
  - The task format, the variant interface, the criteria and the report.
  - [`tools/comparison`](../../tools/comparison/README.md): the assessment ([`assess_candidate.py`](../../tools/comparison/assess_candidate.py)), the review rubric and prompts, and the analysis ([`analyze.py`](../../tools/comparison/analyze.py)).
  - The task portfolio.
  - The baseline variant, classic agentic coding: one coding agent with the task. Today's runner is [`runner.py`](../../tools/comparison/runner.py).
- **Uses, does not own:**
  - The hidden oracles in [`tools/library-oracle`](../../tools/library-oracle/README.md) and [`tools/makerspace-oracle`](../../tools/makerspace-oracle/README.md).
  - The variants. Each lane owns its own variant and how it meets the interface.
- **Variants today:**

  | Variant | Lane |
  |---|---|
  | Markitect with the host as coordinator | 1 |
  | Markitect Apply as a Claude Code plugin | 2 |
  | Agents collaborating through a Git remote ([DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)) | 3 |
  | Classic agentic coding, the baseline | 4 |

## 1. The task

A task is a directory the suite writes:

| Path | Content |
|---|---|
| `task.md` | What to build or change, in ordinary language, with the delivery rules |
| `model/` | The same intent as a canonical model, for variants that work from one. The suite proves that `task.md` and `model/` say the same |
| `start.bundle` | The repository to start from, as a Git bundle with its commit. Empty for a new project |
| `limits.json` | The wall-clock deadline |

A **chain** is a sequence of tasks on one project, such as a fresh build followed by nine changes. For each step the suite builds the start from the variant's own previous result plus the step's change to `task.md` and `model/`.

The hidden material (oracle, probes, rubric) never reaches a variant.

## 2. The variant

A variant is one command:

```text
VARIANT TASK_DIR OUT_DIR [STATE_DIR]
```

- It reads `TASK_DIR` and works however it likes, with whatever provider and model its configuration names.
- It writes `OUT_DIR/result.bundle`: the final repository, one commit that descends from the start. No result means no result.
- It may write `OUT_DIR/run.json` with what it knows about itself: provider and model used, tokens, status and cause. The suite records this; it does not decide anything from it.
- `STATE_DIR` is the variant's own memory within a chain. It is empty at the chain's first step and handed back unchanged at every later step. Only the variant reads it.
- The suite starts the command, measures the wall time and stops it at the deadline.

Plugging in a new variant means writing this one command. Nothing else in the suite changes.

## 3. The evaluation

The same for every variant, from frozen tools:

| Area | How it is measured |
|---|---|
| **Correctness** | The hidden oracle and probes, through the program's public interface |
| **Build and tests** | The result's own build and tests in a fixed container |
| **Scope** | A census of every changed path: what was changed that the task did not ask for, and whether the given task and model were left untouched |
| **Code quality** | A blinded review against a fixed rubric, with every critical finding checked mechanically and confirmed by a second reviewer |
| **Delivery** | Whether a result came back at all, along a chain |
| **Cost** | Wall time, and tokens where the variant reports them |

**The report** shows each variant per area, with its uncertainty, and the predeclared comparison per area. There is no single overall score. The answer is which variant performs better in which areas.

## State

- **Done:** Phases A, B and C compared Markitect with one agent on the Library and Makerspace ([report](../validation/v1-comparison-report-20261026.md)). No quality advantage for Markitect was shown, at 9 to 10 times the time and 9 to 42 times the tokens.
- **Tools:** assessment, review and analysis exist and are tested. They are tied to the Library and Makerspace and to the old arm runners, and the ledger builder lives outside the repository.
- **Variants:**
  - Lane 1's is run by step 4's controller outside the repository.
  - Lane 2's has its own runner on a branch.
  - Lane 3's is a prototype in preparation.
  - The baseline has `runner.py`, which so far runs Codex only.
- **Pilot:** the Makerspace old-versus-new gate cells stay as they ran on `gpt-6-sol`.

## Next

1. **Interface on `main`:** the task format, the variant command, and the suite runner that builds tasks and chain starts, starts a variant under its deadline and collects the result, with offline tests.
2. **Evaluation on `main`:** the assessment, the review and the report per area, reading only `result.bundle` and the suite's own measurements.
3. **Baseline:** one coding agent as a variant command, with Claude Code first and Codex as a configuration.
4. **The other variants:** each lane wraps its variant in the command.
5. **First bounded dispatch,** after the owner's approval: one Library task through every variant on Claude, as a smoke of the suite, not as a counted run.
6. **Portfolio:** new projects with hidden oracles for the counted evaluation.
