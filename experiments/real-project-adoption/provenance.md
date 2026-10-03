# Real-project pilot provenance

## Frozen input

- Upstream: `https://github.com/kgrzybek/modular-monolith-with-ddd`
- License: MIT, with the original `LICENSE` retained in the snapshot.
- Upstream commit: `91c8ef24b4cb6ef558c95d8267fa07d68c7059f8` (`master` at preparation time).
- Isolated baseline branch: `codex/pilot-baseline`.
- Isolated baseline commit: `ec01cb6805f93c8a4dbe37efe2a129a58a7ed28d`.
- Selected upstream inputs: 646 files, 711,461 bytes.
- Manifest: `.artifacts/adoption/baseline-final/source-manifest.json`; each selected input has its original path, byte count, and SHA-256.
- Preparation: `experiments/real-project-adoption/prepare.ps1`; refuses a non-empty destination, performs no deletion, verifies the pinned upstream SHA, and checks every copied file's SHA-256 before commit.

The baseline checkout contains no Markitect Domain, package pin, agent context, projection, or pilot implementation. The manifest, note, and `.gitignore` are generated snapshot metadata. Source files remain at their upstream relative paths. Preparation preserves the Windows working-tree bytes. Git normalizes CRLF/LF in the new subset repositories; both frozen arms are byte-identical across all 646 original paths, with no difference from upstream after newline normalization. The [fixed-source audit](evidence/fixed-source-audit.json) records 693,095 committed bytes versus 711,461 selected working-tree bytes and 47 line-ending-only differences from upstream Git. The selection manifest is not a frozen-blob integrity verifier.

## Integrity correction

Two early snapshot candidates, baseline commits `c56d83544a3c144d442853b0da191c82a8a3e401` and `69a440f17dc236b5ab461a5d1599b4b03cd1bc9a`, are superseded and must not be used. An integrity check found that the first candidate replaced the upstream root `README.md` with a pilot note; the second fixed README preservation but predated the script's explicit copy-hash check. The final candidate preserves the source README and verifies all 646 selected source files before creating its commit. The superseded directories are retained as ignored local artifacts and are not pilot inputs.

## Build evidence and limitation

The selected Meetings Application project was built from a separate clone of the frozen source commit using .NET SDK `8.0.418` and an invocation-scoped `RestoreSources=https://api.nuget.org/v3/index.json`. Result: **Build succeeded, 0 warnings, 0 errors**. No database/runtime calls were made. Build outputs exist only in the separate ignored `build-frozen` clone, not in the baseline commit.

An initial attempt encountered a 401 from an unrelated configured private package feed; its URL and organization are intentionally redacted. No credentials were read, changed, or used. The next public-NuGet-only attempt was denied by the sandbox socket policy; the successful retry used the explicitly authorized escalated operation and public nuget.org source only. No global package-source configuration or authentication was changed.

This proves only that the selected Meetings Application project and its included project-reference closure compile at the frozen source snapshot. It does not prove the whole repository builds, any application runtime behavior, database connectivity, or semantic correctness.
