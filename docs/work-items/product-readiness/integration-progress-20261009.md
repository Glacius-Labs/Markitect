# Product integration progress — 9 October 2026

Owner: Product Integration `01a121a0-0417-71c1-9e4b-be742f8c1146`. Worktree: `product-integration/Markitect`; branch: `codex/product-integration-20261009`. Common worker baseline: `1495e1be7b0046711531fa42c8407fba67b8e814`. The exact owner chat ID is authoritative in backlog.yaml.

## Shared application services checkpoint

CLI setup/doctor, exploration/readiness and durable Brownfield stages now call typed `projectapp.Operations` services. Brownfield Manager-run preview/execution uses the same service with a caller context and explicit `projectadoption.ManagerRunInvoker`; durable attempt outputs and original invocation errors survive partial failure. Setup and provider-free stages do not invoke product roles. Request JSON uses camelCase tags; exploration receives a typed record and clones it before preview mutation. Brownfield requests are an exactly-one-payload action union. CLI owns file-path decoding and output/exit translation.

Files: `internal/host/projectapp/{operations,setup,exploration,brownfield,brownfield_run}.go` and the corresponding `projectcli` wrappers. P05 calls these concrete services without CLI subprocesses. Existing Brownfield fixtures now supply the explicit delegation evidence pools already required by the baseline validator; validator rules were not relaxed.

Focused checks: projectapp suite and CLI exploration/readiness passed; Brownfield context and staged-loop regressions passed after fixture repair; Apply-adoption regression passed after preserving the exact caller-receipt rejection diagnostic. Target setup/parse tests passed. Go vet on projectapp/projectcli passed. Managed artifact accounting passed. Independent static reviews found no substantive extraction regression; nested Selection JSON tags identified by review were corrected and tested. These are developer checks and source reviews, not native acceptance or final supported-platform gates.

## Integration contracts and remaining work

P03 supplies private real Git workspaces and `PrepareCandidate(ctx, Request, []Change, overlayDigest)` so child Managers receive actual parent candidate bytes. Source-WIP freshness remains distinct from materialized candidate inventory. Incoming overlays supply readable context and never widen write ownership. The shared delta validator permits explicit file/directory replacements while rejecting aliases, duplicate operations and write/write collisions.

P04 supplies App Server Run/Fingerprint, durable handle/event callbacks and explicitly selected experimental dynamic tools. The installed 0.162.0 native collaboration protocol bounds concurrent sessions/depth but exposes generic helper-start events after dispatch. A hard lifetime native-helper start cap is not established. P06 will reserve Host-owned roots/reviewers/helpers before start, preserve failed requests, track observed native events and unknowns, and avoid blind replay. A concrete `markitect_start_helper` dynamic-tool proposal allows Host reservation before opening a distinct native helper thread; installed schema support was independently checked by P04. This proposal is not implemented or proven by this checkpoint.

Next eligible work: integrate clean P03/P04/P05 commits, bind App Server/runtime/workspace composition, consume observed deltas into byte/delete-aware candidates, implement dependency-aware concurrency and cumulative lifecycle accounting, then complete P07 before P08 path moves. Current P06 guards remain until a real consumer validates exact harvested inventories. P08 source moves must wait for adapter integration. Optional R03 defaults/provider-projection recommendation is deferred post-Main and does not enlarge this baseline.

A01/A02/A03 remain NOT STARTED. Zero jobs and zero measured role-start requests used from the current separate allocation. Prior closed proof remains unchanged. No release, Main merge, human acceptance or productivity claim follows from this checkpoint.
