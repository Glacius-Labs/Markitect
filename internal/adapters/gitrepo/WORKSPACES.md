# Runtime workspaces

`NewWorkspaces` implements `run.Workspaces` through an injected container engine
and a trusted helper `container.ContainerSpec`. The helper image is an exact
engine image ID. Its configured user, CPU, memory, process and temporary-space
limits apply to every helper. The adapter supplies a single `/work` volume mount;
harvest mounts it read-only. The runtime validates the complete container spec.
No project process, provider, credential or host repository path is involved.

The trusted runtime image must contain `/bin/tar` and `/usr/local/bin/node`, as
the current runtime image does. A check adapter can receive this implementation
through `run.Workspaces`; its actual check image remains a separate choice.

## Materialization and ownership

`Materialize` creates a new volume through `CreateVolumeExclusive`, writes the
supplied tree as a tar stream, then harvests and compares exact paths, content
and regular/executable mode. It returns the observed tree identity only after a
complete readback. A different complete readback is retained and reported as
uncertain. Inputs that contain excluded Git administration or state descendants
are refused before creation, since those bytes cannot enter the repository
census. A regular file named `.markitect/state` remains observable.
Before extraction it observes the new volume as empty, since the engine can
initialize a new volume with the helper image's `/work` files. Unexpected initial
content is retained and refused before any extraction.

The volume name derives from the configured helper name and WorkspaceID. Each
creation carries a random ownership label. The engine verifies its own fresh
creation token as well. Existing IDs are refused before any helper writes.
Only a matching observed creation receipt permits later harvest or removal.
Receipts belong to the adapter instance; a reconstructed instance refuses an
existing volume. Retained volumes therefore need the original instance or
separately authorized recovery. No receipt is inferred from a failed inspect.

Errors never cause automatic deletion. A failed write, incomplete readback,
mount census failure or changed ownership leaves the volume retained and reports
uncertainty. The returned volume identity permits the Host to retain the actual
effect even when the tree could not be established.

## Harvest

Before and after the helper, the engine observes all containers mounting the
volume, including stopped containers and read-only mounts. Any mount or failed
census makes the result uncertain. The Host must end the step's container before
harvest and must not start another user of that volume during observation.
These observations do not provide an engine-wide lock against unrelated clients.

The Linux helper walks with bounded directory buffers and counts directories,
regular files and unsupported entries against `MaxEntries`. It bounds total
regular-file bytes by `MaxBytes`. Empty directories have no snapshot entry.
Regular files use exactly the owner execute bit for their snapshot mode; other
permission bits are ignored. File identity, mode, size and timestamps are checked
around each no-follow read. The stream carries raw path bytes, metadata and exact
content, followed by an explicit terminal frame. Truncation, helper failure,
budget exhaustion, duplicate paths or malformed output makes it uncertain.

The exclusions match repository acquisition: every `.git` path component and
the root `.markitect/state` directory with its descendants. No Gitignore or
semantic ownership rule filters files. Symlinks, devices, sockets and other
nonregular entries are reported in `Unsupported`; their targets are never read.
Invalid snapshot paths are also reported. Invalid UTF-8 names use `bytes:` plus
their exact hexadecimal path bytes. Unsupported entries do not stop the census
or disappear from the result. The Host decides what the observation means.

`Peek` takes the same census while the step may still write: it skips the two
unmounted-volume checks and keeps the ownership and existence checks, and the
helper still mounts the volume read-only. Its tree is advisory; only `Harvest`
after the step's container stopped can yield a candidate.

## Close and evidence

`Close` reports an observed absent volume as closed, including repeat calls.
It removes an existing volume only after its ownership receipt and an empty
all-container mount census are established. It reads back absence after removal.
A failed removal response remains an error and uncertain even if absence was
subsequently established. An existing unowned or replaced volume is retained.

The source tests exercise exact tar data, mode and readback, unsupported census
entries, budgets, mounted and replaced volumes, partial effects and close
observations through fake runtime transports. They establish adapter behavior;
they do not establish real Docker, helper-image, provider, Apply or release
acceptance. The constructor is available for Host injection; this slice adds no
bootstrap or CLI wiring.
