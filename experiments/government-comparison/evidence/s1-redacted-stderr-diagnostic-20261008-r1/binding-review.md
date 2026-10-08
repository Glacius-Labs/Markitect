# Independent request and freeze binding review

I independently checked the committed source, request, freeze, live grant and active slot using read-only file and Git-object inspection. I did not run the project validator, client, tests, candidate executable, or any launcher.

Source commit `4f091251a6be9c57597ef4628474ad6bbd7610b3` is current `HEAD`. All 23 request source pairs match the working-file SHA-256, the corresponding Git blob at S, and the request digest. All 33 freeze file pairs match the current file SHA-256. The request and freeze share the exact key, grant issue time, later slot assignment time, and source commit. The freeze's `requestSha256` matches the request bytes, and `frozenAtUtc` (`2026-10-08T16:56:59.660396Z`) is later than the assigned slot time (`2026-10-08T16:52:42Z`). The packet's profile and authorization-grant pins also match the request.

The copied full grant semantically matches the current canonical Scientist grant, including JSON value types. The active `fullSuiteSlot` matches the Scientist owner, exact key, required assigned status, grant issue time, and recorded activation time. Unrelated coordination-state snapshot changes do not alter that grant/slot match.

The freeze pins the exact candidate executable, Python interpreter, both public cwd input files, and five historical ledgers. I checked the frozen executable and interpreter hashes against current bytes; both cwd files match their profile hashes and are the only files in that public cwd. All five ledger entries match the frozen values. The schema archive hash matches, and all 12 required member names and SHA-256 values match the archive contents.

The exact external evidence root and its reservation file are absent. No reservation was created and no runtime was started. This review establishes only the static binding snapshot; it is not a runtime or source-compatibility result.

Verified identities:

- Source commit / current `HEAD`: `4f091251a6be9c57597ef4628474ad6bbd7610b3`
- Request SHA-256: `b2822e0d59043080f8e6ddd329dd433a125135559d6d76240d6dd251d9dc41be`
- Freeze SHA-256: `c7ef4d58aa0be825d59eca6e34c01ce982ee8736136866203406b5680c8b7dfc`
- Source pairs: 23/23 exact; freeze pairs: 33/33 exact
- Schema members: 12/12 exact; historical ledger pins: 5/5 exact
- Public cwd files: 2/2 exact; external root absent; reservation absent
