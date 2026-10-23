# Library r2

This named successor refines the Library requirements for later matched implementation work. The canonical requirements are the 22 local Definitions under [project/.markitect/model/](project/.markitect/model/library/library.yaml), selected by [model.yaml](project/.markitect/model.yaml), identity `library-r2`, version `1.0.0`. This directory supplies requirements and preparation inputs. It contains no Library implementation or outcome evaluator.

The historical [Library M1/M2](../../../docs/design/canonical-model/library-example/README.md) and its independent expectations retain their original identity. This successor does not replace the first two-Apply feasibility target. The [measurement procedure](../../../docs/measurement.md#quality-first-comparative-evaluation) requires the ordinary documentation arm to receive the same intended meaning, interfaces, relationships and delivery obligations as this model. Future independent expectations must be frozen separately, outside both implementation writers' control. No comparison or provider run is authorized by these files.

## Requirement sequence

1. **Initial realization:** implement all local Definitions together. One initially empty library supports member registration and status, physical-copy registration, borrowing, returns, current-state inspection and read-only overdue notices. `lending/loan-policy` sets at most five open loans per member and a fourteen-calendar-day loan period. Membership eligibility, derived copy availability and overdue assessment have explicit cross-Package relationships. Refusals preserve observable state; repeated returns preserve the original return and any later loan of that copy.
2. **Policy delta:** starting from the accepted first realization, change only `spec.maxOpenLoans` in `project/.markitect/model/lending/loan-policy.yaml` from `5` to `3`. Preserve every other Definition and dependency. The policy already states how a lower admission limit treats existing loans and returns. This changes local model meaning while retaining the external dependency lock. A later implementation task uses the actual prior realization; it receives no prepared final implementation here.

The shared `library/callable-interface` is binding for every area: a local Go standard-library program at `realization/cmd/library`, module at `realization/go.mod`, UTF-8 JSON Lines on standard input/output, one response flushed before the next request, process-local sequential state, no persistence, and the complete record/operation/error contract `markitect.library-jsonl/v1`. `go build ./cmd/library` and `go test ./...` run from `realization/`. `library/realization` requires code, automated tests and concise usage documentation. Exact internal design, package and test layout remain implementation choices within those boundaries. Read the actual Definitions for all obligations; this guide is an orientation and change sequence.

## Derivation and clarification

[sources.yaml](sources.yaml) binds every copied model input to its exact source path, source revision and original Git-blob SHA-256. The baseline is Main `1f4a1ede214d05fd2fee615b3905207df7702426`. Local Definitions are derived from the historical M1 with these differences:

- `library/inputs` now explicitly defers transport, request/response fields and identifier behavior to `library/callable-interface`. Its former statement that transport was free contradicted the already binding Go JSONL contract. Exact error wording and the generated loan-identifier format remain free within that contract.
- `library/realization` expresses internal design freedom within the already binding Go standard-library restriction. Its former unrestricted word “Frameworks” could suggest permission to add external dependencies.
- `lending/borrow` resolves a previously unspecified boundary: a syntactically valid borrowing date whose calculated due date would exceed the four-digit Gregorian wire range is refused with an understandable reason and unchanged state. The exact policy interval and existing error response remain applicable; dates are not widened and the interval is not shortened. The owner accepted this minimal choice for the named successor during refinement. Historical M1 did not explicitly mandate this outcome when this successor was derived; since 20 October 2026 its `lending/borrow` states the same refusal. Its oracle remains unchanged.

All other domain, interface and delivery requirements are retained. The local manifest is renamed and versioned for this successor. The existing local `LendingPolicy` Kind and its two Properties are retained unchanged. The imported ordinary `Software.Concept`, `Software.Rule` and `Software.UseCase` vocabulary and the Core snapshot are copied byte for byte; no new Kind or language feature is introduced. Their identities, versions and canonical digests are bound by [model-lock.yaml](project/.markitect/model-lock.yaml). Physical dependency locations are explicit CLI inputs, never lock fields or machine paths.

The [execution file](project/.markitect/execution.yaml) uses the recursive-passes execution shape with explicit ownership: Root holds the repository, each Package its directory below `realization/`, and `library` owns `realization/application/` plus `realization/cmd/library/`. The application path leaves `realization/library` available as the output of the required Linux command `go build ./cmd/library`. It supplies structural preparation settings and references the model-required test command. Its finite values and profile names are illustrative preparation inputs; they establish no sufficient budget, installed profile, container image, runtime readiness or execution grant. This successor does not author a Runtime profile.

## Portable preparation

Copy the contents of `project/` into a fresh external Git repository, commit that requirement-only starting state, and run the current Main compiler against it. The project contains only protected model/dependency/control sources, so it has no realization registry entries yet. Add exact ownership entries when actual artifacts are authored. Keep this guide and `sources.yaml` outside the target repository; they are provenance and operator guidance, not additional target artifacts.

For example, from a Markitect source checkout in PowerShell:

```powershell
$libraryProject = Join-Path ([System.IO.Path]::GetTempPath()) ('library-r2-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $libraryProject | Out-Null
Copy-Item -LiteralPath experiments/test-projects/library-r2/project/.markitect -Destination $libraryProject -Recurse
git -C $libraryProject init --initial-branch=dev/library-r2
git -C $libraryProject add .
git -C $libraryProject commit -m 'Library r2 requirement baseline'
go run ./cmd/markitect check --repo $libraryProject --model .markitect/model.yaml --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml
go run ./cmd/markitect context lending --repo $libraryProject --model .markitect/model.yaml --dependency .markitect/dependencies/core/model.yaml --dependency .markitect/dependencies/software/model.yaml
```

Inspect the actual model digest, selected Definitions, provenance and inclusion reasons. Repeat `context` for `library`, `members`, `catalog`, `lending/reminders` or `.` when inspecting that responsibility. `check --revision HEAD` binds the committed starting model; an omitted revision observes the actual working draft. To inspect the policy delta, change the one field in the copied project and repeat preparation. Model compilation and static context establish structural consumption only; they do not establish behavior, Apply, comparative benefit or acceptance.
