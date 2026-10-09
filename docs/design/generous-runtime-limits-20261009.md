# Generous runtime limits

Direct human instruction, 9 October 2026: give healthy work enough time, rather than repeatedly restarting it after an unnecessarily short timeout. This prospectively supersedes the short time/start limits of the active product-readiness package. Historical attempts and receipts remain unchanged.

## Current product acceptance allocation

Integration owns the existing three A01–A03 jobs and their central ledger. Revised bounds are four hours wall per job, twelve hours total and 256 total Manager/helper/reviewer start requests including failed requests. The earlier allocation was 1,800 seconds per job, 5,400 seconds total and 64 requests; preserve it as historical, not as the current ceiling. No new job is added. All known counters remain; no automatic refill.

Individual product roles, helpers and reviewers may take up to sixty minutes, bounded by the enclosing job's remaining time. Integration must make configurable runtime validation and every nested Host, App Server, RPC, dynamic-tool, CLI and process timeout consistent. A smaller hidden inner cap must not cancel otherwise authorized work. Keep an explicit timeout reason, layer and owned handles. Do not stop a healthy current run simply to replace its settings.

## Development checks

Use ten minutes for focused checks, thirty minutes for broad package suites and sixty minutes for the full suite, with at least ninety minutes for an enclosing process or CI job. Increase these reasonably when a concrete suite needs more time and record why. These are development defaults, not a guarantee of completion or a command to rerun unchanged checks. A timeout is an observed result; preserve it and inspect the cause before deciding on a retry. Do not run automatic test/provider retry loops. Concurrency is not raised indiscriminately.

## Lifecycle and evidence

Incomplete usage or lifetime-start telemetry does not alone prevent harvest of a reconciled terminal invocation. Require the original owned thread/turn terminal, every known child/start reconciled terminal and owned transport/process handles completed or controlled terminated. A concretely unaccounted child, contradictory or missing turn, open owned tool/process handle or ownership uncertainty preserves the candidate and evidence and prevents an unsupported cleanup/promotion claim. Generic partial accounting remains a cooperative boundary, not an all-child absence or OS-isolation guarantee. Do not fabricate complete accounting or search global process inventories to claim it.

Source binding, ownership, candidate validation, Verify/Apply, normal checks and independent review remain required. No Case Study starts, historical grant reuse, release, installation, purchase, provider integration or protection bypass follows from this time-limit adjustment. Future study budgets must be set generously when separately authorized; current studies remain stopped.
