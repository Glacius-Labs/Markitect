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
  - The tasks' judges in `tools/evaluation-judges`: the [Library M1 judge](../../tools/evaluation-judges/library-m1/README.md) and the series judges of the [Library](../../tools/evaluation-judges/library/README.md) (M1 to M5) and [Makerspace](../../tools/evaluation-judges/makerspace/README.md) (W0 to C9).
  - The assessment, review prompts, rubric check and analysis in [`tools/comparison`](../../tools/comparison/README.md), which the judges and the review reuse.
  - The task portfolio.
- **It uses:** the hidden oracles in [`tools/library-oracle`](../../tools/library-oracle/README.md) and [`tools/makerspace-oracle`](../../tools/makerspace-oracle/README.md).
- **Variants:** each meets the variant interface in its own code; the [workstreams](../workstreams.md#arms) list them.

## 1. The task

A task is a directory the suite writes:

| Path | Content |
|---|---|
| `task.md` | What to build or change, in ordinary language, with the delivery rules. `suite.py task` ends every `task.md` with the same stop rule: if the requirements contradict each other or leave open a behavior at the program's boundary, change nothing and say why in the final report (owner, 28 October 2026) |
| `model/` | The same intent as a canonical model, for variants that work from one. `task.md` is rendered mechanically from `model/`, reviewed once for the same meaning and frozen. Every variant receives both; in the counted series, K gets the model without its registry pages |
| `start.bundle` | The repository to start from, as a Git bundle. For a new project, one empty commit |
| `base-model/` | Optional, with a fixed reference start: the model that start realizes, the step before's. Every variant receives it; a variant that works from a model commits it before the change. The registry pages in `model/` and `base-model/` are only for a variant that works from them: in the counted series, K gets the task without them (`suite.py run --without-pages`; Build, 29 October 2026) |
| `task.json` | The start commit, the deadline and a digest over the rest |

A **chain** is a sequence of tasks on one project, such as a fresh build followed by nine changes. Each step starts from the variant's own previous result. After a step without a result, the next step starts from that step's start. Each counted evaluation states this rule before any run, because it can decide a chain.

A **change series** runs end to end, as a chain, or step by step from **fixed reference starts**: each change is an independent task whose start is the reference result of the step before, the same for every variant ([how](../../tools/evaluation/README.md#change-series)). Either way, each step is judged at its step and cumulatively: every hidden case of that step and of every earlier one.

A **planted task** carries a planted model defect (a contradiction, or a gap that changes behavior at the boundary) or a harmless gap, and its expected outcome: no landing for a defect, a landing for a gap. The expectation is hidden material, recorded with its time before any run (`plant.json` in the judge folder). The defect itself is in the task, in `task.md` and in the model alike, visible to every variant. The portfolio's series and its planted tasks are in [series.py](../../tools/evaluation/series.py) and [tools/evaluation-judges/plants](../../tools/evaluation-judges/plants/README.md). Like the Library and Makerspace oracles, those plants are public pilot material; the new project's plants stay off the machine during runs.

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

**Native conditions.** A variant whose host is trusted, reviewed code that runs its agents only in containers of its own (the concept 2.0 arm) runs with `--native` instead ([how it works](../../tools/evaluation/README.md#the-native-arm-mode)). Its host runs natively on the machine, on Windows and on Linux; the suite stands up the same egress-limited network and hands the host what its containers need to join it. Every container started during the run must join only that network, internal networks the variant created itself (which reach no other network), or none, which the suite checks from the engine's events; a container on any other network makes the run invalid. The 2.0 arm ([how it works](../../tools/evaluation/README.md#the-concept-20-arm)) commits the task's model as the owner's input before its Run, which is not its work, and delivers the start unchanged when the Run does not land. Hidden material stays off the machine: the protocol names where it could lie, and a run refuses to start while any of it is there. The same safety limit is the variant's to apply to its containers, through its own runtime configuration, which the ledger records.

A run with neither runs the command as a host process. That is for tests only, and `suite.json` marks it as not valid for counting.

## 3. The evaluation

The same for every variant, from frozen tools. Each task brings its hidden **judge**: a folder with `judge.json`, which names the judge image by digest and its command, and the judge's files (oracle, probes, rubric), kept apart from the task. `suite.py judge` runs it in one container without network, with the result, the start and the task read-only, and the judge answers the first three areas ([how it works](../../tools/evaluation/README.md#judging)). Each judge is tested: it passes a reference result and fails seeded mutants, at least a broken rule, an unrequested change and a failing build. A **series judge** judges every step of a change series: the suite tells it the step, and it answers for that step and every earlier one, with the counts per step that introduced the cases and the drift checks below; it is tested on a reference per step and on mutants at their steps, including a forgotten rule, a stale file and an unaccounted file.

| Area | How it is measured |
|---|---|
| **Human attention** (primary) | The defects a person would still have to find and repair: failed hidden cases, a failed build or failed tests, unrequested paths, protected inputs touched, confirmed critical code-quality findings and steps with no result. Next to them, for each task, the effort of the input a variant uses, where it is known. These counts stand in for the attention of review, correction and waiting; they do not measure it directly |
| **Correctness** | The judge's hidden oracle and probes, through the program's public interface: cases passed and failed |
| **Build and tests** | The result's own build and tests, run by the judge |
| **Scope** | A census of every changed path against the start: what changed that the task did not ask for. A result that changes `task.md` or `model/` loses here, and is otherwise evaluated with the original files put back |
| **Code quality** (secondary) | A blinded review against the task's rubric (`suite.py review`): tree and diff without `.markitect/` and registry files, the code unchanged. Every critical finding is checked mechanically and confirmed by a second reviewer. Each reviewer also guesses the variant, and the report gives how often it guessed right: what the blinding leaves |
| **Delivery** | Whether a result came back in time, step by step along a chain |
| **Cost** (descriptive) | Wall time, measured by the suite, and tokens counted by the suite over every transcript of a run, subagents included, with the cost the CLI states. Reported, not optimized |

A timeout or a missing result caused by the variant's work counts as a failure in Correctness and Delivery; a technical failure is void and relaunched. A missing result scores as the start, the change not done: the earlier steps' hidden cases pass and the step's own fail.

**Acceptance areas.** After these, the report has one table per criterion of [concept 2.0 §15](../concepts/v2-concept.md#15-acceptance), the same for every variant:

| Criterion | How it is measured |
|---|---|
| **Correct** | Every landed change (a delivered result that differs from its start) passes every hidden case of its step, earlier steps' cases included |
| **No drift** | After a chain, the series judge's drift checks of its end result: earlier steps' cases that fail (forgotten rules), terms a step retired still in the program's source (stale files), and files that neither the build, the documentation nor Markitect's bookkeeping account for. Next to them, the cases that passed at one step and fail at a later one, and the variant's own independent `verify --full`, where it has one |
| **Nothing skipped** | Every Result a variant returns: each checklist item has a verdict, evidence and an approving checker, and the required checks ran on the whole repository. A variant with a checklist returns one Result per Run in the [Result contract](../../tools/evaluation/results.py); a variant without one, such as K, has nothing to check |
| **Stops right** | A planted defect does not land, a harmless gap does. Scored the same way for every variant: a result equal to its start did not land. A launch without a result has no outcome and counts as wrong |
| **At least as trustworthy as K** | Per contrast named in the protocol, the paired difference of one endpoint by task, step and repetition, its 95 % interval (over the pairs, or over each change's mean difference when the protocol blocks by change) and its reading (better, worse, a meaningful advantage is excluded, undecided), and whether the variant is at least as good: the interval's lower end lies above minus the minimum meaningful difference. Where a unit has a result for only one variant, the contrast is read a second time with the other scored as the change not done. The verdict is yes or no only when both readings agree, and otherwise undecided, so that units without a result cannot decide it by dropping out. The counted series reads "at least as good" by the owner's rule: both case contrasts and a landed-change rate not below K's, all at once ([protocol](../validation/counted-series-protocol-20261030.md#reading)) |

A planted defect's launch counts under Stops right only.

**Technical failures.** One pre-coded classifier, the same for every variant, sorts every launch ([classify.py](../../tools/evaluation/classify.py)). A technical failure is void: it is recorded, relaunched and never counted. It is a failure of the infrastructure (the sandbox, the engine or the suite), of the provider (a session ended on an API error for capacity, a rate or usage limit, a server error or a refused login; for a variant whose Results list its agent starts, a Manager's last start failed with such a class), or of the variant's own runtime machinery, but only when it matches a signature written into the series protocol before the series and no agent output led to it. Everything else is a result, including a wrong or incomplete result, a timeout and a deliberate stop such as a model blocker. A run under other conditions or on another model is invalid and not counted.

**The report** (`suite.py report`) reads every launch, judgment and review of a series from its ledger and shows each variant per area, human attention first, with the void rate per variant. It gives the mean, range and count per area, and, for the contrasts the protocol names, the paired reading with its 95 % interval. There is no single overall score. The answer is which variant performs better in which areas.

**Counted evaluations.** Each one has its own protocol, approved before the first counted run, like the [Phase A–C protocol](../validation/v1-comparison-protocol-20261024.md):
- tasks and repetitions;
- run order;
- the model;
- the decision rule and the test per area;
- the frozen tool revisions.

The [series runner](../../tools/evaluation/README.md#the-series-runner) runs a counted series and its stability gate one launch at a time: on a quiet engine, with the login refreshed before a launch it cannot cover, each launch classed with its variant's signatures, and the protocol's relaunch and stop rules.

## Limits

- **Isolation.** The sandbox keeps a variant away from the host's files, the web and the engine. A variant's own code is mounted, so judge sources (references, mutants, oracles) never sit in a folder that is mounted as variant code: the judges live in `tools/evaluation-judges`, apart from `tools/evaluation`. The Library and Makerspace oracles are still public, in this repository and on GitHub; a transcript audit checks that no agent quoted them. The new project's oracle stays off the machine during runs ([design](../design/evaluation-suite-v2.md#a-equal-conditions-one-sandbox)).
- **Native conditions trust the host.** A native variant's host has the machine's files, network and engine; only its containers are held to the network. An internal network the variant creates itself, unlike the suite's, may leave the engine's host reachable at its bridge address; the record shows its gateway mode. That is why only reviewed variant code runs natively, and why hidden material must be off the machine, not just out of reach. The network check reads the engine's events and relies on the engine lock to keep other runners' containers out of them.
- **Drift by mechanism.** The judges' drift checks are mechanical stand-ins for an independent review: retired terms are searched in non-test source only, and a file counts as accounted when the build, the documentation, the module files, test data or Markitect's bookkeeping covers it, not when it is still needed.
- **Egress by name.** The gateway allows provider host names, not addresses: a tool that resolves names itself, without `/etc/hosts`, cannot reach the provider. Content sent to an allowed host is not inspected, so the provider endpoint itself remains a way out for anything the sandbox holds, the access-only login included.
- **Not limited:** disk. `/out` and `/state` of a run and `/out` of a judge are host folders; the safety limit covers CPUs, memory, processes and `/tmp` only.
- **Reviewer family.** The earlier studies used reviewers of another model family than the writers. With Claude writers, a Codex reviewer restores that; if none is available, the report states the same-family review as a limit. The review runs on Claude today.
- **One judge container.** A judge runs the result's own code (its build, tests and program) in the container that holds the oracle. The result's processes cannot reach the network or the host, and the Library judge stops them before it writes its answer, when the suite's container tells it that it may (`EVALUATION_JUDGE_CONTAINER`). They could still read the oracle while they run, which for the public Library and Makerspace oracles adds nothing.
- **Tokens by transcript.** The token count reads Claude Code transcripts and stream output; a session that left only stream output is counted from the totals its CLI states at the end of each process. A variant whose sessions leave no transcript in its output has no token count.
