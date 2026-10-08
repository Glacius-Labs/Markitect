# Government check receipt identity: offline handoff

Status: offline correction and bounded independent review passed.
Base: `130fcd85c866e7643cdb7f5bcb844c8ccd2ec311`.
Source commit: `9dbbb93eea895947f800b1354fe056ac5b4f08d8`.
Final delivery commit is the enclosing clean Git commit, supplied in the
single direct Overseer callback. No push, PR or publication.

The actual pre-Resume consumer `_validate_r4_fresh_check_receipts` now derives
the expected executable from the valid nonempty configured `run` list's first
token. Pinned Government04e225d does exactly this when producing GateResult.Tool.
There is no `tool` fallback or command-string parsing. Exact identity set,
count and duplicate checks, strict integer-zero exit code and bounded integer
duration/timeout checks remain. Invalid configured command and observed identity
shapes fail closed. All surrounding runtime/report/Request/receipt/candidate/
evidence/scope/review/final Ressort/decision/promotion bindings remain unchanged.

Focused tests:13/13 passed on the first run. Twelve new tests read the actual
SHA-bound sealed R6 configuration/report forms; three exercise the actual
pre-reservation helper and real queue-result translator with isolated relocated
copies, including negative tool and runtime/report digest variants. Neither
consumer nor translator is mocked; process starts and SQLite connections are
blocked. Only the directly modified existing R4 helper test was additionally
run. Other suites were not repeated. Syntax for three files and diff checks pass.
The isolated copies rebind temporary file paths/digests, without actual
Authority admission, resource reservation, native execution or model calls.
They establish this receipt-consumer boundary, not an end-to-end Resume.

Independent review: no material findings; `independent-review.md`, SHA-256
`db824adc65df423aeb1083caec41c1e2ee0697ca0bee953e511725a1b7ee7bc2`.
Pinned verify.go/check.go snapshots, source-diff.patch, focused test raw output,
validation, historical preservation baseline and manifest provide the bindings.

New experimental consumption is zero: no actual case preparation, admission,
freeze, native/controller/wrapper/delegate/provider/model/metadata start, study
cell or liveledger mutation. Cumulative evidence stays13native/13wrappers/
10deterministicdelegates/1950reservedseconds; real historical Actorstarts5,
knownTokens53331, total tokens and engineering/review overhead unknown.
All3711protected files,92R6external original/copy pairs and the liveledger remain
unchanged; the prior general handoff content remains as an exact suffix.

R6 remains terminal INCOMPLETE, R5 NOT ADMITTED and R4 incomplete. The corrected
consumer is a pinned R6 runtime input: its changed bytes invalidate that old
freeze. No old replay/revalidation/new freeze was attempted or is authorized.
Product pins Government04e225d and Classicc91363b remain unchanged. S1, actual
terminal Replay, tool capability, semantic quality and human acceptance remain
open. Any fresh finite execution needs separate Overseer authority after review.
Scientist finishes this packet, reports once, then waits.
