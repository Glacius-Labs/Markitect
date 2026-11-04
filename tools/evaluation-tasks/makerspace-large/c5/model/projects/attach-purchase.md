---
kind: Operation
---
# Attach purchase

Adds the purchase to the project's `purchaseIds`, after the ones already there. A member acts only on a project the member owns: ownership is judged on `projectId` alone, not on the purchase, as [[authorization]] states.

Rules of other modules that apply here: [[projects/attachment]] and [[authorization]].

- **Wire name:** `attachPurchase`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `projectId`: identifier, naming an existing [[projects/project]]
  - `purchaseId`: identifier, naming an existing [[materials/purchase]]
- **Refusals** at check 7, in this order:
  1. `project_closed` when the project's status is `closed`.
  2. `not_project_member` when the purchase's member is neither the project's owner nor in its `collaboratorIds`.
  3. `already_attached` when the purchase is attached to any project.
- **Result:**
  - `project`: a [[projects/project]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId/purchaseId`.
