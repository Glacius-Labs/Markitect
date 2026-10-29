# Evaluation suite v2

**Status:** design, approved in direction by the owner on 28 October 2026; E1 and E2 are built. On 28 October 2026 the target became concept 2.0, and E3 to E6 were replaced by steps T6 and T7 of its [build](../implementation-plan.md#concept-20-build), which compare one 2.0 implementation with K; the sections on the arms M, P and G below are historical. T6's parts that do not need the 2.0 implementation are built ([T6 below](#t6-evaluation-against-15)); the converted tasks and the planted defects followed with T6, and the 2.0 arm itself with T7. The [evaluation suite guide](../guides/evaluation-suite.md) stays the contract of what is built and changes with each increment.

## Why

The owner, 28 October 2026: "The evaluation suite is critical and should be suitable to give us data to base our decisions upon." The first version can show that a variant runs; it cannot yet show which variant is better. Four gaps decide that:

1. **The tasks do not separate the variants.** Plain agents solve Library M1 completely in minutes, and Phases A and B showed no difference on the Library ([report](../validation/v1-comparison-report-20261026.md)). Differences appeared only on the larger Makerspace chain, which is also where Markitect's claim lives: intent kept intact across many changes.
2. **The conditions are not equal.** Arm M runs its agents in Linux containers; K and P run as the user on Windows, where an agent could read the hidden tests. The repository is public, so the existing hidden tests are also reachable from the internet.
3. **The judging is wired to two projects** and does not read the suite's output.
4. **Three runs per cell are too few** to compare variants whose results vary from run to run.

Smaller gaps: technical failures were counted as if they were results; the human work of writing requirements or a model is not counted; and blinding of the code review is weaker than assumed.

**What is measured.** The goal is better results for less human attention, not for less computing ([vision](../vision.md#the-thesis)). Human attention is what a person must put in (requirements, a model) and what they must still find and repair afterwards (the defects the hidden judge finds). Time, tokens and machine use are reported, not optimized.

## Decisions

Owner, 28 October 2026: every arm runs on Claude Sonnet 5.5 ([build page](../workstreams/build.md#decisions)); the portfolio is the Library as a smoke only, the Makerspace chain as the main test (later the same day an intermediate; the new project is the real test, section C), and one new, larger project; all gaps above are addressed ("Yes lets go with your recommendation and address all of these.").

## A. Equal conditions: one sandbox

Every variant runs under the same conditions, so that the variant, not the environment, explains a difference, and no agent can reach hidden material.

- **One image** in `tools/evaluation/sandbox/`: Debian with Git and the Go and Python versions of the runtime image, plus pinned Claude Code and Codex CLIs. A series pins it by digest.
- **One container per run,** started by `suite.py run --sandbox IMAGE`. Mounts: `/task` read-only, `/out` and `/state` writable, `/variant` read-only (the variant's own code), and the access-only login copy read-only. Nothing else from the host. A non-root user.
- **No equal compute budget.** Each variant uses the machine and the tokens it needs. One generous safety limit per run, the same for every variant, protects the machine; each variant declares its per-container limits (M through its runtime profile) and the ledger records them. The limit and the task deadline are set well above what the pilot measured; a run that exceeds either is a result, not a technical failure.
- **Network:** one egress-limited network. Its only route out allows the evaluation provider's endpoints, enforced at network level so that no tool needs a proxy setting: no web, no GitHub, no Go module proxy (`GOPROXY=off`). Portfolio tasks therefore require the Go standard library only.
- **Variants that start containers.** M and G run their Managers in containers. This is the one exception to the mount rule: their sandbox also gets the host engine's socket, for their own variant code (trusted and reviewed), never for an agent. The socket gives root over the host; that is acceptable because no hidden material is on the machine during runs (below) and agents cannot reach the socket. The container adapter mounts volumes only, and every Manager container joins the same egress-limited network.
- **Hidden material stays off the machine during runs.** The judge folder is fetched only for judging, after the runs. For the public tasks, a transcript audit additionally checks that no agent read or quoted oracle material, as Phase A–C did.
- **Escape probe:** before any series, a probe variant tries from inside the sandbox to read the host file system (the repository included), to reach github.com and to reach the engine socket; all three must fail. For M and G, the probe runs as an agent inside a Manager container.
- **Runs are serial,** one at a time on this machine, under the shared engine lock, so wall times are comparable.

## B. Judges are plug-ins too

Each task brings its hidden judge, stored apart from the task folder; the suite runs it the same way for every variant.

- **Judge folder** (hidden): `judge.json` naming a judge image by digest and its command, plus the judge's files (oracle, probes, rubric).
- **`suite.py judge`** runs the judge in a container without network, with the result bundle, the start bundle and the task folder read-only. The judge writes `judge.json`: Correctness (cases passed and failed), Build and tests (the result's own build and tests), Scope (every changed path, the unrequested ones and any protected input touched).
- **Judges are tested:** each task carries a reference result that its judge must pass and a few seeded mutants (a broken rule, an unrequested change, a failing build) that it must fail.
- **Code quality, secondary:** `suite.py review` makes a blinded copy: tree and diff with `.markitect/` and registry files removed; the code itself stays unchanged, and the arm-guess rate below measures what blinding is left. A fresh reviewer judges it against the task's rubric; every critical finding is checked mechanically and confirmed by a second reviewer, reusing the Phase A–C rubric check and prompts. Each reviewer also guesses the variant, and the report gives how often it guessed right. All four arms run on Claude, so a Claude reviewer's family bias falls on every arm alike; it is a stated limit, and a reviewer of another family is used when one is available.
- **Delivery** comes from the suite's own `suite.json` and `chain.json`.
- **Technical failures do not count.** Each variant is made to run without technical issues first, then evaluated. One pre-coded classifier, the same for every variant, separates:
  - a **technical failure:** the provider (capacity, rate limit, server error), the infrastructure (engine, sandbox, suite), or the variant's own runtime machinery outside the agents' work (a host crash, a lost session). The run is void: it is recorded and relaunched, and it never enters the results;
  - a **result:** everything the variant's work produced, including a wrong or incomplete result, a timeout, and a deliberate stop such as a model blocker.

  A failure of the variant's own runtime is technical only if it matches a signature written into the protocol before the series, and no agent output (a handoff, a tool call, a report or a blocker) led to it. A crash of the Claude or Codex CLI counts the same for every variant. The report shows the void rate per variant.

  A variant enters a counted series only after its stability gate (section D). If technical failures recur during a series beyond the share the protocol sets, that variant leaves the series and goes back to stabilization; the report says so, it is never left out silently.
- **Cost, descriptive:** wall time, and tokens counted by one suite-owned parser over every transcript of a run, subagents included, plus the subscription quota a run used. They are reported, not a target.
- **Human attention, the primary area:** the defects a person would still have to find and repair (from Correctness, Build and tests, Scope, confirmed critical code-quality findings, and steps with no result), and, for each task, the effort of the input a variant actually uses: the requirements for K, the requirements and the model for M, P and G. For the Library and the Makerspace, `task.md` is rendered from the model and that effort is unavailable. For the new project the author writes neutral requirements first, as `task.md`, and the model second, and times each; the guide's task section changes with E4 to allow a `task.md` that is not rendered. These counts stand in for the attention of review, correction and waiting; they do not measure it directly.
- **`suite.py report`** reads every run, judgment and review of a series and writes one table per area. There is no overall score.
- The existing Library and Makerspace assessment (`tools/comparison/assess_candidate.py`) becomes their judges.

## C. Portfolio

| Task | Role | Hidden material |
|---|---|---|
| Library M1 | Small project: shows that a variant and the suite work technically; never a comparison | Public (in this repository) |
| Makerspace chain: fresh build W0 and changes C1 to C9 | Intermediate: a first idea of what works, without much effort; pilot and descriptive results | Public; arm M was tuned on it ([gate](../validation/v1-makerspace-gate-20261027.md)) and its README marks it pilot-only |
| A new, larger project: a fresh build and about six to eight changes | The confirmatory test, on unseen material | Private, in the private repository `Glacius-Labs/Markitect-Evaluation` (owner, 28 October 2026), published only after the comparison |

**Owner, 28 October 2026:** "Library is just there as a small project, to test that a variant technically works and that the eval suite works. Makerspace is an intermediate, where we can get an idea what works without much effort. As you propose the real test should be the new unseen one". The design review had found that arm M was developed against the Makerspace chain, so a result there cannot confirm anything.

The new project's requirements, model, hidden oracle, seeded mutants and rubric are written by an author who works on none of the variants, and the equivalence of requirements and model is checked and recorded once. The author also writes a reference implementation for every step, in the layout the model's execution configuration names (`realization/`), so that each change task can start from the reference result of the step before, and every variant, M included, can continue from it. The domain is a local energy community: members, meters and readings, tariffs, shared solar credits and monthly bills (owner, 28 October 2026). Build proposed changes such as a new tariff type, a credit cap, a member leaving mid-month and a retroactive meter correction; the author decides the final list. Authoring starts with E1, because it takes longest. The variants are frozen before anyone working on them sees it.

## D. Counted evaluations

- **Stability gate:** before a counted series, each variant runs three Library runs in a row and one Makerspace chain without a technical failure. Technical failures found here are fixed in the variant, not counted.
- **Pilot first, uncounted:** the Library and the Makerspace chain for every variant that will be counted, G included if it is counted. It calibrates time, token use, spread and the plumbing. Pilot data never counts.
- **Protocol per counted series,** approved by the owner before the first counted run: tasks, variants and their frozen revisions, the model, the repetitions, the run order, the series' run and quota budget (how many runs we can afford, not a per-run limit), the stop rules, the technical-failure signatures and the reading.
- **Primary contrasts:** one or two, named in the protocol, for example M against K and P against K. Each has one primary endpoint about human attention, such as the share of hidden cases passed (the defects left for a person), and a minimum meaningful difference that the owner sets.
- **Units:** the new project's changes run as independent tasks, each from a fixed reference start, five repetitions each per variant. This gives more units and no failure cascade. Its full chain runs a few times per variant for sustained coherence, as a secondary result.
- **Order:** by repetition, all variants in turn, the order rotated per repetition, so that time of day and provider load fall on every variant alike.
- **Reading:** the 95 % interval of the paired difference, blocked by repetition and task. Each contrast reads as better, worse, "a meaningful advantage is excluded" or undecided. Secondary areas are reported descriptively. The owner decides on the tables.
- **Ledger:** every launch is recorded, including voided technical failures, timeouts and aborts, with its class.

## Increments

| # | Builds | Done when |
|---|---|---|
| E1 | The sandbox image, `run --sandbox`, the egress limit and the safety limit; K runs in it. The new project's authoring starts | Offline tests pass; the escape probe fails all three reads; a bounded smoke of K on the Library delivers in three runs in a row, each with a result, a model match and no quarantine. It replaces the [host smoke](../validation/eval-k-library-smoke-20261028.md) |
| E2 | `judge`, `review`, `report`, the failure classifier and the token parser; the Library judge; the guide's failure rule becomes: "A timeout or a missing result caused by the variant's work counts as a failure in Correctness and Delivery; a technical failure is void and relaunched." | The Library judge passes its reference and fails its mutants; the classifier sorts recorded provider, infrastructure and variant failures correctly, and the token parser matches a hand count on a recorded transcript with subagents; the E1 smoke's results are judged and reported per area |
| E3 | The variant commands for M (new `tools/arm-m/`) and P, with M's Manager containers on the egress-limited network | M and P each deliver in three Library runs in a row without a technical failure, each with a result, a model match and no quarantine |
| E4 | The Makerspace chain task and judge; the new project's tasks and judge | Each judge passes its reference and fails its mutants; the new project's equivalence check is recorded |
| E5 | The pilot series | Pilot report and a proposed protocol for the owner |
| E6 | The counted series | Report per area, owner decision |

G joins at E3 once it has a provider (Codex tokens or a Claude backend), after its prototype's review fixes. If it misses the pilot, it is not counted in E6.

## T6: evaluation against §15

**Built,** on 28 October 2026 (the [guide](../guides/evaluation-suite.md) and the [runner's README](../../tools/evaluation/README.md) hold the contract):

- **Change series.** Series judges answer for a step and every earlier one, with counts per introducing step, and the suite refuses a judgment that is not cumulative. `judge-chain` judges a whole chain and adds the regressions along it and its final drift. `judges.py start` writes fixed reference starts, so that each change also runs as an independent task (section D's units).
- **Judges.** The [Makerspace judge](../../tools/evaluation-judges/makerspace/README.md) (W0 to C9, wrapped from the Phase C assessment, with the oracle's change census at C3, C7 and C9) and the [Library series judge](../../tools/evaluation-judges/library/README.md) (M1 to M5, the Phase A and B cases, with new reference results for M2 to M5). Each passes its references and fails its mutants: six scope and drift mutants and 31 oracle-family mutants for the Makerspace, seven for the Library.
- **One area per §15 criterion** in the report: correct, no drift (the judges' drift checks, applied the same way to K, and a slot for the 2.0 arm's own `verify --full`), nothing skipped (the [Result contract](../../tools/evaluation/results.py) that T4's Result page fills), stops right, and the paired reading of this design's section D against K.
- **The native arm mode** for the 2.0 arm, with a scripted stand-in arm that also runs the escape probe as an agent in a Run container.
- **The series' inputs and the planted tasks.** [series.py](../../tools/evaluation/series.py) pairs each step's single-agent document with its concept 2.0 model from T2 under the judges' step names. Five [planted tasks](../../tools/evaluation-judges/plants/README.md), recorded on 28 October 2026 before any run: three defects (two contradictions and a threshold left open, at Library M2 and Makerspace C3 and C5) and two harmless gaps (Library M2, Makerspace C6), each edited the same way in the model and in `task.md`; every planted model compiles.

**How K is scored on planted tasks: decided by the owner on 28 October 2026, as recommended here.** `suite.py task` ends every `task.md` with the stop rule (`STOP_RULE` in [suite.py](../../tools/evaluation/suite.py)), which every arm receives; the 2.0 arm also stops by its own rule, a model defect ends the Run at Root (concept 2.0 §6.2). Score every variant with one mechanical rule: a result equal to its start did not land; anything else landed. A planted defect is scored right when nothing landed, a harmless gap when something landed; a gap's landed change is also judged under Correct like any other. For this to be fair to K, the stop must be open to it in words it reads: every task's delivery rules, the same for every arm, should say that if the requirements contradict each other or leave open a behavior at the program's boundary, the agent changes nothing and says why in its final report. Without that sentence K is told to implement, and a K that never stops measures the prompt rather than the practice. The reason K gives is recorded for the reader, not scored; the 2.0 arm's Result says why its Run ended. The alternative, judging K's report text for a named defect, needs a reviewer and is less reproducible.

**Built with T7,** on 29 October 2026: the 2.0 arm's variant command ([markitect_arm.py](../../tools/evaluation/README.md#the-concept-20-arm)). It commits the task's model as the owner's input before its Run, with the project's registry pages, which give each Package its paths: a Run reads ownership from its base, so a fresh project needs them before its first Run, and [tools/evaluation-tasks](../../tools/evaluation-tasks/README.md) now carries one set per project. **Fixed reference starts** are answered by the arm's own adoption step, not by registry pages in the starts: the arm commits the start's model (the task's new `base-model/`) with one registry row without links per file, then the change, and runs on the change. K's starts stay as they were.

**Not built yet, and why:**

- The 2.0 arm's escape probe as an agent in one of its own Run containers: the host offers no way to start a probe as an agent, and the stand-in shows the plumbing. The suite checks the arm's containers' networks on every run instead.
- The new project's judge. Its author wrote it to the judge interface; to judge a series, it reads its step from `EVALUATION_STEP` and declares `steps` in its `judge.json`.
- WSL Linux runs of the native mode needed Docker Desktop's WSL integration for the Linux distribution; WSL Ubuntu reaches Docker since 29 October 2026.

## Open questions

1. **Minimum meaningful difference and the run and quota budget per counted series:** the pilot measures spread and quota use; the protocol asks for both.
2. **The stop rule for K** (T6 above): the owner decides whether every task's delivery rules carry the stop sentence and whether stops are scored mechanically.
