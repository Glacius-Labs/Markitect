# Simplified Apply workflow comparison — 20 October 2026

**Status: experimental comparison candidate, not for integration.** The owner commissioned this separate session to implement and evaluate an Apply without the global Plan and Prepare Passes, next to the running [hardening stream](apply-hardening-20261020.md). It changes no accepted concept: the [accepted workflow](../design/canonical-model/manager-apply-workflow.md) keeps Plan, Prepare and Implement until the owner decides. Branch `dev/apply-simplified-passes`. Implementation, source checks, isolated Verify, complete Apply and owner acceptance are reported separately. No owner acceptance is recorded.

## Hypothesis

Apply runs Model check and File check, then **Implement**, then an independent **Verify**. Findings start a new Implement and a fresh Verify. A clean Verify leads to guarded publication with exact read-back. The hypothesis: this is at least as reliable and complete as Plan → Prepare → Implement → Verify on the same frozen inputs.

Planning and preparation do not disappear: each Manager plans and prepares its own part inside Implement, through the existing base process:

1. settle its own approach, obligations and dependencies;
2. produce the basis its dependent children need;
3. have that basis, including files and registry, checked technically and accepted semantically by its parent;
4. only then release its children;
5. stay available for questions, review and coordinate its direct children's contributions;
6. integrate accepted results and submit its complete responsibility for acceptance.

No new mandatory sub-process replaces the removed Passes under another name. Parents do not plan or implement for their children, and every Manager writes only in its own area.

### Design account (working method)

1. **Observation.** The [first Pass-process report](apply-pass-process-20261020.md) records substantial coordination overhead: each Pass costs every Manager a basis and a result plus parent decisions, three times before Verify. Haiku also exhausted its budget on serial dependencies. Plan's read-only result and Prepare's scaffolding are both re-examined in Implement anyway.
2. **Challenged assumption.** That a whole-tree planning and preparation stage is needed before realization, so that dependencies and foundations are agreed globally before anyone builds.
3. **What changes and what stays.** Only the Apply composition and the Implement mandate change; the Pass engine, its gates, receipts, findings, dispositions, ownership, limits and publication stay. The Implement basis now carries a Manager's approach together with the foundation its children build on, so the parent still reviews an approach before its children start. Dependencies that were declared in plans are now declared by the waits Managers report while they work. A waited-for accepted basis or result is bound to the waiter's work at its accepted version. Its revision or reopening explicitly resets that work, which restarts only on the newly accepted version. Unchanged valid bases stay.
4. **Observable improvement.** On the same frozen cases and model: no additional missed defect, no lost preservation, no new ownership conflict or omission, and question, contradiction and dependency handling at least as correct as the reference. Only then are fewer Passes, Turns or less time a gain. Fewer Passes alone or one green run prove nothing.

## Candidate

| | Reference (hardening) | Comparison |
| --- | --- | --- |
| Product SHA | `8b5040ff294018dbd9d3c9d3c52742013106a980` (reviewed, gated) | CANDIDATE_SHA on `dev/apply-simplified-passes`, parent `8b5040ff` |
| Work Passes | Plan (read-only, plan payload), Prepare, Implement | Implement |
| Correction purposes | plan, prepare, implement | implement |
| Dependencies | declared in accepted plans, start conditions of later Passes; ad-hoc waits | declared by waits, bound to the accepted version |

Changed source, besides the composition (`internal/apply/apply.go`):

