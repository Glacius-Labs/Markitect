# Government project walkthrough: workshop booking

This small project shows how an operator can turn explicitly authored intent into a reviewable Government model, accept the exact model digest, make a bounded implementation on a feature branch, and continue a previously authorized model amendment after an interruption.

The intake form is [`intent.yaml`](intent.yaml). It records domain terms, priorities, fixed rules, allowed discretion, assumptions, an open question, and file assignments. Treat it as working notes: manually reconcile each machine-relevant decision into `government.yaml`, which is the canonical model. The script does not parse prose or infer model semantics. Edit and review both files, and make sure they agree before accepting the model digest. After Continue, `intent.yaml` remains the historical intake snapshot. Apply any accepted amendment to the canonical `government.yaml`; update the intake separately if you want it to reflect later decisions.

The accepted starting model records the current calculation as seat count multiplied by seat price; the separate published fee is fixed at 2 euros but is not yet included. The root Area already has prior `implement`, `review`, and `amend-model` authority for these listed subjects. The first bounded order changes only the independent confirmation label. The second order explicitly adds the fixed fee once per booking and updates its implementation and check. Both Ressorts are frozen in the source Constitution; every run requires their separate final assent. The amendment cannot create or extend its own authority.

## Prepare and review

From the Markitect checkout, build the CLI once and choose a new absolute directory outside the checkout for the example project:

```powershell
go build -o .artifacts/government-p1/markitect-p1.exe ./cmd/markitect
$Project = 'C:\temp\studio-booking-government'
$Markitect = (Resolve-Path .artifacts/government-p1/markitect-p1.exe).Path
pwsh -NoProfile -File examples/government-project/walkthrough.ps1 `
  -Action Prepare -ProjectDirectory $Project -MarkitectExecutable $Markitect
$Walkthrough = (Resolve-Path examples/government-project/walkthrough.ps1).Path
```

Prepare copies the starter project and emits the current model digest using read-only `government inspect --format json`. Review and, if needed, edit both `$Project\intent.yaml` and `$Project\government.yaml`. Inspect the complete source model and the recorded digest:

```powershell
Get-Content "$Project\intent.yaml"
Get-Content "$Project\government.yaml"
& $Markitect government --repo $Project --action inspect --format json --config government.yaml
```

The draft has no acceptance record. A prose review or a green structural check does not count as acceptance.

## Explicit acceptance and bounded work

After reviewing the model, copy its exact `model.digest` value from inspect and pass it as `-AcceptedDigest`. This explicit operator action creates a local acceptance record and runs the bounded work on the already prepared non-protected `codex/studio-booking` branch:

```powershell
$Digest = 'sha256:<paste-the-exact-64-lowercase-hex-digest-here>'
pwsh -NoProfile -File $Walkthrough `
  -Action Continue -ProjectDirectory $Project -MarkitectExecutable $Markitect `
  -AcceptedDigest $Digest
```

Continue saves both read-only scoped plans as JSON under `$Project.government-run` before it starts any mutating Government run or queue process. Inspect those files alongside the accepted commit, run reports, votes, and queue journal.

Continue refuses a missing, malformed, or stale digest. It writes the exact orders into the accepted Git commit before computing the run base, then builds the fixed deterministic role adapter and performs the implementation order. It starts one finite amendment queue, pauses a temporary local Git proxy after the actual `update-ref`, kills the Markitect host process tree, and resumes that same durable queue. The saved amendment report must show the changed model digest, changed model and realization files, passing acceptance check, and both final Ressort assents.

If a process is interrupted for another reason after the durable queue exists, resume it with the exact accepted digest and the queue directory printed by the prior run. Resume checks the durable acceptance record and the project-specific persistent budget before starting the controller:

```powershell
$Queue = 'C:\temp\studio-booking-government.government-run\state\amendment\government-queue-<id>'
pwsh -NoProfile -File $Walkthrough `
  -Action Resume -ProjectDirectory $Project -MarkitectExecutable $Markitect `
  -AcceptedDigest $Digest -QueueDirectory $Queue
```

For a concise record of the explicit digest action, `human-acceptance.json` stores the supplied digest and timestamp. It is provenance for this run; it does not authenticate the operator or make a model correct. A controller smoke run may pass `-MechanicalSmoke`; its record is labelled as a mechanical fixture acknowledgement and must never be presented as human or semantic acceptance.

## Evidence and limits

The complete Continue path starts three mutating Markitect host processes: one implementation `run`, one amendment `queue` that is deliberately interrupted, and one `resume`. A persistent external budget file reserves at most three controller starts and eight deterministic role starts; a failed attempt does not refill either budget. The queue caps the amendment at four actor starts; the implementation uses four. Each role has a 60-second timeout, each Markitect host has a five-minute outer process limit, the queue has a ten-minute wall-time cap for interruption and recovery, and queue parallelism is one. Runtime state, temporary files, binaries, gate markers, JSON plans, and process logs are in the sibling `*.government-run` directory, outside the Git repository.

The Go acceptance check validates the fixture's exact booking arithmetic and published fee. The runner and Git proxy are fixed local fixtures. Their assent responses establish only that the Host binds distinct final vote records to this frozen candidate/evidence; they are not human approvals, model quality evidence, real-model behavior, sandbox proof, or a productivity measurement. A normal operator's exact-digest action is an unauthenticated attestation; a smoke invocation is explicitly marked mechanical-only. Intake, reconciliation, model review, edits, and the decision to proceed remain manual.

Git refs and project evidence stay in the chosen project directory; controller state and process output stay in its sibling `*.government-run` directory. Every controller invocation writes uniquely named stdout and stderr files so a failed retry does not replace earlier evidence. Use a fresh project and sibling run directory for another acceptance; Prepare refuses to overwrite either.
