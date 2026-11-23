# The repository settings grant the usual

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[Every agent](../../agents/README.md)

## Rule
The repository's settings refuse nothing and grant what is usual: reading, writing in one's own working folder, building and testing, committing, pushing one's own branch, bringing a reviewed story to `main`. What is [critical](critical.md) and recognizable as a command stands on "ask" there, so an agent running alone gets no further and aborts. What is an ordinary file change, such as a secret in a commit, the settings do not catch: the agent recognizes it by its document, and the observer [deviations](../../observers/deviations.md) catches what slips through. The exact list arises when the project sets up its settings.
