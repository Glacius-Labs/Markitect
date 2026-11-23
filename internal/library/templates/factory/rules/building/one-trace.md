# One trace from origin to file

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [the Reviewer](../../agents/review.md), [the Builder](../../agents/build.md)

## Rule
One trace serves everything. The story names its origin, a finding or a report, and the places in the spec it changes, and how. The Reviewer's `confirm`, in the story's Result, names for each place in the spec the story realizes the files and the test that proves it. When the place is a document of the spec, the tool's `confirm` binds that document to the fingerprints of those files; the tool binds documents of the spec only. The commit names the story. So the trace leads from the origin to the file and back, and answers "where does it say it should be so?", for the product and for the way of working alike. Who checks names the evidence, not who built. The tool's `confirm` takes the document with the fingerprint that `markitect show DOCUMENT` prints, also when `status` does not list the document, and confirms nothing while the spec has errors, `section.empty` included; then the Result entry stands alone, and the Reviewer says in its report what the tool refused. When a story changes no standing spec, the place is its own target picture: the Result entry is the record, and the tool binds nothing. When a bug changes files but no spec, the place is the document that already says what is right, and the evidence is the test its fixed tasks demand. Files from before this way of working have no entry; they get one when a story touches them.

## Example
`confirm: lending/period.md: src/loans/renew.ts, tests/loans/renew.test.ts`, in the Result of the story "Renew a loan", whose merge commit names the story.
