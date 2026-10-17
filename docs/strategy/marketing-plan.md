# Marketing working plan

Prepared 17 October 2026 from Main `f23eaa2072226d905a8d79b8d6c20f419dfdc136`. Commission: take responsibility for marketing and repository care, understand the documents and new concept, improve the README, and plan pitches and marketing artifacts while development continues.

This is an editorial recommendation and draft copy, not an adopted product decision, implementation queue or launch announcement. [Vision](../vision.md) owns the product thesis, the [concept register](../concepts/register.md) owns decisions, the [delivery plan](../implementation-plan.md) owns development scope, and [documentation maintenance](../development/documentation.md) owns repository documentation care. Update this plan when evidence or audience learning changes it; campaign progress belongs in its actual delivery record.

## Positioning

**Working line:** Define the project. Keep its implementation aligned.

**Category:** Model-first development with delegated realization.

**Problem:** A coding agent can complete the requested feature while missing an existing architecture rule, another area's behavior, a test or a documentation obligation. Repeated consolidation leaves the owner doing the synchronization work again.

**Product idea:** Maintain one canonical description of the intended project. Give its realization explicit responsibility, context, independent review and integration. Keep code, tests, configuration and documentation accountable to that intent through change.

**Desired value:** More attention for design and decisions; fewer forgotten obligations and contradictory realizations. These are target benefits, not measured outcomes. The owner's [origin account](../concepts/origin-and-rationale.md) supports the problem narrative, not a market-size or prevalence claim.

Lead with the user's problem, then explain the mechanism with one concrete change. Introduce Packages, Managers and the Host only as needed to explain that change. The model is the continuing specification; a completed feature does not retire its existing obligations.

### Organizing idea behind the story

