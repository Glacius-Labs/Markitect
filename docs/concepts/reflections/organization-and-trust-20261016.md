# Trust through organized agent work

Date: 16 October 2026. Owner vision, observations and hypotheses from the Markitect outlook discussion. [Original owner statements](../sources/organization-and-trust-20261016.md) preserve the wording. Assistant interpretations are identified below; recording the discussion does not adopt them or establish a measured benefit.

The owner also supplied the [assistant response](../sources/organization-and-trust-assistant-20261016.md) for extraction. Its suggestions remain attributed to the assistant. The topic map below brings both contributions together; the existing concept register retains decision ownership.

## Extracted ideas and their status

| Idea | Origin and status | Detail |
| --- | --- | --- |
| Spend human attention on modeling, direction and important questions | Owner goal | [Desired working life](#the-working-life-the-owner-wants) |
| Compare with the actual story, implementation, AI review and merge workflow | Owner experience; assistant comparison framing | [Current workflow](#the-workflow-being-improved) |
| Distribute responsibility through delegation and escalation | Owner organizational analogy and hypothesis | [Trust mechanism](#organization-as-the-trust-mechanism) |
| Establish perspectives from enduring responsibilities and model obligations | Assistant interpretation of the owner's observation about focused attention | [Attention beyond the current task](#attention-beyond-the-story-and-diff) |
| Use configurable teams for focused work and different review perspectives | Owner follow-up idea; concrete assignments remain proposals | [Team configuration](#different-perspectives-through-configurable-teams) |
| Give an independent reviewer the actual requirements and result, with effective rework | Owner four-eyes direction; assistant operational interpretation | [Independent review](#four-eyes-and-correlated-mistakes) |
| Select capable models per role and evaluate total effort | Owner efficiency hypothesis; assistant evaluation proposal | [Whole-process evaluation](#model-capability-and-the-cost-of-the-whole-process) |
| Delegate local decisions with sufficient context and bounded authority | Owner observation and hypothesis; assistant boundary proposal | [Local decisions](#local-decisions-and-context), [decision boundaries](#local-authority-and-escalation-boundaries) |
| Preserve experience across runs | Assistant proposal, explicitly requested by the owner as an issue to retain | [Institutional memory](#experience-and-institutional-memory) |
| Treat justified no-change, objections and escalation as useful outcomes | Owner incentive analogy; assistant interpretation of success criteria | [Incentives and limits](#limits-of-the-analogy-and-future-autonomy) |
| Use reports for steering and review the process itself | Owner reporting direction; assistant oversight proposal | [Leadership and process oversight](#leadership-reports-and-process-oversight) |
| Earn enough trust to avoid routine human PR review | Owner goal; sufficient evidence remains open | [Autonomy and its limits](#limits-of-the-analogy-and-future-autonomy) |

## The working life the owner wants

The owner wants to spend time modeling, developing ideas and concepts, and setting rules. The organization should turn that intent into clean, consistent and considered implementation, develop features, find and fix bugs, and maintain project hygiene. The owner wants to watch ideas become reality, answer important questions and steer the work without continuously supervising implementation.

Reliability includes preserving earlier obligations as the project changes. The desired organization should keep the project understandable and orderly over time. Avoiding routine human PR review is an explicit goal, conditional on earning sufficient trust in the organized work.

This develops [VIS-001](../register.md#vis-001-cockpit-personal-ai-and-autonomous-project-organization) and the existing [target outcomes](../register.md#promises), especially attention moving to modeling and decisions, sustained consistency and dependable maintenance.

## The workflow being improved

The owner describes this workflow in their own work:

1. Plan a user story.
2. Ask Claude or Codex to implement it and create a PR.
3. Ask an agent to review the PR.
4. Ask an agent to watch findings and respond until the PR is ready.
5. A human code review is expected, but rarely happens in practice.
6. Merge to main.

Other work begins when someone notices a problem and instructs an agent. The owner observes that agents concentrate on a particular task and perspective; a large task can leave relevant aspects unnoticed. This is the owner's experience, not a measured account of every team or coding agent.

The comparison is therefore with this existing delegated workflow and its actual supervision, consistent with [DEC-002](../register.md#dec-002-trust-is-measured-against-todays-practice). Markitect's evaluation still needs a strong conventional workflow that is allowed to perform better.

## Organization as the trust mechanism

The owner compares a company or government to a software system: it has areas of responsibility, solves different subproblems, responds to its environment and pursues a goal. Leadership sets direction, delegates, receives reports and handles important escalations. Incentives and local discretion help avoid constant supervision.

Applied to agents, the hypothesis is that several focused workers, paired with independent reviewers and coordinated through management, can be more dependable and efficient than one very capable worker carrying many responsibilities. Different areas would examine work from different perspectives. Trust would come from the organization and method as well as the capabilities of its members.

The references to one worker replacing five, ten or fifteen people and to twenty perspectives are illustrative. They are not staffing ratios, benchmark results or required review counts. The owner's word “dümmere” expresses relative capability in this analogy; [DEC-007](../register.md#dec-007-more-economical-models-means-capable-mid-tier-models) retains the direction of capable mid-tier models.

This elaborates [CPT-001](../register.md#cpt-001-federalism) and the existing [organizational and model-tier hypotheses](../register.md#assumptions-and-hypotheses). The [Manager north star](../../design/canonical-model/manager-realization-north-star.md) and [Apply concept](../../design/canonical-model/manager-apply-concept.md) explore its realization. Government remains the separate model-change axis described in [DEC-004](../register.md#dec-004-the-manager-hierarchy-is-core-government-belongs-to-a-later-axis).

## Local decisions and context

The owner observes that people already explain local technical questions to an AI and commonly follow its recommendation. They consider missing context a major reason for poor recommendations. This supports their hypothesis that a Manager with sufficient context and clear authority could make many such decisions directly.

The intended result is local discretion with useful escalation. The owner should be asked about material questions rather than routinely choosing implementation details. Reporting should let them see whether work is progressing in the intended direction and intervene when needed.

## Assistant interpretation and open questions

The discussion suggests the following questions for later design and evaluation. These are assistant interpretations, not additional owner decisions.

Follow-up on 16 October 2026: the owner explicitly asked to retain these objections, especially experience, different perspectives and the four-eyes principle, and identified configurable teams as a way to realize the latter. The [follow-up source](../sources/organization-and-trust-20261016.md#retain-objections-and-team-design-questions) records that request. The elaborations below preserve design questions for the architect; they do not select a new team layout.

### Attention beyond the story and diff

The supplied response proposes a concrete mechanism for the owner's observation about concentration: an implementation prompt directs attention toward the requested feature, while a PR review directs attention toward the existing diff. An obligation absent from both may receive little attention.

An enduring responsibility can supply a different starting question. Review assignments would come from the relevant model and responsibilities, including obligations that the story did not repeat. A narrow assignment can still demand substantial reasoning: checking whether suspended members can borrow through any supported entry point is bounded but not trivial.

This is a hypothesis about organizing attention, not a guarantee that every required perspective is known. Missing model connections and incomplete context can still leave gaps. The intended improvement is to make perspective selection explicit and inspectable rather than relying on one agent to invent every useful angle during the task.

### Local authority and escalation boundaries

The assistant proposes four inputs for delegated local judgment: relevant context, goals and binding constraints, authority to choose within the area, and criteria for involving other areas or escalating. For example, choosing between equivalent internal implementations could stay local. Changing public behavior or imposing an obligation on another area requires coordination under the applicable model and authority.

The architect still needs to determine how these boundaries are expressed and checked, how an agent obtains missing context and how uncertainty reaches an empowered recipient. The owner's observation that AI recommendations are commonly followed motivates this delegation; it does not establish that every local recommendation is correct.

### Leadership reports and process oversight

The owner wants to issue direction, receive regular reports and react to material questions or signs that work is going wrong. Upward information should therefore help steering: outcomes, unresolved decisions, relevant failures and reasons to intervene. Detailed evidence remains available for investigation.

The assistant proposes that reduced routine code review can be accompanied by occasional targeted inspection of the process and its rules. This would examine whether responsibilities, review questions and escalation boundaries still work. No reporting frequency, audit schedule or autonomous merge permission was selected in this discussion.

### Model capability and the cost of the whole process

The owner's efficiency hypothesis concerns focused roles that may need less general capability than a single agent handling the whole project. The supplied response proposes evaluating model suitability separately for each role while judging the complete workflow: outcome quality, missed obligations, context preparation, coordination, review, rework and human intervention.

A cheaper invocation does not establish a cheaper or more trustworthy outcome if it causes additional review or repair. Conversely, a capable focused role need not use the most expensive model merely because the overall project is difficult. Which configurations achieve the intended tradeoff remains an evaluation question, connected to [per-role configuration](../register.md#enh-004-per-role-agent-configuration-and-model-mixing).

### Experience and institutional memory

The [experience reflection](experience-without-implicit-rules-20261016.md) develops this question into observation, explanation, experience and binding rules, with a small learning loop and an illustrative library case.

The organizational analogy needs an account of accumulated experience. Human workers can remember why a decision was made, recognize recurring failures and develop judgment about their area. An agent invocation must not be assumed to possess the organization's earlier experience merely because it has the same Manager name.

The model, reasoned decisions, incident findings, rejected approaches and unresolved obligations could supply that continuity. A useful retained lesson would identify what happened, why it mattered, the evidence and the circumstances in which it applies. For example, a previous lending defect might show that checking membership only at one entry point left another path open.

An execution transcript preserves events; it does not by itself turn them into usable experience. The open questions are how lessons are selected, supplied to the right roles, revised when circumstances change and retired when obsolete. Earlier agent judgments must remain distinguishable from binding model rules. Learning from a run must not silently change accepted intent. This connects to [institutional memory](../register.md#idea-001-the-model-as-institutional-memory-for-stateless-agents) and [reasoned precedents](../register.md#idea-005-reasoned-user-decisions-as-project-precedents).

### Different perspectives through configurable teams

Configurable teams provide a place to assign distinct responsibilities without introducing a separate Manager hierarchy for every perspective. A team's roles could differ in their questions, relevant context, tools and model profiles. A named role alone does not establish a different perspective; its actual assignment and findings should show the difference.

Illustrative review questions for the library are:

| Perspective | Question |
| --- | --- |
| Domain behavior | Can a suspended member or a member with five loans borrow through any supported entry point? |
| Existing consumers | Does the changed behavior preserve the contracts relied on by other areas? |
| Tests | Would these tests reject a plausible implementation that permits a sixth loan, and where do their expectations come from? |
| Integration | Does a membership-state change affect lending and reminders consistently? |
| Documentation | Do the usage instructions still describe the realized behavior and its limits? |
| Maintainability | Does the change follow the project's agreed patterns without adding unnecessary mechanisms? |

These are possible assignments within existing team and integration responsibilities, not a mandatory agent per perspective. The architect should consider which perspectives a task needs, which can share a role and when specialization earns its coordination cost. Selecting different models is another configurable option to evaluate; it does not establish independent judgment by itself. The [existing team variants](../../design/canonical-model/manager-realization-north-star.md#team-variants) and [configuration proposal](../../design/canonical-model/manager-apply-concept.md#organization-and-configuration) remain the design owners.

### Four eyes and correlated mistakes

The four-eyes principle needs a reviewer who can form a separate judgment from the requirements and actual candidate, with fresh context and the ability to require rework. The implementer's explanation can be evidence to examine, but must not substitute for examining the result. A reviewer also needs access to relevant detail rather than being limited to a reassuring summary.

Fresh context does not remove every shared blind spot. Implementer and reviewer can receive the same incomplete context, adopt the same mistaken reading, or agree on an incorrect test expectation. More reviews can repeat that error. Distinct questions, appropriate source access, concrete counterexamples and independently stated expectations are ways to challenge it. The library's protected expectations are an evaluation instrument, not a second specification every adopter must maintain.

Local review, parent integration review and final verification should retain different purposes: assess the local realization, assess interactions, and establish the required current evidence for the aggregate. Count useful findings, remaining defects and rework effort; a count of reviewers or passing verdicts is insufficient to establish trust.

### Limits of the analogy and future autonomy

The management analogy supplies useful questions about responsibility and escalation. It does not establish that a particular human organizational structure will work equally well for agents. Its benefit needs to survive context preparation, handoffs, repeated reviews and recovery work.

Likewise, incentives need an explicit operational meaning. A justified no-change result, a grounded objection and a timely escalation should be compatible with success. A system that rewards only completion or activity could encourage unsupported closure or unnecessary maintenance. The discussion selects no reward algorithm or assumption that agents have human motivations.

Routine human PR review becoming unnecessary remains the owner's goal. Evidence for that working style must include what the organization misses, the quality of its escalations and the owner's total attention over repeated changes. The question is whether the whole process earns trust while preserving quality, not whether every individual role reports success.

| Topic | Question or implication |
| --- | --- |
| Different perspectives | Assign distinct questions and relevant inputs. Repeating the same broad review many times may repeat the same blind spot. Judge findings and missed obligations rather than review count alone. |
| Continuity | Model, reasoned decisions, current evidence and open obligations can carry institutional memory between agent sessions. Avoid repeatedly rediscovering settled context. |
| Incentives | Examine what the workflow treats as success. A justified no-change result or timely escalation can be valuable; completion pressure should not reward unsupported success claims. No scoring or reward scheme was selected. |
| Autonomous maintenance | Distinguish a useful repair or simplification from unnecessary churn. Continuous activity is not itself evidence of project health. |
| Human attention | Measure the combined burden of modeling, questions, reviews, recovery and supervision over repeated changes. Less code review is useful only if equivalent work does not return elsewhere. |
| Confidence | Show what was examined, the result and unresolved uncertainty. The aspiration that nothing is overlooked still needs evidence about actual coverage and semantic mistakes. |

The new concept supplies a plausible structure for this working style. Its benefit depends on whether that structure finds errors missed by the current workflow while reducing total human supervision. This reflection does not change the current Apply scope, implementation sequence or review authority.

Return to [product reflections](README.md).
