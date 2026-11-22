# One per session

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The agents](../../agents/README.md), [Report](../../../../packages/factory/report.md)

## Rule
Every session and every run lays one short [report](../../../../packages/factory/report.md) into its context folder at its end, every session that works on a plan item included: planning, building, analyzing, accepting. Every report has the same head: the item, who started it, for whom, when it started and finished, and its outcome. What the Housekeeper starts, its own report lists too, so runs without a report are covered. A crashed session shows as a claim that stands still. A report with nothing noteworthy in it gives the Triager no finding. Reports of agents, such as an aborted build or a smell analysis, are reports already: the Triager reads them directly, and they do not pass through the observer dropped-material.
