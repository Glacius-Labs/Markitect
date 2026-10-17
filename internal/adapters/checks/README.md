# Check adapter

`MetadataPreparer` observes the existing closed static Check profile format. It
does not run the executable or establish runtime capability.

`ContainerProfile.Request(check, tree)` projects explicit Host-supplied container
values and an `apply.ExecutionCheck` into the existing `run.CheckRequest`. The
caller selects the profile for `check.Profile` and supplies the actual immutable
tree. The fixed argument prefix precedes literal project arguments; image,
environment, limits, directory, timeout and check identity are retained exactly.
The helper copies argument/environment slices and refuses incomplete or invalid
requests before effects, using the request and runner's existing conditions.
It supplies no defaults and does not inspect the Host environment.

This pure value is not a Host document format or an acquired profile binding.
It does not translate the existing host-executable `process/v1` metadata or
certify an image. A future source-bound Host decoder must select and validate
the explicit configuration before calling this helper. Source tests connect the
projected request to the existing runner with controlled runtime/workspace peers;
they establish no actual container or Apply execution.

`NewRunner(engine, workspaces)` implements `run.Checks` with the container engine
and a separately configured `run.Workspaces` adapter. The application supplies
the concrete workspace adapter; its helper image is independent of the exact
check image in `CheckRequest`. Production code imports no repository adapter.
The optional composition smoke supplies the concrete repository adapter through
the existing constructor; this adds no bootstrap wiring.

Each `Run` allocates a fresh private workspace identity, materializes the exact
request tree and requires matching read-back identity before execution. The
command receives that disposable volume at `/work`, a separately verified empty
disposable volume at `/home/agent` for tool caches, the requested directory,
literal executable/arguments/environment, and the requested CPU, memory and
process limits. It uses the runtime image's non-root user `1000:1000`, no network
and a 64 MiB temporary directory. Image mount-point ownership follows the
[runtime image contract](../../../packaging/runtime/README.md). A check may
change its copy; those changes are discarded and never harvested into a
candidate.

Both verified volumes disable image copy-up on the check container's mounts.
This preserves an empty input tree even when the check image contains files at
`/work`; [Docker otherwise copies them into an empty volume](https://docs.docker.com/engine/storage/volumes/#mounting-a-volume-over-existing-data).

The request timeout bounds materialization and command execution. Runtime
observations supply the actual image, start, exit and combined bounded output.
Nonzero exit remains a known failed check. Lost observation, cancellation or an
unconfirmed image cannot become a passing receipt. Cleanup runs with its own
bounded context after cancellation; failed or uncertain closure is reported as
uncertainty and an error even when the command exited zero.

Source tests exercise the port composition with controlled runtime/workspace
observations. The runtime and workspace packages own their concrete engine and
materialization tests. These tests do not establish real Docker, provider or
Apply acceptance.

## Optional actual Check and Workspace composition smoke

`TestCheckWorkspaceRuntimeSmoke` composes actual `container.Engine`,
`gitrepo.NewWorkspaces` and `checks.NewRunner`. Its observing workspace wrapper
records actual adapter results without replacing operations. Source preparation
and compile-only validation leave this smoke **unexecuted** and establish no
runtime readiness. Executing it requires separate finite Docker authority; it
does not run a provider, read credentials, invoke Apply or mount a host repository.

The finite smoke runs two fully specified Check requests. A passing request
checks exact binary content, executable mode, paths, working directory and
literal argument/environment values, then mutates its disposable work and home.
An empty-tree request verifies that both mounts remain empty despite files in
the check image, emits more than the fixed output bound and exits 23. Assertions
bind each receipt to its request ID, exact image and immutable tree, distinguish
the known nonzero exit from uncertainty, and observe both closed volumes absent
and the check/helper containers removed. Cleanup retries only the concrete
adapter's owned workspace IDs; a failed cleanup is reported with its volume.

Both images must already exist locally as exact `sha256:<64 lowercase hex>`
engine image IDs. No image is built or pulled by the test. Supply the trusted
workspace helper separately from the check image. The helper follows
[the workspace contract](../gitrepo/WORKSPACES.md): `/bin/tar`,
`/usr/local/bin/node`, empty `/work` owned by `1000:1000`, and compatibility with
the configured container limits. The check image needs `/usr/local/bin/node`
and writable mount-point ownership for `1000:1000`.

To make missing `volume-nocopy` observable, the check image must contain the
regular file `.markitect-smoke-image-seed` in **both** `/work` and `/home/agent`,
each containing exactly `image-copy-up-challenge` followed by LF. The test reads
these files from a never-started image container before running the requests.
A separately authorized image build can derive this check image from the
trusted runtime image using this Dockerfile fragment:

```dockerfile
FROM <explicitly-selected-trusted-runtime-image>
USER root
RUN printf 'image-copy-up-challenge\n' > /work/.markitect-smoke-image-seed \
 && printf 'image-copy-up-challenge\n' > /home/agent/.markitect-smoke-image-seed \
 && chown -R 1000:1000 /work /home/agent
USER 1000:1000
```

After obtaining separate runtime authorization, explicitly configure all four
dedicated settings in the process that runs the smoke:

```powershell
$env:MARKITECT_CHECK_WORKSPACE_SMOKE = 'run-check-workspace-smoke'
$env:MARKITECT_CHECK_WORKSPACE_ENGINE = '<absolute engine executable path>'
$env:MARKITECT_CHECK_WORKSPACE_HELPER_IMAGE = 'sha256:<trusted helper image ID>'
$env:MARKITECT_CHECK_WORKSPACE_CHECK_IMAGE = 'sha256:<distinct seeded check image ID>'
go test ./internal/adapters/checks -run '^TestCheckWorkspaceRuntimeSmoke$' -count=1 -timeout=6m -v
```

Ordinary CI skips the smoke. Generic `MARKITECT_TEST_ENGINE` and image settings
cannot activate it. For source-only validation, remove the four dedicated
settings and generic runtime activation variables from that process, then use
`go test ./internal/adapters/checks -run '^$'` to compile without running tests.
Controlled-peer runner tests remain separate source evidence. This smoke, even
when executed successfully, would establish only the bounded Check+Workspace
composition; it would not establish provider, Apply, release or owner acceptance.
