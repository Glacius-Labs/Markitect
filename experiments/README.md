# Experiments and preserved evaluations

This directory holds bounded exercise harnesses and their evidence. Each exercise defines its own source, runtime, input, authority and result limits. Use the [validation map](../docs/validation/README.md) to locate recorded outcomes and the historical [backlog](https://github.com/Glacius-Labs/Markitect/blob/bc4b49e76f9b9e4b3ec98033a0b8c73d5013b310/docs/work-items/backlog.yaml) for status at the cited previous Main. The [delivery plan](../docs/implementation-plan.md) and assigned chats/PRs own current work.

The deterministic [Shop example](../examples/project-world/README.md) and [Project workflow](../docs/project-workflow.md) are the starting points for ordinary use. An exercise description or retained runtime configuration does not start a new provider run.

## Case studies

| Exercise | Scope |
|---|---|
| [Case playground](case-playground/README.md) | One coding agent through a staged backlog in a fresh Linux container, Conventional versus Markitect, with a separate assessment per wave |

## Earlier bounded evaluations

These directories preserve earlier contracts and outcomes. Their commands require the exact source or release selected in the guide; their results are dated evidence.

| Exercise | Scope |
|---|---|
| [AGENTS.md comparison](agents-md-comparison/README.md) | Matched task observations and a frozen independent evaluation |
| [MCP read-only pilot](mcp-pilot/README.md) | Earlier read-only tool protocol; separate from current product MCP |
| [Projection-first](projection-first/README.md) | Bounded materialization and candidate checks |
| [Policy-failure analysis](policy-failure-analysis/README.md) | Explicit read-only analysis replay |
| [Real-project adoption](real-project-adoption/README.md) | Selected project evidence and offline replay |

Preserve original logs, archives, result labels and workload bytes. Keep private provider transcripts outside public reports. New local outputs use the exercise's explicit output directory; do not overwrite a recorded attempt. [Measurement](../docs/measurement.md) owns comparison procedure, and the [release benchmark](../benchmark/README.md) owns immutable CLI workload versions. Technical results, comparative benefit and human acceptance remain distinct.
