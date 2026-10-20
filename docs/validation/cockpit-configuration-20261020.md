# Cockpit configuration proposals — 20 October 2026

**Scope:** the fourth slice of the [cockpit](../design/cockpit.md#working-through-the-cockpit), adopted by the owner as [DEC-065](../concepts/register.md#dec-065-work-through-the-cockpit-agent-proposed-model-changes-and-configuration): the execution settings form and the one proposal path it shares with the later model sessions, implemented in [authoring](../../internal/authoring/README.md) and the [cockpit adapter](../../internal/adapters/cockpit/README.md#proposals-for-the-execution-settings). No Apply and no provider run were involved.

## What was built

- `#/settings` reads `.markitect/execution.yaml` of the committed source and offers the budget, the role profiles, per-Package profile overrides and the team (helper roles with profile, write right and instruction paths, and the helper limit). Ownership, the top-level and per-Package instructions, scope and checks are not offered and stay as written. Only fields whose value changed are written in the YAML node tree, so unchanged fields keep their text, comments and style; a changed team is written anew.
- Proposing makes the changed file from the committed capture, runs the structural check (`apply.CheckProposal`: Model check, registry, File check, execution settings and context selection, with a second pass for newly referenced instruction and ignore files) on the proposed tree and keeps the proposal with every finding, once, under the stage that found it first. Settings read from an earlier head are refused.
- `#/proposal/{id}` shows the check, the line difference with three lines of context and the decision. Accepting takes two steps and is offered only for a passed check. Bootstrap's fence takes the lock of every observed state directory; Apply's publisher preflights the capture's guard, writes and reads every path back. The receipt is shown as published; only an applied publication accepts the proposal. Nothing is committed and no Apply starts.
- Writes need the cockpit's own JSON request; a started publication runs to its receipt when the page goes away, and a stopping cockpit waits for it.

## Demonstration

A cockpit built from this branch served a scratch clone of the branch's own repository (1,472 files) and the recorded S-N state directory, in headless Chrome.

- **Failing check.** Before the slice's files were registered, the first proposal failed the File check with ten `file.unexplained` findings naming exactly those files; acceptance was disabled. A proposal with the run time `forever` failed with the execution settings' duration finding at `.markitect/execution.yaml` line 4 and the context selection's dependent finding; discarding it wrote nothing.
- **Accepted proposal.** Agent turns 120 → 90, an own team with one writing `worker` helper and a Verify profile for `product/cockpit` produced a three-hunk difference and a passed check. Accepting wrote the file with the outcome `applied` and the read-back state `after` for `.markitect/execution.yaml`; `git status` in the clone showed the one modified file, uncommitted. Reading the settings again was refused because the working tree differs from its commit.
- **Navigation during a publication.** An earlier build tied the publication to the request; navigating away cancelled it before any write (outcome `stale`, file unchanged, proposal failed). The publication now runs detached from the request, and the page says so.
- Reading the settings took about 20 seconds and a publication about 20 seconds on this repository: both prepare or compare the whole working tree. A proposal reuses the capture of the head the owner read.
- At 390 px in dark mode the form had no horizontal scroll; the failing proposal's page kept its findings, difference and the disabled acceptance readable. No browser console errors were logged.

## Checks

`go test` of `internal/authoring`, the cockpit, CLI, apply, bootstrap and tooling packages, `go vet`, `gofmt`, the architecture gate, `markitect check` and the guarded model view write passed locally on Windows. The tests cover: editing only the changed fields, an unchanged edit reproducing the document byte for byte and a budget edit changing one line; the line difference with cut context and added, removed and unchanged files; proposals needing a repository; writes needing the cockpit's own JSON request (missing or wrong header, foreign origin, text/plain, unknown fields, trailing data); on a repository, an acceptance refused while an Apply holds the state lock and then applied with read-back and not committed; a failing check that cannot be accepted and names the edited file without duplicates; a helper instruction file the committed source holds but nothing referenced before passing the check, and a missing one failing it; settings read from an earlier head refused; a source committed after the proposal refusing the acceptance without a write; no new proposal from a capture made before an acceptance; and, in the service and at the HTTP interface with synthetic receipts, only applied or no-change publications accepted, partial, unknown and stale ones failed with their receipt, and a fenced one left open.

The independent review found that the check did not read a newly referenced instruction file, which the capture's second pass reads; the check now does. Its other findings (unchanged fields rewritten, a stale form applied to a newer source, the fence shown as a running Apply, a publication cut by stopping the cockpit, untested receipt mapping and smaller UI issues) are addressed as described above.

## Not established

Owner use of the form; a partial or unknown publication on a real repository (only its presentation and mapping are tested); the cockpit's own health view during a publication, which is composed but not exercised by a test; profile names, which only an Apply resolves against its Runtime profile; model proposals by an Agent session, the fifth slice.
