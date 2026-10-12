# Operations

This document describes current model-first development and release operations. [Project operations](project-operations.md) is the CLI and runtime reference; the repository [distribution guide](../integration/README.md) owns packaging and publication. Dated evidence remains bound to its recorded source and environment in the [production assessment](production-assessment.md).

## Authoring and checking

Use the [Project workflow](project-workflow.md) for ordinary changes and Brownfield adoption. Keep canonical intent in `.markitect/model/`; review generated readable views rather than editing them. A committed model and its digest bind bytes but do not authenticate human approval. Check the selected revision, project coverage and Manager context before planning work. Implementation-only changes do not need an artificial model edit.

The repository's full-coverage project check accounts for source artifacts. Separate module and architecture checks enforce implementation ownership and import boundaries. The current contributor gate list and fixed-revision command forms are maintained in [CONTRIBUTING.md](../CONTRIBUTING.md#verify-a-change). They are source gates, not adopting-project acceptance.

## Rendering and controlled writes

`docs` previews the readable project view. Write only after reviewing the preview and supply the returned digest with `--expect`; generated files are not edited directly. Model changes and generated output must be reviewed together. Other mutations use the operation-specific preview, authorization and digest bindings described in [Project operations](project-operations.md).

Markitect executes with the local caller's authority; project workspaces are not an operating-system sandbox. Guarded Apply checks the captured repository identity and bytes before writing. Multi-file operations can report partial failure and are not filesystem transactions. Preserve the operation's written-path list and recovery guidance when a write stops part-way through.

## Troubleshooting

| Observation | Response |
|---|---|
| Project check reports uncovered paths | Assign each path to the responsible Manager Artifact or add a justified ignore entry, then check the fixed candidate again. |
| A preview digest is stale | Re-read the current model and inputs, preview again and use only the returned digest. |
| A write refuses changed repository state | Inspect the affected paths and repository identity, then create a fresh preview against the intended candidate. |
| A guarded write reports partial completion | Preserve its path report, inspect each path, and recover explicitly before retrying. |
| A gate fails or times out | Inspect bounded output and the exact command; repair the cause and verify a new candidate. |
| Archive or executable integrity fails | Compare against the selected immutable release and provision exact verified bytes. |
| Review evidence is ineligible | Read the diagnostic and assess the new fixed context; do not alter old evidence hashes. |

## Release operations

The manually selected historical [benchmark workflow](../benchmark/README.md) consumes attested binaries from immutable releases using a fixture that matches each release's Project and output contract. It compares only shared scenarios, uploads raw data and summary artifacts, and never modifies a published release. Its timing variation is diagnostic, not a publication gate. The published-release trigger is retired because the v1/v2 workload requires historical Project/Domain commands and cannot measure a model-first release; existing records remain evidence for their original inputs.

Current source distribution preparation uses the independent `markitect-release package`, `bundle` and `install` commands described in [integration](../integration/README.md#current-source-release-tooling). These local operations do not publish a release; installation writes require `--write`.

A source version string does not prove a release exists. Release only a clean reviewed full commit after standalone tests, vet, build, module and architecture checks, documentation validation, and Windows/Linux gates pass. Bundle contents, provenance, tag, source commit, workflow run, and asset digests must agree. The immutable GitHub release is the durable distribution; a workflow artifact is only temporary transport.

Publication uses the authenticated release owner's local GitHub CLI session and a fail-closed tool. CI does not store a personal access token or publish with a token that lacks permission to read the immutable-release setting. The owner reviews the read-only plan before choosing publication. Verify the resulting immutable release and every attached asset. Never overwrite a tag or release. Recovery from a failed draft publication is an explicit owner decision after checking the tag target, run artifact, asset set, and digests.

After a new release is published and verified, update the README's pinned CLI installation examples to its exact tag and attested Windows/Linux SHA-256 digests. Check the public downloads and platform-specific version output before calling that version the current quick-install choice. Never put an unpublished tag or guessed digest in the README.

Use the read-only distribution check to confirm that the marked README installation and example blocks, plus the versioned canonical WinGet manifests, match a selected public release:

```powershell
go run ./src/cmd/markitect-release distribution --tag vX.Y.Z --repo .
```

The command loads only a stable, published immutable release. It verifies the release and each asset attestation, downloads the exact four assets, checks the provenance against the successful release workflow attempt and tag commit, and matches every asset digest. It then checks the README markers and generated WinGet manifest files. `--write` applies those local files after validation; `--export-winget PATH` writes just the three manifests into `PATH/manifests/g/GlaciusLabs/Markitect/VERSION` for review and submission in a separate WinGet community PR. The modes are mutually exclusive. The command does not submit a package or publish a release.

Before a WinGet submission, run `winget validate --manifest` against the exported version directory and review the diff. Verify the package install, upgrade, and uninstall from the public catalog after the community PR is merged.

The release gate proves only the Markitect source and packaged product checks that ran. It does not assert that a project has correct policy, complete prose, a safe runtime, or a human decision. Those are owned where the project operates.
