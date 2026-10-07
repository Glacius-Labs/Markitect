# Classic Commerce: complete canonical alpha example

This source example runs with the published Markitect v0.14.1 native CLI. It contains its own canonical model, all selected .NET and Markdown ProjectionPolicies, four pinned Modules, explicit controller runtime template, deterministic protocol test double, and independently authored .NET build/behavior check. No internal test helper or private readiness packet is required. The controller remains experimental alpha; successful compilation and these finite cases do not establish autonomous agent quality or human acceptance.

The canonical UseCase requires positive integer quantity, non-negative decimal unit price, their decimal product, and ArgumentOutOfRangeException for invalid inputs. A separate console probe compiles the materialized library and checks normal/fractional/zero-price totals, invalid inputs and the explicit application boundary. Overflow outside the bounded cases is unspecified. The .NET policy owns the exact public signature. The supplied actor implements known fixture bytes, not discovery or modeling.

## Prerequisites

- Windows amd64 or Linux amd64, Git, Python 3.10+ available as `python` and an absolute path to the verified Markitect v0.14.1 native binary. Follow the [public installation instructions](../../README.md#install-markitect); `version` must identify 0.14.1. This example does not install or change a global CLI.
- An existing .NET SDK **8.0.418** and compatible .NET 8 runtime. `global.json` disables roll-forward for reproducibility. This example targets `net8.0` and uses no external NuGet packages; `NuGet.Config` clears package feeds. Missing SDK/reference packs are a prerequisite gap, never permission to install or retarget silently.
- Fresh disposable Git repositories and external directories for reports, ledger, private actor logs and build artifacts. All commands run with the local caller's authority; this is not an OS sandbox. The native product does not fingerprint the PATH-resolved compiler/Python as authenticated tools; this example records their actual versions and raw outputs.

.NET is required only for this opt-in example. It is not a new prerequisite for the general Markitect CLI or Go test suite. The model/config/checks and runtime digest must be frozen anew if the SDK, fixed inputs or candidate changes.

## One bounded positive and negative smoke

Run from a clean committed Markitect source checkout containing this example. The binary remains the published 0.14.1 asset; the example source commit is the current checkout's separate identity.

```powershell
$sourceRoot = (Get-Location).Path
$sourceRevision = (git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Cannot bind example source' }
$binary = 'REQUIRED_ABSOLUTE_PATH_TO_VERIFIED_markitect-v0.14.1-windows-amd64.exe'
$output = Join-Path $env:TEMP ('classic-commerce-' + [guid]::NewGuid().ToString('N'))
python examples/classic-commerce/tools/smoke.py --binary $binary --source-root $sourceRoot --source-revision $sourceRevision --output $output --approve-known-fixture
if ($LASTEXITCODE -ne 0) { throw "Example smoke incomplete; preserve $output" }
Get-Content -Raw (Join-Path $output 'receipt.json')
```

On Linux use the verified Linux amd64 executable and an absent output beneath `${TMPDIR:-/tmp}`:

```bash
set -eu
source_root="$PWD"
source_revision="$(git rev-parse HEAD)"
binary='/REQUIRED/ABSOLUTE/PATH/TO/markitect-v0.14.1-linux-amd64'
output="$(mktemp -d)/run"
python examples/classic-commerce/tools/smoke.py --binary "$binary" --source-root "$source_root" --source-revision "$source_revision" --output "$output" --approve-known-fixture
cat "$output/receipt.json"
```

`--approve-known-fixture` explicitly authorizes this harness to review its known candidate bytes and invoke Apply in disposable targets. It is not a general approval or a claim of human code review. The driver calls the native CLI separately for inspection, proposal, Execute, guarded Apply, fresh Verify using the saved Apply-result JSON, and Audit. It preserves exact native argv/cwd/exit/stdout/stderr, source and evidence revisions, all fixed-check/build/probe results and runtime/actor hashes. Existing output is refused; attempts are not automatically retried or deleted.

The positive candidate must actually compile and pass the independent behavior probe for both assurance scopes, with fresh Verify passed and Audit complete. The bad-business candidate still compiles but computes addition instead of multiplication; the independent probe must fail, native Verify must fail and Audit must remain incomplete. Apply can materialize either candidate as **materialized-unverified**. The negative proves refused verification/closure, not refusal to write, a model judgment or automatic rollback. A syntax/transport refusal cannot substitute for the compilable business-defect control.

## Native CLI walkthrough

The following PowerShell 7.4+ steps expose the same positive lifecycle. Run from the source checkout and set `$binary` to the verified native binary. Clone only this committed example into a new disposable repository; retain the `examples/classic-commerce/` path because the config uses explicit repository-relative inputs.

```powershell
$sourceRoot = (Get-Location).Path
$sourceRevision = (git rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Source binding failed' }
$out = Join-Path $env:TEMP ('classic-commerce-manual-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $out -ErrorAction Stop | Out-Null
$repo = Join-Path $out 'repo'
New-Item -ItemType Directory -Path $repo -ErrorAction Stop | Out-Null
git archive --format=zip --output="$out/example.zip" $sourceRevision examples/classic-commerce
if ($LASTEXITCODE -ne 0) { throw 'Example archive failed' }
Expand-Archive -LiteralPath "$out/example.zip" -DestinationPath $repo
git -C $repo init -b codex/classic-commerce-manual
if ($LASTEXITCODE -ne 0) { throw 'Fixture init failed' }
git -C $repo config core.autocrlf false
if ($LASTEXITCODE -ne 0) { throw 'Git configuration failed' }
git -C $repo add -- examples/classic-commerce
if ($LASTEXITCODE -ne 0) { throw 'Fixture staging failed' }
git -C $repo -c user.name='Markitect Example' -c user.email='example@example.invalid' -c commit.gpgsign=false commit -m 'Freeze public Classic example'
if ($LASTEXITCODE -ne 0) { throw 'Fixture commit failed' }
$source = (git -C $repo rev-parse HEAD).Trim()
if ($LASTEXITCODE -ne 0) { throw 'Fixture source binding failed' }
$python = (python -c 'import sys; print(sys.executable)').Trim()
if ($LASTEXITCODE -ne 0) { throw 'Python resolution failed' }
$runner = Join-Path $repo 'examples/classic-commerce/tools/protocol_runner.py'
$runnerHash = (Get-FileHash -Algorithm SHA256 -LiteralPath $runner).Hash.ToLowerInvariant()
$runtime = Join-Path $out 'runtime.json'
$template = Get-Content -Raw (Join-Path $repo 'examples/classic-commerce/runtime.template.json') | ConvertFrom-Json
$template.recordStore = Join-Path $out 'ledger'
$template.privateLogs = Join-Path $out 'private-logs'
foreach ($binding in @($template.executor, $template.verifier)) {
  $binding.command = $python
  $binding.args = @($runner, '--mode', 'positive')
  $binding.runtimeFiles[0].path = $runner
  $binding.runtimeFiles[0].digest = 'sha256:' + $runnerHash
}
$template | ConvertTo-Json -Depth 30 | Set-Content -LiteralPath $runtime -Encoding utf8NoBOM
$env:MARKITECT_CLASSIC_COMMERCE_EVIDENCE_DIR = Join-Path $out 'fixed-checks'
Set-Location $repo
$config = 'examples/classic-commerce/canonical.yaml'
```

Inspection commands emit YAML; controller actions emit JSON. Save both exact streams, including nonzero exits:

```powershell
function Invoke-Classic([string]$Name, [string[]]$Arguments, [bool]$Json = $true) {
  $suffix = if ($Json) { 'json' } else { 'yaml' }
  $stdout = Join-Path $out "$Name.stdout.$suffix"
  $stderr = Join-Path $out "$Name.stderr.txt"
  & $binary @Arguments 1> $stdout 2> $stderr
  $code = $LASTEXITCODE
  if ($code -ne 0) { throw "$Name exit $code; preserve $stdout and $stderr" }
  if ($Json) { return Get-Content -Raw $stdout | ConvertFrom-Json }
  return Get-Content -Raw $stdout
}
& $binary version
if ($LASTEXITCODE -ne 0) { throw 'Native binary failed' }
$model = Invoke-Classic 'model' @('canonical','--repo',$repo,'--config',$config,'--action','model','--revision',$source) $false
$plan = Invoke-Classic 'reconcile' @('canonical','--repo',$repo,'--config',$config,'--action','reconcile-plan','--base',$source,'--revision',$source) $false
$common = @('canonical','--repo',$repo,'--config',$config,'--runtime',$runtime)
$proposal = Invoke-Classic 'proposal' ($common + @('--action','controller-propose','--base',$source,'--revision',$source))
if ($proposal.status -ne 'planned') { throw 'Proposal is not planned' }
$execute = Invoke-Classic 'execute' ($common + @('--action','controller-execute','--base',$source,'--revision',$source))
if ($execute.status -ne 'planned' -or -not $execute.digest) { throw 'Execute report is not a digest-bound plan' }
# Review execute.stdout.json, including candidate paths, modes, contents and digest.
# This is one manual approval pattern; follow the existing project review practice.
$approved = Read-Host 'Enter the exact reviewed Execute digest to authorize materialization, or stop'
if ($approved -cne [string]$execute.digest) { throw 'Review digest differs' }
$apply = Invoke-Classic 'apply' ($common + @('--action','controller-apply','--base',$source,'--revision',$source,'--plan',"$out/execute.stdout.json",'--expect',$approved,'--write'))
if ($apply.status -ne 'materialized-unverified') { throw 'Apply did not return unverified materialization' }
$verify = Invoke-Classic 'verify' ($common + @('--action','controller-verify','--apply-result',"$out/apply.stdout.json",'--write'))
if ($verify.status -ne 'passed') { throw 'Fresh Verify did not pass' }
$audit = Invoke-Classic 'audit' ($common + @('--action','controller-audit','--base',$source,'--revision',$source))
if ($audit.status -ne 'complete') { throw 'Audit did not account for declared scope' }
```

`--apply-result` selects the exact source/evidence revisions in the saved JSON; it does not approve or verify it by itself. Verify reacquires evidence and runs fixed checks plus separate synthetic Verifier invocations for child and parent. Audit is read-only and invokes no actor/check. Failed Verify exits 1 and incomplete Audit exits 2; preserve both reports. For the negative control use the automated fresh trial rather than altering or reusing a passed ledger. These fixed checks execute project-owned C#/MSBuild with local authority and no network-package requirement, not in a security sandbox.

## Evidence and limits

Each fixed-check invocation creates a new `classic-commerce-check-*` directory beneath `MARKITECT_CLASSIC_COMMERCE_EVIDENCE_DIR` (or system Temp), retaining exact restore/build/probe commands, raw streams, candidate hashes, actual SDK selection and `result.json`. `bin`/`obj` and CLI home/package caches are external, so the source and generated target roots stay clean apart from reviewed projected files. Preserve failed and timed-out runs; do not rescore them or silently substitute SDKs/providers.

The independent authored console probe is separate from candidate code and tests known obligations. The protocol Verifier is a deterministic test double that echoes required coverage; it does not supply independent semantic judgment. Build plus a few behavior cases prove only that this particular materialization met the declared checks. No real provider, continuous runtime, study adapter, resume facility, economic benefit or owner acceptance is established. Other Markitect users do not need .NET unless they select checks that require it.

The earlier [canonical projection binding example](../canonical-projection/binding-notes.md) is a structural/binding slice with incomplete Markdown policy selection; its initial escalation remains deliberate rather than being weakened. Use this complete example for a fresh end-to-end operator exercise.

The [dated validation report](../../docs/validation/classic-commerce-entrypoint.md) records the first real Windows compiler smoke, its compiling negative control, exact source/runtime identities and pending full integration gates. Linux commands above are rerun instructions, not a claim of Linux validation for this checkpoint.
