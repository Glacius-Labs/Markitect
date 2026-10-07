# Native integration preflight review

Disposition: **approved for the exact, source-authorized deterministic fixture mechanics described here.** This is a static independent preflight of the frozen inputs and source at the hashes below. It permits only the one bounded Government fixture run/resume and Classic fixture flow already authorized by `native-s1-integration-fixtures-20261008`. It does not approve provider use, real Actor sessions, study cells, semantic claims, human acceptance, publication, or any additional run. S1 remains open until post-run evidence is independently reviewed.

This review started no native process, Actor, role delegate, provider, CLI probe, or test. The two final input-validation receipts report zero native starts and zero provider calls. The prior budget snapshot records one completed Government `inspect` start (reserved 150 seconds, actual process wall 0.453 seconds); it does not record a positive queue, resume, role, or Classic start. No refill is recorded.

## Frozen identity and authority chain

The source authorization is `released-native-grant.json`, SHA-256 `b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e`. Its coordination snapshot is `coordinator-authority-snapshot.json`, SHA-256 `04484d4220dd7383af1d2675a95384554380dccc5a14ee8fc5908c3a474ede58`, at pointer `threads[name=Scientist].evidence.nativeIntegrationPreparationGrant`. The grant authorizes disposable public fixture repositories, separate clearly synthetic ledgers, one positive case per product and only a directly related interruption/resume or replay refusal; per product it caps native CLI/controller starts at 8, deterministic role starts at 12, parallelism at 2, and process-time reservations at 1200 seconds. It explicitly sets real Actor, provider, and study-cell authorization to zero. Role starts must be reserved before effects; controller accounting is distinct from a nullable Actor attempt. Historical usage, stops, and null values must be preserved. The grant status says “Assigned; usage not yet reported”; this review makes no usage report.

The grant also pins Government source `04e225d5caee78c2a198607143863fca1e829750` and executable SHA-256 `12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`, and Classic held source `c91363b7ac4decbe87212ff0f588b5451581a152`, runtime source `7dbd599c81540c8203a1b7f83afbc335174f4f1f`, and executable SHA-256 `2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4`. The Government Actor repository is clean at base `fb6bfc1c6eefa799606dc19cb990bad139f34662`; the Classic Actor repository is clean at base `20c0b4c85135bc5d76fe4d337ff5450b8d7178bf`.

The final current preparation records supersede every provisional preparation record. Their authority bindings match the live files byte-for-byte:

| Input | Government | Classic |
|---|---|---|
| `preparation-current-3.json` | `a586b715cc008c9110d333a5a5969d748448e2057449d6b4a9233a995eefcbbe` | `623f0b6b8e6409d6459fbde99a59f340f04fb2e98e292d4cc5ee4c022f6fa59f` |
| Request | `79c482e910116a57597af9c0a856e6892ff3a0c947a8b66d2462ac58fecc1a97` | `ccbe23720348b96488b63c6ff7a54ac5653e499a99d70d62bf8949bdd5cc0f2a` |
| Runtime | `00c945cbe6ecc10c33da15b4d2ab7f9ac1a57f5c0f74bc1ed58a25ac642268c8` | `878af9d3e01e8afe0a870e29df729956f8375ab4c690f9feea412acff5521c60` |
| Common Grant | `45cdac59df11a345aa0cf9662c4df669cbe258eea41687a569bf3e30320487a2` | `5441e08d00ede1eae95e48492a92907603f52b03bd2dd7890c8e8a9cd5e336c7` |
| Frozen Protocol | `33ac6e763c84e5e97fc85e97438443a41dce8c7844f4f9fc92a0bc2d111125ed` | same |
| Authority locator | `f4b5fad9d4fca3ba0cf2b008cd2c1c817ae869320e0ce1cff07ac4087ff71c14` | `b498f4f4b3d1916ea10c11ed026d8590bf023addc4c4298b5c4eca00ffb0b9fb` |
| Released role authorization | `f549cecbb87a7dde4db8063f32297598464fde65ebbc3b255a169aada16045b2` | `fd096fc4f03d03d97b73f5d5ee17cf6aa74138cf1cef52e7f90d22b5d6f8bd72` |

Each common Grant is approved in mechanical mode, binds the exact Request hash and execution digest, is bounded by finite `notBefore`/`expiresAt`, and binds the same frozen Protocol and source grant. The Protocol is `status=frozen`, `mode=mechanical`, `semanticAcceptance=false`, and `humanAcceptance=false`; its source pins agree with 25 current source files. The Requests identify provider-free native fixtures and distinct synthetic fixture ledgers. The role authorizations are approved, use those same Request/runtime paths and hashes, carry the exact fixture authorization `{sourceGrantKey: native-s1-integration-fixtures-20261008, classification: nativeFixture, providerUse: noProvider}`, and cap each product at 12 deterministic role calls. The common Actor-session ceiling and each role authorization are compiled beneath the source allocation; they do not authorize actual product Actors or provider sessions.

Final root-driver input validation passed for both exact Requests, with no starts or provider calls. Receipt hashes: Government `government-input-validation-final-2.json`, `8faea719fc1771831057e648b8d76361c69a32bcfdcedbdee199c71da8fc42a9`; Classic `classic-input-validation-final.json`, `e1e93013f3739e34ccfb8729f4c0a82574b6fbd5245b35297afac82502f5078d`. All 25 Protocol source pins match the current files. The earlier `preparation-current.json`, `preparation-current-2.json`, and other provisional receipts are superseded and are not authority for these final inputs.

