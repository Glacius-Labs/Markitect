# Survey: earlier plans and assessments

AI-generated, read-only, 11 October 2026, against main `93b33923`. Input for IDEA-01. Not a decision.

## Summary

This survey checks the ideas and claims of eight earlier documents against current main, the [vision](../../vision.md) and the [concept register](../../concepts/register.md). Under [DEC-015](../../concepts/register.md#dec-015-a-fresh-roadmap-earlier-plans-are-inputs) none of these plans is adopted; each row says what became of one idea. Of 116 rows, 34 are done on main, 65 are still relevant and 17 are obsolete. Most still-relevant rows already have a backlog package; three new packages are proposed. The most important still-relevant items:

- **The core claim is still unmeasured.** No package measures impact recall against an independent list of must-change places. The independent assessment's cheapest test (impact against `rg`, no model call) is proposed as a new package.
- **Model upkeep and human review time** are the main risks to the promises. They need explicit metrics in the PLAY-07 protocol and in IDEA-03.
- **Impact is very broad.** One statement change reached 9 of 10 statements and 6 of 6 Managers in the Shop example when the drafts were written. KG-02 implements DEC-021 to DEC-023; a width report per example is in no acceptance yet.
- **Cost accounting is a lower bound.** Native helper starts count as unknown, so cost questions (OQ-003, PLAY-06, PLAY-07) cannot be answered. Proposed as a new package.
- **Government hooks** (owner channel, reviewer per concern, decision classes, a term mapping) belong to GOV-01; the assessment's Government overbuild findings are obsolete under DEC-004.

"Done on main" cites a pull request, file or function, or a register decision that settles the question. Line numbers in the inputs refer to older states.

## 1. Independent assessment

Input: `independent-assessment-20261010.md` on `archive/wip/government-assessment-untracked-docs-20261010`, in German. Finding IDs (K, A, E, U, P, X) are the assessment's own.

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| Core thesis untested; no Markitect arm passed the first station (K1, E1, E7, E12, EX1, EX2) | still relevant | PLAY-07; register DEC-008 | Those attempts allow no conclusion either way. |
| Stage 0: impact against `rg` on a mid-size repository with frozen must-change lists, no model call (§11 rec 6, E11) | still relevant | NEW 1 | Nothing on main measures recall against an independent list. |
| Stage 1: paired agent study, same model, script as study lead, success criteria and stop rules (§11, K10, EX3) | still relevant | PLAY-07 (protocol needs the owner) | Thresholds are proposals without variance data. |
| Harness as a script, frozen product, isolation per run, one budget (rec 10, E3 to E6, UX1) | still relevant | Container harness on main (PLAY-01, PR 103); PLAY-04, PLAY-05 | Product isolation is not claimed: owned workspaces are not an OS sandbox. |
| Same model executes, reviews and evaluates; correlated reviewers (K4, KX1, E10) | still relevant | PLAY-05, PLAY-06; register ASM-005, OQ-004 | RUN-01 (PR 104) made per-role models possible. |
| Cheaper models never tested (K5, A5, E9) | still relevant | PLAY-06, PLAY-07; register DEC-007, ASM-003, OQ-003 | |
| Helper and child usage not aggregated (U10, rec 9) | still relevant | NEW 2 | `projectrun/cost.go` counts each native helper start as unknown. |
| "Nothing forgotten" holds only for declared mappings (K2, KX3) | still relevant | IDEA-02 (ENH-001); coverage per DEC-006 is on main | |
| Impact hull very broad; separate change set from context set; report width per example in CI (U2, rec 7) | still relevant | KG-02 (DEC-021, DEC-023) | The width report is in no acceptance. |
| Fix a binding granularity rule (K3, rec 8, question 2) | obsolete | DEC-005: granularity is the author's choice | The upkeep risk stays (next row). |
| Model upkeep may become a cleanup trap; 731 model lines for 73 code lines (U6, A8, claim 15) | still relevant | IDEA-03 (ENH-002), PLAY-07 | DEC-008 keeps upkeep cost as a real risk. |
| Review load moves from code to model; human minutes never measured (K9, claim 18) | still relevant | PLAY-07 (DEC-009, Measurement) | |
| Namespace hierarchy fits cross-cutting obligations poorly; context request missing (K6, C19) | still relevant | Register OQ-006; PLAY-07 | No package; PLAY-07 can mark each miss as cross-cutting or missing context. |
| A semantic global check repeats the scaling problem (K8, claim 8) | still relevant | Register CPT-001; Full Verify in `projectrun/full_verify.go` | A caution for any future global check; no package needed. |
| Unstated assumption: forgetting, not misunderstanding, is the main failure (claim 17); cheaper alternatives such as fitness functions never compared (K12) | still relevant | PLAY-07 failure classes | Not in the register. The vision keeps "owner document plus architecture tests" as an alternative. |
| Literature: MDE models core parts; spec-as-source inherits MDD and LLM downsides; trace-link upkeep cost; multi-agent token cost (§4) | still relevant | Not in the register | Counter-evidence for PRM-004, PRM-005 and PRM-007. Sources were not re-checked. |
| Test cases below the problem size; brownfield unproven (E2, U5, K7, claim 20) | still relevant | PLAY-02 long-running case (PR 128), PLAY-07, NEW 1 | |
| Real runs only on one provider; Claude adapter had no tools (U3) | still relevant | RUN-03; process executor done (RUN-01, PR 104) | |
| Integration and agent behavior, not mechanics, are the bottleneck; A01 passed after 27 failures (U4, claim 19, UX2, P7) | still relevant | Register OQ-007; TEST-04, CI-04 | DEC-010 separates method and runtime. |
| Three execution stacks (U1, rec 12) | still relevant | ARCH-04 done (PR 118); ARCH-09, ARCH-15; Government code is reference only (GOV-01) | |
| `runOrResume` has about 1,100 lines (U7) | still relevant | ARCH-06, CLI-03 | PR 162 confirms more than 1,000 lines on main. |
| Markitect does not model itself (UX3) | still relevant | ARCH-07 | |
| Government overbuilt: unanimity, cabinet, court, queue without continuous operation, configuration load (A2 to A6, E8, K13, claims 11 and 12) | obsolete | DEC-004 (the former arm is not retested; departments are later designs); DEC-019 | The v1 contract has no courts. Continuous operation stays VIS-001, parked. |
| Government package not marked historical; `docs/README.md` called it current (A1, A7, AX1, AX3, rec 2) | done on main | The Government design package is not on main; status in DEC-012 and DEC-019 | |
| Government and design use two vocabularies without a mapping (A10) | still relevant | GOV-01 | Section 4 lists a draft mapping. |
| Protection against self-empowerment rests on unauthenticated acceptance; no owner channel (AX2, question 3) | still relevant | GOV-01 (v1 contract: owner channel, reserved statements) | |
| Owner goals untracked; concept notes mix owner words and AI derivations (A9, KX2, rec 3) | done on main | `docs/concepts/sources/concepts-chat-20261009/`; the register marks the origin of each entry | |
| C01 to C19 coverage matrix and the claims table (U8, §2) | done on main | Register "Imported collections"; DEC-003 (borrowed compiler) | |
| Status documents contradict the ledger (U12, rec 5) | done on main | "ATTEMPTED, NOT PASSED" is gone; [A01 record](../../validation/a01-native-smoke-20261010.md); ledger retired (OPS-05, PR 105) | |
| Process: unreadable coordination record, status in many files, grant machinery, star topology, context loss (P1, P3 to P6, P8, P9, P12) | done on main | Backlog as single status owner (OPS-05); roadmap "How we work" | |
| Evidence commits on PR heads broke CI; slow Windows gate (P2, U11, rec 4) | still relevant | CI-01 (PR 95) and CI-02 (PR 113) done; CONTRIBUTING lacks the push rule: OPS-04 | The roadmap's Integration rule 2 has it. |
| A `STATUS.md` of at most 40 lines (rec 4) | obsolete | The backlog is the single status owner (OPS-05) | |
| Branch and worktree sprawl; binary evidence in Git (U9, P11, rec 13) | done on main | OPS-02, OPS-03 (archive tags) | |
| Convention drift in language, headings and prefixes (P13) | still relevant | ARCH-10 (PR 162 in review); roadmap branch names; DEC-016 | |
| Owner questions decided: departments (1), granularity (2), thesis test before features (6) | done on main | DEC-004, DEC-005, DEC-016 | Main chose a clean main first. |
| Owner questions open: may agents change the model (3), test repository (4), acceptable cost and review time (5) | still relevant | GOV-01 (IDEA-002); NEW 1; PLAY-07 | |

## 2. Knowledge-graph assessment

Input: `knowledge-graph-assessment-20261010.md` on the same archive branch, about `codex/knowledge-graph` at `ab405644`.

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| The idea fits; the branch is not merge-ready; port a reduced core | done on main | DEC-018 (branch frozen as reference) | KG-02 ports the core. |
| Manager view drops list edges; two privacy leaks; a second visibility owner (H1 to H3) | still relevant | KG-02 acceptance: view from Manager context only, no visibility leaks | The branch code is not ported. |
| One test would have caught H1 and H2: the twelve questions as tests | done on main | KG-01 (PR 119): `projectwork/knowledge_questions_test.go` (5 answered, 2 partial, 2 gaps, 3 outside the model) | The "view within Context" test belongs to KG-02. |
| Decision-cycle findings depend on map order (M1) | still relevant | KG-02 | Permutation tests on main cover impact only. |
| Decision impact narrowed without a decision; `decision.actor-out-of-scope` (M5) | done on main | DEC-022: route through the subject; the actor is provenance | |
| IdentityChange, evidence adapters, coverage and history actions, stale-by-default, write lock on read, Windows `isMissing`, opaque MCP errors (M2, M3, M6 to M8) | obsolete | Not ported (DEC-018 reduced core) | If evidence joins come back: freshness per node, no writer lock on read. |
| A separate `knowledge-mcp` server | obsolete | DEC-018: the existing MCP surface | |
| Wrong derivation labels; docs should start with a real question and an example decision (M4) | still relevant | KG-02 | Not in its acceptance. |
| Foreign timeout and schema-limit changes, 43k lines of raw logs, double-encoded `docs/usage.md` | obsolete | Branch not merged; no such encoding error on main; evidence goes into the PR description (roadmap) | |
| RDF at most as an optional export | obsolete | DEC-018: YAML stays canonical | |

## 3. Knowledge-graph refinement plan

Input: `knowledge-graph-refinement-plan-20261010.md` (draft v2, K0 to K6, decisions D1 to D6).

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| The graph explains and classes impact (change or context); sets never shrink (D3) | done on main | DEC-018, DEC-023 | KG-02 implements. |
| Fix the map-order coverage defect and lost unprojected changes first (K1) | done on main | KG-01 (PR 119): `unprojectedChange` and `TestImpactCoverageIsIndependentOfSeedOrder` in `projectmodel` | PR 144 also widens meaning-free edits for now. |
| Seeded model generator and permutation and never-shrink properties (K1) | done on main | KG-01: `TestImpactIsDeterministicOnGeneratedProjects`, `TestImpactNeverShrinksWhenChangesCombine` | |
| Decisions in Report, own Context and Impact, no schema change; a projected Decision routes through its subject (K2, D2) | done on main | DEC-022 | KG-02 implements. |
| Measure first (K0) with stop rules after K0 and K3 | still relevant | NEW 1 | KG-02 started without it. |
| `project affected`: the same closure on one snapshot, before a candidate (Government step B) | still relevant | GOV-01 (input) | In no acceptance. |
| Shared-file widening in impact (D5) | still relevant | KG-02 (owner question) | DEC-023 keeps the sets unchanged, so D5 needs its own decision. |
| Plans and briefings show each Manager's change set with reasons (K4) | still relevant | KG-02; KG-03 decides on dropping context-only Managers (DEC-023) | |
| Graph index and `explain` only if the catalog shows a gap (K5) | obsolete | DEC-018 ports the reduced core and explain unconditionally | |
| Short IDs `Kind:namespace/name` instead of JSON tuples (D6) | still relevant | KG-02 CLI and MCP slice, after CLI-02 | Not checked against the current ID form. |
| Result limit with `complete:false`; benchmark at 5,000 definitions; CI reports class widths | still relevant | KG-02 | Not in its acceptance. |
| Lean process: tests first, one adversarial reviewer, no raw logs, no foreign changes | done on main | Roadmap "Integration"; BUG-01 verifies every finding | |
| Cross-checking views and record joins later (§6) | still relevant | GOV-01, GOV-02 | Parked. |

## 4. Refinement plan

Input: `refinement-plan-20261010.md` (phases 0 to 4, packages F0 to R4, owner decisions ND1 to ND11).

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| Measure the core thesis before building more, in gated phases | obsolete | DEC-015, DEC-016 (clean main first) | Its measurement steps live on in NEW 1 and PLAY-07. |
| F0.1: secure the owner goals and concept files | done on main | `docs/concepts/sources/` | |
| F0.2: status hygiene W1, W4, W5, W7, W8 | done on main | A01 record (W1); CONTRIBUTING "Published top-level init" (W4); AGENT-01, PR 100 (W5); risk register retired, triage report kept (W7); OPS-05 (W8) | |
| F0.2: no commits on a PR head while CI runs | still relevant | OPS-04 | Roadmap Integration rule 2 has it. |
| W3 and ND9: is the classic core compatibility-only? | done on main | DEC-014; ARCH-07, then ARCH-09 | |
| F0.3: Government disposition, term mapping and hooks V1 to V6 | still relevant | Disposition is DEC-004, DEC-012, DEC-019; mapping and V3 to V6 go to GOV-01 | V1 and V2 are DEC-023 (KG-02). |
| F0.4: decision note; status in `follow-up-backlog.yaml` | obsolete | One backlog (OPS-05) | |
| S0.0: check `impact.go` for determinism | done on main | KG-01 (PR 119) | |
| MS0: blind model, replay and seeded cases by a non-Claude author, drift arm, `rg` arm with equal budget, causes Z1 to Z5, Gate 1 | still relevant | NEW 1 | Repository, author and thresholds (ND3 to ND5) are owner choices. |
| MS1a: Codex with and without read-only Markitect, 6 × 2 × 3 runs, blind human review | still relevant | PLAY-07; read-only tools: MCP-01 (DEC-017 D8) | A separate Windows account is obsolete: DEC-013, playground containers. |
| B3.1: digest-neutral `--explain`; routing only under full coverage | obsolete | DEC-023 adds the class to the impact result; dropping context waits for KG-03 | |
| Non-normative granularity guide | still relevant | IDEA-03 (ENH-002); DEC-005 | |
| B3.2 to B3.5: cost, MS1b, second provider, model tiering | still relevant | NEW 2; PLAY-07; RUN-03 and PLAY-06; PLAY-06 | |
| B3.6 No-Go path: a structure tool, `projectrun` in maintenance | obsolete | Contradicts DEC-001 | |
| N1 to N6: self-model, one stack, CLI and MCP asymmetry, split `runOrResume`, KG rest, branches | still relevant | ARCH-07; ARCH-09, ARCH-15; CLI-02 (PR 161), MCP-01, TEST-02; ARCH-06; KG-02, KG-03 | N6 is done (OPS-02, OPS-03). |
| R4.1 to R4.4: verdict record, cross-cutting reviewer, decision classes and mandated model upkeep, owner back channel | still relevant | GOV-01, GOV-02; register IDEA-002, IDEA-006, IDEA-007 | After Wave 2 (DEC-019). |
| R4.5: port GP1 fencing lease, promotion intent and hash journal | still relevant | BUG-01, on a concrete defect only | Guarded Apply is still not a multi-file transaction. |
| R4.6: queue and continuous operation | still relevant | GOV-02; VIS-001 | Parked (DEC-011). |
| Rejected: courts; per-role permission settings by the user | obsolete | Courts stay a parked owner idea (IDEA-006); setup defaults to `:workspace` (`projectsetup/setup.go`) | |
| Way of working: one Claude session, evidence folder outside Git, both systems green before merge | obsolete | Roadmap "How we work"; DEC-013 (Linux gate) | |
| ND1, ND2, ND6 to ND10 | done on main | DEC-004, DEC-019; roadmap; DEC-005; PLAY stream; DEC-018; DEC-014; `docs/concepts/` | ND2 is superseded by the roadmap. |
| ND3 to ND5, ND11: repository, case author, thresholds, budget | still relevant | NEW 1; PLAY-07 (`needs_owner`) | |

## 5. Native runtime simplification

Input: `native-runtime-simplification-20261010.md`, the owner's direction of 10 October 2026.

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| Ordinary setup selects a working native runtime with inherited environment, MXC on Windows and normal tools; users give work, not permissions | done on main | `projectsetup/setup.go` (`:workspace`, MXC default) | |
| Each Manager, helper and reviewer gets its own chat and owned workspace; review gets scratch without changing the candidate | done on main | `projectworkspace`; reviewer guidance in `projectrun/review.go` | |
| Run A01 through ordinary setup and report once | done on main | A01 record: passed once after 27 failures | Reliability stays open: OQ-007, TEST-04, CI-04. |
| The knowledge graph stays separate; studies wait until the product works | obsolete | KG-01 done, KG-02 active, PLAY stream active | |

## 6. Research R01 to R03

Input: [optional research R01 to R03](../../research/optional-research-r01-r03-20261009.md).

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| R01: do not replace the compiler, Managers or Verify and Apply with Jev | done on main | Nothing on main depends on Jev | |
| R01: optional offline claim and evidence classifier, twelve labelled cases first | still relevant | Register IDEA-016; no package | Parked under DEC-011. |
| R02: keep the typed YAML graph; add a derived Explain view | done on main | DEC-018 | KG-02 implements. |
| R02: defer a query language, RDF, SHACL and persistent rename identity | obsolete | DEC-018 | KG-01 pins rename (Q07) as a gap. |
| R02: one explainable chain from work item or decision to Verify and Apply | still relevant | Register IDEA-014; GOV-02 | |
| R03: refine the ten skills and short entry files; no symmetric Rules and Agents trees | still relevant | AGENT-01 done (PR 100); AGENT-02 | |
| R03: native helpers are disabled | done on main | `markitect_start_helper` (PR 89) | |
| R03-F01: versioned YAML defaults and model-bound provider guidance | still relevant | AGENT-03; AGENT-02 (one source for runtime pins) | |
| R03-F01: A/B study of the setup change | still relevant | PLAY-07 | |

## 7. Product Readiness lessons

Input: the [Product Readiness lessons survey](product-readiness-lessons-20261010.md). It already names packages; this table checks its claims on main.

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| What the program left on main | done on main | PR 89 (`f12ffb00`) | |
| Runtime limits: pinned `codex-cli 0.162.0`, no hard helper-depth cap, unknown loaded instructions, explicit permissions, fingerprints in plans | still relevant | RUN-02, MCP-02, RUN-03 | Pin in `codexappserver/process.go`; only `agents.max_concurrent_threads_per_session` is sent. |
| `go test -race` never ran | still relevant | NEW 3 | No workflow passes `-race`. |
| One regression case per observed provider output error | still relevant | TEST-04, RUN-02, BUG-01 | |
| Helper stays `cleanup-pending` after a failed Close | still relevant | BUG-01 | PR 162 confirms it on main. |
| Workspace, helper and recovery limits | still relevant | RUN-02, RUN-04, TEST-04 | |
| Parent integration review ran before queued child rework | done on main | Commit `0c8d3873` | |
| Verify audits lacked review evidence; calling agents used unknown flags | still relevant | TEST-04; TEST-02, AGENT-02 | |
| Windows, CRLF, file modes, MXC and long paths | still relevant | CI-05 (parked, DEC-013); TEST-01 | |
| Native accounting is a partial lower bound | still relevant | NEW 2; PLAY-06, PLAY-07 | |
| Tests break under load | still relevant | TEST-03, TEST-05 | |
| Gates broken by bookkeeping: artifact accounting, YAML under `docs/`, the `ci.yaml` digest, a smoke calling the removed `init` | still relevant | ARCH-07, ARCH-09 | The smoke no longer calls `markitect init`; the digest still lives in `.markitect/modules/pipelines.config`. |
| Records grew to 400 and 2,500 lines | done on main | OPS-05 (PR 105) | |
| Risk register: impact recall and noise (2 of 69 in PR #59), context selection, explanations, upkeep, fossilization, adoption | still relevant | NEW 1; KG-03, IDEA-02, IDEA-03, PLAY-07 | |
| Deprioritize expansions without a recurring human task | done on main | Vision "Feature decision filter" | |
| GitHub and Azure DevOps command adapters; remote Apply prerequisites | obsolete | Never built; ARCH-09 removes the v0.13 consumers (DEC-014) | No current need. |
| Core change bar | done on main | `docs/development/parallel-work.md` | |

## 8. Marketing research log

Input: `docs/strategy/marketing-research-log.md` on `archive/wip/marketing-notes-20261010` (172 lines). Checked briefly.

| Idea or claim (short) | Status | Evidence or package | Note |
|---|---|---|---|
| Owner wording on product, problem and caveats | done on main | Register sources (`owner-product-description-20261009.md`); DEC-001, PRM-001 to PRM-007 | |
| Freedom and strictness, Government, precedents, escalation, briefings, monitoring, Kubernetes or etcd, visualization, gamification, cockpit, Government evaluator | done on main | Register IDEA-002 to IDEA-015, VIS-001, DEC-011, DEC-012 | Recorded, parked. |
| Markdown is not the semantic model | done on main | Register finding `HIST-MARKITECT-20261009-FORMAT-001`; vision "Common misreadings" | |
| Positioning drafts: working identity, audience hypothesis, candidate narrative, "keep the specification effective" | still relevant | No package | Nothing approved. The integrator may record them as a register idea; CPT-004 already holds the specification idea. |
| "The Historian should wake up by itself" | still relevant | LEARN-01 | Not in the register. |

## Proposed new packages

- **NEW 1: Impact recall and width measurement without a model call.** Zone: free (`experiments/`). Why: nothing measures PRM-001 against an independent must-change list; this is the cheapest test of the core claim, and it shows KG-03 whether the change class is narrow enough. Best after KG-02. The owner chooses the repository, the list author and the thresholds. Inputs: independent assessment §11 Stage 0, refinement plan MS0, knowledge-graph plan K0, risk triage row 6.
- **NEW 2: Complete native cost accounting.** Zone: runtime. Why: native helper starts count as unknown (`projectrun/cost.go`), so PLAY-06, PLAY-07 and OQ-003 get only a lower bound. RUN-02 could absorb it. Inputs: independent assessment U10 and rec 9, refinement plan B3.2, the lessons survey's accounting section.
- **NEW 3: Race detector in the Linux gate.** Zone: ci. Why: `go test -race` has never run, and Managers run concurrently. It could join CI-06. Input: the lessons survey's runtime section.

## Inputs not covered here

- The historian notes on `archive/codex/markitect-historian-working-history`, which the backlog also lists for IDEA-01, were not checked. LEARN-01 covers them.
- Documents only cited by these inputs were not checked: the Government evaluation and v1 contract (GOV-01 inputs), the `codex/knowledge-graph` code (KG-02 input) and the R02 implementation plan on the same archive branch (KG-01 input).
- The literature in §4 of the independent assessment was not re-verified.
