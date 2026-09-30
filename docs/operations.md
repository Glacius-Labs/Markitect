# Operating and releasing Markitect

This document owns the operational contract and release gates. [Usage](usage.md) owns command syntax, [consumer integration](../integration/README.md) owns installation, and the consumer's delivery record owns its acceptance.

## Supported operating model

Markitect is a local Go compiler and authoring toolkit for trusted Git repositories. A consumer pins the tool archive, lock and bootstrap in its own history. Canonical YAML owns typed mechanisms; declared ordinary files and generated reading views retain their documented responsibilities. Agents choose the relevant entry and review meaning.

The currently exercised platforms are Windows amd64 and Linux amd64. Git and Go 1.27.1 or newer are the supported installation prerequisites for immutable evidence and source installation. Existing Konfyra and Cockpit profile checks additionally require Python for their consumer-owned scripts. A first source build may download the Go toolchain and checksum-verified module dependency. An offline machine needs those inputs provisioned beforehand; even a bootstrap cache hit probes the selected toolchain. Other platforms require their own evidence before being advertised as supported.

The tool executes the supported consumer's check scripts with the caller's local authority. Snapshot materialization isolates the input files; it is not a security sandbox for arbitrary repositories. Profile commands are fixed in the adapter. A generic Project can use structural checks and context without a repository gate adapter.

## Work while a review is running

1. Finish the candidate in its worktree, including generated outputs, and commit it. Resolve the commit once for that review.
2. Give the reviewer the context for that exact commit and a bounded question. Declare every required repository input in the graph. Keep external observations separately identified.
3. Continue independent work in another worktree. Changes there do not alter the committed inputs already compiled for the reviewer.
4. After integrating another change, run `impact` from the reviewed commit to the new candidate. Evaluate the recorded review with `review --evidence` using the same executable and review configuration.
5. Reuse only an eligible report for its original question. Review affected meaning whenever inputs change or relevant dependencies were omitted. Record any human decision through the consumer's own workflow.

There is no need to freeze all work during a fixed-context review. Concurrent writes to one checkout still need coordination. Rendered file writes are atomic per file and guarded against source changes and unmanaged collisions; Git remains responsible for integration across files and branches.

## Verification and failure handling

`check` returns structural and generated-output findings. `verify --revision COMMIT` first performs that check, materializes the same snapshot and runs its profile gates. The Konfyra adapter includes its existing renderer and Python regressions. If that snapshot has the Go bootstrap, it also runs the corresponding native Go regression suite; a runner without its paired test, or a test without its runner, is incomplete evidence. Gate processes ignore ambient `GOFLAGS`, Go environment-file configuration and workspace redirects, including Go calls made by existing Python checks. The native Go suite runs with `-count=1` so verification executes its tests again.

Each verification process has a ten-minute limit and at most 1 MiB of captured combined output. Timeout or output overflow cancels the direct process; a two-second pipe-wait limit bounds inherited pipe handles. This mechanism does not guarantee termination of every descendant process created by a consumer script. The YAML report retains completed gate results and classifies the failure. A failed gate returns exit 1; unavailable tools, incomplete evidence and execution limits return exit 2. Neither is a passed verification.

| Observation | Action |
|---|---|
| `output-drift` after editing YAML | Run format, Markitect render, the consumer-owned provider renderer if applicable, then checks. Inspect the complete diff. |
| A write refuses `main` or `master` | Continue in the intended non-protected worktree. |
| Source changed during rendering | Keep the edits, reload the working snapshot and render the complete candidate again. |
| Existing `write.lock` | Establish whether a renderer still owns it. Remove only a confirmed abandoned lock, then rerun. |
| Archive or cached executable integrity failure | Compare the pin with the reviewed consumer commit. Re-provision the exact trusted archive; do not edit its checksum to silence the error. |
| Go cache cannot be written | Set writable `GOCACHE` and `GOTMPDIR` before the outer `go run`, or use a built bootstrap. |
| `verify` says incomplete or tool missing | Install the stated prerequisite or repair the committed integration; rerun the same fixed candidate. |
| Gate timeout or output limit | Inspect its bounded output and the underlying check. Correct the cause before recording new evidence. |
| `review-required` | Read the reason, compile the affected context and perform the required semantic review. |
| Invalid original review record | Recover its original report and source commit, or perform a new review. A record is not repaired by changing hashes. |

Git subprocesses bind to the explicit repository and discard inherited repository/object/config overrides. This prevents a surrounding Git hook or shell from redirecting an immutable read or protected-branch check into another checkout.

## Install, upgrade and rollback

Use the exact successful source commit and CI run described in [consumer integration](../integration/README.md). Check the archive against the lock and execute the downloaded bootstrap's tests and `authoring` smoke command before copying it. Copy the complete archive/lock/bootstrap/test set in one consumer change.

An upgrade needs the consumer's structural, generated-view and repository checks at its new fixed commit. A new tool executable invalidates earlier review reuse, even when the content is unchanged. Keep the prior consumer pin commit available.

For rollback, revert the integration commit through the consumer's normal branch/review route, restoring the archive, lock, bootstrap and paired test together. If the upgrade also migrated content, restore or reverse that content migration in the same candidate. Verify the rollback candidate with the restored pin. Existing digest-addressed caches may coexist; deleting every cache is unnecessary. A rollback is a new candidate and requires its own applicable evidence.

The current tool lock is intentionally small. Changing its schema or the resource API requires an explicit migration and tests for old consumers. Development builds, release candidates and accepted releases must have distinguishable versions. Provider schemas and models belong to their specific adapters and consumer policy.

## Release gates

Release evidence identifies a full source commit and the exact distribution. A release is eligible only when these gates hold:

| Gate | Required evidence |
|---|---|
| Independent source | Clean, reviewed Git candidate; preserved source history; no hidden consumer checkout dependency. |
| Compiler and authoring | Go tests, vet, build, current generated schemas, canonical executable example and packaged core authoring. |
| Installation | Windows and Linux hosted CI pass at the candidate; the produced four-file distribution builds and runs through its own bootstrap. |
| Input isolation | Regression evidence for fixed snapshots, foreign Git environment, safe controlled writes and bounded verification failures. |
| Cache and upgrade | Archive/executable checks; actual toolchain and build policy bound to cache eligibility; complete pin upgrade and rollback exercise. |
| Practical consumer | Konfyra candidate checks and an independent semantic review of the actual changed authoring route; same and changed-input review decisions exercised. |
| Documentation | Commands, ownership, limits, prerequisites, installation, recovery and acceptance state agree with the implementation. |

A successful Markitect release gate establishes the supported tool boundary. Each consumer retains its own provider runtime, hosted pipeline and human acceptance gates. Record those individually rather than inferring them from a compiler test. The dated [Konfyra assessment](konfyra-assessment.md) tracks observed evidence; its dossier remains the delivery owner.

GitHub workflow artifacts currently expire after 30 days. The consumer's committed archive and integration remain the durable pin. A public release needs an explicit API-domain, license and provenance/signing decision. Quantified time or token savings require the controlled observations in [measurement](measurement.md).
