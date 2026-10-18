# Recorded review cases stand for every delivery of a purpose (18 October 2026)

Source of [DEC-048](../register.md#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose).

## Evidence

Bound library run R2-A1-08 (Main a9f62311, with [DEC-047](../register.md#dec-047-recorded-review-cases-stand-until-their-inputs-change)) ended Incomplete after 73 steps and 8945 s. `library` reached the local rejection limit of three. There was no deadlock, owner question or Host fault.

- **Kept cases converged.** Of 8 reviews against cases kept after a rejection or a withheld pass, 7 passed.
- **Fresh cases did not.** Every new delivery recorded a fresh case set, because DEC-047 kept cases only after a rejection or a withheld pass, and only for the same scope. Of 11 fresh case sets across the six Managers, 6 were rejected on their first review. The rejections named scenarios that only the new set specified.
- **`library`, three limited local deliveries:**
  1. 24 cases. Rejected for a real defect, then passed on the same cases.
  2. Fresh, 25 cases. Rejected for due-date scenarios and refusal paths that only the new set named, then passed on the same cases.
  3. Fresh, 27 cases. Rejected because a new case required a name or title larger than 64 KiB. In the first two sets the model's no-line-limit property belonged to another case, and two reviews had judged it evidenced by the same tests with a large identifier.

The expectations step sees neither the candidate, the aggregate nor what a limited delivery contains. A new delivery therefore gave it the same inputs as the last one, and the fresh set only redrew the cases.

## Owner instruction

The owner had authorized the Architect earlier the same day:

> Ja setze alles um was du für sinnvoll hälst und schaue auch, ob du das ganze etwas beschleunigen kannst.

So authorized, the Architect adopted DEC-048 as the completion of DEC-046 and DEC-047. The owner's rejection-limit decision is unchanged. Its allowance is per run, Manager and purpose, and no delivery uses or resets it.
