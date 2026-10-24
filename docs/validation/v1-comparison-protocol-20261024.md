# V1 comparison protocol: Markitect against one agent with ordinary documentation

Date: 24 October 2026. Status: **predeclared, awaiting coordinator approval. No counting run has started.** This protocol is frozen at the commit that the coordinator approves; that commit is the assessment revision R. A change after approval needs the coordinator's approval and a recorded reason before any run it affects, and no change follows the assessment of a counting run except the repair of an assessor infrastructure defect, which is recorded.

The study answers the comparison half of [DEC-072](../concepts/register.md#dec-072-stable-v1-justified-by-data) ([owner source](../concepts/sources/v1-evidence-goal-20261024.md)): is Markitect V1 better than one agent doing everything? It follows the [measurement guide](../guides/measurement.md#quality-first-comparative-evaluation), which is binding, and tests the thesis of [concept §1.4](../concepts/v1-concept.md#14-product-thesis-and-limits-of-evidence). The tools live in [`tools/comparison`](../../tools/comparison/README.md). The stability half of DEC-072 is the step-4 requalification of the [delivery plan](../implementation-plan.md); its Library Applies are this study's arm M.

## 1. Question and expected outcome

**Question.** On the reference scenarios of [concept §26](../concepts/v1-concept.md#26-reference-scenarios), with the same model, effort, resources and an independent hidden assessment, does Markitect deliver more fully conforming first results than one coding agent that receives the same intended meaning as ordinary documentation?

**Honest prior.** Earlier single agents realized Library M1 with 12 of 12 oracle cases in two to five minutes ([archived capability record](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/runtime-capabilities-20261017.md#single-agent-baselines)). Plain behavior on a fresh M1 is likely a ceiling for both arms. The study therefore adds easy-to-miss obligations, a change, a no-change and a subtle seeded defect, and it treats a tie as a likely and legitimate result. Markitect is expected to be slower and to use more tokens.

## 2. Arms

| Arm | What it is |
| --- | --- |
| **M** | The public `markitect apply` command on the frozen V1 source F, run by the step-4 session exactly as its dispatch packets define (`examples/` runtime profile and run configuration, Apply deadline 2,700 s, 128 messages, six Managers, fresh Runs). |
| **B** | One fresh `codex exec` session in the same Go runtime image, started by [`runner.py`](../../tools/comparison/runner.py). Its repository holds `requirements.md` and, from S2 on, its own prior work. Its only instruction is [`prompt.txt`](../../tools/comparison/packet/prompt.txt). No second agent, reviewer, Manager or prescribed process. |
| B+ (optional) | A stronger conventional arm, for example B followed by one fresh self-review session. **Not run in this study**; if added later, it is labeled separately and never replaces B. |

The owner's instruction for B is quoted from the [owner source](../concepts/sources/human-attention-and-baseline-20261017.md#comparison-baseline): „Lies dir die Doku durch und implementiere das so“. The prompt uses its English translation, "Read the documentation in requirements.md and implement it accordingly", so that both arms work in one language.

### Shared conditions

| Condition | Both arms |
| --- | --- |
| Model and effort | `gpt-6-sol`, `high`, in every Markitect role and in B |
| Codex | 0.162.0 from the image's `runtime.env`; the same feature switches (`apps`, `memories`, `multi_agent`, `multi_agent_v2`, `image_generation`, `browser_use`, `computer_use` disabled; `agents.enabled=false`; project documents off; `/work` untrusted); approval `never`, sandbox `danger-full-access` inside the container; working directory `/work`. A self-test reads these from `internal/adapters/codexappserver/profile.go`. |
| Login | The step-4 dedicated normal account login (`dedicated-runtime-login`); each session receives a copy without its refresh token; the file's SHA-256 is checked before and after |
| Image | The Go runtime image built from F in step-4 P0, bound by its engine image Id |
| Session container | Non-root `1000:1000`, read-only root, no capabilities, `no-new-privileges`, `--init`, `/tmp` 256 MiB, 1 CPU, 2 GiB, 192 processes, engine default network; the same values as `examples/runtime-profile.example.yaml`, checked by a self-test |
| Tools | Codex's own shell and file tools, Go 1.27.1, Git; B has no Markitect tools, which are the treatment |
| Wall clock | 2,700 s for the whole M Apply and for the whole B session |
| Freshness | A new session or Run for every scenario step; no Notes or state carried between steps except the repository itself |
| Human help | None after start. The first delivered result is frozen before any human look. |

### Intended differences (the treatment)

Markitect adds the typed canonical model, the model-derived hierarchy of six Managers with Assessment, Implement and fresh Verify, independent reviews, the deterministic `project-tests` check before publication, registry accounting and guarded publication. It also uses up to six concurrent sessions, each with the per-session ceiling above; aggregate compute is part of the treatment and actual use is reported. Its fixture carries Markitect control inputs (`execution.yaml` with ownership and the check, an empty registry) that B does not need; their preparation is setup attention for M.

## 3. Scenarios

All scenarios use the current Library model `testdata/library/m1` at F. There is no seed except in S4.

| ID | Scenario | Start of each run | Profile |
| --- | --- | --- | --- |
| S1 | Fresh M1 realization (§26.1) | M: the step-4 P1 fixture. B: a repository with only `requirements.md` (M1). No realization file in either. | m1 |
| S2 | M1 → M2, loan limit 5 → 3 (§26.2), longitudinal | Each arm's own S1 result, plus one commit: M changes only `maxOpenLoans` (step-4 P2); B replaces `requirements.md` with the M2 rendering, which differs in that one table cell | m2 |
| S3 | Unchanged re-application (§26.3), longitudinal | Each arm's own S2 result, unchanged | m2 |
| S4 | Seeded defect repair | One fixed seeded candidate, byte-identical in both arms outside `.markitect/` and `requirements.md`, as a single neutral root commit; M2 intent | m2 |

Longitudinal rules: S2 and S3 continue each arm's actual frozen prior result, including its defects, never a supplied correct project. If a prior step published nothing or left a broken tree, the next step starts from that actual state. A step is blocked only when no state can be captured; a blocked step counts as a non-conforming result and is reported as blocked.

**S4 base and seed.** The base is the first arm-M S2 candidate, in execution order, that passes C2–C4 below. If no M candidate qualifies, the first qualifying B S2 candidate is used, with an empty registry for M, and the extra registration work is reported. The seed is the step-4 P5 coherent seed class, with the rule the coordinator approves for step 4 (recommended: the overdue assessment includes loans returned after the assessment date, contradicting the last sentence of `lending/reminders/assess-overdue`). Code, its unit test and one usage-documentation sentence agree with the wrong behavior, so the candidate's own tests pass. Before any S4 run: the unseeded base passes C2–C4 (positive control); the seeded tree fails the oracle only in the E10 returned-loan assertions, passes the probes and its own tests (negative control); the coordinator approves the exact diff and the list of seeded lines. Seed descriptions, diffs and locations stay outside both writers' inputs; commit message, branch and identity are neutral.

S4 inputs follow the step-4 P5 rules in both arms: unchanged M2 intent, a single neutral root commit with neutral branch and identity, no seed wording in any file, message, ref or Note, and fresh state for every run.

**M-specific observation.** When an M run in S4 or elsewhere takes the public Verify Reject → fresh Implement → fresh Verify → publication branch, it is recorded as an M-specific observation; step 4 counts such an S4 run as its P5 proof and runs a separate P5 only if none of the three does, which the coordinator decides. B has no equivalent and nothing about the branch enters the comparison.

## 4. Equivalent requirements

B's documentation is rendered from the canonical model by [`requirements.py`](../../tools/comparison/requirements.py). Every one of the 22 local Definition bodies of M1 appears verbatim under a plain heading. Body links become Markdown links to the linked section with a readable link text. The two policy values come from the `loan-policy` spec; the two Property Definitions are rendered as the policy table with their bodies and "Exactly one integer value is required." for their `type: integer, minCount: 1, maxCount: 1` spec. Four phrases that name the model itself are replaced, and each must occur exactly as often as stated or rendering fails:

| Definition | Model wording | Rendered wording |
| --- | --- | --- |
| `lending/LendingPolicy` | "States the numeric" | "The lending policy states the numeric" |
| `lending/loan-policy` (twice) | "in this definition's spec" | "in the table above" |
| `library/callable-interface` | "offers every operation of the model" | "offers every operation in this document" |
| `library/realization` | "must agree with the model." | "must agree with this document." |

Nothing else is added except the title, one introductory sentence, the headings and the table frame. `requirements.py check` re-renders the model at F, compares the committed bytes and confirms that every rendered body occurs in the document; the self-test also confirms that M1 and M2 differ in the one table cell and that no `[[` link remains. At R the committed documents have SHA-256 `1b3dc865af6ed24be2e76b283089be6a8ad89e9257609d58cd74bb84d6179e1b` (M1) and `d0de9dbd9272764715110f8dc9ce09765d625d239a21e5c98f289dcc6b361d0a` (M2). Before the first counting run, `check` runs against `testdata/library/m1` at F; if C1 or anything else changed a Library body, the documents are re-rendered and this protocol is amended before any run.

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

Two assessed duties come from Markitect's Manager harness, not from the model. B receives the same duties in its prompt:

| Duty | Markitect Implement harness (`internal/harness`) | B prompt |
| --- | --- | --- |
| Preservation (S2–S4, R5) | "first reconcile the existing realization with the checked model's obligations … Change artifacts only for those obligations, checks or necessary consequences of that work. Preserve conforming artifacts … Do not introduce optional coverage, refactoring or cosmetic changes solely to produce a diff. Repair genuine defects …" | "If an implementation already exists, first reconcile it with the documentation. Change files only where the documentation, its required build and test commands, or necessary consequences of that work require it. Preserve conforming code, tests and documentation; do not add optional coverage, refactoring or cosmetic changes solely to produce a diff. Repair genuine defects you find against the documentation." |
| Protected requirement source | "Do not modify protected model/control inputs." | "Do not change requirements.md." |
| Unresolved intent | "Do not invent missing material model decisions. Report model_blocker with concrete Findings." | "report … any requirement you could not resolve." |

**B is not handicapped:** it receives every sentence of the model, the same delivery and preservation duties, the same tools, image, limits and wall clock, its own Git history and the same freedom to build and test. The rendered document is the best case for equivalence; a hand-written document would likely be less exact, which favors B.

## 5. Assessment

Every candidate of both arms is assessed by [`assess_candidate.py`](../../tools/comparison/assess_candidate.py) at revision R, in containers without network. For M the candidate is the final published candidate commit from its retained bundle; for B it is the runner's capture commit (all files except those the writer's own `.gitignore` excludes; the raw tree is retained too). The start commit is the run's fixture commit.

| Step | What runs | Limits |
| --- | --- | --- |
| Oracle | The unchanged [container harness](../../tools/library-oracle/README.md#container-harness) with its evaluator exported at F, for the scenario's profile: 12 cases E01–E12 and the paired witness, plus the opposite-profile discrimination. The assessor refuses to run if `tools/library-oracle` differs between F and R. Its record binds the candidate commit, tree and program SHA-256; it is the oracle of record for both arms, so step 4 runs no separate oracle on these candidates. | Oracle's own 2 s per request, 30 s per process; outer 1,200 s |
| Probes | [`probes.py`](../../tools/comparison/probes.py), X01–X06 above, built with the oracle's helpers and run through its transport and state snapshots; evaluator and probe bytes are hashed on the host and checked in the container | Same oracle limits; outer 300 s |
| Own checks | `go build ./cmd/library` and `go test ./...` from `realization/`, as the `project-tests` check profile: 1 CPU, 2 GiB, 64 processes, no network | 120 s each |
| Census | Every changed path against the start commit: comparable files, M registry files, protected inputs (M model and control files, B `requirements.md`), Go build outputs left in the tree | – |
| Scope greps | Non-test Go files using `time.Now(`, `net` or `net/http`, or `os.WriteFile/Create/OpenFile`; input for R4 | – |
| Seed check (S4) | Which approved seeded lines remain verbatim | – |
| Rubric | [`rubric.md`](../../tools/comparison/rubric.md) by a blinded reviewer | – |

**Blinding.** For the rubric, the assessor writes each candidate and its start tree without `.git`, `.markitect/` and `requirements.md`, plus their diff, under a random token. One fresh Claude subagent (Sonnet, high effort) per scenario reviews the six tokens of that scenario in random order with the current requirements text and the rubric. It is not told the arm, the scenario or any seed. The writers use a different model family. Each critical rubric finding must cite `path:line` with quoted text; the analyst checks only that the citation exists verbatim and does not re-judge. All rubric output is retained. Remaining arm hints, such as M's package layout, are a limit (section 9).

Writers never see the oracle, the probes, the rubric, this protocol, the seed or any assessment result. The assessment of a frozen candidate may be repeated after an assessor infrastructure failure; the writer run is never repeated for that reason.

## 6. Outcomes

### Primary outcome: conforming first result

A run is **conforming** when all of the following hold for its first delivered result:

| | Criterion |
| --- | --- |
| C1 | Technical completion. M: the step-4 common Apply pass except the oracle (Apply passed, published or no-change readback, all Runs closed, cleanup without residue, login unchanged). B: the runner's status `completed` (exit 0 within the deadline, one completed turn, no turn failure, capture, no residue, login unchanged, no secret in outputs). |
| C2 | The oracle passes all 12 cases and the opposite-profile discrimination for the scenario's profile. |
| C3 | All six probes pass. |
| C4 | Own `go build ./cmd/library` and `go test ./...` pass. |
| C5 | No critical rubric finding (R1, R2, R3a, R4, and in S2 and S4 an *unjustified* R5 change with a cited removed or unrelated change), and no protected input changed. |
| C6 | S3: if the S3 start was itself conforming by C2–C5, the comparable census is empty and, for M, so is the registry census; otherwise every changed file is classified *required* or *repair*. S4: no seeded line remains. S1, S2: nothing further. |

Build outputs left in B's tree are excluded from the census and reported as hygiene; M cannot leave them because of complete accounting.

### Secondary outcomes

Reported separately per arm and scenario, never folded into the primary outcome:

- each criterion C1–C6 on its own, and the failing oracle families and probes per run (missed obligations);
- **unsupported success**: M published (Verify approved) a candidate failing any of C2–C5; B's final message reports the work as done without naming the failing obligation as unresolved;
- **false blockers**: M reports a model blocker or rejects the valid model in Assessment; B names a requirement as unresolvable or ambiguous although the document states it;
- unnecessary changes in S2–S4: R5 *optional* and *unjustified* counts, comparable files and lines changed;
- R3b test topics covered (0–6);
- S4: E10 result and seeded lines remaining;
- human interventions after start (planned: none);
- elapsed time (M: Apply process; B: session), and for the whole run including setup and cleanup;
- tokens: B from the Codex `turn.completed` usage; M where the retained transcripts carry provider usage, otherwise unavailable, never estimated;
- messages and sessions: M host messages, Runs and Sessions; B one input message and its command and file-change counts;
- M-specific: Verify Reject branches, Assessment and Verify Findings, Notes written.

## 7. Repetitions, order and uncertainty

**N = 3 per arm and scenario**: three longitudinal chains S1 → S2 → S3 per arm and three S4 runs per arm, 24 counting runs. Reason: the measurement guide asks for at least three before describing a trend, and three already cost about seven hours of serial engine time. With a likely ceiling on behavior, only a large difference can show; N = 5 would roughly double cost without making a modest difference demonstrable at this scale. N is fixed: no extension, no early stop for success, no retry-until-green. A later study with a new protocol may follow.

**Order** (serial; one Docker engine and one account user at a time), counterbalanced scenario by scenario: in repetition r, each scenario runs as a pair, M first when r is odd and B first when r is even.

| Repetition | Runs in order |
| --- | --- |
| 1 | M-S1, B-S1, M-S2, B-S2, M-S3, B-S3; then fix the S4 base, author and approve the seed and run its controls; then M-S4, B-S4 |
| 2 | B-S1, M-S1, B-S2, M-S2, B-S3, M-S3, B-S4, M-S4 |
| 3 | M-S1, B-S1, M-S2, B-S2, M-S3, B-S3, M-S4, B-S4 |

If no S4 base exists after repetition 1, all S4 pairs move behind the first repetition that yields one, keeping their internal order. Every engine use, writer runs and assessments alike, holds the shared exclusive lock `NATIVE-RUN.lock` in the session notes folder: created exclusively before the first engine or provider call, holding holder, run and start time, and removed only by its holder after the final census. Assessments run between writer runs when the lock is free; an assessment never overlaps a writer run.

**One attempt per planned run.** A run may be replaced once only for a documented infrastructure failure before the writer produced any change: provider unreachable or erroring, account limit or expired access, engine or host failure, shown by the transcript or logs and classified before the candidate is assessed. The failed attempt stays recorded. A Markitect runtime failure, an incomplete or failed Apply, a no-progress stop, a model blocker, or a B session that ran and did not deliver is a result, not an infrastructure failure.

**Uncertainty.** Counts k of n with exact Clopper–Pearson 95 % intervals per arm and scenario and pooled over the 12 runs per arm. For orientation: 3 of 3 has the interval 29–100 %, 12 of 12 has 74–100 %. The pooled difference D = conforming M − conforming B is reported with a one-sided Fisher exact p; because S2 and S3 depend on S1 within a chain, p is descriptive. As a sensitivity analysis, the number of fully conforming chains (S1–S3) per arm out of 3 is reported. No other tests, no subgroups, no post-hoc metrics; every secondary outcome is reported in full.

## 8. Decision rule

Predeclared reading of the primary outcome, with secondary outcomes reported beside it:

| Result | Reading |
| --- | --- |
| One-sided Fisher p ≤ 0.05 on the pooled counts, M not below B in any scenario, and no critical-error type in M that B does not show | **Supports** "better than one agent" on this Library suite, with the stated limits and the measured cost. With 12 runs per arm this needs about five more conforming M results than B, for example 11 against 6 (or 12 against 8). |
| M ahead, but the first row does not hold | A **tendency** in Markitect's favor, not established. |
| D = 0 | **No quality advantage shown.** If M is also slower or uses more tokens, the simpler baseline did as well for less, which counts against the claim. |
| D < 0 | **Counts against Markitect**: one agent delivered more conforming results. |

Also against Markitect, whatever D is: an M publication that fails the hidden assessment (unsupported success), M false blockers, M unjustified changes or failed strict no-change where B has none, and M technical failures, which already count as non-conforming M results. Failures stay in the record. If the data do not show an advantage, the results document says so plainly.

## 9. Validity threats and mitigations

| Threat | Mitigation |
| --- | --- |
| The representations differ in content | Mechanical verbatim rendering with four listed substitutions and a check against the model at F (section 4) |
| B lacks the preservation duty Markitect's harness gives | The same duty in B's prompt, quoted side by side |
| Ceiling effect on plain behavior | Hidden probes for easy-to-miss obligations; change, no-change and subtle-seed scenarios; tie stated as a likely result |
| Small N | Fixed N, exact intervals, a rule that needs a large difference; descriptive language otherwise |
| Order and time effects (provider load, account limits) | Counterbalanced blocks; start and end times recorded; quota pauses recorded |
| Dependence within chains | Chain-level sensitivity count; p labeled descriptive |
| Assessor bias | Oracle and probes mechanical and fixed before runs; rubric reviewer blinded, of another model family, evidence-cited; analyst does not re-judge findings |
| Arm hints in blinded trees (package layout from M's ownership, registry-shaped READMEs) | Objective rubric items with citations; limit reported |
| S4 base written by an M run | Byte-identical for both arms; neutral single root commit; neither writer has memory of it. It carries Markitect's package layout into B (its registry is dropped for B); this is a stated limit |
| Seed leakage through history, messages or Notes | Single neutral root commit, neutral identity and branch, fresh state directories, as step 4 learned from its earlier seed runs (section 11) |
| Hidden material reachable by a writer | Writers see only `/work` and their home; `tools/` and evidence never mounted |
| Aggregate compute differs | Per-session ceilings equal; aggregate is the treatment; actual use reported |
| Transport differs (`exec` versus `app-server`) | Same binary, version, model, effort and switches; each arm uses Codex the way it normally would |
| Translation of the owner's instruction | Original quoted; one language in both arms |
| The analyst is not blinded | Mechanical computation by the predeclared rule; all raw outcomes published |
| Harness or environment failure mistaken for a method result | Classified per [measurement](../guides/measurement.md#evaluating-the-method-separately-from-its-runtime): runtime failures of Markitect are product findings; harness failures are limitations; replacement only by the rule in section 7 |
| Old evidence was helped by a visible seed message | Not reused; all candidates are new |

## 10. Human attention

The measurement guide counts authoring and maintaining both representations from one predeclared start. Here both derive from the same owner-confirmed business rules: M1's authoring effort is historical and unavailable, and B's document was rendered mechanically from M1, so its authoring cost in practice, where a person would write it, is also unavailable. Both are reported as unavailable, never as zero. M's execution controls and run configuration are setup for M; B's prompt and fixture are setup for B; the study's own preparation is research effort, recorded separately. During runs both arms receive no human input; every operator action (start, freeze, abort) and any intervention is logged per run.

## 11. Division of work

The step-4 session ("Step 4: native requalification") runs every M Apply with its controller and packets on F and retains its evidence; this session runs every B session, every assessment of both arms and the analysis. Before each run the operator confirms the engine is free; after it, the operator reports the run finished. Arm M follows the step-4 packets unchanged; the step-4 lessons about visible seed messages, fresh state and exact-string deadlines apply to both arms. Per M run, step 4 hands over the start bundle and commit, the final candidate bundle and commit, the Apply report, timings, message counts and the transcript location.

Step-4 runs outside this protocol (for example the P4 ordinary seed) are stability evidence only and are not counted here.

## 12. Limits, abort and cleanup

| Item | Value |
| --- | --- |
| B session | 2,700 s; helper containers 120 s each; metadata 20 s; kill and remove 8 s |
| M Apply | The step-4 limits (packet 3,300 s, Apply 2,700 s, 128 messages, 600 s per message) |
| Assessment | Oracle 1,200 s, probe build and run 300 s each, own build and test 120 s each |
| Continue | Every outcome is recorded, failures included, and the series continues while the environment is clean: no residue, source, bound bytes and login unchanged. |
| Abort the series | Drift of any bound byte (packets, prompt, tools at R, image Id, login fingerprint); residue after cleanup; environment doubt; a login secret in retained B outputs (quarantine); an assessor defect that makes runs incomparable; a product defect that would fail every later run. Everything is kept and the coordinator receives a diagnosis. A product fix starts a new series on a new source, under a new protocol. |
| Account limit or token expiry | Pause and resume in the same order; the gap is recorded. Markitect does not qualify login refresh, and B's copy has no refresh token either, so expiry is an environment limit for both arms. |
| Cleanup | B: the runner removes the login copy before capturing, then exactly its two labeled volumes; census before and after must be empty. M: the step-4 cleanup rules. Assessment: its own labeled volume. |

**Budget.** Expected serial engine time about 7 hours: M 12 Applies about 3.5 h, B 12 sessions about 1.5 h, 24 assessments about 2 h. Hard ceilings, never all reached: M 12 × 55 min, B 12 × 50 min, assessments 24 × 30 min, about 33 h. Messages: M at most 128 per Apply (ceiling 1,536, expected about 500); B exactly one input message per run (12), its internal tool calls bounded only by the deadline and reported. Claude: four blinded rubric reviews and one independent results review.

## 13. Retained evidence

Root `C:\Users\Consiliari\Documents\Codex\Markitect-V1-Comparison\<F first 12 hex>\`, outside the repository. Per B run: fixture bundle and its build record, `result.json`, every step's argv, exit, duration and logs, the JSONL transcript, the Codex session files, the raw working tree, the capture bundle and `files.sha256` written last. Per assessment: `assessment.json`, oracle and probe reports, own-check logs, census, blinded copy. M evidence stays where step 4 retains it. The results document, `docs/validation/v1-comparison-results-<date>.md`, lists every run, including failures and replacements, with its raw outcome.

## 14. What this study cannot show

It covers one small greenfield Library project, one model and effort, one provider and one machine. It cannot show benefits in larger projects, cross-area composition defects, sustained coherence over many changes, human-attention savings in real use, other providers, or reliability beyond the stated intervals. A positive result supports the thesis on this suite only; a tie or a negative result does not refute the method for larger projects, but it removes this suite as evidence for it.
