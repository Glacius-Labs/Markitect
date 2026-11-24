# Finding

One thing that deserves attention, taken from a report: an idea, a bug, an oddity. It says what was noticed and where it comes from, never how to solve it. Related statements become one finding. What the model already says, a commitment, a decision on a finding that exists, and praise are no findings ([what is no finding](not-a-finding.md)). The Triager writes it; the Project manager carries it into the plan or the model and closes it. A finding stays, the rejected ones too.

## State
Type: `pending`, `ready`, `closed`

On the first line one of `pending`, `ready`, `closed`: `pending` while it needs a human, `ready` once it is released for work, `closed` when it is done, with its outcome. Every finding ends `closed`. Example: `pending`.

## Kind
Type: `Requirement`, `Idea`, `Question`, `Bug`, `Risk`, `Smell`, `Planning`, `Process`, `Unclassified`

On the first line one of the nine [types of findings](finding-types.md): `Requirement`, `Idea`, `Question`, `Bug`, `Risk`, `Smell`, `Planning`, `Process`, or `Unclassified` when none fits. Example: `Requirement`.

## Importance
Type: `critical`, `high`, `normal`, `low`

On the first line one of `critical`, `high`, `normal`, `low`. It decides when the finding is shown ([when shown](../briefing/when-shown.md)). Example: `high`.

## Module
The module of the model it roughly concerns, as a link to its README, or what it touches as far as it is known. Example: `[Lending](../../lending/README.md)`.

## What
What was noticed, or what the idea is, without a solution. Example: `Members cannot see when a loan is due without asking at the desk.`

## Source
Type: [Report](../observing/report.md)

The [Report](../observing/report.md) and the statement it comes from, as a link, and who said it when. Example: `[dropped material, 23 Nov](../reports/2026-11-23-0600.md), statement 4`.

## Context
What was being worked on, and who said it. For a finding whose answer follows from what is decided, also the place in the model it follows from, as a link, and the answer in one sentence. For a finding that reverses another, a link to that finding. For a finding that was decided where it arose, also the decision in the words it was said in, and who decided. Example: `Said while polishing the member's page.`

## Why this importance
The reason for the importance, in a sentence. Example: `Renewals are half of all desk visits.`

## Likelihood and impact (optional)
For a Bug and a Risk: `likelihood LEVEL, impact LEVEL`, each `low`, `medium` or `high`. Example: `likelihood high, impact medium`.

## Outcome (optional)
What became of it, written when it is carried out. The first line says how it ended: `decided` when people decided on it, `without a decision` when it went on from `ready` without one, `noise`, or `duplicate`. The lines below say what became of it: in the plan, with a link to the story or feature; added to the model, with the place and the sentence that was added; rejected, with the reason; a duplicate of, with a link; attached to, with a link to the claimed feature, while the finding stays `ready` until the Architect, planning in that feature, takes it up into a story or sets it aside, adds that below, and closes it. Example: `without a decision`, then `In the plan: [Renew a loan](../plan/renew-a-loan.md)`.
