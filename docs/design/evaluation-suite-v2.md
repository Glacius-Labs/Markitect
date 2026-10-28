# Evaluation suite v2

**Status:** design, approved in direction by the owner on 28 October 2026; built in increments E1 to E6. The [evaluation suite guide](../guides/evaluation-suite.md) stays the contract of what is built and changes with each increment. One change to the owner's portfolio decision waits for the owner (section C).

## Why

The owner, 28 October 2026: "The evaluation suite is critical and should be suitable to give us data to base our decisions upon." The first version can show that a variant runs; it cannot yet show which variant is better. Four gaps decide that:

1. **The tasks do not separate the variants.** Plain agents solve Library M1 completely in minutes, and Phases A and B showed no difference on the Library ([report](../validation/v1-comparison-report-20261026.md)). Differences appeared only on the larger Makerspace chain, which is also where Markitect's claim lives: intent kept intact across many changes.
2. **The conditions are not equal.** Arm M runs its agents in Linux containers; K and P run as the user on Windows, where an agent could read the hidden tests. The repository is public, so the existing hidden tests are also reachable from the internet.
3. **The judging is wired to two projects** and does not read the suite's output.
4. **Three runs per cell are too few** to compare variants whose results vary from run to run.

Smaller gaps: the failure rule, the resource limits and the token count treat variants with many agents differently from variants with one; the human work of writing requirements or a model is not counted; and blinding of the code review is weaker than assumed.

## Decisions

