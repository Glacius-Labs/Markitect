# Independent R1 source preflight review

Reviewed the bounded R1 packet at source candidate `f24b7e097564d771d3ad438ab6b5f9f5321b6301`. The packet source files and focused-test log match the byte digests recorded in `offline-validation.json` at review time. This is a source and offline-evidence review only; I did not run tests or launch a process, Actor, app-server, or model.

I reviewed the complete active `commonRunnerReadToolGrant` and its current coordination section, the frozen profile/input bindings, collector, client, runner entrypoint, protocol validator, schema contract, and focused test evidence. The scope remains one fresh ephemeral thread and one turn with at most one ordinary native read of the single public probe file. The requested model/provider/cwd/approval and effective read-only permission report are gated before the turn; missing permission-profile provenance is retained as unknown and allowed only with the explicitly reported read-only sandbox. The code does not establish OS enforcement or provider-side request/billing behavior.

The initial collector review found gaps in terminal ordering, early turn-ID correlation, and first-observed command counting. The reviewed snapshot closes them: pending events retain arrival order; the start response must agree with an early turn ID; a `turn/start` response alone cannot establish completion; post-terminal item/tool evidence is rejected; and the sole command slot is reserved at first observation. Success also requires the actual correlated command-completion event with exit code zero and matching output, a completed turn, and the fixed final response. A terminal snapshot alone is insufficient.

The transport accepts only the frozen RPC sequence and allowlisted, schema-validated notifications, stops on server requests/errors/warnings or unknown/malformed events, bounds frames/queue/raw bytes/deadlines, and persists only the sanitized receipt. Reasoning and raw configuration/payloads are discarded. Interrupt is limited to one stop-only request for known owned active IDs. The local protocol validator checks the pinned archive and each loaded member hash, rejects unsupported schema constructs before use, and the focused follow-up includes a positive lifecycle test using the actual pinned schema validator rather than a permissive stub.

The offline record reports a corrected 26/26 focused run and a 9/9 targeted follow-up (31 distinct final cases). The first run was 25/26 because one test expected a later pending-event mismatch reason than the collector correctly returned; the assertion was corrected and the suite rerun. These are synthetic contract checks, not an Actor capability result.

Reviewed source and test hashes:

| File | SHA-256 |
|---|---|
| `client.py` | `798a9443e80515f7675b20b26680ed3790ab68070f2663d42abb960cb25f073e` |
| `collector.py` | `ee1f73d76de17afcf5a5ea892f820446a110b5593356b6b09d6aa1cf0e78a3a3` |
| `protocol.py` | `ebd087657febf63846904d83a75bc226a5412458418d20abe024ed267febb822` |
| `run_once.py` | `16ba63ee83835dc579e0adaa77500e4f2cc9f86ec06cf66a33f01b40fbbbbedf` |
| `test_client.py` | `a88b7881789e87f3fa69ab804260688bb6702a7165737306bca1a2bc81f838e8` |
| `profile.json` | `6a1d9c043758563d59b576c0a2c526e375f6bc819002768a535333b7cb1a128c` |
| `schema-contract.json` | `f13df7b695435aaff3939be94acadcef510be00abd3b14c70d0b04764dc0ea35` |
| `focused-tests.log` | `dab26f9e38503115cf3308ceba4e1dbeb82005de114c9f2684f9d93ca1b92f96` |

Disposition: no remaining material blocker in the reviewed source and offline test scope. This clears only the source preflight; the exact immutable Request/Freeze bindings still require their separate review before any reservation or start. The six study cells remain unrun, historical aggregate token usage remains unknown, and no human-quality or general S1 claim follows from this review.
