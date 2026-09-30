# Operating and releasing Markitect

This document owns the operational contract and release gates. [Usage](usage.md) owns command syntax, [consumer integration](../integration/README.md) owns installation, and the consumer's delivery record owns its acceptance.

## Supported operating model

Markitect is a local Go compiler and authoring toolkit for trusted Git repositories. A consumer pins one release bundle's complete five-file set in its own history. Canonical YAML owns typed mechanisms; declared ordinary files and generated reading views retain their documented responsibilities. Agents choose the relevant entry and review meaning.

The release target boundary is Windows amd64 and Linux amd64; no other OS or architecture is claimed. Git is required for immutable source evidence and install writes. The native release binary can preview or apply an install without Go. The installed source bootstrap requires Go 1.27.1 or newer. Its first build may need to download the selected Go toolchain and checksum-verified module dependency; offline use requires those exact inputs to be provisioned first. A bootstrap cache hit still probes the selected toolchain, and the outer `go run` needs writable `GOCACHE` and `GOTMPDIR`.

The tool executes supported consumer checks with the caller's local authority. Snapshot materialization isolates the input files; it is not a security sandbox for arbitrary repositories. Profile commands are fixed in the adapter. Konfyra retains its existing Python adapter and regressions; the current Cockpit path uses the native Go checker and paired Go tests. When a snapshot includes the Markitect bootstrap and its paired test, `verify` also runs that Go test. A generic Project can use structural checks and context without a repository gate adapter.

The source declares version `0.1.0`. A source version is not evidence of publication or acceptance; verify the exact immutable Release and successful hosted run against the intended source commit. The private repository has immutable Releases enabled. Release CI runs source gates and Windows/Linux bundle smoke tests, then uploads a run-scoped artifact; it does not create a Release. The Actions `GITHUB_TOKEN` lacks admin read access to the immutable-release setting (the live API returned 403), so no PAT is stored in workflow secrets. An authorized owner completes publication locally. Server-side branch rules could not be configured on the current GitHub plan; Markitect's rejection of branch names `main` and `master` does not provide server-side branch protection.

## Work while a review is running

1. Finish the candidate in its worktree, including generated outputs, and commit it. Resolve the commit once for that review.
2. Give the reviewer the context for that exact commit and a bounded question. Declare every required repository input in the graph. Keep external observations separately identified.
3. Continue independent work in another worktree. Changes there do not alter the committed inputs already compiled for the reviewer.
4. After integrating another change, run `impact` from the reviewed commit to the new candidate. Evaluate the recorded review with `review --evidence` using the same executable and review configuration.
5. Reuse only an eligible report for its original question. Review affected meaning whenever inputs change or relevant dependencies were omitted. Record any human decision through the consumer's own workflow.

There is no need to freeze all work during a fixed-context review. Concurrent writes to one checkout still need coordination. Format, render, migration and install use the shared writer lock; their write paths validate plans and report partial writes. Each target replacement is atomic, but a multi-file operation is not a filesystem transaction. Git remains responsible for integration across files and branches.

## Verification and failure handling

`check` returns structural and generated-output findings. `verify --revision COMMIT` first performs that check, materializes the same snapshot and runs its profile gates. The Konfyra adapter includes its existing renderer and Python regressions. If that snapshot has the Go bootstrap, it also runs the corresponding native Go regression suite; a runner without its paired test, or a test without its runner, is incomplete evidence. Gate processes ignore ambient `GOFLAGS`, Go environment-file configuration and workspace redirects, including Go calls made by existing Python checks. The native Go suite runs with `-count=1` so verification executes its tests again.

Each verification process has a ten-minute limit and at most 1 MiB of captured combined output. Timeout or output overflow cancels the direct process; a two-second pipe-wait limit bounds inherited pipe handles. This mechanism does not guarantee termination of every descendant process created by a consumer script. The YAML report retains completed gate results and classifies the failure. A failed gate returns exit 1; unavailable tools, incomplete evidence and execution limits return exit 2. Neither is a passed verification.

| Observation | Action |
|---|---|
| `output-drift` after editing YAML | Run format, Markitect render, the consumer-owned provider renderer if applicable, then checks. Inspect the complete diff. |
| A write refuses `main` or `master` | Continue in the intended non-protected worktree. |
| Source changed during a write | Keep the edits, reload the working snapshot and rerun the complete command. If it already wrote paths, inspect its `written` list and recovery message first. |
| Existing `write.lock` | Establish whether a renderer still owns it. Remove only a confirmed abandoned lock, then rerun. |
| Install preview digest or source commit differs from verified release | Do not use `--write`. Recheck the exact release tag, attested provenance asset, bundle hash and candidate commit. |
| Archive or cached executable integrity failure | Compare the pin with the reviewed consumer commit. Re-provision the exact trusted archive; do not edit its checksum to silence the error. |
| Go cache cannot be written | Set writable `GOCACHE` and `GOTMPDIR` before the outer `go run`, or use a built bootstrap. |
| `verify` says incomplete or tool missing | Install the stated prerequisite or repair the committed integration; rerun the same fixed candidate. |
| Gate timeout or output limit | Inspect its bounded output and the underlying check. Correct the cause before recording new evidence. |
| `review-required` | Read the reason, compile the affected context and perform the required semantic review. |
| Invalid original review record | Recover its original report and source commit, or perform a new review. A record is not repaired by changing hashes. |

