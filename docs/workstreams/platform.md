# Platform (lane 5)

The platform lane covers where and how Markitect runs on Windows and Linux: the Apply host, tests, evaluation runs and the container engine. It is a topic lane: it takes stock and discusses options with the owner, and builds nothing until the owner says so. This page holds the lane's scope, a short state, the known problems with their evidence, options and open owner questions. It links to the pages that own the details.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026.

## Scope

- **Owns:** the questions of where Markitect executes, which engine and host setup it supports, and how development and evaluation runs use Windows, WSL and Docker Desktop.
- **Does not own:**
  - The Linux runner's code and its [design](../design/linux-runner.md) belong to the host variant (lane 1). A change there goes to lane 1 as a request.
  - The Git variant (lane 3, [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)) decides how agents and the host exchange work. This lane asks it what that removes and does not design for it.

## State

- **Decision:** execution runs on Linux and model work keeps working on Windows ([DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution)). Agents and checks run in Linux containers ([DEC-037](../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers)).
- **CI:** Linux only, started by manual dispatch during development ([CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change)).
- **Apply from Windows:** only through the [Linux runner](../design/linux-runner.md) and Docker Desktop; native `apply` on Windows is refused since L2. The runner's checkout mount and bundle hand-back are the current implementation until slice G1 of the [Git variant](../implementation-plan.md#git-remote-collaboration-owner-direction-27-october).
- **Evidence so far:**
  - All V1 native evidence before the runner ran with the Apply host on Windows and Linux containers on Docker Desktop with the WSL 2 backend ([provider matrix](../validation/v1-technical-acceptance-20261026.md)).
  - The runner was observed once from Windows, on Main `00c687b6`, with one fixture Library project of six Managers: the mounts, one Apply handed back as a branch, and an early stop ([record](../validation/v1-l1-runner-qualification-20261027.md)).
