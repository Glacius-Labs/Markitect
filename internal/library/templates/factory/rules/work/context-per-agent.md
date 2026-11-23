# One context folder per kind of agent

Type: [Rule](../../../../packages/core/rule.md)

## Applies to
[The agents](../../agents/README.md), [the observers](../../observers/README.md)

## Rule
Every kind of agent has exactly one folder in `.markitect/context/`, named like its Agent document without `.md`, such as `build/` for the skill `markitect-build`; every observer is a kind of its own, with the folder `observe-NAME/`. The folder always holds the same: one report per session or run, and, where the agent continues from where it stopped, a `checkpoint.md`. Dropped material lies with the skill that drops it, in `context/capture/`, the raw material hidden below it in `.raw/`; the report on it lies in `context/observe-dropped-material/`.

## Example
`.markitect/context/build/2026-11-24-1432-renew-a-loan.md` and `.markitect/context/triage/checkpoint.md`.
