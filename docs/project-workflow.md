# Project workflow

Markitect's purpose is canonical model-first development with delegated realization. Current source exposes structural preparation, a guarded generated-model view and explicit controlled Apply/Answer commands. Apply composes recursive Managers, independent review, container sessions, candidate workspaces, container checks and guarded publication. Source availability does not establish provider readiness or successful complete realization; the [delivery plan](implementation-plan.md) and recorded validation evidence retain those separate scopes.

## Connect and inspect

Use Git with an existing commit and the root-layout CLI:

```text
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context product/model --repo .
```

The default manifest is `.markitect/model.yaml`. `--model` selects another explicit manifest. Supply every required dependency manifest with repeated `--dependency PATH`; there is no discovery from folders, registries or old project formats. The manifest and `model-lock.yaml` must satisfy the [canonical v1 contract](design/canonical-model/minimal-language/language-contract.md).

An omitted revision observes the actual working draft, including Git-ignored files. `--revision REF` resolves once and captures exactly that full commit. Neither preview supplies an Apply publication guard. A fixed preview refuses present-day external inputs without historical revision binding, including an external Orchestrator document even when REF is HEAD.

## Model and readiness

`check` requests structural preparation by default: actual compilation, execution declarations, registry, complete file accounting and static context selection. `context <package>` uses the same preparation and reports selected Definitions, provenance and inclusion reasons for one local Package, or `.` for the ordinary root.

Add `--metadata --host-profile ABSOLUTE_DOCUMENT_PATH` to request static Agent/Check profile metadata and settings preparation. The [Codex Agent profile](../internal/adapters/codexappserver/README.md) uses `profileVersion: 3`; the [Claude Agent profile](../internal/adapters/claudecode/README.md) uses `profileVersion: 1`. Each uses an explicitly configured engine to inspect an immutable image ID and copy whole `/opt/markitect/runtime.env` metadata from a created, never started container. The Codex schema digest is a build assertion, not an observed schema inventory. The [container Check profile](../internal/adapters/checks/README.md) uses `profileVersion: 1` and observes the whole executable and required image paths. The historical `process/v1` check contract remains available for metadata preparation; Apply refuses it because there is no native Host process check runner. A metadata pass establishes neither runtime nor provider capability. Unrequested metadata and settings report `requested: false`; absent or incomplete required observations remain incomplete.

## Generated documentation

```text
go run ./cmd/markitect docs --repo .
go run ./cmd/markitect docs --repo . --expect PREVIEW_DIGEST --write
```

Read the nested `view` in the preview. `canonicalModelDigest` identifies compiled model meaning; `contentDigest` hashes rendered Markdown bytes; `previewDigest` binds observed sources, target bytes/existence/mode and generated content. Pass only the returned `previewDigest` to `--expect`. Changed sources or targets refuse the old plan. Fixed-revision views never write, and protected model/control/instruction inputs cannot become generated targets. A write uses an independent documentation guard and actual read-back, with completed paths retained if a later check fails. It is separate from artifact Apply.

Documentation previews may include explicitly selected external sources. For this increment, `docs --write` requires every acquired model, control and instruction source to have an actual repository binding. Otherwise it returns `not_completed` before effects because target disjointness has not been established. Repository-bound dependencies remain writable; an external source is not silently reclassified or followed through an alternate native spelling.

## Execute, verify, and apply

Use an existing committed repository, explicit dependency manifests, a Host profile document and an absolute invocation configuration path:

```text
go run ./cmd/markitect apply --repo REPOSITORY --host-profile ABSOLUTE_HOST_YAML --run-config ABSOLUTE_RUN_JSON
go run ./cmd/markitect answer QUESTION_ID --text "Owner decision" --repo REPOSITORY --host-profile ABSOLUTE_HOST_YAML --run-config ABSOLUTE_RUN_JSON
```

Both accept the same `--model` and repeated `--dependency` selections as preparation. They capture the actual committed source and complete settings, acquire repository control, open the durable journal and call the controlled Core's `Execute` method. `answer` first records the actual owner answer through `Answer`, then continues execution. Reuse the original invocation configuration and source selections when answering or reopening; a changed captured basis cannot answer the old run. `--revision` and `--metadata` are preview options and are refused here. Missing explicit execution inputs leave Apply unavailable before source acquisition.

### Invocation configuration

`--run-config` selects a closed UTF-8 JSON object with `formatVersion: 1`. Every field below is required, including empty network strings and explicit limits. The one optional field is `ownerDecisions`, a list of the owner's earlier decisions as plain texts; every Manager and reviewer of the run sees them from its start ([DEC-053](concepts/register.md#dec-053-owner-decisions-bind-the-whole-run)). Unknown, duplicate, null, relative-path or invalid values refuse construction. There are no environment defaults. This file supplies concrete invocation inputs; project policies and Agent/Check model, effort, image and resource settings remain in their existing execution and Host profile documents.

