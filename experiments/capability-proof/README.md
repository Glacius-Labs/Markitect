# Capability proof checkpoint v1

This is a falsification checkpoint against integrated main `75031b8eca8161b0a247414741828bfb2e1d9069`, not a product demo. [Protocol](../../docs/research/proof-program.md), [inventory](../../docs/research/implemented-capability-inventory.md) and [matrix](../../docs/research/capability-matrix.md) own its setup and interpretation.

The pre-execution inventory found the owner's global-acquisition stop condition. Only its reproducible witness runs here. Fresh ontology projection, agent independence, recursive behavior, greenfield, actual owner-reviewed brownfield, A/B, longitudinal and re-projection trials remain stopped. Historical tests support narrower claims and retain their original limits.

Run from the repository root with the pinned Go toolchain:

```sh
go test ./examples -run '^TestCapabilityProofGlobalAcquisitionStopWitness$' -count=1 -v
```

The test logs one JSON observation and uses only an isolated temporary Git repository. It mutates no adopter or checked-in fixture. It deliberately removes and restores one unique unrelated loose blob inside that temporary repository to prove ordinary reconciliation depends on its acquisition. A passing harness test means the falsification was reproduced; it does not mean C5 passed.

[Results](results.md) contains the durable run classification. Full raw execution logs and an exact file manifest are retained separately from summaries. No private evidence or sealed prior holdout belongs here.
