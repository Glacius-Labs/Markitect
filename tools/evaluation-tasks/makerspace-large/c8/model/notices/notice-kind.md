---
kind: std/Concept
---
# Notice kind

The kind of a notice. It decides the `subject` of every notice that is not posted by hand. `<ref>` stands for the notice's `ref`.

| Value | Meaning | Subject |
| --- | --- | --- |
| `manual` | Posted by staff with [[notices/post-notice]]. `ref` is `null`. | As supplied. |
| `dunning` | An invoice is overdue. `ref` is the invoice identifier. | `Invoice <ref> is overdue` |
| `suspension` | The member was suspended because an invoice is overdue. `ref` is the invoice identifier. | `Membership suspended: invoice <ref> is overdue` |
| `waitlist` | The member was enrolled in a course from its waiting list. `ref` is the course identifier. | `Enrolled from the waitlist of course <ref>` |
| `cancellation` | A reservation was cancelled. `ref` is the reservation identifier. | `Reservation <ref> was cancelled` |
| `loan` | A tool loan is overdue. `ref` is the loan identifier. | `Tool loan <ref> is overdue` |

Example: a `dunning` notice for invoice `inv-3` has the subject `Invoice inv-3 is overdue`.