- **Implement mandate.** It now holds the duties of the former Plan and Prepare mandates that concern a Manager's own part: claims, input classes and public-boundary evidence; interfaces, needs and outputs; how a parent coordinates its children; and a basis that holds the approach and the foundation children build on. It also says how to declare a dependency. The Verify mandate and guidance are unchanged.
- **Bound dependencies (`internal/run/bindings.go`, opt-in `PassSpec.BindWaits`, set only on Apply's Implement).** Without plans, nothing would invalidate work built on another Manager's revised basis through a wait. This change keeps the invalidation guarantee; it is not a separate improvement.
  - The accepted target of a wait, or one already accepted when the wait is reported, is bound to the event the waiter works toward: its basis or its result. Before, an already accepted target was a refusal; now it is a turn report kind `bind`. It is recorded as `dependency.bound` with proposal and stand.
  - Only Managers outside the waiter's own line bind. Ancestors and descendants are already related by the parent-first and integration order.
  - A binding that would close a cycle among all start, basis and result events, whether or not they already happened, is refused or skipped (`dependency.unbound`).
  - A revision or reopening of a bound submission resets a waiter whose basis built on it, like a declared consumer; its restart waits for the new version and records it as rebound. A waiter whose result alone built on it keeps its basis and children and integrates again: its submitted result is invalidated, or its accepted one reopened with that result's consumers.
  - Verify and Extract bind nothing, so an isolated Verify runs exactly the reference's Verify code path.
- **Self-model and guides.** `product/run/confirmed-run-history`, its generated view, the Apply registry entries, the run/apply guides, the runtime contract and the CLI help line describe the candidate. The workflow, architecture, contributing and project-workflow pages mark it as an unadopted experiment.

The independent complete-diff review of the first draft found one blocking defect. The bindings were not cycle-checked across already completed events, and every binding acted as a start condition. As a result, a parent bound to its own child's result, or two siblings bound to each other's bases, would deadlock on the next reset. Its re-review found that a binding serving only the result still reset the waiter's unchanged basis and subtree. The design above corrects both, and regression tests cover these scenarios plus the cycle check. The review also flagged two mandate sentences as going beyond a minimal change. They are kept because they replace the parent's plan review: "submit your basis once it serves your children instead of holding it back", and "a leaf's basis is its approach and what it has realized so far". Without them a Manager could offer its first basis only after complete realization, and no parent would see an approach before the children build.

The engine's `Plans` capability and the `plan`/`prepare` purposes stay as generic engine features with their tests. Removing them belongs to an adoption, not to this comparison.

**Comparability.** Same kit, packets, oracles, owner answers, profiles, images, model and limits as the reference. The deliberate differences are the composition, the Implement mandate and the Implement-only wait binding. All three follow from the workflow change; no other product change was made. V1 and V2 therefore exercise unchanged Verify code: they replicate the reference on a different host binary rather than test the simplification, so their main value is the same-model variance of the reference cases.

## Frozen comparison plan

Frozen on 20 October 2026, 17:45Z, before any provider work of this session. It was agreed with the hardening session, which ran the reference on the same inputs. Evaluation directory: `/home/user/mk-simplified-eval/` in WSL Ubuntu, container label prefix `cmp-`.

| Binding | Value |
| --- | --- |
| Candidate | `e814c373251b9198e20b2edc87e9255f40c3c1f8`; CLI SHA-256 `2229cb89…44717`, Verify harness `3a32d3e5…3f5` (built from the candidate) |
| Reference | `8b5040ff294018dbd9d3c9d3c52742013106a980`, the hardening session's reviewed and gated evaluation candidate |
| Images | the reference build's own: runtime `sha256:3d773344…de394`, agent `sha256:fc865635…7aa`, Python example `sha256:81a0f8bc…32f`. Claude Code 2.1.301, agentd `development`. The candidate changes no image content. Its own rebuild had different IDs (npm layer not cached) and is kept under tags `-own`. |
| Model | `claude-sonnet-5-5`, effort high, every Execute and Verify role, the existing dedicated runtime token |
| Inputs | the reference's frozen kit copied verbatim: kit digest `973a1413…4d3c`, manifest `208e907b…fa4d`, packets V1 `49c7a616`, V2 `f8173a9c`, A1 `8660dc87`, A2 `876a9940`, A3 `551e34e4`, A4 (S-QV) `59280816`, A7 `95d55288`; scripted A4 owner answer `85b73b90…4c0` with neutral answers to every other question; the A5 model change "at most the maximum name length" → "at most 40 characters" |
| Oracles | `check_public_names.py` `14150f13…f03b` (8 cases), `probe_name_max.py` `6821248c…5baa`, the kit's `oracle.py`, `assess.py`, the project's own tests |
| Case limits | taken over unchanged from the reference (owner instruction of 20 October): Verify 60 min, 120 Turns, 1 Pass; Apply 120 min, 400 Turns, 12 Passes; S-X 30 min; every Turn at most 20 min |
| Series limits | at most 9 provider-backed cases and 4 h summed charged time, first Agent dispatch to outcome. Repetitions and any Haiku case count. |
| Harness | the reference scripts with only paths, label prefix and one addition changed. The addition: the series remainder can shorten, never lengthen, a case's deadline, and such a cut is recorded. |

**Order and rules.** V1, V2, A1, A2, A4, A7, A3, then A5 from this session's own A4 publication (never from the reference's). A6 is not planned; the reference displaced it too.

