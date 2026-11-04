# The large Makerspace's change steps, 4 November 2026

The second half of step U3 of the [delivery plan](../implementation-plan.md#concept-30-build): [eight change steps](../../tools/makerspace-large/CHANGES.md) on the [large Makerspace](../../tools/makerspace-large/README.md), with hidden cases, for [test 2](../concepts/v3-concept.md#13-how-we-know-it-works). No variant has run on them yet.

## How they were made

An author who knew only the requirements document proposed the steps, to a mix set beforehand: two small changes, three that cross areas, a new area, a removal and a renaming. They were taken as proposed. Each step's request lists what it asks for, and a separate list says what it reaches, including what the request leaves unstated.

Then three authors per step, all fresh sessions on Sonnet: one wrote the model from the request and the list, and then one wrote the reference from the model while another wrote the hidden cases from the model and the list. The reference's author saw neither the list nor the cases, and the cases' author did not see the reference until they met.

## Where the reference and the cases met

| Step | What it is | Cases | Failures at the first meeting | New or changed cases | Of those, failing on the program before |
|---|---|---|---|---|---|
| C1 | Bookings in 15-minute steps | 336 | 0 | 12 | 11 |
| C2 | A waiver signature lasts a year | 347 | 0 | 11 | 11 |
| C3 | A fee for no-shows | 365 | 0 | 35 | 35 |
| C4 | Pausing a membership | 408 | 1 | 43 | 43 |
| C5 | A tool library | 444 | 2 | 36 | 36 |
| C6 | Renewing a tool loan | 460 | 0 | 45 | 45 |
| C7 | Prepaid minutes become packs | 470 | 0 | 43 | 40 |
| C8 | Shifts and supervision are removed | 472 | 0 | 23 | 15 |

All three failures were mistakes in a case's own setup or helper, and the reference was right each time. A case that also passes on the program before holds something that must stay: in C1 the off-grid slots, in C7 what must not be renamed, and in C8 the 8 cases that only had a shift taken out of their setup.

The model authors checked their pages with the project's own page tool, which does not look for cycles between Packages. Compiling every step with `markitect check` afterwards found two, from C4 and from C5: a page linked back against the one-way dependencies. Both links became plain words, which changed no operation. [`compile_steps.py`](../../tools/makerspace-large/compile_steps.py) now checks every step.

So a careful author working from the whole model missed nothing here, even where the request does not say it. That says these changes can be done right from a complete model. It says nothing yet about how a strong agent does from the repository alone, which is test 2.

## What this does not show

- **The model was written with the answers in hand.** Each step's model author had the list of what the step reaches. A person keeping a spec has only the request and their own reading, and the list exists because such a reading can miss things. Test 2 asks whether the default misses less than one strong agent; if the default gets these models, it gets part of the answer. This is for the owner to decide before test 2.
- **How strong the cases are.** W0's cases were measured with 89 seeded bugs. These were measured only by failing on the program before the change, which shows that a case can tell the change, not that it would catch every wrong version of it.
- **The authors met only once per step.** Nobody reviewed the cases one by one.

## Dispatch: the judge's self-test

The series judge on each step's reference and on its seeded mutants, in its container. It starts no agent and uses no provider.

| Item | Value |
|---|---|
| Source | Branch `dev/u3-change-steps` at the commit that adds this dispatch, exported with `judges.py export` |
| Image | The judge's `golang@sha256:69a7b9788769bec032d238959b61854e9ae87f57be9029ec04e9885fabf99195`, without network |
| Limits | The judge's own: 3,600 seconds per run, with the limits in `judge.json`. One self-test, and a second only after a fix of a defect the first shows |
| Abort | The engine busy, or a run past its limit |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, through the tool's `--lock`. Other Docker work is checked first and stays off |
| Cleanup | The judge removes its containers; afterwards nothing labelled with the evaluation may remain |
| Evidence | Outside the repository, in `%LOCALAPPDATA%/markitect-large-steps/selftest/`, and the result here |

**Result.** Two runs, about three and a half minutes each, in the judge's Linux container. The first passed 11 of its 13 cases: the judge flagged retired terms at C7 and C8 because the self-test's own tool kept a folder that a step had deleted. With [that fixed](../../tools/evaluation/judges.py), the second passed all 13: every step's reference, from 327 cases at W0 to 472 at C8, and the four seeded mutants. Nothing labelled with the evaluation remained.
