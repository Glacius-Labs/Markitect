# Extract round 2: stability on two projects — 20 October 2026

**Scope:** the second improvement round after the [old/new comparison](library-extract-comparison-20261019.md). The owner asked to improve Extract until it is stable and usable. This record covers every change of the round, the defect it fixed, and the final confirmation on build `143327ec`, on the library and on a second project, Readinglog. These are real runs in WSL with Sonnet `claude-sonnet-5-5`, high effort, for every Execute and Verify turn. Source checks, Main integration and owner acceptance are reported separately at the end. Hosted CI was not run.

## Result

The final build completed every run (9 of 9). The assessor found no statement in them that contradicts the reference; two Readinglog models have small imprecisions against the code (see below). Every owner question was the necessary one. On the second project, Python with no documentation, the model has the same shape as the human one.

| | Old (`f15ea723`) | Round 1 (`cc89d18b`) | Final (`143327ec`) |
| --- | --- | --- | --- |
| Library runs completed | 8 of 8 | 6 of 7 | 7 of 7 |
| Library score, of 92 | 66–72, mean 70 | 77–81, mean 79 | 73–80, mean 77 |
| Statements contradicting the reference | 0 | 0 | 0 |
| Necessary owner question asked (seeded and conflict runs) | 3 of 5 | 4 of 4 | 4 of 4 |
| Unnecessary owner questions | 1 | 2 | 0 |
| Over-specifications per model, mean | 42 | 22 | 29 |
| Rebuild from the extracted model passes the oracle | 3 of 3 | 2 of 2 | 3 of 3 |
| Readinglog score, of 32 | — | — | 29 and 29 |

- **Project obligations:**
  - "Internals are free" (P14) is now stated, at least partly, in every final model. In round 1 it scored 0 in 5 of 6 models.
  - P09 (code, tests and documentation agree with the model) scored 0 in no final model, against 3 of 6 in round 1.
  - P07 (what the tests must cover) regressed. It scored 0 in 6 of 7 final models, against none in round 1.
- **Score:** the mean fell from 79 to 77 mainly through P07 and through order bindings that the library reference leaves free (I09, R20, R21).
- **Over-specification** rose again from round 1's 22. Part of the rise is intended: the order in which data was added is now binding, because callers rely on it, and the library reference leaves it free. The largest remaining category is internal wiring that mirrors the Go packages (X2, 9–14 per model).
- **Owner questions:** the necessary question was always asked. RXZ-2 also sent the owner a note about a defect the documents settle; the assessor did not count it as a question.
- **Readable account:** every run ends with a short summary on standard error. It lists the Packages, questions and answers, deviations per Definition, proposed checks, the Verify result, the publication and the adoption step.

## Changes of the round

Each change fixed a defect observed in a run. Defects that a text change alone did not remove were fixed by changing structure.

| Commit | Observed defect | Change |
| --- | --- | --- |
| `0c959bdd` | Unnecessary questions about points the code settles (RXB-6, RXF-2, RXF-3); "internals are free" never fully stated | Ask only about conflicting sources; Verify blocks a point left open that the artifacts settle; the project Package states the choices it leaves free |
| `4e0580e2` | The owner received only JSON | A readable account on standard error; the JSON on standard output is unchanged |
| `7d93b546` | Both completed round-2 plain runs declared the "source layout" free (the third, RXG-7, ended Incomplete) and so freed the documented entry point (RXG-1, RXG-4) | Only internal code structure and design are free; documented paths, commands and entry points stay binding |
| `82d37dd5` | Three runs ended Incomplete in a registration loop (RXF-7, RXG-6, RXG-7). A dedicated project Package had no artifacts, Root registered the README and go.mod, and the Package's reviewer kept asking for registrations it could not make. The text fix of #514 did not remove the loop. | The Package that owns the project also owns the files directly in the artifact root, through a directory exception for that root; deeper Package directories keep their owners |
| `5cc66a78` | On Readinglog the models declared list order and the stored JSON format free, and proposed Packages that own no files (RXR-1, RXR-2) | Observable behavior callers rely on stays binding unless the artifacts show it is open; every Package owns artifacts or has children |
| `143327ec` | `5cc66a78` over-bound incidental observable details: refusal precedence, distinguishable wording, identifier formats (RXH-1 to RXH-3, 37–38 over-specifications) | The binding part is the contract callers rely on, including the order in which data was added; message wording and detail, generated identifier formats, which failure is reported when several apply and field order are free unless the artifacts fix them |

The harness got its own fix, `38d7e146`: the rebuild now finds the build file's registration in every shard.

## Second project: Readinglog

Readinglog is the supplied Python 3 add/list application of `experiments/test-projects/readinglog`. It has one 62-line file and one test, no README, and its code sits directly in `realization/`. It was scored against the human 00-baseline model with a [fixed rubric](../../experiments/recursive-apply-demo/extract-checklist-readinglog.md) that marks what the two files cannot show. Agents and the proposed check use the Python example runtime image.

| Build | Run | Score | Wrong | Packages |
| --- | --- | --- | --- | --- |
| `82d37dd5` | RXR-1 | 23 | 3: list order and storage format declared free | 3, two without files |
| `82d37dd5` | RXR-2 | 27 | 1: storage format declared free | 3, two without files |
| `143327ec` | RXZR-1 | 29 | 0 | 1, owning the whole artifact root |
| `143327ec` | RXZR-2 | 29 | 0 | 1, owning the whole artifact root |

- **Final runs:**
  - Each finished in about 3½ minutes, with the first round green.
  - Each proposed `python -B -m unittest discover -s tests` with the Python profile the Host offered.
  - No owner question was asked.
- **Open imprecisions:** two remain. RXZR-1 says stored IDs are compared after trimming, which the code does only for new input; its reviewer raised this as an advisory finding. RXZR-2 says stored text has no surrounding whitespace.

## Remaining weaknesses

- **Internal wiring stays binding.** Every model still binds some internal wiring as intent: catalog availability operations, snapshots, the views between areas. Reviewers do not block it reliably. Some models also bind a concurrency claim.
- **Test coverage obligations:** "What the tests must cover" is missing in most final library models.
- **Duplicated use cases:** the library wire layer often restates every domain use case.
- **Policy values** live once, at their owner, but as prose rather than a structured policy Kind.
- **Rounds:** the final model phases took 1 round in 5 runs, 2 rounds in 3 runs and 3 rounds in 1 run (RXZ-7). Every structure phase took 1 round. The round budget of 4 was enough.

## Method

The bench, the scenarios, the scripted owner and the library rubric are those of the [comparison](library-extract-comparison-20261019.md#method). Each model was scored once by an Opus assessor that took no part in the runs.

| Series | Build | Runs |
| --- | --- | --- |
| RXG | `4e0580e2` | 7 library runs |
| RXR | `82d37dd5` | 2 Readinglog runs |
| RXH | `5cc66a78` | 4 library runs, 3 of them assessed |
| RXZ | `143327ec` (final) | 7 library runs |
| RXZR | `143327ec` (final) | 2 Readinglog runs |

The rebuild of RXH-1 failed in the harness and was rerun after the harness fix; it passed the oracle 12/12. The samples are small and no significance is claimed.

## Separate judgments

- **Source checks:** the PR of this record passed build, vet, the tests of `internal/apply`, `internal/adapters` and `internal/run`, `markitect check` and the architecture check (see the PR).
- **Main integration:** by that PR.
- **Owner acceptance:** pending. The binding criterion of [DEC-061](../concepts/register.md#dec-061-what-an-extracted-model-contains) is refined by this round; the register records the refinement as a clarification for the owner to confirm.
