# MyMeetings .NET project-reference evidence check

This project-owned check observes `ProjectReference` items as evaluated by MSBuild for the fixed MyMeetings source snapshot. It compares those edges with the explicitly mapped Core/Module ownership and layer policy; it does not inspect C# dependency semantics.

The map covers 18 projects from the extracted pilot: Application, Domain, and IntegrationEvents for five Modules, plus BuildingBlocks Application, Domain, and Infrastructure. Every observed reference target must be in the map. Cross-Module references may target only another Module's `IntegrationEvents` project. BuildingBlocks may not reference Module projects. Module-to-Core inward references and within-Module references are allowed by this check.

## Use the check in the adopter snapshot

Copy both `check-module-project-references.ps1` and `project-map.json` into the adopter snapshot `scripts/` directory. The project policy check should run `pwsh -NoProfile -File scripts/check-module-project-references.ps1`; its working directory must be the adopter root. This is a project-owned check asset and must be included in the fixed Markitect inputs when a run uses it.

## Run the positive snapshot and the control mutation

From the Markitect checkout, using the fixed baseline snapshot:

```powershell
pwsh -NoProfile -File experiments/real-project-adoption/checks/check-module-project-references.ps1 .artifacts/adoption/baseline-final
pwsh -NoProfile -File experiments/real-project-adoption/checks/check-module-project-references.ps1 .artifacts/adoption/baseline-final -Control
```

The first invocation exits zero only when all mapped edges satisfy the policy. It writes a JSON evidence report to stdout and returns nonzero for a boundary violation, an unmapped/missing project reference, or a missing source manifest, wrong declared upstream commit, or mapped project path absent from its list. It does not rehash every selection-manifest entry against the actual source files. It binds the observed snapshot to the `upstreamCommit` in `source-manifest.json`, records the optional materialized Git revision when available, runs MSBuild from `src` so that the project's `global.json` selects its .NET SDK, and hashes every mapped project plus the check, map, source manifest, and shared MSBuild inputs. The report includes exact evaluated project paths/edges and MSBuild reference-origin metadata. Its input hashes cover the listed project/check/map/manifest and shared files, not the entire SDK or implicit import chain; the recorded run used .NET SDK 8.0.418. It does not require `.git`, so it also runs on a materialized Markitect `--revision` snapshot.

The second invocation proves the negative control through MSBuild itself. It adds one temporary `CustomAfterMicrosoftCommonTargets` import for `Payments.Application -> UserAccess.Application`, evaluates the same project with `dotnet msbuild -getItem:ProjectReference`, and succeeds only when the evaluator reports that injected edge as a prohibited cross-Module dependency. It writes only a uniquely named `.targets` file in the operating-system temporary directory and removes it in `finally`; it does not edit the source snapshot. The edge's `definingProject` in the report identifies that temporary target file.

The frozen baseline returned zero findings for the 18 mapped projects and 35 evaluated edges. `src/Directory.Build.targets` contributes evaluated references; an XML-only scan of each `.csproj` would not be sufficient. The explicit project map binds these observed paths to the corresponding canonical Module resources.

## Evidence boundary and known uncovered edge

This proves only that MSBuild evaluated these mapped project references and that the listed edges satisfy the stated project/layer rule. It does not prove runtime dependency injection, reflection/plugin loading, HTTP or messaging dependencies, business behavior, or that an `IntegrationEvents` assembly contains only suitable stable contracts. It does not prove that the entire upstream repository conforms, because the pilot intentionally extracts only selected projects.

A separate read-only inventory of the full upstream project set found `src/Modules/Registrations/Infrastructure/CompanyName.MyMeetings.Modules.Registrations.Infrastructure.csproj` directly references `src/Modules/UserAccess/Application/CompanyName.MyMeetings.Modules.UserAccess.Application.csproj` and `src/Modules/UserAccess/Infrastructure/CompanyName.MyMeetings.Modules.UserAccess.Infrastructure.csproj`. Those projects are outside the extracted map and are therefore not silently treated as compliant or as allowed exceptions. The relevant adapter code is `src/Modules/Registrations/Infrastructure/Users/UserAccessGateway.cs` and `src/Modules/Registrations/Infrastructure/Configuration/UserAccess/UserAccessAutofacModule.cs`; this is a concrete cross-Module implementation dependency behind an application-owned interface and requires human review if the pilot's boundary is expected to cover Infrastructure.

The evaluated references come from project files and imported MSBuild configuration. They are technology-specific observations; the canonical Domain owns intended ownership, dependencies, and contract boundaries. This check provides evidence against that intent and is deliberately project-owned rather than a Core language feature.