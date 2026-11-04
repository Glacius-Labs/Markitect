---
kind: Operation
---
# Add collaborator

Adds the member to the project's `collaboratorIds`, which stay in ascending order. A member acts only on a project the member owns: ownership is judged on `projectId` alone, not on `memberId`, as [[authorization]] states.

Rules of other modules that apply here: [[projects/attachment]] and [[authorization]].

- **Wire name:** `addCollaborator`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `projectId`: identifier, naming an existing [[projects/project]]
  - `memberId`: identifier, naming an existing [[members/member]]
- **Refusals** at check 7, in this order:
  1. `project_closed` when the project's status is `closed`.
  2. `already_collaborator` when `memberId` is the project's owner or is already in `collaboratorIds`.
- **Result:**
  - `project`: a [[projects/project]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId/memberId`.
