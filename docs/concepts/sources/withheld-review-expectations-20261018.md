# Withheld review keeps its expectations (18 October 2026)

Source of [DEC-046](../register.md#dec-046-a-withheld-pass-keeps-its-recorded-expectations).

## Evidence presented to the owner

During the library Durchstich the Architect reported two real runs in which a pass that the Orchestrator withheld under D4 led to a rejection of an unchanged candidate:

- R2-A1-05 (Main ddbbd4dc): the third local rejection of `lending/reminders` followed a withheld pass; the run finished Incomplete at the rejection limit.
- R2-A1-06 (Main 1d0f31f4), journal revisions 5758, 5889, 5993 and 6122: the complete local candidate of `lending/reminders` passed, the pass was withheld because `catalog` delivered again, the unchanged resubmission received a fresh expectations step with new, stricter cases, and the next reviewer rejected it.

The Architect proposed that a withheld pass keep its recorded expectations for the resubmission while the expectation step's inputs are unchanged. An independent review showed that this changes the literal sequence of core rule T7, so the Architect parked the change as a proposal ([PR #447](https://github.com/Glacius-Labs/Markitect/pull/447)) and asked the owner to decide.

## Owner instruction

> Ja setze alles um was du für sinnvoll hälst und schaue auch, ob du das ganze etwas beschleunigen kannst.

The Architect, so authorized, adopted the change.
