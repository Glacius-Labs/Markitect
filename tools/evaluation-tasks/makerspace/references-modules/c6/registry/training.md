---
paths:
  - realization/training/**
---
# Registry: training

| File | Realizes | Why |
|---|---|---|
| `realization/training/operations.go` | [[training/add-course]] [[training/enroll-in-course]] [[training/complete-course]] [[training/revoke-certification]] [[training/list-certifications]] [[training/certifications]] [[training/courses-and-enrollments]] | Training operations: courses, enrollments and certifications |
| `realization/training/training.go` | [[training/training-interface]] [[training/course]] [[training/enrollment]] [[training/certification]] [[training/certifications]] [[training/courses-and-enrollments]] | Courses, enrollments and certifications, and the certification check other modules ask for |
| `realization/training/views.go` | [[training/course]] [[training/enrollment]] [[training/certification]] | JSON views of the course, enrollment and certification records |
