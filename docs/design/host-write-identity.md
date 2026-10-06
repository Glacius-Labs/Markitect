# Host write identity and partial publication

This source-alpha Host contract strengthens existing mutation boundaries. It changes no Core type, Domain operator, Module SPI, published release or policy acceptance rule. [Architecture](../architecture.md) owns the layer boundaries; the [capability checkpoint](../validation/proof-capability-checkpoint-2026-10-06.md) owns exact-source evidence and remaining gates.

## Authority and object identity

A reviewed plan still binds canonical revision, selected inputs, runtime, ownership, target preimages and intended output bytes. Filesystem path checks alone do not authorize mutation: a parent path can change after inspection. The Host opens an `os.Root`, compares its directory identity with the inspected root, and walks each destination-parent component with reparse rejection and identity comparison. Artifact creation, temporary files, mode changes, rename and cleanup use the opened parent handle and a literal leaf name.

This binds mutation to the inspected object and prevents a substituted path from redirecting the operation to a different directory, including another directory inside the same repository. It does not lock filesystem names against an uncontrolled external process. Such a process can move an already approved object after a check. Named-location and current-state revalidation must expose that change; it cannot become passing verification or convergence. These operations are not an OS sandbox or an ownership/authentication mechanism.

Init preserves exclusive creation and explicit partial state. Its reread must use a regular-file handle matching the inspected leaf identity, and recheck the named leaf before accepting bytes. An alias resolving inside a root is insufficient: exact inspected artifact identity matters.

## Publication and failure

Artifact writes remain atomic per file, not transactional across files. Unpublished temporary files created by the writer may be cleaned up through their pinned parent handle. Once rename publishes an output, a later identity failure must retain that output and expose the partial publication. It must not delete the published leaf, silently restore an old version, or report that no write occurred. Current target-state observation and verification remain separate from publication.

The external controller lease is exclusive, identity-bound to its approved parent and created leaf, and conservatively retains remnants when identity cannot be confirmed. Release must not delete a replacement lease belonging to another operation. There is no automatic crash recovery.

The record store has its own private persistence boundary. Root and events-directory identities bind all subsequent operations. Append still requires expected-head compare-and-swap. After publication, validated readback must contain the exact new sequence and event digest; missing or invalid readback returns `ErrCommittedButUnobserved`. It does not roll back history or retry. A read may observe a valid prior head during a concurrent append; a later write must revalidate that head. The event JSON format, historical bytes, content IDs and explicit active-selection authority are unchanged.

## Evidence scope

Negative controls must confirm that the intended alias or file substitution actually happened. Failure to create a symlink or to rename an open directory is a platform limit, not an executed alias-refusal control. Windows junction tests and Linux symlink tests are distinct evidence. A package pass with skipped controls cannot be described as exercising those controls.

The repository artifact writer covers scoped canonical Apply, projection Apply, generated outputs and schemas, Format, Install and Init. The external controller lease and record store are separate bounded changes. Adoption workspace creation, adapter-input staging, agent temporary-workspace setup and CLI package/distribution outputs have their own boundaries; this contract makes no blanket claim about all Host writes. Uncontrolled mutation within approved directories can still cause validation errors or partial state.

No passing checker, digest, opened handle or supplied review identity proves semantic adequacy, human acceptance, external-provider privacy or authenticity. Apply remains materialized-unverified; fresh independent verification remains required.
