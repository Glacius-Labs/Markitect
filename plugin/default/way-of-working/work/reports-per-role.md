# One module of reports per role

Type: [Rule](../../core/rule.md)

## Applies to
[Every role](../team/README.md), [the watches](../observing/README.md)

## Rule
Every role an agent fills has exactly one module below `reports`, named after what it does: `reports/assist` for the Assistant, `reports/run` for the Housekeeper, `reports/triage`, `reports/record` for the Project manager, `reports/brief`, `reports/plan` for the Architect, `reports/build`, `reports/review`, `reports/analyze`, and `reports/accept` for the Acceptance helper; every watch has a module of its own, `reports/observe-NAME`. The module always holds the same: one report per session or run, and, where the role continues from where it stopped, a `checkpoint.md`. Dropped material lies in the module `drops`; the report on it lies in `reports/observe-dropped-material`.

## Example
`reports/build/2026-11-24-1432-renew-a-loan.md` and `reports/triage/checkpoint.md`.
