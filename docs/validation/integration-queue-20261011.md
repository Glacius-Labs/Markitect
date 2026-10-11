# Integration queue — 11 October 2026

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
- #219: the existing unpushed inventory commit was reviewed and normally pushed. Its image provenance is cold build [38106437991](https://github.com/Glacius-Labs/Markitect/actions/runs/38106437991), image ID `sha256:13cd8225ab21ad16bb2d8169758323b769b6195ee50bc5e983a661643f88f539`, inventory SHA-256 `78f0ab386612ff4b12d37a9202ff3c5f5de61192a92ada65a7b3e17a0061df0c`. Fresh head [CI 38175483226](https://github.com/Glacius-Labs/Markitect/actions/runs/38175483226) and [provider-free playground smoke 38175483212](https://github.com/Glacius-Labs/Markitect/actions/runs/38175483212) passed. Additional Windows Python units on the composition ran 320 tests in 478.531 seconds with two failures and eight skips: unpinned-Dockerfile expected error text and an LF/CRLF inventory byte comparison. These original failures are preserved and a scoped follow-up is being investigated.

## Remaining scoped work

The audit reproduced private child Statement details reaching ancestor briefing loaders. Its separate BUG-01 repair uses each historical Manager context to project event sides and relations while retaining canonical history and opaque event IDs. The Context and brief-list callers also require a separate no-write repair. Neither is accepted merely because an earlier PR or unrelated suite passed.

Timer injection/fakes (TEST-05) remain incomplete. ARCH-08's benchmark and release analysis still has legacy consumers; ARCH-09 removal is future work. Historical recovery findings without a reproduced current failing case remain unconfirmed. Runtime/QA implementation requires explicit zone assignment after the active repairs.

The delegated dashboard follow-ups cover the DEC-023 clarification, assessment-derived assumption provenance and a KG-04 protocol using Shop plus Markitect. KG-04 measurement remains NOT RUN; expectations require an independent author and must be frozen before impact is revealed. No method or provider study, credentials, budget, Government implementation or production acceptance is implied.
