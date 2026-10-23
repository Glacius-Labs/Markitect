# Requirements

All records are proposed. `Must` means binding if adopted; `Goal` is an intended benefit, not a deterministic pass/fail promise. Scopes and verification targets are declared in [Relationships](relations.md).

<a id="req-overview"></a>
## REQ-OVERVIEW — Keep tasks visible

Type: Requirement. Strength: Goal.

A person should be able to see their recorded tasks and distinguish outstanding work from completed work. Whether the application actually improves a person's overview needs user observation; CLI correctness alone does not establish that benefit.

<a id="req-scope"></a>
## REQ-SCOPE — Small first version

Type: Requirement. Strength: Must.

The offered domain actions in version 1 are exactly the three UseCases declared in this model. No editing, deleting, reopening, filtering, task dates, priorities, accounts, shared use, synchronization or reminders are offered. Error handling and file selection are supporting behavior, not new domain actions. Extension of the offered scope requires an explicit intent change.

<a id="req-durability"></a>
## REQ-DURABILITY — Survive an ordinary restart

Type: Requirement. Strength: Must.

After a mutation has successfully stored its result, a later invocation using the same task file observes the same task references, texts and states, except for changes made by another successful mutation. Successful completion of a command includes successful storage when that command changes data. Ordinary process exit and restart must preserve the result.

This obligation does not promise recovery from power loss, process termination during a write or simultaneous writers; the execution boundary is owned by [local execution](#req-local).

<a id="req-safe-storage"></a>
## REQ-SAFE-STORAGE — Preserve data on storage failure

Type: Requirement. Strength: Must.

A missing task file represents an empty collection. Reading it does not create a file. An existing unreadable, malformed or unsupported file is a storage error, never an empty collection. A malformed record, duplicate reference or invalid Task property makes the file invalid; no partial collection is accepted.

For a mutation, construct the complete next state before publishing it. If reading or writing fails and a storage error is returned, the previous task file remains unchanged byte-for-byte. Do not truncate or overwrite an unreadable existing file. An interrupted write's temporary data must not be mistaken for the selected task file. The implementation may select a safe staging/replacement mechanism that fulfills this normal-operation contract.

<a id="req-errors"></a>
## REQ-ERRORS — Distinguish outcomes

Type: Requirement. Strength: Must.

Successful commands use exit code `0`, put their defined result on stdout and leave stderr empty. Invalid invocation, invalid input and unknown task references use exit code `1`; storage failures use exit code `2`. Failed commands leave stdout empty and write one explanatory line to stderr. No failed operation may report success or change stored data.

The CLI's generic syntax failure is `Invalid command or arguments.` Missing or extra arguments, unknown commands, an empty file path or a task reference outside the Concept's reference syntax use this result. Operation-specific validation and unknown-reference messages belong to their UseCase. Storage failures report `Cannot read task file.` or `Cannot write task file.` as appropriate. No success output is emitted before all required storage has succeeded.

All lines end with the platform's normal newline; comparisons normalize CRLF/LF. Public messages in this version are English. Infrastructure details may be logged separately only if they do not change this interface or expose task text as an error dump.

<a id="req-local"></a>
## REQ-LOCAL — Local execution boundary

Type: Requirement. Strength: Must.

The application runs for one person on a local machine without a server, network request or service dependency for its task operations. Invocations against the same task file are sequential. Simultaneous writers, manual file editing during execution and crash/power-loss recovery are outside version 1's guarantees.

The chosen task file is established once per invocation through the CLI contract. A selected file is never silently replaced by another location. The parent directory must already exist; a missing directory is a storage failure, not permission to create a directory tree.

<a id="req-docs"></a>
## REQ-DOCS — Keep instructions consistent

Type: Requirement. Strength: Must.

The planned reader and agent resources must be present when the application is realized. Each resource's exact content obligation belongs to its Artifact definition. When behavior, setup, start, file selection or developer commands change, update the affected owner documents in the same change. Agent instructions route to the specification and developer guide; do not maintain a second independent copy of their rules.

<a id="req-change"></a>
## REQ-CHANGE — Develop against explicit intent

Type: Requirement. Strength: Must.

For a behavior change, identify the affected canonical contracts and propose any intended change before realizing it. Update affected tests and instructions with the candidate. A repair of behavior already required by the model needs no invented intent change. Independent review checks the candidate against the current contracts, relevant Check observations and affected documentation. Repeat affected verification after repairs; retain the actual source revision of reused evidence.

A proposed contract change is not accepted by a passing test or by a commit. The owner decides whether to adopt the intent. This model does not prescribe a hosting service, mandatory external PR policy or a permanent agent hierarchy.
