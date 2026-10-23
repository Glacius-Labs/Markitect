# Project workflow

**Target/source boundary, 23 October 2026:** the [approved V1 plan](implementation-plan.md#apply-rebuild-plan) commissions source replacement. The temporary public Apply already uses independent assignments, focused reviews and an Apply-wide Git lifetime. Fresh full-hierarchy Runs and closed-Run publication remain the S2/S3 target. Source implementation approval grants no real execution.

Markitect's current command surface includes deterministic check/context/docs, explicit Apply and Setup during the S1 transition. Extract, Cockpit and interactive authoring are retired. Public Apply temporarily uses independent Manager assignments with focused model assessment/Verify and guarded publication. The [approved V1 plan](implementation-plan.md) replaces that runtime with fresh full-hierarchy Runs; source tests do not establish native readiness or owner acceptance.

**Closed Runs:** no question or answer crosses Root to the user or invoking AI. A material missing decision or contradiction ends Apply without publication; correct the model outside the Run. The [V1 architecture](design/v1-execution-architecture.md) owns the replacement target.

## Connect and inspect

Use Git with an existing commit and the root-layout CLI:

```text
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context product/model --repo .
```

The default manifest is `.markitect/model.yaml`. `--model` selects another explicit manifest. Supply every required dependency manifest with repeated `--dependency PATH`; there is no discovery from folders, registries or old project formats. The manifest and `model-lock.yaml` must satisfy the [canonical v1 contract](design/canonical-model/minimal-language/language-contract.md).

An omitted revision observes the actual working draft, including Git-ignored files. `--revision REF` resolves once and captures exactly that full commit. Neither preview supplies an Apply publication guard. A fixed preview refuses present-day external inputs without historical revision binding, including an external Runtime profile document even when REF is HEAD.

## Model and readiness

`check` requests structural preparation by default: actual Model check, execution declarations, registry, complete file accounting and static context selection. `context <package>` uses the same preparation and reports selected Definitions, provenance and inclusion reasons for one local Package, or `.` for the ordinary root.

Add `--metadata --runtime-profile ABSOLUTE_DOCUMENT_PATH` to request static Agent/Check profile metadata and settings preparation. The [Codex Agent profile](../internal/adapters/codexappserver/README.md) uses `profileVersion: 3`; the [Claude Agent profile](../internal/adapters/claudecode/README.md) uses `profileVersion: 1`. Each uses an explicitly configured engine to inspect an immutable image ID and copy whole `/opt/markitect/runtime.env` metadata from a created, never started container. The Codex schema digest is a build assertion, not an observed schema inventory. The [container Check profile](../internal/adapters/checks/README.md) uses `profileVersion: 1` and observes the whole executable and required image paths. The historical `process/v1` check contract remains available for metadata preparation; Apply refuses it because there is no native Host process check runner. A metadata pass establishes neither runtime nor provider capability. Unrequested metadata and settings report `requested: false`; absent or incomplete required observations remain incomplete.

## Generated documentation

```text
go run ./cmd/markitect docs --repo .
go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
```

Read the nested `view` in the preview. `canonicalModelDigest` identifies checked model meaning; `contentDigest` hashes rendered Markdown bytes; `previewDigest` binds observed sources, target bytes/existence/mode and generated content. Pass only the returned `previewDigest` to `--expect`. Changed sources or targets refuse the old plan. Fixed-revision views never write, and protected model/control/instruction inputs cannot become generated targets. A write uses an independent documentation guard and actual read-back, with completed paths retained if a later check fails. It is separate from artifact Apply.

Documentation previews may include explicitly selected external sources. For this increment, `docs --write` requires every acquired model, control and instruction source to have an actual repository binding. Otherwise it returns `not_completed` before effects because target disjointness has not been established. Repository-bound dependencies remain writable; an external source is not silently reclassified or followed through an alternate native spelling.

## Set up a Runtime directory

`setup` prepares a Runtime directory so that `apply` needs no harness scripts. Build or pull the runtime image first, then run:

```text
go run ./cmd/markitect setup --repo REPOSITORY --runtime ABSOLUTE_RUNTIME_DIR --image IMAGE_REFERENCE --credential ABSOLUTE_TOKEN_FILE --agent-model MODEL --check go=/usr/local/go/bin/go
```

It reads which Agent profiles and check profiles the project selects from the working draft, so every check profile needs one `--check PROFILE=EXECUTABLE` (the executable inside the image) and an unused one is refused. It asks the container engine for its version and volume-subpath support (Docker 26 or newer), resolves the image to its immutable ID and reads `/opt/markitect/runtime.env` from a created, never started container. It checks the token file's path, type and owner-only mode without opening it; the file must be a regular file outside the repository and its bytes are never read, copied or printed.

The Runtime directory must be outside the repository. Setup creates it private and writes `runtime.yaml` (every Agent profile as a Claude Code profile with the required `--agent-model` (there is no default model) and `--effort`, default `high`; every check profile as a container check on the image, with no `env`) and `run.json` (the [invocation configuration](#invocation-configuration), with `state/` in the same directory). Before writing, it decodes `run.json` as `apply` does and runs `runtime.yaml` through the settings and metadata preparation of `check --metadata --runtime-profile`. Optional `--store`, `--request`, `--deadline` and `--engine` fill the remaining fields; `--model` and `--dependency` select the project as for `check`. Known limit: a check container's `/tmp` holds 64 MiB, and setup cannot yet add `env`, so a Go check whose builds need more room (for example `GOTMPDIR=/home/agent`, as the library demonstration sets) needs that entry added to `runtime.yaml` by hand.

The JSON report lists the bound image ID, agentd and Claude Code versions and engine version, both files with their SHA-256 and whether each was `written` or `unchanged`, and the exact `apply` command to run next. Running setup again with the same inputs changes nothing. It refuses while an Apply holds the state directory. `productBuild` carries the Git revision only for a binary built with `go build` in a checkout; `go run` writes the plain version, so bind frozen runs with a built binary.

## Execute, verify, and apply

Use an existing committed repository, explicit dependency manifests, a Runtime profile document and an absolute invocation configuration path:

```text
go run ./cmd/markitect apply --repo REPOSITORY --runtime-profile ABSOLUTE_RUNTIME_YAML --run-config ABSOLUTE_RUN_JSON
```

`apply` accepts the same `--model` and repeated `--dependency` selections as preparation. It captures committed inputs and passes structural/settings preparation before starting the required independent Manager assignments. Parents delegate, review and integrate fixed Git handoffs; focused model assessment and independent Verify surround Implement. Publication uses the guarded source boundary and never creates a user commit. This temporary composition retains its earlier Apply-wide Git lifetime until S2/S3 replace it. `--revision` and `--metadata` are preview-only options. Missing explicit execution inputs refuse before acquisition; the state-directory lock prevents a second command from acquiring the same state.

Extract is retired from the active command and source tree. Its prior behavior remains in the [verified source archive](development/v1-source-archive-20261023.md).

Managers ask along the tree only; nobody outside it answers during a run ([accepted workflow](design/canonical-model/manager-apply-workflow.md#questions-and-escalation)). A material decision the model leaves open without delegating it, or a contradiction, is a model problem: the Model assessment reports it as a finding, a Manager at work reports it to its parent, and Root's confirmation ends the run Failed without publishing. Author the correction of the model outside the run and start a new Apply, which assesses the corrected model again. No answer command, owner channel or preloaded owner decision exists; a run configuration that still has `ownerDecisions` is refused.

### Invocation configuration

`--run-config` selects a closed UTF-8 JSON object with `formatVersion: 2`. Public Apply requires positive `maxMessages` and `messageTimeoutSeconds` in addition to the required identity, state, engine, helper, harvest and credential mappings. Optional `deadline` is an absolute UTC RFC 3339 bound; it combines with the project execution-time limit. Public Apply refuses the legacy `workDeadline`.
Cleanup still needs finite time outside the Apply ceiling.
A Run is closed, so no owner decision is bound with it: `ownerDecisions` ([DEC-053](concepts/register.md#dec-053-owner-decisions-bind-the-whole-run), superseded) is refused like any unknown key. Unknown, duplicate, null, relative-path or invalid values refuse construction; files of the earlier version 1 (with `allocation`, `restoreLimits` and `engineLimits`) are refused. There are no environment defaults. This file supplies concrete invocation inputs; project policies and Agent/Check model, effort, image and resource settings remain in their existing execution and Runtime profile documents.

The following is an illustrative shape. Supply actual native absolute paths, immutable images, dedicated credentials and limits appropriate to the frozen project; these example assertions do not certify a runtime:

```json
{
  "formatVersion": 2,
  "repositoryIdentity": "library-project",
  "productBuild": "operator-identified-frozen-build",
  "ownerRequest": "Realize the captured model.",
  "stateDirectory": "/absolute/markitect-state",
  "engineExecutable": "/absolute/container-engine",
  "maxMessages": 40,
  "messageTimeoutSeconds": 60,
  "workspaceHelper": {
    "name": "library-workspace-helper",
    "image": "sha256:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
    "labels": {"markitect.store": "library-project"},
    "user": "1000:1000",
    "cpus": 1.5,
    "memoryBytes": 134217728,
    "pids": 32,
    "tmpBytes": 16777216,
    "network": ""
  },
  "harvest": {
    "maxEntries": 100000,
    "maxBytes": 67108864
  },
  "credentials": {
    "runtime-credential-reference": {
      "provider": "codex-app-server",
      "mode": "dedicated-runtime-login",
      "path": "/absolute/dedicated-account-login.json"
    }
  }
}
```

`harvest` bounds repository/candidate captures to `maxEntries` and `maxBytes`, within the technical ceilings of 1048576 entries and 4 GiB. A bound exceeded cannot establish a complete capture. `maxMessages` is the finite assignment invocation allowance; `messageTimeoutSeconds` bounds one message. Cancellation stops active assignments and retains reported partial observations. `productBuild` is the operator assertion identifying the frozen build, not an automatically verified Git SHA.

`engineExecutable` must equal the already validated literal in every selected executable Agent and Check profile. The helper is its own explicit container capability for workspace transfer: immutable image ID, non-root numeric user/group, finite positive CPU/memory/process/tmp limits and labels are required. Its work directory is `/work`; invocation JSON cannot add mounts. Empty `network` explicitly leaves network selection to the engine; use `"none"` to disable networking. Host networking is refused. These helper fields do not replace Agent or Check profile resources.

### Dedicated credential references

`credentials` maps each profile's exact reference to one provider, mode and native absolute file path. Accepted pairs are `codex-app-server` with `api-key` or `dedicated-runtime-login`, and `claude-code` with `dedicated-runtime-token`. The operator must provide a dedicated runtime file; there is no desktop login discovery or environment fallback. Parsing configuration and preparing metadata do not read these files. Only the actual provider resolver reads its mapped regular file, with native path/link checks and a 1 MiB bound. Errors do not include credential bytes.

API keys and Claude tokens retain literal file bytes. A dedicated Codex runtime login retains the supplied subscription-login semantics; the adapter strips refresh material from the container copy. The Orchestrator never refreshes or rotates the source login. Keep credentials and external Runtime profile inputs outside the realization tree, and account for any repository-bound invocation file explicitly.

### Results and evidence

The JSON result includes preparation, `assignment` (model/Root/accepted commits, messages, assessment, reviews, checks and cleanup observations), publication, `outcome` and `evidenceDirectory`. `status: passed` and `outcome: success` require accepted work, publication as applied/no change and no technical error. Model rejection, failed publication and incomplete technical observation remain distinct. Legacy Pass/Turn `run` projections belong to retained historical consumers.

Evidence remains under `STATE/runs/RUN`; the temporary assignment path retains its reported workspace volumes. There is no owner answer channel and no automatic resume. The V1 durable candidate/result replacement is separate source work.

## Existing repositories

Author the intended canonical v1 model and explicit dependencies, declare the `ownership` areas in `.markitect/execution.yaml` and register the realization files in each owner's `.markitect/registry/<package>/registry.yaml`, as exact files or as directories ending in `/`. A file may realize several Definitions but has exactly one effective registration and one owner, the most specific area. An exceptional ignore names one exact observed file with a reason. Missing entries, missing paths, duplicate or wrong-owner registrations, files without an owner and overlaps with protected inputs fail before any Agent work; Apply repairs none of them. Gitignore is not coverage.

There is no automatic conversion, scaffold or Brownfield runtime. Earlier workflows remain reproducible from their [immutable source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).

## Record setup and interrupted work

Preparation and generated-document operations create no run, task, budget, candidate or execution journal. Preserve their acquisition diagnostics and generated-document completed-path reports. Apply instead uses the explicit state directory and its process lock. There is no journal and no restore: an interrupted Apply (deadline, signal or crash) ends incomplete, and a new Apply starts again from the committed repository, never by resuming earlier assignment copies. Evidence of every run stays under `STATE/runs/`. Preserve it and resolve the reported condition before starting again. No standalone recovery command is exposed here. The state directory must not be inside the repository or contain it.

## Acceptance and status

JSON reports name the operation, requested stage and actual statuses. Exit 0 means the requested stage passed; 1 is a known failure; 2 is invalid usage; 3 is incomplete observation, I/O/budget refusal or unavailable operation. `failed` and `not_completed` remain distinct. A committed model binds a specification revision; it does not authenticate a human decision. Local source checks, hosted CI, provider execution, releases and owner acceptance remain separate evidence.
