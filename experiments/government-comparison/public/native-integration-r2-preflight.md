# Native integration r2 independent preflight

Review date: 2026-10-08. Reviewer: independent Scientist strand. This is a static, source- and evidence-bound preflight for the exact corrected mechanical fixtures. It is not product acceptance, semantic validation, or evidence of a successful native run.

## Decision

**Conditional go for the exact one-case-per-product allocation below. Do not start a product process until the reviewed source is committed cleanly and the immutable r2 freeze binds that commit, this review, the exact inputs, and the driver.** At review time the worktree is still based on `f35c8209cd8902bff17e69f05ee7716e352ee4b2` with modified tracked files and untracked run-2 evidence; therefore the clean-commit and freeze preconditions are not yet demonstrated here.

The static input-validation receipts report `validated: true`, `nativeStarts: 0`, and `providerCalls: 0` for both arms. Budget preflight receipts report the original three historical starts, no refills, no provider or real Actor calls, and no study cells. No r2 product start is evidenced by these preflight records. Historical failed/incomplete attempts remain counted and immutable.

## Authority and bounded allocation

The authorization source is the immutable coordination snapshot at `C:\Users\Consiliari\Documents\Scientist-Probes\native-metadata-fixtures-20261008-r2\coordinator-authority-snapshot.json`, SHA-256 `1b2e361ce2a321149a65694b6586bab57e74388dc5486bc2d4f8cbedb2b7a75c`, pointer `threads[name=Scientist].evidence.correctedNativeIntegrationGrant`, key `native-s1-corrected-integration-20261008-r2`. Its released correction envelope is SHA-256 `be78d1156bfe49f46793906d94dfd0ec82ace550cbb107b425d383d45e25870f`. The original envelope remains separately bound at SHA-256 `b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e`; the correction adds execution authority without replacing that original provenance or resetting its history.

The grant is limited to deterministic synthetic fixtures: no real Actor, provider, metadata app-server, or study calls; no purchases or product mutations. The original durable native-start ledger is reused. Its prior accounting is Government 2 starts/300 reserved seconds/1 wrapper attempt and Classic 1 start/150 seconds/0 wrapper attempts, with 0 delegate attempts. R2 caps are Government +2 starts/+6 role invocations including failed wrapper starts/+300 reserved seconds, and Classic +5 starts/+6 role invocations including failed wrapper starts/+750 seconds. Cumulative ceilings are Government 4 starts/600 seconds, Classic 6/900, total 10/1500. Products run sequentially, at most two role sessions in parallel, 150 reserved controller-and-role-session seconds per native start, 38-second native process deadline, and controller windows of 38 seconds (Government) and 180 seconds (Classic). Arbitrary OS-descendant aggregate wall time remains unknown. The granted cases are one Government queue attempt, with related resume/replay only after success, and the ordered Classic Execute, reviewed guarded Apply, fresh Verify, Audit, and stale Apply replay, each later step gated by its prerequisite. Stop on the first failed prerequisite; no repair or relaunch is covered.

## Source and control review

Reviewed source hashes:

| File | SHA-256 |
|---|---|
| `experiments/government-comparison/run-native-integration.py` | `2ba7b3b2cd3c1f47a949f9d42cd75d9db03c44284155d5892a21d5a470ebaaa7` |
| `experiments/government-comparison/runtime/classic_integration.py` | `4bac5c17703b128de4e7d2cec7a53174fd6a8e6e5474930b4be1065a99ba857a` |
| `experiments/government-comparison/runtime/dispatch.py` | `7e90fa68c9455d7e55f0cfe7de4fc3d81afd9d30c161998c8abd29b7dcd32421` |
| `experiments/government-comparison/runtime/government.py` | `ba51b745491ee0393c373a922d0bf63a51bfabccc432b76b5aa9f79715dcf862` |
| `experiments/government-comparison/runtime/government_integration.py` | `374a722a860c53eea1e44a36ebc16b33a223b6f26706fbe9be1be74913c04758` |
| `experiments/government-comparison/runtime/government_roles.py` | `a81dabba3aedf66462c362d88826d07f67aa054877ec5eeb26ceb3c1daab7a6f` |
| `experiments/government-comparison/runtime/native_controller.py` | `96aeac564694cf47e4eee3be2e3565c54faa834d80ef091a7dcdacec7d041e0c` |
| `experiments/government-comparison/runtime/native_fixture_budget.py` | `2b142396a45a105d0ebd73bbf9d9e86a76fdf12960743c637e9b13688a0f9981` |
| `experiments/government-comparison/evidence/native-integration/run-2/release-inputs.py` | `0c7e8ac4b8174e12f59d2ed3e16dd962c7998259c3f8532aba145faae03e4175` |
| `experiments/government-comparison/evidence/native-integration/run-2/preparation-driver.py` | `42d7ae211469c8dd69d5f99473b2cd0dafb58b2a364f432a893e95b914eb47ec` |

