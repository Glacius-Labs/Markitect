# Overview

Type: [Dashboard](../../../packages/factory/dashboard.md)

## For
everyone: where do we stand, who does what?

## Sections
1. Critical: everything critical that is not done ([critical until done](../rules/briefing/critical-until-done.md)), one line each: the findings with `critical` in `.markitect/findings/pending/` and `ready/`, and the stories of priority `critical` that are not `done`, from the plan. The Assistant says it always, first.
2. People: for each [person](../people/README.md), their role, their own features, their stories `in progress` with who works on them, Builder or Analyst, and in `acceptance`; from the plan and the people. The Assistant says only what touches the person.
3. Areas: one line per Area: its features, its stories by state as bars, the size of its housekeeping; from the plan. The Assistant says nothing of it.
4. Landed: what landed on `main` since the last briefing; from the plan. The Assistant says it as a number.
5. Briefing: the open findings as a number per importance, leading to the [Briefing](briefing.md); from `.markitect/findings/pending/`. The Assistant says it as a number.
