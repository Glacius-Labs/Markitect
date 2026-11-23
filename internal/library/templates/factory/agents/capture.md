# Capture

Type: [Agent](../../../packages/factory/agent.md)

## Task
Drop material unchanged into the context.

Capture is the skill every agent has. It runs inside the calling session.

1. Take what the person says or hands over, such as a transcript, or what the calling agent noticed outside its task ([drop what you notice](../rules/work/drop-what-you-notice.md)). One drop per observation.
2. Keep raw material that lies nowhere else, transcripts, conversations, messages, unchanged and hidden in `.markitect/context/capture/.raw/`. Do not copy what lies elsewhere, such as logs or monitoring; point to it.
3. Write the drop into `.markitect/context/capture/`: the wording unchanged, with what only the calling session knows right now: who, when, and what was being worked on.
4. For a [deviation](../rules/deviations/course.md), write who, what, which role would be usual, the level, and the reason in the person's words.
5. For a [merge conflict](../rules/building/merge-conflicts.md), write which files and which two stories.
6. Never take customer data or secrets into the repository ([what is critical](../rules/deviations/critical.md)); where something cannot be shown without them, keep only the pointer.
7. Write straight to `main` ([straight to main](../rules/work/straight-to-main.md)), confirm in one sentence, and carry on. Do not investigate, fix or classify what you dropped.

## Started by
Every agent, at any time.

## Asks
no

## Model
as the session

## Changes
Drops in `.markitect/context/capture/`.
Raw material in `.markitect/context/capture/.raw/`.
