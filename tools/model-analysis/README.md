# Model analysis

[model_analysis.py](model_analysis.py) reads a canonical model the way one reads code: its Package dependency graph, who references a Definition, what a model change touches, and lint signals. It is the separate tool under `tools/` that workstream W5 of the [model track charter](../../docs/design/model-track-charter.md) asks for; it is not part of the product CLI. Python standard library only.

## Input

The tool reads the JSON report of `markitect check`. Its per-Manager contexts, joined, contain every local Definition with its kind, body, typed `spec` references and resolved `bodyLinks`, plus the imported type contracts they use. A Definition is local when its `canonicalModelDigest` equals its context's own digest. The report also carries the registry records and the authored execution ownership.

```text
go run ./cmd/markitect check --repo . > check.json
```

Use the same `--model` and `--dependency` options as for any check. PowerShell 5.1 writes redirected output as UTF-16; the tool reads both encodings. `diagnostics` reads any report and `lock` reads one whose only failure is the missing lock; every other command needs a `passed` report with contexts.

File-local imports are not in the report. `lint --repo DIR` reads them from each Definition's source file. `lock --repo DIR` reads the dependency manifests the same way. A source's `Origin`, `gitrepo:BOUNDARY::MANIFEST`, says what its path is relative to: `.` means the repository given with `--repo`, and an absolute boundary is the directory of a manifest given by absolute path. The files are read from the working tree, also for a report made with `--revision`.

## Commands

```text
python -I -B tools/model-analysis/model_analysis.py diagnostics check.json [--json]
python -I -B tools/model-analysis/model_analysis.py summary check.json [--json]
python -I -B tools/model-analysis/model_analysis.py graph check.json [--json]
python -I -B tools/model-analysis/model_analysis.py refs check.json PACKAGE/ID [--json]
python -I -B tools/model-analysis/model_analysis.py impact old.json new.json [--transitive] [--json]
python -I -B tools/model-analysis/model_analysis.py lint check.json [--repo DIR] [--rules RULES.json] [--json]
python -I -B tools/model-analysis/model_analysis.py lock check.json [--repo DIR] [--out FILE]
```

Output is UTF-8 whatever the console's code page, because Package names may use any script.

**Edges.** An edge is a typed reference anywhere in a local Definition's `spec`, objects and lists included, or a body link. A Definition's `kind` is a type use and not an edge.

**diagnostics** reads any report, also a failed one, and lists its findings by code with the root causes first; the follow-on `*_not_passed` findings come last. Use it when `check` fails: its own output names only the first problem.

**summary** prints what a final report pastes: the status, the local model's ID and digest, and its Definitions per Package.

**graph** counts local and content Definitions (content means not of the Core meta-Kinds `Kind` or `Property`), spec references and body links. It reports:
- distinct Definition edges and how many cross a Package boundary, with their share;
- Package edges: distinct directed pairs of Packages with at least one edge between them;
- cycles: strongly connected sets of two or more Packages;
- per Package: afferent coupling Ca (Packages that depend on it), efferent coupling Ce (Packages it depends on) and instability Ce / (Ca + Ce). Imported Packages that local Definitions point to appear too, marked as imported.

**refs** lists a Definition's incoming edges with their position, the local Definitions of that Kind when it is a Kind, and the registry artifacts that realize it.

**impact** compares two reports. A Definition changed when its kind, title, body, spec or link targets differ; source paths, digests and link coordinates are provenance and do not count. It lists:
- changed, added and removed local Definitions;
- changed imported Definitions, such as a Library Kind after a dependency update. Only those that both reports' contexts carry, that is, the type contracts and targets the model uses, can be compared;
- the local Definitions that reference any of them, by edge or as instances of a changed Kind;
- the registry artifacts that realize either.

`--transitive` follows referrers of referrers.

