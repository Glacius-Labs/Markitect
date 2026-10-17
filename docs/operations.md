# Operations

Current source provides canonical v1 read-only preparation and a guarded generated model view. [Project operations](project-operations.md) is the command reference; [distribution](../integration/README.md) owns separate maintenance packaging and publication. Dated evidence retains its original source/environment scope in the [production assessment](production-assessment.md).

## Authoring and checking

Use the [Project workflow](project-workflow.md). Keep intended behavior in canonical Definitions, exact realization ownership in the registry and operational declarations outside the model. A commit binds a specification revision without authenticating human approval. Check actual structural preparation and the owning Package context before edits. Implementation-only changes need no artificial intent change.

The complete census includes Git-ignored files. Separate architecture and module gates check explicit dependencies, actual v1 registry facts, selected byte pins and literal workflow command scalars. [CONTRIBUTING](../CONTRIBUTING.md#verify-a-change) owns the current gates. These are source checks, not provider or owner acceptance.

## Rendering and controlled writes

`docs` previews the readable canonical view. Use its nested `view.previewDigest` with `--write --expect` after review. `contentDigest` identifies rendered bytes; `canonicalModelDigest` identifies model meaning. The independent documentation guard binds actual sources and target bytes/mode/existence, rejects protected inputs and historical writes, rechecks the plan and reads back actual bytes. Retain completed paths after any partial failure.

Apply, sessions, executable checks, candidate workspaces and artifact publication remain unavailable in the current product. The guarded document write is not Apply. Multi-file/interrupted effects must not be assumed transactional or safe to replay blindly.

Documentation previews may include explicitly selected external sources. For this increment, `docs --write` requires every acquired model, control and instruction source to have an actual repository binding. Otherwise it returns `not_completed` before effects because target disjointness has not been established. Repository-bound dependencies remain writable; an external source is not silently reclassified or followed through an alternate native spelling.

## Troubleshooting

| Observation | Response |
| --- | --- |
| Structural check reports an unaccounted file | Register its exact owner/Definition basis or make an exact explained ignore decision; protected inputs cannot be ignored. |
| Known model/control failure | Read the actual diagnostics and repair the authored input before a new preview. |
| Incomplete acquisition or metadata | Preserve observed facts and diagnose the unavailable bytes, budget or source binding. |
| Preview digest changed | Review a fresh source/target-bound preview; never reconstruct the expected token. |
| Document write reports completed paths and an error | Inspect those actual effects and current sources before deciding recovery. |
| Fixed preview refuses current external sources | Use historically bound inputs or an explicit working preview; do not silently substitute today's bytes. |

## Release operations

The manually selected historical [benchmark workflow](../benchmark/README.md) consumes attested binaries from immutable releases using a fixture that matches each release's Project and output contract. It compares only shared scenarios, uploads raw data and summary artifacts, and never modifies a published release. Its timing variation is diagnostic, not a publication gate. The published-release trigger is retired because the v1/v2 workload requires historical Project/Domain commands and cannot measure a model-first release; existing records remain evidence for their original inputs.

Current source distribution preparation uses the independent `markitect-release package`, `bundle` and `install` commands described in [integration](../integration/README.md#current-source-release-tooling). These local operations do not publish a release; installation writes require `--write`.

A source version string does not prove a release exists. Release only a clean reviewed full commit after standalone tests, vet, build, module and architecture checks, documentation validation, and Windows/Linux gates pass. Bundle contents, provenance, tag, source commit, workflow run, and asset digests must agree. The immutable GitHub release is the durable distribution; a workflow artifact is only temporary transport.

Publication uses the authenticated release owner's local GitHub CLI session and a fail-closed tool. CI does not store a personal access token or publish with a token that lacks permission to read the immutable-release setting. The owner reviews the read-only plan before choosing publication. Verify the resulting immutable release and every attached asset. Never overwrite a tag or release. Recovery from a failed draft publication is an explicit owner decision after checking the tag target, run artifact, asset set, and digests.

After a new release is published and verified, update the README's pinned CLI installation examples to its exact tag and attested Windows/Linux SHA-256 digests. Check the public downloads and platform-specific version output before calling that version the current quick-install choice. Never put an unpublished tag or guessed digest in the README.

Use the read-only distribution check to confirm that the marked README installation and example blocks, plus the versioned canonical WinGet manifests, match a selected public release:

```powershell
go run ./cmd/markitect-release distribution --tag vX.Y.Z --repo .
```

The command loads only a stable, published immutable release. It verifies the release and each asset attestation, downloads the exact four assets, checks the provenance against the successful release workflow attempt and tag commit, and matches every asset digest. It then checks the README markers and generated WinGet manifest files. `--write` applies those local files after validation; `--export-winget PATH` writes just the three manifests into `PATH/manifests/g/GlaciusLabs/Markitect/VERSION` for review and submission in a separate WinGet community PR. The modes are mutually exclusive. The command does not submit a package or publish a release.

Before a WinGet submission, run `winget validate --manifest` against the exported version directory and review the diff. Verify the package install, upgrade, and uninstall from the public catalog after the community PR is merged.

The release gate proves only the Markitect source and packaged product checks that ran. It does not assert that a project has correct policy, complete prose, a safe runtime, or a human decision. Those are owned where the project operates.
