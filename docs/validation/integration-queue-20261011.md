# Integration queue — 11–12 October 2026

This record continues the dated [session handoff](../development/session-handoff.md). The owner delegated review, repairs and normal integration of its six open PRs to the integrator. Source checks, independent scoped review, provider runs, method measurements and human acceptance remain distinct. Historical and dirty worktrees were preserved; no release or provider run was performed.

## Original queue

Each candidate was independently reviewed and composed onto the then-current remote main. Linux Quality and Race passed on the PR heads before their merge commits. Local checks below identify their actual composition; later main commits do not inherit a claim that those exact local tests were rerun.

| PR | Full reviewed head | Local composition | Remote-main merge |
|---|---|---|---|
| [205](https://github.com/Glacius-Labs/Markitect/pull/205) | `919f96bb293d5cc1e5ed56cd705184d91cbb3edb` | `1290150f5974dbee6ace698d5da612cbb9d8ae94` | `8e3d88a54fdf556d3a7484da541ca736bf368c7c` |
| [221](https://github.com/Glacius-Labs/Markitect/pull/221) | `a4da3bfc768038e08da306b0f56e13d181474e08` | `22d8d8ecafc83318b3607f73d6ed9782eb14b315` | `50e6e1d935f3a85316ad46259b96130a715323f3` |
| [218](https://github.com/Glacius-Labs/Markitect/pull/218) | `e6ca3492f3e5c46e7a89a31c6f8a62ff8ac5e666` | `86bf6bbc90214fb9cd7d0fb5cc90fb6faded89f8` | `f1316f49f77ad14ec3956f5686035bd2303488ab` |
| [222](https://github.com/Glacius-Labs/Markitect/pull/222) | `0c2057b079492e79a8190c2088d8918dc79c9fa5` | `a6c745e0dd8923e49e703413023584ead5b7142a` | `372ba98c5ff27047548a11765bacc628706c4476` |
| [223](https://github.com/Glacius-Labs/Markitect/pull/223) | `cc47589d8e3b313ec72247590a21249ccf268319` | `7387c8c65902a4059d9da38f5a171a8d7712987b` | `7f1fde0df4519f8205a45e5d5c917df7d58d5cbd` |
| [219](https://github.com/Glacius-Labs/Markitect/pull/219) | `b64f4748369e3410a414e8a88bd574fd5d975d74` | `7c7cacbbcf12b0d14c685ac544b91ba8a93de29f` | `36215f84656ec321d5096bf23ad5df56d27a895b` |

All listed compositions passed vet, the self-hosted model check and managed-artifact accounting. Relevant documentation links and changed entrypoint examples passed. The prose-only #222 candidate did not repeat the unchanged full local suite.

Local composition locations were `C:\Users\Consiliari\.codex\worktrees\bb5d\Markitect` (`dev/briefing-composition`, #205), `C:\Users\Consiliari\.codex\worktrees\integration-candidates\Markitect` (`dev/playground-composition`, #219 and earlier queue compositions), and temporary clones under `C:\Users\Consiliari\.codex\worktrees\bb5d\integration-evidence` (`mcp-candidate`, branch `dev/mcp-current-composition`; `entry-candidate`, branch `docs/entry-composition`). Candidate sources were clean when the listed checks ran. Raw local command outputs and PR-description files are preserved in that evidence directory. These local paths are provenance references, not portable validation fixtures.

- #205: fresh full `projectbriefing` package passed in 1406.333 seconds on its composition.
- #221: fresh guided-preview history/no-write regressions passed in 477.530 seconds on its composition. Its base was retargeted to main after #205 merged.
- #218: fresh full MCP package passed in 0.294 seconds; schema/parity/tools-list selection passed in 69.471 seconds. Isolated actual stdio `tools/list` responses shrank from 215,476 to 156,404 bytes (27.41%), preserving 23 tools.
- #223: explicit cross-Manager evidence regression passed in 15.779 seconds on `e6281660589500ceb04a08869099755793d72ffd`. The subsequent composition above added only #222's AGENTS instructions; its Go source was unchanged. Structured private/unknown child verdict subjects are filtered. Arbitrary child summary, risk and question prose remains outside that filter.
- #219: the existing unpushed inventory commit was reviewed and normally pushed. Its image provenance is cold build [38106437991](https://github.com/Glacius-Labs/Markitect/actions/runs/38106437991), image ID `sha256:13cd8225ab21ad16bb2d8169758323b769b6195ee50bc5e983a661643f88f539`, inventory SHA-256 `78f0ab386612ff4b12d37a9202ff3c5f5de61192a92ada65a7b3e17a0061df0c`. Fresh head [CI 38175483226](https://github.com/Glacius-Labs/Markitect/actions/runs/38175483226) and [provider-free playground smoke 38175483212](https://github.com/Glacius-Labs/Markitect/actions/runs/38175483212) passed. Additional Windows Python units on the composition ran 320 tests in 478.531 seconds with two failures and eight skips: unpinned-Dockerfile expected error text and an LF/CRLF inventory byte comparison. These original failures are preserved; the separate fixture repair is recorded below.

## Delegated dashboard follow-ups

| PR | Full reviewed head | Local composition | Remote-main merge |
|---|---|---|---|
| [226](https://github.com/Glacius-Labs/Markitect/pull/226) | `7851b74b78b6eb5c3a570742c20ce4762117a209` | `9b0fe19e6e6b7f2c1a8963152bc14fa52ad9b62e` | `a0b79744371387f5c1e7f725fad12395d01f68a6` |
| [227](https://github.com/Glacius-Labs/Markitect/pull/227) | `b7be6cba86ca98472fc8a23cff68dd039684fb25` | `da12912bb635d8819fe0681b53ec3a7f4a9d8e0c` | `a225d0caab868e43bb8afda31a78ae73d07a9461` |

Both documentation heads received independent scoped approval and passed Linux Quality/Race: [226 CI](https://github.com/Glacius-Labs/Markitect/actions/runs/38177825309), [227 CI](https://github.com/Glacius-Labs/Markitect/actions/runs/38178262827). Their compositions in the integration-candidates worktree passed vet, fixed model checks, managed-artifact accounting and relevant links. No unchanged full local suite was repeated for prose.

The [delegation source](../concepts/sources/dashboard-delegation-20261011.md) records the new DEC-023 authority and assessment-derived ASM-006 provenance. A temporary test against fixed source `7f1fde0df4519f8205a45e5d5c917df7d58d5cbd` passed in 2.506 seconds: explicitly realized Statements and their direct consumers receive `change` priority, while Explain preserves impact sets and routing. The injected test's SHA-256 was `F0C4AB7AD0A539B7051CB28BDA093479030C1FA130A853B07E2E43D37B621D86`; it was not committed and did not run a full Plan. The hypothesis comparing forgetting with misunderstanding is untested and is not asserted by the original owner wording.

The [KG-04 protocol](impact-assessment-protocol.md) is **PREPARED / NOT RUN**. It pins Shop seed `069f0c1035b5e1c730539af323d10eac9a268c9e` and tree `d7f5224423e7d8105ebfce3f583e51b6342c890a`, and Markitect source `7f1fde0df4519f8205a45e5d5c917df7d58d5cbd`. Both expected lists remain unauthored and unfrozen. Proposed Explain metadata is a hypothetical measurement case, not adopted product intent. Preparation specifies independent expectation freezing and descriptive recall, missed files, width and excess without PASS thresholds.

## Scoped repair follow-ups

| PR | Full reviewed head | Local composition | Remote-main merge |
|---|---|---|---|
| [228](https://github.com/Glacius-Labs/Markitect/pull/228) | `16f53385ac23623f182dcac84c1ee555365bc1a6` | `03c53d1653642e1d8b8b7ade32532ba9773b6e8d` | `0e1c769f279054c776b2cb54c424a0735f7eeda4` |

#228 changes only two Python fixture writes to explicitly preserve LF. Windows newline translation had changed the raw Dockerfile ARG parser input and inventory bytes. The focused pair passed in 7.278 seconds, and all 320 units passed in 483.538 seconds with eight skips. Both runs preceded the commit on base `36215f84656ec321d5096bf23ad5df56d27a895b` plus those two edits, committed unchanged afterward; they are source-equivalent precommit evidence, not fresh runs at the commit SHA. Independent review approved that exact final head, [Quality/Race 38178623047](https://github.com/Glacius-Labs/Markitect/actions/runs/38178623047) and [Playground smoke 38178622972](https://github.com/Glacius-Labs/Markitect/actions/runs/38178622972) passed there. Its composition on Main after #227 passed vet, fixed model, managed-artifact and module checks. Its clean raw Git worktree is preserved at `C:\Users\Consiliari\.codex\worktrees\playground-windows-fixtures\Markitect`, branch `fix/playground-windows-fixtures`; it could not be attached as an app-managed worktree.

The audit reproduced private child Statement details reaching ancestor briefing loaders. Its separate BUG-01 repair uses each historical Manager context to project event sides and relations while retaining canonical history and opaque event IDs. The Context and brief-list callers also require a separate no-write repair. Neither is accepted merely because an earlier PR or unrelated suite passed.

Timer injection/fakes (TEST-05) remain incomplete. ARCH-08's benchmark and release analysis still has legacy consumers; ARCH-09 removal is future work. Historical recovery findings without a reproduced current failing case remain unconfirmed. Runtime/QA implementation requires explicit zone assignment after the active repairs.

The delegated dashboard follow-ups cover the DEC-023 clarification, assessment-derived assumption provenance and a KG-04 protocol using Shop plus Markitect. KG-04 measurement remains NOT RUN; expectations require an independent author and must be frozen before impact is revealed. No method or provider study, credentials, budget, Government implementation or production acceptance is implied.
