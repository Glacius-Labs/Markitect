---
kind: Kind
---
# Command

A Command Definition describes one command of the project's command-line interface: how a user reaches a Feature from a shell.

- The title is the command as typed, such as `library borrow`.
- Link the [[std/Feature]] it exposes.
- Name its arguments and flags in backticks, with what each means and its default.
- Name what it prints on success, its output format when another program reads it, and its exit codes.
- Name every refusal a user can observe: the message or code and the exit status.
