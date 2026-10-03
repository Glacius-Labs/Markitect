# Copy Me evidence selection: Query handling convention

## Scope

- Public source: https://github.com/kgrzybek/modular-monolith-with-ddd.git
- Fixed public evidence origin revision: `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8` (`https://github.com/kgrzybek/modular-monolith-with-ddd.git`). The immutable ContextRun used by the adopter is pinned separately to adopter snapshot `fd94a689aab857704c3a4a45ac2abe0fc0e3185c`; its copy/input hashes below are authoritative for the helper.
- Selection question: What query request/handler conventions are explicitly recorded and enforced in the selected MyMeetings application evidence, and which details remain implementation choices?
- Selection size and observed effort: 10 UTF-8 files selected; I inspected the content of those 10 exact source files for this exercise. Authoring elapsed time was not instrumented, so no time value is reported. No full-repository scan, issue history, private conversation, credentials, or customer data was used.
- Selection rationale: two accepted ADRs state read/write intent and read-layer policy; a module architecture test gives executable expectations; two Meetings Query examples show the local request/handler shape; one UserAccess example probes implementation variation across modules.
- No frequency or prevalence estimate is claimed. Examples were deliberately selected for relevance, not sampled randomly.
- All paths and SHA-256 values are fixed in `evidence.yaml`. Short excerpts/line anchors below are only navigation aids; the complete selected files are the evidence.

## Selected files

| ID | Stance | Exact path | Purpose |
|---|---|---|---|
| E-01 | supports | `docs/architecture-decision-log/0007-use-cqrs-architectural-style.md` | Accepted stated intent for module CQRS and separate read/write models |
| E-02 | supports / qualifies | `docs/architecture-decision-log/0009-use-2-layered-architectural-style-for-reads.md` | Query handling layers; explicitly accepts application/database coupling and no DB abstraction |
| E-03 | supports | `src/Modules/Meetings/Tests/ArchTests/Application/ApplicationTests.cs` | Architecture tests for Query immutability, QueryHandler suffix, and non-public handlers |
| E-04 | supports | `src/Modules/Meetings/Application/MeetingGroups/GetMeetingGroupDetails/GetMeetingGroupDetailsQuery.cs` | Immutable typed query request |
| E-05 | supports | `src/Modules/Meetings/Application/MeetingGroups/GetMeetingGroupDetails/GetMeetingGroupDetailsQueryHandler.cs` | Internal `IQueryHandler`; Dapper against a read view plus a second count query |
| E-06 | supports | `src/Modules/Meetings/Application/MeetingGroups/GetMeetingGroupDetails/MeetingGroupDetailsDto.cs` | Query response shape returned by the handler |
| E-07 | supports | `src/Modules/Meetings/Application/Countries/GetAllCountriesQuery.cs` | A second, parameterless typed Query request |
| E-08 | supports | `src/Modules/Meetings/Application/Countries/GetAllCountriesQueryHandler.cs` | Internal `IQueryHandler` querying a Meetings read view |
| E-09 | qualifies | `src/Modules/UserAccess/Application/Emails/GetAllEmailsQuery.cs` | Cross-module Query request example |
| E-10 | counterexample to storage-specific interpretation | `src/Modules/UserAccess/Application/Emails/GetAllEmailsQueryHandler.cs` | Same query-handler shape, but selects from `[app].[Emails]` directly instead of a module view |

## Exclusions

The sample does not establish that all project modules or query handlers follow one implementation pattern, that each Query has exactly one Handler, that all reads must use database views, or that an ADR written in 2019 remains an approved future direction. The ArchitectureTests apply to the Meetings Application assembly. The UserAccess files provide one cross-module comparison only. Semantic purity, transactions, authorization, runtime behavior, and whether direct table access is desirable require broader checks or owner judgment.
