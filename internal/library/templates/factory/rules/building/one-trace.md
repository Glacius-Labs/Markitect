# One trace from origin to file

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md), [the Reviewer](../../agents/review.md), [the Builder](../../agents/build.md)

## Rule
One trace serves everything. The story names its origin, a finding or a report, and the places in the spec it changes, and how. The Reviewer's `confirm` names, for each place in the spec the story realizes, the files and the test that proves it; the tool binds it to the files' fingerprints. The commit names the story. So the trace leads from the origin to the file and back, and answers "where does it say it should be so?", for the product and for the way of working alike. Who checks names the evidence, not who built. When a story changes no standing spec, the place is its own target picture; when a bug changes files but no spec, the place is the one that already says what is right, and the evidence is the test its fixed tasks demand. Files from before this way of working have no entry; they get one when a story touches them.

## Example
`confirm: lending/period.md: src/loans/renew.ts, tests/loans/renew.test.ts`, in the Result of the story "Renew a loan", whose merge commit names the story.
