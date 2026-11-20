# Settings

The settings of a project, which the runners and the tool read. Every section is optional; one left out leaves the runner's default. Each line of a section holds a value of fixed form, which the tool reads.

## Models (optional)
The model per role: a line `ROLE: MODEL` per role, the role one of `architect`, `builder`, `reviewer`, `counter-checker`, `sampler`, `reader`, `triage` and `intake`.

## Counter-checks (optional)
The folder of the counter-checks: one path below the project folder, outside the spec and `.markitect/`. `mark` takes a counter-check only from this folder.

## Cadence (optional)
How often each job runs: a line `JOB: TEXT` per job, the text read by the runners.

## Write table (optional)
What each role may write, narrower than the table the tool ships: lines `ROLE: PATTERN`, each pattern written like a line of a `.gitignore` file.
