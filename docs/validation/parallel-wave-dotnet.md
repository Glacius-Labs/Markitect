# Parallel wave: .NET adapter evidence

## Candidate and scope

- Frozen baseline: `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada`.
- Adapter implementation commit: `ab65ca6e676622339b85f66249093d1c1466b358` on `codex/wave-dotnet`.
- Owned changes: `cmd/markitect-adapter-dotnet/main.go` and `main_test.go`.
- Declared adapter version: `markitect-dotnet/v0.1.0`.
- No shared protocol, Core, CLI, schema, dependency, or release files changed.

The parser now requires a `.csproj` root namespace to be empty or the standard MSBuild namespace (`http://schemas.microsoft.com/developer/msbuild/2003`). Recognized MSBuild elements must use the same namespace as that root. A foreign namespace produces `project-semantics-unsupported`; it cannot contribute observed dependency evidence. The existing project-level `Condition` rejection is retained.

## Reproducible fixture evidence

The adapter package tests use a synthetic normalized model and these exact UTF-8 XML fixture bytes, without a BOM or trailing newline:

| Captured input path | SHA-256 | Mapped resource |
|---|---|---|
| `src/Orders/Orders.csproj` | `9afa840da6d7c15b5abcab20a954b38b76d8a034a72612ed59ec0d3dcdb045d4` | `engineering/software.markitect.org/v1alpha1/Module/orders` |
| `src/Core/Core.csproj` | `400b35829b2a391f4048da02e1108b98db09431b2947e806a184743bd1b33c3a` | `engineering/software.markitect.org/v1alpha1/Core/platform-core` |

The mapping scope contains exactly those two keys. The successful executable protocol fixture reports `literal-unconditional-project-reference-xml` with `orders -> platform-core` and no Core dependencies. The standard MSBuild XML namespace control parses the literal `../Core/Core.csproj` reference. Foreign-namespace root and reference controls return `status: incomplete`, finding `project-semantics-unsupported`, and no `observed` value. These fixtures establish parser behavior only; they are not captured inputs from an adopting repository.

The executable built from the implementation commit on `go1.27.1 windows/amd64` had SHA-256 `69b79b37cc11f2d3fa0e16ec30f62d26a73b79ea044d694ac696fff875659a95`. Reproduce the build and digest from the repository root with:

```powershell
$exe = Join-Path $env:TEMP 'markitect-adapter-dotnet.exe'
go build -o $exe ./cmd/markitect-adapter-dotnet
Get-FileHash -LiteralPath $exe -Algorithm SHA256
```

The executable digest identifies these built bytes; it is not signed provenance or publisher identity.

## Validation and limits

`go test ./cmd/markitect-adapter-dotnet` passed on the implementation commit, including the protocol executable build, standard-namespace positive control, foreign-namespace incomplete-status controls, existing project-level and ItemGroup condition controls, and prior mapping/plan/verification cases. `git diff --check` passed.

This evidence establishes correspondence only between explicitly mapped canonical `dependsOn` edges and literal, unconditional references in the exact captured XML subset. It does not establish evaluated MSBuild items, imported SDK behavior, a complete build graph, compilation, C# structure, or business semantics. No adopting-project files were observed in this run.
