---
kind: std/Rule
---
# Maker gate

A [[jobs/job]] starts only from an accepted [[quotes/quote]] and only with a maker who may do the work.

- A quote starts at most one job. A quote that has a job, started or completed, cannot start another.
- The maker must be in good standing ([[members/membership]]).
- For every `machine` line of the quote, the maker must have a [[training/certification]] for the family of the line's machine that is valid on the date of the request time ([[training/certifications]]). A quote with only `material` lines needs no certification. Several lines with machines of one family need that family's certification once.

The checks run in the order of the refusals of [[jobs/start-job]].
