# MCP evaluation (2026-10-01)

## Decision

The authorized bounded assessment is complete with the available evidence:
keep MCP experimental and retain the supported CLI as the installation path.
The adapter is not included in published Markitect binaries. A future product
decision requires repeated comparable tasks and a completed second-client
comparison; these are deferred criteria, not hidden completion claims.

Codex CLI 0.130.0 completed the same three read-only queries through CLI and MCP
on two public synthetic fixtures. All six direct YAML payload comparisons
were byte-identical. The documentation answers were supported; the package
CLI answer added an unsupported bindings claim and the MCP answer omitted
the Project context entry. Single-run times and reported token counters
cannot establish a speed or savings trend.

Claude Code 2.1.233 connected but could not run model tasks because its OAuth
session expired. At the user's explicit direction this remains documented
as blocked; the present assessment does not depend on repairing authentication.

## Evidence and reproducibility

- [Client report](../experiments/mcp-pilot/evaluation.md) owns the exact fixed
  inputs, client versions, setup failures, counters, answer review and limits.
- [Predeclared plan](../experiments/mcp-pilot/evaluation-plan.yaml) and
  [observed results](../experiments/mcp-pilot/evaluation-results.yaml) retain the
  original experiment. Historical snapshots and measured binary identities
  are not relabeled as the integrated candidate.
- [Prototype setup](../experiments/mcp-pilot/README.md) describes fixed startup
  repository/SHA inputs, allowlisted queries, protocol versions and boundaries.
- [Transport harness](../scripts/mcp-evaluation/run-transport.ps1) replays
  find, explain, and context against the onboarding fixture and compares
  CLI/MCP payloads, rejecting unknown tools and extra arguments. It is a direct
  protocol test, not a model task or a second-client comparison.
- [Onboarding](onboarding.md) is a separate scripted CLI workflow replay that
  includes writes and verification. MCP exposes no write or verify operation.

Integrated source preserves protocol negotiation for 2025-06-18 and 2025-11-25,
literal pinned arguments, inherited Git environment sanitization, lifecycle
checks and advisory read-only annotations. Unit and hosted Windows/Linux gates
check the source candidate; historical model runs are not rerun by those gates.

MCP may remove shell command construction after setup, but adds protocol and
permission setup. It changes the interface, not Markitect's deterministic
analysis or project ownership. Neither interface proves complete answers,
semantic correctness or human acceptance.
