# Scientist: measurement profile v2 and context-only draft handoff

Status: delivered mechanics and concrete blocked drafts; **hold**. No live Grant,
inference, native product run or study cell was issued/executed. S1 remains open.

Source candidate: `e07d04011d74f59fa360b2ab800c453dc11074ce` on
`codex/government-scientist`. The following delivery commit adds only frozen
evidence and this handoff; the source candidate remains the execution binding.

Freeze: [profile-v2-freeze.json](profile-v2-freeze.json), SHA-256
`fec39a46e1a231e18e33e105ef520826fbd942249886989bc7a0d088f13c99c2`.
It binds 29 sources, 17 evidence files, 15 external draft/input/metadata files and
337 archived deterministic records. Independent source review is recorded in
[measurement-profile-v2-review.md](public/measurement-profile-v2-review.md);
the separate final hash reconciliation is [profile-v2-freeze-review.json](profile-v2-freeze-review.json).
It passed all 29/29 source, 17/17 evidence, 15/15 draft and 337/337 archive checks;
review-file SHA-256:
`4cb7ead6ba4da503cd03fbeda891525d4444fa18f6c1c755186adc47be557f0a`.

The common `observed-native-usage-v2` semantic digest is
`354c021a25adfbdd84d9768e5d51396e5b89dfc8be3e06de91d2c59eb9e9b51a`.
All three arms share it. Numeric commonLimits bytes remain unchanged. Known tokens
with null provider requests permit admission; unknown tokens still block. Tokens
remain retrospective with explicit overshoot. Provider requests and internal
retries stay null/unobservable/non-enforced; 80/480 and internal retry 1 are not
guaranteed caps. Wrapper retries are zero. Existing pinned provider defaults are
allowed only within the finite process deadline. Each harness-launched child or
reviewer needs its own reservation; unobserved native descendants remain a gap.

The frozen focused run passed **52/52** tests: ten profile migration tests,
34 dispatcher tests and eight runtime tests. Raw log and exact source/runtime
hashes are in [evidence/profile-v2/run-1](evidence/profile-v2/run-1/observation.json).
Tests cover preserving old schema rows/stops/counters/authority, atomic rollback,
explicit cumulative allocations, known tokens/null requests, unknown token blocks,
shared arm profile, original-authority resume without another start, metadata argv
rejection and missing/partial/fabricated cumulative migration bindings. No full
Go suite ran. The reviewer made no runtime/model calls and closed all findings in
this agreed scope.

Authenticated account/model metadata was read at 2026-10-07 17:59:55 UTC via the
existing pinned Codex 0.160.1 binary. ChatGPT metadata advertised exactly
`gpt-6.1-sol/high`, without RPC/process errors. Only initialize, initialized,
account/read without refresh and model/list were sent. No thread/turn RPC or
inference occurred; the catalog may be cached, and serving identity stays null.
The metadata result digest is
`71bc27367e6b07e9dc94b61ca06d916110895293f88155f0e2ae3ea20cd0eef4`.

Concrete draft copies and their original external locations are bound by the
[draft manifest](evidence/profile-v2/run-1/draft-manifest.json):

| Artifact | SHA-256 |
|---|---|
| [Request](evidence/profile-v2/run-1/request.draft.json) | `fc4e07be4d506167ef6dafec3c19b63014732797f70c8bd94386b75731b4f050` |
| Execution (operation excluded) | `074253724e7d7033db3104a886e797de352f07678d6b7e0d0cfcc8a0d339392c` |
| [Protocol](evidence/profile-v2/run-1/protocol.draft.json) | `e102af2c99882174afe490ac4aa9605f5cb9e91645fb50f1a4aa34f45ae8b12b` |
| [Grant](evidence/profile-v2/run-1/grant.draft.json) | `d0d5ca49e98664c60e41148ca8596642fbc7d65737519304e8e2d63c5d456b2f` |
| [Ledger adoption](evidence/profile-v2/run-1/profile-adoption.draft.json) | `dee87b9679eb34d8c73c91ff5d2371faec2abd881a2eac555990560b07626aed` |

External draft root:
`C:/Users/Consiliari/Documents/Scientist-Probes/profile-v2-20261007-1791395973561/context-drafts`.
The proposal permits exactly one **additional** context session: model/high as
above, 180 seconds, parallelism one, one wrapper turn, retrospective 10,000 tokens,
zero children/retries/continuations/repairs and no new purchases. The two named
files are synthetic released `context.txt` and Actor-owned `actor-own.txt`.
Source, Python, runner, config, base commit and exact input bytes are bound.
There is no separate access/write/process probe. Exact-file tool policy is
cooperative; requested read-only sandbox is not proven OS isolation.

Both old diagnostic ledgers are unchanged. Three starts remain consumed, known
subtotal 10,009 tokens, two unknown token attempts, total null, requests/retries
null. Their raw hashes are respectively
`74dfd51e2802bc67131891b28c6858129ff5ef4a50a6f1fe06a8793a7250732f` and
`33bd2ba59b11ab6d54f75c55e6d12b00ec6b90df00106a1419aed239cef2982c`.

**Remaining blocker:** the old diagnostic schemas have not been migrated into
the common dispatcher ledger. The drafts name the existing selected-runner
identity/ledger, retain all three starts, and propose absolute ceiling four
(three consumed plus one additional). They have `blocked-legacy-diagnostic-schema`,
null migration receipt and `liveExecutable=false`; no empty replacement ledger or
new trial is created. Missing/partial ready bindings also fail closed. Unknown
historical tokens remain blocking, so approval/status changes alone cannot launch.
Coordinator must first resolve this cumulative accounting boundary, then approve
the exact one-session Grant/validity interval and adoption, freeze Protocol and
rebind affected digests. This handoff grants none of those actions.

Classic native integration/readiness gaps remain. Government G2 at
`2b604d1aeabd8cef49fb19177cc5d29cfa3b1493` is only the public intermediate
interface; G3–G5 and a final StudyGovernmentVersion are not proven here. No product
candidate was silently advanced. Next action: hold for Overseer.
