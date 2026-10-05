# Service-only predicate correction

This is a measurement correction, not a Markitect product change. The [experiment](design.md) and [operator contract](operator-contract.md) still apply. The product candidate, public checks, application seeds, task cards, model/effort, repair count and deadlines remain unchanged.

The [source-only recurrence record](results/r1-oracle-v2/service-predicate-recurrence-v1.json) confirms two false negatives in the frozen `d60e4975ab34b6519c2db414a58c170898750cacc59e077a199e2504f6bb6ebd` Service evaluator:

| Desired fixture fact | Old recognition | Corrected finite recognition | Negative boundary |
|---|---|---|---|
| Three module ownership rows permit Core and Contracts dependencies. | Labels and parenthesized paths pass; the exact standalone paths `internal/core, internal/contracts` fail because a slash prefix remains after lexical removal. | Accept those two exact known paths as alternate spellings of the same dependency facts. | Wrong owner/package, duplicate/missing dependency, sibling module, nested path, mismatched label/path and conflicting ownership remain failures. |
| Canonical availability Query belongs to Inventory. | The typed name `get-availability` and Inventory relation must additionally repeat the Go token `GetAvailability` in the UseCase YAML. | Compare the existing canonical name and typed Inventory relation. Implementation behavior remains independently checked. | Missing typed owner, wrong module and wrong canonical name fail. No inferred ownership from prose. |

These are bounded fixture predicates, not arbitrary Markdown/YAML semantic analysis. The correction does not establish complete oracle accuracy. The Go implementation identifier remains in the implementation/guidance/check surface that owns it; duplicating it into canonical resource prose is not a hidden task requirement.

The diagnostic executed only AST-extracted pure predicates and fixed synthetic inputs. Its successful invocation exited zero and is bound to the exact frozen evaluator bytes. The earlier diagnostic helper failure is retained separately. A reviewer accidentally emitted result-tree path lines while locating the freeze; the [root validation disclosure](results/r1-oracle-v2/service-predicate-recurrence-root-validation.json) records that incident. No blinded/no-outcome-access attestation is claimed, and no hidden feedback was sent to scored actors.

## Preservation and next cohort

The d60 Service execution stopped after eight prepared, terminal turns: A trial1 tasks01-06 and P01/P02. B, integration, later trials and holdout were not prepared. Their planned comparison is invalid, not a losing-arm result. The exclusion list must name only actual existing run IDs; a cohort decision records the unstarted paired scope. Public helpers and raw statuses are preserved without rescore. Operations continues under its separate unchanged d60 measurement.

The next Service measurement requires a new external arena, exact-source Windows/Linux QA gates, a new immutable freeze and independent byte/parity review before actors start. It contains three fresh pairs from unchanged seeds, with A/B/A first-arm order, development tasks01-08 and the same reserved09-12 outcome gate. Only protocol cohort metadata, the Service evaluator and its QA binding are replaced; every other predecessor input retains its bytes. The correction note, CI record and preparation provenance are separately added and identified. Earlier trajectories are never reused as corrected tasks.

Capacity remains one scored leaf per operator and three globally. Fork workspaces remain isolated but serialized under the measured capacity limitation; this does not prove simultaneous-agent behavior. Every new task retains the same total deadline including preparation, queues, waits and repair. No product fix/no-fix decision or holdout dispatch is implied by this correction.

The [validation report](../../docs/validation/autonomous-ab-gauntlet.md) owns actual outcomes, missing evidence and integration status. Synthetic measurements cannot prove human-attention savings or adoption value.
