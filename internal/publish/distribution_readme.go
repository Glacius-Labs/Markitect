package publish

import "strings"

func renderReadme(version, tag, windowsDigest, linuxDigest string) (string, error) {
	install := `## Install Markitect

The current [v{{VERSION}} release](https://github.com/Glacius-Labs/Markitect/releases/tag/{{TAG}}) provides Windows and Linux amd64 binaries. The commands below download a fixed version from the public release, check its published SHA-256 digest, and install it for your user. No Go installation or GitHub login is needed. Git is needed for Markitect commands that read Git revisions.

**Windows (PowerShell):**

` + "```powershell\n" + `if (-not [Runtime.InteropServices.RuntimeInformation]::IsOSPlatform([Runtime.InteropServices.OSPlatform]::Windows) -or [Runtime.InteropServices.RuntimeInformation]::OSArchitecture -ne [Runtime.InteropServices.Architecture]::X64) { throw 'Only Windows amd64 is released.' }
$tag = '{{TAG}}'
$asset = "markitect-$tag-windows-amd64.exe"
$sha256 = '{{WINDOWS_SHA256}}'
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
if ($LASTEXITCODE -ne 0 -or $installed -ne 'Markitect {{VERSION}} (windows/amd64)') { throw 'Installed CLI version check failed.' }
$installed
Remove-Item -LiteralPath $source, $download
` + "```\n\n" + `**Linux (bash):**

` + "```bash\n" + `(
  set -e
  [ "$(uname -s)" = Linux ] && [ "$(uname -m)" = x86_64 ] || { echo 'Only Linux amd64 is released.' >&2; exit 1; }
  tag={{TAG}}
  asset="markitect-$tag-linux-amd64"
  sha256={{LINUX_SHA256}}
  download="$(mktemp -d)"
  trap 'rm -rf "$download"' EXIT
  curl -fLsS "https://github.com/Glacius-Labs/Markitect/releases/download/$tag/$asset" -o "$download/$asset"
  printf '%s  %s\n' "$sha256" "$download/$asset" | sha256sum -c -
  install -D -m 0755 "$download/$asset" "$HOME/.local/bin/markitect"
  "$HOME/.local/bin/markitect" version
)
export PATH="$HOME/.local/bin:$PATH"
` + "```\n\n" + `Add ` + "`~/.local/bin`" + ` to your shell startup file if it is not already on ` + "`PATH`" + `. For developers who already use Go 1.27.1 or later, ` + "`go install github.com/Glacius-Labs/Markitect/cmd/markitect@{{TAG}}`" + ` is a shorter source-build option. macOS and arm64 binaries are not currently released. For signed release and asset attestation verification, project pinning, and upgrades, follow the [distribution guide](integration/README.md). The CLI's ` + "`install`" + ` command installs a **project pin**, not the CLI on your computer.

<!-- markitect-release:install:end -->`
	try := `## Try the installed CLI

Clone the v{{VERSION}} synthetic example and run a structural check with the installed binary. This reads the example without modifying an adopting repository.

Windows PowerShell:

~~~powershell
git clone --depth 1 --branch {{TAG}} https://github.com/Glacius-Labs/Markitect.git markitect-sample-{{TAG}}
if ($LASTEXITCODE -ne 0) { throw 'Could not get the v{{VERSION}} synthetic example.' }
markitect version
if ($LASTEXITCODE -ne 0) { throw 'Version check failed.' }
markitect check --repo .\markitect-sample-{{TAG}}\examples\minimal
if ($LASTEXITCODE -ne 0) { throw 'Example check failed.' }
~~~

Linux amd64:

~~~sh
set -e
git clone --depth 1 --branch {{TAG}} https://github.com/Glacius-Labs/Markitect.git markitect-sample-{{TAG}}
markitect version
markitect check --repo ./markitect-sample-{{TAG}}/examples/minimal
~~~

The [minimal example](examples/minimal/README.md) is a synthetic, executable fixture. To use Markitect in your own repository, follow the [release installation and verification guide](integration/README.md) to preview and install a project pin.

<!-- markitect-release:try:end -->`
	values := strings.NewReplacer("{{VERSION}}", version, "{{TAG}}", tag, "{{WINDOWS_SHA256}}", windowsDigest, "{{LINUX_SHA256}}", linuxDigest)
	return installStart + "\n\n" + values.Replace(install) + "\n\n" + tryStart + "\n\n" + values.Replace(try), nil
}
