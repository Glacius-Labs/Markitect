# Prospective stderr-observation diagnostic

Fresh R2 authority starts from `5f66346d6c602d573777fcd1522d006a6a6faefb`.
The R1 construction failure and closure remain unchanged. Its ineffective,
whitespace-sensitive no-op replacement assertion was removed; source construction
now checks only meaningful changes. Editable preparation starts at
2026-10-08T17:32:43Z and ends by 17:52:43Z, with at most two ordinary offline
repair iterations and three affected-test invocations. The initial six cases
run once; any later invocation is limited to failed affected cases. Failures and
corrections remain in preparation-history.json.

The sole prospective behavior change is passive, bounded stderr observation.
Nonempty stderr signals the capture event and collects the accepted bounded
redacted excerpt. That event does not close RPC admission or stop consumption.
The unchanged collector receives fragments through the wrapper's cleanup-enabled
feed; its known-pattern redaction, four complete physical lines, 1024 selected
input bytes and 2048 private-output-byte caps remain unchanged. No message text
allowlist or stderr severity interpretation is added.

A separate fatal event closes admission when the 16384-byte observed-stderr
ceiling is reached. Native stderr reads are limited to remaining allowance, so
the actual reader cannot consume more than that ceiling. Raw data stays in
bounded memory and is discarded. RPC errors, schema/config/auth failures,
unlisted notifications, server requests, native end, deadlines and other resource
limits remain terminal. The exact five client methods and the existing thread
identity/read-only/ephemeral/empty-turn gate remain unchanged. A validated
thread/start response ends the attempt before any turn.

Before durable reservation, the freshly committed request/freeze bind all source
inputs, the complete original grant, exact current slot, candidate, schema,
interpreter, public two-file cwd and five read-only historical ledgers. Controller
and worker recheck bindings; Windows Job assignment precedes GO and candidate
launch. The new exact external output root must be absent and is exclusively
created. Once reserved, any failure is terminal; there is one candidate tree and
no retry, fallback, extra RPC, Actor task, turn, tool, metadata query or study cell.

The accepted private writer runs at most once after native exit and all pumps
are quiescent within cleanup time. Only written-private permits the later exact,
owned, nonlink, bounded-JSON read. The actual excerpt, raw text and its content
hash never enter Git, archive or callback. Unknown-secret forms and sanitized
task-history retention remain possible. Cleanup removes only the owned file.

Historical usage remains eight native trees, six Actor reservations including
five historical model attempts, eight CLI metadata calls, 53331 known tokens
with total unknown and native usage 15/16/13/2250. The new grant permits at most
one separate diagnostic reservation/tree, taking actual native history to nine.
Thread response evidence alone cannot establish tool capability, general S1,
serving model, billing stop, source equivalence or a historical failure cause.
