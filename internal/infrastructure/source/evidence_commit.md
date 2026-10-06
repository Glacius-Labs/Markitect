# Immutable selected evidence commits

`WriteSelectedEvidenceCommit` is an explicit Git object write for callers that
need selected UTF-8 evidence represented by a real immutable commit. It takes a
full existing parent commit, exact regular-file bytes, and exact Git modes. It
checks repository identity, requires the worktree `HEAD` to equal the supplied
parent, validates the complete parent tree's portable names, then writes blobs
and a tree through an isolated temporary index.

The operation never updates a ref, `HEAD`, the user's index, or worktree files.
A fixed non-author identity, date, and message make repeated inputs produce the
same commit ID. `WriteSelectedEvidenceCommitContext` accepts cancellation, and each Git process has a timeout. `write=false` performs validation without creating Git objects.
If writing fails after Git has stored objects, the error lists the object IDs;
Git object writes are immutable and are not rolled back.

Only selected blob contents are written or read. Parent tree entries are
inspected as metadata. `write-tree --missing-ok` preserves unavailable,
unselected parent blobs in partial clones, so consumers must expect the result
commit to depend on objects that are not locally available. This API creates
evidence; it does not infer artifact checks, prove domain meaning, or establish
human acceptance.
