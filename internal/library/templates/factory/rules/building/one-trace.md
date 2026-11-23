# One trace from origin to file

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [the Reviewer](../../agents/review.md), [the Builder](../../agents/build.md)

## Rule
One trace serves everything. The story names its origin, a finding or a report, and in its Target changes every place in the spec that its files realize, also where it changes nothing there. When no document of the spec covers its files, something is missing in the spec, and the story's planning adds it; a story with fixed tasks that meets such files aborts and is planned ([abort](abort.md)). A story that changes no file, such as a smell analysis, names none. Whenever files that realize a place change, that place is confirmed again, in building and in accepting alike. For each place, the Reviewer's `confirm` is the tool's: `markitect confirm DOCUMENT@FINGERPRINT ok`, with the fingerprint that `markitect show DOCUMENT` prints, naming with `--file` every file that realizes the document, those `show` lists that still do and the new ones, tests included, and with `-e` the test that proves it, a file of the project outside `.markitect/`; the tool replaces the document's list of files with them. A document that no file realizes, such as an observer's document, which the agent observe reads as it is, is confirmed `skip` with `-m` giving that reason. The state the tool writes into `.markitect/` goes onto the story's branch and lands with it; nothing of it goes into the story's Result. The commit names the story. So the trace leads from the origin to the file and back, and answers "where does it say it should be so?", for the product and for the way of working alike. Who checks names the evidence, not who built. Files from before this way of working have no entry; they get one when a story touches them.

## Example
For the story "Renew a loan", whose merge commit names the story: `markitect confirm product/lending/period@3f9a1c07 ok --file src/loans/renew.ts --file tests/loans/renew.test.ts -e tests/loans/renew.test.ts`.
