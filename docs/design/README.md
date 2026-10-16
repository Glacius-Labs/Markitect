# Design records

This directory contains decisions, proposals, assessments, experiments, and design evidence from different stages of Markitect. A file's presence here does not make it a supported contract or current implementation. For current product behavior use [Architecture](../architecture.md), the [Project workflow](../project-workflow.md), and [Usage](../usage.md); for source plans use the [roadmap](../implementation-plan.md). Each record owns its own date, scope, and stated status.

The [Project World package](project-world/README.md) has its own reading order and status notes. Read that map before interpreting its individual design documents or shop artifacts.

## Product direction and work shape

- [Canonical model v1](canonical-model/minimal-language/language-contract.md) — completed language design baseline, confirmed on 16 October 2026; production Go implementation and broader conformance evidence remain separate work.
- [Canonical model overview](canonical-model/README.md) — the completed baseline, evidence boundaries, working vocabulary and historical sources.
- [Manager realization north star](canonical-model/manager-realization-north-star.md) — Package Managers, configurable teams, artifact registration and context variants from the owner discussion of 16 October 2026.
- [Earlier model-design consolidation](canonical-model/consolidation.md) — chronology from 15 October 2026 and remaining product questions outside the completed language scope.
- [Grundmodell und Bibliotheksentwurf](product-model-refinement/README.md) — typisierte Prosa, dokumentierte Designentscheidungen und Sprachkandidaten.
- [Standalone Tasks project documentation](task-management/README.md) — fachliche Gegenprobe als eigenständige Projektspezifikation; Anwendung noch nicht implementiert.
- [Historical Tasks modeling sketch](task-management-model/README.md) — früherer, abgelöster Modellierungsversuch.
- [Domain-language pressure test](domain-language-pressure.md)
- [Goal-led modeling proposal](goal-led-modeling.md)
- [Product-vision alignment assessment](product-vision-alignment.md)
- [Current product purpose over inherited compatibility](project-simplification-and-api-direction-20261009.md) — dated 9 October 2026
- [Standard operating model](standard-operating-model.md)
- [Work-item comparison](work-item-comparison-20261009.md) — dated 9 October 2026

## Command and tool surface

- [Verb table](verb-table.md) — CLI-01 specification of the top-level verbs and MCP tools (DEC-017), dated 10 October 2026

## Architecture, ownership, and capability boundaries

- [Candidate dependency execution](candidate-dependency-execution.md)
- [Clean architecture consolidation decision](clean-architecture-consolidation.md)
- [Host write identity and partial publication](host-write-identity.md)
- [Managed artifact coverage](managed-artifact-coverage.md)
- [Nested Project output boundary](nested-project-output-boundary.md)

## Projection, verification, and assurance

- [Canonical projection reset](canonical-projection-reset.md)
- [Canonical reverse inference](canonical-reverse-inference.md)
- [Canonical verification reuse provenance](canonical-verification-refresh.md)
- [Canonical Verifier observation coverage](canonical-verifier-observation-coverage.md)
- [Projection assurance direction](projection-assurance-direction.md)
- [Projection evidence refresh](projection-evidence-refresh.md)
- [Projection-first reconciliation](projection-first-reconciliation.md)
- [Proof-readiness capabilities](proof-readiness-capabilities.md)
- [Resolved-target equality assessment](resolved-target-equality.md)
- [Scoped target exclusions](scoped-target-exclusions.md)

## Existing-project adoption

- [Zero-churn representation adoption](brownfield-representation-adoption.md) — source-alpha design
- [Copy Me evidence review and candidate queue](copy-me-evidence-review.md) — explicitly an earlier proposal
- [Existing-project preparation](init-adoption-preparation.md) — explicitly an earlier Init proposal
- [Selective adoption handoff](selective-adoption-handoff.md)

## Capability work and focused analysis

- [Capability skills and native project entrypoints](project-capability-skills-20261009.md) — dated 9 October 2026
- [Native Codex work for Markitect Managers](project-native-codex-work-20261009.md) — dated 9 October 2026
- [Read-only analysis of policy failures](policy-failure-analysis.md)
- [Software architecture language experiment](software-architecture-experiment.md) — explicitly an experiment
