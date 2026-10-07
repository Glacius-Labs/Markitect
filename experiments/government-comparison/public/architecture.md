# Reference architecture and fixture contract

Version: 0.3. These are the common preparation pins. Greenfield actors create the corresponding files themselves; Brownfield inherits its source-bound files. A live freeze repeats the availability checks on the selected runner.

## Runtime baseline

- C# targeting `net10.0`.
- SDK `10.0.103` (`global.json`, rollForward disabled) and .NET / ASP.NET Core runtime `10.0.3` are installed and required for this preparation version.
- Exact direct packages: `Microsoft.Data.Sqlite` `10.0.3`, `SQLitePCLRaw.bundle_e_sqlite3` `3.0.5`, `SQLitePCLRaw.core` `3.0.5`. Use bracketed exact package versions and a package lock. The resolved native package is `SQLite` `3.53.4`; the loaded engine reports `3.53.4`.
- Restore from NuGet.org with a local configuration clearing inherited machine feeds. Locked restore/build and `dotnet list package --vulnerable --include-transitive --no-restore` passed on the preparation host. This is the dated advisory result, not a general security guarantee.
- The first attempted graph selected vulnerable `SQLitePCLRaw.lib.e_sqlite3` `2.1.11` and raised NU1903. That graph was rejected before fixture freeze. The explicit 3.0.5 overrides were restored, locked, built and exercised through the real API. No audit warning is suppressed.

The fixture freeze records what each actor actually uses, including exact SDK/runtime output, resolved package graph or lock file, restore source, and source tree/commit SHA. These values are external shared requirements, not prebuilt Greenfield files. An actor accounts for its setup and restore work. A dependency change needs a new matched study version before affected trials. The final runner/profile decision must be made before a live comparison; if these pins do not work there, revise the public fixture version first.

## Initial boundary

The initial service is one local HTTP process with a local SQLite store. The runner supplies bind address and database location. The initial brief defines catalog reads, health, accepted orders, canonical idempotency, and order reads. Only behavior released on a task card is active; this architecture note does not disclose later task requirements.

The Brownfield cell uses a separate frozen source candidate and evidence packet. Existing useful behavior, provenance, initial checks, and compatibility obligations are established from that candidate. They do not populate Greenfield's empty repository or add requirements to its initial brief.

The contract does not prescribe project count, table names, schema shape, ID format, exact prose, or internal architecture. Agents may add abstractions where useful, with no DDD or Clean Architecture mandate.

## Preparation evidence

Preparation host checks used `dotnet --list-sdks`, `dotnet --list-runtimes`, the exact [Microsoft.Data.Sqlite 10.0.3](https://www.nuget.org/packages/Microsoft.Data.Sqlite/10.0.3), [SQLitePCLRaw bundle 3.0.5](https://www.nuget.org/packages/SQLitePCLRaw.bundle_e_sqlite3/3.0.5) and [SQLite 3.53.4](https://www.nuget.org/packages/SQLite/3.53.4) packages. Before a study freeze, repeat version and restore/build/audit checks on the actual runner and capture their outputs. NuGet package availability does not establish runner compatibility or trial readiness.
