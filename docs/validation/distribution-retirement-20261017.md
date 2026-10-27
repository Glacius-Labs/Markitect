# Distribution retirement, 17 October 2026

The owner authorized removal of the earlier releases, binary downloads and obsolete artifacts while retaining Git tags, required evidence and the ongoing Apply Core. This is an operational retirement record, not a new product contract or a completed Apply.

## Owner instruction

Recorder's note: direct owner message on 17 October 2026; original wording preserved.

> Ja, mach das bitte so. Räume Main und GitHub auf, entferne alte Releases und Downloads und passe die Verweise an. Behalte die Git-Tags und benötigten Nachweise. Den laufenden Apply-Core bitte nicht anfassen.

## Preserved evidence

[The captured release metadata](retired-releases-20261017.json) records all 18 release IDs, source commits, tag objects, asset names, sizes, SHA-256 digests and GitHub attestation verification output before removal. All 18 attestations verified; the first parallel attempt could not initialize Sigstore verifiers for two tags, and the successful serial retries and original observations are both retained. This archive preserves identity and observed verification; it is not a mirror of the binaries.

Git tags are retained without moving or rewriting them. The original [production assessment](https://github.com/Glacius-Labs/Markitect/blob/e6d9b94a64c2083ee1487414a728a520a87def03/docs/validation/production-assessment.md) keeps the recorded source/platform/publisher evidence and its limits. Current [runtime handshake](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/runtime-handshake-20261017.md), [capability probes](https://github.com/Glacius-Labs/Markitect/blob/2389e252a85e3896da91f5f4649a3394b9222b2c/docs/validation/runtime-capabilities-20261017.md), the canonical Library model, independent oracle and runtime baseline remain on Main. Historical diagnostics and Owner sources keep their original conclusions.

## Retired from Main

The complete pre-retirement tree is [revision `1f4a1ede`](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426). Exact source links replace local links to removed files.

| Retired area | Immutable source |
|---|---|
| Project/Domain release benchmark and fixtures | [benchmark](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/benchmark) |
| Shop example and former Manager fixtures | [examples](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/examples) |
| Earlier AGENTS.md, MCP, projection, adoption and Case Playground experiments | [experiments](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/experiments) |
| Old WinGet pins and download-dependent checks | [packaging](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/packaging/winget) |
| Old read-only MCP evaluation scripts | [scripts](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/scripts/mcp-evaluation) |
| Superseded compatibility guides | [earlier docs](https://github.com/Glacius-Labs/Markitect/tree/1f4a1ede214d05fd2fee615b3905207df7702426/docs) |

The source release workflow, local package/bundle/install tools and future distribution-generation markers remain. No Apply Core, CLI/bootstrap implementation, Agent adapter or current runtime source changes as part of this cleanup.

## GitHub cleanup boundary

All 18 release pages and their 72 attached assets were removed. A fresh API read returned no releases, and all 261 captured tag refs (including annotated-tag dereferences) were unchanged. The deletion observation is included in the metadata archive. Do not reuse a retired tag or describe an old binary as the current product. Future publication requires a new reviewed version and separate owner authority.

Manual Actions cleanup removes unreferenced artifacts created before 16 October 2026. Keep artifacts named by retained evidence, failed-run diagnostics, flake reports and all artifacts from the current rebuild period. Do not delete workflow runs or their logs. Future ordinary test outputs retain their 14-day flake-analysis window; source transport archives retain 7 days. [Operations](../guides/operations.md#release-operations) owns that policy. The delivery PR records actual Actions deletion counts and remote tag read-back. The obsolete, owner-authored WinGet v0.5.0 proposal [microsoft/winget-pkgs#445055](https://github.com/microsoft/winget-pkgs/pull/445055) was closed without a replacement package or new publication.
