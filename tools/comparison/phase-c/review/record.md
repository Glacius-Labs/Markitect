# Phase C review record

Session V, 25 October 2026. Independent review of the four Phase C materials before any Phase C run: the neutral requirements (R), the Markitect model with its rendered spec and mapping (M), the hidden oracle (O) and the hidden reference implementations with mutants (I). V is the only session that saw all four. The procedure was fixed in advance in [checklists.md](checklists.md).

**Status: in progress.** Step 1 is done; steps 2 and 3 wait for O's pull request.

## Reviewed heads

| Material | Pull request | Reviewed head | Base |
| --- | --- | --- | --- |
| R | [#702](https://github.com/Glacius-Labs/Markitect/pull/702), merged | Main `795b8a2af0b51a700ca7a7936852813a7b82770f` | – |
| M | [#707](https://github.com/Glacius-Labs/Markitect/pull/707) | `599972d82e11ad0afa6a3d17ec81f6885942da4f`, then `180bf94cd367a6841330e9032e74ceb9ad604711` (adds the execution configuration; no snapshot or spec byte changes) | contains R |
| I | [#703](https://github.com/Glacius-Labs/Markitect/pull/703) | `12fb6b4ce43df9550c41785a64d4b6a0991e2721` | contains R |
| O | follows | – | – |

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

Pending O's pull request.

### Ambiguity decisions

I's list (`tools/makerspace-reference/ambiguities.md`), judged against R:

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

O's list: pending.

## 3. Calibration

Pending. The dispatch is recorded here before the first engine call.

Pre-check without an engine: every distinct reference tree (W0–C6, C8) builds and passes its own tests with Go 1.24.1 on Windows; `c7/` is byte-identical to `c6/` and `c9/` to `c8/`.

## 4. Verdicts

Pending.
