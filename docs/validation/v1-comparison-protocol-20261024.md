# V1 comparison protocol: Markitect against one agent with ordinary documentation

Date: 24 October 2026. Status: **approved as Phase A (Library); predeclared before any counting run.** A change after approval needs the coordinator's approval and a recorded reason before any run it affects. After the first counting run has been assessed, the only permitted change is the repair of an assessor infrastructure failure (section 5), and that repair is re-run on every candidate of both arms.

**Approval record.** On 24 October 2026 the coordinator approved this protocol at commit `fa6ef99720b73a948d6d20aa43cb6e877f006bd2` as Phase A and set the frozen V1 source F = `e11e6cff883ba16da2cddab8d54c0367cae8a5d8` (Main after C1). The assessment revision R is the Main merge commit that carries this protocol; its `tools/comparison` and `tools/library-oracle` are byte-identical to the approved commit, and the results record R. The coordinator also approved the P5 seed rule (the overdue assessment wrongly includes loans returned after the assessment date; step 4 drafts the exact diff for approval before the S4 controls) and **one deliberate deviation from its original brief**: the brief asked for fresh sessions in every run, but within a chain S2 and S3 continue in each arm's normal way (M keeps the chain's state directory, B resumes its Codex thread), because the binding measurement guide requires the method's normal continuation in a longitudinal sequence. A more demanding Phase B needs its own protocol and approval.

