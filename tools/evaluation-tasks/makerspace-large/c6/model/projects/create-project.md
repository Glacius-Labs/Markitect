---
kind: Operation
---
# Create project

Creates the [[projects/project]] with status `open`, no collaborators and no attached reservations or purchases. The owner's standing is not judged. A member creates only projects of the member's own: ownership is judged on `ownerId`, which [[authorization]] states.

Rules of other modules that apply here: [[authorization]].

- **Wire name:** `createProject`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `projectId`: identifier
  - `ownerId`: identifier, naming an existing [[members/member]]
  - `title`: text
- **Refusals** at check 7, in this order:
  1. `already_exists` when a project with `projectId` exists.
- **Result:**
  - `project`: a [[projects/project]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId`.
