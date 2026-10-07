# Native integration post-run review

This is a read-only review of the single Government queue execution authorized by the frozen preflight. It does not alter the preflight at `native-integration-preflight.md` (SHA-256 `b2088a9c1caa4a511273666d223f02fc25c7b182565e8998040cf04598db10cc`). No product CLI, test, source repair, queue resume, or relaunch was performed during this review.

## Government queue result

The frozen Government CLI process returned exit code 0, with no stop reason, after 1.193 seconds. That is process completion only: native stdout and the final durable queue report both say `status=incomplete`. The queue reports `actorStarts=1`, `repairs=0`, `inFlightActors=0`, peak parallelism 1, and 159 ms observed actor wall time. The only job, `positive-overflow-release`, is `incomplete`, with error `external runner failed`. The run report is `stage=execute`, `status=incomplete`; no evidence identity, findings, checks, decision, or votes were produced. No replay or resume was attempted.

There are distinct layers and counts:

| Layer | Observed result |
|---|---|
| Native G5 queue process | One process; exit 0; no stop reason; wall 1.193 s |
| G5 queue / native Actor record | One queue actor start and one `execute` actor record; actor `outcome=failed`, retry count 0 |
| Role bridge wrapper | One invocation was attempted for the configured executor slot; it returned no valid structured response. Its recorded stdout digest is the empty-file digest. |
| Deterministic role delegate | Zero reservations/launches recorded. The synthetic fixture ledger has zero `attempts`, no `government_role_calls` table, the dispatch Actor attempt remains null, and the role-evidence directory is empty. The wrapper's reserve-before-delegate path means no delegate effect was authorized or launched. |
| Outer adapter translation | The written Result is `incomplete`, `receipts=[]`, `usage=null`, and `candidateCommit=null`. It reports both the consumed/ambiguous controller reservation and `Government evidence identity missing`. The identity gap is a secondary translation finding from the incomplete native report, not another queue attempt or the recorded external-runner error. |

The single actor receipt records an empty Response, empty stdout (`sha256:e3b0c442…`), stderr digest `sha256:5a1eb70373ac4e071278441cdec844ec0cd01c8ff64094d428c932447d563477`, `privateLogDigest=""`, `outcome=failed`, and `retryCount=0`. The available controller process stderr file is empty. No raw actor stderr or private log with the matching digest is present in the captured artifacts, so the underlying wrapper failure cause cannot be identified from this evidence. It would be incorrect to infer it from the outer adapter's separate evidence-identity gap.

The common synthetic ledger's read-only snapshot has zero `attempts`, one `controller_runs` row still `running`, no `controller_processes` rows, and the dispatch remains `launching` with `attempt=NULL`. Thus the role delegate was not reserved, but the controller reservation/process finalization is incomplete in the ledger despite the external process and durable queue receipts. The already consumed controller reservation is not a basis for replay. The queue is not to be resumed or relaunched under this result.

## Evidence bindings

| Artifact | SHA-256 |
|---|---|
| Frozen preflight inputs | `746df37019541ffa7c395afc5b7f6189f93d5b7b75a540f7b5d1431e015b2c10` |
| Local adapter Result (`queue-result.json`) | `ea5c1e6484d60906f878af1f72bf427c5e9fb94ae104aedfaafb80188969cf30` |
| Driver summary (`queue-driver.log`) | `103a904391cefef158feeda2411cc72360fd174c4878c146dcb88219955cb949` |
| External G5 process receipt | `a14c90d15e040e7591e2cbbd9d98a016d4e8ec2ab9ae60e74a2fea795c3f8bc1` |
| G5 stdout / final queue report | `86420b46c3476029e939fbdc3068989643d7b9f1a25726874b0b8ef6a972c0be` |
| G5 stderr | `e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855` |
| Queue event journal | `35ae29b7ad40cc897d95d084c303e5e3f7d1bd86bf219132fc49567dd46d9930` |
| Native run report | `cee003ddd3ae118ad20295d588b76b5e1f4e30818869622551266c0fe15f1d98` |
| Native actor record | `7b5e02b4203eafa64f01303c9928bff88d30787ba07ed7589bffe39a529f409c` |
| Synthetic fixture ledger at review | `c00c116064ecc8cd910c72ffb6f6e250b403fded24e20f85e8b55d38f87140a5` |

The ledger was opened read-only. Its controller row and failed dispatch are retained as observed; this review does not finalize or otherwise repair them. Usage remains unknown/null. The fixture's separate known-no-provider quota accounting does not convert the native product report's unknown input, output, cached-token, or tool-call metrics into zero.

## Disposition

Government positive execution failed at the external role-runner boundary and produced no approval evidence. The outer process exit code 0 must not be described as a positive queue result. Preserve the consumed reservation and incomplete report; do not retry, resume, or repair this run. S1 remains open, and no Government semantic or human-acceptance conclusion follows. Classic execution is outside this review and must be assessed separately against its own frozen booking and process evidence.
