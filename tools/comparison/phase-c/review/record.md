# Phase C review record

Session V, 25 October 2026. Independent review of the four Phase C materials before any Phase C run: the neutral requirements (R), the Markitect model with its rendered spec and mapping (M), the hidden oracle (O) and the hidden reference implementations with mutants (I). V is the only session that saw all four. The procedure was fixed in advance in [checklists.md](checklists.md).

**Status: final, 25 October 2026.** All four materials are ready at the heads named in section 4.

## Reviewed heads

| Material | Pull request | Reviewed head | Base |
| --- | --- | --- | --- |
| R | [#702](https://github.com/Glacius-Labs/Markitect/pull/702), merged | Main `795b8a2af0b51a700ca7a7936852813a7b82770f` | – |
| M | [#707](https://github.com/Glacius-Labs/Markitect/pull/707) | `599972d82e11ad0afa6a3d17ec81f6885942da4f`, then `180bf94cd367a6841330e9032e74ceb9ad604711` (adds the execution configuration; no snapshot or spec byte changes) | contains R |
| I | [#703](https://github.com/Glacius-Labs/Markitect/pull/703) | `12fb6b4ce43df9550c41785a64d4b6a0991e2721` | contains R |
| O | [#708](https://github.com/Glacius-Labs/Markitect/pull/708) | `9685ba4d4da07d203a79c8db0e5540033c4d7752`, then `feda573ba79f62a92287c86f6054ec4940163a28` (V's findings applied) | R's merge |

**Isolation by history.** M's commits change only `tools/comparison/phase-c/{model,spec,mapping}` and the shared renderer (`tools/comparison/requirements.py`, its test and the README row). I's commits change only `tools/makerspace-reference/`. I's tree names no model, spec, mapping or oracle path; it carries O's family names only, as its mutant folders.

## 1. Requirements and model

**Result: equivalent at every step W0–C9. No omission, addition, over-specification or distortion with behavioral effect.**

Method. The renderer re-renders each committed spec byte-identically and finds every Definition body verbatim (`render.py check-all`: 75 to 79 bodies per step). Every Definition's `spec` is empty and the model holds no other content field, so each rendered spec is the model's entire product content; the Package tree, Kinds and the imported vocabulary carry no Makerspace obligation. Each neutral text was then compared with its rendered spec sentence by sentence, aligned by script, and every unmatched sentence was judged by hand. Each delta was checked as the sentence diff and the model-file diff between consecutive steps against its change document.

| Step | Model change against the change document | Result |
| --- | --- | --- |
| W0 | All 37 operations, records, codes and policies verbatim. The operation index is split into the permission matrix and the audit target list; all 37 rows match R in every cell. Section references became links. | equivalent |
| C1 | Refusal entries for both kinds of `forbidden`, the two new fields, entry contents, the record, the §3.1 and §4.2 rewordings, the not-recorded list and the example. | equivalent |
| C2 | Lockout record and identifiers, `lockMachine` with its refusals, effect, result, matrix row and audit target, the rule after rule 5, `machine_locked` between `machine_in_maintenance` and `machine_booked`, schedule `lockouts`, utilization and revenue changes, credit balance, list orders, safety and overview. | equivalent |
| C3 | Bulk tier table and its rules, `discountPercent` from the table, examples (arithmetic checked). Only `materials` files change, so the census areas are untouched. | equivalent |
| C4 | Steward role in the enumerations, registry, `registerUser`, audit and incidents; the Steward column is `all` for exactly the nine listed operations and `none` elsewhere, the other columns unchanged. | equivalent |
| C5 | Blocked member (boundary and unblocking), the rule after rule 3, `account_overdue` after `member_suspended` in `createReservation` and `purchaseMaterial`, `blocked` in the account record, the code, the overdue sentence, time-dependent states, overview and example (arithmetic checked). | equivalent |
| C6 | Validity without expiry, certification record without `expiresOn` and `expired`, reissue, revocation, `not_certified` meaning, time-dependent states, listing, overview and the delivery sentence; no stale expiry statement remains. | equivalent |
| C7 | `billing/payments` folds into `billing/invoices` with unchanged local ids; the billing text takes R's restatement (invoice-centered introduction, payment and credit paragraphs, `recordPayment` without the moved sentence). No rule, value, field, code, order or permission changes; the matrix keeps the area "Billing". | behavior-neutral |
| C8 | Granularity 15 minutes, `invalid_slot` bounds, the new opening hours (minute counts checked), no `null` opening hours, overview, the scope sentence for all dates and the replaced examples (weekdays checked). | equivalent |
| C9 | Snapshot, digest and rendered spec byte-identical to C8. | equal |

Further mechanical checks: each snapshot's `files.sha256` and the C9 = C8 identity are covered by the 40 passing unit tests of `tools/comparison`; the shared renderer still reproduces the Library documents m1–m5 byte-identically (Phase A and B hashes unchanged). M's README records `markitect check` passing for all ten snapshots at Main `528ef346`; V did not repeat it.

Notes (no change needed):

| Id | Note | Reason it is not a finding |
| --- | --- | --- |
| V1-N1 | Every operation body opens with "Operation `x`. Request processing, results and delivery apply." | Restates W0 §3.1, §2.7 and §15, which already apply to every operation; it reaches both arms through the rendering. |
| V1-N2 | The W0 framing paragraph, change-relative wording ("Until now …") and "What stays as it is" statements are not modeled; rule-bearing sentences from those sections are carried. | Change control, not product behavior. |
| V1-N3 | At C7 the model keeps three C5 clarifications that R's restated section 11 omits (partial payments and credit do not end the block; the exact 30-day boundary). | R's C7 says that where the restatement seems to differ, the earlier behavior is meant. |
| V1-N4 | Some rules appear twice, as in R (lockouts never interrupt a session; credit balance in the money policy and in the payment Definition). | Same wording, no conflict. |
| V1-N5 | R keeps the W0 date-arithmetic example ("2027-03-01 plus 365 days") after C6 removed its only use. | R's own text; a definition without a user changes no behavior. |

M's `mapping/` was not used as evidence; the comparison above is independent of it.

### Arm M's execution configuration

`execution/w0.yaml` (W0–C6) and `execution/c7.yaml` (C7–C9) hold only ownership and checks, in the Library fixture's form. **Result: no product content beyond the model.** Ownership gives Root `./`, each Package `P` `realization/P/` and `workshop` also `realization/cmd/makerspace/`; the unit test confirms that the owners are exactly each step's Packages. The two checks run `go build ./cmd/makerspace` and `go test ./...` from `realization/`, both stated in R (§2.1, §15), on the example `go` check profile, which has Phase A's `project-tests` limits (1 CPU, 2 GiB, 64 processes). The tests pass at `180bf94c`.

| Id | Note | For whom |
| --- | --- | --- |
| V1-N6 | Two differences from the Library fixture: a second check `project-build`, and a `project-tests` timeout of 5 minutes instead of 2. Both are part of arm M's treatment (a deterministic gate before publication that B does not have). | Protocol addendum C: list them as intended differences, and run the assessment's secondary own-check under the same 5-minute limit for both arms. |
| V1-N7 | C7 needs the owner to replace `execution.yaml` together with the model, because the W0 configuration names the removed Package; ownership is owner input (coordinator's note). | Protocol addendum C: count it as owner attention on arm M's side. |
| V1-N8 | After C7, files under `realization/billing/payments/` fall to the empty `billing` Package by prefix, and their registry links to the moved Definitions become stale. Arm M may move code or relink the registry to follow the fold, where arm B has no such cause. | Protocol addendum C: predeclare whether changes that only follow the Package fold (moves, registry relinks) count as justified at C7, so that C7 churn is judged by a rule fixed in advance. Registry files are already outside the comparable census. |

## 2. Oracle traceability and coverage

O's pull request [#708](https://github.com/Glacius-Labs/Markitect/pull/708) at `9685ba4d4da07d203a79c8db0e5540033c4d7752` (on R's merge). It changes only `tools/makerspace-oracle/` and names no model, spec, mapping or reference path. Its 28 self-tests pass.

**Facts.** The oracle's operation table (`facts.py`) matches R's operation index in every cell and every operation's argument list in order (37 of 37); fees, discounts, the bulk table, opening hours, granularity and record shapes per step match R and the changes.

**Citations.** 2,724 citations over all steps. Each fragment is verbatim in R (O's self-test). V checked the same fragments against the rendered spec of each step: 46 distinct fragments are not verbatim there. They are the change documents' own phrasing ("as before", "Section 3.1 … now reads"), cross-reference wording ("(section 4.2)" against a link), and the split operation index. Each of these sentences was matched by meaning in section 1, so every cited obligation is in both representations. Three w0 fragments stay cited after a change replaced them (§6.1 "Registers a staff member or an admin." from C4, §10's refusal order from C5, a shorter §4.2 fragment from C1), and the §4.4 fragment about closing time is dropped whole at C8 although its first sentence still holds. These are citation bookkeeping only (note V2-N1, sent to O).

**Expectations against the text.** V read the case builders of all 30 probe families, the behavior cases D-C1 to D-C8 and the ten invariants, and recomputed the arithmetic of the utilization, lockout, billing, block, certification and rounding cases at both granularities. Results are compared exactly only where R fixes them (the exact member sets of the requirements' "Transport" rule, the list orders of §4.5); `reason` is checked only as nonempty; refusals are single-fault or follow R's stated order. No expectation goes beyond the text. The census (`census.py`) stays outside the families and does not change pass or fail.

**Coverage.** Every operation family has authorization (all, own, none and the steward column), audit (accepted, silent, forbidden), money and time cases where R gives the operation such a rule; argument validation runs over every argument of every operation. The calibration exposed one gap, V2-F1 below.

**Re-review at `feda573b`.** O's one fix commit changes only `tools/makerspace-oracle/`; its 30 self-tests pass. The matcher's new tokens keep checks exact: `$ref` and `$calc` compare the resolved value exactly, a missing reference is an error, never a pass, and `$filter` matches the selected items completely and in order. `$capture` records a value without checking it, so that a family can state relations without depending on another family's rule; those values stay checked absolutely in their own family. The new P-NO-MUTATION case re-registers member `ada` with another name and tier inside a no-change snapshot. Probes now capture generated identifiers (only P-IDENTIFIERS predicts them), and money probes compare only their own lines and fields; their relations, such as applied, credited and paid amounts and credit applied up to the total, follow from w0 §11. Citations re-checked: 2,713, of which 44 distinct fragments are not verbatim in the spec, all change-document phrasing or cross-reference wording; the stale ones are gone.

### O's ambiguity list

| O# | Point | Decision |
| --- | --- | --- |
| 1 | `actor` with a string that is not an identifier | Untested, agreed. V reads `invalid_argument` (I's reading too), but no case needs it. |
| 2 | Malformed `at` → `invalid_argument` | **Confirmed.** `at` is an argument (§2.2), §4.4 gives the formats, and "any valid `at`" presupposes validity. |
| 3 | Accepted reads advance the request-time order | Text clear (§4.4 "the most recent request answered `ok`"). Tested, correct. |
| 4 | Census compares `reason` only at C9 | Fine: the census is reported outside the families. Protocol addendum C must say how it is used at C9, where a repair of a non-conforming C8 start remains allowed. |
| 5 | Unterminated last line, exact 64 KiB, `White_Space` version changes, duplicate names, invalid UTF-8 | Untested, agreed; R states the first two and excludes the last two. Known limits. |

### I's ambiguity list

`tools/makerspace-reference/ambiguities.md`, judged against R:

| I# | Reading | Decision |
| --- | --- | --- |
| 1 | Malformed `actor` → `invalid_argument` | Text clear: `actor` is an identifier argument and check 1 precedes check 2. |
| 2 | Empty or whitespace-only line → `invalidRequest`, `id` null | Text clear (§2.4: empty or not valid JSON). |
| 3 | `generateInvoices` for `2099-12` always `month_not_ended` | Text clear: the month ends after the last valid timestamp. |
| 4 | Computed dates after 2099 (`dueAt`, `expiresOn`) printed as such | Open: §4.4 applies the §2.5 formats, whose ranges end in 2099. **Untested**: no case may expect a result date after 2099. |
| 5 | Unbilled charges of a skipped month stay unbilled | Text clear (§11.1 step 3, months in any order). |
| 6 | A checked-in reservation stays active after its end, never `missed` | Text clear (§9.1: only stored `reserved` is reported `missed`). |
| 7 | Range "longer than 366 days" means more than 366 days | Text clear: 366 days admitted, 367 refused. |
| 8 | `paymentsCents` by the month of the payment's `at` | Text clear (§13). |
| 9 | Every member argument must name a `member` user, else `not_found` | Text clear (§3.1 check 5). |
| 10 | Lockout cancels by reported status at the request time | Text clear (C2 effect 2). |
| 11 | `forbidden` recorded only after checks 1–3 | Text clear (C1, what is not recorded). |
| 12 | Member's `getMember` for an unknown member → `not_found`, not audited | Text clear (check 5 precedes check 6; only `forbidden` is recorded). |
| 13 | Lone-surrogate escapes and duplicate member names | Open and declared untested by R (§2.4). **Untested.** |
| 14 | Invoice settled by credit at generation is `paid` and never blocks | Text clear (§11.1 step 7, C5 condition `open`). |

## 3. Calibration

### Dispatch (recorded before the first engine call)

Bounded dispatch under the standing evidence authorization, recorded 25 October 2026 before any engine use.

| Item | Value |
| --- | --- |
| Source | Oracle: `tools/makerspace-oracle` at O's head `9685ba4d4da07d203a79c8db0e5540033c4d7752`, run by its own entry point `harness.py`, with the reused Library transport `tools/library-oracle` (tree `9ec6409452e632819d0d234094da5656430c3041`, identical on Main). References: `tools/makerspace-reference` at I's head `12fb6b4ce43df9550c41785a64d4b6a0991e2721`; each mutant is its step tree with `mutant.patch` applied by I's `mutants/apply.sh` on the host. Runner: `calibrate-pass-1.py` (SHA-256 `cfc3110810d32eb06af5520d9997340ea66ac66efaf0ebd64050d185ed075269`), frozen outside the repository. |
| Engine and images | Docker Desktop 29.9.2 on this machine. Build: `golang:1.27.1` (`sha256:162be5298a40…`). Oracle: `python:3.12-slim@sha256:a6e34c598f24…`, the Library oracle's pinned image. |
| Runs (47) | 10 positive: each reference step W0–C9 against its step's cases, with the change census for C3 (start C2), C7 (start C6) and C9 (start C8). 30 negative: each mutant against the cases of its own step. 7 delta discrimination: reference step n−1 against the cases of step n for C1–C6 and C8 (C7 and C9 are byte-identical trees and covered by their positive runs). |
| Expected | Positive: every case passes and the census is empty. Mutant: of the 30 probe families exactly its own fails; any failed behavior or invariant family is recorded as overlap. Delta: at least one family fails; the failed families are recorded. |
| Limits | One attempt per run; the whole pass at most 2 h from the lock. Builds: one container, 1,200 s. Oracle: 1,500 s per container, four at a time, with the oracle's own limits (2 s per request, 120 s per process, 128 KiB per request, 8 MiB output). Every container: no network, uid 1000, no capabilities, `no-new-privileges`, read-only root, 512 MiB `/tmp`, 256 processes, 2 GiB, inputs mounted read-only. |
| Abort | Credential-shaped text in any output (no login is mounted); a container left after its run; a failed census query; the 2 h limit. An abort stops scheduling and is recorded; nothing is retried. |
| Cleanup | Every container (`--rm`) and both volumes carry the label `mk-vcal=pass-1`. The census of labeled containers and volumes must be empty before the pass and after it. `NATIVE-RUN.lock` is taken exclusively (the runner refuses if it is held) and removed only after a clean final census. |
| Disagreements | Each is decided against R's text: the case or the reference is wrong. The author receives only the sentence and the expected behavior, never the other side's material. A corrected material gets one more run of the affected runs, recorded as pass 2 beside pass 1. |
| Evidence | `C:\Users\Consiliari\Documents\Codex\Markitect-Phase-C-Review\calibration-pass-1\`: one oracle report and log per run, `summary.json` with builds, program hashes, results and both censuses. |

Pre-check without an engine: every distinct reference tree (W0–C6, C8) builds and passes its own tests with Go 1.24.1 on Windows; `c7/` is byte-identical to `c6/` and `c9/` to `c8/`.

### Pass 1 (25 October 2026, 10:22:32–10:23:46 UTC)

Lock taken 10:22:32 UTC. All 40 programs built (19 s), 47 runs executed, no abort, no credential-shaped text, no container left after any run. Labeled containers and volumes: none before, none after. The runner had a defect: it compared the final census as a tuple with a list, so it kept the lock although the census was clean. V checked the lock's nonce and an empty census by hand and removed the lock at 10:24:38 UTC. Total engine time 73 s.

| Runs | Expected | Result |
| --- | --- | --- |
| 10 positive (W0–C9) | every case of the step passes; census empty at C3, C7, C9 | **10 of 10.** 49 to 58 cases per step, all passed; all three censuses empty. |
| 7 delta (n−1 against n) | at least one family fails | **7 of 7.** C1: 11 families (audit fields and refusal entries). C2: 17 (an unknown `lockMachine` fails every family that touches it). C3: 5 (bulk tiers). C4: 8 (steward). C5: 7 (block). C6: 10 (expiry). C8: 7 (hours and granularity). |
| 30 mutants | of the probe families, exactly its own fails | **17 of 30.** 29 mutants fail their own family. P-NO-MUTATION is not caught at all (49 of 49 cases pass). 12 also fail other probe families. |

Disagreements between the oracle and the references: none. Every expectation that failed against a mutant is a correct consequence of R; no case had to be corrected against the text.

**Uncaught mutant (finding V2-F1, oracle coverage).** The mutant changes an existing user's name when `registerMember` is refused with `already_exists`, against w0 §3.1 ("no record is created or changed") and §6.1 ("their identifier, name and role never change"). P-NO-MUTATION re-registers the staff user's identifier with its own name, which no operation can observe. Sent to O as the sentence and the expected behavior.

**Cross-family failures (finding V2-F2, family independence).** Twelve mutants fail other probe families as well. By cause:

| Kind | Families affected | Decision |
| --- | --- | --- |
| Predicted generated identifiers | P-TIME-HOURS, -GRANULARITY, -INTERVALS, -CERTIFICATION, P-ELIGIBILITY, P-RULE-ORDER fail under an identifier fault | Avoidable: capture identifiers outside P-IDENTIFIERS. Sent to O. |
| Whole-invoice and total comparisons | P-MONEY-ROUNDING, -USAGE, -COURSE, -MATERIALS, -CREDIT, -REVENUE, P-TIME-BILLING fail under membership-line or course-charge faults | Avoidable: compare the family's own lines and fields. Sent to O. |
| Unneeded assertions or setups | P-MONEY-ROUNDING asserts stock; P-TIME-ORDER uses a member read; P-IDENTIFIERS uses an ownership refusal; P-CHECK-ORDER advances time with a read; touching slots in P-TIME-BOOKING-STATES and P-REPORT-UTILIZATION setups | Avoidable. Sent to O. |
| Rule observed through another rule | six pairs, listed below | Inherent. Accepted by the coordinator as exceptions (25 October 2026). |

**Declared dependences (coordinator's decision).** For these six mutants, calibration passes when the mutant fails its own probe family plus exactly the declared dependent and no other probe family. Every other mutant must fail exactly its own family.

| Mutant family (rule broken) | Declared dependent | Reason |
| --- | --- | --- |
| P-MONEY-USAGE (tier usage discount) | P-MONEY-ROUNDING | Before C3 the only discounts are tier usage discounts (w0 §6.2), and half-up rounding (w0 §4.3) needs a discount that leaves a fraction of a cent. Rounding can only be observed through a discounted usage line. |
| P-MONEY-LOCKOUT (1000-cent lockout credit) | P-MONEY-REVENUE | C2 makes lockout credits part of `creditsIssuedCents`; checking that inclusion needs the credit's amount. |
| P-AUTHZ-OWN (ownership by `memberId`) | P-AUDIT-FORBIDDEN | C1 records both kinds of `forbidden`, including the ownership violation (check 6); recording one needs ownership to be enforced. |
| P-CHECK-ORDER (check 3 before check 4) | P-AUDIT-SILENT | C1's own example: a member request with an earlier `at` is refused `time_regression` and not recorded, although the member also lacks permission. It only holds with checks 3 and 4 in their order. |
| P-ELIGIBILITY (suspension blocks enrollment) | P-RULE-ORDER | `enrollInCourse` refuses `member_suspended` before `already_enrolled` (w0 §7.1); the order can only be observed when suspension refuses. |
| P-TIME-HOURS (opening hours) | P-REPORT-UTILIZATION | `openMinutes` are the minutes inside opening hours (w0 §13). |

### Pass 2 dispatch (recorded before its first engine call)

After O's fixes, the whole pass is repeated once as pass 2; pass 1 stays recorded unchanged. Same dispatch as pass 1 except: oracle at O's head `feda573ba79f62a92287c86f6054ec4940163a28` (the Library transport unchanged, tree `9ec64094…`); references and mutants unchanged (I's head `12fb6b4c`, the same patched trees); runner `calibrate-pass-2.py` (SHA-256 `55ebd0ce5bb051e4e172f4c783f617db77bffabb16edde566645a9fadbd8c2a4`), which differs from pass 1 only by the declared dependences above and the corrected census comparison; label `mk-vcal=pass-2`; evidence in `calibration-pass-2\`. Expected: as pass 1, with the six mutants of the declared dependences passing when they fail exactly their own family and their declared dependent.

### Pass 2 (25 October 2026, 11:02:40–11:03:54 UTC)

The waiter started the runner only after step 4 had released `NATIVE-RUN.lock`; the runner took it at 11:02:40 UTC and released it itself after a clean census. All 40 programs built (18 s), 47 runs, no abort, no credential-shaped text, no residue; labeled containers and volumes none before and none after. Engine time 73.5 s.

| Runs | Result |
| --- | --- |
| 10 positive | **10 of 10**: every case passes at every step (49 to 58 cases); censuses at C3, C7 and C9 empty. |
| 7 delta | **7 of 7** discriminate (C1 11 families, C2 17, C3 5, C4 18, C5 7, C6 10, C8 7). |
| 30 mutants | **30 of 30**: 24 fail exactly their own probe family; the six with declared dependences fail exactly their own family and the declared dependent. P-NO-MUTATION is now caught. |

Behavior and invariant families still fail alongside some mutants (for example B-BILLING with the money faults); they are the secondary measure.

**Limit found in pass 2 (note V2-N2, invariant families).** Probe and scenario requests now refer to captured identifiers. When a scenario step that creates a record is refused, a later request cannot resolve its reference; the scenario then stops, and all ten invariant families fail together with INV-SCENARIO (seen in delta-C4, where a C3 program cannot register the steward). Pass 1 judged the invariants even after status mismatches. This touches only the secondary measure and treats both arms alike. Optional fix for O: in the scenario, skip or neutralize a request whose reference is missing instead of stopping, so each invariant judges the transcript it has. Until then, read failing invariant families together with INV-SCENARIO as one signal.

The primary "missed obligations" measure of protocol addendum C is the number of failed probe families, per step and summed per chain; behavior and invariant families are secondary (coordinator's decision, passed to the comparison session).

In both passes, 17 of the 30 mutants also fail a strict behavior family (B-*, D-*) or an invariant family. That follows from the design: behavior cases compare complete results and the whole audit log of their area. It matters for the metric, not for validity. Counted together with the probe families, one fault would weigh several times as much as another, which is why the primary measure counts probe families only.

## 4. Verdicts

| Material | Head | Verdict | Open limits |
| --- | --- | --- | --- |
| R, neutral requirements | Main `795b8a2a` (#702) | **Ready** | The W0 date-arithmetic example stays after C6 removes its use (V1-N5, no behavior). |
| M, model, snapshots, spec, mapping, execution configuration | `180bf94cd367a6841330e9032e74ceb9ad604711` (#707) | **Ready** | Protocol items V1-N6 to V1-N8 belong to addendum C. The mapping was not used as evidence. |
| I, reference implementations and mutants | `12fb6b4ce43df9550c41785a64d4b6a0991e2721` (#703) | **Ready** | Every step passes every case of its step at O's final head; every mutant meets its calibration criterion. |
| O, hidden oracle | `feda573ba79f62a92287c86f6054ec4940163a28` (#708) | **Ready** | V2-N2 (invariant families fail together when the scenario cannot resolve an identifier; secondary measure only). Untested by design: O#1, O#5, I#4, I#13. |

Findings and their resolution:

| Id | Material | Class | Resolution |
| --- | --- | --- | --- |
| V2-F1 | O | coverage gap: P-NO-MUTATION did not observe a refused re-registration | Fixed at `feda573b`; caught in pass 2. |
| V2-F2 | O | family independence: avoidable cross-family dependences (identifiers, whole invoices, setups) | Fixed at `feda573b`; six inherent dependences declared with the coordinator's approval; pass 2 meets the criterion. |
| V2-N1 | O | stale or over-voided citations | Fixed at `feda573b`. |
| V2-N2 | O | invariant families stop together on an unresolved reference | Recorded limit; optional fix. |
| V1-N1 to V1-N5 | M, R | notes without behavioral effect | No change. |
| V1-N6 to V1-N8 | protocol | arm M's extra checks and timeout, C7 owner effort, C7 churn rule | Passed to the coordinator for addendum C. |

No finding required a change of the neutral text, so nothing went back to R.

Calibration evidence, outside the repository: `C:\Users\Consiliari\Documents\Codex\Markitect-Phase-C-Review\` (`calibrate-pass-1.py`, `calibrate-pass-2.py`, `calibration-pass-1\`, `calibration-pass-2\`, each with one report and log per run and `summary.json`).
