# Linux runner for Apply

Slice L1 of the [Linux Docker way](../implementation-plan.md#linux-docker-way-owner-direction-26-october) ([DEC-077](../concepts/register.md#dec-077-development-pace-and-linux-first-execution)): the Apply host runs in one Linux runner image, started from the owner's machine through the container engine, for example Docker Desktop on Windows. The owner [accepted](../concepts/sources/development-pace-20261026.md#6-repository-and-login-in-the-apply-container) that the runner clones from a read-only mount of the checkout and hands the verified result back as a branch, and that the provider CLI login is mounted read-only, with no API keys. In this design the read-only mount is the checkout's `.git` directory, never its working tree. This page is the coordinator's design and its increment 1 in source: [`internal/adapters/runner`](../../internal/adapters/runner/layout.go), [`cmd/markitect-runner`](../../cmd/markitect-runner/main.go) and the [image recipe](../../packaging/runner/Dockerfile). Native qualification is a separate, dispatched step; see [limits](#limits-and-open-questions).

## Flow

```text
owner host (Windows or WSL)                 runner container (Linux)
markitect apply --runner IMAGE_ID ...
  check the checkout, translate inputs
  docker run IMAGE_ID --base COMMIT  ----->  git clone the read-only .git at COMMIT
                                            markitect apply against the clone
                                              Agents, Git service, checks: sibling
                                              containers through the engine socket
                                              publication into the clone
                                            commit the published paths, write a bundle
  verify and fetch the bundle        <-----  hand-back area: report, result, bundle
  create the result branch
```

Apply inside the runner is the unchanged product: the same preflight, Runs, Verify approval and guarded publication. Publication writes into the runner's clone, never into the owner's checkout.

## Runner image

[`packaging/runner/Dockerfile`](../../packaging/runner/Dockerfile) builds `markitect` and `markitect-runner` from source in the pinned Go image, copies the static Docker CLI from a pinned `docker:cli` image and installs Git on a pinned `debian:trixie-slim`; it needs no CA certificates, because the runner has no network. The entrypoint is `markitect-runner --base COMMIT [--model PATH] [--dependency PATH]...`. The build passes `MARKITECT_REVISION`, which the linker injects as the build revision, so the runner's build identity is `<version>+<revision>`; `/opt/markitect/runner.env` records it with the CLI and Git versions. The image holds no credentials and no repository content.

The runner runs as root. It receives the engine's socket, which is host-level control of the engine, so it is trusted host code like the native Apply host; no Agent runs in it. Agents keep running in the hardened, non-root sibling containers of the [container adapter](../../internal/adapters/container/README.md). The runner itself has no network (`--network none`): it reaches the engine through the socket, and the Agent containers get their network from the runtime profile.

## Mounts

| Target in the runner | Source | Mode | Purpose |
|---|---|---|---|
| `/markitect/source.git` | the checkout's `.git` directory | read-only bind | the clone source |
| `/markitect/invocation` | `invocation/` of the launch directory | read-only bind | the translated run configuration and runtime profile |
| `/markitect/credentials/<n>.json` | each dedicated login of the run configuration | read-only bind of the file | the provider CLI login, never an API key |
| `/var/run/docker.sock` | the engine's socket | bind | Apply's engine CLI |
| `/markitect/output` | `result/` of the launch directory | writable bind | the hand-back area |
| `/markitect/state` | volume `markitect-state-<repositoryIdentity>` | named volume | Apply state and evidence |

Only the checkout's `.git` directory is mounted, never its working tree: untracked and ignored files such as `.env` are not visible to the runner. The runner clones committed objects and refs from that directory, and its clone gets a fresh configuration; the runner can read the Git directory's own configuration, such as remote URLs, but never prints it or copies it.

Each login file is mounted on its own, never its directory. Before the launch, the host observes every login as a native Apply reads it: a regular, unredirected file of at most 1 MiB, with errors that name neither its path nor its content. A login path that the engine's mount syntax cannot carry is refused the same way. Logins stay outside the clone, so they cannot enter a candidate, and the runner never prints them; an engine error can still name a mount path, never a login's content.

## State and evidence

Apply state lives in the named volume `markitect-state-<repositoryIdentity>`: the lock, `applies/`, `runs/`, `candidates/`, `notes/`, `transcripts/` and `temporary/` as in a native state directory ([architecture](../architecture.md#state-and-evidence-storage)). The output mount is not used for state: a bounded probe showed that host binds through Docker Desktop do not preserve executable modes ([record](../validation/v1-qualification-20261024.md#bounded-component-probes)), and the candidate store and publication run Git in `temporary/`. The volume survives launches, so the lock serializes concurrent runners of one project and evidence accumulates as in a native state directory. Read it through any container that mounts the volume, for example `docker run --rm --entrypoint tar --mount type=volume,source=markitect-state-PROJECT,target=/s,readonly RUNNER_IMAGE_ID -C /s -c . > state.tar`.

On the host, each launch gets `<stateDirectory>/<repositoryIdentity>/runner/<launch>/` under the run configuration's state directory:

- `invocation/run.json` and `invocation/runtime-profile.yaml`: the translated inputs, which hold login paths inside the runner, not login contents;
- `result/apply-report.json`: the runner's Apply report, also embedded in the printed report;
- `result/result.json`: the runner's hand-back record;
- `result/result.bundle`: the verified change, when there is one;
- `runner.log`: the runner's diagnostics, also streamed to standard error;
- `launch.json`: the host report as printed.

## Sibling containers and workspaces

Apply starts the Agent containers, the Git service and the check containers through the socket. The engine creates them on its own host, next to the runner, so a bind-mount source would be resolved on that host, not inside the runner. The current [container adapter](../../internal/adapters/container/spec.go) never creates a bind mount: every mount is `type=volume`, optionally with a volume subpath; bytes enter through exec streams and leave through `docker cp ... -`. The [Git workspaces](../../internal/adapters/gitrepo/WORKSPACES.md) already live on private Linux volumes. No path from inside the runner therefore reaches the engine, and sibling containers work unchanged. Code that Apply runs must keep it that way: no bind mount of a runner path.

## Result hand-back

The runner returns a Git bundle in the hand-back area, and the launcher fetches it into the owner's repository as a new branch. This keeps the checkout mount read-only, as the owner accepted, and needs no remote, push credential or writable mount of the checkout. A bundle is self-contained and verified by Git: it names its base as a prerequisite, keeps file modes and binary content, and is the same mechanism as the stored candidates. A patch file would lose modes and binaries and need a second apply step; a writable checkout mount to push into would contradict the accepted read-only mount; a bare inbox repository in the output area would add state for the same result.

Inside the runner:

1. The entrypoint clones the read-only Git directory through Git's own file transport, without hardlinks, checks out the launcher's observed base commit on the branch `markitect/runner` and removes the remote. A moved checkout fails the clone.
2. `markitect apply` runs against the clone with the translated inputs. A stop request reaches Apply as SIGTERM, so it closes its containers and keeps its evidence.
3. Only when Apply passed with a read-back publication, the entrypoint stages exactly the report's `completedPaths`, refuses any other change in the clone apart from Apply's `.markitect/state/` administration, commits with the fixed identity `Markitect Runner <runner@markitect.invalid>`, checks that the commit has the single parent base and changes exactly those paths, and bundles `markitect/runner` without base. A no-change publication returns no bundle.

On the host, the launcher verifies the bundle and that it holds exactly the reported commit, fetches it with object checks and without FETCH_HEAD, tags, submodules or automatic maintenance (`--no-auto-maintenance`, Git 2.29 like `--no-write-fetch-head`), repeats the parent and path check and creates the branch only if it does not exist (`update-ref` `create`). The branch defaults to `markitect/apply-<apply id>`; `--result-branch` names another. A stop request that arrives after the runner handed back a verified change does not cancel this short import; a failed import names the bundle, which stays in the launch directory. The owner's working tree, index and current branch never change: the launcher's Git takes no optional locks, and the import adds only the change's objects and the new branch. Apply still creates no owner commit: the runner's commit exists only on the new branch, which the owner reviews and merges like any other.

## Inputs

The owner's files keep their native meaning; the launcher derives the runner's copies:

- **Run configuration.** The engine executable starts the runner, the state directory receives the launch directory and each dedicated login is mounted. The runner's copy names `/markitect/state`, the runner's engine CLI and `/markitect/credentials/<n>.json`, with logins numbered in reference order. Limits, deadlines, the project identity and `productBuild` stay unchanged. `productBuild` names the runner's build, which the runner checks against its own binary; the host binary's build is not checked.
- **Runtime profile.** Every `engineExecutable` must name the run configuration's engine, as Apply already requires, and becomes the runner's engine CLI. Nothing else changes. Instruction boundaries must be repository-relative; anchors and aliases are refused.
- **Model selection.** `--model` and `--dependency` must be repository-relative.
- **Checkout.** The runner applies committed history only and fails closed on anything a clone would drop: the launcher refuses uncommitted changes to tracked files and untracked files that Git does not ignore, names their count and an example, and says "commit or remove them; runner mode applies committed history only". Nothing is dropped silently. Git-ignored files are not refused and do not reach the runner (see [limits](#limits-and-open-questions)). The checkout must be the main working tree of a repository on a named branch, with its own objects (no `objects/info/alternates`).
- **Branches.** The owner's branch may be any named branch, `main` included: the runner writes only its clone and a new branch, never the owner's branch. A result branch name that already exists is refused, before the launch and again at the import.

The launch is bounded by the run configuration's deadline plus ten minutes for clone, hand-back and stop. Cancellation asks the engine to stop the runner container, not only its CLI, with a four-minute grace period that covers Apply's own cleanup and persistence bounds; if the stop cannot be requested or fails, the launcher asks the engine to remove the container.

A host process that dies without cancelling, for example when it is killed or the machine shuts down, leaves the runner running until Apply's own deadline ends it, with the state lock held. Find it with `docker ps --filter label=markitect.runner`. Stop it with `docker stop --time 240 markitect-runner-<launch>`, so that Apply closes its containers and keeps its evidence; `docker rm --force markitect-runner-<launch>` is the last resort, after which Apply's sibling containers (label `markitect.store`) can remain.

## Path translation

The launcher spells host paths for the engine CLI it runs ([`HostPath`](../../internal/adapters/runner/mounts.go)):

- **Linux and WSL:** a clean absolute path passes unchanged. A WSL launcher names a Windows checkout as `/mnt/c/...`.
- **Windows with Docker Desktop:** a drive path becomes Docker Desktop's forward-slash drive form, `C:\Users\me\repo` becoming `C:/Users/me/repo`; Docker Desktop maps it into its Linux VM itself. The VM-side spelling (`/run/desktop/mnt/host/c/...` with the WSL 2 backend) depends on the backend and is never passed. UNC, device, WSL (`\\wsl$\...`), drive-relative and relative paths are refused; a checkout inside a WSL distribution is launched from WSL.
- **Every platform:** characters that would change the meaning of the comma-separated `--mount` value are refused. The engine socket is the engine-side `/var/run/docker.sock`, which Docker Desktop and local engines provide.

The engine must run on the owner's machine, because bind sources and the socket mount resolve on the engine's host. The launcher refuses a set `DOCKER_HOST` and a current Docker context (including one selected by `DOCKER_CONTEXT`) whose endpoint is not a local socket or named pipe (`unix://`, `npipe://`), such as `tcp://` or `ssh://`; Docker Desktop's `desktop-linux` context is local. Rootless engines with another socket path are not supported in increment 1.

## What stays on Windows

The model commands `check`, `context` and `docs` keep running natively, and so does model editing. On Windows, `markitect apply --runner` validates the inputs, starts the runner through Docker Desktop and imports the result with the host's Git. The native Windows Apply host stays until slice L2 removes it.

## Limits and open questions

- **Native qualification is open.** Source tests use a fake engine and a fake Apply with real Git. Not yet observed: the image build, Docker Desktop's acceptance of the forward-slash drive form and of the socket mount from `docker.exe`, socket access as root on Docker Desktop and a native engine, stop on Ctrl+C, and a complete Apply through the runner with a real login.
- **Git-ignored files the model accounts.** A native Apply captures Git-ignored files that the project accounts, through an exact ignore entry or a registration, as part of its baseline. The clone holds committed files only, so these files are absent in the runner, and an Apply that depends on them sees a different baseline than a native Apply. Uncommitted changes and untracked files that Git does not ignore are refused instead.
- **Linked worktrees, submodules and shared clones** are refused, because their `.git` file or object alternates name host paths that do not exist in the runner.
- **External instruction directories** and external manifests would need further read-only mounts.
- **The state volume** has no garbage collection, and its evidence is read through a container.
- **Apply in CI** remains open under DEC-077.
