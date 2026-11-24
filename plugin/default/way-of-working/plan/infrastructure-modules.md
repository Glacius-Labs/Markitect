# A module for infrastructure

Type: [Rule](../../core/rule.md)

## Applies to
Every module that holds pipeline, environment, delivery or dependencies.

## Rule
Such a module connects to the code on `main`, which it builds, tests and delivers. It never leaves `main` undeliverable, and it never touches production. Whoever is responsible for the module usually plans there.
