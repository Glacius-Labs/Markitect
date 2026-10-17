# Readinglog test project

Readinglog starts with the actual existing Python 3.13 standard-library add/list CLI and its public baseline regression test. The complete canonical snapshots below own the selected incremental requirements. The imported ordinary vocabulary uses only the existing `Software.Concept`, `Software.Rule` and `Software.UseCase` Kinds. One local `readinglog` Package describes this small application; its internal implementation organization remains free.

## Select one requirement snapshot

| Snapshot | Model identity/version | Complete current intent |
| --- | --- | --- |
| [00-baseline](snapshots/00-baseline/model.yaml) | `readinglog` / `0.1.0` | Existing add/list commands, exact book record shape, input normalization, JSON array persistence, output/error behavior, tests and usage documentation. |
| [01-finish-status](snapshots/01-finish-status/model.yaml) | `readinglog` / `0.2.0` | Baseline plus idempotent `finish --id` and optional `list --status unread\|finished`, preserving the unfiltered list envelope and insertion order. |
| [02-import](snapshots/02-import/model.yaml) | `readinglog` / `0.3.0` | Previous intent plus `import --csv`: exact header/columns, the same normalization and identity rules as add, whole-file validation, no partial changes, input-order append and atomic commit. |
| [03-summary](snapshots/03-summary/model.yaml) | `readinglog` / `0.4.0` | Previous intent plus read-only lexical author summaries over all books/pages and finished counts. |

Each snapshot has an ordinary v1 manifest, a complete Definition root and the same external dependency lock. Later snapshots change the relevant existing Definitions and add the newly required UseCase; they do not depend on implicit overlays, folder discovery or a workflow language. No later author filters, export, term rename or historical team/station choreography is included.

## Exact starting code and provenance

[sources.yaml](sources.yaml) binds public requirements, actual baseline code/test bytes, dependencies and illustrative control derivation to immutable Main `dc4910b9810c5aaa4deecf1ff74734ee97badc13`, before retirement of the old case assets. The [preserved public contract](sources/public-contract.md) is a verbatim source copy for authoring provenance. Its full future feature list is operator material, excluded from selected actor packets. The original is also available at the [fixed Git revision](https://github.com/Glacius-Labs/Markitect/blob/dc4910b9810c5aaa4deecf1ff74734ee97badc13/experiments/case-playground/cases/readinglog/README.md).

The shared [application](template/realization/app.py) and [public baseline test](template/realization/tests/test_baseline.py) are copied verbatim. They implement and exercise the original add/list starting point; they contain no finish/import/summary implementation. The initial source has not been repaired or upgraded here, and copying it does not claim full conformance to every modeled requirement. In particular, the public contract requires JSON input errors, while the supplied application's argument parser retains its original usage-error handling. Preserve that visible initial condition for both arms. Future work must preserve accepted baseline behavior and meet the selected model; source and structural checks do not establish behavioral acceptance.

The first fixture wording required an error field but left the response shape too open. All four interfaces now require exactly one JSON object on stdout with only a nonempty string `error` field, exit 2 and unchanged stored data. This corrects the fixture's wording; earlier structural preparation did not establish that exact behavior. The copied application, public test and preserved source contract are unchanged. Ordinary packets must express the same precision and bind the updated model sources.

The [template control file](template/.markitect/execution.yaml) supplies illustrative finite preparation settings and the Python test command bound to `readinglog/interface`. It grants no execution authority and selects no installed Host profile. [Exact registry entries](template/.markitect/registry/readinglog/registry.yaml) account for only the two existing baseline files. Register actual new artifacts when authored. The Core and Software dependency inputs live once under `template/.markitect/dependencies/`, copied byte for byte from the integrated Library r2 inputs with their existing identities and digests.

## Materialize the current packet

An actor packet contains the shared current starting realization, template controls/dependencies and exactly one selected canonical snapshot. Exclude this operator guide, `sources.yaml`, `sources/`, all other snapshots, hidden expectations and evaluator answers. The ordinary documentation counterpart must contain exactly the same selected intent and receive exactly the same initial application/test bytes. It must not receive later-stage requirements.

For a fresh initial repository, from a Markitect checkout in PowerShell:

```powershell
$readinglogSource = 'experiments/test-projects/readinglog'
$readinglogStage = '01-finish-status'
$readinglogProject = Join-Path ([System.IO.Path]::GetTempPath()) ('readinglog-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $readinglogProject | Out-Null
Copy-Item -LiteralPath "$readinglogSource/template/.markitect" -Destination $readinglogProject -Recurse
Copy-Item -LiteralPath "$readinglogSource/template/realization" -Destination $readinglogProject -Recurse
Copy-Item -LiteralPath "$readinglogSource/snapshots/$readinglogStage/model.yaml" -Destination "$readinglogProject/.markitect/model.yaml"
Copy-Item -LiteralPath "$readinglogSource/snapshots/$readinglogStage/model-lock.yaml" -Destination "$readinglogProject/.markitect/model-lock.yaml"
Copy-Item -LiteralPath "$readinglogSource/snapshots/$readinglogStage/model" -Destination "$readinglogProject/.markitect" -Recurse
git -C $readinglogProject init --initial-branch=dev/readinglog
git -C $readinglogProject add .
git -C $readinglogProject commit -m 'Selected Readinglog requirements and actual shared baseline'
go run ./cmd/markitect check --repo $readinglogProject --model .markitect/model.yaml --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml
go run ./cmd/markitect context readinglog --repo $readinglogProject --model .markitect/model.yaml --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml
```

Use `00-baseline` when inspecting the initial intent alone. Inspect actual compilation digests, Definition provenance and context inclusion reasons; `--revision HEAD` binds committed sources. From the target's `realization/`, the CLI is `python -B app.py --db PATH COMMAND ...` and its test command is `python -B -m unittest discover -s tests`. These source commands are not a study protocol or a provider/Apply execution grant.

For an **isolated paired task**, both arms must start from the same explicitly frozen initial artifact bytes for that pair. If a later task needs a prior implementation, that starting implementation must be independently selected and bound for the pair; this fixture supplies no finished extension or gold result.

For a **longitudinal sequence**, share starting application/test bytes only at the first stage. At each later stage replace only the current manifest/lock/Definition sources with the next snapshot, and continue each arm's own actual prior realization, including its code, tests, documentation and artifact registry. Do not recopy the template or reset both arms to a gold implementation. Accumulated drift remains an outcome. A separate comparison harness owns run settings, axes and independent assessment; these domain requirements add no such framework and authorize no runs.
