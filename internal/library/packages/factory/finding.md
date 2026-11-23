# Finding

A single finding from a report: something that deserves attention, an idea, a bug, an oddity. It says what was noticed and where it comes from, never how to solve it. The Triager writes it; the Recorder carries it into the plan or closes it, and the Architect closes one that is attached to a claimed feature. Take it for each topic that triage takes out of reports; related statements become one finding. Do not take it for what the spec already says, for a commitment, which is a story, for a decision already taken, which the Recorder carries out, or for praise, which stays in the context. Its document lies in `.markitect/findings/`, and its folder is its state: `pending` needs a human, `ready` is released for work, `closed` is done, with its outcome. Moving the file changes the state, so a finding has no State section.

## Type
On the first line one of `Requirement`, `Idea`, `Question`, which point at the spec; `Bug`, `Risk`, `Smell`, which point at the product; `Planning`, which points at the plan; `Factory`, which points at the way of working; or `Unclassified` when none fits. Example: `Requirement`.

## Importance
On the first line one of `critical`, `high`, `normal`, `low`. It decides when the finding is shown. Example: `high`.

## Area
The Area of the plan it roughly concerns, as a link, or what it touches as far as it is known. Example: `[Lending](../../../docs/spec/factory/areas/lending.md)`.

## What
What was noticed, or what the idea is, without a solution. Example: `Members cannot see when a loan is due without asking at the desk.`

## Source
The report and the statement it comes from, as a link, and who said it when. Example: `[dropped-material, 23 Nov](../../context/observe-dropped-material/2026-11-23-0600.md), statement 4`.

## Context
What was being worked on, and who said it. For a finding whose answer follows from what is decided, also the place in the spec it follows from, as a link. Example: `Said while polishing the member's page.`

## Why this importance
The reason for the importance, in a sentence. Example: `Renewals are half of all desk visits.`

## Likelihood and impact (optional)
For a Bug and a Risk: `likelihood LEVEL, impact LEVEL`, each `low`, `medium` or `high`. Example: `likelihood high, impact medium`.

## Outcome (optional)
What became of it, written when it is carried out: by the Recorder when it moves the finding to `closed` or attaches it to a claimed feature, by the Triager for noise and duplicates. The first line says how it ended: `decided` when people decided on it, `without a decision` when it went on from `ready` without one, `noise`, or `duplicate`. The lines below say what became of it: in the plan, with a link to the story or feature; added to the spec, with the place; rejected, with the reason; a duplicate of, with a link; attached to, with a link to the claimed feature, while the finding stays in `ready` until the Architect, planning in that feature with its owner, takes it up into a story or sets it aside, adds that below, and moves it to `closed`. Example: `without a decision`, then `In the plan: [Renew a loan](../../plan/lending/renew-online/renew-a-loan.md)`.
