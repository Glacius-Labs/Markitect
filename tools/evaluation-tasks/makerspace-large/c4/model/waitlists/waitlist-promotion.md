---
kind: std/Rule
---
# Waitlist promotion

A promotion fills the free places of a course from its waiting list.

It goes through the entries of the course in join order. Before it tries an entry it checks that the course has a free place, which is its `capacity` less its `enrolledCount` ([[training/training-interface]]); when it has none, the promotion stops and the entries that were not tried stay as they are. Each entry is tried at most once in a promotion.

To try an entry, the member is enrolled in the course exactly as [[training/enroll-in-course]] does at the request time of the promotion: the same refusals in the same order, and when none applies the same enrollment, charge and discounts.

- When the enrollment succeeds, the entry leaves the list, the enrollment is added to `enrollments`, and a notice of kind `waitlist` is posted for the member with `ref` the course identifier ([[notices/notices-interface]], [[notices/notice-kind]]).
- When the enrollment is refused, nothing changes: the entry keeps its place in the list. It is added to `skipped` with the refusal code, and the next entry is tried. The codes that can occur are `member_suspended`, `guardian_consent_missing` and `already_enrolled`.

`enrollments` and `skipped` are in join order.

Example: a course has capacity 2 and one enrolled member. The list is `ann` (suspended), `bob`, `cy`. A promotion skips `ann` with `member_suspended`, enrolls `bob`, and stops because the course is full. `cy` is not tried and stays on the list. The result has `enrollments` with the enrollment of `bob` and `skipped` with `ann` and `member_suspended`.
