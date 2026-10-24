# Project workflow

At the S3 source boundary, `check`, `context` and guarded `docs` consume structural Project preparation. The standalone Run uses a complete model-derived hierarchy for Assessment, Implement and Verify. CLI Apply composes Assessment, Implement and Verify and publishes only the independently reopened stored acceptance after all Run resources close. Extract, Cockpit, interactive authoring and Setup are retired. The [delivery plan](implementation-plan.md) owns current implementation and evidence authority.

## Connect and inspect

```text
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context product/model --repo .
```

The default manifest is `.markitect/model.yaml`. `--model` selects another exact manifest; repeat `--dependency PATH` for every dependency. There is no folder or registry discovery. The manifest and lock satisfy the [canonical v1 contract](design/canonical-model/minimal-language/language-contract.md).

An omitted revision observes the actual draft, including Gitignored files. `--revision REF` resolves once and observes that exact commit. Neither preview supplies a publication guard. Public fixed previews refuse unbound present-day external inputs. Closed Runs retain already captured external model/instruction bytes in their candidate storage and use those exact values for fixed candidate observation; they do not reread mutable external paths or vendor them into product files.

## Model and readiness

`check` defaults to the Model check, registry, complete file accounting, authored execution declarations and static contexts. `context <package>` reports selected Definitions, provenance and inclusion reasons for that Package, including ordinary Root `.`.

`markitect.execution/v2` declares ordered Manager instruction lists, ownership, scope and project checks. Runtime profiles, provider settings and invocation limits have separate owners. `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH` additionally prepares closed static native/check metadata from acquired bytes. The [Codex](../internal/adapters/codexappserver/README.md), [Claude](../internal/adapters/claudecode/README.md) and [check](../internal/adapters/checks/README.md) adapters own their exact profile versions and supported metadata. Image preparation starts no provider or project command. Host `process` check profiles are retired. Static success establishes no native runtime capability.

## Generated documentation

```text
go run ./cmd/markitect docs --repo .
go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
```

Read `view` in the preview. `canonicalModelDigest` identifies model meaning; `contentDigest` hashes rendered Markdown; `previewDigest` binds observed sources, target bytes/existence/mode and generated content. Use only the returned `previewDigest` with `--expect`; changed sources or targets require a fresh preview. Fixed-revision views never write. Protected model/control/instruction inputs cannot be targets. The independent document guard records actual completed paths and read-back.

Previews may include explicit external sources. `docs --write` requires every acquired model, control and instruction source to be repository-bound because external target disjointness has not been established.

## Explicit Runtime and invocation files

Copy [runtime-profile.example.yaml](../examples/runtime-profile.example.yaml) and [run.example.json](../examples/run.example.json) outside the repository and replace every uppercase placeholder. Source configuration does not qualify a provider.

- `ENGINE_EXECUTABLE` is the explicit native absolute engine path. Escape backslashes in JSON and quote YAML paths correctly.
- `IMAGE_ID`, `AGENTD_BUILD` and `CODEX_SCHEMA_DIGEST` identify the pinned image and its actual observed build metadata. `ACCOUNT_MODEL` selects the normal account model; check profile names match declared Project checks.
- `ACCOUNT_LOGIN_PATH` is the dedicated normal-account login JSON outside the realization tree. The `example-account` reference matches both files. The provider owns login provisioning and expiry behavior.
- `STATE_DIRECTORY` is an absolute path outside the repository. `PRODUCT_BUILD` is the operator's build assertion; retain the frozen source SHA separately.
- `DEADLINE_UTC` is an absolute RFC3339 UTC deadline ending in `Z`. Manager count, messages, per-message timeout, cleanup and persistence all have finite positive limits.

The [example tests](../internal/bootstrap/configuration_examples_test.go) substitute both exact files, route the explicit Apply invocation through the CLI, and prepare the captured Runtime with controlled image metadata. They check native v3 settings, the shared account reference, and declared Check requests, then stop at a deliberate boundary before runtime startup or credential reads. A separate negative case keeps invalid preflight free of metadata and state acquisition. These are static composition and CLI transport tests; actual account login and Apply execution require separate native evidence.

## Execute, verify, and apply

Use explicit committed selections and finite invocation limits:

The target must be on an isolated non-protected branch, for example `git switch -c dev/realization`, with its tracked baseline committed. Apply checks this existing write policy before opening any Manager resources and again during publication. Accounted ignored inputs remain part of the captured baseline.

```powershell
& "ABSOLUTE_MARKITECT_BINARY" apply --repo "REPOSITORY" --runtime-profile "ABSOLUTE_RUNTIME_YAML" --run-config "ABSOLUTE_RUN_JSON"
```

