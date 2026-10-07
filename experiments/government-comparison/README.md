# Government comparison preparation

Scientist owns this package. It implements the first finite preparation package
under [evaluation](../../docs/design/government/evaluation.md) and active
[coordination](../../docs/design/government/coordination.md). It changes no
production Markitect source and makes no product benefit claim. Live trials: **0**.

## Reproduce the fixture smoke

Run from the Scientist checkout using Python 3.13.3, Git 2.52.0.windows.1,
.NET SDK 10.0.103 and .NET/ASP.NET Core 10.0.3. NuGet.org access is needed for
the first locked restore. No provider call is made by these commands.

```powershell
$studyRoot = 'C:/Users/Consiliari/.codex/worktrees/government-scientist/Markitect/experiments/government-comparison'
python -m unittest discover -s $studyRoot -p test_harness.py -v
python "$studyRoot/smoke.py" --destination "$studyRoot/.study-data/my-new-smoke"
```

Every destination must be new. Existing attempts are never reset or deleted.
`smoke.py` creates two deterministic start commits and six independent clones,
releases task-1 inputs outside actor repositories, probes all six cells with an
explicit mechanical adapter, restores/builds Brownfield, exercises its public
API, restarts it on the same SQLite database, and checks the legacy retry. It
stops its API process and writes `smoke-result.json`, raw build/API logs and six
request/result/event ledgers. All six adapter results are `readiness_gap`;
successful interchange is not product readiness or an actor solution.

`prepare.py --destination ABSOLUTE_NEW_DIRECTORY` prepares only the starts/cells.
The Greenfield tree contains zero files (even brief, pins and `.gitignore` are
actor work). Brownfield is authored reference source with catalog, validated
orders, SQLite persistence/idempotency and a useful legacy alias. It supplies no
canonical Markitect/Government model. No production/customer history is claimed.
The Brownfield evidence packet names known ownership and the visible unmodeled
legacy path. Exact source, hashes and dated preparation results belong to
[preparation-freeze.json](preparation-freeze.json) and [evidence](evidence/README.md).

## Public handoff and staged requirements

```powershell
python "$studyRoot/release.py" --through-task 1 --condition greenfield --destination "$studyRoot/.study-data/released-greenfield-task1"
```

The operator passes only that release directory and the actor's own repository.
All arms get identical released requirements/checks at the same cutoff. Brownfield
also gets its provenance packet. Later cards and future check bodies are absent:
`release.py` builds a syntax-checked truncated public checker at declared task
boundaries. Each released input has a digest. Do not give actors this study
checkout, rubric, private files, prior cells/results, full checker or later cards.
The full public contract is for operator preparation, not an initial actor prompt.

Public checks consume a fresh isolated test database; do not run them against a
retained application database. The common sequence is an integration smoke, not
an exhaustive oracle. Task 3 additionally uses `--checkpoint PATH`, process
restart with the same database, then `--verify-restart PATH`. Each candidate is
assessed on runner-owned test data; no production data corpus or undeclared
migration/fault protocol is assumed. Actual injected save/recovery faults and
candidate-specific task-6 defects need source-mapped, equivalent procedures
before those checks can be claimed complete.

## Adapter and recording boundaries

[Public adapter contract](public/adapter-contract.md) was published early at
`775338555d2f2743128462af13d998c7d179c7ac`. Worker and Architect provide exact
product inputs, invocation argv, results and receipts against that contract.
`harness.py` currently runs bounded fixture probes only, validates start/input and
answer bindings, and preserves failed attempts. Live mode is deliberately closed.
It does not yet implement live process-tree containment, shared budget scheduling,
provider event ingestion, candidate acceptance or stateful resume semantics.

[Metrics schema](public/metrics.schema.json) preserves raw provider semantics and
unknowns. Input/output, cache and reasoning are not blindly added; missing tokens
or billed costs stay `null`. Parallel elapsed intervals form a union for wall
time; agent/person active time is separately counted. Every actor/reviewer/minister,
modeling step, retry, setup and coordinator/human intervention is charged to the
same trial. Preparation itself used Scientist and three subagents; this is shared
study-design overhead, not free arm modeling. Their token/cost receipts and total
active effort are unavailable and remain unknown. No trial effort has been observed.

The private assessor froze eight public-requirement groups and an independent
HTTP checker under ignored `.study-data/private/`; only digests are in the public
freeze. Syntax/source-mapping inspection has run, no private behavioral holdout
has run. Assessment happens after candidate freeze, separately per fresh test case;
hidden results never steer repair of that same trial. Task-6 mutation mapping is
necessarily per actual task-5 candidate and remains unset. Private files are not
copied into actors. The same-user sentinel probe demonstrates that current OS
rights permit external reads; an actual Actor-runner access probe still remains.
This is cooperative separation, with contamination recorded if access occurs.

## Next finite checkpoint

[Resource proposal](public/resource-proposal.json) proposes one exploratory trial
per six cells, 20 minutes/12 actor calls/80 provider turns/120,000 provider tokens
per task and 120 minutes/72 calls/480 turns/720,000 tokens per trial, parallelism 4,
two semantic repair rounds, one transport retry and 10 minutes human active work.
All descendants, product setup and ministry checks consume those same ceilings.
These are proposals, not measured/enforced limits or live authorization by fixture.

Local `codex-cli 0.130.0` supports noninteractive execution and structured event
output; [official documentation](https://learn.chatgpt.com/docs/non-interactive-mode)
describes `codex exec --json`. No actor/provider request was made. The actual
account-supported model ID, exact executable/config, subagent metering and common
aggregate cap enforcement must be pinned by a real runner-readiness checkpoint.
Unknown billing stays null; no new purchase/subscription is part of this package.

Overseer can adopt the proposed bounded profile within the existing Go. Scientist
then inspects Architect's Classic readiness packet and Worker's G1–G5 evidence,
freezes StudyClassicVersion/Government source/binary/module/config versions, wires
real adapters and runs the same public setup/task smoke. A readiness gap stays
open. Only after those gates and private oracle/rubric/access checks does the first
six-cell live feasibility pilot start. No repetition, precision or superiority
claim follows from one run per cell. Product defects go to their owner; Scientist
does not silently repair candidates.
