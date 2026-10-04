# Relay engineering operations — Arm A

Arm A uses conventional project-owned guidance and offline checks. Begin with [`AGENTS.md`](AGENTS.md), then read [`docs/engineering/architecture.md`](docs/engineering/architecture.md), [`docs/engineering/agent-rules.md`](docs/engineering/agent-rules.md), and the relevant operational runbook. The Python policy checker is a required local and CI gate.

The checker validates declared literal facts: provider entrypoint parity, exact tracked hook digest and owner inventory, named workflow scalar values, and managed-path coverage. It does not parse Go semantics or execute GitHub Actions. Go tests remain the source-level checks.
