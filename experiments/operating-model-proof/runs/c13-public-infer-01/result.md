# C13 public Infer trial result

**Status: PARTIAL.** One source-byte-bound Infer call ran through the v5 CLI and Host on an isolated three-file public subset. The Host returned a proposal (proposed, exit 0, retry count 0); it did not record an owner decision or adoption (adopted: false, no decision, unauthenticated reviewer).

The proposal was classified as a bounded compromise and referenced evidence-handler as support, evidence-legacy-service as counterexample, and evidence-transition-qualifier as qualification. The model supplied three uncertainty statements. Its medium confidence label is uncalibrated. No frequency claim was present. The result file records the candidate digest without copying candidate wording.

The selected public source revision is 6a79b0d015801f3f37e42cacd049179cc795be42. Inference used separate immutable subset commit 846c6ee3d763a6a0563c76adac54ef63f59ac608; its external-run-staging/subset-source root (identity digest 43263f0c370c876784fea6f19c18ebd55a905269c47c6512529da411b02683db) contains exactly the three selected paths, whose blobs were byte-compared to the original source. The subset has zero exclusion entries because all other paths are absent. This proves the selected input boundary, not completeness or representativeness of the full repository.

The handoff digest is f8d395c0dbaaea8edced1262b72fb95448e50d7a08222ca2464108c06fa37a86, selection digest 63f7501c1f44c93d35d8e245dbd8ae743730e6582c9d9230ca52a5446ea27ee0, queue byte digest e3da4d1eeec23bb13948c55987bba955cf996605df461d0b9a9e8d12a34e4eec, and candidate digest a2414699d7e51ec0e04c7a623d947a8e8050497407c6f5944083ec8efb1653a0. The exact runner, build, receipt, raw output, and private log identities are in result.json. Raw output and private provider logs remain in external trial staging and are not committed.

**Setup correction:** an earlier full-checkout-root capture with 79 collapsed exclusion entries was preserved but not used for inference. Review determined that count did not substantiate repository-wide completeness. The inference was recaptured against the isolated three-file subset, with the original source SHA recorded separately.

This is proposal-only technical evidence. It does not establish accepted intent, adoption, a real owner decision, provider reliability, or representation of unselected content. The full C13 proof is partial: owner review is absent and the separate forward zero-churn proof has not run. The complete configured verify gate was not run for this evidence-only record; the fixed-base Markitect check and engineering-change context passed at the source revision.

The standalone managed-artifact check failed with unmanaged findings for result.json and result.md under the existing operating-model-proof root. The registry was not changed under the task scope; the ownership findings remain unresolved.
