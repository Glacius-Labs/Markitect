# Copy Me context-run compilation

The committed `context-run.yaml` selected `engineering/Skill/architecture-review` from the fixed candidate `fd94a689aab857704c3a4a45ac2abe0fc0e3185c`. It carried task `copy-me-query-convention` at `work-items/copy-me-query-convention.md` and compiled successfully (exit 0) with Markitect 0.12.0.

Context digest: `sha256:c9ec63e24bd1cf0fdf8b70a92a0e9479f86c4d2eccd6e6d938e157937b09c30b`. Snapshot digest: `594d527fe69074a9fd7357983ce7c7a8d50307812e638a517c939382673de8ef`. The compiled run contains 38 input records: 11 typed resources, 3 definition inputs (Project, Package, and Domain), 12 project artifact files, and 12 run records (the manifest, task file, and 10 source files). The source manifest declared these 10 files:

- `docs/architecture-decision-log/0007-use-cqrs-architectural-style.md`
- `docs/architecture-decision-log/0009-use-2-layered-architectural-style-for-reads.md`
- `src/Modules/Meetings/Tests/ArchTests/Application/ApplicationTests.cs`
- `src/Modules/Meetings/Application/MeetingGroups/GetMeetingGroupDetails/GetMeetingGroupDetailsQuery.cs`
- `src/Modules/Meetings/Application/MeetingGroups/GetMeetingGroupDetails/GetMeetingGroupDetailsQueryHandler.cs`
- `src/Modules/Meetings/Application/MeetingGroups/GetMeetingGroupDetails/MeetingGroupDetailsDto.cs`
- `src/Modules/Meetings/Application/Countries/GetAllCountriesQuery.cs`
- `src/Modules/Meetings/Application/Countries/GetAllCountriesQueryHandler.cs`
- `src/Modules/UserAccess/Application/Emails/GetAllEmailsQuery.cs`
- `src/Modules/UserAccess/Application/Emails/GetAllEmailsQueryHandler.cs`

The compiled output is `context-run-copy-me.yaml`; `context-run-copy-me.json` retains the exact invocation, stdout/stderr, exit, and unchanged working-tree hashes. `discovery-ledger.json` binds it to the selected Skill, snapshot, model, tool, and package pin evidence. No adopting-project source files were changed during compilation.
