# Source snapshots

This page records the source snapshot boundary shipped in [v0.8.1](https://github.com/Glacius-Labs/Markitect/releases/tag/v0.8.1). The public CLI remains Git-oriented. Verified release evidence is in the [production assessment](production-assessment.md).

Current source owners follow the [Module architecture](development/modules.md); v0.8.1 is the historical behavior baseline, not the final package layout. Identity/digest semantics are unchanged.

## Current value

`internal/core/snapshot` owns a concrete resolved `Snapshot` value:

```go
type Snapshot struct {
    ID          string
    Provisional bool
    Files       map[string][]byte
    Modes       map[string]string
}
```

Paths are slash-separated project-relative names. `Modes` uses the existing regular-file tokens `100644` and `100755`; these preserve executable-bit differences during comparison and materialization. `Provisional` says that the value represents mutable, not-yet-fixed input. It does not identify the adapter that supplied the value.

`Snapshot.Digest()` is a compatibility identity for content. It keeps the existing algorithm: sort paths, then hash each path, mode token, and file bytes using the existing length framing. It deliberately excludes `ID` and `Provisional`. Context and review evidence already depend on this digest, so changing its inputs or encoding requires a separate compatibility decision.

The deterministic `Compare` operation consumes two snapshots and returns a `ChangeSet` for added, modified, and removed paths. A modification includes a byte or mode change. The result is sorted deterministically. `internal/host.Parse`, context compilation, and impact analysis consume the resolved values; impact uses the changed paths and old/new dependency closures. They do not run Git to decide graph or impact semantics. `internal/core` contains no production Git types or command execution.

## Adapter and use-case ownership

| Concern | Current owner | Contract |
|---|---|---|
| Resolve a revision, read a commit tree and blobs, or capture a working directory | `internal/infrastructure/source` | Git remains the fixed-revision acquisition adapter. The directory reader also captures provisional working-tree input, including outside Git. Git process environment is cleaned, the repository is selected explicitly, and path, mode, symlink, and resource limits are checked. |
| Store and compare resolved state | `internal/core/snapshot` | Deterministic values and comparison operate on paths, mode tokens, bytes, and opaque IDs. No Git subprocess is needed. |
| Parse resources, compile context, and calculate impact | `internal/host` plus `internal/core` | These use cases consume resolved values. They retain conservative invalidation for changed inputs whose ownership is unknown. |
| Materialize a selected snapshot | `internal/infrastructure/source` filesystem materializer | It writes the supplied snapshot value into a filesystem tree; it does not resolve a revision or consult Git. Verification runs declared commands against that materialized tree. Materialization is not an operating-system sandbox. |
| Protect repository writes | Write use cases and the Git source adapter | Operations such as initialization, installation, formatting, and rendering retain the branch, HEAD, index, and worktree checks required by their safety contract. Those checks are operational policy, not resource-graph semantics. |
| Create release bundles | `internal/tooling/release` and the release CLI | Source bundle provenance and validation remain tied to a fixed Git source commit. This is distribution policy, not a reason to put Git in the resource model. |
| Pin an offline content package | `internal/host.PackContent` and its CLI caller | The pack operation receives provenance explicitly. The Git CLI supplies `git:<full-commit-id>`, preserving the existing Project pin format and provenance. Archive SHA-256 remains a separate content integrity check. |
| Select CLI input | `internal/host/cli` behind thin `cmd/markitect` | Existing `--revision` and `--base` behavior stays intact: Git accepts revision selectors such as branches and `HEAD`, then the adapter records the resolved full commit ID. The fixed-run manifest and stored review-evidence workflows require full Git commit IDs. No new selector syntax is added. |

Fixed Git acquisition is rooted at the selected Git repository root. Supplying a nested directory with a repository-wide revision does not rebase that commit tree into a nested Project. For an independently verified example or subproject, use a separate explicit Git repository/snapshot; do not silently substitute its configuration for the root Project.

## Identity and compatibility

Snapshot content identity and source identity answer different questions. The digest identifies the exact paths, regular-file modes, and bytes used by the current algorithm. `ID` is an opaque identity supplied by the source, while `Provisional` separately records whether the input is fixed. They are independent fields: the live working-tree reader currently leaves `ID` empty, while the initialization preview uses a provisional snapshot with an ID. A nonempty fixed opaque ID is sufficient for application review logic to require that recorded evidence matches the selected fixed state.

The CLI still validates full Git commit evidence before resolving its fixed input. Public YAML keeps the existing `revision` key, and existing Git consumers continue to see full commit IDs there. The v0.8.1 release does not change the review record schema version. This preserves current consumer behavior while allowing application code to use an opaque fixed ID internally; it does not promise that another provider can be selected through the current CLI.

`PackContent` requires provenance from its caller instead of inventing an origin from the snapshot value. The existing Git command passes `git:<full-commit-id>`, so emitted package pins and the public Project format remain unchanged. The snapshot adapter boundary does not change the package archive or its SHA-256 pin.

The materialized directory is an execution input, not an identity source. A verification report remains bound to the snapshot selected before materialization, so commands cannot silently switch the evidence basis by changing the temporary tree.

## Original v0.8.1 audit (historical)

| Classification | Current findings |
|---|---|
| Already generic | `internal/core` has no Git domain concepts. Context input hashing and most graph, parsing, and render logic operate on explicit values. `internal/app.Changes` already compares old/new file bytes and modes; its identity labels come from the selected snapshots. |
| Small refactoring | Move the snapshot value and deterministic changed-path comparison out of the Git-named `internal/source` package. Keep the existing digest encoding and public result fields. |
| Useful boundary now | Have acquisition return the concrete resolved value, have app use cases consume it, and make materialization consume the value directly. Pass package provenance explicitly. Keep write-safety Git operations at their use-case boundary. |
| Defer until a real consumer | A directory or archive acquisition adapter, public non-Git selectors, an adapter registry, a generic repository interface, and a separate interface for every acquisition step. A future in-memory fixture can exercise the value boundary without adding a production adapter. |
| Do not abstract | Resource identity, graph ownership, dependency edges, rendering semantics, provider configuration, or project policy into Git-aware concepts. Do not infer dependencies from document links. Do not weaken conservative impact or review invalidation. |

The historical audit above retains its inspected package names. Current-source tests exercise the same behavior without adding a second production provider. `internal/core/snapshot/snapshot_test.go` checks the legacy digest and deterministic comparison; `internal/host/snapshot_neutrality_test.go` parses caller-owned opaque snapshots, compiles context, and checks both dependency-scoped impact and conservative invalidation for unknown files. Production Git subprocesses remain owned by `internal/infrastructure/source`, while Host runtime/use-case calls request Git for revision selection and repository-safe writes. Impact compares resolved values without commit semantics. Git package provenance, full commit validation for fixed-run and stored review evidence, fixed verification inputs, and repository write guards remain explicit contracts.

## Security and verification limits

Keeping Git in an adapter does not remove the need for its safeguards. Preserve environment cleanup for inherited `GIT_*` variables, explicit repository selection, option termination for user revisions, replacement-ref handling, bounded tree/blob reads, supported regular-file mode checks, safe path validation, and rejection of symlinks and submodules. Keep write-path reparse checks, branch checks, exclusive creation, and index/worktree preflights wherever their existing operations require them.

Snapshot materialization fixes the bytes supplied to verification. It does not sandbox commands declared by a Project: they execute with the caller's local authority. Snapshot identity and advisory review evidence do not establish that a report is truthful, complete, or human-approved.
