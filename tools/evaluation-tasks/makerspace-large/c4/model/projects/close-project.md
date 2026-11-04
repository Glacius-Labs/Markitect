---
kind: Operation
---
# Close project

The project's status becomes `closed`. A closed project takes no collaborator and no attachment, and its cost can still be read ([[projects/attachment]]). A member acts only on a project the member owns, as [[authorization]] states.

- **Wire name:** `closeProject`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `projectId`: identifier, naming an existing [[projects/project]]
- **Refusals** at check 7, in this order:
  1. `invalid_state` when the project's status is `closed`.
- **Result:**
  - `project`: a [[projects/project]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId`.
