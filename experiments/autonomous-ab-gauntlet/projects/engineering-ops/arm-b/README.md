# Relay engineering operations — Arm B

This project keeps normative agent rules in typed YAML resources under .markitect/areas/operations/. Markdown, Codex and Claude views are generated from those owners. Begin with AGENTS.md, the generated Skills, architecture.md, and the relevant operational runbook.

The harness and CI provide the pinned Markitect 0.13.0 CLI plus markitect-check-artifacts and markitect-check-modules on PATH. During editing, run markitect check and the working-tree helper commands; after committing a candidate, run markitect verify --repo . --revision CANDIDATE_SHA. It checks structure, explicit literals, and configured Go tests; it does not prove Go behavior beyond those tests or claim provider runtime success.