The source review found the following controls in the reviewed snapshot:

- The corrected grant is validated separately from the original envelope, while original source provenance, the original SQLite ledger, and old attempt rows remain bound. Budget admission checks cumulative product/total limits and correction identity; there is no refill path.
- Runtime Requests and released input sets bind the exact Authority, Grant, Protocol, product binary, runtime and role files, corrected source snapshot, and correction envelope. Both role authorizations set `maxCalls` to six. Input validation also requires a clean pinned Actor base and reports no native starts or provider calls.
- Government wrapper diagnostics validate the corrected source and six-call cap before project imports, commit a wrapper-attempt row first, and retain stdout/stderr. Dispatch translates missing evidence/identity conservatively to incomplete, preserves raw process and translation receipts, and terminalizes the controller without inventing a delegate attempt or acceptance.
- Classic validates its exact released budget binding before reservation. Bootstrap failure after claim is terminalized as incomplete without claiming a product process. The flow finalizes through `finally`; the RecordStore precondition checks absence without deleting prior evidence. Apply is guarded by the bound review and Verify is fresh; later operations remain ordered behind their prerequisites.
- The driver validates both complete requests before any start and sequences products. The reviewed allocation uses the existing shared reservation accounting; no extra per-role wall-time reservation is added on top of 150 seconds per start.

## Exact preflight evidence

| Evidence | SHA-256 | Result |
|---|---|---|
| `evidence/native-integration/run-2/authorization-grant.json` | `be78d1156bfe49f46793906d94dfd0ec82ace550cbb107b425d383d45e25870f` | Released correction and canonical source pointer |
| `evidence/native-integration/run-2/history.json` | `8551e76037d32db4665dfc10052785f48cefbd01d0859e9ea73beb2cd9bff581` | Prior counts and ledger identities bound |
| `evidence/native-integration/run-2/government-input-validation.json` | `e15ff5a200f769a94883c951d47269c74d7d46661bd21a36922ff76cc2398c70` | Validated; 0 starts; 0 provider calls; Request `bd9f1bfeb5e85fda594046bcbdb62facd6970d95ce8e94f141632b38ce45e1b5` |
| `evidence/native-integration/run-2/classic-input-validation.json` | `0828ba85566bf5e0377d63a19997d894dea037913e5c1725625f02012b71d672` | Validated; 0 starts; 0 provider calls; Request `bcc244e30b59eb24b8a9708f3e83a44b7167f9dbe6aad6d296d371f21aaa20e7` |
| `evidence/native-integration/run-2/government-budget-before.json` | `7d4f5da393009217081cde0a47b3f4139180deca302c30e1fd67311758af0a9f` | Original ledger and r2 ceilings admitted; no refill |
| `evidence/native-integration/run-2/classic-budget-before.json` | `7d4f5da393009217081cde0a47b3f4139180deca302c30e1fd67311758af0a9f` | Same shared allocation state and ceilings; no refill |
| `evidence/native-integration/run-2/focused-tests.log` | `061b40ff19c95fea83e5fb1d4097799ea863d4a1af647968941154387695b2a5` | 102 focused tests passed, as recorded by the preparation owner |
| `evidence/native-integration/run-2/final-authority-tests.log` | `fe1f7d88a1ca8e136120f8dcfaa7306e0d0cb54b342e57e69794c1273b470c69` | 9 final authority/budget tests passed, as recorded by the preparation owner |

The two budget receipts are byte-identical because they record the same shared allocation state. Test results are reported from frozen logs; this reviewer did not rerun tests or call the native tools.

## Limits of this review

This review is a static safety and binding check. It does not establish successful product execution, semantic correctness, independent human review, final votes, acceptance, or user value. It does not certify arbitrary descendant-process wall time. The controlled fixture remains distinct from actual Markitect Actor/provider or study activity. The clean-source commit and immutable freeze are required to turn this conditional preflight into launch-ready authorization.
