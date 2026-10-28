---
kind: Operation
---
# Add course

Creates the course with `enrolledCount` 0.

- **Wire name:** `addCourse`, a mutation.
- **Arguments** besides `actor` and `at`, in the formats of [[values]]:
  - `courseId`: identifier
  - `familyId`: identifier, naming an existing [[equipment/family]]
  - `title`: text
  - `feeCents`: integer from `0` to `1000000` cents (a course fee)
  - `capacity`: integer from `1` to `1000` (the number of places of a course)
- **Refusals** at check 7, in this order:
  1. `already_exists` when a course with `courseId` exists.
- **Result:**
  - `course`: a [[training/course]] record
- **Permissions:** staff all, admin all.
- **Audit target:** `courseId`.
