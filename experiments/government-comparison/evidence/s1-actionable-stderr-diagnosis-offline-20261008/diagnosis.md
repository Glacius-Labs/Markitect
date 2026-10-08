# Offline explanation of the terminal stderr observation

The first sanitized physical line reported that creating a shell-state snapshot
could not use PowerShell because that shell is not supported by the snapshot
feature. Its severity was WARN. This is a reported implementation limitation of
that feature, not evidence of failed authentication, denied shell execution,
provider rejection or a fatal native error. This paraphrase uses the single
previously authorized sanitized read already present in Scientist's task context;
the removed excerpt was not reread or reconstructed.

The preserved client source and terminal metadata establish this partial order:

1. The metadata exchanges passed their positive gates.
2. The thread-start request was admitted while the shared stderr stop event was
   still clear. Its write completed. An already admitted write can finish while
   the stderr pump sets the event; write completion and first stderr observation
   are therefore not independently ordered by this receipt.
3. Nonempty stderr set that event. The request loop stopped without validating a
   thread response. Whether the native process had created a thread or produced
   a response that the client did not consume remains unknown.
4. Cleanup closed the client's input to the native process. The native process
   returned zero. No turn or tool was requested.

The first captured line is the snapshot warning; later captured lines report a
transport connection no longer available. Native emission time is not client
observation time. There is no common event sequence locating each native message,
the first pipe read, the client's stop decision and input closure. The transport
messages could follow client-initiated cleanup, but that causal ordering is not
proved. Neither their placement in the capture nor native exit zero proves what
thread initialization would have done without the client stop. This result does
not explain the historical R2 attempt.

Yes: our rule that every stderr byte is terminal is sufficient to end this
observed client workflow on a WARN message, without a demonstrated fatal native
failure. The source checks the shared event before consuming a response and then
enters cleanup. That explains the recorded client stop; it does not prove that
the warning was harmless, that thread initialization would succeed, or that the
client stop was the only native-side issue. Keep the protection rule unchanged.

The warning's claimed shell-feature limitation is separate from provider policy,
OS permissions and identity. Positive configuration metadata is not proof of
provider entitlement, tool execution permission or OS sandbox enforcement. No
turn occurred; actual serving model, provider requests and billing remain unknown.

Exactly one proposed next diagnostic action, not performed or authorized here:
conduct a version-bound static source trace of the native PowerShell snapshot
branch and its immediate thread-initialization caller for the sealed candidate.
Test the hypothesis that this unsupported-shell condition is handled as an
optional warning rather than terminating thread initialization.

- If the branch's failure is caught and initialization continues, the hypothesis
  is supported for this condition. Our client then cannot treat this WARN alone
  as evidence of a native fatal failure; other initialization failures remain open.
- If this same condition propagates through the caller and aborts initialization,
  the hypothesis is falsified: the reported feature limitation has a native abort
  path in that build. It still does not prove the path was taken in this attempt.
- If the exact candidate's source provenance or the immediate caller cannot be
  established, the trace is inconclusive. The missing information is the
  candidate-bound error-propagation path; current metadata cannot substitute for it.

This is one narrow source investigation, not another identical runtime experiment,
a proposed rule bypass, or an implementation task. No candidate CLI execution,
test, app-server, Actor, product or provider start was made; no new quota was
allocated. Only two existing local technical sources were inspected: the sealed diagnostic client and
its metadata-only terminal result. Old evidence, usage and study pins remain
unchanged. The explanation supports investigating native error propagation before
deciding whether any different future experiment is warranted.
