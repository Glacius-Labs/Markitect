# Concept consistency and consolidation review

Date: 16 October 2026. Assistant assessment requested for architectural consolidation in the [owner discussion](../sources/working-day-discussion-20261016.md#review-the-concept-as-a-whole). These recommendations are review input, not accepted contract changes, backlog assignments or implementation evidence.

The core concept is coherent. The main opportunity is to align the contracts and keep the first implementation bounded, rather than redesign the architecture. Consolidate configuration resolution, budget protection and lifecycle meanings before adding further organizational mechanisms.

## Review basis

The assessment used committed design `c640c99b704b0b9aead8f57cf52bfe48bd7ffc27` on `docs/manager-realization-north-star`, plus two then-untracked Architect drafts. Fetched Main remained `bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310`. This was a document consistency review, not a new code audit or execution test.

| Draft at assessment time | SHA-256 of inspected bytes |
| --- | --- |
| `docs/design/canonical-model/manager-apply-configuration.md` | `5b7cee96042bfa7f1b20646fe26eebd1f73b2f602a9af4f1c1e059c6a106679f` |
| `docs/design/canonical-model/manager-apply-journal.md` | `cd011428a28b0f115179228949fa53b9ae252e3b082027baa2e808cee3bffdf0` |

The drafts are changing concurrently. References to them below identify the reviewed proposals, not frozen current contents. During recording of this review, the port contract was already being updated to use assignment streams and the proposed journal adapter. Finding 6 therefore includes that observed progress.

The [language v1](../../design/canonical-model/minimal-language/language-contract.md) remains the completed language baseline. This review does not reopen its accepted scope. The [Apply concept](../../design/canonical-model/manager-apply-concept.md), [Go architecture](../../design/canonical-model/manager-apply-go-architecture.md), [ports](../../design/canonical-model/manager-apply-port-contracts.md), [implementation plan](../../design/canonical-model/manager-apply-implementation-plan.md) and [working-day reflection](desired-working-day-20261016.md) supply the compared product and interaction contracts.

## Foundations that fit together

- The model owns intended behavior; execution configuration owns runtime choices.
- Mandates are source-bound views rather than separately maintained business rules.
- An assignment persists across Apply runs, each with fixed inputs.
- Work can carry forward without inheriting its earlier approval.
- Managers judge meaning; the Host controls scope, transitions and observed results.
- Status reports derive from the execution history.
- Experience and operational diagnosis remain separate later extensions.

The new drafts largely express the working-day ideas through existing tasks, questions and history. That preserves the organization rather than introducing another management hierarchy.

## 1 Choose one configuration resolution rule

**Observation:** The Apply concept describes a project default and exact Manager overrides. The configuration draft resolves overrides along the root-to-current-Package path and appends instructions. The latter adds ancestor effects: changing lending can change lending/reminders even without an exact override there.

**Recommendation:** Start with project defaults plus exact Manager overrides. Parent-task restrictions still constrain the delegated work, separately from persistent configuration. If ancestor inheritance is deliberately preferred, specify it consistently in the concept and examples rather than leaving both readings active.

**Benefit and tradeoff:** Exact overrides make effective settings easier to explain and test, at the cost of some repeated explicit configuration. This recommendation concerns execution configuration, not Package semantics in the canonical language.

**Consolidation result:** One authoritative resolution rule and a small example showing what a lending override does, or does not do, to reminders.

## 2 Distinguish dispatch reservations from protected completion capacity

**Observation:** The configuration draft bounds runs, session starts and steps. The journal reserves capacity before dispatch. This does not yet express the working-day requirement that implementation cannot consume the capacity reserved for later review, integration and completion.

**Recommendation:** Keep one ledger with allocations carrying a purpose and protection. Ordinary work, protected completion capacity and free capacity suffice as accounting categories. A correction allocation includes its renewed assessment. Do not turn the illustrative 24/18/12/6 split into four independent budget engines or a fixed allocation algorithm.

**Benefit and tradeoff:** This protects the ability to finish responsibly while retaining a small accounting model. The organization must estimate remaining assessment needs and revise reservations with reasons; protected capacity is not a guarantee of success.

**Consolidation result:** Dispatch rules show which allocations implementation may consume, how protected allocations are released or revised, and when an owner amendment is required. Consumed effort persists across successor runs.

## 3 Separate active execution limits from waiting deadlines

**Observation:** The draft's `assignmentTimeout` becomes an absolute deadline and includes time waiting for the owner. The working-day budget proposal separately describes active execution limits and external delivery deadlines. These protect different resources.

**Recommendation:** Name and define active execution allowance separately from an optional assignment deadline. Make it clear which limit prevents continuation, how waiting is treated and how an authorized extension is recorded. Step and check timeouts remain distinct local bounds.

**Benefit and tradeoff:** An overnight owner response does not unexpectedly consume a limit presented as working time. The runtime must account for the chosen time semantics explicitly; there is no need for a general scheduling subsystem.

**Consolidation result:** An example of a task waiting for an owner answer shows its unchanged execution consumption, any real deadline and the permitted next action.

## 4 Give the authorized model edit an explicit delivery boundary

**Observation:** The journal correctly separates answer received, edit authorized, model revision observed and successor started. It also permits an external owner/caller to perform the model edit. That supports a bounded first Apply but does not by itself deliver the desired one-answer-and-continue experience.

**Recommendation:** Distinguish two delivery levels. The first Apply may expose a pending authorized model-edit step to its caller. The desired working-day flow later needs a small operation that applies the exact authorized edit against the expected source and initiates continuation. Assign that transition a clear owner without introducing general Government machinery.

**Benefit and tradeoff:** This preserves artifact-only Apply and avoids overclaiming the first user journey. A narrow model-edit operation still needs source guards, observed outcomes and recovery; those responsibilities cannot disappear behind an acknowledgement.

**Consolidation result:** The documentation states who performs the approved edit in each delivery level and what the owner sees while it is pending or failed.

## 5 Map lifecycle states to outcomes and displayed status

**Observation:** The concept lists outcomes such as No change, Stale and Interrupted or uncertain. The journal proposes a different set of run states. These can coexist, but their mapping is not yet fully visible. A state, an outcome and a user-facing label need not be the same enumeration.

**Recommendation:** Give one transition contract ownership of lifecycle meaning, then derive outcomes, allowed actions and status views from it. Avoid parallel state machines in the CLI, report and scheduler.

**Benefit and tradeoff:** A single mapping prevents misleading summaries without flattening local, aggregate, publication and deployment evidence into one success flag. The mapping must cover failure and waiting cases, not only the successful path.

**Consolidation result:** Explicitly account for verified no-change completion, one waiting task while others work, stale inputs, and unknown execution or publication outcomes. Prior successful publication remains historical success when a successor begins.

## 6 Consolidate document ownership as the design develops

**Observation:** The working-day reflection now contains detailed proposed rules while the Architect transfers them into technical contracts. At assessment time, the ports described run-scoped append batches while the new journal explicitly moved the stream boundary to the assignment. The architecture still illustrated a generic file-store adapter while the journal selected a dedicated SQLite adapter.

**Recording-time update:** A subsequent uncommitted port edit already changes append scope to the assignment and links the proposed journal adapter. This is progress on the original finding, not evidence that every related document is now aligned. Preserve the original review basis rather than claiming the old port wording is still current.

**Recommendation:** Keep goals, examples and rationale in reflections; give each concrete rule one technical owner and link to it elsewhere. Preserve proposal/decision status and historical sources. Update superseded diagrams, port descriptions and open-question lists when a design choice is adopted.

**Benefit and tradeoff:** This reduces contradictory instructions and repeated maintenance. It requires a deliberate consolidation pass; adding more copied rules would increase the problem.

**Consolidation result:** Readers can identify the governing contract for configuration, lifecycle, history and publication without reconciling several prose accounts themselves.

## Recommended sequence

First reconcile configuration resolution, budget protection and lifecycle mappings. Keep the authorized-model-edit boundary visible. Then complete the already planned two real library Applies: M1 to A1, followed by a model-only M2 change and Apply to A2. Inspect actual behavior and evidence using the planned independent expectations.

A focused continuation case with an owner question, authorized model correction and successor run can follow as the next interaction proof. Experience selection, operational doctor, richer teams, automatic review reuse and a full cockpit need not become prerequisites for the first demonstration.

The principal risk is designing every useful organizational feature in depth before observing the core organization work. Existing boundaries leave room for later extensions. The immediate aim is a small set of consistent contracts and real evidence that the delegated method delivers.

Return to [product reflections](README.md).
