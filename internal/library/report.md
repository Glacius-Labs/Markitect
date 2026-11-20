# Report

A signal that a change is needed: an idea of the owner, a bug report, an incident, a wish, a log, a finding of a job or a problem with the spec itself. It says what is wrong or wanted, with evidence, and proposes no solution. A job writes the first five sections; triage adds the rating, the actions and the closing, and the job that does the action adds the prevention. The first line of Source, Place, Occurred or risk, Violated property, Rating and Closing, and each line of Actions, holds a value of fixed form, which the tool reads.

## Source
Who or what raised it, and the day: `owner`, `intake` or `job JOB ROLE`, then a comma and the day as `YYYY-MM-DD`, such as `job build reviewer, 2026-11-20`. Only a conversation sets the owner.

## Place
The spec document, artifact or folder it concerns: one path below the project folder, or a Markdown link to it relative to this file.

## Finding
What is wrong or wanted, with the evidence.

## Occurred or risk
`occurred`, when it happened as an error, or `risk`, when it may happen.

## Violated property
The property of the spec it violates: `necessary`, `complete`, `consistent`, `unambiguous and verifiable` or `structured`, one of the five properties of a good spec; `correct`, when the spec says the wrong thing; or `none`, when the spec holds and an artifact deviates from it.

## Rating (optional)
Impact and likelihood, set by triage: `impact LEVEL, likelihood LEVEL`, each level `low`, `medium` or `high`.

## Actions (optional)
Set by triage: a line `branch NAME` for each branch that answers it, or a line `escalated` followed by the proposals for the owner.

## Prevention (optional)
The fixed question or counter-check the action adds, so that the same gap shows earlier next time.

## Closing (optional)
How it was closed, set by triage: `YYYY-MM-DD: done`, `YYYY-MM-DD: duplicate of NAME` or `YYYY-MM-DD: nothing to do`, then the reason on the lines below. A report is open while it has no closing.
