---
kind: std/Rule
---
# Posting notices

An operation of another module may post notices for members as part of its own effects. A posted notice is a [[notices/notice]] that is created like the one of [[notices/post-notice]], with these differences:

- `kind` is the kind the posting operation names, and `ref` is the reference it names. Neither is `manual` or `null`.
- `subject` is the subject of the kind, as [[notices/notice-kind]] states.
- `at` is the request time of the posting operation, and `acknowledgedAt` is `null`.
- It takes the next notice identifier from the same counter as the notices posted by hand. Notices posted by one operation are numbered in the order that operation states them.
- Standing is not checked.

A posted notice takes place together with the other effects of the posting operation. A refused request posts none. It adds no audit entry: the entry of the posting operation stands for it.
