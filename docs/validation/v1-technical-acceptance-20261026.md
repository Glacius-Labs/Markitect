# V1 technical acceptance on F_final

Date: 26 October 2026. First of the two final reports the [owner directive](../concepts/sources/v1-acceptance-directive-20261025.md) asks for. It answers whether Markitect works reliably according to the accepted concept, on the jointly checked final source F_final. The product-value question is the [comparison report](v1-comparison-report-20261026.md). Owner acceptance, release and deployment are separate owner decisions.

## Verdict

**Correct and safe as V1 on F_final; not yet reliable at the larger project size.** Every concept guarantee has one clear holder in the host, is exercised by controlled real-Git tests and passed the full hosted CI on exactly the final commit; an independent review found no blocking finding. In every native run, nothing unverified was published, nothing leaked and every stop was honest except one misclassified budget stop. Native runs completed every Library scenario (38 of 38 counted Applies plus P4). On the 21-Manager Makerspace project, only 19 of 32 Applies published: product limits and one product defect stopped the rest. The owner-approved fix plan addresses each of them; a new frozen source F follows only after its native controls.

## Binding

| Item | Value |
| --- | --- |
| F_final | `4d23f7aea0576ff5c477fd301b456a24bbb6dcb3` |
| Runtime image | Go image `sha256:3d75d81f438e71254d01f013cd35e0c61f9eecfb8abba723b34e12158c33e943`, built from F_final |
| Hosted CI | [run 40240295977](https://github.com/Glacius-Labs/Markitect/actions/runs/40240295977) on exactly F_final: Linux quality, race detector and Windows quality all passed |
| Structural | `markitect check --revision 4d23f7ae` passes; documentation link check passes |
| Independent review | [final review](v1-final-review-20261025.md): acceptable as the V1 technical source state, no blocking finding |

## Provider matrix (concept §28.2)

| Provider | Status in V1 | Evidence |
| --- | --- | --- |
| Codex: Codex CLI 0.162.0, model gpt-6-sol, effort high, normal account login, Linux containers on a Windows host (Docker/WSL2) | Supported and qualified | [Requalification](v1-requalification-20261025.md) on F, [Phase B arm M](v1-phase-b-arm-m-20261025.md) on F′, [Phase C arm M](v1-phase-c-arm-m-20261026.md) on F_final |
| Claude Code | Not part of V1 (D18); adapter and image removed (#674, #678) | – |
| API-key login | Not supported (owner decision: CLI account login only) | – |
| Account refresh during a long Apply | Not established | – |

## Guarantees and their evidence

The [final review's table](v1-final-review-20261025.md#guarantees-and-their-holders) names the one holder and the controlled tests of every guarantee; all pass in the F_final CI. Native evidence:

| Guarantee | Native evidence |
| --- | --- |
| No Manager on invalid input; a model blocker ends without publication | A Makerspace smoke stopped on a genuine requirements gap (dates past 2099) and published nothing |
| Complete hierarchy in every mandate; fresh Runs | Every native Apply: 6 Managers (Library), 20–21 Managers (Makerspace), one handoff per Manager per Run |
| Child diagnostics at the parent; parent review | A1 native contract proof; a parent caught a child defect in step 4 |
| Judgment bound to the assignment; bounded idle stop | A1 proof; the refined rule (#696) covered by Phase B on F′; no idle stop in any Library Apply |
| Deleting Definitions and Packages | Definition deletion in all five Phase B chains on F′ (#701); the Package merge of Phase C step C7 published in chains 1 and 2 (#705) |
| Build outputs never in candidates, never silently lost (#706, #724) | Controlled tests; no review flagged build output in any published candidate |
| Per-message clock measures provider work only (#725) | Phase C chain 3: Root turns ran 678–924 s of wall time and were stopped only when provider time reached 600 s, so host-served time did not count |
| Seeded defect repaired by an ordinary Implement | Library S4, 3 of 3 on F |
| Strict no-change | Library: met in every chain. Makerspace C9: failed in both delivered chains through optional test edits (limit 3) |
| Public Verify-reject correction path (P2) | Partly: in Phase C M-c3-C1, Verify rejected with 43 Findings and a fresh correction Implement started with them; the second Verify did not start (limit 2) |
| Publication of exactly the verified candidate; readback | Every published native Apply; readback true |
| Resource closure; no credential leak | Every native Apply: no residue and no credential hit; one closure-record gap after an abort (limit 5) |

## Stability

| Series | Source | Result |
| --- | --- | --- |
| Step 4 requalification | F `e11e6cff` | 12 of 12 counted Library Applies, plus P4 |
| Phase B arm M | F′ `528ef346` | 26 of 26 Library Applies |
| Phase C arm M | F_final | 19 of 32 Makerspace Applies published: chain 1 10/10, chain 2 9/10, chain 3 0/10 |
| Source | F_final | Full CI green on Linux, race and Windows, after every known time-dependent test failure was fixed by cause, including one product data race in `await` (#722, #726) |

## Fixes made during acceptance

- The ELF build-output rule could silently drop legitimate binaries; the registry now decides and every left-out file is reported up to Verify (#724).
- Every observed time-dependent test failure was classified and fixed by cause, with no blanket timeout increase (#722, #726).
- The per-message clock no longer counts host-served tool time (#725).
- Owners can delete Definitions and remove or merge Packages; stale links and orphaned shards are realization work (#701, #705).
- The architecture gate skips nested modules outside the product (#723).

## Limits and open items

1. **Per-message clock on a coordinating Root.** The limit bounds a busy turn's total provider work, not inactivity: a Root actively coordinating 20 children reached 600 s and was stopped (seven Phase C steps). Fix plan WP2 turns it into an idle limit.
2. **Correction pair without a budget check.** A pair started with 39 of 256 messages left although it needs at least 42, and the end was reported as "invalid Run request" instead of an honest budget stop, against concept §20.1 (Phase C M-c3-C1). Fix plan WP3c.
3. **Change minimality.** On an unchanged, verified state, Implement added optional tests (Phase C C9 in chains 1 and 2). Fix plan WP4 with [DEC-073](../concepts/register.md#dec-073-change-minimality-on-an-unchanged-model).
4. **Message allowance and question relay.** Questions to parent or children only, an Apply-wide allowance and no ordering of shared prerequisites let question traffic exhaust the allowance (M-c3-C4). Fix plan WP3a removes the double charge; WP3b, ordering by dependency, stays deferred until its native control.
5. **Open questions to a handed-off Manager** stayed open, and its non-action stopped the whole Run (M-c3-C8); the closure records after that abort confirm 22 of 42 closures, although no containers or volumes remained. Fix plan WP5.
6. **One transient provider overload ends the whole Apply** (M-c2-C5, three times); exposure grows with about 21 parallel sessions. Fix plan WP1 adds a bounded retry.
7. **Windows host:** about 35 ms per Git process launch, about 2,100 launches in one three-Run test; a performance limit, not a correctness one.
8. **Should-fix from the final review, scheduled for step 5:** Cockpit leftovers in `internal/bootstrap/lock.go` (dead code). A directory registry entry does not claim an untracked ELF file; such a file is reported and never fails a check (documented boundary).
9. Account refresh during long Applies and a second provider are not established.

## What the owner decides

Technical acceptance of V1 on F_final as a separate decision; release and deployment separately. The fix plan's native controls and the new frozen source F will extend this record with an addendum.
