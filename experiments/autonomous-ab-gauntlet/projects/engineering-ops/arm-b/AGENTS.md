# Relay engineering guidance

Read .agents/skills/engineering-operations/SKILL.md for canonical task guidance. Its Codex and Claude Skills come from the same Markitect Skill. Read docs/engineering/architecture.md and the applicable operational runbook for implementation context.

During editing run markitect check --repo . plus the configured hook and artifact helpers. After committing a candidate, run markitect verify --repo . --revision CANDIDATE_SHA. CI and the harness provide the pinned Markitect 0.13.0 CLI and helpers on PATH. Do not edit excluded parser fixtures under testdata/vendor-snapshots/.
