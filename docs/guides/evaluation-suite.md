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
  - The suite runner in [`tools/evaluation`](../../tools/evaluation/README.md), with its judging, review, failure classifier, token count and report, and the baseline variant there.
  - The tasks' judges: the Library judge in [`tools/evaluation/judges`](../../tools/evaluation/judges/library-m1/README.md).
  - The assessment, review prompts, rubric check and analysis in [`tools/comparison`](../../tools/comparison/README.md), which the judges and the review reuse.
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
- It writes `OUT_DIR/result.bundle`: the final repository, one branch tip that descends from the start. A result equal to the start counts as delivered, with no changes.
- It may write `OUT_DIR/run.json` with what it knows about itself: the provider and model it actually used, tokens, status and cause. The suite checks only that the reported model is the evaluation's model and records the answer; the report treats a run on another model as invalid.
- `STATE_DIR` is the variant's own memory within a chain. It is empty at the first step and handed back unchanged at every later step.
- The suite starts the command, measures the wall time, stops it at the deadline and checks that the result descends from the start.

Plugging in a new variant means writing this one command. Nothing else in the suite changes.

**Conditions.** A counted run starts the command in the evaluation sandbox, the same for every variant ([how it works](../../tools/evaluation/README.md#the-sandbox)):

```text
python -I -B tools/evaluation/suite.py run --task TASK_DIR --model claude:claude-sonnet-5-5 --out RUN_DIR \
    --sandbox IMAGE_ID --variant-code VARIANT_CODE_DIR --lock LOCK -- VARIANT...
```

- One container per run, from one image pinned by its image ID, with a read-only root and a non-root user. It sees the task read-only at `/task`, writes `/out` and `/state`, has its own code read-only at `/variant` and an access-only copy of the provider login read-only at `EVALUATION_LOGIN`. Nothing else from the host.
- One safety limit per run from the series configuration (CPUs, memory, processes, `/tmp`), the same for every variant. It protects the machine and is not a budget: a run that exceeds it is a result.
- The network's only route out is the evaluation provider's endpoints, enforced at network level, so no tool needs a proxy setting: no web, no GitHub, no Go module proxy. Portfolio tasks therefore need the Go standard library only.
- The suite removes the run's containers and networks afterwards and records what it could not remove. A run is not valid for counting when its sandbox did not stand up, its container never started, its egress gateway stopped, or something is left.
- An escape probe, a variant without a model, checks before a series that the sandbox holds: reading host files, reaching GitHub or the web, and reaching a container engine or the host must all fail.

A run without `--sandbox` runs the command as a host process. That is for tests only, and `suite.json` marks it as not valid for counting.

## 3. The evaluation

The same for every variant, from frozen tools. Each task brings its hidden **judge**: a folder with `judge.json`, which names the judge image by digest and its command, and the judge's files (oracle, probes, rubric), kept apart from the task. `suite.py judge` runs it in one container without network, with the result, the start and the task read-only, and the judge answers the first three areas ([how it works](../../tools/evaluation/README.md#judging)). Each judge is tested: it passes a reference result and fails seeded mutants, at least a broken rule, an unrequested change and a failing build.

| Area | How it is measured |
|---|---|
| **Human attention** (primary) | The defects a person would still have to find and repair: failed hidden cases, a failed build or failed tests, unrequested paths, protected inputs touched, confirmed critical code-quality findings and steps with no result. Next to them, for each task, the effort of the input a variant uses, where it is known. These counts stand in for the attention of review, correction and waiting; they do not measure it directly |
| **Correctness** | The judge's hidden oracle and probes, through the program's public interface: cases passed and failed |
| **Build and tests** | The result's own build and tests, run by the judge |
| **Scope** | A census of every changed path against the start: what changed that the task did not ask for. A result that changes `task.md` or `model/` loses here, and is otherwise evaluated with the original files put back |
| **Code quality** (secondary) | A blinded review against the task's rubric (`suite.py review`): tree and diff without `.markitect/` and registry files, the code unchanged. Every critical finding is checked mechanically and confirmed by a second reviewer. Each reviewer also guesses the variant, and the report gives how often it guessed right: what the blinding leaves |
| **Delivery** | Whether a result came back in time, step by step along a chain |
| **Cost** (descriptive) | Wall time, measured by the suite, and tokens counted by the suite over every transcript of a run, subagents included, with the cost the CLI states. Reported, not optimized |

A timeout or a missing result caused by the variant's work counts as a failure in Correctness and Delivery; a technical failure is void and relaunched.

**Technical failures.** One pre-coded classifier, the same for every variant, sorts every launch ([classify.py](../../tools/evaluation/classify.py)). A technical failure is void: it is recorded, relaunched and never counted. It is a failure of the infrastructure (the sandbox, the engine or the suite), of the provider (a session ended on an API error for capacity, a rate or usage limit, a server error or a refused login), or of the variant's own runtime machinery, but only when it matches a signature written into the series protocol before the series and no agent output led to it. Everything else is a result, including a wrong or incomplete result, a timeout and a deliberate stop such as a model blocker. A run under other conditions or on another model is invalid and not counted.

**The report** (`suite.py report`) reads every launch, judgment and review of a series from its ledger and shows each variant per area, human attention first, with the void rate per variant and the predeclared comparison per area. There is no single overall score. The answer is which variant performs better in which areas.

**Counted evaluations.** Each one has its own protocol, approved before the first counted run, like the [Phase A–C protocol](../validation/v1-comparison-protocol-20261024.md):
- tasks and repetitions;
- run order;
- the model;
- the decision rule and the test per area;
- the frozen tool revisions.

## Limits

- **Isolation.** The sandbox keeps a variant away from the host's files, the web and the engine. The Library and Makerspace oracles are still public, in this repository and on GitHub; a transcript audit checks that no agent quoted them. The new project's oracle stays off the machine during runs ([design](../design/evaluation-suite-v2.md#a-equal-conditions-one-sandbox)). Variants that start their own containers (M, G) need the engine socket in their sandbox; that comes with increment E3 and is not built yet.
- **Egress by name.** The gateway allows provider host names, not addresses: a tool that resolves names itself, without `/etc/hosts`, cannot reach the provider. Content sent to an allowed host is not inspected, so the provider endpoint itself remains a way out for anything the sandbox holds, the access-only login included.
- **Not limited:** disk. `/out` and `/state` are host folders; the safety limit covers CPUs, memory, processes and `/tmp` only.
- **Reviewer family.** The earlier studies used reviewers of another model family than the writers. With Claude writers, a Codex reviewer restores that; if none is available, the report states the same-family review as a limit. The review runs on Claude today.
- **One judge container.** A judge runs the result's own code (its build, tests and program) in the container that holds the oracle. The result's processes cannot reach the network or the host, and the Library judge stops them before it writes its answer. They could still read the oracle while they run, which for the public Library and Makerspace oracles adds nothing.
- **Tokens by transcript.** The token count reads Claude Code transcripts and stream output. A variant whose sessions leave no transcript in its output has no token count.
