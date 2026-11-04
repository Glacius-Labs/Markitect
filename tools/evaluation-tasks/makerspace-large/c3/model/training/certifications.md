---
kind: std/Rule
---
# Certifications

A member has at most one [[training/certification]] per tool family. Completing a course issues it: `issuedOn` is the date of the request time and `expiresOn` is 365 days after `issuedOn`. If the member already has a certification for that family, whether valid, expired or revoked, it is replaced by the newly issued one.

A certification is **valid on date D** when it is not revoked and `issuedOn ≤ D < expiresOn`. It is not valid on its `expiresOn` date. Its reported `status` at a request time is `revoked` when revoked, otherwise `expired` when the date of the request time is on or after `expiresOn`, otherwise `valid`.
