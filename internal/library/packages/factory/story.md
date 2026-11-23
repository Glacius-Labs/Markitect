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
Where it came from: a link to the finding or the report in the context, or to its feature for a story cut from one. Example: `[Renew a loan online](feature.md)`.

## Target picture
What should hold when the story is done, like an interface or a responsibility. It fits every level above it: its feature, what the spec says for its Area, and the spec as a whole. Example: `A member renews an open loan once; the due date moves by the lending period.`

## Target changes
The places in the spec that this story changes, each as a link with one sentence on how, or `none` when it changes no standing spec and its own target picture is the place. These changes lie on the story's branch and land with its realization. Example: `[Lending period](../../../../docs/spec/product/lending/period.md): a renewal extends it once.`

## Plan
How it will be realized, coarse, in concepts and modules: no file names, no code. Example: `The loan gains a renewal; the member's page offers it.`

## Tasks
The tasks as check boxes, in order. A story with fixed tasks has exactly those. A story with something to accept ends with the task for accepting it. Example: `- [ ] QA: polish, verify interface and use, accept`.

## Result (optional)
What changed, written by the Builder, or, after an abort, a link to the report in the context. Below it the Reviewer's `confirm`: for each place in the spec that the story realizes, the files and the test that proves it. Example: `confirm: lending/period.md: src/loans/renew.ts, tests/loans/renew.test.ts`.
