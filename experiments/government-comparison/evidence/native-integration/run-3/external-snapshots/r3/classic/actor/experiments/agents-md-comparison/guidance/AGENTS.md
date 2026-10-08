# MyMeetings engineering guidance

Use this file as a navigation map to current project-owned evidence. Read the referenced ADR before changing a boundary or convention.

## Ownership and communication

- ADR 0004, `docs/architecture-decision-log/0004-divide-the-system-into-4-modules.md`, records the module boundaries and autonomy goal.
- ADR 0014, `docs/architecture-decision-log/0014-event-driven-communication-between-modules.md`, records asynchronous event-driven communication as the preferred module integration.
- Keep each module's Application logic and read model in that module. Do not add an Application-to-Application project reference without an explicitly approved exception.
- Existing cross-module contracts live under each module's `IntegrationEvents` project. Inspect the publisher and consumer before changing integration behavior.
- ADR 0017, `docs/architecture-decision-log/0017-implement-archictecture-tests.md`, explains the role of module architecture tests.

## Application query conventions

- Read ADR 0009, `docs/architecture-decision-log/0009-use-2-layered-architectural-style-for-reads.md`, for the query-layer decision and its tradeoffs.
- Query contracts are in the module's `Application/Contracts/QueryBase.cs` and `IQuery.cs`; handler contracts are in `Application/Configuration/Queries/IQueryHandler.cs`.
- Queries are immutable. Existing architecture tests enforce this and the local handler naming and visibility conventions.
- Query handlers use the module's existing SQL connection factory and Dapper patterns. Start from the nearest existing query and preserve its input predicate and result mapping conventions.
- Read the relevant SQL view and table definition before relying on a column, nullability, or row meaning.
- The Meetings application query tests are in `src/Modules/Meetings/Tests/ArchTests/Application/ApplicationTests.cs`.

## Validation and verification

- Find the governing policy version and scope in `docs/engineering-policy.md` and the associated ADRs before changing policy-sensitive behavior.
- Existing Validators provide the implementation and FluentValidation naming examples; a package selection rule does not itself define each field-level validation rule.
- Read the adopter policy copy at `docs/engineering-policy.md`; use only the copy frozen for this task.
- Run the available existing architecture checks and affected project build.
- Capture command, revision, exit code, and relevant output. Explain what each check establishes and any check that could not run.
- Build success verifies compilation only. Do not infer database execution, runtime registration, or human acceptance from it.
