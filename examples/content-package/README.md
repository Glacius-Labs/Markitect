# Offline review package

This neutral example contains one exported Workflow, plus package-private Skill and Text resources and one declared ordinary text input. “Private” means unexported from direct consumer selection; package archives are not confidentiality boundaries.

## Pack from a fixed Git commit

Copy this directory into a temporary, isolated Git repository, commit it, then pack that exact commit from the Markitect source checkout:

```powershell
$packageSource = Join-Path $env:TEMP 'review-guidance-source'
New-Item -ItemType Directory -Force $packageSource | Out-Null
Copy-Item -Recurse -Force 'examples/content-package/*' $packageSource
git -C $packageSource init -b main
git -C $packageSource config user.name 'Package Author'
git -C $packageSource config user.email 'package-author@example.invalid'
git -C $packageSource add .
git -C $packageSource commit -m 'Add review guidance package'
$revision = git -C $packageSource rev-parse HEAD
$archive = Join-Path $env:TEMP 'review-guidance-1.0.0.zip'
go run ./cmd/markitect pack --repo $packageSource --revision $revision --output $archive
```

`pack` reads only the selected commit. It creates the ZIP only if the output path is absent and prints an exact SHA-256 pin suggestion. The source coordinate is provenance text; Markitect does not fetch it.

## Consume the reviewed archive

[The consumer fixture](../package-consumer/README.md) vendors a deterministic archive built from this example and pins its SHA-256. It demonstrates the exported Workflow, a consumer-owned wrapper, and package-qualified context selection.
