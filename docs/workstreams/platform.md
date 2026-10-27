# Platform (lane 5)

The platform lane covers where and how Markitect runs on Windows and Linux: the Apply host, tests, evaluation runs and the container engine. It is a topic lane: it takes stock and discusses options with the owner, and builds nothing until the owner says so. This page holds the lane's scope, a short state, the known problems with their evidence, options and open owner questions. It links to the pages that own the details.

The lane exists since the owner split Markitect's work into lanes on 27 October 2026.

## Scope

- **Owns:** the questions of where Markitect executes, which engine and host setup it supports, and how development and evaluation runs use Windows, WSL and Docker Desktop.
- **Does not own:**
  - The runner's code (the launcher in `internal/bootstrap`, the image in `packaging/runner/`) and its [design](../design/linux-runner.md) belong to the host variant (lane 1). A change there goes to lane 1 as a request.
  - The Git variant (lane 3, [DEC-078](../concepts/register.md#dec-078-agents-collaborate-through-a-git-remote-without-owner-review)) decides how agents and the host exchange work. This lane asks it what that removes and does not design for it.

## State

- **Decision:** [DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution): Markitect's execution, meaning the Apply host, tests and CI, runs on Linux; model work keeps working on Windows. It replaced [DEC-013](../concepts/register.md#dec-013-linux-first-for-tests-and-the-playground)'s manual Windows job. Agents and checks have run in Linux containers since [DEC-037](../concepts/register.md#dec-037-agent-sessions-and-checks-run-in-linux-containers).
- **CI:** Linux only (`ubuntu-24.04`): quality and race, started by manual dispatch during development. Code that the model commands or the runner launcher reach is checked with `GOOS=windows go vet ./...` by hand ([CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change)).
- **Apply from Windows:** `markitect apply --runner IMAGE_ID` starts the Apply host in one Linux runner container through Docker Desktop, with the checkout's `.git` mounted read-only, the login files read-only, the engine socket, and a hand-back bundle that becomes a new branch ([design](../design/linux-runner.md)). Native `apply` on Windows is refused since L2 ([PR #783](https://github.com/Glacius-Labs/Markitect/pull/783)). The model commands `check`, `context` and `docs` run natively on Windows.
- **Apply on Linux:** native `markitect apply` ([project workflow](../guides/project-workflow.md)).
- **Qualified:** from Windows with Docker Desktop: the image build, the mounts, one Library Apply handed back as a branch, and an early stop ([record](../validation/v1-l1-runner-qualification-20261027.md)).
- **Not observed:** the WSL launch form, a native Linux engine, a stop during publication or import, a host process that dies without cancelling, and larger projects through the runner.
- **Day-to-day practice:** Linux tests and evaluation runs are run by hand in the WSL distribution `Ubuntu` against Docker Desktop's engine. This practice is not documented in the repository.

## Known problems

| # | Problem | Evidence | Current handling |
|---|---|---|---|
| P1 | Binds through Docker Desktop lose file modes: `0644` and `0755` arrived as `0777` | [Bounded component probes](../validation/v1-qualification-20261024.md#bounded-component-probes) | Agents, checks and the Git service use only named volumes; the runner keeps its state in a volume. Code must never bind a runner path |
| P2 | Path spelling differs by launcher: `C:/…` for Docker Desktop on Windows, `/mnt/c/…` from WSL, `/run/desktop/mnt/host/c/…` inside the VM | [Path translation](../design/linux-runner.md#path-translation); [Q1](../validation/v1-l1-runner-qualification-20261027.md#q1-mounts) | The launcher spells the Windows form and refuses UNC, `\\wsl$` and relative paths. From WSL, the engine refused `/mnt/c/…` because Docker Desktop's WSL integration is off on the owner's machine, so the WSL form is unproven |
| P3 | File ownership and UID: bind sources appear as `0:0` in the runner; agents run as `1000:1000`, which is also the WSL user; the Run's Git service needs `CAP_CHOWN` to set up ownership | [Q1](../validation/v1-l1-runner-qualification-20261027.md#q1-mounts); [container adapter](../../internal/adapters/container/README.md) | Works on Docker Desktop. Docker Desktop is known to hide bind ownership problems that a native Linux engine shows; a native engine was never tried |
| P4 | Shared volumes: one Run volume holds every Manager's repository, each container mounts its subpath (Docker 26 or later); the runner's state volume grows without garbage collection and is read only through a container | [Run Git workspace service](../../internal/adapters/gitrepo/README.md#run-git-workspace-service); [runner limits](../design/linux-runner.md#limits-and-open-questions) | Works today. Lane 3 may replace the shared Run volume |
| P5 | Volume operations under load: about 20 ms idle, 10 to 40 s while six Agents work, so close bounds had to grow (Run cleanup is 45 s) | Session notes, 20 October; [requalification limits](../validation/v1-requalification-20261025.md) | Larger bounds |
| P6 | Long runs on the shared Windows machine break: Claude Code reaps idle background shells under memory pressure and orphans an Apply in WSL; a PowerShell 5.1 call dropped inner quotes and a `pkill` killed another session's Apply; other sessions' tests broke a study's quiet runs | Session notes, 20 October; [Phase C incident 7](../validation/v1-comparison-results-phase-c-20261026.md) | Detached starts in WSL with done markers, script files instead of inline commands, own labels and PIDs, the quiet-machine rule of DEC-077 |
| P7 | A host process that dies without cancelling leaves the runner running until Apply's deadline, with the state lock held | [Result hand-back](../design/linux-runner.md#result-hand-back) | Manual `docker stop` by label |
| P8 | Inputs the runner cannot see: Git-ignored files that the model accounts, linked worktrees, submodules, shared clones, external instruction directories | [Runner limits](../design/linux-runner.md#limits-and-open-questions) | Refused or documented as a limit |
| P9 | Windows-specific test friction: real Git makes `internal/adapters/gitrepo` take 23 to 40 minutes on Windows instead of about two; `internal/testkit.TempDir` fails when `TEMP` is `C:\WINDOWS\TEMP`; Windows path identity (8.3 names, case aliases, reparse points) cost several fixes | [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change); session notes, 24 October; [PR #314](https://github.com/Glacius-Labs/Markitect/pull/314), [PR #315](https://github.com/Glacius-Labs/Markitect/pull/315) | Native Windows Apply is gone; tests run on Linux CI and in WSL |

## Options

These are directions to discuss, not plans. They are not exclusive.

1. **Keep today's form.** Windows launches the runner through Docker Desktop; Linux runs Apply natively; tests run in CI and by hand in WSL. Cost: none now. Leaves P2's WSL form, P3's native engine and P6 open.
2. **WSL as the development host.** Enable Docker Desktop's WSL integration for `Ubuntu`, launch Apply and tests from WSL, and document that setup. Closes the WSL form in P2; the engine is still Docker Desktop's VM. Needs the owner's system setting.
3. **A native Linux engine.** Docker Engine inside WSL without Docker Desktop, or a separate Linux machine. Removes the Docker Desktop VM layer, its bind semantics and its path translation, and qualifies P3 on the engine that CI and Linux users have. Costs setup and a second engine on the machine; the runner refuses remote engines today.
4. **A dedicated machine for long and counted runs.** A Linux host that only runs evaluations and native controls, so development sessions cannot disturb them (P6). Costs a machine and moving logins there.
5. **After the Git variant.** If agents and the host exchange work through a Git remote instead of a mounted `.git` and a shared Run volume, the runner's mounts and part of P1, P4 and P8 may disappear. Whether the host could then run natively on Windows again is a question for lane 3, recorded below.

## Open owner questions

1. **Docker Desktop's WSL integration.** May it be enabled for `Ubuntu`, so the runner's WSL form can be qualified (option 2)? It is a system setting that sessions must not change themselves.
2. **Native Linux engine.** Should one be set up and qualified (option 3), and where: inside WSL or on a separate machine?
3. **Long runs.** Should counted studies and native controls move to a dedicated Linux host (option 4), or stay on the shared machine under the quiet-machine rule?
4. **Windows as a development platform.** Is it enough that the model commands and the runner launch work on Windows, or should contributors also be able to run the tests there?

## Dependency on the Git variant

Asked lane 3 on 27 October, answer pending:
- Where does the host run under G0: Linux only, or natively on Windows if it only talks to the engine and a remote?
- Does the shared Run volume with per-Manager subpaths go away?
- Is the local remote a container, a host-side bare repository or a volume?
