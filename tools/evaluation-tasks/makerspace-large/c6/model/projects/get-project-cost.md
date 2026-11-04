---
kind: Operation
---
# Get project cost

Returns the cost of the project as [[projects/project-cost]] states. A member reads only the cost of a project the member owns, as [[authorization]] states.

Rules of other modules that apply here: [[projects/project-cost]] and [[authorization]].

- **Wire name:** `getProjectCost`, a read.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `projectId`: identifier, naming an existing [[projects/project]]
- **Result:**
  - `projectId`: identifier
  - `usageCents`: cent amount
  - `materialCents`: cent amount
  - `totalCents`: cent amount
  - `byMember`: a list of [[projects/cost-share]] records, in ascending `memberId` order
- **Permissions:** member own, staff all, admin all.
