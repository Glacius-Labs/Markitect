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
