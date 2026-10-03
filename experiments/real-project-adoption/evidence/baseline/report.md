# Baseline pilot report

## Scope and evidence

The task ran in the clean public MIT MyMeetings subset at `C:\Users\Consiliari\.codex\worktrees\689d\Markitect\.artifacts\adoption\baseline-final`, starting at `ec01cb6805f93c8a4dbe37efe2a129a58a7ed28d`. The baseline scope and provenance are in `PILOT_BASELINE.md`; its upstream license is MIT. No Markitect source or docs, other pilot arms, modeling overlay, inferred policies, database, or runtime service were consulted.

The implementation adds `GetMeetingAttendeeCountQuery : QueryBase<int>` with `MeetingId` and a handler that executes `COUNT(*)` over `[meetings].[v_MeetingAttendees]`, using the same `MeetingId` predicate as `GetMeetingAttendeesQueryHandler`. The scalar result contains no attendee details. It counts rows from the existing attendee view; it does not add `GuestsNumber`, consistent with the requested attendee-record count. No endpoint was added.

## Architectural rule and source evidence

- `docs/architecture-decision-log/0009-use-2-layered-architectural-style-for-reads.md`, Decision and Consequences (source lines 13-21): query handling belongs in the application service layer, which queries the database directly without a separate mapping layer. This is the governing rule discovered for the change.
- `src/Modules/Meetings/Application/Meetings/GetMeetingAttendees/GetMeetingAttendeesQueryHandler.cs` (source lines 16-34): existing query selects attendee columns from `v_MeetingAttendees` with `WHERE MeetingId = @MeetingId`.
- `src/Modules/Meetings/Application/MeetingGroups/GetMeetingGroupDetails/GetMeetingGroupDetailsQueryHandler.cs` (source lines 43-57): existing scalar count convention uses `COUNT(*)` and Dapper `ExecuteScalarAsync<int>`.
- `src/Modules/Meetings/Application/Configuration/Queries/IQueryHandler.cs`: query handlers implement the module's MediatR-based handler contract.

These observations describe the implementation and compile result only. No database/runtime result was exercised, and no human acceptance or product-level correctness claim is made.

## Verification

`dotnet build src/Modules/Meetings/Application/CompanyName.MyMeetings.Modules.Meetings.Application.csproj --verbosity minimal -p:RestoreSources=https://api.nuget.org/v3/index.json` succeeded with 0 warnings and 0 errors. Restore used the explicitly specified public NuGet v3 feed. No tests, application execution, database access, or provider calls were performed.

## Read and activity records

- `reads.jsonl` records each successfully read source file with its UTF-8 byte count, SHA-256, and purpose.
- `inventory.jsonl` records filename-only inventory; it contains no source content.
- `searches.jsonl` records search scopes and matches separately from content reads.
- `activity.jsonl` records the corrected path-guessing mistake.
- `implementation.patch` contains the two added source files.
- `pilot-meta.json` records uncontrolled start/end wall-clock times, baseline identity, changed files, compile result, and the lack of separately instrumented call/token telemetry.

## Process notes

One content-read attempt used three nonexistent guessed paths; all failed before source content was obtained. The corresponding empty invalid ledger entries were removed and the mistake was recorded in `activity.jsonl`. The correct paths were discovered via filename inventory. `GetAllCountriesQueryHandler.cs` was a low-value extra read after the relevant scalar count convention had already been found; it confirmed only the general Dapper query style. The `MeetingId` filter was rediscovered in the existing attendee query and applied unchanged. The task's constraints against details, guest expansion, and an endpoint were carried into the scalar query shape.