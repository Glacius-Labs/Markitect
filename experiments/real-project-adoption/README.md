# Real-project adoption replay

This experiment asks whether a small, adopter-owned Markitect model can compile useful architecture context and policy for a task in a real public codebase. The input is a selected, byte-preserving part of the MIT-licensed [`modular-monolith-with-ddd`](https://github.com/kgrzybek/modular-monolith-with-ddd) repository, pinned to upstream commit `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8`. The extracted baseline contains 646 upstream files (711,461 bytes); its manifest records each copied Windows working-tree path, byte length, and SHA-256. Git normalizes line endings when freezing the subset; the [fixed-source audit](evidence/fixed-source-audit.json) proves both arms are byte-identical and differ from upstream Git only by CRLF/LF normalization. See [source selection](source-selection.md) and [provenance](provenance.md) for exclusions and build limits.

The already frozen unassisted agent run is a separate, uncontrolled one-run comparison documented in [falsification notes](falsification-notes.md). It took about 7 minutes 19 seconds in that run; calls and tokens were not instrumented. This replay does not rerun that agent or turn the observation into a productivity claim. The assisted task and architecture evolution runs are complete and documented in the [pilot report](../../docs/validation/real-project-adoption-pilot.md). Human review and adoption of the discovery candidate remain pending.

## Replay the technical setup

Use Windows PowerShell 7, Git, a clean checkout of the upstream repository at the pinned commit, and the public Markitect v0.12.0 Windows executable. No Python package is required. The replay refuses to reuse existing baseline, adopter, package-source, or evidence paths, and it does not delete files. Choose a new run directory each time.

```powershell
$repo = (Get-Location).Path
$experiment = Join-Path $repo 'experiments/real-project-adoption'
$runRoot = Join-Path $repo '.artifacts/adoption/replay-2026-10-03-a'
$null = New-Item -ItemType Directory -Path $runRoot

& (Join-Path $experiment 'replay.ps1') `
  -UpstreamRoot 'C:/src/modular-monolith-with-ddd' `
  -BaselineDestination (Join-Path $runRoot 'baseline') `
  -AdopterDestination (Join-Path $runRoot 'adopter') `
  -PackageSourceRoot (Join-Path $runRoot 'package-sources') `
  -MarkitectExe 'C:/tools/markitect-v0.12.0-windows-amd64.exe'
```

Before running, check that the upstream checkout is clean and at the required commit. `replay.ps1` calls [`prepare.ps1`](prepare.ps1), which owns source selection, file-copy hash checks, and pristine baseline commit creation. Replay clones that prepared baseline into `codex/pilot-adopter`, copies the model's `resources/` and `agent-guidance/`, copies the project-owned MSBuild reference check, and copies only the Copy Me task prompt and ContextRun. It builds package versions 1.0.0 and 2.0.0 in separate Git repositories on `codex/adoption-constitution-v1` and `codex/adoption-constitution-v2`, commits each with a fixed identity and timestamp, then packs each exact commit with Markitect.

The generated `*.replay-evidence.json` sidecar records the upstream and baseline identities, tool version and executable hash, package branch/commit identities, package content hashes, archive SHA-256 values, and the copied adopter inputs. The adopter Project initially pins package `mymeetings-architecture` 1.0.0 to its generated archive hash. The script preserves YAML formatting while substituting that digest and checks that the configured `scripts/check-module-project-references.ps1` was copied. It ends after `format --write`; it does not run an agent, make an adoption decision, or claim acceptance.

The discovery candidate, evidence dossier, and decision template are deliberately not copied into the adopter. The task prompt and `context-run.yaml` are inputs for a possible future context run; they do not represent a completed discovery or authorize a candidate. To run context, first inspect the generated files, follow the explicit projection-reconciliation steps below, and check/model the resulting setup. Add `.artifacts/` to the pilot `.gitignore`, review and commit the canonical setup and generated outputs on the named pilot branch. Then bind the ContextRun to that commit:

```powershell
$tool = 'C:/tools/markitect-v0.12.0-windows-amd64.exe'
$adopter = Join-Path $runRoot 'adopter'
$setupCommit = (git -C $adopter rev-parse HEAD).Trim()
& $tool check --repo $adopter
& $tool model --repo $adopter
& $tool context --repo $adopter --revision $setupCommit --run context-run.yaml
```

The `context` result is fixed-snapshot technical evidence. Record its snapshot digest, run manifest hash, selection hash, context digest, tool identity, and resource/input set in a separate evidence record. Do not fill the discovery decision as accepted from this output. The existing candidate remains a proposal until a human reviewer records a decision and the exact candidate/evidence hashes.

## Reconcile generated projections

The assembly replay does not apply generated files. Keep reconciliation's write boundary visible: observe, create and inspect a plan, then apply only that plan with `--write`, and verify the result.

```powershell
$tool = 'C:/tools/markitect-v0.12.0-windows-amd64.exe'
$adopter = Join-Path $runRoot 'adopter'
$reconcile = Join-Path $adopter '.artifacts/markitect/reconcile'
$null = New-Item -ItemType Directory -Force -Path $reconcile

& $tool reconcile --repo $adopter --action observe --adapter markitect-render
& $tool reconcile --repo $adopter --action plan --adapter markitect-render |
  Set-Content -LiteralPath (Join-Path $reconcile 'plan.yaml') -Encoding utf8
# Inspect plan.yaml and resolve any conflicts before the explicit write step.
& $tool reconcile --repo $adopter --action apply --adapter markitect-render `
  --plan (Join-Path $reconcile 'plan.yaml') --write
& $tool reconcile --repo $adopter --action verify --adapter markitect-render `
  --plan (Join-Path $reconcile 'plan.yaml')
```

For an optional idempotence check, make a second plan after verification, inspect that it contains no operations, apply that empty plan with `--write`, and verify it. Preserve both plans and command output with the replay evidence. This shows convergence for the recorded projection snapshot; it does not establish semantic correctness or human acceptance. Reconciliation never authorizes discovery adoption.

## Evidence limits

This replay reproduces the technical setup from pinned public source and a pinned public CLI release. It does not recreate the uncontrolled baseline run, measure agent time or tokens, demonstrate productivity or savings, prove production acceptance, or establish that an active adopter would maintain this model. The public reference app is an MIT-licensed subset, not a production customer system. The selected Meetings Application project previously compiled with .NET SDK 8.0.418, but that does not prove the full repository builds or runs. Keep technical candidate checks, assisted agent observations, human candidate review, fachliche acceptance, and any production release as separate evidence states.

## Replay the exact frozen states offline

[evidence/adopter-snapshots.bundle](evidence/adopter-snapshots.bundle) retains the exact local commits, public source, original MIT LICENSE, both package archives, and the completed task/evolution states. [snapshots.json](evidence/snapshots.json) records the bundle digest and revision identities. It contains no build caches or Markitect executable. Obtain and verify the public v0.12.0 binary separately.

```powershell
git clone experiments/real-project-adoption/evidence/adopter-snapshots.bundle .artifacts/adoption/frozen-replay
$tool = 'C:/tools/markitect-v0.12.0-windows-amd64.exe'
$adopter = '.artifacts/adoption/frozen-replay'
& $tool check --repo $adopter --revision fd94a689aab857704c3a4a45ac2abe0fc0e3185c
& $tool context --repo $adopter --revision fd94a689aab857704c3a4a45ac2abe0fc0e3185c --namespace engineering --kind Skill --name architecture-review
& $tool check --repo $adopter --revision a0f89e7c0ce43a1be556c17c41cd19f8e7f32d7f # Expected two failed policies.
& $tool verify --repo $adopter --revision c507172b24da9005904422c03cc3d66a2b5efcec # Requires the recorded .NET SDK/environment.
& $tool impact --repo $adopter --base c507172b24da9005904422c03cc3d66a2b5efcec --revision fdc32ac004d17e3b4e91443b7c59fe035ce0d7b3
```

The bundle also retains the baseline task outcome and package source commits. Branches have the `codex/` prefix. Fixed read-only commands can evaluate any preserved revision; select an appropriate named feature branch before mutations. The original timestamped exception remains an experimental record, not human production acceptance. [command-evidence.zip](evidence/command-evidence.zip) retains the captured technical commands once; its [index](evidence/command-evidence-index.json) gives per-file hashes. The local `.gitattributes` preserves captured evidence and the hash-bound candidate/ledger bytes across checkouts. Replays reproduce technical states, not past agent cognition, timings, missing token telemetry or human adoption.
