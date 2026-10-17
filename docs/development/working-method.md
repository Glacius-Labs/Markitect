# Learning through complete workflows

The owner adopted this working method on 17 October 2026 after the project-history discussion ([DEC-038](../concepts/register.md#dec-038-learn-through-complete-workflows), [original instructions](../concepts/sources/project-history-working-method-20261017.md)). Its binding intent lives in the canonical [repository Rule](../../.markitect/model/repository/learning-driven-workflow.yaml); this guide makes that Rule usable in day-to-day design and delivery. It governs how we develop Markitect, not the behavior required of every adopting project.

## Start from the workflow

Read the [delivery plan](../implementation-plan.md) for the current target, scope and dependencies. Keep one complete user workflow as the next reference point. For the first reliable Apply, that is the existing two-Apply library demonstration. The later working-day scenarios keep their own sequence and scope.

A useful change advances that workflow or resolves a concrete failure, missing accepted obligation or necessary blocked step. Implementation choices within the accepted design can proceed normally. A genuinely new decision follows the existing owner process. A confirmed defect can require a design correction; stability is an opportunity to learn from use, not a requirement to preserve a known wrong design.

## Explain a substantial design change

Use the existing owning design or decision document. Add a short account answering:

1. What new observation have we made? Name the relevant example, source or failure.
2. Which prior assumption does it challenge?
3. What changes in the design as a result, and what stays applicable?
4. Which observable result would demonstrate improvement?

Keep observed facts, owner experience, interpretation and proposal distinct. Reconstructing a similar design from the original problem supports conceptual stability. It is not a fully independent test when the same person brings the same experiences and preferences, and it does not establish successful use. Fixing a recurring failure in a complete workflow supplies a different kind of evidence.

## Investigate uncertainty before building on it

Identify the technical boundary on which the next design depends most. Use the smallest useful real probe before dependent contracts and implementation harden. It might examine a provider handshake, isolation boundary or interruption behavior. State what the result establishes and what remains unknown. Preserve unsuccessful attempts.

This is a planning practice, not new authority to run providers, studies, releases or deployments. Use existing finite grants and explicit scope. No new general test matrix, framework or mandatory prototype is required for an already established boundary.

## Review the need for mechanisms

Alongside correctness and consistency, independent design review asks:

- Which observed failure or accepted obligation does this mechanism address?
- Could a simpler approach serve the same purpose?
- What does the addition cost in coordination, knowledge maintenance and future verification?

Keep the existing quality and safety requirements. Simplification must preserve their intent; a second reviewer label or a larger collection of green checks does not itself establish a better outcome. Use the existing relevant review rather than creating a separate review institution for every change.

## Keep the owner oriented

At a completed design decision, a changed workflow boundary or a delivery milestone, the responsible chat gives a short account of:

- what the owner can now do, with the evidence scope;
- what Markitect takes over;
- where the owner still has to intervene or what blocks progress;
- which next observation could change our assessment.

The delivering chat explains its result; the coordinating chat connects cross-package milestones to the overall workflow. Use existing owning documents and chat/PR evidence. Do not add another dashboard, recurring report automation or approval round. A package count alone does not describe a usable workflow.

## Keep one knowledge owner

[Documentation maintenance](documentation.md) owns placement and historical/current boundaries. Record intent at its canonical owner, preserve owner decisions with their source, and link from guides and maps. Keep dated assessments as assessments. An accepted working-method change does not turn an earlier assistant judgment into a measured result.

The [parallel-work guide](parallel-work.md) explains bounded execution. The [delivery plan](../implementation-plan.md#integration) owns exact-head review, CI and direct delivery merges; this method does not replace those rules or require unrelated repeated checks.
