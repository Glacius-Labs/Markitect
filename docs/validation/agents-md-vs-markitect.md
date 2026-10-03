# AGENTS.md versus Markitect: matched MyMeetings comparison

The overall product decision is **D: inconclusive**. The simple approach produced credible implementation candidates without Markitect. Markitect demonstrated a useful, explainable policy-migration workflow, but this study did not establish that its additional maintained model pays for itself. No Core feature, release, or broad product claim follows from these results. The next step is a repeated adopter study with stronger instrumentation and matched policy history.

This is new evidence; the historical [real-code pilot](real-project-adoption-pilot.md) and its negative findings remain unchanged. Results distinguish source inspection, executable checks, agent reports, operator work, and review proxies. There was no human acceptance, database execution, runtime integration verification, measured productivity gain, or token measurement.

## Integration and frozen protocol

[PR #58](https://github.com/Glacius-Labs/Markitect/pull/58) integrated the authorized policy-analysis candidate `1996ef2106e2df6aa8f5955359fa4a5542004799` at `a23db5f48854cbf7723789843aaa10ddf06cdb0b`. The integrated tree matched the reviewed head. The head-bound [Windows/Linux CI](https://github.com/Glacius-Labs/Markitect/actions/runs/37126319833) and [main CI](https://github.com/Glacius-Labs/Markitect/actions/runs/37126611702) passed. The actual public diff was independently reviewed; that agent review is not a human GitHub approval. [Integration evidence](../../experiments/agents-md-comparison/integration.json) records the identities. No release was published; v0.12.0 remains the published release.

The [design](../../experiments/agents-md-comparison/design.md), [knowledge-parity table](../../experiments/agents-md-comparison/knowledge-parity.md), [three task prompts](../../experiments/agents-md-comparison/tasks/), [hidden oracles](../../experiments/agents-md-comparison/oracles/), guidance and evaluator were frozen before dispatch. The frozen design is preserved as written, including its preparation-status text; this report owns completion and protocol deviations. The R2 manifest SHA-256 is `e085ec47e2da0a62e07bf1d526be5dab9a35c7c53cf9d5e7cde185cb31125c3e`; its oracle freeze is `819f127c60ed4ee19ad0d9e7f2fc8cd5a6290dfa7de6baeb5a4c47b17027f74e`.

Both arms received the same task body and identical 646-file public MIT-derived source subset: 711,461 exact working-input bytes from upstream `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8`. The preserved bundle SHA-256 is `af9dbdc750f376b5250b61a451c1d124fe363fd87cce87684804b292149311cc`. Package/source v1 and v2 code inputs were unchanged, with exact package archives retained. Git-normalized blobs and working-file bytes can differ in line endings; the manifest binds the actual working inputs, not an assumed newline convention. Fresh trial repositories exposed no historical completed task patches.

Arm A received a realistic 29-line, 2,666-byte [AGENTS.md](../../experiments/agents-md-comparison/guidance/AGENTS.md), existing ADRs/source/tests, and a short adopter policy note. It links the read-model and module-integration decisions rather than repeating ADR prose. Arm B received the existing canonical constitution, exact package, and compiled task-specific Context; Task 2 also received Impact. Missing guidance was not manually inserted after compilation. The operator rebound Task 1's historical Skill task text before freezing because it otherwise requested a different task. That duplicated prompt and one prepared Skill projection are counted as setup/authoring work.

Both arms received the same bounded existing-test facade. The extracted upstream test project is incomplete; the facade links each candidate's project-owned `ApplicationTests.cs` with the same public ported base. The hidden evaluator separately adds two Task 2 behavior checks. The frozen evaluator README inaccurately describes an upstream-path test link: the actual frozen project links `$(AdopterRoot)/.../ApplicationTests.cs`, including candidate additions. This erratum does not change the frozen files. No check here represents the full upstream test suite.

There is a policy-history imbalance: A's Task 2 input contains the current v2 note; B receives a precomputed v1-to-v2 comparison. Both know the same selected cohort and field-level implementation goal, but historical change explanation is not matched. That limits comparative policy-evolution conclusions. The Task 2 oracle's generic allowed-source paragraph also conflicts with the specific predeclared design/parity allowance for narrow A-side test additions. Evaluation uses the specific frozen allowance and records the conflict; it does not revise the oracle after seeing outcomes.

## Environment, isolation, and available measurement

The local Markitect binary was built from clean integrated source `a23db5f`, Go 1.27.1, Windows amd64, SHA-256 `6aae29acf67217b9538a2076ff199102b44c84183f2e0dff4718584e84e064df`, with `vcs.modified=false`. Its embedded version is 0.12.0; it is an unreleased source build, not the published v0.12.0 binary. Policy-failure analysis is therefore evaluated as integrated source behavior.

Six scored runs used fresh collaboration agents with `fork_turns=none`, no model override and no shared conversation. The CLI was unauthenticated, so CLI JSONL metering was not used. Backend model identity, model calls, complete tool-call/search logs and token counts are **unavailable**. The helper records only content reads performed through it, including repeated and verification reads. A compiled Context embeds other files, so one Context read is not comparable to one source-file read. Neither read-event counts nor bytes measure architectural rediscovery, irrelevant-context use, tokens, or productivity.

Independent operator evaluation used .NET SDK 8.0.418, net8.0 and the same public NuGet source for all candidates. Agent self-verification is separate: Tasks 1/2 reported 8.0.418; Task 3 A reported 10.0.103 despite the requested `src` SDK selection. Raw execution telemetry is unavailable to explain that discrepancy. Operator SDK evidence must not overwrite the agent report. Initial feed/restore failures and later public-source successes are different executions, not contradictory results. Timings in raw logs are uncontrolled and are not compared.

Persistent-file audits hashed the trial and known original/parent/unrelated Markitect checkouts before and after each agent. None of the scored runs had a detected out-of-checkout persistent change. Helper, executable and compiled packets stayed unchanged. A's Task 2 policy copy and B's Task 1 canonical Skill changed and are explicitly reported as guidance/maintenance changes. The audit excludes build caches, ignored scratch/tool directories and Git metadata, skips reparse entries, and cannot prove absence of transient writes or access to unknown OS paths.

The initial R1 attempt was excluded after a read-helper defect rejected valid relative reads. Its abort was preserved; fresh R2 agents used the corrected helper. Pre-dispatch harness fixes and re-freezes are operator setup, not agent performance. R3 whole-run finalization reused the exact already-recorded first-agent pre-run audit after collection because relocation had not created its extra alias; it did not manufacture a new baseline. The original whole-R2 audit began before the final two-file harness/manifest correction and Task 1 prepared Context/projection. Its four additional deltas are accounted for separately from agent changes.

Auto-review then rejected the first R2 Task 3 A patch because its temporary adopter checkout was outside writable roots, and prohibited workaround edits. That agent made no source changes; the run is preserved and unscored. Two new Task 3 checkouts were prepared within the permitted ignored workspace under a separate R3 protocol, and both arms used new agents. Source bytes, tasks, oracles, facade and compiled Context were reverified before dispatch. Git line-ending/index preparation failures are retained as operator preflight work. The two runner changes only restrict the scratch root and record common inherited AGENTS instead of rejecting the location. Common parent guidance now affects both R3 arms; one agent encountered its absent `docs/README.md` link. This is an environmental confound, and Task 3 is not a causal cross-task comparison.

## Task 1: add a people-count Query

The task asks for one person per returned attendee row plus its guest count, with NULL guests treated as zero, through an immutable Query and one internal Handler. No HTTP endpoint is requested.

A added exactly two C# files. Its scalar SQL uses the existing attendee view and MeetingId predicate, with nested `COALESCE` to handle NULL guests and an empty result. B added the Query/Handler, made the existing attendee DTO's guest count nullable, created two canonical records and linked the new UseCase from the Skill. B dispatches the existing attendee Query through MediatR and sums in memory.

Both compile and pass the ten bounded architecture tests and the independent project-reference check. Source inspection supports the requested arithmetic; no database/run-time execution establishes it. B does not satisfy the oracle's scalar-query expectation and changes the DTO outside its listed allowance. Its new use-case-to-use-case source coupling needs architectural judgment and is not proved absent by its canonical policy results. A duplicates the view/predicate, introducing a possible future synchronization obligation. Neither tradeoff was repaired after the run.

B left generated projections unchanged because it considered isolated `render --write` outside its trial-write boundary. Its post-run strict `check` exits 1 for missing/drifting projections, while `model` exits 0 and ordinary policies pass. These are distinct states: its model is not a proof of completed governance or acceptable source coupling. The agent's feed-blocked self-checks did not compile; later operator checks passed against unchanged candidate source.

| Dimension | A: simple | B: Markitect |
|---|---|---|
| Functional correctness | Arithmetic supported by source; build passes | Arithmetic supported by source; build passes |
| Architectural correctness | Listed placement/conventions and scalar-query oracle satisfied | Placement/conventions pass; DTO scope and Query coupling require review; scalar-query oracle unmet |
| Context completeness | ADR 0009 directly navigable and read; source/view/table inspected | Typed ownership/slice available; compiled packet omits ADR 0009, DTO and table; no recorded separate ADR 0009 read |
| Context noise | Inline boundary/validation navigation broader than the Query task | Entire Domain, Core/project inputs and unrelated constraint definitions also supplied |
| Implementation effort | 2 manual artifacts | 6 manual artifacts; no reliable effort/time comparison |
| Governance upkeep | No guidance update | 2 new resources plus Skill link; projections remain incomplete |
| Review proxy | Inspect 2 new source files, SQL/null semantics | Inspect 3 source + 3 canonical files, coupling/scope and projection failure |
| Impact quality | Not evaluated as a separate task result | No task-completion Impact claim |
| Reconciliation | None | No completed post-change projection convergence |

## Task 2: repair the selected-Command Validator migration

Both candidates added internal FluentValidation classes for `AddMeetingAttendeeCommand` and `CancelMeetingCommand`, enforcing `MeetingId.NotEmpty()`, without changing Commands, cohort, package pins, archive bytes or integration code. A added two project-owned presence/behavior tests; B added one combined test plus two Validator resources and two UseCase links. Hidden behavior checks confirm empty GUID rejection and nonempty GUID acceptance. These runtime unit checks do not establish dependency-injection registration or business correctness.

A also promoted its frozen policy note from presence-only to field-level behavior and removed the caveat. It is source-relevant but outside the frozen guidance allowance. Preserve this deviation for review; do not silently accept it as an approved policy change. The narrower predeclared test additions were permissible. Neither arm weakened existing assertions.

B's prepared failing candidate emits two failed Validator PolicyResults. Ordinary `check` remains exit 1. Explicit analysis Context and Impact emit useful marked output and also exit 1, without an exception. Context selects AddMeetingAttendee and supplies its current failing rule; Impact identifies CancelMeeting as the other direct subject. After implementation B's strict `check`, `model` and `render --check` pass. No waiver was created. A demonstrates failing-before/passing-after through its project-owned tests and identifies the same cohort from its note/source.

Impact reports four changed result identities: the added selector guard and Validator rule for each of the two subjects. Exactly two new Validator outcomes fail; four deltas do not mean four failing Commands. The direct subject count is **2** and conservative affected count is **69**. The recorded causes are `configuration: markitect.yaml` and the Project resource change in that same file. The exact package binding change conservatively broadens the set. The other 67 resources are outside the two-Command implementation cohort; they are not automatically 67 erroneous review inclusions, because contract review can legitimately be broader. No smaller human review set was demonstrated.

The integrated CLI/application negative tests, run with the normal Go gates, confirm unresolved references, malformed relations/traversal and other structural errors still block analysis. This was product verification outside the agents' migration; it did not mutate a trial or turn structural failures into exceptions.

| Dimension | A: simple | B: Markitect |
|---|---|---|
| Functional correctness | Both validators pass hidden behavior checks | Same |
| Architectural correctness | Internal validators and retained checks; policy-copy scope deviation | Explicit resource ownership/cohort retained; policies pass |
| Context completeness | Selected names/current policy and source/test patterns available | Failed rule, source-bound provenance, Context and historic result delta supplied |
| Context noise | Generic Query/boundary instructions unnecessary to this migration | Read-model ADR/view in Command Context; Domain includes unrelated definitions; broad Impact |
| Implementation effort | 4 manual artifacts, including out-of-scope note | 7 manual artifacts |
| Governance upkeep | 1 test file and policy-note edit | 4 resource edits; 74 generated views; no package edit during remediation |
| Review proxy | 2 validators, tests and note scope | 2 validators, tests, 4 canonical records and verified projection ownership |
| Impact quality | Cohort from note/test selection; no supplied historical delta | 2 direct subjects, 69 conservative resources with explicit causes; unmatched history input |
| Reconciliation | No generated target | Agent regeneration plus `render --check` converged |

## Task 3: expose fee-paid status across an existing module boundary

Both candidates changed exactly the existing attendee SQL view, DTO and Dapper QueryHandler to expose the boolean Meetings-owned `IsFeePaid`. Neither changed Payments Application, the integration route, table schema, project references, ADRs/tests or added an endpoint. Source inspection follows the existing integration event, Meetings consumer and mark-paid command; the actual payment-to-read-model runtime remains unexecuted.

B additionally updated three canonical summaries (Handler, UseCase and attendee-view Narrative) to mention the new field, then regenerated their three projections. These are actual maintained prose updates, not new ownership/source inventories or Domain edits. They were optional descriptions of the same implementation change; do not assume every future source edit necessarily requires all three. Strict `check`, `model` and `render --check` pass. Both independent operator builds, ten-test architecture runs and the 18-project reference check pass.

A read ADRs 0009/0014/0017 and the implementation path. B received typed ownership, SQL factory/view and CQRS/test narrative, but its packet did not include ADR 0009, ADR 0014, the table, DTO or event path. It inspected the existing event/command/domain files separately and also read the large source README while navigating governance. Both encountered the common inherited documentation-map link absent from this extraction. No missing rule was manually supplied by the operator.

| Dimension | A: simple | B: Markitect |
|---|---|---|
| Functional correctness | Stored boolean exposed through view/DTO/query; source/build evidence | Same |
| Architectural correctness | Existing Meetings state/event boundary preserved | Same; declared graph remains passing |
| Context completeness | ADR links plus source inspection cover boundary | Typed local read-model context; integration semantics still rediscovered from source |
| Context noise | Common parent map link missing in extraction | Same missing map; full Domain/project inputs and source README also inspected |
| Implementation effort | 3 manual artifacts | 6 manual artifacts |
| Governance upkeep | No guidance edits | 3 canonical summary edits, 3 regenerated projections |
| Review proxy | Inspect 3 source changes and existing integration path | Same source/path plus 3 canonical summaries; projection check succeeds |
| Impact quality | Independent project references unchanged | Same observed references; no task-specific Impact accuracy claim |
| Reconciliation | None | Render output converged under `render --check` |


## Actual measured artifacts and context

| Run | Helper read events | Metered bytes | Manual changed artifacts | Generated changed artifacts | Independent tests passed | Project-reference check |
|---|---:|---:|---:|---:|---:|---|
| T1 A | 16 | 21,386 | 2 | 0 | 10 | Passed |
| T1 B | 21 | 76,291 | 6 | 0 | 10 | Passed |
| T2 A | 16 | 32,275 | 4 | 0 | 14 | Passed |
| T2 B | 19 | 96,233 | 7 | 74 | 13 | Passed |
| T3 A (R3) | 18 | 27,149 | 3 | 0 | 10 | Passed |
| T3 B (R3) | 25 | 194,573 | 6 | 3 | 10 | Passed |

Each independent build passed. T2 totals contain A's 12/B's 11 candidate architecture checks plus two hidden behavior checks; they are different test arrangements, not different numbers of defects. Telemetry files, build outputs and operator scaffolding are excluded from manual/generated change counts. Generated *file changes* are not individual plan-operation counts.


Read counts include duplicate and post-edit verification reads. Bytes are actual helper-reported input bytes, not tokens. T1 B's supplied Context is 53,013 bytes / 26 inputs (13 resource records, 12 opaque file inputs, one Domain); T2 B's Context is 43,712 bytes / 17 inputs plus 18,319-byte Impact; T3 B's Context is 48,434 bytes / 23 inputs. Counts include the Project policy record. They do not establish that a reader consumed or needed each embedded statement.

| Context category | Simple delivery | Markitect delivery | Observation and limit |
|---|---|---|---|
| Required and present | Query conventions/ADR 0009, module/event ownership navigation, current Validator cohort | Typed Module/Feature/UseCase/Handler identities; declared rules, package source/digests; Task 2 failures and two direct subjects | Both still need actual implementation source. B's provenance makes supplied typed inclusion explainable. |
| Required but missing from supplied packet | Detailed implementation semantics remain in linked source; no historic v1 policy packet for Task 2 | ADR 0009; Task 3 ADR 0014/event path and table/DTO; validator field-level behavior not encoded by presence constraint | Accessible source is not the same as supplied compiled Context. Missing content remains evidence. |
| Present but irrelevant or of unproven relevance | Boundary/Validator navigation during a local Query addition | Command Context includes attendee view/read-model decisions; Query Context includes full Domain and several Core project files | Qualitative coverage observations; no audited count of all irrelevant reads. |
| Duplicated | Query/Handler summary overlaps ADRs/tests; two Task 2 command names overlap tests | Canonical source inventories repeat file-to-concept mapping; task text repeated in Skill; declared Validator cohort overlaps implementation checks | Generated projections are copies owned by canonical sources, not additional independent truth. |

## Maintenance, review and falsification

The simple arm maintains the 29-line map and its short policy copy in addition to ADRs/tests/source already maintained. B starts with 66 local canonical files (65 architecture resources plus one Skill), Project/package bindings, one effective packaged Domain, two pinned archive versions and the declared project-reference check's mapping. The unused older archive in a checkout is historical evidence, not a second active policy. Inventory's empty local `domainPaths` means the Domain is package-resolved, not that B has no Domain. The adapter maps 18 explicit projects; observed ProjectReference evidence covers 35 edges. It proves those declared project references, not absence of runtime coupling or other dependency mechanisms.

No authoritative ADR, original test, source declaration or simple policy owner was removed from the project because Markitect existed. Canonical source ownership still duplicates some mapping. T1 adds model/Skill upkeep; T2 adds resource links and generated reconciliation. The package update was operator-prepared once from exact historical versions, then held fixed during agent remediation. There is no cost estimate for authoring the original Domain or future recurring maintenance. Generated files are not counted as manually maintained truths; their operations/consistency checks are counted as workflow work.

Independent fresh reviewers evaluated sealed candidates, not the agents' confidence. They are reviewer proxies, not humans. Artifact counts describe inspectable changes, not minutes of review. Deterministic builds/tests can replace redoing the same bounded calculation when trusted, but both approaches have tests. B's PolicyResults verify declared links/cohort and B's renderer checks verify projections; they do not verify C# validation behavior, SQL, dispatch registration or source-level architecture. No unique concrete human review step was observed removed. T1 B added coupling/scope/drift review; T2 A added a policy-copy scope question. No operator source correction was applied after evaluation.

| Falsification hypothesis | Evidence | What remains unproved |
|---|---|---|
| Short guidance plus tests gives most ordinary implementation benefit | A produced viable source candidates with fewer manually changed artifacts in Tasks 1/3 | General equivalence or a measured fraction of upkeep cost |
| B's value concentrates on policy evolution | Task 2 exposes exact failing rule/source, noncompliant read-only analysis and direct result transitions | That this capability pays for its package/resource upkeep; A lacks matched historic input |
| Context is explainable but not demonstrably smaller | Explicit input `reason`/`via`, package provenance; tens of KB of compiled input and missing ADRs | Token savings or reduced cognitive load |
| B catches governance drift | T1 strict check detects missing/drifting generated outputs | That this eliminates source defects; it is also new drift to maintain |
| Modeling adds duplication | No existing authority removed; resource/source mappings and cohort/test overlap remain | Whether future adopter consolidation removes an independent truth owner |
| Conservative Impact fails to shrink review | 69 overall affected resources despite two implementation subjects | Which of the other 67 genuinely need contract review, and actual reviewer time |
| Versioned contracts/exceptions offer different governance | Exact package pin/result provenance and reproducible analysis shown; no exception needed | Unique benefit over Git/tests or temporary waivers in this matched study; exceptions were not exercised here |
| Additional upkeep can exceed avoided sync | More manual B artifacts and projection work; A required no model edits | Monetary/time comparison across a sustained project lifecycle |

## Decision questions

1. **Ordinary implementation: what extra did B provide?** Typed ownership/slice context, explicit rule/package provenance and generated-governance checks. Source files, ADRs and normal tests remained necessary.
2. **Did that materially improve the outcome?** Not established. Both produced plausible functional candidates; T1 B introduced scope/coupling questions and incomplete projections.
3. **Less architectural rediscovery?** B supplied ownership up front; A navigated concise ADR links. Partial read logs do not establish an overall winner, and B still discovered missing source semantics.
4. **More irrelevant context?** B visibly supplied broad Domain/project/read-model content; A also supplied generic unrelated navigation. The qualitative excess is clearest in B's Task 2 read-model inputs; a total noise metric is unavailable.
5. **More manually maintained governance data?** B: resources, Project/package binding, Domain and source/check maps. Generated views are excluded from this count.
6. **Any independent source of truth removed?** No.
7. **Additional policy-evolution capability?** Yes: exact package/result provenance, direct PolicyResult delta and read-only noncompliant analysis. A uses current docs/source/tests and manual scope interpretation; its historic input was not matched.
8. **Did failure analysis improve migration?** It enabled the concrete failing-candidate inspection without an exception while check remained strict. Comparative effort improvement is unmeasured.
9. **Did Impact reduce human review?** No demonstrated reduction. It distinguished two direct subjects from an explained conservative set of 69.
10. **Any concrete review step removed by PolicyResults?** None uniquely demonstrated. They verify declared graph policy; code behavior still needs checks/review in both arms.
11. **Does benefit justify machinery at this size?** Not demonstrated; the simple approach remains the credible default for ordinary changes.
12. **Where does B appear more useful?** Reviewing versioned contracts, policy-failing migrations and multiple owned projections across adopters. This is a supported workflow hypothesis, not demonstrated market fit or lower total cost.
13. **Where does it appear unnecessary?** Small local source changes already governed by a few clear ADRs and architecture tests, without multiple synchronization targets.
14. **Strongest contrary evidence?** No truth owner removed, simple candidates work, B has missing Context and additional maintained records, and Impact remains broad.
15. **Strongest supporting evidence?** A policy-failing package change can now be inspected with exact source-bound results, repair Context and distinct direct subjects without inventing an exception.
16. **Build another capability?** No. Repeat the matched adopter/task study with common old/new policy history, consistent permissions/SDK checks and fuller metering; measure actual governance upkeep and review decisions before choosing a feature.

## Reproduction and evidence limits

[Experiment entrypoint](../../experiments/agents-md-comparison/README.md) identifies preparation, raw evidence aliases and replay. [Machine-readable results](../../experiments/agents-md-comparison/results/summary.json) retain run identities, counts, changed-file hashes, checks, Context inclusion and audit outcomes. The [raw index](../../experiments/agents-md-comparison/results/raw-evidence-index.json) distinguishes original hashes from presentation-normalized copies. Exact changed-file archives retain newly created files omitted by ordinary `git diff`; source MIT notices are included. Original raw operator artifacts remain locally preserved, not all publicly portable or committed. Aliases replace machine-specific paths in durable presentations.

The study has one pair per task, ordered rather than randomized, common facade/operator preparation, restore/permission confounds, partial read telemetry, no database and no repeated human review. Historical evidence and published package/release files remain untouched. Normal Go tests, vet and schema regeneration checks pass for unchanged integrated product source; commit-bound Windows/Linux gates above belong to `1996ef2`/`a23db5f`, not to a new unpublished study commit. New evidence/documentation receives its own content/hash/script checks. The first staged content check found presentation CRLF/literal patch whitespace; the commit script continued despite that failure. Presentation line endings were then normalized and the local commit rechecked. Original raw hashes, frozen inputs and exact changed-file archives were preserved.

Repeated difficulties belong chiefly to adopter source selection, authoring/projection UX, stale or duplicated mappings and specialized evidence coverage. [Study language-pressure observations](../../experiments/agents-md-comparison/language-pressure.md) classify them without a language expansion. Neither new graph operators nor a Pattern, Trait, inheritance/composition mechanism is justified. No public positioning claim stronger than these specific feasibility and workflow observations is supported.
