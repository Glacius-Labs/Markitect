# Scientist: authorized context session consumed; hold

Exactly one approved dispatcher invocation ran from clean delivery
`95cc60d5f936f732a39493a0b2e622b718b19679`, with execution source candidate
`efc78ef619b3f8bcafe9cffee0c2c7ebbc34b0a0`. All 34 frozen source pins and 15
old external draft/input/metadata hashes remain unchanged. No source edits,
new test wave, resume, retry, child, repair, extra access probe or study cell ran.

Overseer approval receipt SHA-256:
`f5b44019a85a4bdf69d139bd47699288bb28db8f10ebef597f5d7f93ca518b24`.
Grant `dbe31c7da231df51e1d0c554a225f1ad384b740742e0bd6a0822d75b00e16de2`,
Protocol `1f8b7688efb73fd2b880cb7ed2be4dffca05ca03dc1b1ec889e7ea9a6c95bbab`,
Request `da65b4b722c8c9582a9ca5b2cbd06a021ae1d055dab7a54f84aab4c051339138`.
Validity was 18:34:01–19:04:01 UTC on 2026-10-07. Preflight was read only;
the successor database, result and process evidence were absent before dispatch.
Original drafts remain preserved. Exact command and paths are in
[invocation](evidence/context-additional-live/run-1/invocation.json).

The Actor produced a completed turn and final JSON, with `released:null` and
`actorOwn:null`, reporting that no file-reading tool was available. Neither
expected sentinel matched. The raw stream records **zero tool calls and zero
file-read events**. This establishes no successful context access and does not
prove that OS access was impossible. Only the same two synthetic files were
requested; their exact bytes are retained in the evidence inputs. The raw stderr
contains a PowerShell shell-snapshot unsupported warning. Tool availability is
Actor-reported; no separate tool inventory or access probe was run.

The native process ran **11.379776 seconds**, returned **1**, and recorded
`retrospective_session_token_threshold`; the dispatcher returned **0** after
writing the Result. Result status is **stopped**. The completed answer and
`turn.completed` were already present when the threshold was observed; a
successful native process exit is not claimed.

Reported usage is **20,267 input + 234 output = 20,501 tokens**. The retrospective
10,000 threshold was exceeded by **10,501**. Cached input 9,856, reasoning output
139 and cache write input 0 are retained raw and not added again. There was one
started and one completed agent turn; these are not provider-request counts.
Provider requests, internal transport retries and resolved serving model remain
null. The original Result's `inferencePerformed:null` is preserved, alongside the
observed turn/answer/usage evidence. Requested model/reasoning remained exactly
`gpt-6.1-sol/high`, using the unchanged pinned Codex 0.160.1 runner/config.

The successor ledger records **one** completed reservation with status stopped:
three predecessor starts plus one additional start = **four consumed starts**.
No fifth start is authorized. Historical tokens remain null/null/10,009; with
the new 20,501, the known cumulative subtotal is **30,510**, while the full total
remains **null**. Old overall budget compliance remains unknown. Both historical
database hashes remain unchanged:

| Historical database | SHA-256 |
|---|---|
| actual-runner-authorized-starts.sqlite | `74dfd51e2802bc67131891b28c6858129ff5ef4a50a6f1fe06a8793a7250732f` |
| runner-01601-diagnostic.sqlite | `33bd2ba59b11ab6d54f75c55e6d12b00ec6b90df00106a1419aed239cef2982c` |

Successor database SHA-256:
`b4f13c5053ad1f00e477bf9be9831a5075f2ea0d8d33d8db6ce8a2173a6a0be8`.
Its raw copy and read-only table export preserve the predecessor binding,
approved profile history, exact reservation and terminal Result. No old database
was migrated. Shared limits and study profile definitions remain unchanged.

[Manifest](evidence/context-additional-live/run-1/manifest.json) binds exact copies
of approval documents, raw stdout/stderr/process receipts, Result, input
snapshots, successor SQLite, read-only export, preflight and observation.
[Observation](evidence/context-additional-live/run-1/observation.json) separates
answer, access, usage, process and cumulative accounting. Independent reading is
recorded separately in `context-additional-live-review.json`.

**S1 remains open: effective context access was not demonstrated. Hold for
Overseer.** No product-quality or OS-isolation conclusion; no fifth start,
continuation, repair, extra access probe, comparison cell or further study run.
The Worker retains the heavy test slot. This delivery adds evidence only.
