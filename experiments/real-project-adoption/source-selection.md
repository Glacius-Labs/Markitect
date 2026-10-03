# Real-project pilot source selection

The pilot uses a faithful, byte-preserving subset of the MIT-licensed [`modular-monolith-with-ddd`](https://github.com/kgrzybek/modular-monolith-with-ddd) repository at commit `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8`. It is an adopter-owned architecture and application, not a Markitect default or a synthetic graph fixture.

The frozen baseline selection contains 646 upstream files (711,461 bytes):

- Root `README.md` and `LICENSE`.
- The 17 Markdown architecture decision records in `docs/architecture-decision-log/`.
- `src/global.json`, shared build properties/targets and package declarations, `.editorconfig`, and `stylecop.json`.
- All `.cs` and `.csproj` files under `Application`, `Domain`, and `IntegrationEvents` for Meetings, Payments, UserAccess, Administration, and Registrations.
- All `.cs` and `.csproj` files under the BuildingBlocks `Application`, `Domain`, and `Infrastructure` areas.
- The Meetings application architecture test `ApplicationTests.cs` as bounded implementation evidence for Copy Me.
- The Meetings `MeetingAttendees` table and `v_MeetingAttendees` view definitions as exact read-model evidence.

`source-manifest.json` in the isolated baseline lists each selected upstream relative path, byte length, and SHA-256. The preparation script verifies the upstream checkout SHA and each copied file hash before committing. The snapshot retains original relative paths and bytes. The snapshot's `PILOT_BASELINE.md` and `.gitignore` are pilot-authored files; they are not upstream inputs.

## Exclusions and limits

The snapshot deliberately excludes API and Infrastructure projects for the five Modules, the full solution, application settings, Docker/deployment files, database contents/seeds, and all other tests except the one architecture test file. It includes no credentials, customer data, local configuration, or Markitect package/context. No front end is copied. The included Meetings Application project and its selected project references are buildable; this slice is not a claim that the entire upstream application was built or can run without its omitted components and database.

The omissions keep the baseline focused on engineering guidance, module/Application and Domain structure, a technology-specific architecture check, and a concrete Meetings query task. They limit conclusions about hosting, HTTP/API integration, deployment, runtime module composition, database operations, and full-system correctness. The excluded Module Infrastructure code also means this pilot cannot claim to observe every dependency boundary in the upstream system.

This source-selection record is experiment administration. It is not part of the cold task prompt. The cold task agent should receive only [`task-request.md`](task-request.md) and the pristine baseline checkout; architecture guidance must be discovered from the repository itself.
