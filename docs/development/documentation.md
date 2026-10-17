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
| Release asset claims | [Production assessment](../history/distribution/production-assessment.md) |

Use the [Project World design map](../design/project-world/README.md), [Research map](../research/README.md), and [Strategy map](../strategy/README.md) to keep dated proposals and supplied research in their stated role. A proposal or assessment does not become product behavior merely because it is documented.

## Maintain navigation and evidence boundaries

When adding a document or directory, add a useful link to its local map and, when it creates a new route, to the parent map. Prefer links to the owner over copied status summaries. Keep the document's source, date, release, or experiment scope visible where it affects how readers should interpret the content. A dated report remains tied to its recorded inputs and conclusions; use the current owner document for later status.

Markdown links help readers navigate but do not declare artifact or semantic dependencies. When a path changes, inspect its Markdown links and anchors, repository entrypoints, and model Artifact paths that name it. Generated views have canonical sources; update the source and regenerate the view as described in [CONTRIBUTING.md](../../CONTRIBUTING.md).

The old Project documentation-router feature is preserved in the immutable [v0.14.1 release](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1). This guide describes where maintainers find and update current repository documentation.

## Retire obsolete guidance without losing evidence

Keep current guidance useful as the product changes. Classify a page by its present purpose before deciding whether to update, merge, move or delete it. Age alone does not establish that it is obsolete.

| Finding | Maintenance action |
| --- | --- |
| A current guide names removed behavior or an old command | Correct it against the actual integrated source and its owning contract. Update the examples and navigation in the same slice. |
| Two current pages repeat a changing rule or status | Keep the existing canonical owner and replace the duplicate account with a useful link. |
| A replaced design or assessment still explains a decision or failed attempt | Keep its date, source and outcome visible; route it through the appropriate historical, design or validation map. |
| A file has no current purpose and its useful content already has an owner | Remove it with its exact registry entry after checking incoming links, anchors and source/tool references. |
| A release instruction or generated view needs changing | Use the owning generator or release workflow; edit the canonical source where applicable. |
| A branch, PR, worktree or local output looks abandoned | Inspect its exact state, owner, unmerged work and active assignment before cleanup. A stale-looking name is not evidence that another session has finished. |

Before a cleanup, fetch Main and verify the isolated checkout, branch, full SHA and local changes. Check current delivery reservations and relevant open PR paths. Separate a small documentation correction from a cross-package removal; the latter follows the delivery plan's ownership rules. Preserve dirty or unmerged work instead of sweeping shared directories. Never stash across worktrees.

After moving or removing maintained documentation, update its local map, actual incoming links and exact registry accounting together. Keep original owner statements and fixed evidence in their original scope; a replacement guide can link to them without repeating their old instructions. Use the checks below and the relevant contributor gates. Report concrete findings and candidate evidence in the actual chat or PR instead of maintaining another cleanup backlog or status mirror.

The [marketing working plan](../strategy/marketing-plan.md) owns pitch drafts and promotion recommendations. When a complete user workflow becomes available or its public instructions change, review the README's stage and invitation against the same evidence. A new package, passing check or design decision alone does not establish that workflow.

## Check changed navigation

From the repository root, run:

```powershell
python -B scripts/check-docs.py
python -B scripts/check-docs.py docs/development/modules.md
```

The default corpus is the maintained entry documents listed in [the checker](../../scripts/check-docs.py): root guidance, current workflow and architecture guides, and category maps. Explicit paths check additional revised pages. It checks local targets and heading/explicit HTML anchors, ignores external URLs and code examples, and does not evaluate prose or infer dependencies. Historical reports and generated views are outside the default source corpus; links into them still have their destinations checked. Review changed historical pages explicitly without rewriting their evidence to satisfy a current-status convention.

Navigation maps and checker scripts are owned through the repository's full-coverage project model. Keep Artifact path changes in their owning Manager. [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change) owns the validation commands.
