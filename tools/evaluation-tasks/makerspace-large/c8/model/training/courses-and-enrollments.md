---
kind: std/Rule
internal: true
---
# Courses and enrollments

A [[training/course]] teaches the safe use of one tool family. It has a fee and a capacity. A member enrolls; staff record completion, which certifies the member for the course's family. There is one [[training/enrollment]] record per course and member. It is created by the first enrollment; a later enrollment after completion reuses and resets it.

The course's `enrolledCount` is the number of its enrollments with status `enrolled`. A place becomes free when an enrollment is completed.
