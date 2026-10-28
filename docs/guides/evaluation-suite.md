# Evaluation suite

The evaluation suite answers one question: which way of developing a project performs better, and in which areas? It works in three steps:

1. **Task:** what we want to build.
2. **Variant:** a solution is plugged in and builds it.
3. **Evaluation:** the result is judged against fixed criteria.

The suite treats every variant as a black box. It does not care how a variant works or which agents and tools it uses. Every evaluation can run with Claude or with Codex. The evaluation names one model, the suite hands it to every variant, and the variant uses it however it runs. A difference in results then comes from the variant, not from the model.

The Build and evaluation session owns the suite and its variants ([build page](../workstreams/build.md)); this guide is the suite's contract.

## Scope

- **The suite owns:**
  - The task format, the variant interface, the criteria and the report.
  - The suite runner in [`tools/evaluation`](../../tools/evaluation/README.md) and the baseline variant there.
  - The assessment, review and analysis in [`tools/comparison`](../../tools/comparison/README.md).
  - The task portfolio.
- **It uses:** the hidden oracles in [`tools/library-oracle`](../../tools/library-oracle/README.md) and [`tools/makerspace-oracle`](../../tools/makerspace-oracle/README.md).
- **Variants:** each meets the variant interface in its own code; the [workstreams](../workstreams.md#variants) list them.

## 1. The task

A task is a directory the suite writes:

| Path | Content |
|---|---|
| `task.md` | What to build or change, in ordinary language, with the delivery rules |
| `model/` | The same intent as a canonical model, for variants that work from one. `task.md` is rendered mechanically from `model/`, reviewed once for the same meaning and frozen. Every variant receives both |
| `start.bundle` | The repository to start from, as a Git bundle. For a new project, one empty commit |
| `task.json` | The start commit, the deadline and a digest over the rest |

A **chain** is a sequence of tasks on one project, such as a fresh build followed by nine changes. Each step starts from the variant's own previous result. After a step without a result, the next step starts from that step's start. Each counted evaluation states this rule before any run, because it can decide a chain.

The hidden material (oracle, probes, rubric) never reaches a variant.

## 2. The variant

A variant is one command:

```text
VARIANT TASK_DIR OUT_DIR [STATE_DIR]
```

- It reads `TASK_DIR` and works however it likes. The evaluation's model is in the environment variable `EVALUATION_MODEL`, as provider and exact model ID, for example `claude:claude-sonnet-5-5`.
- It writes `OUT_DIR/result.bundle`: the final repository, one commit that descends from the start.
- It may write `OUT_DIR/run.json` with what it knows about itself: the provider and model it actually used, tokens, status and cause. The suite checks only that the reported model is the evaluation's model; a run on another model is invalid.
- `STATE_DIR` is the variant's own memory within a chain. It is empty at the first step and handed back unchanged at every later step.
- The suite starts the command, measures the wall time, stops it at the deadline and checks that the result descends from the start.

Plugging in a new variant means writing this one command. Nothing else in the suite changes.

## 3. The evaluation

The same for every variant, from frozen tools:

| Area | How it is measured |
|---|---|
| **Correctness** | The hidden oracle and probes, through the program's public interface |
| **Build and tests** | The result's own build and tests in a fixed container |
| **Scope** | A census of every changed path against the start: what changed that the task did not ask for. A result that changes `task.md` or `model/` loses here, and is otherwise evaluated with the original files put back |
| **Code quality** | A blinded review against a fixed rubric. Every critical finding is checked mechanically and confirmed by a second reviewer |
| **Delivery** | Whether a result came back, step by step along a chain |
| **Cost** | Wall time, measured by the suite. Tokens are reported as the variant states them, descriptively, and are not compared, because providers count them differently |

A timeout or a missing result is never dropped: it counts as a failure in Correctness and Delivery.

**The report** shows each variant per area, with its uncertainty, and the predeclared comparison per area. There is no single overall score. The answer is which variant performs better in which areas.

**Counted evaluations.** Each one has its own protocol, approved before the first counted run, like the [Phase A–C protocol](../validation/v1-comparison-protocol-20261024.md):
- tasks and repetitions;
- run order;
- the model;
- the decision rule and the test per area;
- the frozen tool revisions.

## Limits

- **Isolation.** Variants run as processes of the user on the host, so a variant could read the hidden oracles in this repository. Before counted runs, variants run in an isolated environment that sees only `TASK_DIR`, `OUT_DIR` and `STATE_DIR`, and the portfolio's oracles live outside the repository.
- **Reviewer family.** The earlier studies used reviewers of another model family than the writers. With Claude writers, a Codex reviewer restores that; if none is available, the report states the same-family review as a limit.
