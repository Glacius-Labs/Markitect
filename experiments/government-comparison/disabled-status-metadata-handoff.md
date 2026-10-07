# Disabled-status metadata session: completed bounded read

The single allocation `local-policy-read-disabled-status-20261008` is consumed.
The separately frozen [preflight](public/policy-disabled-status-preflight.md)
remains unchanged; the [post-run review](public/policy-disabled-status-postrun.md)
is a different file. The earlier preflight archive loss is still unresolved and
no replacement bytes or historical Remote Control status were reconstructed.

The [request](evidence/policy-read-disabled-status/run-1/request.json),
[freeze](evidence/policy-read-disabled-status/run-1/freeze.json),
[reservation](evidence/policy-read-disabled-status/run-1/reservation.json), and
both exclusive one-start markers bind the exact approved code, schema, grant,
Python runtime, pinned Codex 0.160.1 executable, original cwd and fifteen inline
config pairs. The client passed 22 pure regression tests before reservation.
The four outbound methods were each sent once: `initialize`, `initialized`,
`config/read`, and `configRequirements/read`. No other RPC or retry was added.

The [sanitized result](evidence/policy-read-disabled-status/run-1/sanitized-result.json)
records one `remoteControl/status/changed` notification with the schema-valid
status `disabled`. Only its method, class and status were retained. Identities
and the remaining parameters were discarded. Both read responses were obtained,
with no warning and `provisionalConfig=false`. Client and native exit codes were
zero; native elapsed time was 0.103207800 seconds and the bounded outer tree
0.456172800 seconds. 49,985 raw bytes were consumed then discarded; native and
outer stderr were empty. There was no Remote Control enable/connect operation.

The returned configuration reports `approval_policy=never`, null `sandbox_mode`,
and `windows.sandbox=elevated`. SessionFlags supplied the requested
`shell_tool=true` and `unified_exec=true` and the disabled app/goal/hook/memory/
multi-agent/plugin features. The returned user layer supplied the Windows
sandbox setting. The filtered requirements response retained `network=null`.
These are observations on this app-server route. Null does not establish an
effective default or absence of policy. The route includes user configuration
and omits the historical exec-only isolation/ignore flags; it proves neither
historical exec policy, the cause of rejected CreateProcess requests, nor actual
file/tool access. It justifies no permission change or sixth Actor start.

Three metadata process trees and six earlier CLI metadata calls are now
consumed. Five historical actual Actor starts, the known input-plus-output
subtotal 53,331, and the unknown historical token total remain unchanged.
Native deterministic integration fixtures use separate allocations and ledgers;
they cannot replace that history or count as study evidence.

The final independent [delivery validation](evidence/policy-read-disabled-status/run-1/delivery-validation.json)
has SHA-256 `174f0776b7590da9bc62a0b2c41aeddbec6dae1196f65b2e99284fd81541430b`.
It confirms all fifteen frozen inputs, the original reservation and both
one-start markers, and all four historical ledger hashes. The four prior
adapter-source entries were verified against accepted Git blobs at `0e62d5f`,
because the separately authorized native integration changes those sources.
