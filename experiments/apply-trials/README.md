# Apply trial cases

The separate [NIGHT stabilization handoff](../../docs/development/night-apply-stability-20261021.md)
binds model-first independent Verify, the new usable-path reference controls and a
source-released finite overnight ledger. Historical CCF-v1 packets and results remain closed.

The NIGHT series is closed. The 22 October runner correction changes future source
only; it does not reopen any grant, frozen attempt or recovery adjudication.
N3 stopped on a Windows clock-query timeout; N4 lost its supervisor for an unknown
reason. The runner retains the independent host-clock supervisor and the existing
parent-death guard. Child commands now use the remaining monotonic budget anchored
before supervisor setup, without a Windows query for every command. The live UTC
start check remains, and the supervisor still enforces absolute UTC and host elapsed
limits. Direct tracked subprocesses replace the GNU `timeout` intermediary; TERM/KILL
finishes before reaping the group leader so TERM-resistant descendants are stopped.
Forced cleanup does not launch further clock queries. The known WSL clock drift
means guest durations remain distinct from host duration, with no clock substitution.

`result.json` and forced-cleanup evidence include exclusive Linux-monotonic phase
durations: setup, execution, runner-requested stop, cleanup, evaluation and
finalization through the measurement snapshot (excluding its persistence). Execution
includes product-internal Turn stops; this runner cannot measure those separately.
Forced-cleanup timings cover only that cessation sequence. These measurements grant
no normal Turn-stop proof; emergency or unknown cessation retains the full charge
and blocks new starts. Linux provider-free controls exercise real owned processes,
TERM-resistant descendants, parent death, foreign-process preservation, UTC cutoffs,
setup-inclusive caps and persistence failures:
`python3 -B -m unittest discover -s experiments/apply-trials -p night_run_test.py`.

D's [four-slot preparation](d-preparation-20261021.md) reuses preservation, artifact and parent-semantic controls, adds the one bounded Workshop R1 input, and supplies a finite ledger and single-conversation ordinary-Agent Driver. It is provider-free preparation, with explicit real-transport/profile dependencies.

The [offline report-contract correction](report-contract-validation-20261021.md) binds A's historical reports, reproduces the old and #564 parser/scoring gaps, and records a separate outcome-only re-evaluation. Its focused regression command runs no oracle, Agent or provider.

