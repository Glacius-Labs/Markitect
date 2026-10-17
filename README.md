# Markitect

**Don't chase the code. Let the code chase you.**

You shape the idea. Markitect organizes the work that brings your project into alignment.

[![CI](https://github.com/Glacius-Labs/Markitect/actions/workflows/ci.yaml/badge.svg)](https://github.com/Glacius-Labs/Markitect/actions/workflows/ci.yaml)
[![License: Apache-2.0](https://img.shields.io/badge/License-Apache--2.0-blue.svg)](LICENSE)
[![Status: In development](https://img.shields.io/badge/Status-In%20development-orange.svg)](docs/implementation-plan.md)

[Why Markitect](#why-markitect) · [Working with a model](#working-with-a-model) · [An example](#let-the-project-do-the-catching-up) · [Development status](#development-status) · [Explore the source](#run-the-current-source)

You have a picture of what your project should be: its behavior, architecture, concepts and rules. As that picture develops, bringing the project along can become a job of its own.

Markitect is model-first development with delegated AI realization. Make your intended world explicit in a canonical model. Markitect is being built to let the project take on the work of aligning itself with that intent: working out the changes, carrying them through and checking the result across code, tests, configuration and documentation.

The goal is a clean, uniform and understandable repository, without drift, forgotten rules or contradictions. Your attention stays with the idea. The project's realization follows it, with enough trust that you no longer need routine implementation review.

## Why Markitect

The difficult question after an AI implementation is:

> Does what the AI built actually match what I intended?

A feature can work while an existing rule is overlooked, another area breaks or the documentation describes something different. You then have to find the gap, explain what you meant, prompt again and check the repair. Some omissions may never be noticed. Keeping the project close to the picture in your head becomes a continuing responsibility.

Markitect aims to reverse who carries that responsibility. **You develop the intended world; the project works toward it.** Its organization takes on the translation, implementation, coordination and verification needed to bring the actual project into agreement.

That treats **human attention as the scarce resource**. Focused agents can spend effort on implementation and checking while people concentrate on the intended system and decisions that need their judgment.

The method draws on familiar practices from human teams: clear responsibility, delegation, escalation, the four-eyes principle and integration of independently examined work. Their effectiveness for AI realization must be demonstrated in use.

## What Markitect aims to deliver

| Goal | What it means for your project |
| --- | --- |
| **Model first** | One authoritative account of behavior, architecture and rules remains effective through every change. |
| **A coherent repository** | Code, tests, configuration and documentation agree with that model and with one another. |
| **Continuing obligations** | A new feature preserves the rules and behavior that still apply. |
| **Trustworthy realization** | Systematic implementation, independent review and verification support confidence in the result. |
| **Your attention on intent** | Routine human implementation review becomes unnecessary; meaningful decisions remain yours. |

These are the product's target outcomes. The [development status](#development-status) below describes what you can run today.

## Working with a model

1. **Shape the idea.** Define the project's behavior, concepts, architecture and rules in the canonical model. AI can help you explore and author that intent.
2. **Let the project work toward it.** Markitect organizes the changes, independent review and integration needed to align the realization with the model.
3. **Keep developing your vision.** Refine the model as your understanding evolves. The next realization works toward that new intent while preserving the obligations that still apply. Deviations from unchanged intent can be repaired against the same model.

The idea becomes the shared point of reference: you develop it, and the project brings its realizations into alignment. Code, tests and documentation follow that intent without asking you to rediscover every consequence of every change.

## Let the project do the catching up.

Imagine a library where a member may borrow five books. The rule changes: members may now borrow three.

You express the new rule in the model. Working out where that change belongs and carrying it through is the project's realization work. The intended outcome is coordinated:

- The implementation enforces the new limit.
- Tests check the new boundary against the requirement.
- The usage guide explains the changed behavior.
- Existing rules, including returning books and handling suspended members, still hold.

The first complete demonstration applies the original library model, changes that one policy value and applies the revised model against the actual first result. It is the workflow we are building toward, with [independent expectations](docs/design/canonical-model/library-example/evaluation-contract.md) assessing the result.

## Where trust should come from

Markitect combines **AI judgment with deterministic structure**. Explicit model relationships, accountable responsibilities and required process steps organize work and make it inspectable. Agents interpret and realize intent; independent assessment examines their work and how it fits together.

The organization matters because agents can overlook things. Implementation, tests and review can share a mistaken assumption. Markitect's method aims to expose those gaps through distinct responsibilities, relevant context, independent judgment and integration checks.

Model calls, elapsed time and human attention are different costs. Lower-cost models can make focused collaboration practical; making development cheaper is not the product goal. What matters is the quality of the result and how much human attention it needs. Parallel execution is a later opportunity to reduce elapsed time.

For the full explanation, read [Markitect in brief](docs/vision.md#markitect-in-brief) and [origin and rationale](docs/concepts/origin-and-rationale.md).

## What we want to find out

The intended comparison is straightforward: give an agent ordinary documentation containing the same project intent as the canonical model, with the instruction to read it and implement it.

Does Markitect's structured realization preserve that intent more reliably through a sequence of changes? Does it leave fewer omissions, contradictions and manual corrections? Can people trust the result without routine implementation review?

The conventional approach is allowed to succeed. [Measurement](docs/measurement.md) owns evaluation procedure; an early successful example does not establish a general advantage.

## Development status

**Markitect is under active development. Full Apply is not available yet.**

Current source supports model compilation, repository file accounting, context preparation and a guarded readable model view. Session, workspace, check and publication foundations are integrated, while the core realization workflow and its public Apply entrypoint are being completed.

Real runtime probes and small single-agent library baselines have passed. They establish their [recorded scope](docs/validation/runtime-capabilities-20261017.md); they do not demonstrate the complete Manager organization or the two-Apply workflow.

The next complete milestone is the [two-Apply library demonstration](docs/design/canonical-model/manager-apply-playground.md). Follow the [delivery plan](docs/implementation-plan.md) for scope and dependencies.

## Run the current source

Use Git and the Go version in `go.mod`, then run from this checkout:

```text
go run ./cmd/markitect help
go run ./cmd/markitect check --repo .
go run ./cmd/markitect context product/model --repo .
go run ./cmd/markitect docs --repo .
```

These commands inspect and prepare the model. They do not start a realization. The [project workflow](docs/project-workflow.md) explains current commands, and the [self-model](.markitect/README.md) provides a complete authoring example.

## Contribute

Start with [CONTRIBUTING](CONTRIBUTING.md), the [delivery plan](docs/implementation-plan.md) and the [documentation map](docs/README.md). Concrete feedback about the model, workflow and missing obligations is useful alongside implementation contributions.

## Earlier published release

Published v0.14.1 belongs to the earlier product generation. Its commands and model formats differ from the current source. Use its versioned guides when trying that release.

<details>
<summary>Install and try the published v0.14.1 release</summary>

<!-- markitect-release:install:start -->

## Install Markitect

The current [v0.14.1 release](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.14.1) provides Windows and Linux amd64 binaries. The commands below download a fixed version from the public release, check its published SHA-256 digest, and install it for your user. No Go installation or GitHub login is needed. Git is needed for Markitect commands that read Git revisions.

**Windows (PowerShell):**

```powershell
if (-not [Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([Runtime.InteropServices.OSPlatform]::Windows) -or [Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne [Runtime.InteropServices.Architecture]::X64) { throw 'Only Windows amd64 is released.' }
$tag = 'v0.14.1'
$asset = "markitect-$tag-windows-amd64.exe"
$sha256 = '2cad55efad64f15d7f57638bea78312918fbf7d181c730f188b9921504da71c4'
$download = Join-Path $env:TEMP ("markitect-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $download -ErrorAction Stop | Out-Null
$source = Join-Path $download $asset
Invoke-WebRequest "https://github.com/Glacius-Labs/Markitect/releases/download/$tag/$asset" -OutFile $source -UseBasicParsing -ErrorAction Stop
if ((Get-FileHash -LiteralPath $source -Algorithm SHA256).Hash -ne $sha256) { throw 'Release binary SHA-256 mismatch.' }
$bin = Join-Path $env:LOCALAPPDATA 'Programs\Markitect'
New-Item -ItemType Directory -Path $bin -Force -ErrorAction Stop | Out-Null
Copy-Item -LiteralPath $source -Destination (Join-Path $bin 'markitect.exe') -Force -ErrorAction Stop
$userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
if ($bin -notin ($userPath -split ';')) {
    [Environment]::SetEnvironmentVariable('Path', ((@($userPath, $bin) | Where-Object { $_ }) -join ';'), 'User')
}
$env:Path = "$bin;$env:Path"
$installed = & (Join-Path $bin 'markitect.exe') version
if ($LASTEXITCODE -ne 0 -or $installed -ne 'Markitect 0.14.1 (windows/amd64)') { throw 'Installed CLI version check failed.' }
$installed
Remove-Item -LiteralPath $source, $download
```

**Linux (bash):**

```bash
(
  set -e
  [ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] || { echo 'Only Linux amd64 is released.' >&2; exit 1; }
  tag=v0.14.1
  asset="markitect-$tag-linux-amd64"
  sha256=f515f8fb37335ff5bd36f77cbbd0af227c3558d0d5aa8ba80503bae40314fbdc
  download="$(mktemp -d)"
  trap 'rm -rf "$download"' EXIT
  curl -fLsS "https://github.com/Glacius-Labs/Markitect/releases/download/$tag/$asset" -o "$download/$asset"
  printf '%s  %s\n' "$sha256" "$download/$asset" | sha256sum -c -
  install -D -m 0755 "$download/$asset" "$HOME/.local/bin/markitect"
  "$HOME/.local/bin/markitect" version
)
export PATH="$HOME/.local/bin:$PATH"
```

Add `~/.local/bin` to your shell startup file if it is not already on `PATH`. For developers who already use Go 1.27.1 or later, `go install github.com/Glacius-Labs/Markitect/cmd/markitect@v0.14.1` is a shorter source-build option. macOS and arm64 binaries are not currently released. For signed release and asset attestation verification, project pinning, and upgrades, follow the [distribution guide](integration/README.md). The CLI's `install` command installs a **project pin**, not the CLI on your computer.

<!-- markitect-release:install:end -->

<!-- markitect-release:try:start -->

## Try the published v0.14.1 compatibility CLI

Clone the v0.14.1 synthetic example and run a structural check with the installed binary. This reads the example without modifying an adopting repository.

Windows PowerShell:

~~~powershell
git clone --depth 1 --branch v0.14.1 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.14.1
if ($LASTEXITCODE -ne 0) { throw 'Could not get the v0.14.1 synthetic example.' }
markitect version
if ($LASTEXITCODE -ne 0) { throw 'Version check failed.' }
markitect check --repo .\markitect-sample-v0.14.1\examples\minimal
if ($LASTEXITCODE -ne 0) { throw 'Example check failed.' }
~~~

Linux amd64:

~~~sh
set -e
git clone --depth 1 --branch v0.14.1 https://github.com/Glacius-Labs/Markitect.git markitect-sample-v0.14.1
markitect version
markitect check --repo ./markitect-sample-v0.14.1/examples/minimal
~~~

The [minimal example from v0.14.1](https://github.com/Glacius-Labs/Markitect/tree/v0.14.1/examples/minimal) is a synthetic, executable fixture. To use Markitect in your own repository, follow the [release installation and verification guide](integration/README.md) to preview and install a project pin.

<!-- markitect-release:try:end -->

</details>

## Documentation

[Vision](docs/vision.md) · [Origin](docs/concepts/origin-and-rationale.md) · [Workflow](docs/project-workflow.md) · [Design](docs/design/canonical-model/README.md) · [Evidence](docs/validation/README.md) · [All documentation](docs/README.md)

## License

Markitect is licensed under [Apache-2.0](LICENSE). Third-party notices are available through `markitect licenses` and [the canonical notice file](internal/tooling/licenses/notices.md).
