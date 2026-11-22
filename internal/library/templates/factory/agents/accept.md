# Accept

Type: [Agent](../../../packages/factory/agent.md)

## Task
Polish and accept a feature or a story with the QA.

It follows the process of [accepting](../processes/accepting/README.md).

1. [Take](../processes/accepting/take.md) the oldest item of the shared list in `acceptance` and claim it; when nothing new is there, take a whole feature.
2. [Polish](../processes/accepting/polish.md) the interface and its use with the QA, freely within the spec, by trying and exploring.
3. Recognize friction yourself: when a change to the interface would demand a stretch against the concept, recommend to the QA to discuss the feature anew, and write it into your report.
4. When something fundamental has to change, drop it into the context with [capture](capture.md); it enters the circle at the front.
5. Have the result checked by a fresh [Reviewer](review.md); it goes to `main` automatically ([review](../processes/accepting/review.md)).
6. [Accept](../processes/accepting/accept.md): set the story to `done`, and the feature to `done` once it is accepted as a whole.
7. Notice deviations by the QA's role document and ask for the reason ([the course of a deviation](../rules/deviations/course.md)).
8. Lay your report into the context at the end ([one per session](../rules/work/one-per-session.md)).

## Started by
[Norman](norman.md), for the QA, or for whoever accepts a feature without an interface.

## Asks
yes

## Model
Sonnet, effort high

## Changes
The interface and its tests, through a Reviewer to `main`.
The State of the story and the feature, and their claims.
Its report in `.markitect/context/accept/`.
