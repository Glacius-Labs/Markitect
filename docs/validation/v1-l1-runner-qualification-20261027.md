# L1 Linux runner: native qualification on Main 00c687b6 — 27 October 2026

This record covers the native qualification of slice L1, the [Linux runner](../design/linux-runner.md) ([PR #773](https://github.com/Glacius-Labs/Markitect/pull/773)), under the [Linux Docker way](../implementation-plan.md#linux-docker-way-owner-direction-26-october). The coordinator dispatched it under the [standing evidence authorization](../concepts/sources/v1-source-implementation-approval-20261023.md#subsequent-standing-evidence-authorization). It ruled that the owner's [quota waiver](../concepts/sources/owner-night-decisions-20261027.md#3-when-the-comparison-runs), given for the comparison gate, still applied.

It has three questions:
- **Q1:** do the runner's mounts work from Windows and from WSL?
- **Q2:** does one bounded Apply through `markitect apply --runner` work on the Library project?
- **Q3:** does a cancellation early in a second launch stop cleanly?

The record is descriptive, with one run each.

## Result

| Question | Verdict |
| --- | --- |
| Q1, mounts from Windows | **Works.** `docker.exe` with Docker Desktop's `desktop-linux` context accepted the runner's complete mount set in the launcher's `C:/…` form, and also in backslash form and as the VM path `/run/desktop/mnt/host/c/…`. The read-only binds refused writes, root in the runner reached the engine socket, the hand-back folder and the state volume were writable, and the runner had no network |
| Q1, mounts from WSL | **Not observed.** Docker Desktop's WSL integration is off for the distribution `Ubuntu` on this host, and `/var/run/docker.sock` does not exist there. The setting was not changed. Docker Desktop's proxy socket in `/mnt/wsl` is root-only, and through it the engine refused the `/mnt/c/…` bind source with "bind source path does not exist". That is consistent with the integration doing the path translation. The WSL launch form needs the integration enabled |
| Q2, one Apply through the runner | **Passed.** The Library M1 Apply passed: Assessment approved, Implement submitted, Verify approved, with 20 messages. The verified change arrived as the new branch `markitect/apply-a615f9c93f6ccd56a726ce6567ea9dc1`. The input checkout's HEAD, branch, index, status and working tree were unchanged. No credential pattern was found ([Q2](#q2-one-apply-through-the-runner)). The runner and Agent containers were gone afterwards, and the state volume was kept |
| Q3, cancellation | **Clean stop at the start of the Assessment.** CTRL_BREAK reached the launcher 64 s into a second launch, 1 to 5 s after the first Agent containers had started. The launcher exited 5 s later with `not_completed`, and by then every runner, Agent and Git-service container was gone. The Apply's evidence was kept, with every session, working copy and the Git service recorded as closed. There was no new branch, and the input checkout and the login were unchanged |

The host Codex account's usage, in its weekly window, went from 94 % to 95 %. No residue and no credential pattern occurred, and the runtime login was unchanged.

The input is a fixture repository the controller prepared, standing in for an owner's checkout.

## Binding

| Item | Value |
| --- | --- |
| Source | Main `00c687b6f6a2035aea69de05b6a7b5db36b557b5`, tree `49ad047ee663b577e21e095c1cceff07e42a5008`. It contains L1 (#773), #767, the publication-guard fix #776 and the R3c cleanup. Fresh clean clone |
| Product binary and agent images | Built 06:31:12–06:32:15Z with `-trimpath` and go1.27.1. SHA-256 `2f58f12cd516ccbf365c492088d37eb255652117bbc98e97e4f53cc55df2b267`, productBuild `0.14.1+00c687b6…`, `vcs.modified=false`. Base `sha256:2825153d039b62b2389ef0faa09e8ec9ea338708d70016188b39cc8f05ba3830`; Go `sha256:0fd0ae932336629e46a6c1f7a78380570307976cd3ba7e46c1a11881164ad5b4`, with `codex-cli 0.162.0` and `agentd` `00c687b6` |
| Runner image | `docker build -f packaging/runner/Dockerfile --build-arg MARKITECT_REVISION=00c687b6… .` from the clean clone, finished at 06:32:46Z. Image `sha256:170abf0308545b4fc5cd2c3950091bfce6b17465d339085767fd135a5d4c3866`. Its `/opt/markitect/runner.env`, captured to `runner.env.txt`, reads productBuild `0.14.1+00c687b6…` (the host binary's build), revision `00c687b6…`, Docker CLI 29.8.2 and Git 2.47.3. The base images are the Dockerfile's digest pins |
| Engine | Docker Desktop, engine 29.9.2, WSL 2 backend, containerd image store, context `desktop-linux` (`npipe://`). `DOCKER_HOST` and `DOCKER_CONTEXT` were unset: by the launch shell for Q2 and by the driver for Q3 |
| Provider | Codex 0.162.0, `gpt-6-sol`, effort `high`, normal account login. The dedicated runtime login was mounted read-only by the launcher, and otherwise only hashed: unchanged before and after Q2 and Q3 |
| Input (Q2, Q3) | A fresh Library M1 project from the source's `testdata/library/m1`, with `testdata/library/vocabulary` and `testdata/core`, prepared by the comparison gate's controller (`controller-gate`): head `317ac4510cc23f1aade29a4f826a0928f62b27e1` on `dev/library-m1`, tree `6d2b71bf`, model digest `sha256:c7051fec…`. 6 Managers (`.`, `catalog`, `lending`, `lending/reminders`, `library`, `members`). Committed and clean, in the main working tree |
| Run configuration | The source's example, format 4: `maxMessages` 128 (the example's standard), `providerIdleTimeoutSeconds` 600, Apply deadline 5,400 s from preparation, 32 Managers, productBuild `0.14.1+00c687b6…`. One attempt each |
| Reservation | `NATIVE-RUN.lock` from 06:31:11Z to 06:52:58Z, released by nonce |

## Q1: mounts

A probe container of the runner image ran with exactly the launcher's arguments (`docker run --rm --init --pull never --network none --security-opt no-new-privileges` and the six mounts of the [design](../design/linux-runner.md#mounts)). It used `/bin/sh` and a probe script instead of the entrypoint. The material was a scratch repository with one executable file and a dummy login file, never the real login.

| Check inside the runner | Windows, `C:/…` (the launcher's form) | Windows, `C:\…` | Windows, `/run/desktop/mnt/host/c/…` |
| --- | --- | --- | --- |
| `docker run` accepted the mounts | yes | yes | yes |
| User | root (`uid=0`) | root | root |
| Clone of `/markitect/source.git` through `file://`, with and without `safe.directory` (bind sources appear owned by `0:0`) | ok | ok | ok |
| Checkout of the base on `markitect/runner`; mode 100755 kept | ok | ok | ok |
| `.git`, invocation and login mounts read-only (writes refused) | yes | yes | yes |
| Login file readable (24 bytes, hash only) | yes | yes | yes |
| Engine socket: a socket, mode 660, `0:0`; `docker version` and `ps` and `volume ls` as root | engine 29.9.2, ok | ok | ok |
| Hand-back folder writable; the file appears on the host | yes | yes | yes |
| State volume writable | yes | yes | yes |
| No network (`deb.debian.org` does not resolve) | yes | yes | yes |

**From WSL** (the `Ubuntu` distribution, Docker CLI present):
- With Docker Desktop's WSL integration off for the distribution, there is no `/var/run/docker.sock` (`q1-wsl-socket.log`).
- The shared proxy socket `/mnt/wsl/docker-desktop/shared-sockets/host-services/docker.proxy.sock` is `root:root` with mode 660. The user `1000` got "permission denied".
- As root through that socket, the CLI reached engine 29.9.2. But `docker run` refused the first bind source `/mnt/c/…/repo/.git` with "bind source path does not exist". That is consistent with the engine not translating WSL's `/mnt/c` paths without the integration. The later mounts, including the socket's, were not reached.

The launcher's WSL form (`/mnt/c/...` unchanged, [path translation](../design/linux-runner.md#path-translation)) was therefore not observed. It needs the integration enabled in Docker Desktop, an owner setting.

## Q2: one Apply through the runner

`markitect apply --repo <input> --model … --dependency … --runtime-profile … --run-config … --runner sha256:170abf03…` ran from Windows. Snapshots before and after the launch covered:
- the input checkout: HEAD, symbolic HEAD, index hash, full status with ignored files, refs and working-tree hashes;
- the login hash;
- the engine's containers and volumes.

| Item | Value |
| --- | --- |
| Launch | 06:37:55Z to 06:49:05Z, about 670 s including clone, Apply, hand-back and import. Runner container `markitect-runner-cbd2bf6869104e69` |
| Siblings | Agent containers (`markitect-native-…`) and the Git service (`git-…`) ran next to the runner through the socket. The transcripts show 6 sessions per Run, 18 in all, and one `docker ps` sample at 06:47:55Z showed 6 Agent containers and the Git service |
| Apply | `a615f9c93f6ccd56a726ce6567ea9dc1`, passed. Runs: assessment approve (6 messages), implement submitted (8), verify approve (6): 20 of 128. `project-tests` on the final candidate, read-back publication, 0 provider retries. 10,997,490 tokens over 18 threads |
| Hand-back | `result.json`: applyStatus passed, base `317ac451`, commit `aba21e73861c081e47ad7c5388d6cbd2b457feb1`, ref `refs/heads/markitect/runner`. The runner logged "Apply passed; commit aba21e73… handed back" |
| Result branch | `refs/heads/markitect/apply-a615f9c93f6ccd56a726ce6567ea9dc1` at `aba21e73`, author `Markitect Runner <runner@markitect.invalid>`. Its single parent is the base. It changes 21 paths, exactly the runner report's `completedPaths`, compared separately from the driver's `post.json` |
| Input checkout | HEAD, branch `dev/library-m1`, index, status and working tree unchanged. The new branch is the only new ref, and no ref was removed |
| Credentials | A pattern scan for token key names (`access_token`, `id_token`, `refresh_token` and their camel-case forms) and the JWT shape. The login's values themselves are never read. There was no match in the launch directory (7 files), the host logs (3), the result branch's checked-out files and commit text, or the extracted state volume (36 files). There was none either in the inflated objects and `log -p` of all ten Git bundles of Q2 and Q3, `result.bundle` and the stored candidates (`bundle-credential-scan.txt`) |
| Cleanup | No container left. The runner was started with `--rm`, and the Agent and Git-service containers were closed by Apply. The only new volume is the state volume `markitect-state-markitect-example`, kept as designed, with `apply.lock` and the project's `applies`, `approvals`, `candidates`, `runs`, `temporary` and `transcripts` |
| Usage | 94 % → 95 % |

## Q3: cancellation

A second launch on the same input, with the same configuration and a fresh deadline, ran in its own process group. When the first Agent container was up, the driver sent `CTRL_BREAK_EVENT`. Go's runtime delivers it, like CTRL_C, as `os.Interrupt`, the path a Ctrl+C in a console takes.

| Time (UTC) | Event |
| --- | --- |
| 06:50:49 | launch |
| 06:51:53 | 4 Agent containers and the Git service up, runner up about a minute; signal sent |
| 06:51:58 | the launcher exited with code 3. Status `not_completed`, error `Apply in the runner did not complete: container engine volume: signal: killed`. The runner logged "Apply not_completed; nothing to hand back" |
| 06:51:59 | no runner, Agent or Git-service container left |

- Apply `236ff93181247f64a66fda0bf66ad13b` kept its evidence in the state volume: `not_completed`, the Assessment Run incomplete with 0 messages, and all 12 cleanup resources (5 sessions, the Manager workers, 5 working copies and the Git service) `closed: true`.
- No result branch was created. The input checkout, the login and the volume set were unchanged, and no credential pattern was found in the launch directory, the logs or the bundles.
- The stop took 5 s, well inside the 240 s grace. Two polls observed it: one at the signal, and the census at 06:51:59. Whether the state lock was released was not checked separately.
- The error names the engine call that the stop interrupted, a `docker volume` command cancelled with its context (`signal: killed`), rather than the cancellation itself. That is a wording note, not a defect.

## Limits

- **The WSL launch form was not observed**, because Docker Desktop's WSL integration is off on this host. Enabling it for `Ubuntu` and repeating Q1 from WSL closes this.
- **Only Docker Desktop.** A native Linux engine, with its own socket ownership and bind semantics, was not tried.
- **Cancellation:**
  - It was a programmatic `CTRL_BREAK_EVENT` to the launcher's process group, not a keyboard Ctrl+C in a console window. Both reach Go as `os.Interrupt`.
  - The stop came at the start of the Assessment. A stop during publication or during the branch import was not tried.
  - Neither was the path in which the host process dies without cancelling.
- **One Apply** on the smallest native case (6 Managers). Larger projects, a correction pair, a no-change publication and a Git-ignored file that the model accounts were not exercised through the runner.
- **The state volume** `markitect-state-markitect-example` stays on the engine with the two Applies' evidence. The design has no garbage collection for it.
