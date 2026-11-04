---
kind: Operation
---
# Attach reservation

Adds the reservation to the project's `reservationIds`, after the ones already there. A member acts only on a project the member owns: ownership is judged on `projectId` alone, not on the reservation, as [[authorization]] states.

Rules of other modules that apply here: [[projects/attachment]], [[bookings/reservations]] and [[authorization]].

- **Wire name:** `attachReservation`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `projectId`: identifier, naming an existing [[projects/project]]
  - `reservationId`: identifier, naming an existing [[bookings/reservation]]
- **Refusals** at check 7, in this order:
  1. `project_closed` when the project's status is `closed`.
  2. `not_project_member` when the reservation's member is neither the project's owner nor in its `collaboratorIds`.
  3. `already_attached` when the reservation is attached to any project.
- **Result:**
  - `project`: a [[projects/project]] record
- **Permissions:** member own, staff all, admin all.
- **Audit target:** `projectId/reservationId`.
