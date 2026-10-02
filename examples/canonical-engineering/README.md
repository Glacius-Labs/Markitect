# Canonical engineering model

This fixture registers two project-owned domains without adding kinds to the Markitect executable. `Software` describes Modules, Core, UseCases, and Interfaces. Its `dependsOn` relation contributes task context and invalidates affected checks; `owns` records membership and invalidation without automatically pulling owned resources into context. A structured constraint permits application Modules to depend only on Core resources.

`Delivery` separates desired release policy from recorded evidence. `ReleasePolicy` declares the requested checks and channel, `Release` binds that intent to a source revision, and `ReleaseEvidence` records check results. Constraints reject failed evidence and incomplete observations. These declarations do not claim that a check was truly run; evidence is an explicit input with a named source revision.

The `architecture-review` Skill explicitly uses the Software Module and Delivery Release, so its context includes those resources and the selected domain definitions containing their structured policy. The same Software constraint feeds deterministic checking and the generated Module view. The test suite changes the `allowed-targets.values` from `Core` to `Module`: that single policy edit changes the generated view and Skill context, makes a Module dependency fail under the old policy, and pass under the revised policy.

From this directory, run:

```powershell
markitect check --repo .
markitect context --repo . --kind Skill --name architecture-review --namespace engineering
markitect reconcile --repo . --action observe --adapter markitect-render
New-Item -ItemType Directory -Force .artifacts/markitect/reconcile | Out-Null
markitect reconcile --repo . --action plan --adapter markitect-render | Out-File .artifacts/markitect/reconcile/render-plan.yaml -Encoding utf8
markitect reconcile --repo . --action apply --adapter markitect-render --plan .artifacts/markitect/reconcile/render-plan.yaml --write
markitect reconcile --repo . --action verify --adapter markitect-render --plan .artifacts/markitect/reconcile/render-plan.yaml
```

The render plan reports exactly which generated files would be written. Apply requires a named non-protected feature branch and a saved, unchanged plan.

The fixture also declares a read-only .NET project-reference adapter. Build `cmd/markitect-adapter-dotnet` and place its executable on `PATH` before using it. It stages only the two declared project files, compares their literal `ProjectReference` elements with the canonical `dependsOn` relationship, and reports structural mismatches without compiling or modifying the projects. No live service account is involved.

```powershell
markitect reconcile --repo . --action observe --adapter dotnet-dependencies
New-Item -ItemType Directory -Force .artifacts/markitect/reconcile | Out-Null
markitect reconcile --repo . --action plan --adapter dotnet-dependencies | Out-File .artifacts/markitect/reconcile/dotnet-plan.yaml -Encoding utf8
markitect reconcile --repo . --action verify --adapter dotnet-dependencies --plan .artifacts/markitect/reconcile/dotnet-plan.yaml
```


