# Government scope identity offline handoff — 2026-10-08

Package: `government-scope-identity-offline-20261008`. Scope: offline-only Scientist adapter correction, from clean base `1ea3fc970797e01206e98d15e1951021b4ee5586`. Reviewed source commit: `bb64f4999f826ca2d07e9cad7b576d905ff58755`. Final artifact commit is reported separately to Overseer.

## Result and exact change

`runtime/government.py` decodes native ActorRecord scopes as JSON arrays in the fixed order apiVersion, kind, namespace, name and compares all four values with each planned integration-review identity. There is no namespace/name fallback, trimming, case folding or lossy joining. Malformed shapes/types and identity syntax are rejected; an empty namespace is valid under the pinned core contract. JSON formatting and escaping do not alter identity.

The directly neighboring cabinet duplicate key, vote membership key and cabinet lookup had the same shortened-identity issue. These now use the same complete tuple. Display receipt labels remain unchanged. Result and Resume both call this validator; no additional Resume identity conversion exists. Candidate/evidence/round, exact receipt/role/input, root review, prior mandate/digest, explicit passing observations, complete decision vote set and every final positive Ressort vote remain required.

Product source remains `04e225d5caee78c2a198607143863fca1e829750`, binary SHA256 `12241f325e4af59451e4021b31d9e6f5b829b5de06c94d35eaabfb9d663aa51f`. No host, binary, P1 source, global setting, installed release or canonical Overseer document changed.

## Evidence and independent review

- `source-diff.patch` is the exact base-to-source diff, including corrected synthetic fixture and 13 new regression tests.
- Six complete Git-bound native source snapshots in `pinned-contract/` establish DefinitionIdentity/Key, valid identity syntax, integration-review plan and ActorRecord scope construction. `contract-bindings.json` records source SHA and bytes.
- `native-contract-example.json` is a read-only extraction of old R4 identity shapes, bound to raw report SHA256 `448e2c5d48ec703e0a2ac3a008ee8e3f3e5508991e0d32f05d1b8dc92e8e8fd2`. The old report was never run through the new validator or rescored.
- `focused-tests-2.log`: 29 pure Python tests passed (16 existing translation tests plus 13 identity regressions). Tested native representation, all four mismatched fields, incomplete/malformed identities, invalid syntax repeated on both sides, valid empty namespace/Unicode escaping, missing review coverage, vote mismatch, unchanged positive-vote/receipt/decision requirements and both Queue/Resume translation. The complete run mocked `subprocess.Popen` to raise on any subprocess attempt.
- Independent contract audit identified the exact four comparison sites and empty-namespace rule. `independent-review.md` binds final source hashes and records closure of the syntax-validation finding. Initial 28-test passing run is retained separately as pre-review evidence.
- `historical-bindings.json` verifies all 3282 prior evidence/pin/handoff files byte-for-byte against base Git blobs. The current native ledger SHA remains `dd617d58a9021fce0b11482b740d78ccb5ddd143ff0c6fa4fbe08983af7f6705`.

## Reproduce only the focused offline test check

From `C:/Users/Consiliari/.codex/worktrees/government-scientist/Markitect/experiments/government-comparison`, use `C:/Python313/python.exe -B` with the following in-process test body. The original test stdout/stderr is retained in the log; do not overwrite that sealed receipt.

```python
import sys, unittest
from pathlib import Path
from unittest.mock import patch
sys.path.insert(0, str(Path("runtime").resolve()))
suite = unittest.defaultTestLoader.loadTestsFromNames([
    "test_government", "test_government_scope_identity"])
with patch("subprocess.Popen", side_effect=AssertionError("offline test attempted subprocess")):
    result = unittest.TextTestRunner(verbosity=2).run(suite)
raise SystemExit(not result.wasSuccessful())
```

`git diff --check` passed. A read-only preparation lookup of nonexistent decision.go failed before source edits; actual bindings.go replaced it. An initial subdirectory-relative historical-file query returned zero; inspection caught it and repository-root Git/blob validation established the full3282files. These local preparation corrections are recorded in offline-checks.json. No experimental process was launched by either correction.

## Limits and stop

New experimental consumption is zero for native/controller starts, wrappers, delegates, real Actors, model/provider calls, metadata sessions, study cells, reserved runtime and full product suites. Historical cumulative values remain 12 native starts / 10 wrappers / 7 deterministic delegates / 1800 reserved seconds; real history remains 5 Actor starts, 53331 known reported tokens, actual total usage unknown/null.

R4 stays closed and incomplete, with no Resume/retry/requalification. Its residual quota has expired and its slot is released. Old freezes bind their original source; this later authorized source correction does not make an old freeze executable again. S1 stays open pending new finite authority, actual end-to-end evidence and real Actor/tool capability. No semantic or human acceptance follows from these unit tests. No further actual execution is authorized by this package.
