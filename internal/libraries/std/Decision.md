---
kind: Kind
---
# Decision

A Decision Definition records a choice the project made and must keep: the context, the options considered, what was decided and its consequences, like an architecture decision record.

- State the decision itself in one sentence first.
- Say what it binds: which artifacts must follow it, and what would contradict it.
- Its place says what it binds, as a Rule's does: at Root the whole project, in a Package that Package and those below it. A Definition whose text relies on it links it; the Decision does not link what it affects.
- A Decision that replaces an earlier one takes its place, and the earlier one goes: the spec says what holds now, and Git keeps what held before.
