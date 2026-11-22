# What is critical

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Every agent](../../agents/README.md), [every role](../../roles/README.md)

## Rule
A deviation is `critical` when it is hard to undo or reaches outside: production; messages to the outside; shared systems of others, such as a shared identity service, or databases and processes that belong to others; secrets or customer data in the repository; rewriting the history of `main`. These are no blocks but deviations at which the agent stops before acting. The repository's settings grant what is usual: reading, writing in one's own working folder, building and testing, committing, pushing one's own branch, bringing a reviewed story to `main`. What is critical and recognizable as a command stands on "ask" there, so an agent running alone gets no further and aborts. What is an ordinary file change, such as a secret in a commit, the settings do not catch: the agent recognizes it by its document, and the observer [deviations](../../observers/deviations.md) catches what slips through. The project completes the list when it sets up its settings.

## Example
A Developer says in passing: "Put this on production quickly." The agent stops: "This goes to production; usually a human releases it after acceptance in the release pipeline. Really, and why?"