- A case starts only while at least 30 minutes of the series remain. A cut by the series remainder is recorded as series exhaustion, not as a workflow result.
- Cases run one at a time and never while a reference case runs on the shared Docker engine.
- No repetition until green. A failed case keeps its result, and a documented infrastructure repeat counts as a case.
- **Haiku stays unrun.** The reference has no Haiku case on this build, and the owner made Haiku conditional on Sonnet working.

**Amendment before the first provider run (20 October, about 20:10Z).** The reference series ended at that time and showed that its candidate moved during the series:

- V1-b, V2, A1 and A2 ran on `8b5040ff`;
- A2-b ran on `e9643e4e`, a mandate fix after A2 overreached;
- A4, A5 and A7 ran on `922ab2f6`, which also lets the owner's reviewer judge an ancestor's adverse receipt;
- A3 and A6 were not run.

The packet trees of all three freezes are byte-identical, so the inputs are equivalent and only the builds differ. Each comparison therefore uses the reference's own base:

- **C1** = `e814c373` (base `8b5040ff`) runs V1, V2, A1 and A2.
- **C2** = the same simplification applied to `922ab2f601f12703b913663ebfe6d9c2bb01f9ee` runs A4, A7 and A5. C2 carries the reference's mandate fixes into the merged Implement mandate, word for word where the text corresponds, so that the comparison still varies only the workflow. C2 gets its own independent review and gates, built only in gaps between cases.
- **Remaining budget, in this order:** A2 on C2 against A2-b (confounded by `922ab2f6`'s receipt change, named as such), then A3 without a reference.
- **Excluded:** later reference commits such as `fcd0fb54`, which no provider run evaluated.

**Correction before any Implement ran (C1′).** The review of C2 found a gap in C1's merged Implement mandate. It had dropped the Plan mandate's sentence "Existing artifacts and tests that already meet an obligation are its evidence as they are", which the reference's Plan Pass carries on `8b5040ff` as a counterweight to optional changes. C1′ restores that sentence and changes nothing else. A1 and A2 run on C1′. V1 and V2 use only the unchanged Verify mandate, so they are equally valid on C1 and C1′.

**Expected outcomes** (the reference manifest's):

- V1: a genuine, owner-bound finding of the hyphen-name refusal with receipts; no clean Verify.
- V2: clean, no fabricated mandatory work.
- A1: the fault repaired; d1 oracle 21/21 and name checker 8/8; fresh Verify; findings closed.
- A2: no-change publication.
- A3: both faults repaired; s1 21/21.
- A4: the question routed to Root and the user; the scripted answer permits completion; s1 21/21.
- A5: exactly the public change; name maximum N=40 10/10; s1 without its hard-coded name rule 20/20.
- A7: Root reports the model error; the run ends Failed without publication.

## Results

RESULTS

## Assessment

ASSESSMENT
