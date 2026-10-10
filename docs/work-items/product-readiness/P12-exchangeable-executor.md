# P12 — Exchangeable executor and per-role configuration

Separate the delegated method from its execution runtime so it can be used and evaluated with executors other than the native Codex App Server. This schedules two concept-register entries: ENH-005 (exchangeable executor for the delegated method) and ENH-004 (per-role agent configuration and model mixing), recorded by the owner on 10 October 2026 in `docs/concepts/register.md`. That register is introduced by the concept-record branch `claude/vision-concept-record-20261010` and is not yet on Main; when it lands, both entries link back to this item.

The method stays deterministic and provider-neutral: compiled model, coverage, impact, Manager scopes and briefings, required reviews, candidate validation, Verify and guarded Apply. Scope, Verify and Apply are not weakened for any executor. Cost reports stay honest: an unpriced executor is unknown cost, never zero. Submitted reviews remain AI evidence, not human acceptance. No release follows from this item.

## Increment 1 — process executor and role profiles (this change)

- Agent cost mode `metered` (default) or `unmetered`. Unmetered process executors declare no rates, may omit usage and are recorded as unknown cost; start, duration and timeout limits are unchanged. Metered process executors that omit usage still stop. The Codex App Server reports usage and stays metered.
- `project setup` selects provider (`codex` or `process`), model, effort and cost mode per role class: Managers, reviewers, verifier. The default remains Codex App Server, `gpt-6-luna`, `high`; mixed profiles are allowed. The CLI takes the default profile from flags and per-role profiles from `--input`; MCP takes the same options record.
- The process transport is documented as the bring-your-own executor contract in [Provider adapters](../../provider-adapters.md#bring-your-own-executor), with the reference adapter `markitect-exchange-executor` (`request.json`/`response.json`).
- End-to-end tests drive Plan, Run, Review, Verify and Apply with mixed role profiles and scripted fault injection: a write outside scope, a forgotten file, a review finding with repair, and an integration failure despite green children.
- Windows: declared checks no longer run in a materialized candidate under the repository run store. Below a deep repository or `GOTMPDIR` that directory exceeded the 258-character process working-directory limit, and every check failed with "The directory name is invalid".

Acceptance: focused unit, CLI and end-to-end tests pass on Windows; the existing Codex default profile produces the same runtime as before; the full suite and hosted gates run on the final candidate.

Known limits after increment 1: invocations are synchronous and end at the role timeout (at most 60 minutes); process candidates cannot delete or rename files; process executors have no native workspace or helpers; Brownfield manager stages reject unmetered agents because their ledger seals a priced estimate per attempt; a Manager that reports completion without writing its required artifact is caught by declared checks or review, not by a separate structural guard.

## Increment 2 — external work (separate change)

- An `awaiting-external` run state instead of a blocking invocation.
- `project packet` exports the deterministic work packet of one role; `project submit` accepts its candidate or verdict.
- Git candidate intake for external candidates, including deletions and renames.
- Review verdicts bound to the candidate and scope digests; they remain AI evidence.
- Plans that do not bind executor fingerprints, so an executor can change without invalidating the plan.

## Increment 3 — later

Claude Code as a full inner executor with native workspace work, comparable to the Codex App Server adapter.
