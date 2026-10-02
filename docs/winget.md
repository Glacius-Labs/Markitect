# WinGet package

The submitted portable package manifests for `GlaciusLabs.Markitect` v0.5.0 are in [`packaging/winget/GlaciusLabs.Markitect/0.5.0`](../packaging/winget/GlaciusLabs.Markitect/0.5.0/). They use WinGet manifest schema 1.12.0 and the immutable Windows amd64 release asset, with its published SHA-256. The package supports Windows x64; Markitect does not publish a Windows arm64 binary.

The package is submitted to the WinGet community source in [PR #445055](https://github.com/microsoft/winget-pkgs/pull/445055). It is not available through `winget install` until that PR is accepted and indexed.

## Validate and test

The repeatable Windows install and upgrade check is [`Test-InstallUpgrade.ps1`](../packaging/winget/Test-InstallUpgrade.ps1). Run it from an elevated PowerShell 7.2 or later session on a Windows x64 machine with App Installer installed:

```powershell
$appInstaller = Get-AppxPackage Microsoft.DesktopAppInstaller
$winget = Join-Path $appInstaller.InstallLocation 'winget.exe'
.\packaging\winget\Test-InstallUpgrade.ps1 -WingetBinary $winget -OutputDirectory "$env:TEMP\markitect-winget-test"
```

The script validates the submitted v0.5.0 manifests and a test-only v0.4.1 manifest, installs the genuine v0.4.1 release, upgrades to v0.5.0, and checks WinGet's recorded version, the executable's `version` output, and the installed binary and PATH alias hashes. It writes a command transcript and a YAML-compatible JSON report to the output directory. It tests only `GlaciusLabs.Markitect`; by default it stops if that package is already installed and removes the package registration it creates. `-AllowPackageReplacement` is intended only for disposable CI runners. Add `-LeaveInstalled` if the tested v0.5.0 package should remain installed.

WinGet requires the administrator-controlled `LocalManifestFiles` setting for local manifest installation. The script snapshots its original administrative setting, enables the feature for the test, and restores the previous state in `finally`. An elevated process is required. WinGet normally adds its portable command directory `%LOCALAPPDATA%\Microsoft\WinGet\Links` to the user's `PATH` and creates a `markitect.exe` alias targeting the installed binary. If symlink creation fails, WinGet can add the package directory instead. The script verifies the selected route, release SHA-256, and command/version in a fresh PowerShell process using the persisted PATH. It then confirms that uninstall removes the registration, executable, and alias. Open a new terminal to use `markitect` after installation. See [the pinned WinGet implementation](https://github.com/microsoft/winget-cli/blob/v1.29.380/src/AppInstallerCLICore/PortableInstaller.cpp).

The [Windows workflow](../.github/workflows/winget-validation.yaml) installs the official WinGet v1.29.380 bundle and dependencies with pinned SHA-256 hashes. It removes the unrelated Microsoft Store source only on that disposable runner, avoiding Store agreement prompts during inventory queries. On a local machine, prepare trusted sources yourself; the script does not remove sources or accept an inventory agreement prompt.
## Later releases

Each Markitect release needs its own three-file manifest set with that release's version-specific Windows binary URL and exact SHA-256. Keep each community-repository submission to one package version and follow its PR template's schema recommendation. A manifest in this source checkout does not publish or update the community package by itself.

## Local upgrade correlation

Before the community package is indexed, a local manifest install has the product code `GlaciusLabs.Markitect__DefaultSource`. The pinned WinGet client searches a composite source when upgrading a manifest; automatic portable product-code matching uses that source identity. The runner therefore creates a test-only v0.5.0 manifest copy with an explicit `ProductCode` matching its own local registration. Release URLs, versions, and hashes are unchanged, and the three canonical submitted manifests are not edited. The report records this test-only correlation and marks public-catalog upgrade testing as unavailable. Test the unmodified catalog install and upgrade separately after publication. See [the pinned manifest lookup implementation](https://github.com/microsoft/winget-cli/blob/v1.29.380/src/AppInstallerCLICore/Workflows/WorkflowBase.cpp).

## Prepared release update and remaining catalog gates

The generated [v0.7.0 manifests](../packaging/winget/GlaciusLabs.Markitect/0.7.0/)
use the verified immutable Windows release and passed native WinGet manifest
validation. They are prepared for a separate update after the initial v0.5.0
submission is accepted; generation is not submission or public availability.

The remaining distribution gates are Microsoft's manual review and catalog
indexing, a fresh-runner public-source installation of v0.5.0, the v0.7.0
community update and indexing, and the real public-source upgrade from v0.5.0
to v0.7.0. The public tests must retain versions, release digests, fresh-process
PATH/alias checks and cleanup evidence. The local-manifest ProductCode is
test-only and must not be added to the community update.

Only after the public installation succeeds should the README advertise
WinGet commands. The current README already installs the verified v0.9.0
binary directly; that route does not require Microsoft catalog approval.

## Current verified release metadata

The generated [v0.9.0 manifests](../packaging/winget/GlaciusLabs.Markitect/0.9.0/) bind the immutable Windows binary and its attested SHA-256. The Windows workflow validates every canonical versioned manifest directory, including v0.9.0. Its portable install/upgrade exercise still checks the submitted v0.5.0 package against v0.4.1; it does not demonstrate a public-catalog v0.9.0 installation or upgrade. The initial community submission and later version submissions remain separate catalog gates.


## Prepared public-catalog runner

The same workflow now has explicit manual modes. Its default remains the
local-manifest lifecycle. After the initial submission is merged and indexed,
run:

```powershell
gh workflow run winget-validation.yaml --repo Glacius-Labs/Markitect --ref main -f mode=catalog-install
```

After the v0.7.0 community update is separately merged and indexed, run:

```powershell
gh workflow run winget-validation.yaml --repo Glacius-Labs/Markitect --ref main -f mode=catalog-upgrade
```

The first mode installs the public v0.5.0 package. The second installs that same
version and upgrades it explicitly to v0.7.0. Both bind the community source
identifier and URL, use its actual portable ProductCode, and verify the release
digest, executable version, correct alias or PATH fallback, and command
resolution in a fresh process. They never enable local manifests or inject a
test ProductCode. They refuse replacement or retention, require fresh evidence,
and uninstall with purge; registration, executable, alias, package directory and
test-created PATH entries must be gone. Reports retain the mode, tested versions,
source identity, settings state, PATH before/after and cleanup results.

Contract checks cover public commands, ambiguous registrations, source identity,
pinned versions and digest consistency without installing anything. Existing
local-manifest lifecycle CI remains the regression gate. The public modes are
prepared but have not been exercised against Microsoft's catalog: PR #445055
is still awaiting manual review. Their preparation is not publication evidence.
The current direct-download README installs v0.9.0; the already-authorized
catalog follow-up is explicitly v0.7.0.
