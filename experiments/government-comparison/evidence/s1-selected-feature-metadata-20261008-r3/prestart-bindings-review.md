# Independent R3 prestart binding review

**Disposition: binding checks pass for this exact Request/Freeze candidate; no reservation or start has occurred.** The reviewed source commit is `78e90f3170a5409762645497e3bb93b76392f7f6`. Request SHA-256: `542ec68d319549cd21b50e81f8f13105670a36c0fed7e182a768e2ab21068f68`. Freeze SHA-256: `01dd523fc0ada72be555f63dcc8a566333dea6dbd5d4b2709bd815fe39e0bd17`.

Independent byte checks confirmed all 21 request source pins against the current files and Git blobs at the source commit, and all 30 freeze input hashes against current bytes. The request binds the R3 profile and authorization grant, the freeze binds the request, and both identify the same source commit. The profile matches the prior route: the same 17 configuration pairs, exact four-RPC sequence, and public cwd. The cwd contains only the expected README at its pinned hash.

The authorization snapshot’s recorded coordination SHA-256 (`3f57d6b2aa4f8ecfac9314ad97be0bafa76bbf556e7478880b28eaa4839e4043`) matches the current coordination file. Direct Python 3.13 byte-based JSON reads show the live R3 grant equals the authorization snapshot on every field, including `sentPrompt`; the live grant status and assigned Scientist slot match the exact expected values. Calling the pure `validate_live_authority()` helper on those parsed objects succeeds. The external evidence directory is empty and has no reservation.

Review correction: an earlier comparison based on PowerShell-rendered strings appeared to show a `sentPrompt` mismatch. That comparison was invalid; direct byte-based JSON parsing shows no differing grant fields, and the client’s own validator accepts the live grant. The earlier blocker conclusion is withdrawn.

The clean final freeze commit and exclusive reservation remain required before launch. This review does not run `load_request()` or the one-shot entrypoint, create a reservation, start a controller/worker/app-server, send an RPC, or write a ledger. The operation remains unexecuted. This binding review establishes no actual metadata observation, enforcement, active permission, tool capability, serving identity, historical cause, or S1 readiness.