- **Not observed:**
  - An Apply with the host on Linux, native or in WSL.
  - A native Linux engine; rootless engines are not supported ([path translation](../design/linux-runner.md#path-translation)).
  - Through the runner: a stop during publication or import, a host process that dies without cancelling, a keyboard Ctrl+C, the lock after a stop, a correction pair, a no-change publication, and projects larger than six Managers.
- **Day-to-day practice:** broad Go suites run on Linux, for example in WSL ([CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change)). Evaluation runs have also been run by hand in the WSL distribution `Ubuntu`; how they reach the engine is not documented (session notes), and on 27 October Docker Desktop's WSL integration was off for `Ubuntu`.

## Known problems

| # | Problem | Evidence | Current handling |
|---|---|---|---|
| P1 | Binds through Docker Desktop lose file modes: `0644` and `0755` arrived as `0777` | [Bounded component probes](../validation/v1-qualification-20261024.md#bounded-component-probes) | Agents, checks and the Git service use only named volumes; the runner keeps its state in a volume. Code must never bind a runner path |
| P2 | Path spelling depends on the launcher. From Windows, Docker Desktop accepted `C:/…`, `C:\…` and `/run/desktop/mnt/host/c/…`. From WSL, the engine refused `/mnt/c/…`, probably because the WSL integration is off (not confirmed) | [Q1](../validation/v1-l1-runner-qualification-20261027.md#q1-mounts); [path translation](../design/linux-runner.md#path-translation) | The launcher uses `C:/…` and refuses UNC, `\\wsl$` and relative paths. The WSL form is unproven and no longer pursued ([owner direction](#owner-direction-27-october)) |
| P3 | File ownership: bind sources appear as `0:0` in the runner; agents run as `1000:1000`; the Run's Git service needs `CAP_CHOWN` to set up ownership | [Q1](../validation/v1-l1-runner-qualification-20261027.md#q1-mounts); [container adapter](../../internal/adapters/container/README.md) | Works on Docker Desktop. Ownership and bind behavior on a native Linux engine are unobserved |
| P4 | Shared volumes: one Run volume holds every Manager's repository and each container mounts its subpath (Docker 26 or later); the runner's state volume has no garbage collection and is read only through a container | [Run Git workspace service](../../internal/adapters/gitrepo/README.md#run-git-workspace-service); [runner limits](../design/linux-runner.md#limits-and-open-questions) | Works today. The Git variant may replace the shared Run volume |
| P5 | Volume operations slow down under load: about 20 ms idle, 10 to 40 s while six Agents work, so close bounds grew (Run cleanup is 45 s) | Session notes, 20 October; [requalification limits](../validation/v1-requalification-20261025.md) | Larger bounds |
| P6 | Long runs on the shared Windows machine break: Claude Code reaps idle background shells under memory pressure and orphans an Apply in WSL; a PowerShell 5.1 call dropped inner quotes and a `pkill` killed another session's Apply; other sessions' tests broke a study's quiet runs | Session notes, 20 October; [Phase C incident 7](../validation/v1-comparison-results-phase-c-20261026.md) | Detached starts in WSL with done markers, script files instead of inline commands, own labels and PIDs, and DEC-077's quiet-machine rule |
| P7 | A host process that dies without cancelling leaves the runner running until Apply's deadline, with the state lock held | [Result hand-back](../design/linux-runner.md#result-hand-back) | Find it by label and stop it by hand with a 240 s grace; forced removal can leave Agent containers behind |
| P8 | Inputs the runner cannot see. Git-ignored files that the model accounts are silently absent, so the Apply sees a different baseline than a native one. Linked worktrees, submodules, shared clones and external instruction directories are refused | [Runner limits](../design/linux-runner.md#limits-and-open-questions) | The silent case is documented, not refused |
| P9 | Windows test friction: real Git makes `internal/adapters/gitrepo` take 23 to 40 minutes instead of about two; `internal/testkit.TempDir` fails when `TEMP` is `C:\WINDOWS\TEMP`; Windows path identity and root aliases cost several fixes | [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change); session notes, 24 October; [PR #314](https://github.com/Glacius-Labs/Markitect/pull/314), [PR #315](https://github.com/Glacius-Labs/Markitect/pull/315) | Tests still run on Windows, slowly; broad suites run on Linux |

## Owner direction, 27 October

In the lane's first discussion the owner set two rules:
- **No running from WSL.** It needs too much fiddling; Markitect wants simple solutions.
- **No Linux machine required.** Users cannot be expected to have one.

For a Windows user this leaves one way: install Docker Desktop and run `markitect` on Windows, which starts the Linux runner. Docker Desktop's own use of WSL 2 as its backend is internal to it and needs no setup. The runner's WSL launch form, WSL as a development host and a Docker Engine inside WSL are dropped as options.

## Options

These are directions to discuss, not plans. They are not exclusive.

1. **Docker Desktop as the only Windows requirement.** Keep today's form: Windows launches the runner through Docker Desktop, and Linux users run Apply natively. Leaves a Linux-host Apply, a native Linux engine and P6 open.
2. **A dedicated machine for our own long and counted runs.** A host that only runs evaluations and native controls, so development sessions cannot disturb them (P6). It is for Markitect's own studies, not a user requirement. Costs a machine and moving logins there.
3. **After the Git variant.** If agents and the host exchange work through a Git remote instead of a mounted `.git` and a shared Run volume, the runner's mounts and parts of P1, P4 and P8 may disappear. Whether the host could then run natively on Windows again, still with Docker Desktop for the agents, is a question for the Git variant, recorded below.

## Open owner questions

1. **Long runs.** Should our counted studies and native controls move to a dedicated machine (option 2), or stay on the shared machine under the quiet-machine rule?
2. **Our own Linux test runs.** [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change) suggests running broad Go suites in WSL. Does the no-WSL rule cover this too? If so, the simple replacements are CI only, or the same suites in a Linux container through Docker Desktop.
3. **Linux users.** Native Apply on a Linux host has never been observed. Should it be observed once, and where?

## Dependency on the Git variant

The Git variant's prototype runs where the host variant's Apply runs today. These questions wait until the owner opens this lane's discussion:
- Where does the host run under G0: Linux only, or natively on Windows if it only talks to the engine and a remote?
- Does the shared Run volume with per-Manager subpaths go away?
- Is the local remote a container, a host-side bare repository or a volume?
