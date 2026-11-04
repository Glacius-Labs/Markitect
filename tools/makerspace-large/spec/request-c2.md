# Change request: A waiver signature lasts a year

Our insurer no longer accepts waiver signatures older than a year, so from now on a signature lapses.

- A signature's **lapse date** is the date 365 days after the date of its `signedAt`. It counts up to the day before its lapse date and not on the lapse date itself. Example: a signature with `signedAt` `2026-03-02T09:00Z` has the lapse date `2027-03-02`; it counts at `2027-03-01T23:59Z` and does not count at `2027-03-02T00:00Z`.
- A member **has signed** a family's waiver when the member's latest signature for the family has the family's current version and has not lapsed on the date of the request time. The date of a slot plays no part in this.
- A lapsed signature no longer counts as signed wherever the program judges whether a member has signed.
- A member can now have more than one signature of the same version of a waiver, so a member's latest signature for a family is the one recorded last.
- The signature record keeps its fields; the lapse date is neither stored nor shown. There is no new operation and no new refusal code.
