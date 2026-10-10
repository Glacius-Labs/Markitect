# Quality criteria

The CLI behavior is specified in README.md and BACKLOG.md. All requirements are visible.

Finished work meets these criteria:

- **Behavior.** Every released requirement works through real CLI calls on fresh data,
  and a second process on the same data sees the committed result.
- **Errors.** Invalid input and business conflicts return an error object with exit
  code 2 and leave stored data unchanged.
- **No regressions.** Existing commands, output shapes and stored data keep working.
- **Consistency.** Code, tests, documentation and configuration agree with each other
  and with the requirements.
- **Maintainability.** The code is readable and easy to change.
- **Tests and docs.** Tests are meaningful and cover the risky cases. The README explains
  how to run and test the tool, how failures and stored data behave, and its limits.
- **Integration.** Finished work is merged to `main`. A broken required behavior or work
  that never reached `main` means the item is not done.

The final rename (the last wave) must change active behavior, interfaces, code, tests,
docs and configuration consistently, including any project model and generated views.
Keep the compatibility aliases the backlog asks for and keep old stored data readable.
Historic notes and deliberate aliases are fine. A text search alone does not show that a
rename is complete; behavior and references count.

Each wave is reviewed against its released items, and the end state is reviewed as a
whole by someone who did not do the work. Passing tests are evidence, not acceptance.
