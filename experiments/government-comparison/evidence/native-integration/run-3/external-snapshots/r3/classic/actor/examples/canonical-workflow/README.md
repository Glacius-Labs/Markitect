# Canonical workflow projection example

This fixture keeps workflow meaning in project-owned Rule, Process, Responsibility, and Gate Definitions. Five explicit Foundation Projections select those same four Definitions independently for Codex, Claude, Markdown, Git hooks, and Azure Pipelines. Each target has its own policy per selected Kind; provider output is a reviewable representation, while the canonical Definitions and their Schema remain authoritative.

The Codex and Claude projections select agent-rules-codex and agent-rules-claude through separate installable manifests, markitect-agent-rules-codex and markitect-agent-rules-claude. Their module-owned files remain AGENTS.md and CLAUDE.md. Markdown uses the existing Markdown Projection Module. The Git hooks Module writes .githooks/pre-commit; the Azure Pipelines Module writes ci/azure/azure-pipelines.yml. Those Projection Modules consume the same explicitly supplied canonical-workflow-check command without embedding project check names in their target implementation.

The public project-owned check source is check/main.go. Its command checks assertions in the canonical Rule, Process, Responsibility, and Gate resources; it defines this example's check contract rather than a universal CI language.
