# Common measurement profile v2

Current diagnostic follow-up: Overseer explicitly authorized a separate one-start
resource allocation despite unknown historical total tokens. See
[context-additional-allocation.md](context-additional-allocation.md). This replaces
the in-place diagnostic migration prerequisite described below; the v2 study
profile and its unknown-token rule remain unchanged. Earlier frozen drafts stay
unapproved historical evidence.

Overseer adopted this common measurement correction on 2026-10-07 at 19:31
Europe/Berlin. This changes admission semantics for all three arms equally;
it grants no Actor session and clears no S1 readiness gate. The canonical files
are [v1](measurement-profile-v1.json) and [v2](measurement-profile-v2.json).
The bytes of [resource-proposal.json](resource-proposal.json), including every
numeric common limit, are unchanged.

| Observation/control | v1 | v2 |
|---|---|---|
| Finished attempt with unknown tokens | Blocks later admission | Blocks later admission |
| Known tokens, unknown provider requests | Blocks later admission | Permits admission; request counter remains null and a gap |
| Provider-request 80/480 limits | Known-count retrospective admission gate | Explicitly unsupported and not enforced |
| Tokens | Retrospective; overshoot possible | Retrospective; overshoot possible |
| Internal transport retry limit 1 | Unobservable | Unobservable, unsupported and not enforced |
| Wrapper retries | Zero | Zero |
| Provider transport defaults | No permission from the profile alone | Unchanged pinned defaults allowed within finite process wall time |

Agent-turn events never stand in for provider requests. Cache/reasoning counters
are retained raw; only reported input plus output is added once. No exact hard
provider-request or token cap is claimed. Every harness-launched Actor, reviewer,
setup or child has a separate reservation. Unobserved native descendants remain
an explicit gap. Time, session, parallel, human, token and repair numeric limits
remain owned by the unchanged common proposal.

## Append-only adoption and authority

`Ledger.adopt_profile` supports one explicit v1-to-v2 transition, recording the
old/new profile digests, unchanged limits digest, trial identity, exact absolute
ledger path, decision reference, approval digest and migration-source digest.
Active or ambiguous attempts prevent adoption. Original starts, tasks, attempts,
stops, human entries, counters, receipts and dispatch authority are preserved.
There is no identity change, reset, refill or token-unknown escape.

A new Grant on an existing ledger needs an explicit cumulative successor
allocation: `priorAuthoritySha256`, `priorActorSessions` and positive
`additionalActorSessions`, with `maxActorSessions` equal to the existing count
plus the additional allocation. The original authority row stays intact; the
successor authority/profile/ceiling is appended to a history. Each dispatch binds
its original authority and measurement profile. Its stored Result and original
binding are verified on resume. Recovery-only ledger handles cannot reserve.

The dispatcher commits profile adoption and successor authority together: an
invalid successor rolls both back, preserving all previous rows and counters.
Unknown historical tokens still prevent a new reservation after migration.
The separate future one-session diagnostic allocation must retain the exhausted
older diagnostic accounting; it is not an implicit budget refill.

## One proposed context session

`runtime/context_drafts.py` prepares concrete Request, Protocol, Grant and ledger
adoption **drafts** from clean committed sources. It performs only authenticated
`account/read(refreshToken=false)` and `model/list` metadata RPCs via the existing
pinned Codex 0.160.1 binary. No thread, turn, login or inference RPC is allowed.
Catalog advertisement, which may be cached, does not establish serving identity.

The proposal is exactly `gpt-6.1-sol/high`, 180 seconds, parallelism one, one wrapper
turn, retrospective 10,000 tokens, zero children/retries/continuations/repairs and
no new purchases. Its Actor may read only two explicitly named synthetic files:
one released context file and one Actor-owned file. The prompt is the launch input;
the task card and input bytes are separately hash-bound. There is no separate
access, write or process probe. Requested read-only sandbox and tool restrictions
are cooperative and not a proven filesystem boundary. All existing live gaps
remain explicit. A Coordinator must approve the final exact allocation, validity
interval and ledger adoption, freeze the protocol and rebind affected hashes.
The draft gate rejects launch before that action.

The three historical diagnostic starts remain consumed. Their known token subtotal
is 10,009; two attempts have unknown tokens, so the total remains null. Provider
requests and internal retries remain null. Both historical ledgers are read only
and hash-checked before and after verification/preparation.

The concrete drafts name the existing selected-runner diagnostic ledger and
authority identity. Their absolute proposed session ceiling is four (three
consumed plus one additional), with one additional session and parallelism one.
They explicitly remain non-executable: the legacy diagnostic schemas are not the
common dispatcher schema, and migration of both ledgers is unresolved. No empty
replacement ledger or new trial identity is created. Unknown historical tokens
remain blocking. Coordinator must resolve this cumulative binding before a later
live grant; changing statuses alone still fails an explicit binding gate.
Missing or partial ready bindings also fail. The gate cross-checks the existing
trial/path, both historical ledger hashes, consumed count, absolute ceiling,
additional allocation, approved migration receipt and actual migrated attempt
rows, retaining both null token observations. This package creates no such
migration receipt and performs no migration of the historical diagnostic files.

Run the bounded local verifier with a fresh external destination:

```powershell
python experiments/government-comparison/runtime/verify_profile.py --evidence ABSOLUTE_NEW_DIRECTORY
```

It runs only profile, dispatcher and runtime risk tests with deterministic Python
fixtures, retaining raw test output and dispatch records. It launches no model,
native product binary, study cell or metadata session and runs no full Go suite.
Classic's native gaps remain; Government G2 is an intermediate public interface,
not a final StudyGovernmentVersion. Hold after the frozen draft handoff.
