# Published release benchmark

`run.ps1` compares two published Markitect binaries with a deterministic synthetic rollback-review task. The benchmark intentionally measures CLI work only: each command runs in a fresh process, while fixture creation, Git commits, and output preparation happen before timing.

## Fixture contract

Fixture directories are immutable workload versions. `v1` preserves the pre-0.9 control-plane layout, with typed resources and generated Markdown alongside ordinary project documents under `docs/`. `v2` preserves the same rollback scenario while placing canonical typed YAML under `.markitect/areas/sample/`, keeping an ordinary change input under `docs/inputs/`, and selecting the optional `markdown` target that writes views beneath `docs/markitect/`.

The harness hashes every fixture file by normalized relative path and content. Reusing a fixture version with different bytes is an error; add a new version instead. Each binary gets a copied, isolated Git repository. If both releases use identical fixture bytes, the harness prepares one shared repository so both binaries receive the exact same baseline and candidate commit IDs. When fixture versions differ, each binary receives its compatible fixture and the record marks the run as incomparable; the Markdown summary reports `n/a` for performance change percentages.

The baseline is committed, then the harness finds the single controlled `Text` resource named `rollback-procedure` in namespace `sample`, changes “Pause the rollout” to “Stop the rollout,” renders with the measured binary, and commits the candidate. The benchmark commands are `check`, `render`, `context`, `impact`, and `verify`. Context targets the `rollback-review` Skill. Git initialization, fixture copying, baseline checks, rendering the candidate, and commits are setup costs and are excluded from command timings. Baseline checks run before mutation with each compatible binary so stale or missing checked-in views fail instead of being repaired by candidate rendering.

## Run locally

Use PowerShell 7 and provide the compatible fixture root for each binary. The output directory receives `results.json` and `summary.md`.

```powershell
./benchmark/run.ps1 `
  -CurrentBinary ./dist/markitect-current.exe `
  -PreviousBinary ./dist/markitect-previous.exe `
  -CurrentTag v0.9.0 `
  -PreviousTag v0.8.0 `
  -CurrentFixtureRoot ./benchmark/fixtures/v2 `
  -PreviousFixtureRoot ./benchmark/fixtures/v1 `
  -OutputDirectory ./artifacts/release-benchmark
```

Use `v2` for releases at or after 0.9.0 and `v1` for earlier releases. The release workflow resolves and verifies both immutable published assets, then chooses the previous fixture from the previous release version. It runs on Windows and Linux. Dispatch it with a published stable tag to repeat a failed measurement without changing release assets.

## Measurements and limits

For each command and release, one first fresh-process run is labeled cold and three later fresh-process runs are labeled warm. Release order alternates for each iteration. The operating-system file cache is not cleared, so “cold” means first run on that runner, not a cache-free run. Wall time is process elapsed time; peak working set is sampled while the process runs and is null when a short process exits before a sample can be observed. Standard output and error are capped at 4 KiB each in the JSON record.

The JSON uses schema version 2. It records the fixture version, absolute root, digest, base and candidate commit IDs for each release, plus runner and toolchain metadata and every raw command result. The summary gives medians and ranges for the three warm runs. If fixture version and bytes match, it also reports the median percentage difference; otherwise that field is `n/a`. These measurements have no performance threshold and do not gate a release. Runner load, filesystem state, and short process duration can affect results; treat timing differences as evidence to inspect, not as a causal conclusion.