Use the same quoted binary without `&` in a POSIX shell. `--model` and repeated `--dependency` select committed inputs. `--revision` and `--metadata` remain preparation options. A successful preflight starts real Run environments, provider Sessions and required checks. Exactly one executable Codex account profile is selected by its presence in the captured Runtime document; every profile uses the invocation engine. Claude is not an executable Apply provider.

The standalone [Run](../internal/run/README.md) gives every Manager its own fresh environment and Git copy, direct delegation/questions and reviewed upward integration. Questions stay inside Root's hierarchy. A material missing decision or contradiction produces a model blocker; correct the model outside the Run. There is no answer command or preloaded owner decision. Assessment judges model readiness and Approve always leads to Implement, even for no-change. Implement submits a candidate; fresh Verify alone can approve publication. Verify Reject carries mandatory Findings into fresh Implement and Verify Runs with the original shared allowance and deadline. Assessment Reject, model blockers and technical or closure failures stop publication.

### Invocation configuration

The closed UTF-8 JSON uses `formatVersion: 3`. Required fields are `repositoryIdentity`, `productBuild`, `stateDirectory`, `engineExecutable`, `deadline`, `maxManagers`, `maxMessages`, `messageTimeoutSeconds`, `cleanupTimeoutSeconds`, `persistenceTimeoutSeconds` and `credentials`. Optional `workDeadline` reserves part of the whole deadline and must precede it. Unknown, duplicate, null, relative-path and invalid values are refused. Old strategy/role, ownerRequest, helper, harvest and ownerDecisions fields are not accepted. There are no ambient defaults. See the exact [example](../examples/run.example.json).

The example allows 600 seconds per message Exchange, including awaited child work and terminal drain. An observed successful native Library Turn alone exceeded the previous 120-second example limit; 600 seconds was the finite bound used in that native evidence. Configure this limit for the project within the original overall deadline and message allowance; the example does not guarantee that every Exchange will fit.

### Dedicated credential references

Each exact credential reference selects a provider, mode and absolute native file. Codex accepts `dedicated-runtime-login`; Claude accepts `dedicated-runtime-token`. The Codex API-key path is retired. Parsing and static metadata preparation do not read credentials. Only the native resolver reads the selected regular file with path/link checks, safe errors and a 1 MiB bound. The provider owns normal-login validation and the session copy; source files are not rotated by the resolver.

### Results and evidence

The CLI names the project state directory and an immutable `<state>/<project>/applies/<id>.json` report. Run records live under `<state>/<project>/runs/`, candidates under `candidates/`, Notes under `notes/` and native transcripts under `transcripts/<apply-id>/`. The retained Apply report and printed JSON use the same final status/outcome projection. Standalone Run persists the exact input reference, Project binding, technical and semantic outcomes, required findings, handoffs, actual check receipts and closure observations, plus concrete native identity observations when available. Candidates are independently verified self-contained bundles; external captured inputs remain in that same candidate directory. Shared Notes retain their Run/Manager/context and survive concurrent appends. New adapter instances reopen candidates/results after temporary environments close.

## Existing repositories

Author the intended canonical v1 model and explicit dependencies, declare the `ownership` areas in `.markitect/execution.yaml` and register the realization files in each owner's `.markitect/registry/<package>/registry.yaml`, as exact files or as directories ending in `/`. A file may realize several Definitions but has exactly one effective registration and one owner, the most specific area. An exceptional ignore names one exact observed file with a reason. Missing entries, missing paths, duplicate or wrong-owner registrations, files without an owner and overlaps with protected inputs fail before any Agent work; Apply repairs none of them. Gitignore is not coverage.

There is no automatic conversion, scaffold or Brownfield runtime. Earlier workflows remain reproducible from their [immutable source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).

## Record preparation and interrupted work

Preparation and generated-document operations create no Run or candidate. Preserve actual diagnostics and completed document paths. A Run that exhausts its limits or has uncertain observation/closure remains incomplete; its mandatory result records what happened. Stored candidates and Notes survive closure, but no automatic resume of old Manager environments exists. Apply uses a state-directory invocation lock and guarded per-path publication, preserves intervening user changes, reports partial transfers and never commits or pushes the owner checkout.

## Acceptance and status

Exit 0 means the requested stage passed; 1 is known failure; 2 is invalid usage; 3 is incomplete observation, I/O/budget refusal or unavailable operation. A committed model binds a specification revision, not an authenticated human decision. Local source checks, physical container proofs, native provider behavior, hosted CI, integration, releases and owner acceptance remain distinct evidence.
