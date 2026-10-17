# Check adapter

`MetadataPreparer` observes the existing closed static Check profile format. It
does not run the executable or establish runtime capability.

`NewRunner(engine, workspaces)` implements `run.Checks` with the container engine
and a separately configured `run.Workspaces` adapter. The application supplies
the concrete workspace adapter; its helper image is independent of the exact
check image in `CheckRequest`. This package imports no repository adapter and
adds no bootstrap wiring.

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