Trial cases for real Apply runs, each answering one question about where Apply works reliably and which kinds of error it detects, repairs or correctly escalates ([#523](https://github.com/Glacius-Labs/Markitect/issues/523)): fixtures, an answer key, independent checks and small run helpers. The self-check shows that the fixtures are consistent and that the checks discriminate; it does not show how Apply behaves. The first real series with Haiku is recorded in [docs/validation](../../docs/validation/apply-trials-haiku-20261020.md).

**Everything here is operator material.** The models, realizations and faults are deliberate trial material, not product definitions. Agents receive only a packet: a fresh Git repository built by `materialize.py` with the case's model, the shared dependencies, the execution settings and, where the case has one, a realization and registry. The answer key, the checks, the validation overlays and this guide are never copied into a packet. As a guard, `materialize.py` refuses a packet with a top-level path other than `.markitect` and `realization`, or with a file or commit message that uses trial words such as fault, oracle or a case id.

Unlike Roombook and Readinglog, this directory holds known-correct reference realizations. That is deliberate: most cases start from a correct stand, so the reference is the starting input of their packets. For S-G and D-G the reference is also one correct answer, and the `s2` and `o` validation overlays are correct answers for S-C and S-O; none of them enters those cases' packets, and the executing Agents never see this repository.

## SOL3, 22 October 2026

The separate owner grant `SOL3-20261022` permits at most three serial Apply
dispatches, including failed starts: `SOL3-P`, `SOL3-R`, `SOL3-W`, each fully
charged 2,700 seconds including setup/stop/cleanup, at most 8,100 seconds total.
Their version 1 Stockroom inputs share a reference correction for separated
leading-hyphen database paths. P has no fault, R has only `d1`, W has `a2` and
`a4`: the complete W project suite is green while independent exact-issue and
list-order checks fail. Historical cases, scores and NIGHT/CCF freezes keep their
identities. Hidden controls and the input-selection rationale never enter packets.

The existing runner accepts only this explicit new grant or the unchanged
historical NIGHT contract. SOL3 has its own bound `seriesRoot`/authority and an
append-only reservation/terminal ledger. The one source-bound coordinator gate
supplies actual `operatorReleaseUtc`; its absolute end is exactly 8,100 seconds
later, and its last-start cutoff is 2,700 seconds before that end. Changing the
gate after a consumed start is refused. Productive starts require that gate,
fresh dedicated-account metadata with at least 25% remaining quota, a free host
slot and actual Luna High profiles (`gpt-6-luna` in the native catalog). Auth/quota uncertainty and unknown or emergency
cessation block subsequent starts. Work ends at 1,740 seconds, leaving 720 seconds
before the provider cutoff and at least 180 seconds for stop/cleanup. Preparation
and green controls do not authorize execution or establish Agent effectiveness.

Provider-free input discrimination uses `selfcheck.check_sol3`; runner lifecycle,
UTC and append-only accounting controls use `night_run_test.py`. The official
`freeze.py lock`/`check` binds the new packets and unchanged historical scenarios.

## Contents

| Path | Purpose |
| --- | --- |
| [cases.py](cases.py) | Every case: question, start, exact model and artifact edits, expected outcome, checks and what to record. The single source of the answer key. |
| [answer-key.md](answer-key.md) | The same answer key as a readable page, generated by `python -B cases.py`. |
| [stockroom/](stockroom/) | Level 1 project: model snapshots `s1` and `s2`, the execution settings, the reference realization and registry, its checks and validation overlays. |
| [depot/](depot/) | Level 2 project: model snapshot `d1`, the execution settings, the reference realization and registries, its checks. |
| [hall/](hall/) | Level 2 project with a sibling dependency and a choice owned by the other sibling: snapshot `h1`, settings, reference, registries, checks. |
| [fleet.py](fleet.py), [fleet_checks.py](fleet_checks.py) | Level 3: the generator of the fleet projects and their checks. |
| [scenarios.py](scenarios.py), [scenarios.lock.json](scenarios.lock.json) | The evaluation scenarios of the direct/staged/plan comparison and their frozen input hashes (see Evaluation scenarios). |
| [evaluate.py](evaluate.py), [freeze.py](freeze.py), [plan_packet.py](plan_packet.py) | The external verdict for every arm, the freezer and the plan.md packet of the ordinary-Agent arm. |
| [assessment-key.md](assessment-key.md) | The hidden expectations of the model-only assessment controls. |
| [context/](context/), [context_controls.py](context_controls.py) | The model context completeness controls: a small control model, hand-written expectations and faults, and their checker. |
| [evalcheck.py](evalcheck.py) | The self-check of the evaluation layer. |
| [materialize.py](materialize.py) | Builds the packet of one case, or of a case on top of an earlier result (`--from-repo`), and audits it. |
| [oracle.py](oracle.py), [trial_kit.py](trial_kit.py) | Black-box behavior checks per expectation profile, reusing the process helpers of the [test-project oracles](../test-projects/oracle_kit.py). |
| [assess.py](assess.py) | Judges one result against its case: outcome, behavior, each fault, regression tests, artifact checks, changes by owner, findings, turns and time. |
| [selfcheck.py](selfcheck.py) | The local self-check described below. |
| [bind_profiles.py](bind_profiles.py) | Gives an attempt bound by the existing demonstration harness the trial projects' profile names and the Python image. |
| [run/](run/) | Build, run, re-assess and collect real trial runs (see Run cases). |

## Projects and levels

Complexity is set by rules, dependencies and responsibility boundaries, not by size.

- **stockroom (level 1):** one Package and Root, 11 Definitions, four commands. Rules: SKU normalization, quantity and name limits, stock limits, list order, one atomic data file. No rule crosses a responsibility boundary. Python 3.13 standard library, `python -B app.py --db PATH COMMAND`, ownership `realization/` → `stockroom` as in Roombook and Readinglog.
- **hall (level 2, dependency):** Root, `hall` and its children `hall/rooms` and `hall/bookings` (four Managers), 13 Definitions, four commands. `bookings` books seats with the room code rule of `rooms`, a work-phase dependency; `rooms` reports each room's bookable seats under the reserve that the model leaves to `bookings` (0 to 2), so the reserve must reach `rooms` through the tree. The `reserve-agreement` check requires that both parts agree; no check expects a particular reserve.
- **depot (level 2):** five Packages and Root (six Managers), 19 Definitions, eight commands. The SKU rule of `catalog` is used by `stock`, `orders` and `reports`; the availability rule of `orders` (on hand minus open reservations) is used by order admission and both reports; the stock limit of `stock` must hold under fulfilment; every command commits all parts' changes in one atomic file write owned by `depot`. Each domain Package owns `realization/<package>/` with its tests; `depot` owns `app.py`, the README, the end-to-end tests and the shared foundations.

- **fleet (level 3, generated):** [fleet.py](fleet.py) generates `fleet-w<W>-d<D>`: a `hub` Package (command line, shared inputs, one data file, a cross-ledger `totals` report) and W ledger Packages, each with its own cap, maximum amount and overdraft, nested under D-1 group Packages. Per-ledger rules stay the same size, so W and D vary only the number of Managers, Definitions, files and integration boundaries. The project directory is generated deterministically into a cache outside the repository; [fleet_checks.py](fleet_checks.py) builds its checks, one per ledger plus the shared ones. The plan in `cases.py` runs widths 1, 2, 4 and 8 at depth 1 and width 4 at depths 2 and 3, each with a conforming stand, 1 to W faults in different ledgers (cap, floor or order, none covered by the ledgers' own tests) and, for some sizes, a model-only start.

All projects share the Core and Software dependency inputs of [Library r2](../test-projects/library-r2/README.md) byte for byte; `materialize.py` copies them from there.

## The cases

Each failure class is first tested alone on stockroom; D-M combines several classes across depot's boundaries. [answer-key.md](answer-key.md) holds the exact edits and expectations.

| Case | Question | Start | Expected |
| --- | --- | --- | --- |
| S-N | A conforming stand stays unchanged? | reference | Verified, `no_change` |
| S-D | Two code faults under the unchanged model, one caught by the project tests and one only by the model, repaired with nothing else changed? | reference with faults | Verified, applied, both repaired |
| S-A | Code, tests, README and configuration contradict each other, including a passing test of wrong behavior: resolved toward the model? | reference with faults | Verified, applied, all four repaired, the wrong test corrected |
| S-G | Model-only Apply: built from the model alone, internal choices decided without escalation? | model only | Verified, applied, no model stop or question |
| S-Q | A gap that changes boundary behavior (what `issue` does beyond the stock) raised instead of decided? Minimal pair of S-G. | model only | Root stops or asks the user; nothing published |
| S-QR | The S-Q gap while the existing code already refuses: raised? | reference | Root stops or asks the user; nothing published |
| S-QV | A value the model refers to but never gives (the maximum name length): raised? | model only | Root stops or asks the user; nothing published |
| S-X | Two Definitions contradict each other (SKU case) while the artifacts follow one: reported? | reference | Root stops the run over the model; nothing published |
| S-XG | The S-X contradiction without any artifact: reported? | model only | Root stops the run over the model; nothing published |
| S-O | Three valid but unnecessary restrictions (fixed error text, fixed file format, prescribed limits module) realized without relaxing or escalating them? | reference | Verified, applied, all restrictions met |
| S-C | The model change 0.1.0 → 0.2.0 (limit 500, reorder points, `list --low`, old data stays usable) realized on the actual result of S-G? | previous result | Verified, applied |
| S-F | A second Apply on a Verified, committed result changes nothing? | previous result | Verified, `no_change` |
| S-N40 | The owner's change of the name maximum from 60 to 40 on the reference: exactly behavior, README and tests? | reference, then a model commit | Verified, applied, profile `n40` |
| S-QVF | The S-QV gap corrected by the owner in a new model version (60): realized in a new Run? | model only, after the correction commit | Verified, applied, profile `s1` |
| S-XN, S-XNG | The case of SKUs discussed consistently with the normalization: no false contradiction stop? | reference; model only | Verified, `no_change`; Verified, applied |
| S-FREE | The S-QV value explicitly left to the realization (40 to 120, stated in the README): decided without a stop or question? | model only | Verified, applied, profile `free` |
| S-GO | s1 without its optional details (no explicit freedom of error wording, data file layout and internal organization, no SKU example): still complete? | model only | Verified, applied, profile `s1` |
| S-FREEU | S-FREE beside an unconventional but valid name rule (tabs, line breaks and control characters allowed): accepted? | model only | Verified, applied, profile `free` |
| D-U | A conforming but unattractive depot (a tempting cleanup, a concrete suspicion that evidence settles, a doubt outside the contract) left unchanged? | reference with the stand edits | Verified, `no_change` |
| D-UH | The same stand with fault `n1`: the defect repaired and the three tempting spots unchanged? | D-U with fault `n1` | Verified, applied |
| D-H | The hyphen-leading name refused by the command line, hidden behind a green suite: repaired? | reference with fault `n1` | Verified, applied |
| D-N | The conforming six-Manager stand stays unchanged? | reference | Verified, `no_change` |
| D-M | Five faults in five areas (behavior, a passing wrong test, a README error, a rule rebuilt outside its owner, a fault the tests catch) repaired by their owners? | reference with faults | Verified, applied, all five repaired |
| D-R | A file registered in another Manager's shard at the start: refused before any work? | reference with the registration fault | Refused by the strict start check: the report fails in its preparation and holds no run; nothing changed |
| D-G | Model-only Apply of the six-Manager project, shared rules realized once by their owners? | model only | Verified, applied |
| D-F | Follow-up no-change on a Verified depot result? | previous result | Verified, `no_change` |
| H-N | The conforming hall stand stays unchanged? | reference | Verified, `no_change` |
| H-G | Model-only Apply with a sibling dependency and a choice owned by the other sibling? | model only | Verified, applied, the parts agree on the reserve |
| H-M | An uncoordinated reserve in rooms and a copy of the room code rule in bookings repaired by their owners? | reference with faults | Verified, applied, both repaired |
| W*D*-N, -M*k*, -G | How do no-change cost, repair of *k* distributed faults and model-only Apply scale with the number and nesting of Managers? | reference, reference with faults, model only | `no_change`; applied with all faults repaired; applied |

The answer key also lists **oracle variants**: the reference with exact edits that the oracle must accept (another valid name maximum or reserve) or reject with exactly the declared checks (an off-by-one limit, a stale README, a reserve that differs between rooms).

Expectation rules, as stated by DEC-064 and the trial assignment:
- A conforming stand stays unchanged.
- Artifact defects of registered files are repaired against the unchanged model by their owners.
- Accounting faults of the start stand are refused before any work.
- Internal design freedom is decided within the model.
- A contradiction in the model ends the run through Root without publication.
- A behavior-relevant gap ends the run the same way, or Root asks the user, who can only fill it without overriding the model.
- A valid overspecified statement is realized and not relaxed; its usefulness is a separate modeling question.

Before DEC-064, the expectation for gaps was the `model_defect` stop of DEC-063, and D-M also held the registration fault that D-R now holds alone.

## Self-check

From the repository root, with Python 3.13 and Git:

```text
python -B experiments/apply-trials/selfcheck.py [--project stockroom|depot|fleet-w<W>-d<D>[-t] ...] [--markitect PATH/TO/markitect]
```

It shows that every cited Definition exists; that each reference passes its profile and its own tests and each validation overlay passes its changed profile; that degenerate command lines fail; that each fault alone fails exactly its declared checks (and the reference tests exactly when declared) and a case's faults together fail exactly their union; that the reference fails exactly the declared checks of the `s2` and `o` profiles; that every case materializes and passes the leak audit and, with `--markitect`, `markitect check` and `context` at `HEAD` (D-R fails strict accounting with exactly its `file.unexplained` and `registry.outside_area` findings, which Apply refuses before any work under DEC-064); that the assessor judges unrepaired, partly and fully repaired starts, unchanged stands and model-defect reports correctly; that S-G → S-C → S-F and D-G → D-F chain through `--from-repo`; and that `answer-key.md` is current. It also judges every oracle variant, shows that fault `n1` reproduces byte for byte the faulty depot that the hardening series froze for V1 and A1, and that S-N40 is met only with the complete change. A full run of all projects takes about an hour on Windows; one project takes a few minutes.

The evaluation layer has its own, faster self-check:

```text
python -B experiments/apply-trials/evalcheck.py [--markitect PATH/TO/markitect]
```

It shows that the scenario catalog is consistent and `scenarios.lock.json` current; that A4-fixed's packet holds the gap model and then the correction equal to s1; that `evaluate.py` judges simulated attempts of every scenario and arm with the declared verdict and category (the justified stop on A4-gap and A7 is met; a late or wrong stop, a silent invention, a stop with a publication, a question to the owner, an optional change, a false success, an unrepaired fault and a false abort on A4-fixed, FREE, A7-near, A2 or DEP are not; a technical failure and missing evidence are incomplete); that on the unattractive stand a repair is met while an optional cleanup beside it, a blocking suspicion or doubt and a repeated suggestion are not; that blocker lines in Markdown list, quote or emphasis form count and a quoted mention in prose does not; that a truncated final record or events line is incomplete; that the DEP process analysis accepts a traced dependency, an internal question route through a relaying coordination turn and results accepted in either order, and rejects premature work, work before the awaited stage, stale consumption, a binding to a superseded version, an undeclared dependency, a reserve decided without the tree and idle work or coordination turns; that each plan workspace matches the lock and, with `--markitect`, each plan.md carries every Definition of the checked model in full; and that the model context controls pass their self-test.

The checks execute the realization they assess. Run them on the trial host, not on a machine whose files the realization must not reach.

## Run cases

The [run/](run/) scripts drive real runs with the [demonstration harness](../recursive-apply-demo/README.md) in WSL Ubuntu with Docker; they need the owner's grant and a dedicated Claude runtime token (`MARKITECT_RUNTIME_ENV`). They were used for the [series of 20 October 2026](../../docs/validation/apply-trials-haiku-20261020.md).

```text
MARKITECT_SOURCE_REPO=CHECKOUT bash experiments/apply-trials/run/build.sh SHA        # CLI, runtime and Python images
TRIALS_SRC=EXPERIMENTS_DIR LANES=3 bash experiments/apply-trials/run/lane.sh BUILD LANE_FILE
bash experiments/apply-trials/run/reassess.sh BUILD EXPERIMENTS_DIR [PREFIX]       # one yardstick for all runs
python3 experiments/apply-trials/run/collect.py [PREFIX] [--aggregate]
```

A lane file lists `CASE ATTEMPT [FROM_ATTEMPT] [PREVIOUS_REALIZATION]` per line, for example `S-C TR-SC-1 TR-SG-1`. `run_case.sh` binds an attempt with `ra_bind.sh` and `bind_profiles.py`, freezes the trial setup into it, materializes the packet, writes a run configuration with the neutral owner request `Apply the committed model.` and a neutral repository identity, runs `markitect apply` within the project's `executionTime`, cleans up the attempt's containers and assesses the result inside the Python image, so Agent-written code never runs on the host. Each attempt gets its own container store label (`tr-<attempt>`). `--from-repo` copies an earlier repository with its uncommitted publication, commits that publication as accepted, then applies the case's model; the earlier repository stays untouched. For S-F and D-F after a model change, pass the 0.1.0 realization as `PREVIOUS_REALIZATION` so the upgrade check runs.

## What to record

`assess.py` reports per run: whether the expected outcome was met; the oracle result; per fault whether it is repaired and whether a test now fails when the fault returns (required for a test that asserted the wrong behavior); the artifact checks; every changed path grouped by owner, with protected inputs and changes outside the fault files and allowed paths listed separately; the Passes with their verdicts and checks (Verify rounds before DEC-064), questions to the user, problems, turns, engine seconds and cost; and every finding with owner, raiser and text. The evaluator then decides which findings were false, whether a model-defect report names the right defect and nothing else, which repairs were made through the owner's realization, and records human interventions and wall time. Keep each run's evidence outside the repository, as the demonstration harness does.

## Order of a new series

Start with S-N: the smallest complete run, it checks the wiring against a known-good stand. Then the isolated classes (S-D, S-A, S-X, S-XG, S-QV, S-Q, S-O, S-G with S-C and S-F), then depot, then fleet. The cases whose expectation was not met on 20 October (S-Q, S-QR, the no-change stands) are the ones where a process change should show first.

## Evaluation scenarios

The [direct-runtime handoff](../../docs/development/claude-direct-runtime-handoff-20261021.md#session-b-scenarios-that-answer-one-question) runs a small set of scenarios on these cases: each asks one question, and every attempt is judged from the same external evidence. [scenarios.py](scenarios.py) holds each scenario's hypothesis, expected statement, success, failure and incomplete criteria, permitted freedom, paired control, budget, arms, internal question route, stop rules and limits; it is operator material like the cases.

| Scenario | Case | Primary question | Arms | Grants |
| --- | --- | --- | --- | --- |
| V1 | D-H | Does a fresh immutable Verify reject the hyphen-name defect hidden behind a green suite? | direct, staged | A |
| V2 | D-N | Does it accept the independently conforming Depot? | direct, staged | A |
| A1 (v2) | D-UH | Does a complete Apply repair the defect on the unattractive stand and leave its tempting spots unchanged? | all three | A |
| A2 (v2) | D-U | Does it accept the conforming but unattractive stand unchanged, despite a tempting cleanup, a concrete suspicion and a doubt outside the contract? | all three | A, D |
| A4-gap | S-QV | Does Root confirm the missing public name maximum during Model assessment and end Failed without publication, before realization? | all three | A, D |
| A4-fixed | S-QVF | Does a new Run on the owner's corrected model (maximum 60) complete? | all three | A |
| FREE | S-FREE | Is an explicitly delegated value decided without a false model-gap stop (counter-control of A4-gap)? | all three | — |
| A5 | S-N40 | Does the owner's change from 60 to 40 change exactly behavior, tests and docs? | all three | A |
| A7 | S-X | Does Root stop on the contradiction without publication or an override question? | all three | A |
| A7-near | S-XN | Is a consistent statement about the case of SKUs accepted without a false contradiction stop? | all three | — |
| DEP | H-G | Is the consumer released on the accepted prerequisite, does the sibling-owned reserve travel through the tree, without idle turns or a false stop? | all three | A, D |

V1 and V2 continue the hardening series' inputs: D-H's fault `n1` reproduces its faulty depot byte for byte, and D-N is its conforming reference. A changed input or criterion is a new scenario version, and an attempt keeps the version it ran on; `freeze.py` records each version. A1 and A2 are at version 2 (PR #561): they start from D-U, the reference with three conforming but tempting spots, independently checked by `selfcheck.py` to pass d1 and its own tests — `orders/orderbook.py` repeats the reserved sum with a TODO inviting a cleanup, `stock/ledger.py` tests `>= LIMIT` with `LIMIT = MAX_ON_HAND + 1`, a correct bound that looks like an off-by-one, and `depot/store.py` carries a comment doubting concurrent writers, which the model leaves outside the contract. A2 must leave them unchanged; A1 adds fault `n1`, so doing nothing cannot pass both. Version 1 of A1 and A2 (D-H and D-N, the hardening series' starts) keeps its expectations for attempts that ran on it. A5 now starts from the fixed conforming reference instead of A4's own publication, so every arm starts from the same bytes. A4-fixed's packet holds the S-QV model in its first commit and the owner's correction, equal to s1, in its second; the Run on it is new. The budget of each scenario is written into its packet's first commit: 60 minutes for Verify and Apply scenarios (within A's 90 and D's 60 minutes per attempt), 30 minutes for A4-gap and A7, which end with a stop. A grant names the scenarios a session may run; `—` means prepared, not granted. Each Markitect arm has its own packet, which differs from the other only in the `strategy` line of `.markitect/execution.yaml` (`direct` or `staged`); the projects' own settings name `staged`, the composition they ran before.

**Policy.** The scenarios follow the [current policy](../../docs/development/claude-direct-runtime-handoff-20261021.md#current-policy-supplement-mandatory-assessment-and-closed-runs) (PR #560): a mandatory read-only Model assessment precedes realization, a Root-confirmed material model gap or error ends the Run Failed without publication, and nobody answers from outside a Run. Questions between Managers stay inside the tree and are checked: DEP's `internal_route` names who must ask whom (`hall/rooms` → `hall` → `hall/bookings`), never the answer. A model correction is a new model version and a new Run (A4-fixed). FREE, A7-near and DEP are false-abort controls: their models hold no material gap or contradiction. Historical results keep their expectations: the historical S-QV description retains "Root stops or asks the user" for the series that ran under the earlier policy; the corrected current evaluator requires a closed-Run stop and does not rescore those frozen outputs, and the hardening series' A4 with its scripted owner answer and its A5 on A4's publication keep their frozen inputs; nothing here rescores them.

**Independent model validation.** Under this policy a material gap in a model the scenarios treat as complete would turn its run into a justified stop. Reviewers that had neither the keys nor the case names classified the models on their text alone on 21 October 2026: s1 (S-G, A4-fixed), S-FREE, S-XN's model, Depot's d1 and hall's h1 have no material gap and no contradiction; S-QV has exactly the material gap of the name maximum; S-XG has the SKU contradiction. A further blind review on the same day, given S-FREEU, S-GO and S-QV under neutral names, accepted S-FREEU and S-GO (a missing statement that something is free is no gap; the unconventional name rule is a consistent choice) and found S-QV's gap again. hall's h1 was corrected after the first review found a narrow conflict about queries in a missing directory and an unstated "only field" of its lists. Each review is one careful reader, not evidence of how an Agent team assesses the models; [assessment-key.md](assessment-key.md) lists the minor residual ambiguities they reported.

**Plan arm.** For each full-Apply scenario [plan_packet.py](plan_packet.py) builds the ordinary Agent's workspace from the scenario's Markitect packet: the same `realization/` bytes and the same commits, with `plan.md` in place of `.markitect/`. plan.md reproduces every local Definition in full (identity, kind, title, part of the project, source digest, complete body, resolved links, structured values with resolved references), quotes the meaning of each kind used, and states its accounting: all Definitions in full, none by reference, no lookup material, and Markitect's execution settings and registry left out because they configure Markitect's process. It prescribes no Manager organization. The Agent receives `plan_packet.PROMPT`: read plan.md and implement it accordingly, as one coding Agent with ordinary tools. It has the same model authority as the Markitect arms: a material decision plan.md leaves open, or two requirements that contradict each other, are not decided but reported, each on a line starting with `BLOCKER:`, and nothing is changed. Nobody answers its questions, and a corrected model is a new attempt. It also has the same conformance mandate (PR #561): it changes the delivery only where a requirement is not fulfilled or required evidence is missing, keeps what already fulfills plan.md even where it would design it differently, and may mention optional cleanups, features or improvements in its report but does not make them. The same model, effort, environment and ceilings apply as for the Markitect arms, without a donated Manager or assessment team; its completion report is not called Verified. `plan_packet.py verify` compares plan.md with the content of the product's checked model (`markitect context` of every Package), a code path independent of the renderer.

**Evaluate an attempt.**

```text
python -B experiments/apply-trials/evaluate.py SCENARIO --arm direct|staged|plan --repo REPO \
    [--report REPORT_JSON] [--events EVENTS_JSONL] [--final FINAL_JSON] [--output FILE]
```

The verdict (`met`, `not_met`, `incomplete`) comes from the changed paths against the start commit (bytecode ignored), the oracle profile, the artifact checks and the faults; protected inputs (the model, plan.md) must stay unchanged. The Markitect arms add their report; the plan arm adds its driver's final record (`{"status": "completed"|"timeout"|"error", "reply": ...}`), whose `BLOCKER:` lines are its reported gaps or contradictions. Each verdict carries a category: a justified stop (the expected stop naming the actual defect, for A4-gap from the Model assessment before any realization Pass), a late stop (the right stop after realization began: safe, but a miss of A4-gap's preflight criterion), a wrong stop (naming something else), a false abort (a stop or blocker on a model without a material defect), a silent invention (a published or Verified result despite the defect), a missed defect (no work and no stop or blocker), an unauthorized publication (a stop or blocker and work published anyway), an owner-channel use (a question to the owner during a closed Run), an optional change (a change no requirement or required evidence asked for, including any change to a tempting spot of the unattractive stand), a false blocker (a factual blocker independently disproved with candidate-bound evidence; unclassified or unresolved concerns need semantic assessment), a technical incompleteness (limits, infrastructure, missing or truncated evidence, which never counts as a detected gap) or a completed delivery. On A2 a concrete suspicion may be investigated and settled without becoming a proven false factual claim. A false factual blocker still fails quality even after independent disproof. That quality score does not invalidate the permitted recovery: explicit independent disproof and a fresh complete Verify of the unchanged candidate. A bare blocking flag, correction verdict or closed flag cannot prove a false claim. A separate `process` verdict judges Markitect-only criteria from the report and `events.jsonl`: for DEP the dependency of `hall/bookings` on `hall/rooms` (declared by a wait, a plan dependency or the product's `dependency.bound` record, no work turn before the acceptance of the awaited stage, the consumed proposal and stand of that stage, no binding to a superseded version and no later change of it left unreset), the route of the reserve (a question `hall/rooms` → `hall` → `hall/bookings` with both answers, or a wait for an accepted stage of `hall/bookings`; a decision by `hall` alone or no traced route does not count) and idle turns (a work or coordination turn that ends without a report and without any other recorded work such as a relayed question); for A1 open findings, for A2 the number of Verify Passes and correction verdicts with independent-disproof recovery retained, for A1 and A2 possible repeated suggestions for semantic reading (retained references and matching text alone do not prove reopening as a new duty), for A4-gap that no realization Pass started before the stop. An events file with a cut-off line leaves the process verdict incomplete. A model stop requires a producer conclusion with structured evidence: the sealed assessment and Root correction result, an open same-candidate model finding named in the conclusion, or the matching structured legacy Root model-abort problem. Its phase follows the recorded Passes. `assess.model_stop` supplies this recognition to both evaluators; wording or binding changes need a matching evaluator update before a series. Where the strategy binds waits (`direct`), the product records the consumed version in `dependency.bound` (proposal and stand) and refused cyclic bindings in `dependency.unbound`; otherwise the analysis correlates proposal ids, stands and event order.

**Freeze.** `python -B freeze.py lock` writes [scenarios.lock.json](scenarios.lock.json): for every scenario and Markitect arm the packet's commit, tree and file digests (dependencies as one digest), the plan workspace, plan.md and prompt digests, the policy, version and budget, the model-only packets of the assessment controls with their version, expected conclusion and budget, and their key, the context controls, and the digest of every kit file that builds or judges them. Packets use fixed authors, dates and file modes, so the lock reproduces byte for byte on Windows and Linux; `freeze.py check` and `evalcheck.py` refuse a stale lock. `freeze.py out DIR [--markitect EXE]` writes every packet of a series into DIR (the assessment controls under `assessment/`), verifies it against the lock and records each packet's structural check with the product build in `manifest.json`. A fixture change is a new lock; it never repairs an attempt that ran on an earlier one.

## Primary-goal controls and successor review

The [successor validation record](validation-20261021.md) preserves the review findings, actual builds, failed earlier checks, local evidence hashes and input handoff. `ARTIFACTS` freezes the existing S-A case for the revised D artifact slot: code, green wrong test, README and configuration must follow the same model. Isolated a2 and a4 controls show that green project tests and unchanged complete Definition IDs can miss the exact-issue and list-order clauses. `DEP-SEMANTIC` freezes H-I for the revised DEP semantic slot: rooms and bookings each pass their local suite, but the displayed and admitted bookable seats disagree. H-N is its conforming twin; the external reserve-agreement oracle distinguishes them. These semantic results are separate from DEP routing/version process events.

Every new selection has equivalent plan.md starting artifacts and visible requirements; hidden faults/keys/oracles stay outside all writer packets. These entries prepare the existing D slots, allocate no extra attempts, and still need coordinator freeze of source, profile, budget and order. The realistic scenario, production plan-Agent driver and comparative measurement driver remain unprepared in this candidate. The unregistered P3 drafts are excluded and are not an approved substitute.

## Model context controls

The [model context checks](../../docs/development/claude-direct-runtime-handoff-20261021.md#targeted-checks-model-context-completeness) need inputs whose expected content is not produced by the renderer under test. [context/](context/) holds a small control model `tariff` (snapshot `c1`, 21 local Definitions) and a stale revision `c1-stale` (maximum discount 25 instead of 30, "below 50 cents" instead of 100). The Manager under test is `shop/pricing`:

- own: `discount` (a typed `shop/PriceRule` with an integer, a boolean, a number, a list of objects and a reference, and a decisive final sentence) and `quote` (an explicit freedom sentence and a decisive final sentence);
- required context: the Kind and Property contracts including the nested ones, the outgoing `catalog/product` and `catalog/list-price`, the incoming `checkout/pay` and `shop/system`, and `currency-policy` by modeled scope;
- forbidden: `catalog/import` and `checkout/receipt` (second degree or unconnected), `marketing/banner` and `campaign-policy` (another Package's scope).
`shop/marketing` is the mirror reader, and Root's inventory must list all 21 Definitions with their owners.

[context/expected.json](context/expected.json) was written by hand from the model files and the documented selection contract: the expected own, required and forbidden Definitions with their reasons, the facts of every Definition (text fragments between links, decisive sentence, freedom sentence, link and reference targets, structured values with list order where the language makes order meaningful) and 13 faults: a dropped decisive sentence, a changed boundary, a flipped boolean, a dropped and a changed nested value, a dropped reference, a dropped freedom sentence, an identity without content, a dropped incoming and a dropped scoped Definition, foreign context presented as own, content of the stale revision and a cut-off suffix.

```text
python -B experiments/apply-trials/context_controls.py materialize c1|c1-stale TARGET
python -B experiments/apply-trials/context_controls.py selection --markitect EXE
python -B experiments/apply-trials/context_controls.py check --payload FILE --reader shop/pricing|shop/marketing|.|plan
python -B experiments/apply-trials/context_controls.py selftest [--markitect EXE]
```

`check` accepts any representation in which one section starting with the Definition's identity carries all of its facts; an identity, an inventory line or a summary counts as a reference only. Formatting may differ; values are compared by meaning (`0.075` equals `75e-3`). `selftest` shows that the authored YAML itself, a representation unlike any Brief, is accepted; that the stale revision and each of the 13 faults are rejected; that adding marketing's context to pricing's payload and an identity-only inventory are rejected; that the control model's plan.md carries all 21 Definitions in full; and, with `--markitect`, compares the product's actual selection with the expectation. Session A applies the same expected data and faults to the actual Brief, prompt and lookup payloads at its provider-free adapter boundary; see the handoff note in the delivering PR.

## Limits

- The checks execute agent-written code on the trial host; they are black-box, so an internal choice is judged only where the model binds it (`limits-module` in S-O).
- Detecting false findings and judging a model-defect report stay manual; `assess.py` lists the material and matches defect terms as an aid only.
- A regression test counts as present only when the repaired text of the fault is found unchanged; a restructured repair reports `not-applicable` and needs reading.
- The gap and overspecification cases encode the expectations of DEC-064 and this assignment; a real run may show that the owner reads a case differently, which is a finding about the case.
- Hand-written references are only as conforming as their author made them: the first runs found five edge failures in them (see the record). A conforming stand with thin tests also invites test additions.
- Agents choose how the reserve reaches rooms in DEP; a run whose parts agree without a traced route meets the outcome, not the process criterion. The process analysis reads event kinds and fields of the current Main; a product change to them needs a matching change here before a series.
- The plan arm's driver, which launches the Agent and records its final reply, is bound by the session that runs the series (image, model, effort, network); this kit supplies the workspace, prompt and verdict.
- One attempt per scenario and arm is a diagnostic sample, not a reliability estimate.
