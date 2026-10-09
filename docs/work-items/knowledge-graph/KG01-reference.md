# KG01 fixed-source inventory

Reference: `1495e1be7b0046711531fa42c8407fba67b8e814`. Source paths below refer to that commit. This records implemented mechanisms, not a new full Project Verify or production acceptance.

## Input and compilation

`internal/host/projectwork/config.go` selects `.yaml`/`.yml` under the manifest's exact model paths; `project.go:FromSnapshot` binds the repository snapshot, parses definitions, checks path/namespace consistency, calls Core compilation and then ProjectModel analysis. YAML uses `go.yaml.in/yaml/v3`. RDF would replace a representation, not remove identity, ownership, privacy, freshness or project meaning.

`internal/core/types.go` defines identity `[apiVersion, kind, namespace, name]`, separate source path/digest/line, typed normalized definitions and resolved reference edges. `compile.go:Compile` validates closed shapes, type/cardinality, target kind/existence and duplicate references; sorting and semantic digest are deterministic. Literal ordered Check argv preserve order and repetition. Int64 and finite floating values remain distinct. Core edges carry the exact property path and source; they do not declare a scheduler or traversal policy. Source revision and input provenance are excluded from the Core semantic digest.

## Current project capabilities and gaps

| Area | Implemented at reference | Gap / boundary |
|---|---|---|
| Vocabulary | `projectmodel/projectmodel.go:Schema`: Manager, Statement, Artifact, Check, Decision; parent, uses, requires, realizes, checks, subject, actor references | Decision is present in Core but absent from `projectmodel.Report` collections |
| Ownership | `analyze.go:Analyze`: nearest namespace Manager for definitions; longest path selector for files, deeper Manager tie-break; overlap/path/delegation validation | No source-language symbol, call graph or prose-link inference |
| Coverage | Required Artifact paths and observed file matches; missing required paths are incomplete | No universal Statement-to-Artifact requirement; nonempty required checks enforced later by `projectrun/plan.go:planChecksWithImpact`, not by Analyze alone |
| Context | `impact.go:Context`: own facts; direct public foreign uses AND requires targets; public filtering of those targets' outgoing relations; children omit Instructions | Does not take transitive contract closure; selection is not OS isolation; cross-owner artifacts/checks remain outside Context |
| Impact | Both base/candidate graphs, removed edges, reverse consumers, ancestor routing; uses adds routing/context and requires adds target realization/check coverage | Coarse change findings, no complete witness path service; incomplete/unknown inputs broaden the review universe, never an empty success |
| History | `projectbriefing/briefing.go:Build` identity diffs; store persists accepted baseline, events, resolutions and verified resolution bindings | Rename = removed + added; caller-provided actor/acceptance is not authenticated consent |
| Operational trace | `projectexplore/types.go` bindings/receipts; `projectrun/types.go` plans, candidates, checks, review, verify; `deliver.go`, `full_verify.go` and briefing resolution | Split authoritative ledgers; no single generic trace query; missing join is unknown, not inferred |
| Brownfield | `projectadoption/types.go` explicit Claim/EvidenceRef/Contradiction/Question/Resolution; validated evidence methods and per-scope resolution | No arbitrary contradiction detection or semantic truth engine |

All non-Core projections must preserve the binding distinction: model semantic digest, report/project digest, source snapshot digest, runtime/selection/plan/receipt digests. Historical results cannot become current PASS through reindexing. Expected, present, inspected, checked, verified, applied and human accepted are separate states.

## Checks actually executed for startup

Both source-built commands exited 0 in this isolated branch at the pinned reference:

```powershell
go run ./cmd/markitect check --repo . --revision 1495e1be7b0046711531fa42c8407fba67b8e814
go run ./cmd/markitect context --repo . --revision 1495e1be7b0046711531fa42c8407fba67b8e814 --namespace development --kind Skill --name engineering-change
```

- Fixed snapshot: `ecadf3c7eae1dd79daf868bd4ce98ff6f67ef62230105ab1a1a54fe0f5bc2117`.
- Tool digest: `sha256:4c7b7010d8a359a4dbee00fad466323c738b04b7bfd92a0884c43c205165a911`.
- Context digest: `sha256:69b47d1e00b03e8f56f1328a1251656f1ac9ae8229150edaace25ff7d3cde1cf`.
- Check log SHA256: `4324050C02F76DCE17641A78A308DCBEC06979B6A8D4F9C1003BA0B89CE82DDA`.
- Context log SHA256: `A1107A799EDD01ECBB6149A2037DFD962140AF58D4D201ECCCFEE366B22F71CF`.

These are root engineering-resource checks. The source version label is not installed-release proof. Shop model check, Shop cancellation execution, full Go suite, native provider journey and user acceptance were not run by KG01. Existing test cases in the matrix are source witnesses until separately executed. KG01 closes with one independent focused Reader review of this inventory and exactly twelve rows; further discovery requires a specific new scope.
