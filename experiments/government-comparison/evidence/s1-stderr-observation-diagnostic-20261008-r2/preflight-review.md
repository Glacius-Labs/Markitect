# Independent source preflight review

**Qualified verdict: the reviewed source implements the authorized passive-stderr change without relaxing the existing RPC/thread gates.** I compared it with the prior redacted-stderr client and independently checked the fresh grant against the canonical grant and active slot. The full grant semantics match, including key `s1-stderr-observation-diagnostic-20261008-r2`, issue time `2026-10-08T17:31:23Z`, status, and base. The slot owner/key/status/issue time and assignment time match. The allowance advances the historical app-server count from eight to a maximum of nine; prior Actor and CLI metadata counts remain six and eight.

The profile retains the existing 17 configuration pairs and five ordered RPC methods, exact read-only/never/ephemeral thread payload, and public input identities. It binds the same pinned candidate and schema archive, the unchanged 12-member schema contract, the accepted redaction collector, and the fresh external output root. The accepted thread gate, schema contract, and method-enum data are byte-identical to the prior packet. The only intended behavior change in the source diff is separating passive capture from the fatal stderr ceiling, alongside fresh authority, identity, history, and preparation-window bindings. The previously identified eager native-poll change has been removed; the ordinary EOF/native-exit, deadline, cleanup, controller, Windows Job, and gated-worker lifecycle remains as before.

The capture event is now separate from the fatal admission event. Any nonempty stderr read still triggers bounded capture, but does not itself reject the next allowed RPC or stop response consumption. The stderr pump limits each read to the remaining 16,384-byte total allowance. Reaching that ceiling is terminal, including exact equality; exceeding it through a direct feed is also terminal. Only that fatal event blocks RPC admission and terminates request waiting. The unchanged collector is fed bounded fragments through its cleanup-compatible path, while its selected-input, four-line, redaction, output-size, and private-write constraints remain bound by the grant, profile, and diagnostic contract. No raw output or excerpt fingerprint is added to receipts.

Other stop conditions remain active: strict metadata/configuration, authorization, protocol/schema, notification, and thread-response checks; the five-method allowlist; native end, resource, and deadline handling; and terminal completion after a valid thread response before any turn. The controller still validates the source/request/freeze/reservation binding before launching its worker, after Job assignment before `GO`, and the worker rechecks before launching the app-server. No private excerpt or transcript was read.

This is a source review only. I did not run tests, the client, the candidate, or any launcher. The focused binding/consumer tests remain the test agent's scope and are not claimed here; exact request/freeze binding review remains pending. No source blocker remains after removal of the eager native-poll check.

Reviewed current source pins:

| File | SHA-256 |
|---|---|
| `client.py` | `70fd65b3c0ced08dd638676fe52dad9ff1f3fd2a56399e84605643245950f3b5` |
| `run_once.py` | `a2f53477c0eca6f394568270ce536afdc213ba484f187f9301b257fc98f9ecd0` |
| `profile.json` | `b8ccffb43d7248c8082ad23abad7cb71c8bb03d21bf36fe7d296dea3a7dbe72c` |
| `authorization-grant.json` | `cf988b417c3ac6c0bc227c28f4a9306dd555627a9375e424c3f2a7cb5ed729da` |
| `diagnostic-contract.json` | `c16a7be1096bba49acb97e02b5b5d0d5ad0d9753bcc578505c2e8b7dbb8deb27` |
