Task: Add GetMeetingAttendeeCount Query to Meetings. Accept MeetingId and return only the count of rows that GetMeetingAttendees returns. Guest numbers are fields on attendee rows. Do not add an HTTP endpoint.

Run boundary and starting point
- Adopter: this isolated project at .artifacts/adoption/assisted-final.
- Starting committed SHA: fd94a689aab857704c3a4a45ac2abe0fc0e3185c (verified before work).
- No commits, upstream checkout, or other pilot-arm files were changed or inspected. A postrun parent-root audit found three unintended root-level files matching paths from the earlier apply_patch call (the new UseCase resource and the two C# files). This was a cross-boundary write by the operator, not an adopter product change; the parent agent is preserving the exact files for audit and removing only those verified paths and empty directories. I did not inspect or remove those parent-root files.
- Start time: 2026-10-03T05:17:29.9360826+02:00. End time is recorded in metadata.json.

Implementation and architecture
- Added public GetMeetingAttendeeCountQuery : QueryBase<int> with a MeetingId property.
- Added an Application handler that runs SELECT COUNT(*) against the same meetings.v_MeetingAttendees view and filters by MeetingId. It returns a scalar int; it does not load attendee details or add GuestsNumber.
- Added canonical UseCase and Handler resources that bind the query and handler to the existing meeting-attendance Feature and shared SQL connection factory.
- No endpoint, feature taxonomy change, or narrative change was needed. The supplied context and existing Narrative show the attendee view has one row per attendee and the CQRS read model already governs this behavior.
- Relevant context was present: the frozen context supplied the task, the existing modeled UseCase/Handler, Feature, Common, and vertical-slice workflow. No architecture rediscovery was needed. The context filename was temporarily renamed by the setup operator during the run; the initial successful read and later restored reread have the same SHA. A failed repeat-read entry was mistakenly written as 0 bytes/null hash and is excluded from successful read metrics; see activity.jsonl.
- No competing count implementation or missed canonical update was found. The new query uses the existing read model and the two new model records. Other unselected context inputs were treated as irrelevant to this bounded read-side task.

Changed adopter files
- src/Modules/Meetings/Application/Meetings/GetMeetingAttendeeCount/GetMeetingAttendeeCountQuery.cs
- src/Modules/Meetings/Application/Meetings/GetMeetingAttendeeCount/GetMeetingAttendeeCountQueryHandler.cs
- resources/usecase-get-meeting-attendee-count.yaml
- resources/handler-get-meeting-attendee-count.yaml

Build
- Required command: dotnet build src/Modules/Meetings/Application/CompanyName.MyMeetings.Modules.Meetings.Application.csproj --verbosity minimal -p:RestoreSources=https://api.nuget.org/v3/index.json
- First attempt could not access the public NuGet service index in the sandbox and failed restore with NU1900 vulnerability-data errors treated as errors.
- Retried the same command with public NuGet network access. Restore and build succeeded with 0 warnings and 0 errors. Full attempts are in build-results.jsonl; console output is in build-output.txt and build-output-network.txt.
- No tests or runtime/API/DB/cloud calls were run.

Evidence and telemetry
- Official run evidence is under .artifacts/adoption/evidence/assisted-final. patch.diff contains the patch; reads.jsonl records byte counts and raw UTF-8 SHA256 for successful reads; inventory.jsonl and searches.jsonl record path listings and searches.
- Call-count and token telemetry were unavailable. File byte counts are not token counts; this single uncontrolled run supports no productivity claim.
