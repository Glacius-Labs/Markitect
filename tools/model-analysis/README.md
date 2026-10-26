# Model analysis

[model_analysis.py](model_analysis.py) reads a canonical model the way one reads code: its Package dependency graph, who references a Definition, what a model change touches, and lint signals. It is the separate tool under `tools/` that workstream W5 of the [model track charter](../../docs/design/model-track-charter.md) asks for; it is not part of the product CLI. Python standard library only.

## Input

The tool reads the JSON report of `markitect check`. Its per-Manager contexts, joined, contain every local Definition with its kind, body, typed `spec` references and resolved `bodyLinks`, plus the imported type contracts they use. A Definition is local when its `canonicalModelDigest` equals its context's own digest. The report also carries the registry records and the authored execution ownership.

```text
go run ./cmd/markitect check --repo . > check.json
```

Use the same `--model` and `--dependency` options as for any check. PowerShell 5.1 writes redirected output as UTF-16; the tool reads both encodings. Only a `passed` report with contexts is accepted.

File-local imports are not in the report. `lint --repo DIR` reads them from each Definition's source file, whose path in the report is relative to the checked repository. `lock --repo DIR` reads the dependency manifests the same way.

## Commands

```text
python -I -B tools/model-analysis/model_analysis.py graph check.json [--json]
python -I -B tools/model-analysis/model_analysis.py refs check.json PACKAGE/ID [--json]
python -I -B tools/model-analysis/model_analysis.py impact old.json new.json [--transitive] [--json]
python -I -B tools/model-analysis/model_analysis.py lint check.json [--repo DIR] [--rules RULES.json] [--json]
python -I -B tools/model-analysis/model_analysis.py lock check.json [--repo DIR]
```

**Edges.** An edge is a typed reference anywhere in a local Definition's `spec`, objects and lists included, or a body link. A Definition's `kind` is a type use and not an edge.

**graph** counts local and content Definitions (content means not of the Core meta-Kinds `Kind` or `Property`), spec references and body links. It reports:
- distinct Definition edges and how many cross a Package boundary, with their share;
- Package edges: distinct directed pairs of Packages with at least one edge between them;
- cycles: strongly connected sets of two or more Packages;
- per Package: afferent coupling Ca (Packages that depend on it), efferent coupling Ce (Packages it depends on) and instability Ce / (Ca + Ce). Imported Packages that local Definitions point to appear too, marked as imported.

**refs** lists a Definition's incoming edges with their position, the local Definitions of that Kind when it is a Kind, and the registry artifacts that realize it.

**impact** compares two reports. A Definition changed when its kind, title, body, spec or link targets differ; source paths, digests and link coordinates are provenance and do not count. It lists changed, added and removed local Definitions, the Definitions that reference any of them (by edge or as instances of a changed Kind), and the registry artifacts that realize either. `--transitive` follows referrers of referrers.

**lint** gives warnings, never failures ([ENH-002](../../docs/concepts/register.md#enh-002-model-quality-linter-and-metrics)); it exits 0 whatever it finds. The thresholds are provisional until W6 of the charter.

| Signal | Warns when |
|---|---|
| `size` | a content Definition's body has more than 150 words; more than 300 is `strong` |
| `isolated` | a content Definition has no edge in or out |
| `no-area` | a Package holds content Definitions but no registry record or execution ownership area names it as owner |
| `unset-field` | a field of a Kind that local Definitions use is set by none of them; only top-level `spec` fields, Core meta-Kinds excluded |
| `undefined-term` | a backticked term is used in 3 or more Packages and no Definition has that ID (case-insensitive) |
| `unused-import` | a Definition file imports a Package that none of its edges and not its kind points into (needs `--repo`) |
| `import-cycle` | Packages import each other in a cycle (needs `--repo`) |
| `direction` | a Package edge breaks the rules file (needs `--rules`) |

A backticked term counts when it matches `--term-pattern`, by default `[A-Za-z][A-Za-z0-9]*`: field-name-like words such as `memberId`. Codes like `invalid_state` and phrases are left out.

The rules file is JSON with `layers`, `forbid` or both. Layers run from top to bottom, and a Package may depend on Packages of its own or a lower layer only. Patterns are shell-style globs over Package paths:

```json
{
  "layers": [["workshop"], ["bookings", "billing/*"], ["policies/*"]],
  "forbid": [{"from": "policies/*", "to": "members/*", "reason": "policies name no member area"}]
}
```

**lock** prints the `model-lock.yaml` text for every non-local model in the report: its version from the dependency's manifest and its `canonicalModelDigest`. A version it cannot read is printed as `UNKNOWN`, with a note on standard error.

## Limits

- The import reader understands the forms the [language contract](../../docs/design/language-contract.md#imports) allows when written on one line or as a block list of `package`/`as` mappings. A file it does not understand is named in a note, not guessed.
- The joined contexts hold every Definition of a Package that has a Manager. When the report's Definition count for the local model differs, `graph` says so.
- `undefined-term` reads the term spelling literally: `issueCredit` is not matched to a Definition `issue-credit`.

## Tests

```text
python -B -m unittest discover -s tools/model-analysis
```

The tests build small synthetic check reports and Definition files.
