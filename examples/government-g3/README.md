# Government G3 recursive mechanics fixture

This standalone fixture exercises Government's recursive Area execution through the native Host. The Constitution assigns the invoice root three Requirements: quantity, unit price, and their composition. It delegates quantity and price to two sibling Areas with disjoint writer paths. Their outputs are independently valid (`units=3` and `unit=4`), while the initial root executor deliberately writes `total=11`. The child checks pass. The root's fresh whole-project test and independent reviewer inspect the assembled bytes and reject the inconsistent total. A bounded second root attempt reads the actual child artifacts, writes `total=12`, reruns its check and review, then the Host obtains fresh unanimous root Ressort votes and promotes only that final root candidate.

Each child has its own configured executor and verifier slot. The trial's child programs use a unique external start-marker barrier keyed by the current shared input digest and attempt. Each process waits until its sibling is running, records process ID, invocation ID, input digest, attempt, and UTC start/overlap/finish events in its Host-provided private log, then remains alive for 800 ms. The Host receipt binds the log bytes with `privateLogDigest`. `summary.json` hashes those files and checks that the two distinct recorded process intervals overlap. A repair gets a new input-digest generation and cannot reuse old markers.

## Run the positive trial

Build Markitect from the current source checkout to an absolute path outside the checkout, then use an existing external output directory:

```powershell
go build -o "$env:TEMP\markitect-g3.exe" ./cmd/markitect
.\examples\government-g3\trial.ps1 -OutputDirectory $env:TEMP -MarkitectExecutable "$env:TEMP\markitect-g3.exe"
```

The script creates a new unique Git fixture, resolves the prior Constitution digest from the source-built `government inspect`, commits the base, and initializes `refs/markitect/government/active/example`. Source state, runner binaries, runtime JSON, Host records, private actor logs, workspaces, raw stdout/stderr, and summary are retained outside the checkout in the trial directory. The user's checkout `HEAD` must stay at the fixture base while only the managed active ref advances.

## Exercise bounded failures

Each command uses a fresh unique fixture directory and retains raw report evidence:

```powershell
# No root repair: both child checks pass, but the incorrect assembled total cannot promote.
.\examples\government-g3\trial.ps1 -OutputDirectory $env:TEMP -MarkitectExecutable "$env:TEMP\markitect-g3.exe" -MaxRepairs 0

# Exhaust the shared actual-process call limit before all fresh root evidence and votes complete.
.\examples\government-g3\trial.ps1 -OutputDirectory $env:TEMP -MarkitectExecutable "$env:TEMP\markitect-g3.exe" -MaxCalls 9

# A child proposes the root-owned invoice path; frozen Writer validation must reject it.
.\examples\government-g3\trial.ps1 -OutputDirectory $env:TEMP -MarkitectExecutable "$env:TEMP\markitect-g3.exe" -OutsideScope
```

The runner also has a `bridge` mode for a deeper cloned hierarchy. A structural bridge with no local writer returns no files and reviews descendant report lineage plus actual quantity bytes. A bridge that owns `integration/invoice.txt` can sit above two quantity and price children: its initial integration writes `total=11`, its review requires both actual child reports and reads the three composed files, and a bounded retry computes the total from child artifact bytes. The structural root can then review the final aggregate candidate without acquiring a writer path. This bridge configuration is available for focused recursive regression; the one-command example below demonstrates two direct root children.

These deterministic actors are executable process fixtures. They prove bounded delegation, disjoint child material, parent integration review, repair freshness, process isolation, vote binding, and final managed-ref gating. They do not prove independent authorship, semantic adequacy outside these fixed checks, human acceptance, model quality, or production sandboxing. All prior-Constitution authority remains fixed; child/intermediate review results never substitute for a fresh root review or final Ressort votes. G4 amendment authority and G5 restart/recovery are outside this fixture.
