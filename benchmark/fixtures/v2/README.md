# Benchmark fixture v2

This fixture preserves the rollback-review scenario from `v1` while exercising the current opt-in Markdown output contract. Canonical typed YAML lives under `.markitect/areas/sample/`; the ordinary change input remains under `docs/inputs/`. The Project selects the `markdown` target, which generates resource views and local README navigation under `docs/markitect/`.

`v1` remains unchanged for comparisons against immutable pre-0.9 releases.