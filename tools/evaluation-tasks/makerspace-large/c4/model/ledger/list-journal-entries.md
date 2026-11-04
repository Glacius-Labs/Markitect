---
kind: Operation
---
# List journal entries

Returns the journal entries whose `at` lies in the month, that is, whose UTC date is in the month. A month without entries gives an empty list.

- **Wire name:** `listJournalEntries`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `month`: month
- **Result:**
  - `month`: month
  - `entries`: a list of [[ledger/journal-entry]] records, in creation order (ascending entry number)
- **Permissions:** admin all.
