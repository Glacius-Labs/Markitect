# Parallel wave: .NET adapter evidence

## Candidate and scope

- Frozen baseline: `0ba7a7218f2ceae65b8db90bb8133c2ffa583ada`.
- Adapter implementation commit: `2b1a8176c82bca334873b1dc38b424aaaded8387` on `codex/wave-dotnet`.
- Owned changes: `cmd/markitect-adapter-dotnet/main.go` and `main_test.go`.
- Declared adapter version: `markitect-dotnet/v0.1.0`.
- No shared protocol, Core, CLI, schema, dependency, or release files changed.

The parser now requires a `.csproj` root namespace to be empty or the standard MSBuild namespace (`http://schemas.microsoft.com/developer/msbuild/2003`). Recognized MSBuild elements must use the same namespace as that root. Recognized attributes (`Include`, `Condition`, `Update`, `Remove`, and `Exclude`) must be unqualified XML attributes, as MSBuild item attributes are; foreign-namespaced lookalikes produce `project-semantics-unsupported` and cannot contribute observed dependency evidence. The existing project-level `Condition` rejection is retained.

## Reproducible fixture evidence

The adapter package tests use a synthetic normalized model and these exact UTF-8 XML fixture bytes, without a BOM or trailing newline:

| Captured input path | SHA-256 | Mapped resource |
|---|---|---|
| `src/Orders/Orders.csproj` | `9afa840da6d7c15b5abcab20a954b38b76d8a034a72612ed59ec0d3dcdb045d4` | `engineering/software.markitect.org/v1alpha1/Module/orders` |
| `src/Core/Core.csproj` | `400b35829b2a391f4048da02e1108b98db09431b2947e806a184743bd1b33c3a` | `engineering/software.markitect.org/v1alpha1/Core/platform-core` |

The mapping scope contains exactly those two keys. The successful executable protocol fixture reports `literal-unconditional-project-reference-xml` with `orders -> platform-core` and no Core dependencies. The standard MSBuild XML namespace control parses an unqualified `Include="../Core/Core.csproj"`. Foreign-namespace root, element, and `Include`/`Condition`/`Update`/`Remove` attribute controls return `status: incomplete`, finding `project-semantics-unsupported`, and no `observed` value. Negative-control tests substitute only the Orders fixture bytes. These fixtures establish parser behavior only; they are not captured inputs from an adopting repository.

One executable built from the implementation commit on `go1.27.1 windows/amd64` had SHA-256 `01e38848e26ff81110ded23acc299cf4daf13fbad26cedb8b9350a15ae0dda5b`. This identifies that specific build instance. A later build can have different bytes because Go embeds build metadata, including VCS state and build paths. The command below shows how to build the adapter and inspect that build's digest; it does not promise the same digest across commits, worktree states, or toolchains:

```powershell
$exe = Join-Path $env:TEMP 'markitect-adapter-dotnet.exe'
go build -o $exe ./cmd/markitect-adapter-dotnet
Get-FileHash -LiteralPath $exe -Algorithm SHA256
```

The executable digest identifies these built bytes; it is not signed provenance or publisher identity.

## Validation and limits

`go test ./cmd/markitect-adapter-dotnet` passed on the implementation commit, including the protocol executable build, standard-namespace positive control, foreign element and attribute incomplete-status controls, existing project-level and ItemGroup condition controls, and prior mapping/plan/verification cases. `git diff --check` passed.

This evidence establishes correspondence only between explicitly mapped canonical `dependsOn` edges and literal, unconditional references in the exact captured XML subset. It does not establish evaluated MSBuild items, imported SDK behavior, a complete build graph, compilation, C# structure, or business semantics. No adopting-project files were observed in this run.
