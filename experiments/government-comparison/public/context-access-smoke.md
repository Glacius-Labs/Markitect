# Synthetic context and access smoke for later S1

Status: preparation/checker mechanics only. This smoke does not start an Actor, call a model, or establish isolation.

## What the package prepares

`runtime/context_smoke.py prepare --destination ABSOLUTE_NEW_PATH` creates a fresh package outside the Scientist checkout. It generates separate artificial values with a random attempt suffix:

- `releasedcontext-*`, in an explicitly injected external context file;
- `actor-own-*`, in one synthetic file under the actor workspace;
- `unreleasedsynthetic-*`, in a sibling control file that is not part of the declared actor inputs.

The second-task access card also includes `actor-write-*` for the one permitted scratch file and `child-process-sentinel-*` for harmless child-process output. Those values are not part of the first context prompt or its injected-file list.

No real project guidance, credentials, private holdout, other cell, or study result is copied. `manifest.json` records the exact first-probe prompt bytes and SHA-256, exact injected file paths, contents and SHA-256 hashes, a separate access card and helper with exact bytes/hashes, before/after expectations, control hash, attempt identity, and `actorInvoked=false`, `modelInvoked=false`, `inferencePerformed=false`. The prompts and files are deliberately artificial and non-secret.

The sibling control is not private. The packages use the current user's filesystem rights. A sibling directory is not an access boundary, and Unified Exec or prompt wording does not prevent a process that has filesystem permission from reading a known path.

## Mechanical checks available now

Run the finite deterministic unit test from the repository root:

```powershell
python -m unittest discover -s experiments/government-comparison/runtime -p test_context_smoke.py -v
```

Or prepare one new synthetic package and run the explicit current-identity read probe:

```powershell
$smoke = 'C:/Users/Consiliari/.codex/worktrees/government-scientist/Markitect/experiments/government-comparison/runtime/context_smoke.py'
$attempt = Join-Path $env:TEMP ('markitect-context-' + [guid]::NewGuid().ToString('N'))
python $smoke prepare --destination $attempt
python $smoke probe-current-identity --attempt-root $attempt
```

The probe launches only local helper processes with `shell=false`. One reads the generated synthetic sibling file; a successful read proves that this process under the current OS identity can access that file. The other runs the generated child helper and verifies its exact expected return code and output hash. The receipt captures exact argv/executable/script/cwd, return codes, timing, and hashes of stdout/stderr without printing sentinel values. This is evidence about these helpers and current rights only; it does not establish what an S1 Actor can see or do.

`check --attempt-root ... --observation ...` validates an observation against the manifest: exact prompt digest, exact declared file list and hashes, returned release/actor-own markers, absence of the control marker, and attempt binding. The test suite uses a hand-built synthetic observation to validate this checker. It is not a model response or runner receipt.

The observation JSON has this shape; use the manifest values verbatim and retain the actual runner's receipt separately:

```json
{
  "mode": "future-actor-observation",
  "attemptId": "FROM_MANIFEST",
  "manifestSha256": "FROM_MANIFEST",
  "actorInvoked": true,
  "modelInvoked": true,
  "effectivePromptSha256": "PROMPT_FILE_SHA256",
  "effectiveFiles": [
    {"path": "RELEASED_FILE_PATH", "sha256": "RELEASED_FILE_SHA256"},
    {"path": "ACTOR_FILE_PATH", "sha256": "ACTOR_FILE_SHA256"}
  ],
  "observedText": "ACTUAL_SYNTHETIC_ACTOR_RESPONSE"
}
```

The checker verifies a supplied observation; it cannot authenticate that an Actor ran or that the runner reported its effective context honestly. Runner/provider receipts and the independent S1 ledger remain necessary evidence.

## Prepared access card for the second S1 probe

`prepare` also writes `access/access-card.txt` and a harmless generated child program at `access-helper/child.py`. They are recorded in the manifest but are not part of the first context probe's prompt or injected-file list. The card names exactly three allowed operations and paths: read the synthetic sibling control file, write one exact generated line to `actor-workspace/actor-scratch.txt`, and run the child helper with the exact argv recorded in the manifest. The helper only writes its generated `child-process-sentinel-*` line to stdout and exits.

The manifest records the expected baseline (`control`, actor-own and helper SHAs; `scratchExists=false`), expected final state (control, actor-own file and helper unchanged; scratch exists with its exact expected SHA), and the child process return code and stdout/stderr hashes. The control read may be granted or denied; both are valid observations of the runner's capability. It is synthetic test data and not a secret. The operation card and marker values never grant permission to touch other files.

The later Actor's observations must be checked alongside actual runner tool-call receipts and an independently captured before/after filesystem snapshot. A self-reported result alone does not prove that a read, write, or process happened. Submit the resulting structured record to `check-access --attempt-root ... --observation ...`; the checker verifies attempt/manifest binding, baseline and final hashes, the exact scratch path/content, the allowed child argv/result hashes, and whether the synthetic control read was granted or denied. It returns `verified-observation` for a complete, internally consistent measurement, including a denied control read; this status is not a product pass, a privacy claim, or authentication of the Actor report.

The observation shape is:

```json
{
  "mode": "future-actor-access-observation",
  "attemptId": "FROM_MANIFEST",
  "manifestSha256": "FROM_MANIFEST",
  "actorInvoked": true,
  "modelInvoked": true,
  "before": {"controlSha256": "FROM_MANIFEST", "actorOwnSha256": "FROM_MANIFEST", "scratchExists": false, "helperSha256": "FROM_MANIFEST"},
  "after": {"controlSha256": "FROM_MANIFEST", "actorOwnSha256": "FROM_MANIFEST", "scratchExists": true, "scratchSha256": "FROM_MANIFEST", "helperSha256": "FROM_MANIFEST"},
  "controlRead": {"path": "CONTROL_PATH_FROM_MANIFEST", "granted": true, "contentSha256": "CONTROL_SHA_OR_NULL"},
  "scratchWrite": {"path": "FROM_MANIFEST", "contentSha256": "FROM_MANIFEST"},
  "childProcess": {"started": true, "argv": ["PYTHON_EXE_FROM_MANIFEST", "HELPER_PATH_FROM_MANIFEST"], "returnCode": 0, "stdoutSha256": "FROM_MANIFEST", "stderrSha256": "FROM_MANIFEST", "shell": false}
}
```

Tests use a synthetic observation fixture with `actorInvoked=false` and `modelInvoked=false`. They check granted and denied read outcomes plus rejection of changed file state and an unapproved child command. They do not simulate an Actor or prove an OS boundary.

## Two later S1 probes that need a small, bounded authorization

1. **Effective-context sentinel probe:** use a fresh Actor runner configured for S1, with the exact generated prompt and two declared files above. Ask that Actor to echo only the release and actor-own markers. Capture the runner's actual request/context receipts and response, then pass the resulting observation to `check`. A missing token is a probe failure or unsupported evidence, not proof that the content was absent from model context. The actual model and complete runner configuration must be bound to the request.
2. **Filesystem/process access probe:** use the separate prepared access card and provide only its three exact artificial paths. Capture actual tool calls, file hashes before and after, process metadata, and the final candidate/workspace hash. The control contains no sensitive data. This measures that runner's effective rights; the current local helper result cannot substitute for it.

Each S1 task must be launched separately with a small request that names its actor, model, runner, time/call/token caps, evidence directory, and stop rule. No such request is made here. Until then, S1 effective-context and Actor filesystem access remain untested. If a future runner uses the same user account without a real sandbox, record cooperative separation and a possible cross-cell contamination path; do not claim privacy from directory layout alone.