**lint** gives warnings, never failures ([ENH-002](../../docs/concepts/register.md#enh-002-model-quality-linter-and-metrics)); it exits 0 whatever it finds. The thresholds are provisional until W6 of the charter.

| Signal | Warns when |
|---|---|
| `size` | a content Definition's body has more than 150 words of prose; more than 300 is `strong`. Markdown table rows and fenced code blocks do not count (quality criteria S1) |
| `decisions` | a content Definition has more than 10 decisions, the model's counterpart of cyclomatic complexity: each typed refusal is one, and so is each condition word (`when`, `if`, `unless`, `otherwise`, `except`) in its prose; more than 15 is `strong`. Thresholds are provisional; a CRAP-style risk needs tests bound to statements first (register ENH-003) |
| `isolated` | a content Definition has no edge in or out and names no Package in `appliesTo`, `mayDependOn` or `realizedIn`. The project anchor (`Practice.Project`) is exempt, as quality criteria S9 says. A model written with empty `spec` and no links gets this for nearly every Definition |
| `unknown-package` (strong) | `appliesTo`, `mayDependOn` or `realizedIn` names a Package this model does not have; `.` is the Root. A wrong name in a modeled scope stops context selection for every Manager |
| `dependency-rule` (strong) | a Package edge between local Packages breaks a `markitect/architecture` DependencyRule of the model: a Package covered by the rule's `appliesTo` (with the Packages below it) depends on a Package outside that subtree and outside `mayDependOn` (with the Packages below them). Edges into Libraries are not counted |
| `interface` (strong) | a typed reference from another Package uses a Definition that the governing `markitect/architecture` Interface does not expose in `operations` or `types`. The nearest Package above the target that has an Interface governs it, the Packages below it included; that Package's own subtree is not bound, and an Interface binds only the Packages its `appliesTo` covers. Body links are not uses. Refusal codes are counted in a note, because an Interface cannot list them yet. An Interface lists only operations and field types, so a typed use of anything else in another Package, such as a Rule, is reported too: move it to a shared Package or justify it |
| `no-area` | a Package holds content Definitions but no registry record or execution ownership area names it as owner |
| `unset-field` | a field of a local Kind is set by none of the local Definitions; only top-level `spec` fields. Imported Library Kinds may have optional fields a model never needs, so they are left out |
| `undefined-term` | a backticked term is used in 3 or more Packages and no local Definition has that ID (case-insensitive). Imported IDs do not count, because a model field `name` is not the Core Property `name`. Names a local `spec` sets (`name`, `value`, `wireName` and `code` values, such as field names, enumeration values, wire names and refusal codes) count as defined. `true`, `false` and `null` are never terms |
| `unused-import` | a Definition file imports a Package that none of its edges and not its kind points into (needs `--repo`) |
| `import-cycle` | Packages import each other in a cycle (needs `--repo`) |
| `direction` | a Package edge breaks the rules file (needs `--rules`); for a model with Architecture DependencyRules, `dependency-rule` reads the rules from the model itself |

A backticked term counts when it matches `--term-pattern`, by default `[A-Za-z][A-Za-z0-9]*`: field-name-like words such as `memberId`. Codes like `invalid_state` and phrases are left out.

The rules file is JSON with `layers`, `forbid` or both. Layers run from top to bottom, and a Package may depend on Packages of its own or a lower layer only. Patterns are shell-style globs over Package paths, and `*` also matches `/`: `billing/*` covers `billing/a/b`.

```json
{
  "layers": [["workshop"], ["bookings", "billing/*"], ["policies/*"]],
  "forbid": [{"from": "policies/*", "to": "members/*", "reason": "policies name no member area"}]
}
```

**lock** prints the `model-lock.yaml` text for every non-local model in the report: its version from the dependency's manifest and its `canonicalModelDigest`. Every scalar is double-quoted, so a version such as `1.0` stays a string. It also accepts a report whose own lock check failed, because dependency digests are reported even then; that is how a missing or outdated lock is written. A version or digest it cannot find is printed as `UNKNOWN`, with a note on standard error. `--out FILE` writes UTF-8 with LF line ends; PowerShell 5.1 redirection would write UTF-16.

## Limits

- The import reader understands the forms the [language contract](../../docs/design/language-contract.md#imports) allows: a flow list on one line, or a block list whose items are `package`/`as` mappings in block style or written as one flow mapping each, `- {package: markitect/core, as: Core}`, as the standard Libraries do. A file it does not understand is named in a note, not guessed.
- The joined contexts hold every Definition of a Package that has a Manager. When the report's Definition count for the local model differs, `graph` says so.
- `undefined-term` reads the term spelling literally: `issueCredit` is not matched to a Definition `issue-credit`.

## Tests

```text
python -B -m unittest discover -s tools/model-analysis
```

The tests build small synthetic check reports and Definition files.