## Reviewed implementation and accounting

The source reviewed for this exact input set has these SHA-256 values:

| File | SHA-256 |
|---|---|
| `run-native-integration.py` | `d1b8ba6baa9576a3b724084d6815c60b46edca81a9d9c65c048c94bfc77bd7ed` |
| `runtime/dispatch.py` | `50ad2a9bf46f039eaf360508f9090f133db848d9c3f130ecd2fdf4f5d6f8bff9` |
| `runtime/government.py` | `f14869c0d1977163a40900512ddb249cefae593a119169a17a5220875c503cd3` |
| `runtime/government_roles.py` | `7c14e1490cb0d500dd39d5cb7929d7f7b879bdce2f0d64ba66b81861f80d7a87` |
| `runtime/ledger.py` | `d4e3aa98ea8486cc3821a658e93c823794750856e27e48cb96fdcb216f6afe8f` |
| `runtime/native_controller.py` | `1f2e92652593805c8e4f38e3db92b7ebcd6cb4fcb5553be4752420159b440ec6` |
| `runtime/native_fixture_budget.py` | `379e6109de05d909a2867d84b6f9b75f401b5f661da7ae6f06c9e91c32732d34` |
| `runtime/classic_integration.py` | `63860426ac9412f591ffe2b8ab1d3944bba5bc49a2c7f45c2c8bd0634a83b48c` |

The bootstrap locator and digest do not grant authority by themselves. The loader verifies the digest-bound request/grant/protocol/role authorization and exact source grant, revalidates the Request, checks product/trial/dispatch/execution bindings, and requires the already claimed controller dispatch with its separate active controller row. There is no placeholder Actor attempt. Before each role delegate, the wrapper validates the exact released runtime and role authorization, invocation and slot, expiry, executable/runtime pins, and current controller context; it reserves the unique invocation and common-ledger attempt before starting one delegate. A duplicate invocation cannot relaunch, retries are not automatic, and ambiguous reservations remain occupied.

For the exact mechanical fixture mode only, `government_roles.py` separately records zero provider calls/turns/tokens when the source grant, fixture authorization, and one of the two hash-pinned closed deterministic delegates all match. It preserves raw `providerRequests`, `providerTurns`, and `reportedInputPlusOutputTokens` as null and labels the distinct accounting basis; it does not synthesize a provider report. If any of those bindings fail, usage remains unknown and the existing admission rule blocks the next role. The focused fixture-accounting tests include both the authorized zero-admission case and the negative unknown-usage case.

Government reserves the bounded controller dispatch before queue dispatch and finalizes that same reservation from process evidence; role starts use separate prior reservations. The directly related resume requires a completed predecessor, uses the existing queue-state directory, reserves another start, and checks that Actor-session ledger state did not change. Exceptions consume the reservation and are not retried. Classic uses the exact ordered `execute → apply → verify → audit → apply-replay` controller flow. Apply is gated by a separately supplied review bound to exact Execute bytes and digest; Verify consumes saved Apply, Audit is read-only, and replay requires the same review. Each process is reserved and captured; missing or invalid review cannot advance to Apply. These records establish only the bounded fixture protocol mechanics.

Quota interpretation: the fixture grant permits at most 8 native CLI/controller starts per product, 12 deterministic role starts, parallelism 2, and 1200 reserved process seconds per product. The parent controller reserves 150 seconds per native start (a conservative start envelope; the Government request itself has a 38-second wall cap). These reservations, controller accounting, and role-process accounting are distinct; they are not a claim of measured aggregate wall time across arbitrary OS descendants. Cleanup does not refund quota. No extra quota or start is inferred from this review.

## Validation and limits of the conclusion

The relevant successful focused checks are retained as evidence: 88 tests in `focused-tests.log` (SHA-256 `b3a04566cdc939d438583e7dcdd21fbfafe2f02c14945690c8fbdbeab48e2053`), 34 in `final-delta-tests.log` (`3dcf8b402ba92f20bf6944dd97b6551a970d13632d69513dfe015a674ff4f716`), and 23 in `final-fixture-accounting-tests.log` (`7ce006c569bf51158790ef19f952faf17846313905d5bf96cf6bb09998551ee8`). Each log ends `OK`. The two latter focused gates cover the late input-binding and no-provider accounting changes.

Government's acceptance translation now requires an AcceptanceDecision, the correctly bound candidate/evidence/round/provenance, complete configured cabinet coverage, and positive assent outcomes before a positive completed result. The pinned G5 source treats vote slot/run identifiers as provenance labels, not identity authentication. The deterministic delegates deliberately produce protocol records and cannot establish real reviewer identity, independence, semantic correctness, or human acceptance. Those claims remain false/unproven even if the eventual fixture process exits successfully.

Classic is pinned to the exact readiness artifact and packet digest. Its protocol test double is fixed, provider-free, and explicitly labeled “no semantic judgment.” The native ordered flow has not yet been run at this preflight point; guarded Apply, fresh Verify, Audit, and replay refusal still require actual captured process and review evidence in the post-run review.

This approval is conditional on executing only the already authorized bounded paths against these exact hashes, with the source grant and finite deadlines still valid. Any change to source, driver, Request, Grant, Protocol, runtime, role authorization, binary, or input-validation receipt invalidates this preflight and requires a new independent freeze review. It does not close S1 or certify product readiness.
