# Accept

Type: [Agent](../../../packages/factory/agent.md)

## Task
Polish and accept a feature or a story with the QA.

It follows the process of [accepting](../processes/accepting/README.md), step by step.

1. [Take](../processes/accepting/take.md) an item and claim it.
2. [Polish](../processes/accepting/polish.md) with the QA, as that step says, friction and fundamental changes included.
3. Have the result [reviewed](../processes/accepting/review.md) by a fresh [Reviewer](review.md).
4. [Accept](../processes/accepting/accept.md): set the states the step names and release the claim.
5. Notice deviations by the QA's role document and ask for the reason ([the course of a deviation](../rules/deviations/course.md)).
6. Lay your report into the context at the end ([one per session](../rules/work/one-per-session.md)), with any friction you recognized.

## Started by
[Norman](norman.md), for the QA, or for whoever accepts a feature without an interface.

## Asks
yes

## Model
Sonnet, effort high

## Changes
The interface and its tests, through a Reviewer to `main`.
The State of the story and the feature, and the story's Claimed by.
Its report in `.markitect/context/accept/`.
