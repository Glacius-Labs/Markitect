# R2 Classic post-run review: Execute incomplete

**The corrected R2 Classic Execute failed. No Execute approval was issued, Apply was not started, and no resume or replay followed.** This report records only the already completed process and saved receipts; it authorizes no further work or start.

## Execution binding

The exact Classic Request SHA-256 `bcc244e30b59eb24b8a9708f3e83a44b7167f9dbe6aad6d296d371f21aaa20e7` matches both the released R2 Request bytes and the R2 preflight freeze input. The preflight freeze is SHA-256 `55c53bf49b006165c3305e470c3258ba1807079511793ffc899e8dd46845d0b9`; its review SHA matches [the R2 preflight](native-integration-r2-preflight.md) at `0109d475f15697ad1025f8bdae524d42fb1a617966959eb75b2b645dc0d41dad`.

The released correction envelope is pinned at SHA-256 `be78d1156bfe49f46793906d94dfd0ec82ace550cbb107b425d383d45e25870f`. The outer result binds common Grant SHA-256 `29cfe80418bdad5d9e7e93e6bd2b9ca20c875b4068d9e104b23a6c5316b87de0` and Protocol SHA-256 `bfa289b3f40f7399d8c2f85d12b95cb82908c2abb948298fb6431cbc09b42864`.

The native process receipt records Classic v0.14.1 binary SHA-256 `2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4` and runtime source `7dbd599c81540c8203a1b7f83afbc335174f4f1f`, matching the Classic pin. The R2 Actor checkout was clean at the exact base and revision `60b6835b4bbd3af7ad1103af85e52a01412f83cf` used by Execute. The initial R2 failure is not the prior `store.json` RecordStore setup error; this run failed later in the configured role wrapper.

## Observed failure

The native Execute report bytes have SHA-256 `cffa90f77ba974af5dbe30c06394f395d69984a2c8299af1bacd0fe86e947ec2` and native report digest `sha256:ade4a5c804c813ae9e9bbc1d226a929513ffff924f8ccc554a05ec48e584961a`. The report status is `incomplete`; the outer flow carries the same digest and status, has `candidateCommit=null`, `usage=null`, and `inferencePerformed=false`.

The pinned native process exited 2 after 3.360 seconds, without timeout. Its 23-byte stderr is `external runner failed` (SHA-256 `67b2944b65106ea64c392608a653bc5a690979e5e4b8e2f90cf2c687ac64240e`). The underlying configured wrapper invocation is captured once. Its stdout is empty; its stderr SHA-256 is `cbc7874bc14238a0e17d5a730bf833854e06e17ceba4d105273a1515ce00fea8`. That diagnostic reports a `ValueError`: the Classic Invocation is missing the exact configured scope identity, raised while the local role bridge resolves the invocation before delegate admission. This is the new observed boundary failure. It is not the old RecordStore error. The evidence does not determine whether the invocation lacked an expected field or the adapter expected scope information not provided by this native call, and it does not establish a product defect.

Read-only SQLite checks found one Classic wrapper start, zero common-ledger `attempts`, one controller run `incomplete`, and one controller process `incomplete`. The shared native-start ledger records this single R2 Classic Execute as a 150-second reservation. No Execute review file, successful Execute capture, Apply report, or Apply process exists. The incomplete report cannot pass the approved review gate, so no exact stdout SHA or report digest is approved for Apply.

## Receipts and limits

| Receipt | SHA-256 |
| --- | --- |
| Native Execute report bytes | `cffa90f77ba974af5dbe30c06394f395d69984a2c8299af1bacd0fe86e947ec2` |
| Native process receipt | `0a3cb0952702bcc3d2e8065fa5c48f0851ffecaa3b93d05ab5a8768510abe1df` |
| Native Execute stderr | `67b2944b65106ea64c392608a653bc5a690979e5e4b8e2f90cf2c687ac64240e` |
| Outer R2 flow result | `c4af3133a70ff5d4f17feffb150f18f69ec1683dfe2fd5dbdd517c96fcd61aca` |
| Wrapper start record | `ec963d50a94304270bd877d1e8dd67b69bbe834e7c79eae4cdda3caeee9aef14` |
| Wrapper finish record | `3cd29caedfd11864eebfc4371bbae91788a72c34a5863199be112fc4e788d395` |
| Wrapper stderr | `cbc7874bc14238a0e17d5a730bf833854e06e17ceba4d105273a1515ce00fea8` |
| Classic fixture ledger at review | `52c6e65e64db90b00d8a65cc5e4d8714c726b0f7052f27187091b89ed1d1b7c6` |
| Wrapper start ledger at review | `c9ed8599752ab17fc603649eb29b6326840e2bb92340cf49abebcb06d4ea29c0` |

The wrapper recorded provider usage as null. The outer result also leaves usage null; it is unknown, not measured zero. No common-ledger delegate attempt or real Actor/provider session was made. The R2 Classic allocation is consumed for this Execute; no retry, repair, Apply, or replay is authorized by this result. S1 and product readiness remain open.
