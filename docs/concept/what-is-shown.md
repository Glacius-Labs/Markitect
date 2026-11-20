# What is shown and what is not

The concept rests on a few results of real runs, each run once and on the tool of an earlier concept, so none of them is a rate. Most of what this concept adds is not shown yet; the [roadmap](../roadmap.md#current-steps) builds it and then tries it.

| Shown | What it supports | Where |
|---|---|---|
| A separate reviewer found 17 of 18 planted bugs blind, with no false alarm; on a project five times the size of Makerspace it found the 20 bugs its tests miss in each of five launches; on a small project it found two bugs behind weakened tests | A fresh reviewer, and control samples as a measure of it | [blind test](https://github.com/Glacius-Labs/Markitect/blob/d0c23a31efc928e95f3acc0633deb01a5f887884/docs/validation/k-checker-test-20261103.md), [#1050](https://github.com/Glacius-Labs/Markitect/pull/1050), [#1074](https://github.com/Glacius-Labs/Markitect/pull/1074) |
| Readers of a spec, without the code, found all three contradictions another agent had planted, and four of the nine later spec problems of a full build before it started | Reading the spec as a source of its quality | [#1079](https://github.com/Glacius-Labs/Markitect/pull/1079), [#1083](https://github.com/Glacius-Labs/Markitect/pull/1083) |
| A builder with its reviewer built a spec of 170 items from nothing in six slices, with no hang and nothing repaired by hand | The build loop of builder and reviewer over a long list | [#1083](https://github.com/Glacius-Labs/Markitect/pull/1083) |
| A first spec drawn from a project of 131 files held 113 of 119 statements after three rounds of correction, the rest being the project's own contradictions | The first design of an existing project | [#1040](https://github.com/Glacius-Labs/Markitect/pull/1040), [#1047](https://github.com/Glacius-Labs/Markitect/pull/1047) |
| A pass per perspective, such as security or operation, described little more than a single pass at several times its cost | Perspectives as questions to ask, in catalogs, not as passes | [#1040](https://github.com/Glacius-Labs/Markitect/pull/1040) |
| Types of their own for deployment, operating procedures and roles, tried by hand on one spec, did not make it better; what helped was that a type's sections show what a document leaves unstated | The content model as the sections of a few types | [#1051](https://github.com/Glacius-Labs/Markitect/pull/1051) |

**Not shown:** the language on a real software project; counter-checks, control samples and spot checks; the landing rule and the check of the write table; jobs without a session; reports, triage and what the system learns of the owner's decisions; a series of changes; and the loop over weeks.

## When Markitect counts as proven

First the lab test: two clean passes in a row over the eight change steps of the 5× Makerspace project, every step landing by the landing rule and every hidden case passing. Then the field test: a project that is not the owner's runs four weeks in the loop with real work in CI, and four figures from the tool and Git hold ([DEC-120](../decisions.md#dec-120-how-markitect-is-judged-and-when-it-is-proven)):

| Figure | Holds when |
|---|---|
| **Work** | Actions that change spec and code land every week |
| **Convergence** | Open reports and `status` do not grow over the weeks |
| **Checking** | The hit rate of control samples does not fall, and spot checks do not contradict earlier verdicts more often |
| **No rescue** | No landed action is reverted in Git except through a report |

All four can turn out negative without anyone reading. Being proven brings in what waits for it: a user interface and other clients, another provider, and telemetry for reference values ([roadmap](../roadmap.md#conduits)).
