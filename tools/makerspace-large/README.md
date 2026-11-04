# The large Makerspace

An evaluation project about five times the size of [Makerspace](../makerspace-oracle/README.md): 48 modules, 188 operations, 413 model pages and about 53,000 words. It tests how each approach does when a project strains one session. Public, not counted. The new sealed project is separate and has nothing to do with it.

Old behavior stays: every one of Makerspace's 37 operations answers as it did unless a record of a new module exists, so Makerspace's 54 hidden cases stay as they are and must pass. The [design](DESIGN.md) holds the rules of the growth, the module table and the binding contracts.

| Path | Content |
| --- | --- |
| [DESIGN.md](DESIGN.md) | The design every author worked from |
| [`tools/evaluation-tasks/makerspace-large`](../evaluation-tasks/makerspace-large) | The model and the registry pages: what a variant working from a model receives |
| [spec/requirements-w0.md](spec/requirements-w0.md) | The requirements document, rendered from the model: what every variant receives as `task.md` |
| [pages.py](pages.py) | Parses the pages, checks their form, renders the requirements document and writes the facts |
| [reference/w0](reference/w0) | The reference program, hidden from the variants |
| [oracle/](oracle) | The hidden cases and their kit, hidden from the variants |
| [notes/](notes) | What the authors chose where the design was silent |
| [`tools/evaluation-judges/makerspace-large`](../evaluation-judges/makerspace-large/README.md) | The judge |

The model is the one source. After a change to a page:

```text
python -I -B tools/makerspace-large/pages.py check tools/evaluation-tasks/makerspace-large/w0/model tools/evaluation-tasks/makerspace-large/registry
python -I -B tools/makerspace-large/pages.py render tools/evaluation-tasks/makerspace-large/w0/model tools/makerspace-large/spec/requirements-w0.md
python -I -B tools/makerspace-large/pages.py facts tools/evaluation-tasks/makerspace-large/w0/model tools/makerspace-large/oracle/facts.json
```

## The hidden cases

| Class | Cases | Written |
| --- | --- | --- |
| Makerspace's own | 54 | Unchanged |
| G | 151 | Generated per new operation from the parsed pages: arguments, permissions, referenced records, no audit entry on a refusal |
| B | 90 | By hand per module, from the requirements document alone: effects, refusal orders, boundaries, result shapes |
| X | 32 | By hand across modules: the gates in order, `checkOut` with passes and consumables, the money chain through the journal, dunning through suspension, escalation, the waiting list through discounts |

The reference and the hand-written cases were written independently from the model and the requirements document, and met only when both were done. 316 of 326 passed at once, and the other 10 were one mistake in the cases.

To measure the cases, an author who did not see them seeded 89 plausible bugs in the reference ([oracle/mutants.json](oracle/mutants.json)): boundaries off by one, refusals in the wrong order, a forgotten gate, a missing effect, the wrong rounding. 88 were caught at once and a new case catches the last. A bug fails a median of 2 cases.

```text
python -I -B tools/makerspace-large/oracle/catalog.py cases
python -I -B tools/makerspace-large/oracle/catalog.py run --program PATH --cwd DIR
python -I -B tools/makerspace-large/oracle/calibrate.py tools/makerspace-large/oracle/mutants.json
```

## Limits

- W0 only so far: no change steps, no plants, no fixed start. The reference's registry pages, which a fixed start would need, are not written.
- The reference has only the tests it inherited; its hand-checked operations are not covered by tests of its own, though the hidden cases cover them.
- The hand-written cases were not reviewed one by one. Their strength is measured by the seeded bugs, which are bugs one author chose.
