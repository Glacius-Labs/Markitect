# Recorded review cases stand for every delivery of a purpose (18 October 2026)

Source of [DEC-048](../register.md#dec-048-recorded-review-cases-stand-for-every-delivery-of-a-purpose).

## Evidence

Bound library run R2-A1-08 (Main a9f62311, with [DEC-047](../register.md#dec-047-recorded-review-cases-stand-until-their-inputs-change)) ended Incomplete after 73 steps and 8945 s. `library` reached the local rejection limit of three. There was no deadlock, owner question or Host fault.

`library` made three limited local deliveries. Each new delivery recorded a fresh case set, because DEC-047 kept cases only after a rejection or a withheld pass, and only for the same scope.

1. 24 cases. Rejected for a defect, then passed on the same cases.
2. Fresh, 25 cases. Rejected for due-date scenarios and refusal paths through the composition that this set named, then passed on the same cases. These may have been real gaps.
3. Fresh, 27 cases. Rejected because a new case required a name or title larger than 64 KiB. In the first two sets the model's no-line-limit property belonged to another case, and two reviews had judged it evidenced by the same tests, which use a large identifier.

The expectations step sees neither the candidate, the aggregate nor what a limited delivery contains. A new delivery therefore gave it the same inputs as the last one, and the third set only redrew the cases.

Across the six Managers, 5 of 11 fresh case sets passed their first review, and 7 of 8 reviews against kept cases passed. This favors kept cases, but the comparison is confounded: a review against kept cases follows rework against those cases.

## A related reading conflict

All three `library` sets read "unknown argument fields" of a known operation as a refusal. The model says such fields "make the request invalid". The owner's answer recorded in R2-A1-04, and the independent oracle, both expect `invalidRequest`. The Manager first implemented `invalidRequest`, then changed it to satisfy the first rejection. No owner question was raised. Because cases now stand, such a reading would persist for the whole run. The prompts therefore treat recorded cases and findings as readings of the model, judged by its text. A Manager keeps behavior that the text supports and cites the passage, and the next review marks a contradicted case not applicable. The owner is asked only when the text leaves a consequential reading open, or when a review keeps its reading against the cited passage. This is the interim rule recorded at DEC-043.

## Owner instruction

The owner had authorized the Architect earlier the same day:

> Ja setze alles um was du für sinnvoll hälst und schaue auch, ob du das ganze etwas beschleunigen kannst.

So authorized, the Architect adopted DEC-048 as the completion of DEC-046 and DEC-047. The owner's rejection-limit decision is unchanged. Its allowance is per run, Manager and purpose, and no delivery uses or resets it.
