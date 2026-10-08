# C1 executable canonical CLI proof

## Scope

This proof tests the public source-alpha canonical model CLI against five existing, project-owned typed ontologies. It establishes structural Schema/Kind/Property/Definition compilation and exact reference diagnostics through committed YAML and pinned schema-only Modules. It does not establish business correctness, ontology completeness, policy sufficiency, human acceptance, source-code meaning, or general expressiveness. No AI invocation was used.

The immutable production baseline is 5c4b1689df94858b57604918238f92e11c4264ae. Protocol expectations were committed before canonical model trials in 3308e7c246f60e7a435b50b6389d03418508b398. The exact module pins were committed in be01d4bbb20bae5d5028195497bc41f044f3d5db. The pretrial and model-trial raw output receipts are in [c1-cli-evidence.json](c1-cli-evidence.json); every model trial retains stdout, stderr, and a Git bundle under c1-cli-runs/.

## Frozen CLI provenance

- Public source commit: 216e101baa2021fb6bf091206b323407aa471b26
- Git archive source SHA-256: 58ca0898cca858daf5c7991738e136e3bd5606bf40cc122a0b949e769bde84f6
- Binary SHA-256: ba0fe8b3b6bd1676091eaa492c58600a08429f2f6e3ff79d9d8cff2d142b30f9; size: 12518912 bytes.
- Go: go version go1.27.1 windows/amd64; argv: go build -trimpath -o <temporary binary> ./cmd/markitect.
Model command: markitect canonical --action model --repo <isolated-repository> --revision <full-Git-commit> --config examples/capability-ontologies/canonical.yaml.
The exact five package pins were produced by frozen CLI module preview. Their manifest and Schema hashes are in the machine-readable report. The pretrial module preview returned exit 0; its stdout and stderr hashes are in [the preview receipt](c1-modules-preview-3308e7c.receipt.json).

## Observed trials

| Isolated trial | Git revision | Result | Model counts / diagnostic |
|---|---|---|---|
| combined-positive | fe78ca35541e... | passed / exit 0 | 5/12/7; sha256:b25f1e8f74858a867a8f9adddd1005e9f5d029a80b5a1e320adf316837978d59 |
| combined-reordered | 42a84894e55d... | passed / exit 0 | 5/12/7; sha256:b25f1e8f74858a867a8f9adddd1005e9f5d029a80b5a1e320adf316837978d59 |
| single-architecture | 70b2763382f3... | passed / exit 0 | 1/3/2; sha256:b82d642d32f2dba349bf6140dd765bd49d4914e8a9a7571a0d757a20c92765b9 |
| single-delivery | 71e1ef73f3ae... | passed / exit 0 | 1/2/1; sha256:c8d2596bc3208d79ffe02ea4a2794337f86086b34d36a89d5278631f8080606f |
| single-responsibility | 2536080612df... | passed / exit 0 | 1/3/2; sha256:1a4bb4bf97bcf8cba333430c6eece12d8ceabed18d68e9fb33b9781087bac881 |
| single-card-combat | 08325243603c... | passed / exit 0 | 1/2/1; sha256:86235d35961349a8b10ac7e12784581cfa9a8461c60f14888693ec1b47f8b30f |
| single-filing-review | d5a1ae438776... | passed / exit 0 | 1/2/1; sha256:852ba1b145f19cc59ea18e5f9f70a49bb36a9b00c61eb57e2a9f6eb6d4949fc4 |
| negative-unresolved | cd69dca8edd4... | failed / exit 1 | reference.unresolved |
| negative-wrong-kind | c1ae90b6da27... | failed / exit 1 | reference.target-kind |
| negative-cardinality | c933cb473cdd... | failed / exit 1 | property.cardinality |
| negative-kind-reference | 06446711196c... | failed / exit 1 | kind-reference.unresolved |

All 11 frozen outcomes matched. The combined and reordered configurations both produced the same model digest and exact normalized edge-block hash. Combined counts were 5 Schemas, 12 Definitions, and 7 Edges. The single-ontology counts matched the frozen 3/2, 2/1, 3/2, 2/1, and 2/1 Definition/Edge expectations. Each negative case returned its exact expected diagnostic and exit 1.
The output logs and repository bundles make each full target revision reproducible. The JSON evidence records per-input SHA-256 hashes and byte lengths, canonical source digests, model digests, exit codes, diagnostics, and package/module paths.

## Modeling friction and limits

The fixtures express closed typed Properties, primitive and enum values, cardinality bounds, explicit Definition references, and typed kindReference values. The Filing Review kindReference survives normalized compilation; a missing target Kind returns kind-reference.unresolved.
The five independent schemas compose because the Source explicitly selects and pins their Modules and lists exact Definitions. These examples contain no cross-ontology constraint vocabulary or business-invariant primitive. Such adopting-project constraints would need project-owned modeling and/or checks. The tested kindReference identifies a Kind; it is not a general business relationship or constraint language.
The output supports a bounded structural capability claim only. It says nothing about whether sample concepts are complete or correct for their domains, whether this language is sufficient for other projects, or whether an operator accepts the models.

## Preserved prior evidence and collection notes

The original c1-8e1a430 helper-test stdout, stderr, and examples/capability_ontologies_test.go hashes still match the original receipt. The helper-test source and all five Schema/twelve Definition fixture files were not edited.
One cardinality package-preview attempt initially returned exit 2 because the changed Schema no longer matched its supplied pin. That exact stderr is retained. The supported recovery was to remove the stale pin only in an intermediate preview commit, obtain the new digest from the frozen CLI, and pin it in the final negative snapshot; the final model command then returned property.cardinality, exit 1.
Temporary PowerShell collector compatibility errors also occurred. One prevented a CLI invocation from starting; another occurred after a successful model command because the runner called a SHA API absent from Windows PowerShell .NET Framework. The latter raw output and matching repository bundle are retained as combined-positive-collector-error2.*. A replay of the original positive commit reproduced the exact stdout SHA-256 observed on its first invocation; its raw output and bundle are retained separately. These are evidence-collector issues, not CLI failures.
