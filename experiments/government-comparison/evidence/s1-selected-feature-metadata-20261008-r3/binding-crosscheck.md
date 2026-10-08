# R3 selected-feature metadata exact-binding cross-check

Date: 2026-10-08
Scope: independent offline binding review only. No reservation was written and no worker, controller, app-server, Actor, test, or product process was started. This review creates no authorization and does not authorize launch.

## Frozen identities

The package is `experiments/government-comparison/evidence/s1-selected-feature-metadata-20261008-r3`. Inputs were read as bytes and parsed with Python `json.loads` (a duplicate-key-rejecting hook was used for the grant/coordinator comparison); JSON objects were compared canonically with type-preserving serialization. No PowerShell JSON object coercion was used.

| Item | SHA-256 / identity | Check |
| --- | --- | --- |
| Source commit | `78e90f3170a5409762645497e3bb93b76392f7f6` | Request and freeze agree |
| `request.json` | `542ec68d319549cd21b50e81f8f13105670a36c0fed7e182a768e2ab21068f68` | Matches freeze's request binding |
| `freeze.json` | `01dd523fc0ada72be555f63dcc8a566333dea6dbd5d4b2709bd815fe39e0bd17` | Freeze source and request bindings match |
| `authorization-grant.json` | `d2d49ccda7f07c9f60fa641213373aa5b28b8cf7006c37c2df47463cc2c68c48` | Matches request's grant binding |
| Live coordination state | `3f57d6b2aa4f8ecfac9314ad97be0bafa76bbf556e7478880b28eaa4839e4043` | Matches grant's source-coordination binding |
| `profile.json` | `daf3f2b775e73cb194f48f8731e73f82a93ca0b7f2976b980ee7ed45110fb416` | Matches request's profile binding |

The complete local `authorization-grant.json` `grant` object is equal as decoded JSON data (canonical JSON encodings also match) to the live `threads[name=Scientist].evidence.selectedFeatureMetadataR3Grant`: all 69 fields match, including `sentPrompt`. Its UTF-8 SHA-256 is `9227dd133d770767b1a54ba1aa33473e575a62f324be0ff2c6f6877b3e689f13`. The local `slot` object also exactly matches the live `fullSuiteSlot`. There is no grant or slot mismatch; the previously reported sent-prompt difference was a false positive.

## Source and freeze inputs

All 21 entries in `request.sourceFiles` were independently checked three ways: request-pinned SHA-256, current worktree file SHA-256, and SHA-256 of the bytes read from the pinned Git commit. All 21 matched; there were zero missing files or mismatches.

All 30 absolute paths in `freeze.files` existed and their current bytes matched their frozen SHA-256 values; there were zero mismatches. The request and freeze both bind source commit `78e90f3170a5409762645497e3bb93b76392f7f6`. The freeze records one new tree, four prior policy trees, cumulative maximum five, external reservation required, and immutable request/freeze plus exact-binding review prerequisites.

## Profile, workspace, and external evidence

The current profile's 38-token argv is exactly equal to the prior `s1-common-runner-policy-metadata-20261008/profile.json` argv; it retains all 17 `--config` pairs unchanged. The four RPC envelopes match the prior profile except for the intended new `initialize.params.clientInfo.name`. The bounded method sequence is `initialize`, `initialized`, `config/read`, `configRequirements/read`; no other request is present.

The profile cwd and grant cwd are the same actor directory. Its declared `README.txt` SHA-256 (`85c25997e1ee463e2d78fab8443110b3ee5066217edce20f0358fc9c64a3a8ad`) equals both the grant's public cwd inventory and the freeze inventory; the file is covered by the 30-input freeze. The external evidence directory exists and is empty: there is no `reservation.json`, controller receipt, worker receipt, or sanitized result.

## Loader and start-order review

The statically inspected loader is `client.py` (worktree SHA-256 `29d2c97a7c34ff3c580789a777967051221339bd6da3a39b492f729271b7b223`); the one-shot outer runner is `run_once.py` (SHA-256 `ccbc7c080d01fb536792c162a69cf6577959cb525de8b5a4154cfc2539869b19`). The source-to-Git/worktree hashes above include these files.

- `client.load_request` checks the grant against live coordination state and slot, reservation binding, clean freeze commit, request/freeze/profile/grant hashes, interpreter and schema pins, required source/input sets, historical ledger pins, all 30 freeze paths, and all 21 Git/worktree source pins before returning a binding (`client.py:201-250)). Required pinned inputs include `client.py`, `run_once.py`, profile/request/freeze/grant files, frozen method enums, test/review/offline-validation source records, `runtime/selected_feature_contract.py`, the preserved classified client, `runtime/process.py`, protocol schema archive, public plan and source-evidence records, cwd README, Python interpreter, and accepted executable. This is the declared validation/load boundary, not proof that these values are effective product policy.
- The outer runner verifies a clean tree and freeze/source relation, then exclusively writes the reservation with request/freeze hashes and limits before calling the controller (`run_once.py:19-30)). The current external directory is empty, so this reservation step has not occurred.
- The controller validates the reservation before spawning a gated worker. It assigns that worker to the Windows kill-on-close Job, writes the bound job receipt, and only then sends `GO\n` (`client.py:401-412`). The worker blocks on `GO\n`, reloads/validates the request and job receipt, then loads the pinned prior client and method enums before its app-server launch path (`client.py:258-268,337-346)). This ordering is visible in source; no process was started to exercise it.

## Finding

The exact live-grant and slot binding, 21 source Git/worktree hashes, 30 frozen inputs, profile/RPC comparison, cwd inventory, empty external evidence state, and source start-order checks all pass. No exact-binding blocker was found. This is a bounded offline review only: it does not constitute a reservation, completed prelaunch gate, process-start evidence, a successful metadata observation, or permission to start the operation.
