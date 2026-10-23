# Trust through organized agent work

Date: 16 October 2026. Owner vision, observations and hypotheses from the outlook discussion. The [owner's statements](../sources/organization-and-trust-20261016.md) and the [assistant response](../sources/organization-and-trust-assistant-20261016.md) keep the original wording. Assistant interpretations are marked; recording them adopts nothing and measures nothing.

## Extracted ideas and their status

| Idea | Origin and status | Detail |
| --- | --- | --- |
| Spend human attention on modeling, direction and important questions | Owner goal | [Working life](#the-working-life-the-owner-wants) |
| Compare with today's story, implementation, AI review and merge workflow | Owner experience | [Current workflow](#the-workflow-being-improved) |
| Distribute responsibility through delegation and escalation | Owner analogy and hypothesis | [Trust mechanism](#organization-as-the-trust-mechanism) |
| Derive perspectives from enduring responsibilities | Assistant interpretation of an owner observation | [Attention](#attention-beyond-the-story-and-diff) |
| Configurable teams for focused work and different perspectives | Owner idea; endorsed as important ([ENH-006](../register.md#enh-006-configurable-teams-for-perspectives-and-four-eyes)) | [Teams](#different-perspectives-through-configurable-teams) |
| An independent reviewer with the actual requirements and result | Owner four-eyes direction | [Four eyes](#four-eyes-and-correlated-mistakes) |
| Capable models per role, judged on the whole process | Owner hypothesis; assistant evaluation proposal | [Cost](#model-capability-and-the-cost-of-the-whole-process) |
| Local decisions with sufficient context and bounded authority | Owner observation and hypothesis | [Local decisions](#local-decisions-and-context), [boundaries](#local-authority-and-escalation-boundaries) |
| Experience across runs | Owner: important, not to be forgotten ([IDEA-001](../register.md#idea-001-the-model-as-institutional-memory-for-stateless-agents)) | [Institutional memory](#experience-and-institutional-memory) |
| Justified no-change, objections and escalation count as success | Owner analogy; assistant interpretation | [Limits](#limits-of-the-analogy-and-future-autonomy) |
| Reports for steering and occasional review of the process | Owner direction; assistant proposal | [Oversight](#leadership-reports-and-process-oversight) |
| Earn enough trust to stop routine human PR review | Owner goal; evidence open | [Limits](#limits-of-the-analogy-and-future-autonomy) |

## The working life the owner wants

The owner wants to spend time modeling, developing ideas and concepts and setting rules. The organization turns that intent into clean, consistent implementation, develops features, finds and fixes bugs and keeps the project tidy. The owner watches ideas become reality, answers important questions and steers without supervising implementation. Avoiding routine human PR review is an explicit goal, conditional on earned trust. This develops [VIS-001](../register.md#vis-001-cockpit-personal-ai-and-autonomous-project-organization) and the [promises](../register.md#promises).

## The workflow being improved

The owner's current workflow: plan a user story; have Claude or Codex implement it and open a PR; have an agent review it; have an agent answer the findings until the PR is ready; a human review is expected but rarely happens; merge. Agents concentrate on the task in front of them, and a large task can leave relevant aspects unnoticed. The baseline is this workflow, as in [DEC-002](../register.md#dec-002-trust-is-measured-against-todays-practice).

## Organization as the trust mechanism

The owner compares a company or government with a software system: areas of responsibility, subproblems, a goal, leadership that delegates, receives reports and handles escalations. The hypothesis: several focused workers, paired with independent reviewers and coordinated by management, are more dependable and efficient than one very capable worker with many responsibilities. Trust comes from the organization and method, not only from its members. The owner's word "dümmere" means relative capability; [DEC-007](../register.md#dec-007-more-economical-models-means-capable-mid-tier-models) keeps the direction of capable mid-tier models. This elaborates [CPT-001](../register.md#cpt-001-federalism) and [ASM-004](../register.md#assumptions-and-hypotheses).

## Local decisions and context

People already ask AI about local technical questions and usually follow its recommendation; the owner sees missing context as the main reason when it is wrong. A Manager with sufficient context and clear authority could therefore decide many such questions itself, and the owner is asked only about material ones.

## Assistant interpretation and open questions

The owner asked to keep these objections, especially experience, different perspectives and the four-eyes principle through configurable teams ([source](../sources/organization-and-trust-20261016.md#retain-objections-and-team-design-questions)).

### Attention beyond the story and diff

An implementation prompt directs attention to the requested feature, and a PR review to the diff; an obligation absent from both gets little attention. Review assignments derived from enduring responsibilities and model obligations supply a different starting question. This organizes attention; it does not guarantee that every needed perspective is known.

### Local authority and escalation boundaries

Delegated judgment needs relevant context, goals and binding constraints, authority within the area and criteria for involving others or escalating. The [mandate section](../../history/apply-design-before-dec-056/manager-apply-configuration.md#the-mandate-section) and the [escalation rules](desired-working-day-20261016.md#decision-authority-and-escalation) express them.

### Leadership reports and process oversight

Upward information should help steering: outcomes, open decisions, relevant failures and reasons to intervene, with details available. Less routine code review can be paired with occasional targeted inspection of whether responsibilities, review questions and escalation boundaries still work. No frequency or audit schedule was chosen.

### Model capability and the cost of the whole process

Focused roles may need less general capability than one agent handling everything. Suitability is evaluated per role but judged on the whole workflow: quality, missed obligations, coordination, review, rework and human intervention. A cheaper call is not a cheaper outcome if it causes more repair ([ENH-004](../register.md#enh-004-per-role-agent-configuration-and-model-mixing)).

### Experience and institutional memory

Human workers remember why decisions were made and recognize recurring failures; an agent invocation does not inherit that merely by having the same Manager name. The model, reasoned decisions, incident findings, rejected approaches and open obligations can carry continuity. How lessons are selected, supplied, revised and retired, without becoming binding rules on their own, is developed in the [experience reflection](experience-without-implicit-rules-20261016.md).

### Different perspectives through configurable teams

Configurable teams are the place for distinct responsibilities without a Manager hierarchy per perspective. Roles can differ in question, context, tools and model profile; a role name alone is not a perspective.

| Perspective | Example question for the library |
| --- | --- |
| Domain behavior | Can a suspended member or a member with five loans borrow through any supported entry point? |
| Existing consumers | Does the changed behavior keep the contracts other areas rely on? |
| Tests | Would these tests reject an implementation that allows a sixth loan, and where do their expectations come from? |
| Integration | Does a membership change affect lending and reminders consistently? |
| Documentation | Do the usage instructions still describe the realized behavior? |
| Maintainability | Does the change follow agreed patterns without unnecessary mechanisms? |

The [north star's team variants](../../design/canonical-model/manager-realization-north-star.md#team-variants) keep the options; the first strategy is Manager plus reviewer.

### Four eyes and correlated mistakes

The reviewer forms a separate judgment from the requirements and the actual candidate, with fresh context and the power to require rework. Fresh context does not remove shared blind spots: both can receive the same incomplete context or agree on a wrong expectation. Distinct questions, source access, concrete counterexamples and independently stated expectations challenge that. Local review and integration review keep different purposes, and the Host's final evidence check makes sure both happened on the current result. Trust is judged by useful findings and remaining defects, not by the number of reviewers.

### Limits of the analogy and future autonomy

The analogy supplies good questions about responsibility and escalation; it does not prove that a human structure works the same way for agents. A justified no-change result, a grounded objection and a timely escalation must count as success; rewarding only completion invites unsupported closure. Stopping routine PR review stays the owner's goal, to be earned with evidence about what the organization misses, the quality of its escalations and the owner's total attention over many changes.

Return to [product reflections](README.md).
