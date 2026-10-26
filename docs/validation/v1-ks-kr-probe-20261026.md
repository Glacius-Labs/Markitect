# K-S against K-R: exploratory drift probe on Phase C material — 26 October 2026

This record covers an exploratory probe without Markitect. It asks whether one coding agent that maintains its own documentation from change tickets (K-S) drifts away from correct behavior and true documentation, compared with the same agent receiving a re-rendered specification at every step (K-R). The probe is section 10 of the coordinator's design draft for the follow-up consistency study (`folgestudie-konsistenz-v0.2.md`, sections 4 and 10, not in the repository). The owner approved it on 26 October 2026 in the coordinator session, verbatim: „Ja, ich bestätige die Vorab-Probe K-S gegen K-R (Folgestudie §10, 3 Ketten je Arm, ca. 60 M Tokens). Starte sie jetzt als eigene Session unter einem begrenzten Dispatch.“ (English translation: "Yes, I confirm the early probe K-S against K-R (follow-up study §10, 3 chains per arm, about 60 M tokens). Start it now as its own session under a bounded dispatch.") The coordinator dispatched it under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). Everything here is descriptive: the probe has no test and no confirmatory reading.

## Result

- **No drift in either arm.** All 60 counted runs completed on their first attempt, and every candidate passed every hidden case of its step: 54 at W0, rising to 63 at C8 and C9.
  - No failed probe, behavior or invariant family, so no missed obligation, no regression and no persistent unmet load. The E5 proxy is 0 in all six chains, at C9 and at C8.
  - Change discipline held at C3, C7 and C9 in every chain. At C7 (no behavior change) K-R changed nothing and K-S changed only its own `requirements.md` and 2–4 lines of its README. At C9 (no change) neither arm changed anything.
  - Own build and tests passed in both limit profiles in every run.
- **The documentation stayed true.** In 36 blinded doc-truth quizzes at C3, C6 and C9, no answer was stale, conflicting or wrong.
  - With all Markdown documentation, including `requirements.md`, both arms answered 100 % of the questions correctly at every point.
  - With the agent-written usage documentation alone (`realization/README.md`), K-S answered 7–26 of 12–30 questions and K-R 6–23. Every other answer was "not stated".
  - K-S's self-maintained `requirements.md` contained none of the stale phrases searched for (post hoc, below).
- **K-S cost more.** K-S used 54.1 M provider-reported tokens and 4,779 s of session time; K-R used 37.1 M and 3,826 s. Over C1–C9 alone, K-S used 1.50 times K-R's tokens and 1.35 times its time.
- **Hygiene:** no residue and no credential hit in any run. The login file was unchanged across all runs. Every run and every assessment has a recorded cleanup check.
- **Budget:** 91.2 M Codex tokens in total, about 50 % above the dispatch's estimate of about 60 M. The dispatch set no token cap, only the 90 % quota guard. The weekly quota went from 52 % to 60 %. The quiz agents' Claude tokens are not counted here.
- **Reading, within the caveats below:** with tickets this precise, over ten steps of this material, one agent that maintains its own documentation from tickets did not degrade relative to one that receives the maintained specification. Section 10 of the design names this case: "If K-S does not degrade relative to K-R, the claim 'Markitect keeps the project consistent' is in trouble, whatever the hierarchy does." Both arms sit at the ceiling with zero variance, so the probe gives the follow-up study no effect size, and it says nothing about harder tickets or longer chains.

## Binding

