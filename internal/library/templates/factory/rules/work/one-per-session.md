# One per session

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The agents](../../agents/README.md), [Report](../../../../packages/factory/report.md)

## Rule
Every session and every run lays one short [report](../../../../packages/factory/report.md) into its context folder at its end, every session that works on a plan item included: planning, building, analyzing, accepting. A run of an observer that observed nothing writes no report, only its checkpoint; its trial run is the exception, since a person accepts it: it always writes a report, saying what it read and, when so, that it found nothing. A report with nothing noteworthy in it gives the Triager no finding.
