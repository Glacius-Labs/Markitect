# .NET project reference adapter

This standalone reference executable checks declared `dependsOn` relationships
between explicitly mapped resources against literal MSBuild
`<ProjectReference Include="..." />` entries in mapped `.csproj` inputs. It consumes only the normalized
semantic model on standard input and the exact captured files staged by the
Markitect command adapter. It does not read authoring YAML, run MSBuild, parse
C# source, change project files, or contact external services.

The evidence is scoped to the resources listed in `parameters.projectMappings`
and their explicit, unconditional project-reference entries in captured XML.
Unmapped, unrelated model resources stay outside this adapter's scope. SDK
selection on `<Project Sdk="...">` is accepted as project metadata; SDK
evaluation is not performed. `Condition`, `Import`/`ImportGroup`, item
`Update`/`Remove`/`Exclude`, property or item expressions, wildcards,
references escaping the captured repository, malformed XML, or observed target
projects without canonical resource mappings yield `status: incomplete`. A
complete result with `dependency-forbidden` or `dependency-missing` findings
means the observed literal references differ from the canonical model.

## Build and install

Build the adapter with the Go version declared by the repository:

```powershell
$adapterDirectory = Join-Path (Get-Location) ".artifacts/adapters"
New-Item -ItemType Directory -Force $adapterDirectory | Out-Null
go build -o (Join-Path $adapterDirectory "markitect-adapter-dotnet.exe") ./cmd/markitect-adapter-dotnet
$env:PATH = "$adapterDirectory;$env:PATH"
```

This places the adapter on `PATH` for the current PowerShell session. The Markitect release
currently ships the native `markitect` CLI only; this reference adapter is
included as source and is built separately.

## Configuration

Declare the project files as exact command-adapter inputs and map each
project-backed canonical resource using its full `identity.key` from the
semantic model. These keys are opaque identifiers; do not reconstruct them
from a kind or resource name. A mapped resource's canonical `dependsOn`
targets must also be mapped so observed ProjectReference paths resolve to
canonical identities.

```yaml
spec:
  adapters:
    - name: dotnet-architecture
      type: command
      version: markitect-dotnet/v0.1.0
      config:
        inputs:
          - src/Orders/Orders.csproj
          - src/Core/Core.csproj
        observe: [markitect-adapter-dotnet, observe]
        plan: [markitect-adapter-dotnet, plan]
        verify: [markitect-adapter-dotnet, verify]
        parameters:
          projectMappings:
            - resource: engineering/software.markitect.org/v1alpha1/Module/orders
              projectFile: src/Orders/Orders.csproj
            - resource: engineering/software.markitect.org/v1alpha1/Core/platform-core
              projectFile: src/Core/Core.csproj
```

The `resource` values above illustrate the qualified key shape used in the
canonical engineering example. In another project, copy the corresponding
exact `identity.key` values from its normalized model. The mapping list is the
check scope and is echoed as `observed.scope`; `observed.evidence` identifies
the bounded XML evidence kind and `observed.dependencies` carries per-resource
target identities.

## Read-only workflow

The adapter supports `observe`, `plan`, and `verify`; it does not declare an
`apply` capability. The Markitect CLI saves a plan and verifies the same
captured project inputs through the configured command:

```powershell
markitect reconcile --repo . --action observe --adapter dotnet-architecture
markitect reconcile --repo . --action plan --adapter dotnet-architecture > .artifacts/markitect/reconcile/dotnet-plan.yaml
markitect reconcile --repo . --action verify --adapter dotnet-architecture --plan .artifacts/markitect/reconcile/dotnet-plan.yaml
```

Plan results contain the canonical model digest, an explicit scope manifest,
stable observed dependency sets, and deterministic findings. Unsupported MSBuild semantics remain
incomplete; the adapter does not claim a complete evaluated build graph.