The following is an illustrative shape. Supply actual native absolute paths, immutable images, dedicated credentials and limits appropriate to the frozen project; these example assertions do not certify a runtime:

```json
{
  "formatVersion": 1,
  "repositoryIdentity": "library-project",
  "productBuild": "operator-identified-frozen-build",
  "ownerRequest": "Realize the captured model.",
  "stateDirectory": "/absolute/markitect-state",
  "allocation": {
    "runs": 2,
    "sessionStarts": 80,
    "steps": 160,
    "executionTime": 28800000000000
  },
  "restoreLimits": {
    "pageSize": 256,
    "batches": 100000,
    "events": 1000000,
    "bytes": 268435456
  },
  "engineExecutable": "/absolute/container-engine",
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
  "engineLimits": {
    "startTimeout": 30000000000,
    "stopTimeout": 10000000000,
    "harvestEntries": 100000,
    "harvestBytes": 67108864,
    "record": {
      "encodedBytes": 16777216,
      "fileBytes": 16777216,
      "manifestEntries": 100000,
      "metadataRows": 100000
    }
  },
  "credentials": {
    "runtime-credential-reference": {
      "provider": "codex-app-server",
      "mode": "api-key",
      "path": "/absolute/dedicated-runtime-credential"
    }
  }
}
```

All duration numbers are integer nanoseconds, matching the existing Core types. Capacity and byte/count limits are positive integers; `restoreLimits.pageSize` is at most 256 and `engineLimits.record.encodedBytes` is at most 67108864. Core also enforces the captured project's budgets and policies. `productBuild` is the operator's explicit assertion identifying the frozen build, not an automatically verified Git SHA. `repositoryIdentity`, owner request and allocation participate in the bound run inputs.

`engineExecutable` must equal the already validated literal in every selected executable Agent and Check profile. The helper is its own explicit container capability for workspace transfer: immutable image ID, non-root numeric user/group, finite positive CPU/memory/process/tmp limits and labels are required. Its work directory is `/work`; invocation JSON cannot add mounts. Empty `network` explicitly leaves network selection to the engine; use `"none"` to disable networking. Host networking is refused. These helper fields do not replace Agent or Check profile resources.

### Dedicated credential references

`credentials` maps each profile's exact reference to one provider, mode and native absolute file path. Accepted pairs are `codex-app-server` with `api-key` or `dedicated-runtime-login`, and `claude-code` with `dedicated-runtime-token`. The operator must provide a dedicated runtime file; there is no desktop login discovery or environment fallback. Parsing configuration and preparing metadata do not read these files. Only the actual provider resolver reads its mapped regular file, with native path/link checks and a 1 MiB bound. Errors do not include credential bytes.

API keys and Claude tokens retain literal file bytes. A dedicated Codex runtime login retains the supplied subscription-login semantics; the adapter strips refresh material from the container copy. The Host never refreshes or rotates the source login. Keep credentials and external Host inputs outside the realization tree, and account for any repository-bound invocation file explicitly.

### Results and publication

The JSON result includes actual preparation diagnostics, control binding, execution state, tasks, checks, publication receipt and open owner questions. A waiting run returns `not_completed`; use its actual question ID with `answer`. Successful completion requires an actual `Applied` or `NoChange` result without an execution error. Failed, incomplete, exhausted and unavailable outcomes remain distinct. Publication uses the same Git repository adapter that captured the source, checks the captured guard and protects model/control/instruction inputs. It does not create a Git commit or push. No MCP server is provided by these commands.

## Existing repositories

Author the intended canonical v1 model and explicit dependencies, then register exact realization files in `.markitect/registry/<package>/registry.yaml`. File granularity is the author's choice; a file may realize several Definitions but has one explicit owner. An exceptional ignore names one exact observed file with a reason. Missing entries, missing paths, duplicate ownership and overlaps with protected inputs fail. Gitignore is not coverage.

There is no automatic conversion, scaffold or Brownfield runtime. Earlier workflows remain reproducible from their [immutable source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).

## Record setup and interrupted work

Preparation and generated-document operations create no run, task, budget, candidate or execution journal. Preserve their acquisition diagnostics and generated-document completed-path reports. Apply/Answer instead use the explicit state directory, process lock and actual journal. Reopening the same invocation restores recorded state under the configured restore bounds; it does not silently replay uncertain provider effects. Preserve incomplete observations and resolve the reported condition before continuing. No standalone recovery command is exposed here.

## Acceptance and status

JSON reports name the operation, requested stage and actual statuses. Exit 0 means the requested stage passed; 1 is a known failure; 2 is invalid usage; 3 is incomplete observation, I/O/budget refusal or unavailable operation. `failed` and `not_completed` remain distinct. A committed model binds a specification revision; it does not authenticate a human decision. Local source checks, hosted CI, provider execution, releases and owner acceptance remain separate evidence.
