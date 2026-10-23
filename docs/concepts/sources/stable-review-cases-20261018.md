# Recorded review cases stand until their inputs change (18 October 2026)

Source of [DEC-047](../register.md#dec-047-recorded-review-cases-stand-until-their-inputs-change).

## Evidence

Two bound library runs ended Incomplete at the rejection limit of three, without a code defect asserted:

- R2-A1-05 (Main ddbbd4dc): `lending/reminders`; see [DEC-046's source](withheld-review-expectations-20261018.md).
- R2-A1-07 (Main 9c7227e5), `members`, three limited local submissions. Each resubmission received fresh expectations (19, 11 and 12 cases with new names and wordings) and a fresh reviewer. The invalid-status case was assessed as evidenced in rounds one and two and as missing in round three, on the same tests. The Manager saw only the missing-case findings, never the cases its next review would use.

The review did not converge across rounds: each round redrew the case set and re-judged every case, so a Manager that needed several deliveries was likely to exhaust the limit.

## Owner instruction

The owner had authorized the Architect earlier the same day:

> Ja setze alles um was du für sinnvoll hälst und schaue auch, ob du das ganze etwas beschleunigen kannst.

The Architect, so authorized, adopted DEC-047 as the extension of [DEC-046](../register.md#dec-046-a-withheld-pass-keeps-its-recorded-expectations) to rejections.
