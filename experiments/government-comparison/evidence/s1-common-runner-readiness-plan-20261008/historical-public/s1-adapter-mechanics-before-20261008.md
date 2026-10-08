# S1 adapter mechanics and remaining readiness gaps

This work implements pure Request binding, queue/resume argv planning, and
Result/receipt translation against the public v1.2 contract and the accepted
Government G5 handoff at source `04e225d5caee78c2a198607143863fca1e829750`.
The acceptance provenance is Overseer thread
`01a11367-a781-7683-a20f-46e12614dcb4`, recorded separately from the handoff's
stale pending-Overseer status. The exact handoff and binary digests, accepted
source identity, acceptance provenance, and prior provisional pin are recorded
in `runtime/government-pin.json`. This does not enable dispatch or establish
product quality or autonomy.

`runtime/government.py` treats `Request.product.government` as the explicit
product-specific pin/input envelope. It binds the handoff and executable
digests; binds the Government YAML and one Order to paths and bytes inside the
exact Actor repository; binds runtime and backlog files as external
`releasedInputs`; and requires the backlog to name only the current task. It
checks backlog actor-start, repair, wall-time and parallelism bounds against
the common task limits. The configured runner slots are listed for review.
Queue and resume plans use the documented exact argv. Plans remain
`dispatchable: false`.

The mapping is an adapter-local interpretation of the existing `product` field;
it does not change the public Request schema:

```json
"product": {
  "government": {
    "handoff": {"path": "ABS_HANDOFF_JSON", "sha256": "..."},
    "executable": {"path": "ABS_MARKITECT", "sha256": "...", "sourceCommit": "FULL_SOURCE_COMMIT"},
    "projectConfig": {"path": "government.yaml", "sha256": "..."},
    "order": {"path": "orders/task.yaml", "sha256": "..."},
    "runtime": {"path": "ABS_RUNTIME_JSON", "sha256": "..."},
    "backlog": {"path": "ABS_BACKLOG_JSON", "sha256": "..."},
    "queueStateDirectory": "ABS_QUEUE_STATE_ROOT"
  }
}
```

The handoff, runtime and backlog bindings must also appear in `releasedInputs`;
the project config and Order are bound within `actorRepository`. This helper
does not itself prove Git cleanliness or verify `baseCommit`; those remain
common harness preconditions.

For a recorded queue result, the translator receives the exact Request file
bytes, verifies they decode to the Request object, and computes `requestSha256`
from those bytes. It verifies the G5 queue API,
queue/backlog identity, the stdout record against its immutable numbered queue
report, every run-report path and digest under the exact runtime `stateDirectory`
run-id directory (the native run directory is a sibling of the queue directory),
and the event-log receipt. It checks
actor Response/Receipt identity against configured role slots; checks votes
against the selected cabinet, prior mandate, candidate, evidence and round;
and checks the decision's vote set before recording role/vote/promotion receipt
kinds. Product-local accepted states remain attached to their raw report;
`candidateCommit` stays null. `completed` means only that the queue process
completed with native status `complete` and no in-flight Actor. Exit zero with
native `blocked` stays `blocked`; other nonterminal states remain `incomplete`.
Provider requests, provider turns and inference are null. Token totals are
reported only when both native input/output counters are explicitly known;
cached tokens are not added.

