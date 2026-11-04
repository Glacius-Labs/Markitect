# The first side-by-side page: the large Makerspace's builds, 4 November 2026

Step U3 of the [delivery plan](../implementation-plan.md#concept-30-build): what [concept 3.0 §13](../concepts/v3-concept.md#13-how-we-know-it-works) keeps and compares, on the six builds of the [large Makerspace](../../tools/makerspace-large/README.md) that exist. All six passed all 327 hidden cases, so the hidden cases cannot tell them apart. No result is recorded yet.

## Dispatch

Blind reads of the six kept builds, one fresh session each. The measuring runs no agent.

| Item | Value |
|---|---|
| Source | Branch `dev/u3-large-project`: [`blind_read.py`](../../tools/evaluation/blind_read.py) and [`reviewer.py`](../../tools/evaluation/reviewer.py) at the commit that adds this page |
| Material | The six launches in the [lasting store](../../tools/evaluation/kept.py): K on Sonnet, arm S on Sonnet, K with a Haiku lead and Opus workers, and K on Haiku alone three times. Each as a copy of its program with a random label, without its history and without the files that name how it was made. The code stays as it is, comments included, so a comment or a test that names the model can still show how a build was made, though the reader is not told what that means |
| Readers | One session per build, with the requirements and the same questions. Read, Glob and Grep only. It does not know of the others |
| Provider | `claude-opus-5-5` for all six, its subagents forced onto it. Judging architecture is the one thing a session here is asked for |
| Login | The owner's subscription login, as an access-only copy in the suite's sandbox. Every other `ANTHROPIC_*` and `CLAUDE_*` variable is cleared |
| Image | The suite's sandbox `sha256:67e2cb76d76c27fd20b183e8d209e6407c49f758c35a454168b5b37cc2e24ce8`, which K's launches used |
| Limits | 50 minutes per session, one at a time, and one more attempt for a label whose session fails, so at most 12 sessions. Expected use is 3 to 6 points on the runner's weekly reading; above 8 points the series stops after the current session |
| Abort | A refused login, the engine busy, or a second failed attempt on a label, which the page then shows as no read |
| Lock | `NATIVE-RUN.lock` in the shared notes folder, through the tool's `--lock`. Other Docker work is checked first and stays off |
| Cleanup | The sandbox removes its container and network after each session; afterwards nothing labelled with the evaluation may remain |
| Evidence | Outside the repository, in `%LOCALAPPDATA%/markitect-kept/pages/large-w0/`: the figures, the material, the key and every attempt. The tool's answers and transcripts are scanned for credentials |
