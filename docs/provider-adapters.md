# Provider adapters

Markitect renders Codex and Claude entrypoints only when `Project.spec.targets` selects them. `Skill` resources create `.agents/skills/<name>/SKILL.md` and `.claude/skills/<name>/SKILL.md`; `Agent` resources create `.codex/agents/<name>.toml` and `.claude/agents/<name>.md`. Canonical YAML supplies descriptions, text, explicit dependencies, and provider settings. Unsupported fields fail strict parsing. `render --write` changes owned outputs; `check` reports missing, changed, stale, retired, and unregistered outputs.

`spec.providerAdapters` optionally configures shared entrypoints and a strict inventory:

```yaml
spec:
    targets: [codex, claude]
    providerAdapters:
        inlineAgentText: true
        strictInventory: true
        agentContract: docs/agents/README.md
        roleRegister: docs/roles.md
        ruleSources:
            change-review:
                - docs/rules/change-review.md
                - docs/workflows/review.md
        retiredSkills: [old-review]
        retiredAgents: [old-reviewer]
```

`ruleSources` maps each Claude rule entrypoint to at least one ordered repository-relative Markdown source. These paths must exist; strict inventory requires every local canonical `Rule` companion view in a mapping when Claude is selected. The Project owns each aggregate rule output; individual `Agent` and `Skill` outputs have one resource owner. `agentContract` and `roleRegister` are explicit shared paths; role pointers are emitted only for selected targets. An Agent entrypoint links to `agentContract` whenever it is declared, whether its own text is inline or linked through the companion view. `inlineAgentText` copies canonical Agent text into selected provider outputs; without it, adapters route to the generated companion view. Strict inventory requires explicit Agent settings for each selected target and rejects provider files outside the generated set in the owned skills, agents, and rule directories. Retired names must be valid, unique, and absent from active resources; old files fail the inventory check. Remove those files deliberately in the same reviewed migration; `render` does not delete them automatically.

Markitect validates its output paths, source paths, declared settings, collisions, and drift. It does not infer mappings from prose links. A repository still owns its root `AGENTS.md` and `CLAUDE.md`, documentation navigation, hook policy, and other domain-specific checks. Generated provider prose is routing, not a second source of policy. A package's resources cannot generate host provider entrypoints directly; use a local wrapper.
