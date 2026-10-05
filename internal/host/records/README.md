# Host operational records

This package validates operational projection and verification facts without
loading source files, inspecting the filesystem, executing checks, or persisting
data. The Host supplies the fixed canonical source revision, semantic model,
reviewed plan, exact captured source/target input snapshot, request binder
digest, post-materialization target facts, declared verifier/check identities,
and currently active projection records.

A ProjectionRecord identifies the canonical Projection Definition, semantic
scope and policies, installable Module/Projector identity, and exact artifacts.
It binds four distinct inputs: the full canonical source revision, the semantic
ModelDigest, the reviewed PlanDigest, and the pre-apply InputSnapshotDigest.
The RequestDigest is an opaque sha256 value from the canonical Host binder; this
package validates its format and binds it into the record ID without trying to
reconstruct the request. TargetSnapshotDigest separately describes the
post-materialization path, byte-digest, and mode set.

Verification results separately describe checks performed against one exact
record and target snapshot. Adding plan/input/request binding fields to a
ProjectionRecord does not change verification semantics: results continue to
bind through record ID, model/revision, post-target digest, and exact declared
verifier/check identities. Neither type is canonical desired intent, approval,
authentication, or a claim of semantic sufficiency. A passing result means only
that every declared check supplied by the caller passed with nonempty digests.

Record IDs and append payloads use deterministic JSON and SHA-256. Payload
helpers return bytes only; the caller owns append persistence, recovery, safe
path handling, and partial-write behavior. Deletion is rejected in this first
contract. Artifact paths are portable relative POSIX paths and file modes are
limited to regular files.

BuildOwnershipIndex receives caller-selected active records and explicit
inventory facts. It rejects multiple active owners for one artifact, preserves
unknown/excluded/ignored facts, and reports observed drift. It does not infer
which historical record is current or whether the supplied inventory covers
the repository. An unobserved owned path stays unobserved.