Git subprocesses bind to the explicit repository and discard inherited repository/object/config overrides. This prevents a surrounding Git hook or shell from redirecting an immutable read or protected-branch check into another checkout.

## Install, upgrade and rollback

Follow the exact release and install procedure in [consumer integration](../integration/README.md). Verify the immutable release and the attached bundle, platform binary, and provenance assets with GitHub CLI. Take the outer bundle digest from the verified provenance asset and compare it with the downloaded bundle before invoking `install`. Preview the plan and check its version/source commit before adding `--write` on a committed feature branch. After installation, check the consumer and run the paired bootstrap tests before committing the complete five-file set.

An upgrade needs the consumer's structural, generated-view and repository checks at its new fixed commit. A new tool executable invalidates earlier review reuse, even when the content is unchanged. Keep the prior consumer pin commit available. The installer recognizes the complete, committed RC3 four-file pin set for upgrade; it refuses partial or unknown pin sets.

For rollback, revert the integration commit through the consumer's normal branch/review route, restoring `release.yaml`, archive, lock, bootstrap and paired test together. If the upgrade also migrated content, restore or reverse that content migration in the same candidate. Verify the rollback candidate with the restored pin. Existing digest-addressed caches may coexist; deleting every cache is unnecessary. A rollback is a new candidate and requires its own applicable evidence.

The flat tool lock remains intentionally small and is bound into `release.yaml` with the bootstrap, test, and source archive. `package` still produces only the low-level archive and lock; it is not the consumer release bundle. Changing lock or resource schemas requires an explicit migration and tests for old consumers. Provider schemas and models belong to their specific adapters and consumer policy.

## Release gates

Release evidence identifies a full source commit and the exact distribution. A release is eligible only when these gates hold:

| Gate | Required evidence |
|---|---|
| Independent source | Clean, reviewed Git candidate; preserved source history; no hidden consumer checkout dependency. |
| Compiler and authoring | Go tests, vet, build, current generated schemas, canonical executable example and bundled core authoring. |
| Bundle | `bundle` reads the exact full commit; tag matches source version; deterministic result and all manifest/lock/file digests validate. |
| Installation | Windows and Linux amd64 hosted CI install and smoke-test the same bundle through the bootstrap and example. |
| Run artifact | Successful tagged workflow with source gates, Windows/Linux bundle smoke tests, exact four-file artifact and provenance bound to the full source commit and workflow run. |
| Owner publication | Authorized owner tool preflight validates live immutable-release state and byte-compares the authenticated Actions artifact; explicit publish performs draft upload/readback, digest checks, publication and final immutable attestation/asset verification. |
| Input isolation | Regression evidence for fixed snapshots, foreign Git environment, safe controlled writes and bounded verification failures. |
| Cache and upgrade | Archive/executable checks; actual toolchain and build policy bound to cache eligibility; complete pin upgrade and rollback exercise. |
| Practical consumer | Konfyra candidate checks and an independent semantic review of the actual changed authoring route; same and changed-input review decisions exercised. |
| Documentation | Commands, ownership, limits, prerequisites, installation, recovery and acceptance state agree with the implementation. |

A successful Markitect release gate establishes the supported tool boundary. It does not establish a consumer's provider runtime, hosted pipeline, semantic review, server-side branch rules, or human acceptance. Record those individually rather than inferring them from a compiler test. The dated [Konfyra assessment](konfyra-assessment.md) remains historical evidence and its dossier remains the delivery owner.

The run-scoped Actions artifact is a short-lived transport for the verified release files, not the durable consumer distribution; use an accepted immutable Release. GitHub states that immutable Releases lock the published tag and attached assets and create a release attestation. The auto-generated GitHub source archive is not the Markitect bundle; use the attached `markitect-vTAG-bundle.zip` and `verify-asset`. The provenance YAML is a self-declared digest/run record; the separate GitHub immutable-release attestation is authoritative for the published tag, commit and assets. See [GitHub's immutable release guidance](https://docs.github.com/en/code-security/concepts/supply-chain-security/immutable-releases) and [release verification instructions](https://docs.github.com/en/code-security/how-tos/secure-your-supply-chain/secure-your-dependencies/verify-release-integrity). The API-domain placeholder and public distribution license remain separate decisions. Quantified time or token savings require the controlled observations in [measurement](measurement.md).

The owner publisher fails closed if a release or draft already exists for the tag. It never resumes uploads or clobbers existing assets. If publication fails after creating a draft, inspect it manually: resolve its tag to a commit and compare it with the intended source; check exact asset names, GitHub API SHA-256 values and provenance against the completed workflow run. If anything is incomplete or mismatched, stop and have the authorized owner choose a reviewed correction or discard path. A published immutable Release is never replaced or retried.

Use [the owner publication procedure](../integration/README.md#owner-publication) for the exact source-checkout, `gh auth` and `go run ./cmd/markitect-release` commands. The workflow's `GITHUB_TOKEN` cannot read the admin-only immutable-release setting (the live API returned 403), so the owner runs a read-only preflight and then explicitly selects `--publish`; no PAT secret is needed.
