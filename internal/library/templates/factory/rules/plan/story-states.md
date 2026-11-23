# The states of a story

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Story](../../../../packages/factory/story.md)

## Rule
A story has five states, named neutrally, since a story may also be an analysis:

| State | Means | Who acts |
|---|---|---|
| `draft` | created, not planned yet, or set back by an aborted build | the Architect with a person; a story with fixed tasks is created `ready` and comes here only after an abort |
| `ready` | no human has anything left to conceive | waits to be started |
| `in progress` | claimed, someone works on it | Builder and Reviewer, an Analyst, or a person |
| `acceptance` | the result is on `main` and waits to be accepted | the person who accepts |
| `done` | finished, or given up ([reopen after a finding](../planning/reopen-after-finding.md)) | nobody |

A story with nothing to accept goes from `in progress` straight to `done`. An aborted build sets the story back to `draft` ([abort](../building/abort.md)).
