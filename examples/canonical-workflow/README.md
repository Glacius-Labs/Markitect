# Canonical workflow projection example

This fixture keeps workflow meaning in project-owned Rule, Process, Responsibility, and Gate Definitions. Three explicit Foundation Projections select those same four Definitions independently for Codex, Claude, and Markdown. Each target has its own policies; provider output is a reviewable representation, while the canonical Definitions and their Schema remain authoritative.

The Codex and Claude projections select `agent-rules-codex` and `agent-rules-claude` through separate installable manifests, `markitect-agent-rules-codex` and `markitect-agent-rules-claude`. The Go implementation remains a single cohesive Agent Rules module; these manifests separately register the installable projection capabilities required by the Host. Their module-owned filenames are `AGENTS.md` and `CLAUDE.md` beneath their declared repository-relative target prefixes. The Markdown projection uses the existing Markdown projector package.
