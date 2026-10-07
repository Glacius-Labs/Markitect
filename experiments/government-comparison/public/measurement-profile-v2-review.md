# Independent review of common measurement profile v2

Date: 2026-10-07. Reviewer: separate `independent_review` subagent, read only.
Parent owns implementation and verification. The reviewer ran no tests, metadata
RPCs, native product binaries or model sessions and changed no files.

Reviewed scope: common profile definitions, admission and append-only ledger
migration, successor authority/accounting, original dispatch recovery, metadata
client/argv pinning, diagnostic history, context-only draft preparation and finite
verification/freeze helpers. Final narrow review checked the cumulative historical
attempt validator and its regression. Exact delivered sources are in the package
freeze; this note records source review, not a separate runtime test result.

Findings and disposition:

1. Original singleton authority prevented explicit later grants: append-only
   authority/profile/ceiling history now preserves the original row and binds each
   dispatch to its original authority and profile. Recovery validates that binding;
   recovery-only handles cannot reserve. New ceilings are cumulative allocations.
2. Profile adoption could commit before an invalid successor failed: adoption and
   authority insertion now share one transaction. The regression compares the
   entire original snapshot and authority history after rejection.
3. A fresh context ledger would bypass three consumed diagnostic starts: drafts
   now name the existing selected-runner ledger/authority and propose an absolute
   ceiling of four, meaning only one additional session. Both legacy ledgers and
   unknown counters stay preserved. Their common-ledger migration is unresolved;
   the draft explicitly remains blocked, with a null migration receipt.
4. Missing or partial ready bindings could bypass that block: live v2 requires the
   complete, identical Grant/Protocol history/path/identity/allocation binding,
   approved hash-bound migration receipt, actual migrated trial and exact mapped
   attempt records. Missing, partial and fresh-ledger bindings fail closed.
5. A migrated zero could replace unknown provider requests: exact mapped source
   identities, finished flags, statuses, request and token counters must match the
   preserved history, including nulls. Regressions mutate each of those fields.
6. The standalone metadata client accepted arbitrary argv: it now verifies the
   existing binary pin and exact metadata-only app-server argv before spawning.
   A negative test proves an exec argv never reaches Popen. RPC whitelist remains
   initialize/initialized/account-read without refresh/model-list only.

Final reviewer conclusion: no remaining material issue within the agreed scope.
The draft cannot authorize a session; no legacy diagnostic migration or live
session has occurred. Native Classic/Government readiness, actual context/access
proof and S1 acceptance remain outside this mechanics/draft checkpoint.
