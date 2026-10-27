# Modules and static composition

The repository uses one Go module. Packages are organized by concrete responsibility and consumer-owned ports. There is no active legacy product layer or second Model check. [Architecture](../architecture.md) explains behavior; the generated [code map](code-map.md) inventories actual packages.

## Go capability packages

Pure model and snapshot packages consume supplied bytes and values. Organization and realization use those pure values. Project owns structural acquisition, complete accounting and static model-derived contexts. Run consumes Project and organization directly, retaining one complete hierarchy per mandate. `runtimeconfig` owns closed static native/check metadata. Workflow selects complete fresh Assessment, Implement and Verify Runs. Apply owns preflight and stored-candidate publication after closure; Bootstrap composes the concrete native and Git/check/store adapters. Native Apply qualification and owner acceptance remain separate.

Run owns Session, WorkspaceService, CandidateChecks and Records ports. The repository adapter implements real private Git copies and self-contained stored candidates. The check adapter binds declared checks to prepared pinned container profiles and disposable input/home copies. The file record adapter provides immutable terminal results and concurrent project Notes. The native harness and provider adapters remain governed by their separately reviewed source and bounded feasibility evidence.

Concrete adapters implement physical acquisition and protocols; the values they serve stay independent of execution and infrastructure. Outer runtime configuration acquires static native and check profile metadata and never resolves model intent. [Architecture](../architecture.md) describes the behavior behind each responsibility.

## Package invariants

This list owns the package-level invariants; the [concept](../concepts/v1-concept.md), the architecture and the guides own the behavior around them:

- **Snapshot** represents exact repository-relative paths, modes, immutable bytes, trees and deterministic content identities, and performs no repository I/O. Tree and content digests stay distinct from canonical model digests. Invalid spellings, collisions and unsupported entries fail explicitly.
- **Model** checks the complete canonical language v1 from supplied immutable sources, without filesystem, Git, process or provider dependencies. It keeps exact identities, values, links, provenance and deterministic model digests, and rejects invalid or incomplete constructs. The fixed Core Definitions hold the normative language contracts.
- **Project** acquires explicitly selected model and control inputs through its own ports and prepares everything from one complete capture. It keeps original bytes, source provenance and authored absence, and distinguishes known invalid input from incomplete acquisition. It does not decode runtime profiles, resolve strategies, roles or budgets, start Sessions, execute checks, allocate working copies or publish.
- **Repository adapter** captures the complete repository census and the selected inputs with exact paths, modes and bytes. It resolves a symbolic revision once, keeps the full identity, rejects redirects and changed directory identity, and never replaces historical sources with current external bytes.
- **Agent profile metadata** binds the immutable runtime image identity and the whole metadata file bytes. Preparation may inspect the image and copy build metadata from a created, never started container; it never reads a host provider executable, probes a version, starts a Session or invokes a model. A build-recorded schema digest is an assertion, not an observed schema inventory. Instructions come only from captured inputs, and a static pass establishes neither runtime nor provider capability.
- **Check adapter** keeps literal arguments, the fixed environment policy, finite resource and output limits, and exact image and executable identities. It runs each declared check against the fixed candidate on an independently verified disposable copy with an empty private home and keeps the actual image, tree, start, exit, output, timeout and closure observations. Unknown or uncertain execution or cleanup never counts as a passing check.
- **CLI** reports actual statuses, sources and inclusion reasons; retained reports and printed JSON use the same final projection. Usage errors, known failures and incomplete work have distinct exit statuses.
- **Model view** renders a deterministic view from compiled Definitions and keeps the canonical model digest, rendered byte digest and source/target-bound preview digest distinct. Its guarded write touches only the reviewed generated target after source and target rechecks, never model, control or instruction inputs, and keeps observed partial effects on failure.

## Dependency direction

Imports are allowed by layer. The one rule table in [`rules.go`](../../internal/tooling/architecture/rules.go) assigns every package a layer, lists which layers each layer may import and names the exceptions, some of them for test files only; the generated [code map](code-map.md) shows it. No sibling adapter or common tooling parent grants a general import exemption: adapters share only the container client.

## Apply target dependencies

This heading retains the design's historical link. The current gate covers actual root-layout responsibilities rather than a transitional target beside `src/`. Future packages require an explicit responsibility decision and map entry when implemented; no stub or success gate is required for an absent package. Consumers cannot import an absent producer.

## Responsibility map

Every actual Go package has one row in the rule table with its layer and owning document, and one package doc comment whose first sentence is its summary. Generated [code-map.md](code-map.md) is byte-checked. The table is an actual inventory, not proof that an unimplemented design package exists.

## Adding, changing or removing a capability package

Coordinate shared contracts first. Add, move or remove the package's row in [the rule table](../../internal/tooling/architecture/rules.go), give it a package doc comment and regenerate the code map. Source packaging takes its roots from the same table; update its data selection and tests for a new embedded asset. Keep one root module and reject nested product modules. Delete obsolete source, commands and gates together.

## Mechanical dependency gate

`TestRepositoryArchitecture` in `internal/tooling/architecture` inspects all production, test and platform Go files; `go test ./...` runs it, and `go test ./internal/tooling/architecture` runs it alone. It checks unknown/removed packages, layer directions and named exceptions, table completeness, owner-document links, external-test self imports and test-only testkit use. The document adapter may use only the reviewed guarded-write selectors. Third-party imports remain permitted; this is a Markitect dependency check, not a general proof of I/O purity.

`internal/testkit` imports no Markitect package. Only test files may import it. The standalone consumer bootstrap (`packaging/bootstrap`) and source smoke import no Markitect packages. Go fixture directories and independent evaluation assets are not product package ownership exemptions.

Regenerate the view with:

```text
go test ./internal/tooling/architecture -run TestCodeMapPage -update
```

## Historical module links

Earlier capability/module and old product-layer contracts remain in the [historical guide](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/history/development-modules-legacy-sections-20261010.md) and [preceding source](https://github.com/Glacius-Labs/Markitect/tree/5ff3206917fa3b7482d9daedb903d57a919f56b8).
