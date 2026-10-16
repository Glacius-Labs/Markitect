# Documentation maintenance

This guide routes maintainers through the repository's existing documentation conventions. It does not change product contracts or add a product-level documentation rule.

## Find the owner before writing

Start at the [documentation map](../README.md) and the nearest local map. Search for the document that already owns the subject, then update that owner instead of creating a parallel account.

| Subject | Canonical owner |
|---|---|
| Product thesis and human/agent responsibilities | [Vision](../vision.md) |
| Technical behavior and boundaries | [Architecture](../architecture.md) and the relevant current workflow or operations guide |
| Current source scope and future implementation | [Implementation roadmap](../implementation-plan.md) |
| Delivery scopes, dependencies and reservations | [Delivery plan](../implementation-plan.md); chats/PRs own candidate evidence |
| Contribution and validation commands | [CONTRIBUTING.md](../../CONTRIBUTING.md) |
| Product-specific future design records | [Design map](../design/README.md) |
| Fixed source, adoption, or method evidence | [Validation map](../validation/README.md) and the individual report |
| Release asset claims | [Production assessment](../production-assessment.md) |

Use the [Project World design map](../design/project-world/README.md), [Research map](../research/README.md), and [Strategy map](../strategy/README.md) to keep dated proposals and supplied research in their stated role. A proposal or assessment does not become product behavior merely because it is documented.

## Maintain navigation and evidence boundaries

When adding a document or directory, add a useful link to its local map and, when it creates a new route, to the parent map. Prefer links to the owner over copied status summaries. Keep the document's source, date, release, or experiment scope visible where it affects how readers should interpret the content. A dated report remains tied to its recorded inputs and conclusions; use the current owner document for later status.

Markdown links help readers navigate but do not declare artifact or semantic dependencies. When a path changes, inspect its Markdown links and anchors, repository entrypoints, and model Artifact paths that name it. Generated views have canonical sources; update the source and regenerate the view as described in [CONTRIBUTING.md](../../CONTRIBUTING.md).

The old Project documentation-router feature is preserved in the immutable [v0.14.1 release](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1). This guide describes where maintainers find and update current repository documentation.

## Check changed navigation

From the repository root, run:

```powershell
python -B scripts/check-docs.py
python -B scripts/check-docs.py docs/development/modules.md
```

The default corpus is the maintained entry documents listed in [the checker](../../scripts/check-docs.py): root guidance, current workflow and architecture guides, and category maps. Explicit paths check additional revised pages. It checks local targets and heading/explicit HTML anchors, ignores external URLs and code examples, and does not evaluate prose or infer dependencies. Historical reports and generated views are outside the default source corpus; links into them still have their destinations checked. Review changed historical pages explicitly without rewriting their evidence to satisfy a current-status convention.

Navigation maps and checker scripts are owned through the repository's full-coverage project model. Keep Artifact path changes in their owning Manager. [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change) owns the validation commands.
