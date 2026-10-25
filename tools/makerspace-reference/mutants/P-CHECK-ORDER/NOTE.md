# P-CHECK-ORDER

Step tree: `w0`. Apply with `../apply.sh P-CHECK-ORDER DEST`.

Breaks: The permission check (4) comes before the request-time check (3): a request that fails both is refused forbidden instead of time_regression. Both codes still exist.
