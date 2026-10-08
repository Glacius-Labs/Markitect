# R3 Classic offline preflight evidence

Date: 2026-10-08

This record covers a finite offline check of the exact Classic R3 Request/Grant and bootstrap boundary. It is preparation evidence only; it is not a native Classic run, a delegate execution, or an S1 result.

## Bound inputs

The test required and checked these immutable source-envelope and coordination-snapshot hashes before preparing its disposable case:

| Input | SHA-256 |
| --- | --- |
| Original R1 `released-native-grant.json` | `b917f5a5eb99f0a607fd282a81acdf7b085e6a1dba14f89c8d0c32f349c82c3e` |
| Original R1 coordination snapshot | `04484d4220dd7383af1d2675a95384554380dccc5a14ee8fc5908c3a474ede58` |
| R3-A1 `released-r3-a1-grant.json` | `b45a048923102feaa2040a769281cc2c258d66be09c2ce3561238b4b74c3f932` |
| R3-A1 coordination snapshot | `29595284a6e4202129d9f014aee6e1fec545d19bdfe028411687d41ec6b8fa17` |

The synthetic Request used dispatch `classic-native-contract-corrected-r3`, task `native-classic-positive`, the pinned Classic packet/source/binary/configuration, the retained R1 grant plus additive R3-A1 grant, and a temporary Coordinator grant with six role calls and a 180-second request window. Overseer amendment R3-A1 clarifies the existing authorization and adds no quota. All mutable outputs, including the SQLite ledger, were under the Python temporary directory and removed at test completion.

## Observed checks

Executed from the Markitect repository root:

```powershell
python -m unittest experiments.government-comparison.runtime.test_r3_classic_preflight -v
```

Package-cwd reproduction from `experiments/government-comparison`:

```powershell
$env:PYTHONPATH = "runtime"
C:/Python313/python.exe -m unittest runtime.test_r3_classic_preflight -v
```

Result: **1 test passed**. It exercised real `dispatch.Authority.validate`, `classic_integration.bind_request`, `native_controller.validate_native_fixture_grant`, Classic role preflight, `native_controller.begin_classic_controller`, and the resulting `load_context`. No mocks replaced those boundaries.

The passing path reached the controller bootstrap, recorded zero `controller_processes`, finalized as `incomplete` because no Classic captures existed, and reported no candidate commit or semantic acceptance. Separate re-authorized malformed temporary Requests rejected missing R3 binding and mixed R2/R3 correction before the ledger was created. A temporary role-authorization copy with a changed Classic native subject was rejected by the real Classic request binder before any ledger claim.

Fixture preparation invoked Git and the packet's existing Python preparation script to create a disposable, clean fixture repository. The test did **not** start the pinned Markitect executable, wrapper, deterministic delegate, Actor, model/provider, metadata session, or study cell. This checks offline binding and bootstrap preparation; it provides no product behavior, isolation, performance, or semantic-quality evidence.

## Source digests at the passing focused gate

| File | SHA-256 |
| --- | --- |
| `runtime/test_r3_classic_preflight.py` | `4b92852b2775af1a40a01ea1157d94aa62205b1088f32adf2e66a5884b6fc4a0` |
| `runtime/classic_integration.py` | `ace20174055c28a41d582430abe370bc4ccad4957eb0ca40183fed880a736fbd` |
| `runtime/native_controller.py` | `17facc87a5473a6b96ea544c36e24ee3312e484d41fb25301af9499be72a244b` |
| `runtime/native_fixture_budget.py` | `6b2c8ad5f92f1e5c6f50e20e25edaca4af5bda36e30002e043566fc3f0705889` |
| `runtime/government_roles.py` | `de8c60a1891c82b1bfedbea03c0e458102287e75eb1a91cb86d50997d40b8153` |

These source digests bind the passing R3-A1 focused-test checkpoint. The adjacent validator gate `runtime/test_native_fixture_budget.py` is SHA-256 `5c78b0315e29e31ce5fd4c1eca27f6f1487609d98c11ab435fa74177e59c1edc` and was reported by its owner as 9/9 passing; it is separate from the one-test Classic run recorded here.