| Capability | Current adapter mechanics | Remaining S1 evidence or blocker |
|---|---|---|
| Government source and binary | The accepted G5 source, handoff SHA-256 and binary SHA-256 are pinned; `bind_request` rejects a Request that substitutes a different source, handoff, or executable. The earlier `ce021ce…` candidate and its handoff/binary digests remain recorded as superseded provisional history. | Use the accepted pin for preparation. Native dispatch still requires the remaining S1 setup and common-ledger gates. |
| Request inputs | The helper binds executable, handoff, project config, Order, runtime, backlog, queue state and digests. It accepts one task with no dependency jobs and checks native actor, repair, wall and parallel limits against common task limits. | Freeze an accepted product-specific Request mapping and actual project inputs at the later S1 setup checkpoint. |
| Queue and resume | Review plans emit exactly `markitect government --repo ABS --action queue --backlog ABS --write` and `markitect government --repo ABS --action resume --backlog ABS --queue ABS --write`; no command is started. | Accepted pin, operator authorization, exact setup smoke and bounded process/receipt storage. |
| Native result and roles | Queue stdout, immutable queue report, run reports, configured actor slots, votes, decision, promotion and event log can be translated into digest-bound v1.2 receipts. Run reports and promotion files are checked beneath the exact native `stateDirectory/runId`. | These schemas are observed at accepted G5 `v1alpha1`. Independent task/candidate acceptance remains outside this translator. |
| Per-role wrapper and common trial ledger | G5 `agentexec` documents a versioned JSON Invocation on stdin, response identity echoes, configured wrapper commands, and `runtimeFiles` digests. Importable `runtime/government_roles.py` validates that wire shape, the exact outer Request through an already validated `dispatch.Authority`, operator authorization, configured role/phase and delegate file pins. In one ledger transaction it checks the bound Coordinator grant ceiling and cumulative reported tokens for the current dispatch (its outer booking plus its nested role attempts) against the grant's retrospective threshold, then reserves one unique invocation before a bounded delegate call; the timeout is capped by remaining outer-request, grant, role-authorization, trial and task deadlines. It returns response bytes unchanged and preserves unknown request/turn usage as unknown. | This is a library helper only: its CLI entrypoint is deliberately disabled. Dynamic Request/Grant/Protocol paths and digests cannot be pinned in `RunnerSpec` without creating a runtime/outer-Request/grant digest cycle. The native configuration must stay limited to static script and role-authorization pins; an acyclic outer Authority bootstrap, native queue/top-level wiring, and integration test remain open. The helper treats `inputDigest` as the native opaque `sha256:` identity and requires exact response echo; it does not reimplement Go Request canonicalization. G5 does not supply provider request/turn counts; a provider may report no usage, and an individual call may overshoot this per-dispatch retrospective token threshold before the next call is blocked. Common ledger trial/task token caps remain cumulative across dispatches. The current outer dispatch reservation is also recorded as an Actor attempt, so native integration must resolve that orchestrator/Actor accounting mismatch before claiming role-session totals. |
| Resume/recovery | The planner binds resume to the exact queue directory and backlog; the translator rejects a different queue identity and retains queue/run/event receipts. An invocation identity cannot be launched twice after its ledger reservation. | Native queue interruption/recovery has not been joined to the shared ledger or exercised end-to-end; an interrupted role reservation remains consumed and blocks replay. |
| Classic | Existing `runtime/classic.py` owns documented native argv construction and bounded subprocess capture; `runtime/classic-pin.json` holds the study candidate `c91363b7ac4decbe87212ff0f588b5451581a152` / v0.14.1. The new stdin-JSON wrapper boundary is provider-neutral and could serve a configured Classic role if its runtime supplies the same pinned contract. | Classic role middleware is not wired or integration-tested. `dispatch.py` still returns `readiness_gap` for Classic stateful dispatch. The held pin is unchanged; do not replace it with later local `8927704`. |

The importable wrapper's focused tests construct a real `dispatch.Authority` and
real temporary Ledger from synthetic grant/protocol files, then use a synthetic
delegate only. They cover source-shaped
Invocation digests, slot/phase authorization, pre-effect reservation, duplicate
replay rejection, unchanged response forwarding, and provider-reported token
accounting. They do not establish native queue wiring, provider behavior,
independent quality, or human acceptance. The wrapper CLI remains disabled
because an acyclic Authority bootstrap does not yet exist. The current outer
dispatch reservation consumes one Actor attempt before nested roles; native
controller wiring must reconcile that conservative placeholder booking before
reporting nested Actor totals. `dispatch.py` keeps both native stateful arms
closed pending that bootstrap and an integration proof for the actual native
runner.

Focused deterministic tests cover accepted-pin binding, request/input binding,
queue and resume argv, role/vote/promotion receipt mapping, status handling
despite exit zero, unknown usage, path/digest rejection, and the synthetic
role-wrapper boundary. They use temporary JSON and files; they start no product
process, native Actor, study cell, or provider. The adapter remains gated before
native queue dispatch until the remaining S1 inputs and end-to-end integration
are validated.