| Item | Value |
| --- | --- |
| Source | Phase C materials freeze v2 = R_C `4840fb20f52c1e0e0367e909d833919ca8a215c6` ([protocol C2](v1-comparison-protocol-20261024.md#c2-materials-and-sources)), unchanged on Main since. K-S's tickets are byte copies of `tools/comparison/phase-c/requirements/c1.md`–`c9.md` at R_C. Both arms' W0 document and K-R's step documents are byte copies of Phase C's packet `packet-4840fb20f52c` (`spec/requirements-<step>.md`), checked against its `SHA256SUMS` |
| Runner | A copy of Phase C arm B's frozen runner (`249f8ab7…`, [protocol C8](v1-comparison-protocol-20261024.md#c8-frozen-study-infrastructure)); copy `13f5f995833ad75670c23a811a40d854321e85663933ee6b0a9d7e4c14530660` |
| The one configuration change, both arms | The switches `-c project_doc_max_bytes=0` and `-c project_doc_fallback_filenames=[]` are removed, so Codex reads the project `AGENTS.md` with its defaults. Mechanics only: `fixture new --agents FILE` writes `AGENTS.md` into the W0 input, `run --prompt FILE` takes the step's message, `run --arm` labels the result, and the lock holder is `ks-kr probe`. A dry run against the runner's fake engine confirmed that the Codex command line equals the frozen one apart from the two removed switches |
| Provider and image | Codex CLI 0.162.0 in `sha256:3d75d81f438e71254d01f013cd35e0c61f9eecfb8abba723b34e12158c33e943` (the F_final image of Phase C), `gpt-6-sol`, effort `high`, the dedicated ChatGPT runtime login with its refresh token removed. Arm B's session hardening, network, switches and six-Manager aggregate ceiling; deadline 5,400 s per session. Windows Git Bash with Docker Desktop as the host, as Phase C arm B ran |
| Project `AGENTS.md` | Identical for both arms, at the repository root of both W0 inputs, sha256 `1b8705a87169ca7e55355daa8df67ba3567312107499d9297e0e94d88f0c3f84`; full text below |
| Assessor | A clean clone at R_C: `assess_candidate.py` `a6a953d9…`, hidden Makerspace oracle tree `663f6eea…`, Library oracle transport from F `e11e6cff…`, as Phase C. No blinded review |
| Family catalog | Phase C's ten positive controls on the reference (`e2e-v2/assess-<step>/assessment.json`), hashes as in Phase C's `FREEZE.sha256` |
| Freeze | Every run input and run script, the quiz questions and prompt, and `probe/ANALYSIS.md` are in `probe/SHA256SUMS` (54 files, sha256 `57554fe861400fe50dfdda6d791adf2886cab97cb03362025f72da4991fa5896`), checked before every step. The dispatch record `DISPATCH.md` (`5ebd31b8…`) and the predeclared measures `probe/ANALYSIS.md` (`a907198e…`) were written before the first counted run. The analysis scripts came later and are not in the freeze ([Deviations](#deviations)) |
| Shared engine | `NATIVE-RUN.lock` for every run and assessment. No other holder took it during the series, and no run waited |

The `AGENTS.md` both arms received:

```text
# Working agreement

- Keep the code, the automated tests and the documentation consistent with each other and with the current requirements.
- When a change alters behavior, update every affected part in the same change: code, tests, usage documentation and examples. Remove what the change makes obsolete.
- Before you finish, build the program, run the whole test suite and fix what fails.
```

An uncounted live check before the series (one session of 5.6 s, 12,659 input and 5 output tokens) showed that Codex injected this text verbatim as `# AGENTS.md instructions for /work`. It also showed that a message holding two change documents, with quotes, backticks and Unicode, arrived byte-exact through `docker.exe`.

## Design

| | K-S | K-R |
| --- | --- | --- |
| W0 | a new repository with `requirements.md` = `requirements-w0.md` and `AGENTS.md`; arm B's prompt (`8a6c99a7…`); a new thread | the same |
| C1–C9 input | its last deliverable result, unchanged: it keeps its own `requirements.md` | its last deliverable result plus one commit that replaces `requirements.md` with the step's re-rendering, as Phase C arm B. C9's rendering equals C8's, so C9's input is the C8 result unchanged |
| C1–C9 message | a fixed head (`d62303bf…`) followed by the step's change document verbatim | arm B's prompt |
| Thread | resumes the last deliverable result's Codex thread | the same |

- **The K-S head.**
  - It lifts arm B's "do not change requirements.md". K-S is to deliver the change in code, tests and usage documentation and to update `requirements.md`, which it now maintains.
  - It keeps arm B's preservation duty, adapted from `requirements.md` to the change request and the requirements, and arm B's reporting duty word for word. It drops arm B's "If an implementation already exists, first reconcile it with requirements.md".
  - It adds a fixed section index. The change documents cite the numbered sections of `w0.md`, while K-S's `requirements.md` is the unnumbered rendering with the same sections in the same order; its operation index is the rendering's permission matrix and audit target list. This aid is a recorded deviation from "only the change documents".
- **Backlog rule:** a K-S step without a candidate would have handed its ticket over again with the next one. No step lacked a candidate, so the rule never applied.
- **Order:** the steps in order; at each step chains 1 to 3, one K-S/K-R pair per chain. K-S went first when the step index (W0 = 0) plus the chain number was odd, 15 of 30 pairs.
- **Attempts:** one per step, with a repeat only for a proven infrastructure failure before any candidate (Phase C's C4 rule). None was needed.
- **Abort rules:** a credential hit, container or volume residue, a weekly quota reading of 90 % or more (read before each pair), or a third infrastructure failure at one step. None fired.

## Runs

The series ran from 17:39:45Z to 21:08:16Z on 26 October 2026. Each cell gives hidden cases passed of those planned, session seconds, and provider-reported tokens (input including cached, plus output) used by that step. A resumed thread reports cumulative usage, so a step's own tokens are its total minus the total of the result it resumed, as Phase C counted them. † marks a run with a loud quietness sample; ‡ marks a run during which no quietness sample was taken (samples come once a minute), so its quietness is unknown. Every run is attempt 01 and completed; every candidate was assessed once.

**K-S:**

| Step | Chain 1 | Chain 2 | Chain 3 |
| --- | --- | --- | --- |
| W0 | 54/54, 386 s, 1.57 M | 54/54, 391 s, 1.29 M | 54/54, 373 s, 1.00 M |
| C1 | 55/55, 123 s, 1.17 M † | 55/55, 107 s, 1.03 M † | 55/55, 127 s, 1.07 M † |
| C2 | 57/57, 316 s, 4.36 M † | 57/57, 316 s, 3.51 M † | 57/57, 318 s, 3.17 M † |
| C3 | 58/58, 115 s, 2.20 M | 58/58, 94 s, 1.40 M | 58/58, 96 s, 1.47 M |
| C4 | 59/59, 207 s, 3.54 M | 59/59, 164 s, 2.41 M † | 59/59, 172 s, 2.03 M † |
| C5 | 61/61, 147 s, 1.31 M † | 61/61, 128 s, 0.90 M † | 61/61, 175 s, 1.18 M |
| C6 | 62/62, 99 s, 1.50 M | 62/62, 92 s, 1.10 M | 62/62, 130 s, 1.85 M † |
| C7 | 62/62, 94 s, 1.42 M | 62/62, 66 s, 1.02 M | 62/62, 74 s, 1.23 M |
| C8 | 63/63, 170 s, 3.83 M | 63/63, 106 s, 2.26 M | 63/63, 136 s, 2.84 M |
| C9 | 63/63, 21 s, 0.99 M ‡ | 63/63, 15 s, 0.45 M ‡ | 63/63, 22 s, 0.96 M |

**K-R:**

| Step | Chain 1 | Chain 2 | Chain 3 |
| --- | --- | --- | --- |
| W0 | 54/54, 424 s, 1.58 M | 54/54, 365 s, 1.07 M | 54/54, 347 s, 1.05 M † |
| C1 | 55/55, 107 s, 1.24 M | 55/55, 158 s, 1.44 M † | 55/55, 113 s, 1.22 M † |
| C2 | 57/57, 200 s, 1.84 M † | 57/57, 225 s, 2.58 M † | 57/57, 250 s, 2.35 M † |
| C3 | 58/58, 68 s, 1.17 M | 58/58, 78 s, 1.23 M † | 58/58, 77 s, 1.06 M |
| C4 | 59/59, 91 s, 1.12 M † | 59/59, 112 s, 1.85 M † | 59/59, 122 s, 1.78 M † |
| C5 | 61/61, 113 s, 2.01 M † | 61/61, 127 s, 0.81 M | 61/61, 129 s, 1.48 M |
| C6 | 62/62, 82 s, 2.18 M | 62/62, 68 s, 0.78 M | 62/62, 100 s, 0.58 M † |
| C7 | 62/62, 39 s, 0.40 M | 62/62, 29 s, 0.46 M | 62/62, 33 s, 0.33 M ‡ |
| C8 | 63/63, 88 s, 0.63 M | 63/63, 111 s, 2.17 M | 63/63, 90 s, 0.98 M |
| C9 | 63/63, 26 s, 0.37 M | 63/63, 28 s, 0.81 M | 63/63, 24 s, 0.57 M ‡ |

Per-step means over the three chains:

| Step | K-S s | K-R s | K-S tokens (M) | K-R tokens (M) | K-S lines outside protected inputs | K-R lines outside protected inputs | K-S changed `requirements.md` lines |
| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: |
| W0 | 383 | 379 | 1.29 | 1.23 | 1,416 | 1,410 | 0 |
| C1 | 119 | 126 | 1.09 | 1.30 | 87 | 87 | 17 |
| C2 | 317 | 225 | 3.68 | 2.26 | 307 | 247 | 61 |
| C3 | 102 | 74 | 1.69 | 1.15 | 77 | 76 | 17 |
| C4 | 181 | 109 | 2.66 | 1.58 | 131 | 110 | 98 |
| C5 | 150 | 123 | 1.13 | 1.43 | 127 | 86 | 31 |
| C6 | 107 | 83 | 1.48 | 1.18 | 56 | 44 | 29 |
| C7 | 78 | 34 | 1.22 | 0.40 | 3 | 0 | 87 |
| C8 | 137 | 96 | 2.98 | 1.26 | 94 | 82 | 19 |
| C9 | 19 | 26 | 0.80 | 0.58 | 0 | 0 | 0 |

"Lines outside protected inputs" are the added and removed lines of every changed path except the assessor's protected inputs, against the step's input: code, tests and the README. `requirements.md` is a protected input and is counted apart. K-R never changed `requirements.md`.

## Drift measures

The measures are those `probe/ANALYSIS.md` predeclared, with two rules taken from Phase C and the analysis script ([Deviations](#deviations)). Every catalog family counts as one S1/S2 obligation, because Phase C has no per-obligation severity catalog.

| Measure | K-S chains 1 / 2 / 3 | K-R chains 1 / 2 / 3 |
| --- | --- | --- |
| Missed probe families, sum over ten steps | 0 / 0 / 0 | 0 / 0 / 0 |
| Failed behavior and change families, sum | 0 / 0 / 0 | 0 / 0 / 0 |
| Invariant signals, sum | 0 / 0 / 0 | 0 / 0 / 0 |
| Conforming steps (technical, hidden cases, own checks, no other protected change, change discipline) | 10 / 10 / 10 | 10 / 10 / 10 |
| E5 proxy at C9 and at C8 (probe, behavior and invariant classes) | 0 / 0 / 0 | 0 / 0 / 0 |
| E5-c and E5-u | 0 / 0 / 0 | 0 / 0 / 0 |
| Accumulation slope over C1–C9, failed families per step | 0 / 0 / 0 | 0 / 0 / 0 |
| Regressions | 0 / 0 / 0 | 0 / 0 / 0 |
| Change census differences at C3, C7, C9; changed paths at C9 | none | none |
| Files outside the billing area at C7 (Phase C's word rule, on the diff without `requirements.md`) | none | none |

For K-S the assessor's recorded `mechanical_pass` is false at C1–C8, because K-S changed `requirements.md`, the assessor's protected input. That change is the arm's design, so the conforming count leaves `requirements.md` out of K-S's protected check. Every other recorded `mechanical_pass`, including K-S's at W0 and C9, is true.

## Doc truth

- **The questions:** 30 frozen closed questions about behavior (`probe/doc-truth/questions.json`, `9b5ad5de…`). Each has its truth at the step and the stale value that documentation missing the change would show. They target the changes C1–C8, plus four unchanged controls.
- **The quizzes:** at C3 (12 questions), C6 (23) and C9 (30), the candidate's Markdown files except `AGENTS.md` went into a directory named by a random token, in two variants.
  - Each of the 36 quizzes was one fresh Claude Code general-purpose subagent: model `claude-sonnet-5-5`, effort high. It got one instruction: read its token's prompt file, the frozen prompt (`c6920021…`) with that step's questions, and do what it says.
  - The arm key was moved out of the quiz folder while the agents ran. Blinding otherwise rested on the instruction, because `questions.json` with its answer key was readable two directories up.
  - The 36 transcripts are retained. An audit of every tool call (`audit_transcripts.py`) found 180 named paths, all within the agent's own prompt, documentation directory or answer file: no agent read the key, the questions file or another agent's files.
- **Scoring:** mechanical, by `doctruth.py` (`a019ed2e…`). A self-check of its classification (`scorer_selfcheck.py`, 18 cases of stale, current and absent answers, all classified as expected) ran after the scoring and is retained.

| Point | All docs: K-S chains 1 / 2 / 3 | All docs: K-R chains 1 / 2 / 3 | Usage docs only: K-S chains 1 / 2 / 3 | Usage docs only: K-R chains 1 / 2 / 3 |
| --- | --- | --- | --- | --- |
| C3 (12) | 12 / 12 / 12 | 12 / 12 / 12 | 8 / 7 / 10 | 7 / 8 / 6 |
| C6 (23) | 23 / 23 / 23 | 23 / 23 / 23 | 17 / 19 / 19 | 13 / 16 / 15 |
| C9 (30) | 30 / 30 / 30 | 30 / 30 / 30 | 24 / 26 / 26 | 21 / 23 / 23 |

Each cell counts answers matching the step's truth. Every other answer was "not stated". No answer in any quiz was stale, in conflict or otherwise wrong. In every quizzed candidate the documentation is `requirements.md` and `realization/README.md`. K-R's `requirements.md` is the re-rendering, true by construction, so its all-docs result says only that its README did not contradict it.

## Cost and conditions

| Arm | Session time | Input tokens (cached) | Output tokens | Total |
| --- | ---: | ---: | ---: | ---: |
| K-S | 4,779 s | 53,751,182 (52,625,280) | 326,859 | 54.08 M |
| K-R | 3,826 s | 36,871,932 (36,046,592) | 240,097 | 37.11 M |

- At W0 the arms cost about the same. From C1 on K-S cost 1.50 times K-R's tokens and 1.35 times its time: it also rewrote its `requirements.md` (1,077 changed lines over the three chains, a mean of 87 at C7), and at C2, C4 and C8 it did more work per step.
- **Quota:** the account's weekly window read 52 % before the first run and 60 % after the last. The window resets on 1 November 2026.
- **Quietness:** K-S had 17 quiet runs, 11 with a loud sample and 2 unsampled; K-R had 16, 12 and 2. The loud samples show other sessions' host test suites (`go.exe`, `gitrepo.test.exe`, `codexappserver.test.exe`) or Markitect processes. The probe did not gate on quietness, as dispatched, so cost is descriptive only.

## Observations outside the predeclared measures

- **Wording drift without fact drift (post hoc).** At C9, K-S's own `requirements.md` has 933–946 lines against the re-rendering's 994. Git's line diff against the re-rendering counts 97, 102 and 112 added and 158, 160 and 160 removed lines in chains 1, 2 and 3. A search of every arm's C9 documentation found none of these stale phrases (date examples that add 365 days are not certification validity):
  - certification expiry (`expiresOn`, the `expired` status, 365-day validity);
  - 30-minute granularity;
  - Sunday closed;
  - the W0 weekday and Saturday hours.
- **Phase C arm B is not a comparison arm here.** Phase C's arm B ran without project docs and at an earlier time. In one of its three chains it carried an audit-target defect through all ten steps, missing `P-AUDIT-ACCEPTED` at every step ([Phase C results](v1-comparison-results-phase-c-20261026.md#secondary-outcomes-c6-predeclared)). None of K-R's three chains had a defect. The probe was not designed to compare the two arms, and one defective chain in three cannot separate an `AGENTS.md` effect from variance.

## Deviations

- **The section index in K-S's message**, an aid beyond "only the change documents" (Design above). It was recorded in the dispatch before the first run.
- **K-R received no ticket.** Design section 2 gives K-R "the change ticket plus the full requirements". The dispatch followed section 10 and Phase C arm B, so K-R received only the re-rendered requirements and arm B's prompt. The arms therefore differ in the form of the change request as well as in the source of their knowledge, and the cost and change-discipline comparisons carry that difference.
- **Analysis scripts after the freeze.** `probe/doc-truth/doctruth.py` was written at 17:42Z, after the series had started and before any quiz input existed. `probe/analyze_probe.py` was written during the series. Its last change during the series, at 20:50Z, only skipped rows still running or awaiting assessment. After the review it also made the C9 gate count protected inputs and build residue; the ledger and summary are byte-identical before and after that change. Neither script is in `probe/SHA256SUMS`; their final hashes are under [Retained evidence](#retained-evidence).
- **Two rules not in `ANALYSIS.md`:**
  - the C7 billing-area word rule, taken from Phase C (protocol C5);
  - K-S's `requirements.md` left out of the protected-input check in the conforming count, a consequence of the K-S design.
  Both are applied identically to both arms where they apply. At the ceiling neither changes any outcome.
- **Budget.** 91.2 M Codex tokens against the dispatch's estimate of about 60 M. The dispatch set no token cap, and the quota guard did not come near 90 %.
- **Post-hoc additions:** the stale-phrase search, the diff of K-S's `requirements.md`, the quiz transcript audit and the scorer self-check.

## Caveats

These were recorded before the run (design section 10 and `ANALYSIS.md`):

- The neutral change documents are far more precise than real tickets. They state what must not change, which favors K-S.
- The material was seen before (Phase C), and the probe is exploratory, without a confirmatory reading.
- The doc-truth check is not calibrated per the evaluation contract's section 8, and it stays descriptive.
- K-S's message adds a fixed section index, an aid real tickets would not carry.
- Severity is not graded.

Added after the data:

- Both arms sit at the ceiling of the hidden oracle and of the quiz. The probe therefore bounds drift only at this length, with these tickets, this agent and this oracle. It says nothing about vaguer tickets, renames across areas, removals or longer chains, which the follow-up design plans.
- The quiz covers 30 facts, mostly at the changes. It cannot see a stale statement about another fact, test semantics or comment drift, and no blinded review ran.
- **Neither arm relied on its documents alone.** Both resumed one Codex thread per chain, as Phase C arm B did, so each carried its earlier steps in context. Each chain's thread was compacted once: K-S's at C4 in all three chains, K-R's at C7, C5 and C6 in chains 1, 2 and 3. After compaction Codex injected `AGENTS.md` again. A fresh session per ticket would test self-maintained documentation harder.
- Three chains per arm, one model and effort, one machine.

## Retained evidence

Outside the repository, in `C:\Users\Consiliari\Documents\Codex\Markitect-KS-KR-Probe`:

- `DISPATCH.md` and `probe/`: the dispatch, the frozen inputs and scripts, `SHA256SUMS`, `ANALYSIS.md` and the dry run;
- `checks/config-01`: the live configuration check;
- per run, `runs/<arm>-c<chain>-<step>/`: fixture, message, attempt with result, transcript, Codex sessions and bundle, driver log and cleanup check;
- `assess/` and `assess-logs/`: the assessments and their cleanup checks;
- `quiet/`: the quietness samples;
- `series.log`: the series log;
- `analysis/`: `ledger.json` `aceb4689…` and `summary.json` `6338b999…`, written by `probe/analyze_probe.py` `e85aeb37…`;
- `probe/doc-truth/`:
  - the blinded exports, prompts and answers;
  - `key.json` `8e199e57…` and `scores.json` `57f8bb43…`, written by `doctruth.py` `a019ed2e…`;
  - the 36 quiz transcripts and their audit (`audit_transcripts.py` `06ff3320…`, `transcripts-audit.json` `6ad461b0…`);
  - the scorer self-check (`scorer_selfcheck.py` `e1b6345d…`, `scorer-selfcheck.json` `cfcae960…`).
