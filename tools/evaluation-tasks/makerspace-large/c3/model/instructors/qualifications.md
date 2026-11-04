---
kind: std/Rule
internal: true
---
# Qualifications

A user is an **instructor** for a family while the user holds a [[instructors/qualification]] for it. A user holds at most one qualification per family and may hold several.

Who may be qualified, judged on the date of the request time when the qualification is made:

- A staff user may be qualified for any family.
- A member may be qualified for a family only when the member holds a certification for it that is valid on that date ([[training/certifications]]).
- An admin is never qualified.

The check is made once. A later expiry or revocation of the member's certification does not end the qualification. A qualification ends only when it is withdrawn, and the user may then be qualified for the family again.
