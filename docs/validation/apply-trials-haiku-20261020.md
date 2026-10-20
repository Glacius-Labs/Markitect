# Apply trial series with Haiku — 20 October 2026

**Scope:** the baseline for comparing later Apply versions on the [Apply trial cases](../../experiments/apply-trials/README.md) (#523). It holds 64 real runs on the Apply of Main `a0928a1b` (DEC-056 to DEC-063), before DEC-064 replaced its strategy, defect path and start check. The runs covered 16 case ids (13 base cases and the variants S-QR, S-QV and S-XG) and 22 generated fleet cases. Every Execute and Verify turn used `claude-haiku-5-5` at high effort with the owner request `Apply the committed model.` The record does not judge that Apply further. The first DEC-064 runs on the same cases are in [Apply Pass process trials](apply-pass-process-20261020.md). No owner acceptance is recorded.

## Core numbers

"Met" is the expected outcome of that time. For a build, repair or no-change case, it is Verified with publication `applied` or `no_change`. For a defect or gap case, it is a `model_defect` stop with nothing published (DEC-063). DEC-064 now also accepts a question to the user for a gap, and refuses D-M's registration fault before any work; that fault is now case D-R. Times are wall-clock seconds of the whole Apply, median and range.

| Case | Runs | Met | Faults repaired | Rounds | Time s |
| --- | --- | --- | --- | --- | --- |
| S-D, S-A, D-M (D-M with 6 faults) | 6 | 6 | 24/24 | 1 (S-A once 2) | 328 (265–421) |
| Fleet -M*k*: 1 to 8 faults, 3 to 10 Managers | 9 | 9 | 29/29 | 1 | 432 (318–791) |
| Model-only: S-G, D-G, fleet -G (2 to 10 Managers) | 8 | 8 | – | 1 to 2 | 576 (408–929) |
| S-C model change on the S-G result | 2 | 2 | – | 1 | 422 (400–445) |
| S-O valid overspecification | 2 | 2 | – | 1 | 443 (412–475) |
| S-X, S-XG contradiction | 5, 2 | 4, 2 | – | stop | 142 (64–312) |
| S-QV gap without a natural default | 3 | 3 | – | stop | 61 (59–64) |
| S-Q, S-QR gap with a natural default | 5, 2 | 0, 0 | – | 1 to 2 | 434 (264–500) |
| S-F, D-F follow-up on a Verified result | 2, 2 | 1, 1 | – | 0 to 1 | 337 (170–481) |
| S-N, D-N hand-written conforming stand | 3, 4 | 1, 0 | – | 1 to 2 | 435 (296–924) |
| Fleet -N, -TN generated conforming stand, thin / thorough tests | 6, 3 | 0, 1 | – | 1 | 365 (315–674) |

## What the later comparison should look at

- **Repair:** 53 injected faults in 15 runs were all repaired by the owner of their area. Of 43 faults where a regression test could be checked, 42 got one. In both D-M runs, the rule rebuilt outside its owner was replaced by a call to the owner's code (checked in the diffs).
- **Gaps:**
  - A gap with a natural default (what `issue` does beyond the stock) was decided as a refusal in 7 of 7 runs. A keyword search over all their findings shows that no reviewer named it.
  - In S-QR-1 the existing refusal was kept, and the run published `no_change`.
  - A value the model refers to but never gives (S-QV) was reported every time.
- **Contradictions:** in the one miss (S-X-2), both reviewers described the contradiction but filed it as non-blocking, and the stand was published.
- **Follow-up:**
  - 2 of 4 runs stayed unchanged.
  - S-F-1 stopped, because the 0.2.0 model promises compatibility with a 0.1.0 realization that existed only in Git history.
  - D-F-2 changed 3 files.
- **Conforming stands:**
  - Managers added tests in almost every run whose tests did not cover every modeled boundary. The thin fleet stands got 26 to 134 inserted lines; the generated thorough ones got 0 to 19 (`git diff --shortstat` against the start).
  - Code changed in 5 of the 13 S-N, D-N and thin fleet runs.
  - In D-N, one round had two blocking findings that asked for a command-line test per command.
- **Behavior under the final checks:**
  - All stockroom and fleet results built from the model pass.
  - Both depot model-only results reject names that start with a hyphen, and one crashes on an amount with 5000 leading zeros.
  - Runs that started from a reference kept the edge defects that reference had at the time, because their Managers did not find them. Every reference before `ec4f5ece` rejected hyphen names; references before `1d73c38a` also crashed on very long numbers.
  - DN-3 was the only run that repaired the hyphen defect. Verify did find the other real reference edges: unusable paths, deep nesting and long numbers.

## Setup and method

- **Product:** Main `a0928a1b7a95`, runtime images from `experiments/recursive-apply-demo/ra_build.sh` plus the Python example image, Claude Code 2.1.301. Up to 3 runs in parallel on WSL Ubuntu (16 cores, 30 GB). S-QV and DN-3/DN-4 shared the machine with another session.
- **Trial setup per run:** each run froze the setup it started with.
  - TR-SN-1 used the setup on Main.
  - The first repeats of the base cases used `bad0be23`; the second repeats and the thin fleet runs used `1d73c38a`.
  - The thorough fleet runs used `a65b9cb5`; the escalation repeats used `36735b16`.
  - DN-3/DN-4 used `8652adb6`, which added coverage; S-QV used `87a84cf3`.
  - All 64 runs were then re-assessed with the checks of `ec4f5ece`.
- **Changes to the trial setup only:**
  - Hardened references in `bad0be23` and `1d73c38a`, and the hyphen-name fix in `ec4f5ece`.
  - A fleet cache-key race and the writing of reports with undecodable bytes.
  - Two checks stricter than the model were relaxed or removed: undecodable name bytes, and ASCII-only standard output.
  - The Apply under test was the same in every run.
- **Assessment:** black-box checks inside the Python image after each run; the evaluator read the findings, defect reports and diffs. Run evidence stays under WSL `~/mk-tr`, outside the repository.
- **Limits:** one model family, 1 to 5 runs per case, generated fleet projects that measure organization size rather than domain difficulty. Rates are indications.

## Evidence kinds, kept apart

- **Source checks:** `selfcheck.py` and `markitect check` for the delivering pull request.
- **Real runs:** this record.
- **Main integration:** through the delivering pull request.
- **Owner acceptance:** none recorded.
