# Project workflow

Markitect's purpose is canonical model-first development with delegated realization. Current source exposes structural preparation, a guarded generated-model view and explicit controlled Apply/Answer commands. Apply composes recursive Managers (parallel Execute in one shared working tree, fresh recursive Verify of one immutable candidate), container sessions, candidate workspaces, container checks and guarded publication. Source availability does not establish provider readiness or successful complete realization; the [delivery plan](implementation-plan.md) and recorded validation evidence retain those separate scopes.

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

## Set up a Host

`setup` prepares a Host directory so that `apply` needs no harness scripts. Build or pull the runtime image first, then run:

```text
go run ./cmd/markitect setup --repo REPOSITORY --host ABSOLUTE_HOST_DIR --image IMAGE_REFERENCE --credential ABSOLUTE_TOKEN_FILE --agent-model MODEL --check go=/usr/local/go/bin/go
```

It reads which Agent profiles and check profiles the project selects from the working draft, so every check profile needs one `--check PROFILE=EXECUTABLE` (the executable inside the image) and an unused one is refused. It asks the container engine for its version and volume-subpath support (Docker 26 or newer), resolves the image to its immutable ID and reads `/opt/markitect/runtime.env` from a created, never started container. It checks the token file's path, type and owner-only mode without opening it; the file must be a regular file outside the repository and its bytes are never read, copied or printed.

