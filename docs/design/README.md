# Design

Design documents. Earlier design records are not kept on Main; the [archive index](../archive.md) lists them. For current product behavior use the [model language guide](../guides/model-language.md), the [architecture](../architecture.md) and the [project workflow](../guides/project-workflow.md); for source plans use the [delivery plan](../implementation-plan.md).

| Document | Owns |
|---|---|
| [Long-lived Managers](long-lived-managers.md) | Proposal of 30 October 2026: one maker and one checker per working Manager for the whole Run, and plain integration by the host; built on a branch and compared with `main` before adoption |
| [Evaluation suite v2](evaluation-suite-v2.md) | How the evaluation suite becomes decision-grade: one sandbox, judges as plug-ins, the portfolio, counted series; increments E1 to E6 |
| [Canonical model track charter](model-track-charter.md) | Goal, workstreams and owner decisions of the track that makes the canonical model serve typed programming in prose |
| [Markitect Desktop](markitect-desktop.md) | Proposal of 31 October 2026 for DEC-089: the app over the CLI that users know as Markitect, its areas, architecture and slices D1 to D7; nothing built |
| [Outside Findings](outside-findings.md) | Proposed concept 1.0 amendment, superseded before adoption by DEC-082's one Finding input |

Superseded by [concept 2.0](../concepts/v2-concept.md) and its T1 build, kept for their reasoning:

| Document | Recorded |
|---|---|
| [Language contract](language-contract.md) | The normative specification of canonical language v1: typed YAML Definitions with Core 1.2.0 |
| [Diagnostic contract](diagnostic-contract.md) | Diagnostic coordinates of the v1 language |
| [Kind interfaces](kind-interfaces.md) | Kind interfaces and the `valueType` vocabulary of Core 1.2.0 |
| [Standard Libraries](standard-libraries.md) | The v1 standard Libraries `values`, `practice`, `data`, `api` and `architecture` |
| [Modeling guide](modeling-guide.md) | What a well-modeled project contained in the v1 language |
| [Model quality criteria](model-quality-criteria.md) | Hard rules, review criteria and signals for v1 models |
| [Manager context](manager-context.md) | What each V1 Manager received as context |
| [Linux runner](linux-runner.md) | Apply in V1's Linux runner image (slice L1), removed with V1's Run host |
