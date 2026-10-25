# Phase C review: checklists

Working draft of session V (review and calibration), written before any Phase C material existed. It fixes how the four materials are reviewed, so that the review does not adapt to what it finds. The final record (`record.md` in this folder) reports every finding, its resolution, the calibration results and one verdict per material.

Sources of the method: the Phase C design draft of 25 October 2026 (sections 2 and 3; kept in the coordinator's session notes, to be superseded by protocol addendum C), the [measurement guide](../../../../docs/guides/measurement.md), and how Phase B was reviewed: the obligation table and the independent review in [protocol B7](../../../../docs/validation/v1-comparison-protocol-20261024.md#b7-materialization-record-condition-2), the verbatim renderer [`requirements.py`](../../requirements.py) and the cited, self-tested families in [`phase_b_cases.py`](../../phase_b_cases.py).

## Materials and roles

| Material | Author | Location | May see |
| --- | --- | --- | --- |
| R: neutral requirements, W0 and deltas C1–C9 | session R | `tools/comparison/phase-c/requirements/` | the product brief |
| M: canonical model, per-step snapshots, rendered spec, mapping | session M | `tools/comparison/phase-c/model/`, `spec/`, `mapping/` | R |
| O: hidden oracle (cases, probes, invariants, runner, ambiguity list) | session O | `tools/makerspace-oracle/` | R only |
| I: hidden reference per step, one mutant per family, ambiguity list | session I | `tools/makerspace-reference/` | R and O's family list only |
| V: this review | session V | `tools/comparison/phase-c/review/` | everything |

R is the source. Every other material is judged against R's text, never against another derived material.

## Grading rules

- **Classes.** Omission (a neutral obligation is missing), addition (a new obligation without neutral basis), over-specification (a freedom the neutral text leaves open is narrowed), distortion (the obligation is present but changed: condition, quantifier, modal force, boundary, number, unit, actor, timing or scope). For the oracle also: untraceable (no cited sentence or a wrong one), beyond-text (expects more than the cited sentence requires), coverage gap, arithmetic error. For calibration: case wrong or reference wrong.
- **Severity.** *Blocking* when it can change an assessed behavior, lets one representation say more than the other, or makes a hidden expectation untraceable. *Note* when it changes no obligation, such as a faithful clarification or wording without behavioral effect (proportionate grading, owner 22 October 2026).
- **Direction that matters most.** An oracle-tested neutral sentence that is missing or distorted in the model punishes both arms for an obligation neither saw; it is always blocking. A model addition reaches both arms through the rendering and is blocking only when it adds or narrows behavior.
- **Ambiguity.** A sentence that admits two readings with different observable behavior is either decided from the text (one reading is clearly meant), left untested (no case may depend on it; recorded as a known limit, like Phase B's `loanClass: null`), or sent to the coordinator when the study needs the obligation. A text change goes to R; M, O and I then adapt, and the review repeats for the changed sentences.
- **Isolation in messages.** A finding sent to an author carries only the neutral sentence and the expected behavior. O and I never receive model, spec, mapping, reference, mutant or each other's material; M never receives oracle or reference material. Every message is logged in the record.
- **Frozen heads.** Each material is reviewed at a recorded head SHA. A later push is reviewed as a delta against that head, and the verdict names the final head.

## 0. Intake (when the coordinator reports R merged and M, O, I open)

- [ ] Record R's merge commit on Main and the head SHA, base and changed paths of each PR (M, O, I).
- [ ] Isolation by history: each PR changes only its own paths; O's and I's branches contain no M, spec, mapping or each other's commits (`git log --name-only <merge-base>..<head>`); no cross-references by path or content (grep O for `model/`, `spec/`, `mapping/`, `makerspace-reference`; grep I for `model/`, `spec/`, `mapping/`; I may hold O's family names only).
- [ ] Hidden material is unreachable for writers: the spec, model and prompt mention no oracle, probe, case id, mutant or reference; hidden folders are outside anything a writer fixture would contain.
- [ ] Run each material's own checks: R's structure check if any; M: `markitect check` on every materialized snapshot and the renderer's `check` for every step; O: its unit tests and transcript self-tests; I: build and test of every step tree.
- [ ] Atomize R: give every normative sentence a stable id `<step>-<area>-<n>`, kept in the record, so that every later check refers to it.

## 1. Requirements and model, both ways, per step

For W0, then each delta C1–C9, on the materialized snapshot of that step.

**Forward (R to model).**
- [ ] Every atomized R sentence maps to at least one Definition sentence that carries the same obligation. M's `mapping/` is checked, not trusted.
- [ ] Same condition, quantifier (exactly, at least, at most, every, no), modal force (must, may, must not), boundary (inclusive or exclusive, half-open intervals), number and unit (cents, minutes, days, roles), actor and timing.
- [ ] Every value in the text (fees, rates, tiers, granularity, opening hours, 30-day overdue rule, expiry) equals the model's spec or Property value and the rendered table value.
- [ ] The JSON Lines contract (envelope, operations and arguments, records, statuses, error rule, actor and timestamp in every request, integer cents) is complete and identical in meaning.
- [ ] The four cross-cutting policies (authorization matrix, audit, money, time) are stated with the same scope: which operations, which outcomes, which actors.

**Backward (model to R).**
- [ ] Every Definition sentence, spec value and interface entry traces to an R sentence, or is structure only (Package scope that restates, links, model vocabulary) and carries no obligation.
- [ ] No over-specification: no ordering, error wording, identifier format, data structure, file layout, test topic or precedence between failures that R leaves open.
- [ ] Consequences R leaves implicit stay implicit; consequences R states explicitly are stated (Phase B lesson: two sentences that spelled out easy-to-miss consequences were removed).

**Each delta (step n-1 to n).**
- [ ] Diff R's text and the model snapshot between the steps. Every R change has a model change and vice versa; each extra model edit is a necessary follow-on of a listed change, recorded with its reason.
- [ ] Removals (C6 expiry) leave no stale sentence in any Definition: training, bookings, reporting, audit, interface, delivery.
- [ ] C1 audit of refusals reaches every operation as R states it; C4 steward role reaches the whole permission matrix; C8 granularity and opening hours reach every time rule R names.
- [ ] C3: the model change stays inside materials and money as R states; bookings, memberships and training Definitions are byte-identical to C2.
- [ ] C7 is behavior-neutral: after mapping renamed ids and Packages, the multiset of normative sentences and values equals C6; only Package placement, ids, headings and link targets differ. R's C7 text describes the reorganization only.
- [ ] C9 equals C8: R's step text, the model snapshot, the rendered spec and the oracle's case set are identical (byte or hash), and I's c9 is c8.
- [ ] Every snapshot passes `markitect check`.

**Mechanical spec rendering, each step.**
- [ ] The renderer re-renders every committed spec byte-identically and finds every Definition body verbatim.
- [ ] Substitutions: each is listed, with an exact occurrence count enforced by the renderer, and each only replaces a phrase that names the model itself. No other text is added except title, introduction, headings and table frames.
- [ ] No model vocabulary or `[[` link remains; every Markdown link resolves to a heading; property tables come from spec values.
- [ ] The Phase A/B rendering rules are kept (or each deviation is stated and harmless); Library profiles m1–m5 still render byte-identically if the renderer is shared.

## 2. Oracle traceability and coverage

- [ ] Every case, probe and invariant check cites an R sentence id or quotation; the quotation is verbatim in R's text of that step and, by section 1, in the model and the spec.
- [ ] The cited sentence requires the expected behavior, not merely permits it. Nothing beyond the text: no result ordering, error wording, reason codes, identifier format, field set beyond the stated records, precedence between simultaneous failures (cases stay single-fault), rounding or time-zone behavior that R does not state. Matching allows every free choice (captured ids, non-empty reason, sets compared as sets, extra fields where the contract allows them).
- [ ] Arithmetic is right: dates and UTC minutes, half-open intervals, opening-hour edges, 30- and 15-minute grids, line rounding in cents, credits, bulk tiers at and around each threshold, the overdue boundary at 30 and 31 days.
- [ ] Coverage matrix: every operation family against the four policies. Each cell has at least one case or probe, or is marked not applicable with an R reason (for example, no charge on a read-only operation). Authorization: an allowed role succeeds and a disallowed role is refused. Audit: an accepted mutation records actor and time; refusals per step (none before C1, recorded with actor role from C1 on). Money: integer cents and rounding. Time: grid, opening hours and intervals.
- [ ] Cross-area invariants (for example, no charge without an audit entry; a locked-out machine has no future reservation) cite the R sentences they combine and follow from them.
- [ ] Cumulative steps: earlier families are carried; a carried case is transformed only where a delta changes its sentence (as Phase B's M5 transform), and no carried case expects removed or changed behavior.
- [ ] Each delta has at least one case that the previous step's correct behavior fails, except C7 and C9.
- [ ] Families: each case belongs to exactly one family; family names are stable across steps; the family list matches I's mutant list one to one. Families are the unit of the "missed obligations" count, so a family that one fault can fail together with unrelated families is flagged.
- [ ] Self-tests: each family's expected transcript passes and one wrong transcript fails, through the oracle's own transport.
- [ ] Runner: limits finite and stated; evaluator and case bytes hashed on the host and checked in the container; network off; the program cannot reach the report (as the Library oracle's container harness).
- [ ] Ambiguity lists of O and I: each entry is decided (text clear, untested, or to the coordinator), and the decision is recorded with the sentence. Ambiguities found here are added the same way.

## 3. Calibration (bounded dispatch, recorded before running)

Recorded in the record before the first engine call: source (O and I heads, oracle runner path, image Id), the planned runs and expected outcomes, limits, abort, cleanup.

- [ ] **Positive control:** reference step n passes every case of step n (W0 to C9; c9 is c8 and needs no separate run unless its tree differs).
- [ ] **Negative control:** each mutant fails at least one case of its own family and no case of any other family. A crash, a timeout or a broken build is not a family failure.
- [ ] **Delta discrimination (to be declared in the dispatch if the budget allows):** reference step n-1 against the cases of step n fails exactly the delta's families; for C7 the C6 reference passes the C7 cases.
- [ ] **Limits:** one attempt per step and mutant; the whole pass at most 2 h; the oracle's own per-request and per-process limits.
- [ ] **Engine discipline:** take `NATIVE-RUN.lock` in the session notes folder exclusively before the first engine call and refuse if it is held; remove it after the final census. Runs start detached in WSL with done markers.
- [ ] **Abort:** any container or volume residue after a run; any credential-shaped text in an output (no login is mounted; outputs are scanned for token and auth-file patterns); a census query that fails.
- [ ] **Cleanup:** labeled containers and volumes removed; census of containers and volumes with the runner's label before and after, both empty, recorded.
- [ ] **Each disagreement:** decide against R's text whether the case or the reference is wrong; send the author only the sentence and the expected behavior; record both sides and the decision. A failed attempt stays in the record.

## 4. Record and verdict

- [ ] Findings table: id, stage, step, material and location, R sentence, class, severity, decision, routed to, resolution commit, status.
- [ ] Obligation table, as Phase B's: each hidden family, its R sentences, its Definitions and its spec sections.
- [ ] Calibration table: run, input, expected, actual, duration, cleanup census.
- [ ] Verdict per material at a named head: ready or not ready, with open limits stated.
- [ ] The PR adds only this folder, which the registry already covers as part of `tools/comparison/phase-c/`; it is reviewed independently before merge. When all four materials are ready, tell the coordinator.
