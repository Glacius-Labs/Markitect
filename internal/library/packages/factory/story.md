# Story

A part of a feature with its own target picture, complete in itself once its predecessors are done: afterwards something can be used, and the next story adds to it. A story is what gets carried out, by a Builder with Reviewers, by an Analyst, or by a person. Take it for every piece of work in the plan, also for a bug, a maintenance job or a smell analysis, which are stories with fixed tasks in the Area's feature `housekeeping`. Do not take it for an idea nobody has decided on: that is a [finding](finding.md). Its document lies in `.markitect/plan/`, in its feature's folder; its place there is its identity, it has no number.

## State
On the first line one of `draft`, `ready`, `in progress`, `acceptance`, `done`. Example: `ready`.

## Priority
On the first line one of `critical`, `high`, `normal`, `low`: how urgent it is. The order within the feature stands in the feature. Example: `normal`.

## Depends on
The stories it waits for, as links, one per line, or `none`. Whoever creates the story sets it, always. Example: `[Renew a loan](renew-a-loan.md)`.

## Claimed by
Who works on it now, the person, or the agent and its run, or `none` while nobody does. Claiming the story is writing yourself here on `main` before you start. Example: `build, 2026-11-24 14:32`.

## Origin
Where it came from: a link to the finding or the report in the context, or to its feature for a story cut from one. A smell story lists every finding that references it, one link per line. Example: `[Renew a loan online](feature.md)`.

## Target picture
What should hold when the story is done, like an interface or a responsibility. It fits every level above it: its feature and the spec. Example: `A member renews an open loan once; the due date moves by the lending period.`

## Target changes
Every place in the spec that this story's files realize, each as a link with one sentence on how the story changes it, or `unchanged` when it changes nothing there. When no document of the spec covers its files, something is missing in the spec, and the story adds it: that is a change. `none` only for a story that changes no file, such as a smell analysis. A story cut in a feature's planning holds the places as far as they are known; its own planning completes them. The changes lie on the story's branch and land with its realization. Example: `[Lending period](../../../../docs/spec/product/lending/period.md): a renewal extends it once.`

## Plan
How it will be realized, coarse, in concepts and building blocks: no file names, no code; `not planned yet` while the story waits in `draft` for its planning. Example: `The loan gains a renewal; the member's page offers it.`

## Tasks
The tasks as check boxes, in order, ticked when done; `not planned yet` while the story waits in `draft` for its planning. A story with fixed tasks has exactly those. A story with something to accept ends with the task for accepting it. Example: `- [ ] Polish, verify interface and use, accept`.

## Result (optional)
What changed, written by the Builder, or, after an abort, a link to the report in the context. Nothing else: the Reviewer's `confirm` is the tool's and stands in the state. Example: `Loans can be renewed once; the member's page offers it.`
