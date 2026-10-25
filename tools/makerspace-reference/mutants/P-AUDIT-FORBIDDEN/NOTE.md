# P-AUDIT-FORBIDDEN

Step tree: `w0`. Apply with `../apply.sh P-AUDIT-FORBIDDEN DEST`.

Breaks: In W0 a forbidden refusal for missing permission appends an audit entry, which W0 does not allow. (From C1 on refusal entries are required, so this mutant is defined for W0 only.)
