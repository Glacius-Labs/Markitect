# Operations

This document defines product-side development and release operations. [Usage](usage.md) owns command syntax; [Integration](../integration/README.md) describes distribution provisioning; the [production assessment](production-assessment.md) records release evidence.

## Authoring and checking

Use a named feature branch. A new repository can preview `init` before creating its first Project; the write also works on an unborn feature branch. Commit that baseline before subsequent controlled edits. Edit canonical YAML and declared files, format them, render the owned generic and explicitly configured outputs, then run `check`. Commit the complete candidate before collecting fixed-revision `context`, `impact`, or `verify` evidence. A source edit creates a new candidate and its applicable evidence must be refreshed.

`check` validates resource structure and managed-output drift. `verify --revision COMMIT` repeats structural checks on that immutable snapshot and executes only its declared `Project.spec.checks`. A check is an argv array: the executable name must be a bare name resolved using `PATH`, remaining values are literal arguments, and no shell is involved. To run a script, declare its interpreter as the executable. Missing declarations yield `incomplete-evidence`; unavailable commands or execution limits are also incomplete, not successful verification.

After a schema-changing upgrade, an older baseline may no longer parse with the current CLI. In that case, cross-version `impact` and review reuse are unavailable. Keep prior evidence as historical, commit the complete upgraded configuration and content, and establish a new baseline with the current `check`/`verify`, fresh context, and any required new review. Preserve strict parsing; do not manufacture an impact result across incompatible schemas.

Each command is bounded by a ten-minute execution limit and a one MiB captured-output limit, with a two-second pipe-wait limit. Cancellation targets the direct process; it is not a security sandbox and cannot guarantee termination of every child process. Checks use local caller authority. Declare read-only commands scoped to the fixed project snapshot.

## Rendering and controlled writes

`render` checks or writes generic managed views and outputs selected by the Project's explicit target, rule, and provider-adapter configuration. It does not choose provider outputs from repository identity or inspect local conventions to infer a renderer. The adopting project owns its root navigation, hooks, custom formats, and import scripts.

Writers take a shared lock, validate their plan, and check relevant source state. Existing-output writers use atomic replacement for each file; `init` exclusively creates new files so that a concurrent collision is never overwritten. A multi-file write is not a filesystem transaction. Read the command's complete written-path list and recovery guidance after any partial failure. Git is the boundary for reviewing, integrating, and reverting a multi-file candidate. Do not overwrite unmanaged collisions to make a command pass.

## Troubleshooting

| Observation | Response |
|---|---|
| `output-drift` | Run format, render the declared outputs, then check again. Inspect the full diff. |
| Write refuses a protected branch | Continue on a named feature branch. |
| Source changed during a write | Re-read the working snapshot and rerun the full command; inspect paths already written first. |
| An existing `write.lock` blocks a command | Establish that no writer owns it before removing an abandoned lock. |
| `verify` returns `incomplete-evidence` | Check that the fixed Project declares required commands and that each executable is available. |
| Gate fails or times out | Inspect bounded output and the exact command. Repair the cause and verify a new candidate. |
| Archive or executable integrity fails | Compare against the selected immutable release; provision exact verified bytes rather than editing a digest. |
| Review evidence is ineligible | Read the diagnostic and assess the new fixed context; do not alter old evidence hashes. |

## Release operations

The post-publication [benchmark workflow](measurement.md) consumes attested binaries from two immutable releases on one versioned fixture. It uploads raw data and a summary as workflow artifacts and never modifies the published release. Its timing variation is diagnostic, not a publication gate.

A source version string does not prove a release exists. Release only a clean reviewed full commit after standalone tests, vet, build, schema/example checks, and Windows/Linux gates pass. Bundle contents, provenance, tag, source commit, workflow run, and asset digests must agree. The immutable GitHub release is the durable distribution; a workflow artifact is only temporary transport.

Publication uses the authenticated release owner's local GitHub CLI session and a fail-closed tool. CI does not store a personal access token or publish with a token that lacks permission to read the immutable-release setting. The owner reviews the read-only plan before choosing publication. Verify the resulting immutable release and every attached asset. Never overwrite a tag or release. Recovery from a failed draft publication is an explicit owner decision after checking the tag target, run artifact, asset set, and digests.

The release gate proves only the Markitect source and packaged product checks that ran. It does not assert that a project has correct policy, complete prose, a safe runtime, or a human decision. Those are owned where the project operates.
