# Owner decisions bind the whole run (18 October 2026)

Source of [DEC-053](../register.md#dec-053-owner-decisions-bind-the-whole-run).

## Evidence

Bound run R2-A1-09 (Main 011e2122) stopped Waiting after 6264 s. It had four open owner questions about two topics:

- **Unknown argument fields:** `catalog` and `library` asked how to classify them.
- **Due dates after 9999-12-31:** the root, `lending` and `library` asked whether such a borrow is refused.

The run waited 54 minutes until the answers were recorded. The same topic came from several Managers, because an answer reached only the task that asked. Later runs of the same model would ask the same questions again.

The owner decided both topics:

- Unknown argument fields return `invalidRequest`.
- A borrow whose due date would fall after 9999-12-31 is refused atomically.

## Owner instruction

> Das mit den Fragen darf nicht wieder passieren. Wir wollen nicht, dass die Sessions so lange einfach nur hängen. Dafür brauchen wir eine Lösung

The Architect proposed three parts:

1. Owner decisions bind the whole run and can be carried into later runs.
2. Reading conflicts go to an arbitration step instead of the owner.
3. The run never waits silently.

The owner answered:

> Setze das gerne so um

This decision covers the first part.
