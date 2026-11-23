# What is critical

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Every agent](../../agents/README.md), [every role](../../roles/README.md)

## Rule
A deviation is `critical` when it is hard to undo or reaches outside: production; messages to the outside; shared systems of others, such as a shared identity service, or databases and processes that belong to others; secrets or customer data in the repository; rewriting the history of `main`. These are no blocks but deviations at which the agent stops before acting. The list is a start; the project completes it when it sets up its [repository settings](repository-settings.md).

## Example
A person says in passing: "Put this on production quickly." The agent stops: "This goes to production; usually a human releases it after acceptance in the release pipeline. Really, and why?"