On 17 October 2026 the owner supplied the [conceptual influences](../concepts/sources/conceptual-influences-20261017.md) behind Markitect. Their single explanatory owner is [Origin and rationale](../concepts/origin-and-rationale.md#conceptual-influences). Our marketing interpretation is: **Make delegated AI work trustworthy through explicit intent, accountable responsibility and an inspectable process.** This is a proposed story, not a claim that trust has been measured.

The public explanation can unfold in four steps:

1. Describe the desired project world once in a canonical model.
2. Divide its realization into responsibilities with focused context, local authority and escalation.
3. Require independent judgment of the parts and their integration: the four-eyes principle at both scopes.
4. Inspect the actual result and evidence before accepting it.

"Trust the Process" names the aim of putting control into structure and procedure. Show what that process requires and what it actually observed. Type checks catch expressible structural errors early; semantic judgments still need assessment. Lower model cost is a hypothesis about focused work, evaluated over the full process. Stakeholders, incentives and multiple perspectives enrich the rationale; their mention introduces no reward system or additional reviewer institution. For a short pitch, translate these roots into a user's experience rather than reciting every contributing concept.

### First audience hypothesis

Start with hands-on technical leads, maintainers and small teams already using coding agents on repositories with continuing architecture, testing and documentation obligations. Their recurring reconciliation work is the proposed reason to try Markitect. This audience and its demand have not been validated.

| Audience | Useful opening | Evidence they will need |
| --- | --- | --- |
| Technical lead or maintainer | Keep the architecture and existing obligations effective as agents change the project. | One change across responsibilities, visible review and integration, and an honest account of model upkeep. |
| Developer already using agents | Describe the intended change once and inspect what was actually realized. | A reproducible example, clear decisions needing attention and actual setup effort. |
| Contributor interested in agent engineering | Help build an inspectable process for delegated realization. | Current source commands, a bounded contribution and the relevant design owner. |

For now the useful invitation is to inspect the model, run preparation and give concrete feedback or contribute. A later invitation can ask people to reproduce a demonstrated Apply. Avoid making production adoption the first call to action during the rebuild.

### Adjacent products

Primary sources checked on 17 October 2026:

| Source | What its own documentation says | Implication for our positioning |
| --- | --- | --- |
| [GitHub Spec Kit](https://github.com/github/spec-kit) | Structured processes, reusable templates and documented outcomes for coding agents, including specification-driven development. | Specification-first work and agent workflow structure are shared ground. |
| [Kiro specs](https://kiro.dev/docs/specs/) | Specs connect requirements, design and implementation tasks. | Describing software before implementation is already a recognizable approach. |

Our editorial inference is to make Markitect's intended combination concrete: a continuing canonical project model, exact artifact accountability, Package responsibilities and review of both realization and integration. Explain that combination through observable behavior. These sources establish neither uniqueness nor superiority; this is a small positioning check, not an exhaustive competitor assessment or feature comparison.

## Pitch drafts

All drafts below describe the direction and explicitly name the development stage. They can be revised without changing the product concept.

### One sentence

Markitect is being built to turn a canonical project model into coordinated AI work that keeps code, tests and documentation aligned with your intent.

### Thirty seconds

Coding agents can implement a change and still miss a rule, a related test or a documentation update. Markitect starts with a canonical model of what your project should be. It is being built to organize realization through clear responsibilities, focused context, delegation and escalation. Independent reviewers assess both each area's work and how the areas fit together. The goal is to keep the project consistent while you focus on design and decisions. Model preparation works today; recursive realization and Apply are still being built.

### Two minutes

The hard part of using coding agents on a growing project is keeping the whole project faithful to its intent through many changes. A feature can work while a rule, a test or the usage guide falls behind. The owner then has to rediscover what should have changed.

Markitect starts with one canonical model of the desired project: its concepts, behavior, architecture and rules. Code, tests, configuration and documentation realize that model. Its structure makes ownership and explicit relationships inspectable before work starts.

The design draws on architectural decomposition and organized collaboration: clear responsibilities, delegation and escalation, abstraction, focused context and the four-eyes principle. Each model Package has a Manager. A deterministic Host supplies explicit context and propagates the same operation unchanged to every local Manager. Managers assess their responsibilities, make needed changes, integrate their children's results and return evidence. Fresh reviewers assess local change or no-change and integration. Required checks and a final evidence check support guarded publication. A missing decision or incomplete result stays visible to the owner.

Consider a library whose borrowing limit changes from five books to three. The intended demonstration applies the first model, changes that one policy value and applies the second model against the actual first result. Code, tests and documentation must follow the new limit together, with independent expectations checking the result.

That demonstration is the current development target. Source already supports model compilation, complete file accounting, context preparation and a guarded readable model view. Recursive realization and Apply are still being built. We want to learn whether the method preserves intent more reliably and reduces routine supervision after counting modeling, review and recovery too.

## Claims and their basis

| Kind of statement | Source to consult before using it | How to phrase it |
| --- | --- | --- |
| Product purpose | [Vision](../vision.md#markitect-in-brief) and [origin](../concepts/origin-and-rationale.md) | What Markitect is being built to do. |
| Current executable behavior | [Workflow](../project-workflow.md), [Architecture](../architecture.md) and the exact source being shown | What the demonstrated revision actually does. |
| Manager and Apply behavior | [Apply concept](../design/canonical-model/manager-apply-concept.md) and implementation evidence | Target behavior until demonstrated by the actual product. |
| Published installation | [Production assessment](../production-assessment.md) and the fixed release | Name the release and its product generation. |
| Reliability, quality, attention or cost benefit | [Measurement](../measurement.md) and the relevant source-bound result | Hypothesis until measured; report setup and upkeep as well as execution. |

At the preparation baseline, Apply returns unavailable. Container and session foundations or a provider handshake do not establish recursive product execution. A generated project view is a readable model, not proof that the repository fulfills it.

Use language such as "aims to", "being built" and "this run showed" at the appropriate scope. A small successful demonstration can establish that workflow for its inputs. It cannot establish general correctness, zero drift, lower total cost or unattended acceptance. Future Government, cockpit and operational-doctor ideas stay in their [existing concept records](../concepts/register.md), outside the first Apply story.

## Artifact sequence

The sequence below is a recommendation about marketing material, not an assignment to the development teams or a second feature roadmap.

| Priority and trigger | Artifact | What it should contain | Reviewable completion |
| --- | --- | --- | --- |
| Now | README opening | Problem, product, target workflow, honest stage and a clear route to runnable source. | A newcomer can explain the purpose and distinguish source from the earlier release. |
| Now | Pitch set in this document | One sentence, thirty seconds and two minutes using the same product story. | Each pitch makes the user benefit and current limitation understandable. |
| Next editorial artifact | One-page explainer | Problem, library change, model-to-realization flow, present stage and invitation to inspect. | Every behavior claim points to its owner or demonstrated source. |
| Next editorial artifact | Small workflow diagram | Human intent → canonical model → Manager work/review → integrated result. Mark target steps visibly. | A reader can distinguish model authorship, realization and acceptance. |
| After real M1/M2 evidence | Reproducible demo and short recording | Frozen product/profile, M1 result, one model-only M2 edit, actual second result, relevant reviews and failures. | Another developer can follow the published steps; edited footage does not hide intervention. |
| After that demo is reproducible | Technical article and launch draft | Concrete before/after behavior, how it works, what failed, scope and setup cost. | Links reach the actual example and evidence; the invitation matches what people can run. |
| After repeated use | Case study | Starting conditions, continuing changes, quality, omissions, interventions and total upkeep. | Conclusions match the evidence and comparison method. |

Prepare the explainer in Markdown using existing examples before commissioning a site, deck, video production or brand redesign. A shared story and usable demonstration have more immediate value than many formats that need to be synchronized.

### Proposed demo story

1. Show the model's loan limit and the first realized library's observable behavior.
2. Change only `maxOpenLoans` from five to three in the model, using the established [Playground contract](../design/canonical-model/manager-apply-playground.md).
3. Show the second actual Apply, its responsibilities, review and integration. Inspect the changed code, tests and guide.
4. Demonstrate the new boundary and an existing obligation that still holds, using the independent evaluation.
5. Show the result, any owner intervention and the remaining limits. Link the full record behind the short recording.

Use the existing [acceptance gates](../design/canonical-model/manager-apply-implementation-plan.md) and [evaluation contract](../design/canonical-model/library-example/evaluation-contract.md); this story adds no test requirement or permission to run providers. Until real evidence exists, label a walkthrough as an illustration.

## Promotion recommendation

Start with technical explanation and specific feedback. The channel order is a hypothesis about where useful learning may come from, not evidence of demand.

| Stage | Proposed channel or format | Purpose and useful signal |
| --- | --- | --- |
| During the rebuild | Repository README and a short development article when there is a meaningful result | Help developers understand the idea and give concrete feedback on a model, command or limitation. |
| Once the example can be reproduced | A technical walkthrough shared in relevant agent-development communities and the owner's professional network | Find people who can identify the same problem and attempt the example. Check each community's posting rules before publishing. |
| Once people can actually try the product | Consider Show HN with a direct link to runnable work | Obtain technical questions and reproduction feedback. [Show HN guidelines](https://news.ycombinator.com/showhn.html) require something people can try; a landing page alone is unsuitable. |
| Once onboarding survives outside use | A broader launch, additional channels or a dedicated website | Expand a working invitation instead of sending traffic into unfinished onboarding. |

This plan drafts material only. Publishing posts, contacting people and spending money need their own actual delivery request. No campaign or recurring automation is created here.

Record what feedback changes: can readers explain the product, identify a matching problem and complete the offered workflow? Keep failed starts and model-maintenance objections. Stars, views and package counts show attention or output; they do not demonstrate product benefit. Any comparative study follows the existing Measurement procedure and its own authority.

## Reading basis and maintenance observations

The baseline contains 402 tracked Markdown files, including 285 under `docs/`; examples, fixtures, experiments and historical sources contribute to that count. The initial pass inventoried document paths, titles and opening scope notes. It did not semantically audit every historical source or establish that every document is current.

The substantive reading focused on the vision and common misreadings, concept register and origin, Manager/Apply concept and library flow, design and evidence maps, source/release boundaries, contributor instructions and adopted working method. Technical contracts are consulted at their owning boundary; the document count is not an audit-completion claim.

| Observation at the fixed baseline | Action or proposed follow-up | Benefit and tradeoff |
| --- | --- | --- |
| README moves into commands before explaining the problem and realization workflow. | Improve the opening and reader routes in this documentation slice. | Easier first understanding; retain a short explicit stage statement so the direction is not mistaken for shipped behavior. |
| The strategy map mixes historical release summaries into an archive introduction. | Replace those summaries with current-owner links while preserving all original source documents. | Less stale guidance; readers follow a link for historical detail. |
| Some category-map summaries still call the production v1 compiler future work. | Reconcile those sentences with current Architecture in a focused follow-up, checking active PR overlap first. | Removes a current-status contradiction without rewriting source-bound design history. |
| Many older short guides already route to immutable release documentation. | Check actual incoming links and present purpose before retaining or deleting each router. | Can reduce navigation clutter; deletion may break old links, so file age alone is insufficient. |
| Older open documentation PRs overlap subjects already present on Main. | Compare exact candidate content with Main and ask the owning delivery to reconcile or close superseded work. | Reduces stale queues; open PR age does not establish that its content is disposable. |
| Many worktrees and branches belong to other sessions. | Identify their owner and unmerged or dirty work before any cleanup. | Preserves active work; a listing alone cannot establish safe deletion. |

Routine care follows [the maintenance guide](../development/documentation.md#retire-obsolete-guidance-without-losing-evidence). Actual source/release status continues to live in its existing owner, so this dated intake does not become a live status board.