The study answers the comparison half of [DEC-072](../concepts/register.md#dec-072-stable-v1-justified-by-data) ([owner source](../concepts/sources/v1-evidence-goal-20261024.md)): is Markitect V1 better than one agent doing everything? It follows the [measurement guide](../guides/measurement.md#quality-first-comparative-evaluation), which is binding, and tests the thesis of [concept §1.4](../concepts/v1-concept.md#14-product-thesis-and-limits-of-evidence). The tools live in [`tools/comparison`](../../tools/comparison/README.md). The stability half of DEC-072 is the step-4 requalification of the [delivery plan](../implementation-plan.md); its Library Applies are this study's arm M.

## 1. Question and expected outcome

**Question.** On the reference scenarios of [concept §26](../concepts/v1-concept.md#26-reference-scenarios), with the same model, effort, maximal resources and an independent hidden assessment, does Markitect deliver more fully conforming first results than one coding agent that receives the same intended meaning as ordinary documentation?

**Honest prior.** Earlier single agents realized Library M1 with 12 of 12 oracle cases in two to five minutes ([archived capability record](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/runtime-capabilities-20261017.md#single-agent-baselines)). Plain behavior on a fresh M1 is likely a ceiling for both arms. The study therefore adds easy-to-miss obligations, a change, a no-change and a subtle seeded defect, and it treats a tie as a likely and legitimate result. Markitect is expected to be slower and to use more tokens.

## 2. Arms

| Arm | What it is |
| --- | --- |
| **M** | The public `markitect apply` command on the frozen V1 source F, run by the step-4 session exactly as its dispatch packets define (`examples/` runtime profile and run configuration, Apply deadline 2,700 s, 128 messages, 600 s per message, six Managers, fresh Runs). |
| **B** | One `codex exec` session in the same Go runtime image, started by [`runner.py`](../../tools/comparison/runner.py). Its repository holds `requirements.md` and, from S2 on, its own prior work. Its only instruction is [`prompt.txt`](../../tools/comparison/packet/prompt.txt). No second agent, reviewer, Manager or prescribed process. |
| B+ (optional) | A stronger conventional arm, for example B followed by one fresh self-review session. **Not run in this study**; if added later, it is labeled separately and never replaces B. |

The owner's instruction for B is quoted from the [owner source](../concepts/sources/human-attention-and-baseline-20261017.md#comparison-baseline): „Lies dir die Doku durch und implementiere das so“. The prompt uses its English translation, "Read the documentation in requirements.md and implement it accordingly", so that both arms work in one language.

### Shared conditions

| Condition | Both arms |
| --- | --- |
| Model and effort | `gpt-6-sol`, `high`, in every Markitect role and in B |
| Codex | 0.162.0 from the image's `runtime.env`; the same feature switches (`apps`, `memories`, `multi_agent`, `multi_agent_v2`, `image_generation`, `browser_use`, `computer_use` disabled; `agents.enabled=false`; project documents off; `/work` untrusted); approval `never`, sandbox `danger-full-access` inside the container; working directory `/work`. A self-test reads these from `internal/adapters/codexappserver/profile.go`. |
| Login | The step-4 dedicated normal account login (`dedicated-runtime-login`); every session receives a copy without its refresh token; the file's SHA-256 is checked before and after each run; neither arm can refresh or rewrite it |
| Image | The Go runtime image built from F in step-4 P0, bound by its engine image Id |
| Container hardening | Non-root `1000:1000`, read-only root, no capabilities, `no-new-privileges`, `--init`, engine default network, as `internal/adapters/container/spec.go` |
| Maximal resources | M: each Manager session 1 CPU, 2 GiB, 192 processes, 256 MiB `/tmp`, and up to six at once. B: the same aggregate in its one container: 6 CPUs, 12 GiB, 1,152 processes, 1.5 GiB `/tmp`. The engine has 16 CPUs and 31 GiB. Actual use is reported. |
| Tools | Codex's own shell and file tools, Go 1.27.1, Git; B has no Markitect tools, which are the treatment |
| Wall clock | 2,700 s for the whole M Apply and for the whole B session |
| Continuation | S1 and every S4 run start fresh. Within a chain, each arm continues in its normal way: M keeps the chain's state directory, so Notes survive, and starts its fresh Runs as always; B resumes its own Codex thread (`codex exec resume`). |
| Human help | None after start. The first delivered result is frozen before any human look. |

### Intended differences (the treatment)

Markitect adds the typed canonical model, the model-derived hierarchy of six Managers with Assessment, Implement and fresh Verify, independent reviews, the deterministic `project-tests` check before publication, registry accounting and guarded publication. Its sessions receive the Manager harness as base and developer instructions, which replace Codex's default base prompt; B runs with Codex's default base prompt. Its fixture carries Markitect control inputs (`execution.yaml` with ownership and the check, an empty registry) that B does not need; their preparation is setup attention for M.

## 3. Scenarios

All scenarios use the current Library model `testdata/library/m1` at F. There is no seed except in S4.

| ID | Scenario | Start of each run | Profile |
| --- | --- | --- | --- |
| S1 | Fresh M1 realization (§26.1) | M: the step-4 P1 fixture. B: a repository with only `requirements.md` (M1). No realization file in either. | m1 |
| S2 | M1 → M2, loan limit 5 → 3 (§26.2), longitudinal | Each arm's own S1 result, plus one commit: M changes only `maxOpenLoans` (step-4 P2); B replaces `requirements.md` with the M2 rendering, which differs in that one table cell | m2 |
| S3 | Unchanged re-application (§26.3), longitudinal | Each arm's own S2 result, unchanged | m2 |
| S4 | Seeded defect repair | One fixed seeded candidate, byte-identical in both arms outside `.markitect/` and `requirements.md`, as a single neutral root commit; M2 intent | m2 |

Longitudinal rules: S2 and S3 continue each arm's actual frozen prior result, including its defects, never a supplied correct project. If a prior step published nothing or left a broken tree, the next step starts from that actual state. A step is blocked only when no state can be captured; a blocked step counts as a non-conforming result and is reported as blocked.

**S4 base and seed.** The base is the first arm-M S2 candidate, in execution order, that passes C2–C4 below. If no M candidate qualifies, the first qualifying B S2 candidate is used, with an empty registry for M, and the extra registration work is reported. The seed is the step-4 P5 coherent seed class, with the rule the coordinator approves for step 4 (recommended: the overdue assessment includes loans returned after the assessment date, contradicting the last sentence of `lending/reminders/assess-overdue`). Code, its unit test and one usage-documentation sentence agree with the wrong behavior, so the candidate's own tests pass. Before any S4 run:

- the unseeded base passes C2–C4 (positive control);
- the seeded tree fails the oracle only in the E10 returned-loan assertions, passes the probes and its own tests (negative control);
- `runner.py same-tree` shows the M and B fixtures identical in every path, mode and blob outside `.markitect/` and `requirements.md`;
- the coordinator approves the exact diff, the list of seeded lines and M's registry texts, which must not describe the seeded behavior or the seed.

S4 inputs follow the step-4 P5 rules in both arms: unchanged M2 intent, a single neutral root commit with neutral branch and identity, no seed wording in any file, message, ref or Note, and fresh state for every run. Seed descriptions, diffs and locations stay outside both writers' inputs.

**M-specific observation.** When an M run in S4 or elsewhere takes the public Verify Reject → fresh Implement → fresh Verify → publication branch, it is recorded as an M-specific observation; step 4 counts such an S4 run as its P5 proof and runs a separate P5 only if none of the three does, which the coordinator decides. B has no equivalent and nothing about the branch enters the comparison.

## 4. Equivalent requirements

B's documentation is rendered from the canonical model by [`requirements.py`](../../tools/comparison/requirements.py). Every one of the 22 local Definition bodies of M1 appears verbatim under a plain heading. Body links become Markdown links to the linked section with a readable link text. The two policy values come from the `loan-policy` spec; the two Property Definitions are rendered as the policy table with their bodies and "Exactly one integer value is required." for their `type: integer, minCount: 1, maxCount: 1` spec. Four phrases that name the model itself are replaced, and each must occur exactly as often as stated or rendering fails:

| Definition | Model wording | Rendered wording |
| --- | --- | --- |
| `lending/LendingPolicy` | "States the numeric" | "The lending policy states the numeric" |
| `lending/loan-policy` (twice) | "in this definition's spec" | "in the table above" |
| `library/callable-interface` | "offers every operation of the model" | "offers every operation in this document" |
| `library/realization` | "must agree with the model." | "must agree with this document." |

Nothing else is added except the title, one introductory sentence, the headings and the table frame. `requirements.py check` re-renders the model, compares the committed bytes and confirms that every rendered body occurs in the document; the self-test also confirms that M1 and M2 differ in the one table cell and that no `[[` link remains. At R the committed documents have SHA-256 `1b3dc865af6ed24be2e76b283089be6a8ad89e9257609d58cd74bb84d6179e1b` (M1) and `d0de9dbd9272764715110f8dc9ce09765d625d239a21e5c98f289dcc6b361d0a` (M2); the prompt has `8a6c99a79b4854d7e7dae3158fbd43ef26beeaf57b8530bca758c65974c05eaa`. Before the first counting run, `check` runs against `testdata/library/m1` at F; if C1 or anything else changed a Library body, the documents are re-rendered and this protocol is amended before any run.

M's Managers also see the imported vocabulary (Concept, Rule, UseCase) and Core Definitions and the execution controls. These carry no Library obligation.

### Every assessed obligation in both representations

| Assessed obligation | Canonical Definition | Section of B's document |
| --- | --- | --- |
| E01 membership | `members/member`, `register-member`, `set-member-status`, `inspect-member` | Members: Member, Register a member, Set member status, Inspect a member |
| E02 copies and availability | `catalog/copy`, `availability`, `register-copy`, `inspect-copy` | Copies |
| E03 successful loan | `lending/loan`, `borrow` | Loan, Borrow |
| E04 admission limit, overdue loans count | `lending/loan-policy`, `maxOpenLoans`, `LendingPolicy` | Lending policy and its table |
| E05 suspension | `members/eligibility`, `lending/borrow`, `lending/return` | Eligibility, Borrow, Return |
| E06 fourteen calendar days | `loanPeriodDays`, `loan-policy` | Lending policy and its table |
| E07, E08 returns, repeated old return | `lending/return`, `catalog/availability`, `callable-interface` | Return, Availability, Callable interface |
| E09 loan inspection | `lending/inspect-loans` | Inspect loans |
| E10 overdue, including the returned-loan rule (and the S4 seed rule) | `lending/reminders/assess-overdue` | Assess overdue loans, last sentence verbatim |
| E11 invalid input | `library/inputs`, `callable-interface` | Shared input rule, Callable interface |
| E12 transport | `library/callable-interface` | Callable interface |
| M1/M2 witness | `loan-policy` spec `maxOpenLoans` 5 / 3 | Policy table value 5 / 3 |
| X01 due date after 9999-12-31 refused, 9999-12-31 admitted | `lending/borrow` | Borrow |
| X02 case-variant field name → `invalidRequest` | `library/callable-interface` | Callable interface, Wire format |
| X03 `null` required argument → refusal | `library/callable-interface` | Callable interface, Wire format |
| X04 missing or non-object `args` → `invalidRequest` | `library/callable-interface` | Callable interface, Wire format |
| X05 non-string or missing `id` echoed as `null` | `library/callable-interface` | Callable interface, Wire format |
| X06 dates only in years 0001 to 9999 | `library/callable-interface`, `library/inputs` | Callable interface, Shared input rule |
| R1 module, entry point, tests, usage documentation | `library/realization`, `callable-interface` | Delivery, Callable interface |
| R2, R3a code, tests and documentation agree | `library/realization` | Delivery |
| R3b required test topics | `library/realization` | Delivery |
| R4 no reset, test-only operation, clock, network or persistence | `library/callable-interface`, `library/library`, `library/inputs` | Callable interface, Library, Shared input rule |
| R7 no invented requirements | `library/library` (scope and exclusions), `library/realization` | Library, Delivery |

Three assessed duties come from Markitect's Manager harness, not from the model. B receives the same duties in its prompt:

| Duty | Markitect Implement harness (`internal/harness`) | B prompt |
| --- | --- | --- |
| Preservation (S2–S4, R5) | "first reconcile the existing realization with the checked model's obligations … Change artifacts only for those obligations, checks or necessary consequences of that work. Preserve conforming artifacts … Do not introduce optional coverage, refactoring or cosmetic changes solely to produce a diff. Repair genuine defects …" | "If an implementation already exists, first reconcile it with requirements.md. Change files only where requirements.md, its required build and test commands, or necessary consequences of that work require it. Preserve conforming code, tests and documentation; do not add optional coverage, refactoring or cosmetic changes solely to produce a diff. Repair genuine defects you find against requirements.md." |
| Protected requirement source | "Do not modify protected model/control inputs." | "Do not change requirements.md." |
| Unresolved intent | "Do not invent missing material model decisions. Report model_blocker with concrete Findings." | "report … any requirement you could not resolve." |

The prompt names `requirements.md`, not "the documentation", wherever it says what to reconcile against, so that in S4 the seeded README cannot pass as the requirement source.

**B is not handicapped:** it receives every sentence of the model, the same delivery and preservation duties, the same tools, image, maximal resources and wall clock, its own Git history and conversation, and the same freedom to build and test. The rendered document is the best case for equivalence; a hand-written document would likely be less exact, which favors B.

## 5. Assessment

Every candidate of both arms is assessed by [`assess_candidate.py`](../../tools/comparison/assess_candidate.py) at revision R, in containers without network. For M the candidate is the final published candidate commit from its retained bundle. For B it is the runner's capture commit: the whole working tree the writer left, without `.git`; files the writer's own `.gitignore` excludes stay out, except Go sources, module files and Markdown, which the capture adds so that an over-broad pattern cannot drop a delivered source. The raw tree is retained too. The start commit is the run's fixture commit.

| Step | What runs | Limits |
| --- | --- | --- |
| Oracle | The unchanged [container harness](../../tools/library-oracle/README.md#container-harness) with its evaluator exported at F, for the scenario's profile: 12 cases E01–E12 and the paired witness, plus the opposite-profile discrimination. The assessor refuses to run if `tools/library-oracle` differs between F and R. The record binds the candidate commit, tree and program SHA-256; it is the oracle of record for both arms, so step 4 runs no separate oracle on these candidates. | Oracle's own 2 s per request, 30 s per process; outer 1,200 s |
| Probes | [`probes.py`](../../tools/comparison/probes.py), X01–X06 above, built with the oracle's helpers and run through its transport and state snapshots; evaluator and probe bytes are hashed on the host and checked in the container | Same oracle limits; outer 300 s |
| Own checks | `go build ./cmd/library` and `go test ./...` from `realization/` on a cold cache, without network. Primary: the arm aggregate (6 CPUs, 12 GiB, 1,152 processes), 600 s each. Secondary: the `project-tests` check profile (1 CPU, 2 GiB, 64 processes, 120 s), which M's publication had to pass and B was never told about | As stated |
| Census | Every changed path against the start commit: comparable files, M registry files, protected inputs (M model and control files, B `requirements.md`), and build residue: any path that is a compiled executable on either side, whether added, rebuilt or deleted | – |
| Scope greps | Non-test Go files using `time.Now(`, `net` or `net/http`, or `os.WriteFile/Create/OpenFile`; input for R4 | – |
| Seed check (S4) | Which approved seeded lines remain verbatim | – |
| Contamination audit | Every transcript of both arms is searched for web access, Git remotes, GitHub, `library-oracle`, `probes` or the contract identifier used as a search term. A flagged run keeps its result and is listed with the hit; the conclusion states whether the reading of section 8 changes without the flagged runs. | – |
| Rubric | [`rubric.md`](../../tools/comparison/rubric.md) by blinded reviewers (below) | – |

**Blinded review.** For each candidate the assessor writes its tree and its start tree without `.git`, `.markitect/`, `requirements.md` and compiled executables, plus their diff, under a random token. Each token gets its own fresh Claude subagent (Sonnet, high effort) with the frozen review prompt in [`review-prompt.md`](../../tools/comparison/review-prompt.md), the current requirements text and the rubric; it sees no other token and is not told about arms, scenarios, a comparison or any seed. The writers use a different model family. [`rubric_check.py`](../../tools/comparison/rubric_check.py) then checks every critical finding mechanically: the cited text exists at the cited line of the candidate, or for a removal of the start tree or the diff; a quoted requirement is verbatim in the requirements; a claimed absence is true on disk; and a classified file is changed in the diff. For an S3 run whose start was not conforming, every file classified *optional* or *unjustified* is checked in the same way (`--s3`). A valid critical finding or classification counts only when a second fresh reviewer, with the frozen confirmation prompt, confirms it. The analyst does not re-judge. All review output is retained.

**Calibration before the first counting run.** The assessor runs on the retained native A1 candidate `8e2f24a79d49c66d43f785bfd0c6959d05cb083b` (tree `29f0dd29a0189b3f3f2fd9d8541e15a021b80682`) from the [V1 qualification record](v1-qualification-20261024.md), taken from `Documents\Codex\Markitect-S3-Qualification-Preparation\2e14fe25\m1-qualification-1ef8b910-recovery20\attempt-01\state\small-library\candidates\8b9b208b01d56ca75f416245aaf075c9\candidate.bundle`, with profile m1 (the oracle is expected to pass, as its earlier outside oracle did under older limits) and profile m2 (the oracle is expected to fail the witness). The candidate was built on the old contract and Core; it calibrates behavior assessment only. Its probe results are inspected: a probe that fails is checked against the requirement text, and only a probe that contradicts the text is corrected, before any counting run and with the correction recorded here. The S4 controls above are the second calibration.

**Assessor infrastructure failure** means an engine or container start error, or the harness ending before any case ran. A candidate's timeout, crash, refused build or wrong answer is a result. After such a failure the assessment of that frozen candidate is repeated; the writer run is never repeated for it.

Writers never see the oracle, the probes, the rubric, this protocol, the seed or any assessment result.

## 6. Outcomes

### Primary outcome: conforming first result

A run is **conforming** when all of the following hold for its first delivered result:

| | Criterion |
| --- | --- |
| C1 | Technical completion. M: the step-4 common Apply pass except the oracle (Apply passed, published or no-change readback, all Runs closed, cleanup without residue, login unchanged). B: the runner's status `completed`: exit 0 within the deadline, exactly one completed turn and no turn failure, capture, no residue, login unchanged. Codex stream `error` events are recorded but are not failures, as the Manager adapter tolerates retried provider errors. |
| C2 | The oracle passes all 12 cases and the opposite-profile discrimination for the scenario's profile. |
| C3 | All six probes pass. |
| C4 | Own `go build ./cmd/library` and `go test ./...` pass under the primary limits. |
| C5 | No confirmed critical rubric finding (R1, R2, R3a, R4, and an *unjustified* R5 change with a citation), and no protected input changed. |
| C6 | S3: if the S3 start was itself conforming by C2–C5, the comparable census is empty and, for M, so is the registry census; otherwise no changed file has a valid, confirmed *optional* or *unjustified* classification. S4: no seeded line remains. S1, S2: nothing further. |

Build residue is excluded from the census and reported as hygiene; M cannot leave it because of complete accounting. A login secret found in retained outputs quarantines that run's evidence until it is cleaned; it is not a criterion, because M's transcripts are redacted and B's are not.

### Error types

Every failed criterion is recorded as an error type: an oracle case id, a probe id, a rubric item, a C1 failure class (timeout, turn failure, Apply failed or incomplete, no-progress stop, capture failure, residue), a C6 failure, an unsupported success or a false blocker. The decision rule uses these types.

### Secondary outcomes

Reported separately per arm and scenario, never folded into the primary outcome:

- each criterion C1–C6 on its own, and the failing oracle cases and probes per run (missed obligations);
- **unsupported success**: M published (Verify approved) a candidate failing any of C2–C5; B's final message reports the work as done without naming the failing obligation as unresolved;
- **false blockers**: M reports a model blocker or rejects the valid model in Assessment; B names a requirement as unresolvable or ambiguous although the document states it;
- **invented requirements**: R7 count;
- unnecessary changes in S2–S4: R5 *optional* and *unjustified* counts, comparable files and lines changed;
- R3b test topics covered (0–6) and the own checks under the `project-tests` limits;
- S4: E10 result and seeded lines remaining;
- contamination audit hits;
- human interventions after start (planned: none);
- elapsed time (M: Apply process; B: session), and for the whole run including setup and cleanup;
- tokens: B from the Codex `turn.completed` usage; M where the retained transcripts carry provider usage, otherwise unavailable, never estimated;
- messages and sessions: M host messages, Runs and Sessions; B one input message and its command and file-change counts;
- M-specific: Verify Reject branches, Assessment and Verify Findings, Notes written.

## 7. Repetitions, order and uncertainty

**N = 3 per arm and scenario**: three longitudinal chains S1 → S2 → S3 per arm and three S4 runs per arm, 24 counting runs. Reason: the measurement guide asks for at least three before describing a trend, and three already cost about seven hours of serial engine time. With a likely ceiling on behavior, only a large difference can show; N = 5 would roughly double the cost without making a modest difference demonstrable. N is fixed: no extension, no early stop for success, no retry-until-green. A later study with a new protocol may follow.

**Independent units.** A chain depends on its first step, so the primary analysis uses 6 independent units per arm: each chain, conforming when S1, S2 and S3 are all conforming, and each S4 run.

**Order** (serial; one Docker engine and one account user at a time), counterbalanced scenario by scenario: in repetition r, each scenario runs as a pair, M first when r is odd and B first when r is even.

| Repetition | Runs in order |
| --- | --- |
| 1 | Calibration; M-S1, B-S1, M-S2, B-S2, M-S3, B-S3; then fix the S4 base, author and approve the seed and run its controls; then M-S4, B-S4 |
| 2 | B-S1, M-S1, B-S2, M-S2, B-S3, M-S3, B-S4, M-S4 |
| 3 | M-S1, B-S1, M-S2, B-S2, M-S3, B-S3, M-S4, B-S4 |

If no S4 base exists after repetition 1, all S4 pairs move behind the first repetition that yields one, keeping their internal order. Every engine use, writer runs and assessments alike, holds the shared exclusive lock `NATIVE-RUN.lock` in the session notes folder: created exclusively before the first engine or provider call, holding holder, run and start time, and removed only by its holder after the final census. Assessments run between writer runs when the lock is free; an assessment never overlaps a writer run.

**One attempt per planned run.** A run may be replaced once only for a documented infrastructure failure before the writer produced any change: provider unreachable or erroring, account limit or expired access, engine or host failure, shown by the transcript or logs and classified before the candidate is assessed. The failed attempt stays recorded. A Markitect runtime failure, an incomplete or failed Apply, a no-progress stop, a model blocker, or a B session that ran and did not deliver is a result, not an infrastructure failure.

**Uncertainty.** Counts k of n with exact Clopper–Pearson 95 % intervals per arm and scenario, per arm over the 6 units, and per arm over all 12 runs. For orientation: 3 of 3 has the interval 29–100 %, 6 of 6 has 54–100 %. The primary comparison is a one-sided Fisher exact test on the 6 units per arm. The 12-run counts and their difference are descriptive. No other tests, no subgroups, no post-hoc metrics; every secondary outcome is reported in full.

## 8. Decision rule

Predeclared reading of the primary outcome, with the secondary outcomes reported beside it:

| Result | Reading |
| --- | --- |
| One-sided Fisher exact p ≤ 0.05 on the 6 units per arm, M not below B in any scenario, and no error type in M that B does not show | **Supports** "better than one agent" on this Library suite, with the stated limits and the measured cost. This needs 6 of 6 against at most 2 of 6, or 5 of 6 against at most 1 of 6. |
| M ahead in conforming units, but the first row does not hold (including any error type only M shows) | A **tendency** in Markitect's favor, not established. |
| Equal conforming units | **No quality advantage shown.** If M is also slower or uses more tokens, the simpler baseline did as well for less, which counts against the claim. |
| B ahead in conforming units | **Counts against Markitect**: one agent delivered more conforming results. |

The following count against Markitect in every reading and are named in the conclusion: an M publication that fails the hidden assessment (unsupported success), M false blockers, M unjustified changes or a failed strict no-change where B has none, and M technical failures, which already count as non-conforming M results. Failures stay in the record. If the data do not show an advantage, the results document says so plainly.

## 9. Validity threats and mitigations

| Threat | Mitigation |
| --- | --- |
| The representations differ in content | Mechanical verbatim rendering with four listed substitutions and a check against the model at F (section 4) |
| B lacks the duties Markitect's harness gives | The same duties in B's prompt, quoted side by side; `requirements.md` named as the source |
| Different base prompts | Disclosed as part of the treatment (section 2) |
| Ceiling effect on plain behavior | Hidden probes for easy-to-miss obligations; change, no-change and subtle-seed scenarios; tie stated as a likely result |
| Small N and dependent chain steps | Fixed N; primary test on independent units; exact intervals; a rule that needs a large difference |
| Order and time effects (provider load, account limits) | Scenario-by-scenario counterbalanced pairs; start and end times recorded; quota pauses recorded |
| Assessor bias | Oracle and probes mechanical, fixed before runs and calibrated on a real candidate; one blinded reviewer per candidate, of another model family; mechanical citation check and a second confirming reviewer; the analyst does not re-judge |
| Arm hints in blinded trees (M's package layout, registry-shaped READMEs) | Binaries and arm markers stripped; objective, cited rubric items; limit reported |
| Capture drops or adds files | Whole tree captured, ignored delivery files force-added, raw tree retained; binaries classified on either side |
| S4 base written by an M run | Byte-identical for both arms, checked mechanically; neutral single root commit; neither writer has memory of it. It carries Markitect's package layout into B (its registry is dropped for B); this is a stated limit |
| Seed leakage through history, messages, registry texts or Notes | Single neutral root commit, neutral identity and branch, fresh state, registry texts in the seed approval, as step 4 learned from its earlier seed runs |
| Hidden material reachable by a writer | Writers see only `/work` and their home; `tools/` and evidence never mounted; both have network for the provider, so a contamination audit of every transcript is reported |
| Continuation differs between arms | Each arm continues in its own normal way within a chain, as the measurement guide requires; S1 and S4 start fresh |
| Transport differs (`exec` versus `app-server`) | Same binary, version, model, effort and switches; each arm uses Codex the way it normally would |
| Translation of the owner's instruction | Original quoted; one language in both arms |
| The analyst is not blinded | Mechanical computation by the predeclared rule; all raw outcomes published |
| Harness or environment failure mistaken for a method result | Classified per [measurement](../guides/measurement.md#evaluating-the-method-separately-from-its-runtime): runtime failures of Markitect are product findings; harness failures are limitations; replacement only by the rule in section 7 |
| Old evidence was helped by a visible seed message | Not reused; all candidates are new |

## 10. Human attention

The measurement guide counts authoring and maintaining both representations from one predeclared start. Here both derive from the same owner-confirmed business rules: M1's authoring effort is historical and unavailable, and B's document was rendered mechanically from M1, so its authoring cost in practice, where a person would write it, is also unavailable. Both are reported as unavailable, never as zero. M's execution controls and run configuration are setup for M; B's prompt and fixture are setup for B; the study's own preparation is research effort, recorded separately. During runs both arms receive no human input; every operator action (start, freeze, abort) and any intervention is logged per run.

## 11. Division of work

The step-4 session ("Step 4: native requalification") runs every M Apply with its controller and packets on F and retains its evidence; this session runs every B session, every assessment of both arms and the analysis. Arm M follows the step-4 packets unchanged except that, within a chain, S2 and S3 use the chain's state directory from the previous step: the controller copies it into the new attempt before any provider start, verifies the copy against a hash manifest of the source, excludes and records `temporary/` contents, and records the source path, its manifest SHA-256 and the previous Apply ID; the earlier attempt stays untouched. Per-run facts (fresh threads, tokens, Finding delivery) come only from that Apply's own transcript directory, `state/markitect-example/transcripts/<applyID>/`. Per M run, step 4 hands over its `run-summary.json` with the start bundle (path, SHA-256, commit, tree), the final candidate bundle (path, SHA-256, commit, tree), the fixture commit and tree, and the transcript directory. Its controller takes `NATIVE-RUN.lock` as holder `step4-armM` for build, Apply and any engine use, and refuses rather than waits when the lock exists.

Step-4 runs outside this protocol (for example the P4 ordinary seed) are stability evidence only and are not counted here.

## 12. Limits, abort and cleanup

| Item | Value |
| --- | --- |
| B session | 2,700 s; helper containers 120 s each; metadata 20 s; kill and remove 8 s. After a deadline, the session container must be confirmed gone, with one more removal attempt, before anything reads `/work`; otherwise nothing is captured and the run is incomplete with residue. |
| M Apply | The step-4 limits (packet 3,300 s, Apply 2,700 s, 128 messages, 600 s per message) |
| Assessment | Oracle 1,200 s, probe build and run 300 s each, own build and test 600 s (primary) and 120 s (secondary) each |
| Continue | Every outcome is recorded, failures included, and the series continues while the environment is clean: no residue, source, bound bytes and login unchanged. |
| Abort | Drift of any bound byte (packets, prompt, tools at R, image Id, login fingerprint); residue after cleanup; a failed census query; an assessor defect that makes runs incomparable; a Markitect defect that fails a run in a way that will fail every later M run. Everything is kept and the coordinator receives a diagnosis. |
| After an abort | An abort is part of this protocol's result; a new series never replaces it. If a Markitect defect caused it, every remaining planned M run counts as non-conforming with that error type, and the remaining B runs still run in their order. If the environment caused it, the series resumes on the same F in the same order once the environment is clean; if it cannot resume, the study is reported as incomplete with the data so far. A product fix belongs to a new study on a new source. |
| Account limit or token expiry | Pause and resume in the same order; the gap is recorded. Neither arm can refresh the login, so expiry is an environment limit for both. |
| Cleanup | B: the runner removes the login copy before capturing, then exactly its two labeled volumes; census before and after must succeed and be empty. M: the step-4 cleanup rules. Assessment: its own labeled volume. |

**Budget.** Expected serial engine time about 7 hours: M 12 Applies about 3.5 h, B 12 sessions about 1.5 h, 24 assessments and the calibration about 2 h. Hard ceilings, never all reached: M 12 × 55 min, B 12 × 50 min, assessments 25 × 40 min, about 38 h. Messages: M at most 128 per Apply (ceiling 1,536, expected about 500); B exactly one input message per run (12), its internal tool calls bounded only by the deadline and reported. Claude: 24 blinded reviews, one confirmation per valid critical finding, and one independent results review.

## 13. Retained evidence

Root `C:\Users\Consiliari\Documents\Codex\Markitect-V1-Comparison\<F first 12 hex>\`, outside the repository. Per B run: fixture bundle and its build record, `result.json`, every step's argv, exit, duration and logs, the JSONL transcript, the Codex session files, the raw working tree without `.git`, the capture bundle and `files.sha256` written last. Per assessment: `assessment.json`, oracle and probe reports, own-check logs, census, blinded copy, review and confirmation outputs and the citation check. M evidence stays where step 4 retains it. The results document, `docs/validation/v1-comparison-results-<date>.md`, lists every run, including failures and replacements, with its raw outcome.

## 14. What this study cannot show

It covers one small greenfield Library project, one model and effort, one provider and one machine. It cannot show benefits in larger projects, cross-area composition defects, sustained coherence over many changes, human-attention savings in real use, other providers, or reliability beyond the stated intervals. A positive result supports the thesis on this suite only; a tie or a negative result does not refute the method for larger projects, but it removes this suite as evidence for it.

## Addendum: Phase B

Status: **approved by the coordinator on 24 October 2026 at commit `4708c1b55aa90cdb6b95ecc3c4ec04d1e9c8b77e`, with the N = 5 option of B5, under four conditions:**

1. The two extra chains per arm run S1–S3 first. Their S1–S3 results are reported separately as additional observations and are not counted in Phase A's predeclared N = 3 analysis; only their T1–T4 segments count as Phase B units.
2. This addendum is frozen on Main by an exact-head merge before any Phase B run. Then the model snapshots are materialized, the renderer is extended and the hidden cases are written, and an independent review of them is recorded here or in the results before the first Phase B run.
3. Phase B starts only after every Phase A run is done and assessed, and keeps the same serial engine discipline with step 4.
4. If a stop condition or a systematic infrastructure failure uses up the budget, the study stops and reports instead of extending ceilings.

Phase A's text, tools and decision rule above stay unchanged; Phase B is read by its own rule and never pooled with Phase A. Before any Phase B data, B8 moved arm M's Phase B to a later source F′ and gave up the counterbalanced pairs for Phase B.

### B1. Why another phase, and which project

Phase A is likely to tie, because the fresh Library is small enough for one agent to keep every obligation in view. The [measurement guide](../guides/measurement.md#evidence-for-the-primary-goal) names the risks Markitect is meant to address: an omitted condition inside a covered rule, artifacts that disagree, local success hiding a composition failure, and coherence decaying over successive changes. Its prescribed test for the last one is a bounded sequence that continues each arm's own output "through creation, a cross-area model change, removal of a requirement and unchanged Apply".

Of the archived test projects ([archive index](../archive.md)), Readinglog and Roombook each have one domain Package, so Markitect's hierarchy would be one Manager below Root and no rule could span responsibilities. Library r2 is the same project as Library M1 with an equivalent model. Library is the only one with five Packages, cross-Package references and six Managers. Phase B therefore uses a **larger Library variant**: it continues each Phase A chain through four more ordinary product changes that each cut across areas, which is the guide's sequence on the one archived project with real cross-area structure.

### B2. Change sequence

Each step is a complete model snapshot, written as a canonical model change and rendered for B by the same verbatim renderer. Each chain continues from its own Phase A S3 result; S1–S3 plus T1–T4 make a seven-step trajectory per chain.

| Step | Ordinary change | Areas it cuts across | Easy-to-miss consequences |
| --- | --- | --- | --- |
| T1 (M3) | **Short-loan copies.** A copy has a loan class, `standard` (default) or `short`, set at registration; short copies are lent for 7 days, standard for 14. | catalog (copy, registration, inspection), lending (policy, due date), interface (optional `loanClass` argument, copy record field), delivery | The 9999-12-31 due-date limit now depends on the class; overdue notices for short loans; an invalid or wrongly cased class is a refusal; a missing class means `standard` |
| T2 (M4) | **Renewals.** An open loan can be renewed once, on or before its due date, which extends the due date by its copy's loan period. | lending (new `renew` use case, loan record, policy `maxRenewals`), members (eligibility: suspension also blocks renewal), reminders (no text change), library scope (renewals no longer excluded), interface (`renew` operation, `renewals` field in every loan record) | Overdue assessment must use the renewed due date although its Definition does not change; renewal by a suspended member is refused; the due-date limit applies to renewals; short copies renew by 7 days; the scope sentence "no … renewals" in documentation becomes wrong |
| T3 (M5) | **Removal of member suspension.** The library stops suspending members: there is no member status and no `setMemberStatus` operation. | members (member, eligibility; `set-member-status` deleted), lending (borrow, renew, return), reminders (suspended-member sentence removed), library scope, interface (member record, operations table) | `setMemberStatus` becomes an unknown operation (`invalidRequest`); borrowing and renewal are never refused for status; documentation and tests that describe suspension become stale |
| T4 (M5) | **Unchanged re-application** after the sequence | all | Strict no-change, as S3 |

Predeclared intended behavior after the removal (T3), as the guide requires: `setMemberStatus` requests return `invalidRequest`; member records need only `memberId` and `name` and may still carry a constant extra field, because responses may carry additional fields; unreachable suspension code may remain. Documentation that describes suspension as existing behavior, and tests that expect a working `setMemberStatus`, contradict the requirements.

The exact Definition edits are written in prose here and are materialized as model files only after approval, then checked by `markitect check` and against this table by the independent reviewer:

- **T1:** `catalog/copy` gains the loan class sentence and links the lending policy; `catalog/register-copy` accepts the optional class and refuses other values; `catalog/inspect-copy` returns the class; `LendingPolicy` gains the Property `shortLoanPeriodDays` (new Definition, value 7); `loanPeriodDays` is stated for standard copies; `loan-policy` states the due date per class; `callable-interface` adds the optional argument and the record field; `realization` asks for tests of both loan classes.
- **T2:** new UseCase `lending/renew` (open loan, member active, renewal date from the borrowing date to the current due date, fewer renewals than `maxRenewals`, new due date not after 9999-12-31; the due date extends by the copy's period from the current due date; count, availability and borrowing date unchanged; refusals without state change); `LendingPolicy` gains `maxRenewals` (new Definition, value 1); `loan-policy` states that a renewal replaces the loan's due date, which is then the loan's current due date; `loan` records its renewal count; `members/eligibility` adds renewal to what suspension prevents; `library/library` drops renewals from the exclusions; `callable-interface` adds the operation and the `renewals` field; `realization` asks for renewal tests. `lending/reminders/assess-overdue` is deliberately left unchanged.
- **T3:** `members/set-member-status` is deleted; `members/member` and `members/eligibility` lose status and suspension; `lending/borrow`, `lending/renew` and `lending/return` lose their status conditions; `lending/reminders/assess-overdue` loses the suspended-member sentence; `library/library` and `callable-interface` lose status, the status values and the operation; `realization` drops suspension tests.

No step is seeded. Every Phase B difficulty comes from an ordinary change.

### B3. Equivalence, conditions and fixtures

The Phase A conditions, arms, prompt, continuation, resources, lock, order principle and assessment apply unchanged. The renderer gains the new Definitions as sections; its check proves again that every body of each snapshot is in B's document verbatim, and the item table of section 4 is extended with every new hidden case family. M's fixture for step k is its own previous candidate plus one commit replacing the model snapshot; B's is its own previous capture plus one commit replacing `requirements.md`. The step-4 controller keeps the chain's state directory, B resumes its thread.

### B4. Hidden expectations

The unchanged Library oracle cannot judge M3–M5: T3 removes behavior that several of its cases exercise. Phase B therefore adds one assessor module in `tools/comparison` that reuses the oracle's case builder, helpers, transport and state snapshots without changing the oracle:

- **Carried cases:** the oracle's m2 cases and the six Phase A probes, cumulative at every step. For M5 they are transformed mechanically: steps calling `setMemberStatus` are dropped, the suspension case E05 is dropped, and member expectations no longer require `status`; snapshots are regenerated from the transformed steps. The same transform applies to the T1 and T2 families at M5.
- **New families, cumulative from their step on:**
  - T1: registration with and without a class, invalid and wrongly cased classes, 7- and 14-day due dates, the due-date limit per class (short 9999-12-24 admitted, 9999-12-25 refused; standard 9999-12-24 refused), overdue notices for short loans.
  - T2: one renewal admitted and a second refused; renewal on the due date admitted and after it refused; a suspended member's renewal refused until reactivation (M4 only); closed, unknown and invalid renewals refused; overdue assessment after renewal uses the renewed due date (the composition check); short copies renew by 7 days; renewals hitting the due-date limit refused; `renewals` present in every loan record; argument and unknown-field validation for `renew`.
  - T3: `setMemberStatus` returns `invalidRequest` and changes nothing; member records valid without `status`.
- Each family cites the sentence of its snapshot that it checks; the independent reviewer confirms the citation in both representations before any run. A self-test replays every expected transcript and one wrong transcript per family, as for the Phase A probes.

The rest of the assessment (own checks, census, blinded per-candidate review with confirmation, contamination audit) is Phase A's, with the rubric judged against the current snapshot.

### B5. Outcomes, N and decision rule

- **Primary outcome per run:** conforming, defined as in Phase A with C2 and C3 replaced by "all carried and new hidden cases for the step pass", and C6 strict no-change for T4. T3 judges stale documentation and tests through the rubric (R2, R3a) and the hidden cases.
- **N:** the three Phase A chains per arm continue, and two fresh chains per arm are added (approved option below). **Unit:** a chain's Phase B segment, conforming when T1–T4 all conform.
- **Rule:** "supports" needs one-sided Fisher exact p ≤ 0.05 on the units, M not below B at any step, and no error type that only M shows. Anything else is a tendency, a tie or against, read as in section 8, with the same error-type downgrade. Without the option, 3 units per arm would have needed complete separation, 3 of 3 against 0 of 3.
- **Secondary, never folded in:** conforming step counts per arm (k of 20; k of 12 for the three continued chains), missed hidden cases per run, stale obligations (documentation or tests still describing removed behavior), regressions of earlier families, unnecessary changes, time and tokens.
- **Approved option, N = 5:** two more fresh chains per arm (S1–S3 and T1–T4) give 5 units per arm, and "supports" then needs 5 of 5 against at most 1 of 5, or 4 of 5 against 0 of 5 (one-sided Fisher exact p ≤ 0.05). This costs 14 more runs per arm; the extra chains' S1–S3 results are additional observations only (condition 1). The extra chains run in the same counterbalanced pairs under Phase A's parity rule, which governs every chain: chain r has M first when r is odd, so chain 4 has B first and chain 5 M first.

### B6. Budget

| Item | N = 3 chains | With the option (N = 5) |
| --- | --- | --- |
| M Applies | 12, expected about 4.5 h | 26, about 9.5 h |
| B sessions | 12, about 2 h | 26, about 4 h |
| Assessments | 24, about 2.5 h | 52, about 5 h |
| Expected serial engine time | about 9 h | about 18.5 h |
| Hard ceilings | M 12 × 55 min, B 12 × 50 min, assessments 24 × 40 min: about 37 h | about 80 h |
| Messages | M at most 128 per Apply (ceiling 1,536, expected about 500); B 12 | M ceiling 3,328; B 26 |
| Claude | 24 blinded reviews plus confirmations | 52 plus confirmations |
| Preparation before approval of runs | three model snapshots, renderer extension, hidden-case module and self-tests, one independent review | same |

Phase B starts only after every Phase A run is complete, so it never delays Phase A. If Phase B also ties, that is its result.

### B7. Materialization record (condition 2)

Written after the addendum was frozen on Main (merge `1d60e38c1bf51c4e2c5b1b22524fcf574839b5f7`), before any Phase B run.

- **Snapshots.** [`tools/comparison/phase-b/`](../../tools/comparison/phase-b) holds one overlay per step on top of M2: `m3/` (9 Definition files, one new: `shortLoanPeriodDays`), `m4/` (9 files, two new: `lending/renew`, `maxRenewals`) and `m5/` (11 files and `deleted.txt`, which deletes `members/set-member-status`). `requirements.py materialize --profile m3|m4|m5` writes the complete snapshot for M's fixture; M5 has 24 local Definitions. `lending/reminders/assess-overdue` is unchanged from M2 to M4, as B2 requires, and the M4 `loan-policy` states that a renewal replaces the due date. Each materialized snapshot (M2 for reference, M3, M4, M5) passes `markitect check --revision HEAD` at F with the Library vocabulary, Core and the operator script's execution controls.
- **Documents for B.** The same renderer, extended by the new sections and policy rows, renders [`requirements-m3.md`](../../tools/comparison/packet/requirements-m3.md) (`e6454eaa6c6e248c663837c970d9b2b8c04594d092b54dcb98d7624dd4d47a54`), [`requirements-m4.md`](../../tools/comparison/packet/requirements-m4.md) (`fd5fa5e1565f9ab5051ba3aeeae5e630cfd1fc2d8c4e05d7fb0827d762b955ee`) and [`requirements-m5.md`](../../tools/comparison/packet/requirements-m5.md) (`058372a09c2be34835c5a49c1daecba2b055fd8d17ed89e62a5d29d53e994adb`); its check proves every body verbatim, and M1 and M2 still render byte-identically to Phase A's packets.
- **Hidden cases.** [`phase_b_cases.py`](../../tools/comparison/phase_b_cases.py) builds 21 cases for M3, 27 for M4 and 26 for M5: the oracle's 12 m2 cases and the six Phase A probes, carried, plus T1 (loan class, short loans, due-date limit per class), T2 (renew once, renewal window, suspended member at M4 only, closed and invalid, unknown field, short copies and the due-date limit; overdue assessment after renewal is the composition check) and T3 (no member status). For M5 the carried and T1/T2 cases drop `setMemberStatus` steps and E05 and expect no member status. Each family's docstring quotes the snapshot sentences it checks. The five-loan witness must still fail exactly at the fourth loan. The shared assessor runs these cases instead of the oracle harness and the probes for the profiles m3–m5, with the oracle's per-request and per-process limits, an outer limit of 1,500 s (Phase A's oracle and probe limits together) and the same own checks. The oracle's expectation files, which the carried cases read, are hash-bound on both sides like the evaluator files.
- **Wording kept to the frozen edits.** Two sentences that went beyond B2 were removed before the review closed, so that the easy-to-miss consequences stay consequences: M5's interface no longer says that a `setMemberStatus` request names an unknown operation (the operations table lacks it and the unknown-operation rule applies), and M4's `loan-policy` ends its renewal sentence at "which is then the loan's current due date", as B2 states.
- **Necessary follow-on edits** outside the B2 list, each a direct consequence of a listed change: M3 and M4 `LendingPolicy` wording for the added values; M4 and M5 `library` sentences about which records change ("renews", the edit exception for due date and renewal count, and in M5 "member suspensions" among the exclusions); M5 `register-member` ("creates exactly one member") and `inspect-member` (identifier and name only).
- **Fixtures for M.** Each step's commit replaces `.markitect/model/` wholesale with the materialized snapshot, so that M5 really deletes `members/set-member-status.yaml`; `model.yaml` and `model-lock.yaml` stay unchanged.

#### Every new assessed obligation in both representations

This extends the table of section 4, which stays unchanged.

| Hidden family | Canonical Definitions (snapshot) | Section of B's document |
| --- | --- | --- |
| T1 loan class at registration and inspection; invalid and wrongly cased classes refused | `catalog/copy`, `catalog/register-copy`, `catalog/inspect-copy`, `library/callable-interface` (M3) | Copy, Register a copy, Inspect a copy, Callable interface (records, `loanClass` sentence) |
| T1 7- and 14-day due dates; overdue notices of short loans | `lending/loan-policy`, `loanPeriodDays`, `shortLoanPeriodDays` (M3), `lending/reminders/assess-overdue` | Lending policy and its table, Assess overdue loans |
| T1 due-date limit per class | `lending/borrow` with `lending/loan-policy` (M3) | Borrow, Lending policy |
| T2 renewal admitted once, window from borrowing date to current due date, extension from the current due date by the copy's period, count, availability and borrowing date unchanged, closed and unknown loans, the due-date limit, invalid inputs | `lending/renew`, `maxRenewals`, `lending/loan-policy`, `lending/loan` (M4) | Renew, Lending policy and its table, Loan |
| T2 suspended member's renewal refused until reactivation (M4 only) | `members/eligibility`, `lending/renew` (M4) | Eligibility, Renew |
| T2 overdue assessment after renewal (composition) | `lending/loan-policy` ("which is then the loan's current due date"), `lending/reminders/assess-overdue` unchanged (M4) | Lending policy, Assess overdue loans |
| T2 `renewals` in every loan record; `renew` arguments and unknown fields | `library/callable-interface` (M4) | Callable interface (records, operations, wire format) |
| T3 `setMemberStatus` is an unknown operation; member records without status; borrowing and renewal not blocked by membership | `library/callable-interface`, `members/member`, `members/eligibility`, `lending/borrow`, `lending/renew` (M5) | Callable interface (operations table, unknown-operation rule, records), Member, Eligibility, Borrow, Renew |
| Carried E01–E12, witness, X01–X06 at M5 without suspension | as in section 4, with the M5 edits | as in section 4 |

- **Independent review (condition 2), 24 October 2026.** A fresh Opus reviewer checked the materials at `999a6242` read-only against B1–B6. Its verdict: the overlays contain every T1–T3 edit and no other semantic change; the snapshots are coherent with no stale statement; the renderer reproduces M1 and M2 byte-identically and proves M3–M5; every new case is an unambiguous consequence of sentences in both representations, with correct date arithmetic. It built an independent reference library from the documents, which passes all 21, 27 and 26 cases and the discrimination at each profile, also through the oracle's real transport, and nine deliberately wrong variants were each caught. Its one must-fix, the obligation table above, and its should-fix items (the outer time limit, hash-binding the expectation files, the two wording additions, the program hash in the assessment record, follow-on edits and the fixture note) are applied in this commit; anchors are now checked for every profile. Left as known limits: no offline test exercises the assessor's m3–m5 container path, which the reviewer exercised end to end, and `loanClass: null` or non-string classes stay untested because the interface leaves them ambiguous.

### B8. Pre-data amendment: arm M runs Phase B on F′ (25 October 2026)

Recorded on the coordinator's decision of 25 October 2026, before any Phase B run of either arm. No Phase B run existed, so this is a change approved with a reason before any run it affects, as the protocol's header requires. It moves arm M to a new source, as section 12 asks of a product fix, but inside Phase B instead of a new study.

- **Finding of F.** Step 4's offline rehearsal showed that on F the Apply preflight refuses M's T3 input. The M5 snapshot deletes `members/set-member-status`, and the registry links to that Definition are then invalid, although [concept §6.3](../concepts/v1-concept.md#63-registry-maintenance-is-realization-project-intent-is-not) makes their cleanup part of the Managers' realization work. This is a product gap of F and is recorded as a finding of F: the rehearsal's T3 input was refused before any Manager started. Every Phase B unit needs T3, so on F every M unit would have been non-conforming.
- **F′.** Arm M runs every Phase B run on F′, the Main commit that merges the fix for this gap (being made by the A4 session). Its SHA is recorded here before the first M run of Phase B. F′ also contains every change merged to Main after F. Besides the coming fix, the only product-code change among them is #696 (the Run's no-progress stop counts every action, so read-only review is never a standstill); the others are documentation, a test-only fix and the Phase B materials. **M's Phase A ran on F; M's Phase B runs on F′.** Phase A is not re-run, and Phase B measures F′, not F. M uses the `markitect` binary built from F′. Its chains 1–3 continue from S3 results and state directories produced on F; chains 4 and 5 run every step on F′. The runtime image Id M uses is recorded beside the SHA; if it differs from B's `sha256:1271a363…0ba9`, the shared-image condition of section 2 no longer holds for Phase B and is a stated limit.
- **Unchanged.** Arm B (runner, prompt, image and the documents m3–m5), the T fixtures and model snapshots, the hidden cases and the assessor at `43d0365e` stay exactly as frozen. The owner change at each step stays model-only.
- **F′ recorded.** F′ = `528ef346d3149002f53bf8eca1c54ea5613453ed`, the Main merge of #701 ("a deleted Definition's registry link is stale realization, not invalid input"). Between F and F′ the only product-code changes are #696 and #701; #698 changes one test only. M's runtime image on F′ is `sha256:dd0ddd7089b52024e96cc35655c4d8590caabaa6b6327923717584408403736f`, which differs from B's `sha256:1271a363…0ba9`, so the shared-image condition of section 2 does not hold for Phase B and is a stated limit. Step 4 started M-c1-T1 on F′ at 09:08 UTC on 25 October, before this record was written, so the record misses the "before the first M run" timing stated above; F′ was the source the coordinator dispatched to step 4 once #701 merged.
- **Order.** Arm B starts before F′ exists, so Phase B gives up the counterbalanced pairs; at least B's first runs precede every M run. This replaces B3's order principle and B5's sentence that the extra chains run in counterbalanced pairs under Phase A's parity rule. Each arm keeps the chain order of B5 (chains 1–3 T1–T4, then chains 4 and 5 with S1–S3 first). Time effects between the arms, such as provider load or model behavior changing over the hours between them, are a stated limit of Phase B.

## Addendum: Phase C (Makerspace)

Status: **approved by the coordinator on 26 October 2026 (N = 3), before any counted Phase C run.** Uncounted smokes have run (C7); no counted run existed at approval. Phase A's and Phase B's text, tools and readings stay unchanged; Phase C is read by its own rule (C6) and never pooled with them.

### C1. Why another phase, and which project

Phases A and B tied on the small Library: behavior hit the ceiling in both arms. The thesis says Markitect's advantage should appear when one agent can no longer keep every obligation in view across areas and changes. Phase C therefore uses **Makerspace**, a community-workshop program written for this purpose from a one-page [brief](../../tools/comparison/phase-c/brief.md). Neutral requirements came first ([requirements](../../tools/comparison/phase-c/requirements/README.md), #702); the model, its renderings, the hidden oracle and the reference implementations were derived from them by separate sessions.

| | Library (Phases A, B) | Makerspace (W0) |
| --- | --- | --- |
| Definitions | 22 | 75 |
| Operations | 10 | 37 |
| Packages / Managers | 5 / 6 | 20 / 21 |
| B's document (words) | 2,669 | 8,222 |

Cross-cutting policies (authorization, audit, money, time) apply to almost every operation, and several eligibility checks span three or four areas. The whole document still fits in a modern context window, so the claim under test is about attention and change fanout, not memory.

### C2. Materials and sources

- The first Phase C freeze was **F_C = `6595ebf672b9a5e08d7309665615834982a8e8b2`**. Its product code is identical to `3347cf0a`: F′ plus #706 (build outputs out of candidates, checks run once) and #705 (Package removal or merge, needed at C7). The first W0 smoke ran on F_C and its runtime image, `sha256:9593bdd0cb63b4d7a5fc403f5303cff17d3ccb591dd605ba08f40e4c3df9c8fc`, which was built by arm M's controller. In every Phase C run both arms use the same image, so the shared-image condition of section 2 holds again.
- Materials, all reviewed and merged before this addendum:
  - Requirements: #702 (`795b8a2a`).
  - Model snapshots, renderings, mapping and execution configuration: #707 (`fed1fe22`).
  - Reference implementations and mutants: #703 (`a559283b`).
  - Hidden oracle: #708 (`6595ebf6`).
  - Equivalence, traceability and calibration review: #709 (`9cee5f5c`).
- V found the model equivalent to the requirements at all ten steps. Its oracle calibration, pass 2, matched expectations in 47 of 47 checks.
- **Fix-and-resmoke.** The W0 smoke found boundary gaps in the requirements text (C7 below). R closed them and swept the whole text for the same class of gap (#713, #717). M, O, I and V adapted, and V certified the new input version, materials freeze v2 = **`4840fb20f52c1e0e0367e909d833919ca8a215c6`** (tree `b53319d1`; M #719, O #716, I #714; V's calibration pass 3: 48 of 48). **Both arms run every counted step on the runtime image built from F_final**: the final product state after P1, P3 and the A4 timeout fix, **F_final = `4d23f7aea0576ff5c477fd301b456a24bbb6dcb3`** (#725), image **`sha256:3d75d81f438e71254d01f013cd35e0c61f9eecfb8abba723b34e12158c33e943`** built from it by step 4 (base `c2920c61…`, codex-cli 0.162.0). Between materials v2 and F_final only V's review record changed under `tools/comparison/`. The study tooling, oracle, reference, model snapshots and B's documents are byte-identical, so R_C stays `4840fb20`. M's last smoke runs on F_final too. The study is rebound to that image once (`prepare-pc.sh`, which refuses any rebind after a counted run), and the ten positive controls run at that binding before the first counted run. Earlier smokes ran on the F_C image `9593bdd0`, recorded per run.
- Assessment revision **R_C = `4840fb20`**, the materials freeze itself: `assess_candidate.py` profiles `w0`, `c1`–`c9` (#711), the adapted hidden oracle, `analyze.py --phase C` and `cost_breakdown.py` (#718). Its end-to-end positive controls on the reference passed before the fix (W0 49 of 49; C3 53 of 53 with an empty change census). On R_C and the F_final image they ran for all ten steps (`e2e-assessor-v2.sh`, each step from the previous step's reference), and **all ten pass**. Hidden cases grow from 54 at W0 to 63 at C9 (54, 55, 57, 58, 59, 61, 62, 62, 63, 63), families from 48 to 56, probe families 30 at W0 and C1 and 31 from C2. The change census is empty at C3, C7 and C9, and own checks pass at every step. Their family lists are the ledger's catalog per step. A first run failed at C7–C9 in the script, not the assessor: those references keep the previous tree, so an empty commit is needed. It is kept on record.
- **One binding.** `phase-c/study.env` in the evidence root names R_C, the image and arm B's document packet. Every Phase C script reads it: run, assess, record and ledger. `prepare-pc.sh` writes it once and checks out the assessor clone at R_C. The packet holds the ten `spec/requirements-<step>.md` of the materials freeze, with their hashes, and is checked before every chain.

### C3. Arms, conditions and treatment differences

Phase A's arms and shared conditions apply, with these Phase C values:

| | M | B |
| --- | --- | --- |
| Source and image | F_final, `3d75d81f…` | the same image; the Phase A runner and prompt (`8a6c99a7…`) |
| Input per step (C4) | its last deliverable candidate plus one neutral commit that replaces `.markitect/model/` with the step's snapshot (`tools/comparison/phase-c/model/<step>` at the materials freeze) | its last deliverable capture plus one commit that replaces `requirements.md` with the step's rendering (`spec/requirements-<step>.md` at the materials freeze, from the packet) |
| Continuation in a chain | the chain's state directory | the chain's Codex thread |
| Deadline per run | 5,400 s (Apply) | 5,400 s (session) |
| Messages | maxMessages **256** per Apply, maxManagers 32 | one input message |
| Resources | Manager session limits as in Phase A | the six-Manager aggregate ceiling of Phase A (6 CPUs, 12 GiB, 1,152 processes) |

**Intended treatment differences**, beyond Phase A's:

- M's execution configuration (`tools/comparison/phase-c/execution/w0.yaml`) adds a `project-build` check and a 5-minute `project-tests` timeout compared with the Library fixture. The assessment's own checks run under the same 5-minute limit for both arms.
- **Owner attention on arm M's side.** At C7 the owner replaces `execution.yaml` together with the model (`execution/c7.yaml`, which folds `billing/payments` into `billing/invoices`). B receives only the C7 document.

### C4. Steps, order and attempts

The ten steps are W0 (fresh build) and C1–C9:

1. C1: audit of refusals.
2. C2: safety lockouts.
3. C3: bulk material prices.
4. C4: the steward role.
5. C5: overdue invoices block bookings and purchases.
6. C6: certifications no longer expire.
7. C7: payments fold into invoices, without a behavior change.
8. C8: 15-minute booking granularity and new opening hours.
9. C9: unchanged re-application.

C8 and C9 render identical documents.

Each step continues the arm's own actual result, defects included. A chain runs its ten steps in pairs, step by step, under Phase A's parity rule: M first in odd chains, B first in even chains. The serial engine lock applies. A step has one attempt unless the infrastructure rule below allows a repeat.

**Deliverable candidate.**
- For M, a deliverable candidate is a published one: Apply status `passed`.
- For B, it is the capture of a session that completed. A session that hits its deadline or fails leaves none, just as an Apply that does not publish leaves none.

**Input of a step.** Each step starts from the arm's last deliverable candidate in the chain, plus the step's model snapshot (M) or document (B). B also resumes that candidate's Codex thread. Until a chain has any deliverable candidate, a step starts as W0 does: M from the neutral input, B from an empty repository and a new thread. C9's document equals C8's, so after a deliverable C8 the C9 input is that candidate, unchanged.

**A step without a candidate** stops the series until the operator records a decision in the step's run directory. There are two:
- **`repeat`:** only when the attempt is shown to be an infrastructure failure, that is, the harness ended before any case ran or a technical fault provably independent of the candidate. The next attempt then runs, both stay recorded, and the last one counts.
- **`no-candidate`:** every other case. The step counts **every probe family of the step as missed** (30 at W0 and C1, 31 from C2 on) and is not conforming, and the chain continues as above.

The same rule applies to both arms. A failure that cannot be clearly separated from the candidate counts as the first result.
- An attempt that left a candidate, or for M raised a model blocker, is never repeated.
- The study numbers each step's attempts from 01, because step 4 makes M's replacement attempt as a new packet.
- Every attempt's time and tokens count toward the step's cost.

**Machine quietness** (coordinator, after the re-smoke).
- No heavy parallel work may run on the machine during any counted run of either arm: no host test suite, no other Apply.
- `quiet-monitor.sh` samples the host every minute while the engine lock is held. It records:
  - test suites on the host (`go.exe`, `*.test.exe`);
  - test processes in a running WSL Ubuntu;
  - every Markitect process with its executable;
  - running containers;
  - CPU load.
- A run is quiet when no sample shows a test suite or a Markitect process other than M's own Apply, which is the binary in its packet. A probe that fails is recorded as unknown, so a run with an unknown sample is reported as "unknown", never as quiet. The record is reported per run.
- A counted run is started only on a quiet machine.
- A run that turns out not to be quiet keeps its quality result, because repeating on that ground would let the operator choose results. Its time and tokens are reported but left out of the cost comparison. A technical failure in such a run may be repeated only under the infrastructure rule above.

**P2 observation.** Step 4 also records a P2 observation (the Verify-reject path) on each M Apply. It is a technical observation for the acceptance report and never changes a study outcome.

**Model blocker in a counted run (M).**
- The Apply publishes nothing, so the step has no candidate and is scored as above.
- The blocker is classified separately: **justified** when V confirms a genuine gap or contradiction in the step's text, otherwise **false**. The class is recorded before the ledger is built, and a false blocker also counts under section 8.
- The results state the trade-off plainly: M stops on gaps, B guesses.

### C5. Hidden assessment

Every candidate of both arms is assessed with `assess_candidate.py` at R_C under the step's profile. The steps are:

- **Hidden cases:** the Makerspace oracle's cumulative case set of the step, through the Library oracle's transport, with the oracle's own limits:
  - behavior cases (B-* per area, D-C1–D-C8 per change);
  - **30 probe families at W0 and C1, 31 from C2 on** (materials v2 add P-LIMITS);
  - ten invariant checks over one mixed scenario, whose statuses are judged once as INV-SCENARIO.
- **Change census:** at C3, C7 and C9, the step's cases are also run against the arm's own previous program.
- **Own checks:** `go build ./cmd/makerspace` and `go test ./...` from `realization/`, under 300 s each in both limit profiles.
- **Path census, scope greps, a blinded per-candidate review** with citation check and confirmation, against the step's rendered document, and the **contamination audit**, as in Phase A.

**How the census is used:**

- **C9** is the strict no-change gate. Any census difference to the arm's own C8 program, `reason` included, makes C9 non-conforming. So does any changed comparable path, and for M any changed registry path.
- **C3 scope rule.** A C3 census difference, with `reason` ignored, is a change-discipline failure: any case without an accepted purchase of ten or more units must answer as the arm's C2 program does.
- **C7 churn rule.** The same rule applies to both arms. A census difference, with `reason` ignored, is a change-discipline failure. Changes needed to keep the project valid under the documented fold are justified: for M, registry relinks and ownership-forced moves out of `realization/billing/payments/`. Behavior-neutral moves or renames that follow the fold are justified too. Any behavior change, and any edit outside the billing area, counts as unjustified churn. Moved or relinked lines per arm are reported as a separate secondary measure, so their cost stays visible.
- **The billing area** is decided mechanically and identically for both arms, because B's layout is its own. A changed file in the blinded diff is outside the area when neither its path nor any of its changed lines contains `payment`, `invoice`, `billing`, `paid` or `pay` (case-insensitive, whole words for the last two). Registry shards are exempt. The rule is lenient by design. It catches edits with no billing connection; a behavior change anywhere is caught by the census, and removed behavior by R5's *unjustified* class.
- A census that cannot run, because the arm's own previous program does not build, fails change discipline at that step.

### C6. Outcomes, N and decision rule

- **Conforming step:** all five criteria hold. Their names avoid the step names; the ledger keeps Phase A's keys, shown in brackets.
  - **Technical** [C1]: as in Phase A.
    - M: outcome `technical_pass` from step 4's controller, no residue, no failure. The controller's Makerspace profile must not apply the Library's "exactly one project-tests check" publication rule, which wrongly failed the F_final smoke. That fix lands before the first counted run, and the controller version is frozen with this addendum (C8).
    - B: the session completed, no residue, login unchanged.
  - **Hidden cases** [C2, C3]: the program built within the oracle's limits, and every hidden case of the step passes.
  - **Own checks** [C4]: build and tests pass in both limit profiles.
  - **Review** [C5]: no confirmed critical rubric finding, and no protected input changed.
  - **Change discipline** [C6]: the C3 rule, the C7 rule and the C9 gate.
- **Primary metric:** missed obligations per chain, the sum over the ten steps of **failed probe families**. Behavior-case families and invariant families are not added: in V's calibration, 17 of 30 single faults also failed a strict behavior or invariant family, so a combined count would weight one fault two to ten times.
- **Declared dependences:** six probe families have declared inherent dependences, listed in V's review record. They count as they fall, the same for both arms, and are named as a limit.
- **Secondary, reported per arm and step:**
  - failed behavior families;
  - invariant failures together with INV-SCENARIO, read as one signal, because a refused record-creating step stops the scenario and fails all ten invariant families with it;
  - conforming steps;
  - stale obligations (above all at C6);
  - regressions of earlier families;
  - R5 classes and changed lines;
  - moved and relinked lines at C7;
  - false blockers and unsupported success;
  - M's Apply completion rate;
  - time, tokens, messages and quota.
- **Unit and test:** the unit is the chain. The primary comparison is a one-sided exact permutation (Mann–Whitney) test that M's per-chain totals are lower.
- **Reading:**
  - **"Supports"** needs all three: p ≤ 0.05; M at no step missing more probe families than B, summed over chains; and no critical error type that only M shows.
  - M lower without that: a **tendency**, not established.
  - Equal totals: **no quality advantage shown**. With M's higher cost, this counts against the claim.
  - B lower: **counts against Markitect**.
  - Section 8's list applies: unsupported success, false blockers, unjustified churn at C7, a failed C9 gate, and technical failures, each where B has none.
- **Error types** (section 6) are, per step:
  - every failed criterion;
  - every failed family;
  - `no-candidate` or `not-built`;
  - `false-blocker`;
  - `unsupported-success` (M published a candidate that fails any of C2–C5).
  A step without a candidate, or whose candidate did not build, fails every family of the step. Each step's family list is the catalog of its positive control on the reference. This keeps the "only M" check fair when one arm has no program at a step.
- **Regressions** are counted for candidates only. Steps without a candidate are reported as their own measure, so a blocked step is not also counted as dozens of regressions.
- **N = 3 chains per arm** (C7). With N = 3, p ≤ 0.05 requires complete separation (minimum p = 0.05). With N = 5 the minimum p would be 0.004.

**The owner's measures** and where each one is read:

| Measure | Read from |
| --- | --- |
| Complete domain correctness | Hidden cases per step: probe families (primary), behavior families |
| Cross-cutting invariants | Invariant families with INV-SCENARIO, as one signal |
| Regressions after changes | A family that failed at a step but passed at the arm's previous deliverable step in the chain |
| Unintended changes | Census differences at C3 and C7, files outside the billing area at C7, R5 *unjustified* |
| No change and preservation | The C9 gate; the C3 scope rule |
| Runtime and tokens | Per run and chain, below |
| Human interventions | M's recorded interventions; the owner's C7 `execution.yaml` replacement (owner attention, M only); every operator decision, manual fix or restart, per arm |
| Technical failure rates, incomplete runs | Per arm: repeats for infrastructure, steps without a candidate, model blockers by class, deadline hits, M's Apply completion rate |

**Cost analysis**, predeclared under the owner's principle: if B handles Phase C equally well, the study explains why M costs more instead of raising the difficulty.
- Every M Apply is broken down by `cost_breakdown.py` from its own session transcripts. It gives tokens (input, cached, output, reasoning), turns, tool calls and session time:
  - by Run type (assessment, implement, verify, with correction pairs);
  - by Manager level (Root = 0);
  - by Package;
  - by tool.
- Messages come per Run type from the run summary.
- B's cost per step is its session time and its own tokens: the thread's cumulative usage minus that of the result it resumed.
- Reported per step and per chain:
  - M/B ratios of time, tokens and messages;
  - the shares of M's tokens by Run type and Manager level;
  - the tool calls that dominate;
  - whether the cost scales with the step's change or stays fixed per Apply.
- These are descriptive measures; they do not enter the decision rule.

### C7. Measured smoke (uncounted) and budget

All smokes are uncounted. Every attempt is listed.

| Smoke | Arm, materials, image | Result | Time | Tokens (in / cached / out) | Messages | Weekly quota |
| --- | --- | --- | --- | --- | --- | --- |
| F_C extra | B, F_C, 1271a363 | completed; 49 of 49 W0 cases | 386 s | 1,288,128 / 1,215,360 / 28,376 | 1 | – |
| F_C | M, F_C, 9593bdd0 | **justified model blocker** at Root after Assessment: the 2099 range gap | 283 s | 7,504,914 / 6,568,704 / 22,092 | 22 | 14 → 15 % |
| v2 attempt 01 | M, v2, 9593bdd0 | preflight infrastructure failure: the 60 s `docker cp` image observation timed out before any Run | 84 s | none | 0 | – |
| v2 attempt 02 | M, v2, 9593bdd0 | **no candidate**: Assessment approved; Implement ended technically incomplete (16 of 21 handoffs) | 1,023 s | 30,907,760 / 28,617,600 / 129,685 | 61 (21 + 40) | 15 → 16 % |
| v2 s2 | B, v2, 9593bdd0 | completed; 54 of 54 W0 cases, no failed family, own checks pass | 284 s | 843,829 / 782,208 / 25,090 | 1 | 15 → 15 % |
| v2 official (s3) | B, v2, F_C image 9593bdd0, quiet (4 of 4 samples) | completed; 54 of 54 W0 cases, no failed family, own checks pass | 281 s | 1,223,533 / 1,124,608 / 26,112 | 1 | 16 → 16 % |
| F_final attempt 01 | M, v2, F_final image | no candidate: Assessment approved; Implement stopped at 161 s on a controller defect (step 4 renamed the `go` check profile) | 412 s | 14,575,110 total | 63 (21 + 42) | – |
| **F_final attempt 02** | M, v2, F_final image, quiet (19 of 19 samples) | **published**; Assessment, Implement and Verify approved, no correction; **54 of 54 W0 cases**, no failed family, own checks pass | **1,265 s** | **50,566,157 / 47,108,736 / 156,840** | **102 (21 + 60 + 21)** | 18 → 20 % |

Notes:
- **The 2099 gap.** Root found the boundary gap that V's equivalence review had missed, and stopped as DEC-063 requires. B passed all W0 cases with a silent choice: it formats dates past 2099 with no rule. R closed the gap and its class (#713, #717). On v2 M's Assessment approved the model.
- **s2 is an extra measurement.** It ran on v2 before the coordinator's quietness rule. It overlapped no Apply, but quietness was not recorded. The official B smoke ran on a quiet machine afterwards.
- **W0 on F_final: equal quality, very different cost.** Both arms pass all 54 W0 cases, with no failed family and their own checks passing.

  | W0 | M (attempt 02) | B (official s3) | M / B |
  | --- | --- | --- | --- |
  | Time | 1,265 s | 281 s | 4.5× |
  | Tokens (input + output) | 50.72 M | 1.25 M | 41× |
  | Messages | 102 | 1 | |
  | Changed files / lines | 44 + 20 registry / 3,396 | 6 / 1,363 | 2.5× lines |
  | Test files / test lines | 21 / 1,722 | 1 / 410 | |

- **Where M's cost goes** (the predeclared breakdown of the complete Apply, 63 sessions):
  - By Run type: Assessment 7.25 M tokens, Implement 28.51 M, Verify 14.96 M.
  - By Manager level: Root 8.67 M over 3 sessions; level 1 23.55 M over 30; level 2 18.50 M over 30.
  - By tool, the heaviest calls: 328 shell commands, 107 awaits, 67 judgments, 65 checks, 64 delegations and 64 reviews.
  - 93 % of M's input tokens are cached.
  - Every Assessment Run measured so far costs 7.1–7.8 M tokens over 21 sessions, about six times B's whole W0 build, before any code is written. Every mandate runs the complete hierarchy, so this cost recurs at each step.
- **M's complete W0 Apply** (`ee91c7e6`, packet `m-w0-02`), in step 4's record:
  - Input: the revision-2 W0 (manifest `1f652258`, `w0.yaml` `2741e461`, model digest `cfb7a9fe`, 21 Managers). Limits 5,400 s, 256 messages, 32 Managers, messageTimeoutSeconds 600.
  - Assessment approved, Implement submitted, Verify approved, with no correction.
  - It published 64 paths (final `14d0c992`, tree `e0e47844`). project-build and project-tests passed on the final candidate, and the readback is true.
  - By Run: Assessment 21 messages, 7.25 M tokens, 2.1 min; Implement 60 messages, 28.51 M tokens, 624 s; Verify 21 messages, 14.96 M tokens, 3.7 min.
  - #725 (the per-message clock) is consistent with this run but not proven by it. Root's longest Implement turn took 573.6 s (338.5 s in host tools, 235.1 s with the provider), and no turn in any Run exceeded the 600 s limit.
  - The P2 path (Verify reject, fresh Implement, fresh Verify) was not exercised, because the first Verify approved.
- **M's earlier W0 attempts.**
  - The v2 attempt 01 timed out on the engine in preflight while WSL was loaded.
  - The v2 attempt 02 hit the old per-message timeout during Root's turn, before #725.
  - The F_final attempt 01 failed on step 4's controller defect and was repeated under the infrastructure rule.
  - Step 4's controller still marked the published attempt 02 `technical_fail`, through a Library assumption (`requalify.py:470`, exactly one project-tests check). That is a harness false negative; the verdict is kept as recorded, and the rule is fixed for the counted runs (C6, C8).
- **The F_final attempt 01 was not quiet.** Two of its seven samples show another session's `go run ./cmd/markitect`, so its time is not a clean cost measurement. This is why the monitor now records executables.
- **Quota.** The account reports integer percents of a weekly window, which next resets 2026-11-01 06:10Z.
  - Usage rose from 14 % to 20 % over the smokes. M's complete W0 moved it from 18 to 20 % with 50.7 M tokens.
  - That is roughly 20–25 M tokens per percent: a coarse estimate, and other use of the account cannot be excluded.

**Budget for counted runs** (proposal):
- **M:** maxMessages 256 per Apply, maxManagers 32, a 5,400 s deadline.
  - That is 2.5× the 102 messages and 4.3× the 1,265 s of the complete W0, with 21 of 32 Managers used.
- **B:** one session, a 5,400 s deadline, 19× its W0 time.
- **Estimate per chain:**
  - M: W0 about 51 M tokens. Each change step costs about 22 M for Assessment and Verify over the complete hierarchy, plus Implement, estimated at 10–25 M against 28.5 M for the full W0 build. That gives about 32–47 M per change step, so **about 340–480 M tokens per chain, about 14–19 % of the weekly quota**, and about 3–4.5 h.
  - B: about 12.5 M tokens (under 1 %) and about 50 minutes per chain.
- **Quota guard**, an operational rule that does not affect any reading: no counted run starts while the weekly quota is at 90 % or more. The series then waits for the reset and continues in order.

**N = 3 chains per arm** (proposal).
- **Quota:** three chains need about 45–60 % of the weekly quota, with margin for repeats, on top of the 20 % already used. That fits the window that resets 2026-11-01. N = 5 would need about 75–100 % on top and does not fit.
- **Power:** with N = 3, only complete separation of the arms' per-chain totals reaches p = 0.05. A smaller advantage reads at most as a tendency. This limit is stated in advance.
- **Expectation, stated before the data:** W0 already shows a ceiling. Both arms pass every W0 case, so any difference must come from the changes C1–C9: missed obligations, regressions, change discipline and the C9 gate.

### C8. Frozen study infrastructure

The study infrastructure is frozen with this addendum and does not change in content during counted runs. A defect found during the series is recorded as a finding. Any repair is a new, separately reported amendment; it is never presented as the original rule, and runs already made keep the version they ran with.

| Part | Identity |
| --- | --- |
| Assessor, hidden oracle, reading and cost tools | R_C `4840fb20`: `assess_candidate.py`, `tools/makerspace-oracle/`, `analyze.py`, `cost_breakdown.py` |
| Arm B's runner and prompt | R `74b7d7df`: `runner.py` `249f8ab7…`, prompt `8a6c99a7…`, unchanged through R_C |
| Arm B's documents | `phase-c/packet-4840fb20f52c/SHA256SUMS` |
| Study binding | `phase-c/study.env`: RC `4840fb20`, image `3d75d81f…`, the packet |
| Family catalog | `phase-c/e2e-v2/assess-<step>/assessment.json`, the ten positive controls |
| Evidence-root scripts and records | `phase-c/FREEZE.sha256` (sha256 `848da6f75500800c8aa8c788d0ada715d4f528f63c880f75293d171b5461b62e`): prepare-pc `49aecfba…`, run-b-pc `b7edb6ee…`, drive-b-pc-chain `c961a3b0…`, assess-pc `9aae63b3…`, assess-m-pc `3d97708b…`, build-ledger-pc `7795b9ae…`, quiet-monitor `20e1647d…`, e2e-assessor-v2 `c4efe155…`, study.env `44a908ab…`, the packet and the ten catalog records. Reviewed by an independent Opus review over four rounds; the last round accepted after a one-line fix, which is applied. |
| Arm M's runner | step 4's counted-run controller `controller-pc-review-02` (Markitect-V1-Requalification*8ef346), reviewed READY: `requalify.py` `cafb9fcd31400c8fdd94e1ff6f356b083bba48a8c60e3a8369c28be881ecef43`, 51 self-tests, `launch.py` `492d4874c60b53ec…`, its `SHA256SUMS` `535b0e31a4413e78c322fb3177fccbbc5fe01da628d2b7eb7688004f9a3cde56`. Its published checks are the fixture's declared IDs, each exactly once, and cleanup recognizes check containers by those IDs, so the retained `ee91c7e6` report now passes. |

### C9. Limits

- One model and effort, one provider, one machine, and a small N.
- The project is larger but still fits in one context.
- The six declared probe-family dependences.
- INV-SCENARIO folds the invariant families.
- The mapping from neutral text to Definitions was not reviewed, because it is not an arm input.
- The snapshot checks rest on M's record.
- The blind review is judgment.
- B's resource ceiling is the Phase A aggregate, not 21 Manager sessions.
- The owner's C7 execution change is attention on M's side only.
- The analyst is not blinded.
- The billing-area rule at C7 is a word match: it can miss churn in a file that merely mentions billing.
- The `repeat` and `no-candidate` decisions are the operator's, under C4's rule. Every decision is recorded with its evidence.
- W0 already reaches the ceiling in both arms, so the comparison rests on nine change steps per chain.
- Quota percents are integers and the account may carry other use, so cost-per-percent is approximate. Tokens are the cost measure; quota only gates the series.
- Quietness is sampled once a minute, so short bursts of parallel work between samples can be missed.