The Host directory must be outside the repository. Setup creates it private and writes `host.yaml` (every Agent profile as a Claude Code profile with the required `--agent-model` (there is no default model) and `--effort`, default `high`; every check profile as a container check on the image) and `run.json` (the [invocation configuration](#invocation-configuration), with `state/` in the same directory). Before writing, it decodes `run.json` as `apply` does and runs `host.yaml` through the settings and metadata preparation of `check --metadata --host-profile`. Optional `--store`, `--request`, `--decision` (repeatable), `--deadline` and `--engine` fill the remaining fields; `--model` and `--dependency` select the project as for `check`.

The JSON report lists the bound image ID, agentd and Claude Code versions and engine version, both files with their SHA-256 and whether each was `written` or `unchanged`, and the exact `apply` command to run next. Running setup again with the same inputs changes nothing. It refuses while an Apply holds the state directory. `productBuild` carries the Git revision only for a binary built with `go build` in a checkout; `go run` writes the plain version, so bind frozen runs with a built binary.

## Execute, verify, and apply

Use an existing committed repository, explicit dependency manifests, a Host profile document and an absolute invocation configuration path:

```text
go run ./cmd/markitect apply --repo REPOSITORY --host-profile ABSOLUTE_HOST_YAML --run-config ABSOLUTE_RUN_JSON
go run ./cmd/markitect extract --repo REPOSITORY --dependency DEPENDENCY_MODEL_YAML --host-profile ABSOLUTE_HOST_YAML --run-config ABSOLUTE_RUN_JSON
go run ./cmd/markitect answer QUESTION_ID --text "Owner decision" --run-config ABSOLUTE_RUN_JSON
```

`apply` accepts the same `--model` and repeated `--dependency` selections as preparation. It captures the actual committed source and complete settings, takes the state directory's exclusive lock, runs the permanent Manager team over one shared working tree until a candidate is verified or the budget ends, and publishes only a verified candidate. `--revision` and `--metadata` are preview options and are refused here. Missing explicit execution inputs leave Apply unavailable before source acquisition. Only one Apply runs per state directory; a second one refuses at the lock before it captures anything.

`extract` is the first, bounded implementation of the reverse direction (artifacts to model, [DEC-057](concepts/register.md#dec-057-three-fundamental-starting-situations)) for a repository that has artifacts but no model. It takes `--repo`, repeated `--dependency`, `--host-profile` and `--run-config` like `apply` and no `--model`: the Host prepares `.markitect/model.yaml` and `.markitect/model-lock.yaml` with no local Definitions (the model root holds only a placeholder file Git can track), the imported vocabulary, and `.markitect/execution.yaml` with budget, profiles, the artifact roots under `layout.roots` and checks without `requiredBy`. Extract refuses a model that already has Definitions (use `apply`) and a project without artifact roots. Phase one lets Root alone propose the Packages with an overview Definition each and the layout exceptions they need; phase two runs the proposed Manager tree to derive the model of every Package and register the artifacts. Existing artifacts are never changed or published. A Verified second phase publishes the changed `.markitect` model, registry shards and `execution.yaml` (layout exceptions and proposed checks) through the guarded Publisher, never a commit. The JSON report states model completion, the final Verify result, the realization deviations per Definition (artifact findings), the `proposedChecks`, open owner questions, per-phase rounds, turns and usage, and the publication. The [runtime contract](design/canonical-model/manager-apply-runtime.md#extract-first-bounded-implementation) owns the details and limits. Owner questions use the same channel and `answer` command as Apply.

`answer` is for a question Root asked while `apply` or `extract` is running. `apply` prints one line to stderr per question, naming the question ID and this command. `answer` takes the same `--run-config`, writes `answers/QUESTION_ID.txt` in its state directory atomically and reports the file it wrote. It refuses when no Apply holds the state directory, when the question was not asked, and when the question is already answered; an answer is final. Work that does not depend on the answer continues meanwhile. If the Apply ends with the question still open, put the decision into `ownerDecisions` of the next Apply's configuration. That holds only for a decision within the model: when the question shows a model defect, a contradiction in the model or a gap that changes behavior at the project boundary, correct the model and start a new Apply instead ([DEC-063](concepts/register.md#dec-063-the-model-is-authoritative-defects-escalate); the run-side behavior of that decision is not implemented yet).

### Invocation configuration

`--run-config` selects a closed UTF-8 JSON object with `formatVersion: 2`. Every field below is required, including the empty network string and the explicit harvest bounds. Two fields are optional. `deadline` is an absolute wall-clock time in RFC 3339 UTC form (`2026-10-19T18:30:00Z`); the run's own deadline is the earlier of its start plus the project's `executionTime` budget and this value, so the run ends gracefully, with its evidence and report written, before an external hard stop. `ownerDecisions` is a nonempty list of at most 64 of the owner's earlier decisions as plain texts of at most 16 KiB each; every Manager of the run sees them from its start ([DEC-053](concepts/register.md#dec-053-owner-decisions-bind-the-whole-run)). Unknown, duplicate, null, relative-path or invalid values refuse construction; files of the earlier version 1 (with `allocation`, `restoreLimits` and `engineLimits`) are refused. There are no environment defaults. This file supplies concrete invocation inputs; project policies and Agent/Check model, effort, image and resource settings remain in their existing execution and Host profile documents.

The following is an illustrative shape. Supply actual native absolute paths, immutable images, dedicated credentials and limits appropriate to the frozen project; these example assertions do not certify a runtime:

```json
{
  "formatVersion": 2,
  "repositoryIdentity": "library-project",
  "productBuild": "operator-identified-frozen-build",
  "ownerRequest": "Realize the captured model.",
  "stateDirectory": "/absolute/markitect-state",
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
  "harvest": {
    "maxEntries": 100000,
    "maxBytes": 67108864
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

`harvest` bounds every observation of the shared working tree: at most `maxEntries` entries and `maxBytes` bytes, within the Host's ceilings of 1048576 entries and 4 GiB; an observation that exceeds a bound is uncertain and ends the run incomplete. Interrupt and termination signals cancel the run the same way: every turn and container is stopped and the report is still written. Turn timeouts, rounds and budgets come from the project's execution settings, not from this file. `productBuild` is the operator's explicit assertion identifying the frozen build, not an automatically verified Git SHA.

`engineExecutable` must equal the already validated literal in every selected executable Agent and Check profile. The helper is its own explicit container capability for workspace transfer: immutable image ID, non-root numeric user/group, finite positive CPU/memory/process/tmp limits and labels are required. Its work directory is `/work`; invocation JSON cannot add mounts. Empty `network` explicitly leaves network selection to the engine; use `"none"` to disable networking. Host networking is refused. These helper fields do not replace Agent or Check profile resources.

### Dedicated credential references

`credentials` maps each profile's exact reference to one provider, mode and native absolute file path. Accepted pairs are `codex-app-server` with `api-key` or `dedicated-runtime-login`, and `claude-code` with `dedicated-runtime-token`. The operator must provide a dedicated runtime file; there is no desktop login discovery or environment fallback. Parsing configuration and preparing metadata do not read these files. Only the actual provider resolver reads its mapped regular file, with native path/link checks and a 1 MiB bound. Errors do not include credential bytes.

API keys and Claude tokens retain literal file bytes. A dedicated Codex runtime login retains the supplied subscription-login semantics; the adapter strips refresh material from the container copy. The Host never refreshes or rotates the source login. Keep credentials and external Host inputs outside the realization tree, and account for any repository-bound invocation file explicitly.

### Results and evidence

The JSON result includes the preparation status and diagnostics, the run (`status`, `reason`, every Verify round with its check results and Manager assessments, findings, questions, turns with usage and duration, and the total usage), the publication receipt with its read-back, `retainedWork` (the shared work volume kept when the run did not verify, for inspection) and `evidenceDirectory`. `status` is `passed` only when a verified candidate was published as applied or no change without an error. A run that ended incomplete (budget, open question, uncertain observation), a failed run and a failed publication stay distinct; `answer` reports the file it wrote.

Evidence stays on the Host under `STATE/runs/RUN/`: `events.jsonl` (one event per line, synced when recorded), `transcripts/TURN.jsonl` (each turn's provider stream with known secrets redacted) and `report.json` (the printed result, written at the end). Question files are in `STATE/questions/` and answers in `STATE/answers/`; question IDs restart for every Apply, so right after taking the state lock an Apply removes the question files (`.json`) and answer files (`.txt`) of earlier Applies, nothing else, and records its run in `STATE/current-run`. `markitect answer` writes only for a question file that carries that run, and only while an Apply holds the lock. Publication uses the same Git repository adapter that captured the source, checks the captured guard and protects model/control/instruction inputs. It does not create a Git commit or push. No MCP server is provided by these commands.

## Existing repositories

Author the intended canonical v1 model and explicit dependencies, then register exact realization files in `.markitect/registry/<package>/registry.yaml`. File granularity is the author's choice; a file may realize several Definitions but has one explicit owner. An exceptional ignore names one exact observed file with a reason. Missing entries, missing paths, duplicate ownership and overlaps with protected inputs fail. Gitignore is not coverage.

There is no automatic conversion, scaffold or Brownfield runtime. Earlier workflows remain reproducible from their [immutable source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).

## Record setup and interrupted work

Preparation and generated-document operations create no run, task, budget, candidate or execution journal. Preserve their acquisition diagnostics and generated-document completed-path reports. Apply and answer instead use the explicit state directory and its process lock. There is no journal and no restore: an interrupted Apply (deadline, signal or crash) ends incomplete, and a new Apply starts again from the committed repository, never from the earlier shared working tree. Evidence of every run stays under `STATE/runs/`. Preserve it and resolve the reported condition before starting again. No standalone recovery command is exposed here. The state directory must not be inside the repository or contain it.

## Acceptance and status

JSON reports name the operation, requested stage and actual statuses. Exit 0 means the requested stage passed; 1 is a known failure; 2 is invalid usage; 3 is incomplete observation, I/O/budget refusal or unavailable operation. `failed` and `not_completed` remain distinct. A committed model binds a specification revision; it does not authenticate a human decision. Local source checks, hosted CI, provider execution, releases and owner acceptance remain separate evidence.