Owner, 28 October 2026: every arm runs on Claude Sonnet 5.5 ([build page](../workstreams/build.md#decisions)); the portfolio is the Library as a smoke only, the Makerspace chain as the main test, and one new, larger project; all gaps above are addressed ("Yes lets go with your recommendation and address all of these.").

## A. Equal conditions: one sandbox

Every variant runs under the same conditions, so that the variant, not the environment, explains a difference, and no agent can reach hidden material.

- **One image** in `tools/evaluation/sandbox/`: Debian with Git and the Go and Python versions of the runtime image, plus pinned Claude Code and Codex CLIs. A series pins it by digest.
- **One container per run,** started by `suite.py run --sandbox IMAGE`. Mounts: `/task` read-only, `/out` and `/state` writable, `/variant` read-only (the variant's own code), and the access-only login copy read-only. Nothing else from the host. A non-root user.
- **One resource ceiling per variant,** the same for every variant in a series: CPU, memory and processes summed over all of a variant's containers. Docker limits each container, so each variant declares its per-container limits (M through its runtime profile), the suite checks that their sum stays within the series ceiling, and the ledger records them. A variant with many Managers shares the ceiling among them, as the Phase A–C baseline did with the six-Manager ceiling.
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
- **Failures are classified the same way for every variant.** One pre-coded classifier sorts every unsuccessful run as variant, provider (capacity, rate limit, server error) or infrastructure. The same retry rule for provider errors applies to every variant, and the report shows both readings: all runs, and runs without provider or infrastructure failures.
- **Cost:** wall time, and tokens counted by one suite-owned parser over every transcript of a run, subagents included, plus the subscription quota a run used. With one provider and one model, tokens are comparable across variants.
- **Human input:** for each task, the effort of the input a variant actually uses: the requirements for K, the requirements and the model for M, P and G. For the Library and the Makerspace, `task.md` is rendered from the model and that effort is unavailable. For the new project the author writes neutral requirements first, as `task.md`, and the model second, and times each; the guide's task section changes with E4 to allow a `task.md` that is not rendered.
- **`suite.py report`** reads every run, judgment and review of a series and writes one table per area. There is no overall score.
- The existing Library and Makerspace assessment (`tools/comparison/assess_candidate.py`) becomes their judges.

## C. Portfolio

| Task | Role | Hidden material |
|---|---|---|
| Library M1 | Smoke and stability check; never a comparison | Public (in this repository) |
| Makerspace chain: fresh build W0 and changes C1 to C9 | Pilot and descriptive results: size, a long chain, cross-cutting rules | Public; arm M was tuned on it ([gate](../validation/v1-makerspace-gate-20261027.md)) and its README marks it pilot-only |
| A new, larger project: a fresh build and about six to eight changes | The confirmatory test, on unseen material | Private, outside this repository, published only after the comparison |

**Waiting for the owner:** the owner chose the Makerspace chain as the main test. The design review found that arm M was developed against it, so a result there cannot confirm anything. The recommendation is to make the new project the confirmatory test and report the Makerspace descriptively.

The new project's requirements, model, hidden oracle, seeded mutants and rubric are written by an author who works on none of the variants, and the equivalence of requirements and model is checked and recorded once. The author also writes a reference implementation for every step, in the layout the model's execution configuration names (`realization/`), so that each change task can start from the reference result of the step before, and every variant, M included, can continue from it. Build proposes the domain; the owner may veto it. Authoring starts with E1, because it takes longest. The variants are frozen before anyone working on them sees it.

## D. Counted evaluations

- **Pilot first, uncounted:** the Library and the Makerspace chain for every variant that will be counted, G included if it is counted. It calibrates time, token use, spread and the plumbing. Pilot data never counts.
- **Protocol per counted series,** approved by the owner before the first counted run: tasks, variants and their frozen revisions, the model, the repetitions, the run order, the budget, the stop rules and the reading.
- **Primary contrasts:** one or two, named in the protocol, for example M against K and P against K. Each has one primary endpoint, such as the share of hidden cases passed, and a minimum meaningful difference that the owner sets.
- **Units:** the new project's changes run as independent tasks, each from a fixed reference start, five repetitions each per variant. This gives more units and no failure cascade. Its full chain runs a few times per variant for sustained coherence, as a secondary result.
- **Order:** by repetition, all variants in turn, the order rotated per repetition, so that time of day and provider load fall on every variant alike.
- **Reading:** the 95 % interval of the paired difference, blocked by repetition and task. Each contrast reads as better, worse, "a meaningful advantage is excluded" or undecided. Secondary areas are reported descriptively. The owner decides on the tables.
- **Ledger:** every launch is recorded, including failures, timeouts and aborts, with its failure class.

## Increments

| # | Builds | Done when |
|---|---|---|
| E1 | The sandbox image, `run --sandbox`, the egress limit and the resource ceiling; K runs in it. The new project's authoring starts | Offline tests pass; the escape probe fails all three reads; a bounded smoke of K on the Library delivers in three runs in a row, each with a result, a model match and no quarantine. It replaces the [host smoke](../validation/eval-k-library-smoke-20261028.md) |
| E2 | `judge`, `review`, `report`, the failure classifier and the token parser; the Library judge | The Library judge passes its reference and fails its mutants; the classifier sorts recorded provider, infrastructure and variant failures correctly, and the token parser matches a hand count on a recorded transcript with subagents; the E1 smoke's results are judged and reported per area |
| E3 | The variant commands for M (new `tools/arm-m/`) and P, with M's Manager containers on the egress-limited network | M and P each deliver in three Library runs in a row, each with a result, a model match and no quarantine |
| E4 | The Makerspace chain task and judge; the new project's tasks and judge | Each judge passes its reference and fails its mutants; the new project's equivalence check is recorded |
| E5 | The pilot series | Pilot report and a proposed protocol for the owner |
| E6 | The counted series | Report per area, owner decision |

G joins at E3 once it has a provider (Codex tokens or a Claude backend), after its prototype's review fixes. If it misses the pilot, it is not counted in E6.

## Open questions

1. **Confirmatory test:** the new project instead of the Makerspace chain (section C).
2. **Where the new project's hidden material lives.** Recommendation: a private GitHub repository owned by Glacius Labs, so that it is versioned and backed up but not public.
3. **Minimum meaningful difference and budget per counted series:** the pilot measures spread and cost; the protocol asks for both.
