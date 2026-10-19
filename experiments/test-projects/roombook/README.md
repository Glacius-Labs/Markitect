# Roombook test project

Roombook is a greenfield Python 3.13 standard-library room reservation CLI. These complete canonical snapshots describe the selected incremental intent. There is no supplied application, test implementation, reference solution or outcome evaluator. The ordinary imported `Software.Concept`, `Software.Rule` and `Software.UseCase` vocabulary is unchanged; one local `roombook` Package describes this small application.

## Select one requirement snapshot

| Snapshot | Model identity/version | Complete current intent |
| --- | --- | --- |
| [01-book-list](snapshots/01-book-list/model.yaml) | `roombook` / `0.1.0` | Book/list, normalized case-sensitive rooms, exact UTC-minute timestamps, increasing non-reused integer IDs, half-open active conflicts, ordered listing, durable atomic JSON state and errors without mutation. |
| [02-cancel](snapshots/02-cancel/model.yaml) | `roombook` / `0.2.0` | Previous intent plus idempotent cancel, retained canceled history and release of its interval. |
| [03-summary](snapshots/03-summary/model.yaml) | `roombook` / `0.3.0` | Previous intent plus lexical room summaries of active counts and exact minutes, omitting rooms without active reservations. |

Every snapshot contains its complete Definition root, ordinary v1 manifest and the same exact dependency lock. No overlay, implicit discovery or workflow language is required. The first snapshot retains the public book/list contract's recognition of canceled history and active-only conflicts; it supplies no `cancel` command until the second snapshot. Later filters, export, domain renaming and historical team/station/isolated-Main choreography are excluded.

## Source and explicit refinements

[sources.yaml](sources.yaml) binds the [verbatim public contract](sources/public-contract.md), dependencies and control derivation to immutable pre-retirement Main `dc4910b9810c5aaa4deecf1ff74734ee97badc13`. The original contract also remains available at its [fixed Git revision](https://github.com/Glacius-Labs/Markitect/blob/dc4910b9810c5aaa4deecf1ff74734ee97badc13/experiments/case-playground/cases/roombook/README.md). The preserved source and this guide are operator material outside actor packets; the source's historical process instructions do not become domain requirements.

The owner accepted one previously unspecified domain boundary for this new fixture: Summary groups only active reservations. A room whose entire history is canceled is omitted, and all-canceled state yields an empty `rooms` array. Canceled history stays in `list`. This is an explicit choice for the new fixture, not a claim that the old README already mandated that exact result or that an oracle observed it. It introduces no new status or historical oracle change. The ordinary documentation counterpart must express the same choice in the Summary stage.

The fixture also fixes a delivery boundary for portable preparation: invoke `python -B app.py --db PATH COMMAND ...` from `realization/`, and run `python -B -m unittest discover -s tests` there. Internal source/test organization remains free within that standard-library entry point; no production framework is prescribed. Code, meaningful tests, run/test instructions, data/failure behavior and the concurrent-writer limitation remain required.

The shared [execution file](template/.markitect/execution.yaml) is derived from the integrated Library r2 preparation example with Roombook profile names and the declared Python test command. Its budget (20 minutes in total, 8-minute turns, 60 Agent turns, 4 rounds) is sized for one Apply of one snapshot. The check profile `python-tests` is meant for the [Python runtime image](../../../packaging/runtime/README.md); its Host profile names the image's small wrapper `/opt/markitect/bin/python` as executable (a regular file in a small directory; the Agents use the `python` and `python3` links), and the command is `-B -m unittest discover -s tests`. Profile names and settings establish no installed Host profile, runtime capability or execution authority.

**Ownership layout.** The delivery boundary puts `app.py` and `tests/` directly in `realization/`, and under a layout the files directly in an artifact root belong to Root. The file declares the artifact root `realization` and one directory exception that gives the whole root to the `roombook` Package, so the domain Package realizes the application and tests while Root coordinates and writes no domain logic. A mirrored `realization/roombook/` directory with a Root-owned entry point and a second exception for `tests/` was rejected as more structure than a single-Package project needs. Core and Software inputs live once under `template/.markitect/dependencies/` with their original identities, versions and digests. No initial artifact registry is needed because there are no realization artifacts; register exact files when authored.

## Materialize the current packet

Include template controls/dependencies and exactly one selected canonical snapshot. Exclude this guide, `sources.yaml`, `sources/`, `oracle/`, all later snapshots, hidden expectations and evaluator answers. The ordinary documentation actor receives exactly the same current intended meaning and the same explicitly selected initial artifact state.

For a fresh initial repository, from a Markitect checkout in PowerShell:

```powershell
$roombookSource = 'experiments/test-projects/roombook'
$roombookStage = '01-book-list'
$roombookProject = Join-Path ([System.IO.Path]::GetTempPath()) ('roombook-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $roombookProject | Out-Null
Copy-Item -LiteralPath "$roombookSource/template/.markitect" -Destination $roombookProject -Recurse
Copy-Item -LiteralPath "$roombookSource/snapshots/$roombookStage/model.yaml" -Destination "$roombookProject/.markitect/model.yaml"
Copy-Item -LiteralPath "$roombookSource/snapshots/$roombookStage/model-lock.yaml" -Destination "$roombookProject/.markitect/model-lock.yaml"
Copy-Item -LiteralPath "$roombookSource/snapshots/$roombookStage/model" -Destination "$roombookProject/.markitect" -Recurse
git -C $roombookProject init --initial-branch=dev/roombook
git -C $roombookProject add .
git -C $roombookProject commit -m 'Selected Roombook requirements and greenfield starting state'
go run ./cmd/markitect check --repo $roombookProject --model .markitect/model.yaml --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml
go run ./cmd/markitect context roombook --repo $roombookProject --model .markitect/model.yaml --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml
```

Inspect actual model digests, selected Definitions, source provenance and context inclusion reasons. `--revision HEAD` binds the committed packet. Structural compilation and context establish source consumption; no application behavior or acceptance is established by these commands.

For an **isolated paired task**, freeze identical initial artifact bytes for both arms of that pair. The first task can use this empty realization. A later task needing existing code must bind a separately selected common starting implementation; this source fixture supplies no completed extension or gold result.

For a **longitudinal sequence**, share the empty initial state only at the first stage. Later, replace only current manifest/lock/Definition sources with the next snapshot and continue each arm's own actual prior code, tests, documentation and registry. Do not recopy the template or reset to a gold implementation; accumulated drift remains an outcome. A separate harness owns any axes, run configuration and independent expectations. These files prepare source requirements and authorize no provider, Apply or study runs.

## Assess a realization

The [oracle](oracle/oracle.py) is a black-box assessor outside every actor packet: it never feeds back into Agents and runs only after an Apply. It drives the realized CLI through `python -B app.py --db PATH COMMAND ...` from `realization/` and prints a JSON report with a pass or fail per check, each citing the Definitions it checks. Checks are cumulative and follow the Definitions of the snapshot and the recorded owner choices; error wording and internal organization stay free.

```text
python -B experiments/test-projects/roombook/oracle/oracle.py --realization PATH/TO/realization --stage 1|2|3 [--python CMD]
```

Its [discrimination self-check](oracle/selfcheck.py) verifies that every cited Definition exists, that silent, always-error, empty and crashing command lines fail, and, for a reference or mutants supplied from outside the repository, that the reference passes every stage and each mutant fails a check. The repository holds no reference solution.
