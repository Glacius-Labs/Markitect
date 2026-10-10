# Survey: agent-facing files

AI-generated, read-only, 10 October 2026, against main `02c7e529`. Input for AGENT-01 to AGENT-03 and ARCH-07.

## In this repository

**Entry files:**
- `AGENTS.md` is the only entry file; there is no `CLAUDE.md` and no `.codex/`. Its first paragraph is the legacy v0.13 workflow (`check --revision BASE`, `context --namespace development --kind Skill --name engineering-change`). Later paragraphs describe the model-first product. Nothing says which commands to use for which task.
- `.markitect/README.md` and `markitect.yaml` describe the legacy layout only.

**Legacy skill chain:**
- Canonical sources: `.markitect/areas/development/engineering-change.skill.yaml` and two rules, `repository-boundaries` and `module-verification`.
- Generated from them:
  - near-empty link stubs in `.claude/skills/{authoring,engineering-change}/SKILL.md` and `.agents/skills/…`;
  - the readable view under `docs/markitect/**`, declared in `.markitect/modules/projections.config`.
- `repository-boundaries.rule.yaml` is one long paragraph that mixes architecture rules, history and mandates, with about 22 file inputs. Undeclared YAML under `docs/` is parsed as a model resource by the legacy compiler, so new YAML files there must be added as inputs.

**Product resources:**
- Embedded resources in `src/internal/host/embedded/resources/`: `skill-authoring`, three workflows, one rule and one text.
- The `markitect-first-change` workflow is the v0.13 flow. `docs/markitect-first.md` repeats it and points to v0.13.0.

## What Markitect generates for adopting projects

Source: `src/internal/host/projectonboarding/{render.go,capabilities.go}`, run through `markitect project onboard --provider codex|claude`, preview then write.

**Generated files:**
- **Shared:** `.markitect/workflows/model-first.md`.
- **Codex:** an `AGENTS.md` managed block, `.agents/skills/<capability>/SKILL.md` with references.
- **Claude:** `CLAUDE.md` and `.claude/skills/<capability>/…`.
- Custom content outside the managed blocks is preserved.

**Content:**
- Nine capability skills: init, extract, design, suggest, configure, implement, cleanup, verify, apply, check.
- The generated text names `project_*` MCP tools, `markitect project init`, `markitect project onboard`, `markitect project mcp` and client setup lines. All of these exist today, but every one contains the `project` noun.
- Runtime pins (Codex CLI 0.162.0, `gpt-6-luna` high, Claude Code 2.1.295) are written in prose, not taken from a data source.

## Dogfooding gap

The repository uses the legacy stack throughout:
- `markitect.yaml` with 7 areas;
- `markitect-artifacts.yaml`, about 2.6k lines;
- three module configs;
- the legacy skill and rules;
- the generated `docs/markitect/**`.

It has no `.markitect/project.yaml`. Markitect's own agents therefore never see the files Markitect generates for others.

## Recommendations

- **AGENT-01, now:**
  - Reduce `AGENTS.md` to a short router with one command path.
  - Add a `CLAUDE.md` that points to it.
  - Split the `repository-boundaries` rule so it holds rules only.
  - Move runtime pins into one data source, with a golden test.
  - List `mcp` in `project --help`.
- **After CLI-02:**
  - AGENT-02: re-render all generated text for the new verbs. Add a test that fails when generated text names a command or tool that does not exist.
  - ARCH-07: the dogfood migration.
