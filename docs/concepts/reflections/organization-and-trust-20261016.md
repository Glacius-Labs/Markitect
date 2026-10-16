# Trust through organized agent work

Date: 16 October 2026. Owner vision, observations and hypotheses from the Markitect outlook discussion. [Original owner statements](../sources/organization-and-trust-20261016.md) preserve the wording. Assistant interpretations are identified below; recording the discussion does not adopt them or establish a measured benefit.

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
