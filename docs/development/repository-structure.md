# Repository structure: inventory and target layout (R1)

Slice R1 of [step 3 of the delivery plan](../implementation-plan.md#repository-structure), written on 24 October 2026 against Main `79b9baa55f6e1dce89005f91878de0ae7f139ed3` and checked again after merging Main `41e661e0`. That later Main adds two owner sources to the concept record (`parallel-writers-20261024.md` and the English translation `v1-concept-v018-20261024-en.md`, both current), lands B2 and translates the canonical-model documents; the counts below refer to the base. It classifies every tracked file, proposes the target layout and gives R2 an ordered, mechanical work list. It moves, deletes and rewrites nothing. The owner accepted all six recommendations on 24 October 2026 ([DEC-071](../concepts/register.md#dec-071-repository-structure-and-archive)) and R2 executed this work list; the document stays as the record of the inventory. Paths below name the layout before R2. Since slice R3a ([DEC-076](../concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project)) the repository keeps no `.markitect/` self-model, registry, hook or module checks; the statements about them below describe the layout of that time.

## 1. Result

Main tracks 1,584 files. Every file outside `.markitect/` has exactly one registry entry; the registry holds 1,524 entries, four of them for `.markitect/` files. R1 proposes:

| Class | Files | Meaning |
|---|---:|---|
| Current, unchanged path | 637 | Needed for V1, its development or its qualification |
| Current, moved | 72 | Needed, but in the wrong place today |
| Archive | 873 | Historical value only; removed from Main, kept at a named archive point and listed in an archive index |
| Delete | 2 | Redirect stubs with no content of their own |

By area: `internal/` (428), `cmd/` (5), `.markitect/` (64), the root files (11) and the tooling folders stay as they are. `docs/` shrinks from 542 to 120 files (382 archived, 2 deleted, 41 fixture files leave `docs/`; the two wordmarks and the new archive index join). `experiments/` (502) disappears: 491 files are archived and the Library oracle (11 files) moves to `tools/`. Three top-level folders disappear (`assets/`, `experiments/`, `integration/`) and one appears (`testdata/`).

Two findings change more than paths:

- The step-4 qualification inputs are the canonical Library model under `docs/design/canonical-model/library-example`, the Core dependency inside the language design example and the oracle in `experiments/library-playground`. The test projects in `experiments/test-projects` are **not** step-4 inputs: the concept, plan step 4, the assignments and the qualification record never name them, and no test or gate of a V1 component uses them. Their current consumers are the comparison procedure in the measurement guide, the CI step that tests their own packet tool and the three vendored Core copies in stream C's scope (§6, §7.1).
- GitHub has no Git tags any more (the API returns 404 for `refs/tags`; `git ls-remote` lists none), although the self-model Rule `tooling/maintenance-boundaries`, `docs/vision.md`, `docs/operations.md` and `integration/README.md` say they are preserved. The three current links to `tree/v0.14.1` are dead. Five commits from before the 23 October history reset that remaining files cite (and `7dbd599c`, the commit behind `v0.14.1`) are in no branch; GitHub still serves them, but no ref pins them (§8.1).

## 2. Target layout

```text
.github/       CI (manual and release-called), release and flake-report workflows, issue and PR templates
.githooks/     pre-commit hook: `markitect check` on the staged tree
.markitect/    Markitect's own canonical self-model, registry shards, ignore rules and module pins
cmd/           Go entrypoints: markitect and markitect-agentd (shipped); markitect-check-architecture,
               markitect-check-modules and markitect-release (maintenance)
internal/      private Go packages of the product and its maintenance services
docs/          concept and decisions, design, guides, development, current evidence, archive index
examples/      copyable explicit runtime-profile and run configuration examples, tested by bootstrap
packaging/     what ships besides the binary: runtime container image, consumer bootstrap, distribution guide
scripts/       maintainer scripts: documentation links, CI test and flake reports, native Library Apply operator script
testdata/      byte-stable shared fixtures: the markitect/core dependency and the canonical Library model M1
tools/         independent outside checks that import no product code: packaged-source smoke, Library oracle
```

Root files stay: `README.md`, `AGENTS.md`, `CLAUDE.md`, `CONTRIBUTING.md`, `LICENSE`, `SECURITY.md`, `SUPPORT.md`, `go.mod`, `go.sum`, `.gitattributes`, `.gitignore`.

Two layout facts are deliberate and stay:

- **No `src/` folder.** Markitect is one Go module (`go.mod` at the root, module `github.com/Glacius-Labs/Markitect`). Go places commands under `cmd/` and private packages under `internal/` at the module root; the `internal/` visibility rule is relative to that root, and a `src/` folder is not idiomatic Go. The source packager enforces this: [source_layout.go](../../internal/tooling/release/source_layout.go) refuses a `src/` tree ("legacy src layout is unsupported") and every path inside `cmd/` and `internal/` that is not one of its listed roots.
- **`.markitect/` in Markitect's own repository.** It holds Markitect's canonical self-model (`model.yaml`, `model-lock.yaml`, `model/`), the registry shards that account for every file, the ignore rules, the execution declarations and the module pins ([self-model guide](https://github.com/Glacius-Labs/Markitect/blob/242f2c5b1efa6c7ecf2396b019593ed5a8a8098f/.markitect/README.md)). Markitect develops itself model-first with its own product ([concept §5.5](../concepts/v1-concept.md#55-protected-foundation-and-self-model), [§6](../concepts/v1-concept.md#6-registry-ownership-and-complete-file-accounting)); `markitect check` runs in CI and in the pre-commit hook and fails on any unaccounted file. It is the same layout Markitect recommends for target projects.

## 3. Method and classification rules

Evidence for each file was collected mechanically and then read where the mechanical evidence did not decide:

- inbound Markdown links, resolved with the parser of `scripts/check-docs.py`, from every tracked Markdown file;
- literal path mentions in every tracked text file (Go, Python, PowerShell, shell, YAML, CI, `.gitattributes`);
- Go fixture paths built with `filepath.Join`, read by hand;
- registry shard and Definition of each file;
- source packaging (`internal/tooling/release/source_layout.go`, `package_source.go`), the architecture gate (`internal/tooling/architecture/target.go`, `imports.go`), CI (`.github/workflows/*.yaml`, `.markitect/modules/pipelines.config`), the pre-commit hook and `scripts/check-docs.py`;
- the concept, the delivery plan, the bounded assignments and the V1 qualification record.

Rules:

- **Current** when a V1 component, test, gate, CI step, the release packager, the qualification or a current guide consumes the file, or when it is part of the concept record (the register is the binding decision log and its owner sources are its provenance).
- **Archive** when the only consumers are other archived files, navigation maps, or provenance links from current documents (those links are rewritten to the archive point), and the content records a superseded design, a closed experiment, retired scope ([concept §2.2](../concepts/v1-concept.md#22-not-commissioned): Extract, Cockpit, authoring, Setup) or dated evidence about retired source.
- **Delete** when the file has no content of its own beyond a pointer and its target is archived. Everything removed from Main stays in the archive point either way; the class only decides whether the archive index lists it.

Translation follows from the class: archived German records keep their bytes and are not translated (§11, decision 2).

## 4. Top-level inventory

| Entry | Purpose | Consumers and evidence | Class | Target |
|---|---|---|---|---|
| `.githooks/` (1) | Pre-commit `markitect check` on the staged tree | Digest pinned in `.markitect/modules/githooks.config`; `internal/tooling/modules/githooks` tests | current | unchanged |
| `.github/` (6) | `ci.yaml` (manual and called by release), `release.yaml` (tags `v*`), `flake-report.yaml`, templates | `ci.yaml` digest and six expected checks pinned in `pipelines.config`; `release.yaml` calls `ci.yaml`; `flake-report.py` reads CI artifacts | current | `ci.yaml` edits (§9.5) |
| `.markitect/` (64) | Self-model, registry shards, ignore rules, execution declarations, module pins | `markitect check` in CI, pre-commit and AGENTS.md; generated view | current | registry and Check edits (§9.6) |
| `assets/` (2) | README wordmarks | `README.md` `<picture>`; `docs/strategy/explainer.md` (archived) | current, moved | `docs/assets/` |
| `cmd/` (5) | Product, agent daemon and three maintenance entrypoints | `sourceRoots` in `source_layout.go`; `target.go`; runtime `Dockerfile` builds `markitect-agentd`; CI and `release.yaml` call the maintenance commands; Check Definitions name them literally | current | unchanged (§7.2) |
| `docs/` (542) | Documentation | §5 | mixed | §7.4 |
| `examples/` (3) | Explicit runtime-profile and run configuration examples | `internal/bootstrap/configuration_examples_test.go`; `commandConfigurationExamples` in `source_layout.go` (packaged); README and project workflow | current | unchanged |
| `experiments/` (502) | Trials, demonstrations, test projects, Library oracle | §6 | mixed | removed; oracle to `tools/library-oracle/` |
| `integration/` (3) | Consumer bootstrap `run-markitect.go`, its test, distribution guide | `release/bundle.go` and four release tests read the bootstrap path; CI `publish-source-artifact` copies it; `target.go` layer `standalone`; `tools/smoke/main.go` copies and tests it | current, moved | `packaging/bootstrap/`, guide to `packaging/README.md` |
| `internal/` (428) | All Go packages | Build, tests, code map, architecture gate | current | unchanged in R2; two package moves proposed for step 5 (§7.5) |
| `packaging/` (7) | Runtime image and project-image examples | `docker build -f packaging/runtime/Dockerfile`; runtime and container guides; stream B2 | current | gains `bootstrap/` and `README.md` |
| `scripts/` (7) | `check-docs.py`, `go-test-report.py`, `flake-report.py` with tests; `run-library-apply.ps1` | CI steps 4, 15, 16; `flake-report.yaml`; the operator script drives native Library Apply through the current `apply` flags | current | path edits only |
| `tools/` (3) | Packaged-source smoke | CI step 14, `release.yaml`; `target.go` layer `standalone` | current | gains `library-oracle/` |
| Root files (11) | Entry, contribution, policy, module files | GitHub community files; `check-docs.py` corpus; `go.mod` | current | link edits only |

## 5. Documentation inventory

`docs/` holds 542 files. Counts in parentheses.

### 5.1 Current documentation

| Group | Files | Evidence | Target |
|---|---|---|---|
| Entry (4) | `README.md`, `vision.md`, `architecture.md`, `implementation-plan.md` | AGENTS.md, README.md, `check-docs.py` corpus; 42, 34 and 81 inbound links for vision, architecture and plan | unchanged path; `README.md` rewritten as the new map (§7.4) |
| Guides (6) | `project-workflow.md`, `project-operations.md`, `provider-adapters.md`, `operations.md`, `measurement.md`, `source-snapshots.md` | Current command, provider, evaluation and snapshot guides; all but `source-snapshots.md` are in the `check-docs.py` corpus; `distribution_readme.go` names `docs/project-workflow.md` | `docs/guides/` |
| Repository guide (1) | `repository-layout.md` | `check-docs.py` corpus; development map | `docs/development/repository-layout.md`, content replaced (§12) |
| Concept record (83) | `concepts/**`: concept 1.0, register, origin and rationale, reflections (5), sources (74 files including two READMEs) | Register cites 120 files; AGENTS.md requires reading it; DEC-069 record rule for sources | unchanged |
| Owner sources outside the record (3) | `design/canonical-model/sources/manager-realization-20261016.md` and its two supplied attachments `concept-assessment-20261016.txt`, `concept-refinements-20261016.txt` | Verbatim owner discussion behind DEC-024 to DEC-031 (register lines 398–456 on Main `41e661e0` cite it) | `docs/concepts/sources/` |
| Design (4) | `design/README.md`, `design/v1-execution-architecture.md`; the normative language specification `minimal-language/language-contract.md` and `diagnostic-contract.md` | V1 architecture cited by AGENTS.md, plan, architecture; the language contract is the protected foundation ([concept §5.5](../concepts/v1-concept.md#55-protected-foundation-and-self-model)) cited by README.md, project workflow and plan | architecture unchanged; specification to `docs/design/` |
| Development (10) | `README.md`, `working-method.md`, `documentation.md`, `parallel-work.md`, `modules.md`, `code-map.md`, `v1-workstream-assignments.md`, `v1-source-archive-20261023.md`, `v1-architecture-audit-20261023.md`, `concept-conformance-audit-20261024.md` | Code map generated by `internal/tooling/architecture`; `modules.md` owns 38 code-map entries; assignments drive steps 2 and 3; the archive record must stay ([plan](../implementation-plan.md#historical-references)); the v0.17 counter-review and the conformance audit are the provenance of the V1 architecture and of concept 1.0 (§29) | unchanged |
| Validation (2) | `README.md`, `v1-qualification-20261024.md` | Qualification record cited by AGENTS.md, plan and provider guide | unchanged; map rewritten |
| Release evidence (3) | `history/distribution-retirement-20261017.md`, `history/retired-releases-20261017.json`, `history/distribution/production-assessment.md` | The self-model Rule `tooling/maintenance-boundaries` requires "required historical release evidence" to be preserved; README.md, `examples/README.md`, `integration/README.md` and `operations.md` link the record; the JSON is the only mapping of the 18 retired release tags to commits; the record names the production assessment as the keeper of the recorded source, platform and publisher evidence, and `docs/development/documentation.md` names it the owner of release asset claims | `docs/validation/` |
| Generated view (1) | `markitect/project.md` | Path fixed in `internal/modeldocs/documentation.go`; CI freshness check | unchanged |
| This document (1) | `development/repository-structure.md` | R2 input | archived when R2 is complete |

### 5.2 Fixtures inside `docs/` (current, moved out)

| Group | Files | Evidence | Target |
|---|---|---|---|
| Core dependency (10) | `design/canonical-model/minimal-language/exchange-example/dependencies/core/**` | Read by `internal/bootstrap/main_test.go`, `internal/project/model_test.go`, `internal/adapters/gitrepo/{fixture,root_dot,run_vertical}_test.go`, `internal/adapters/codexappserver/{agents,agents_hierarchy_runtime}_test.go`, `tools/smoke/library.go`, `scripts/run-library-apply.ps1`; digest pinned in the Library M1 lock; `.markitect/model/markitect/core` is a byte copy | `testdata/core/` |
| Library model M1 (30) | `design/canonical-model/library-example/`: `README.md`, `business-rules.md`, `m1/**`, `vocabulary/**` | `internal/bootstrap/main_test.go` (the M1 lock), `internal/project/model_test.go`, `internal/adapters/gitrepo/fixture_test.go`, `tools/smoke/library.go`, `scripts/run-library-apply.ps1`; the six Managers of the V1 qualification are its Packages (Root, `catalog`, `lending`, `lending/reminders`, `library`, `members`) | `testdata/library/` |
| Library evaluation contract (1) | `design/canonical-model/library-example/evaluation-contract.md` | Specification of the outside oracle; evaluator material that Agents must not receive | `tools/library-oracle/` |

### 5.3 Archived documentation

| Group | Files | What it holds | Evidence for archive |
|---|---|---|---|
| A1 Pre-V1 product design (60) | `docs/design/*.md` except README and V1 architecture (32): adoption and Init (`brownfield-representation-adoption`, `copy-me-evidence-review`, `init-adoption-preparation`, `selective-adoption-handoff`, `project-capability-skills-20261009`, `modeling-skill-guidance`), projection and assurance (`canonical-projection-reset`, `canonical-reverse-inference`, `canonical-verification-refresh`, `canonical-verifier-observation-coverage`, `projection-assurance-direction`, `projection-evidence-refresh`, `projection-first-reconciliation`, `proof-readiness-capabilities`, `resolved-target-equality`, `scoped-target-exclusions`), Project/Domain boundaries (`candidate-dependency-execution`, `clean-architecture-consolidation`, `host-write-identity`, `managed-artifact-coverage`, `nested-project-output-boundary`, `policy-failure-analysis`), product shape (`cockpit`, `domain-language-pressure`, `goal-led-modeling`, `product-vision-alignment`, `project-native-codex-work-20261009`, `project-simplification-and-api-direction-20261009`, `software-architecture-experiment`, `standard-operating-model`, `verb-table`, `work-item-comparison-20261009`); `docs/design/project-world/**` (28) with the shop example | Designs for the Project/Domain and projection line, Init/skills, Cockpit and the earlier model-first workflow | No code, CI or packaging consumer; Cockpit, Setup and authoring are outside V1; register cites 3 files (`cockpit.md`, `work-item-comparison-20261009.md`, `project-world/operation-scopes-and-model-briefings.md`) |
| A2 Apply and canonical-model design (35) | `docs/design/canonical-model/*.md`: `README`, Apply design (`manager-apply-concept`, `-implementation-plan`, `-library-flow`, `-playground`, `-runtime`, `-workflow`, `apply-architecture-review-20261017`, `manager-realization-north-star`, `reconciliation-model-proposal`), commands (`commands`), model consolidation (`consolidation`, `current-main-comparison`, `connector-example`, `task-management-example`), standard library (`standard-library`, `-architecture-platform`, `-data-and-views`, `-history`, `-project-coverage`) with its working-vocabulary pages (`actor`, `artifact`, `check`, `component`, `concept`, `condition`, `decision`, `interface`, `procedure`, `project`, `relationships`, `rule`, `scenario`, `use-case`, `workflow`) | Apply designs DEC-024 to DEC-067, the Extract/Verify/Optimize command design, the standard-library design | Concept 1.0 supersedes the Apply designs ([DEC-068](../concepts/register.md#dec-068-v1-concept-10)); Extract is outside V1; the standard library is an accepted later direction with no V1 consumer; register cites 9 files; current guides link them as earlier owners (§10) |
| A3 Language and model design history (128) | `minimal-language/` except the specification and the Core dependency (88): `README.md` (earlier Core explorations), `language-history.md`, `procedure-proposals.md`, `connector.yaml`, `vocabulary.yaml`, `structural-probe.json`, `exchange-walkthrough-probe.json`, `bootstrap/` (5), `identities/` (10), `probes/` (8), the exchange example without its Core (58: Python probes, `verify.py`, 13 verification reports, the 26-file connector model, `canonical/`); `library-example/verify-model.py` and `verification.json` (2, a pre-compiler model probe that needs `exchange-example/verify.py`); `docs/design/product-model-refinement/**` (13), `task-management/**` (13), `task-management-model/**` (12) | Evidence behind the language v1 baseline, the Command/command collision example, earlier model drafts and the Tasks counter-probe | No Go, CI or packaging consumer; `canonical_core_vector_test.go` names `core.json` only in a comment; nothing runs the Python probes; register cites none |
| A4 Handoffs and contracts (8) | `docs/development/`: `claude-apply-hardening-handoff-20261020`, `claude-apply-implementation-handoff-20261020`, `claude-cockpit-handoff-20261020`, `claude-direct-runtime-handoff-20261021`, `codex-apply-conformance-handoff-20261021`, `codex-cockpit-org-chart-handoff-20261021`, `night-apply-stability-20261021`, `shared-contracts` | Pre-V1 assignments and the v0.13 contract table | The development map marks the four Apply handoffs as historical and superseded by V1 and the two Cockpit handoffs as retired from V1; `shared-contracts.md` and `night-apply-stability-20261021.md` state their historical or closed scope themselves; register cites 6 |
| A5 Validation records (77) | All of `docs/validation/` except the map and the qualification record: the runtime handshake and capability probes of 17 October (provider guide line 35 calls them "historical observations"; the container guide cites them as the origin of its rules, a provenance link), Apply, Extract and recursive-Apply evidence of 19–23 October (`apply-*`, `extract-round2`, `library-extract*`, `recursive-apply-*`, `ccf-apply-preparation`, `session-a-b0-b1-runs`), Cockpit (18 records and 8 screenshots), Project/Domain line (`a01-native-smoke`, `project-*`, `parallel-*`, `proof-capability-checkpoint`, `independent-proof-capability-review`, `canonical-projection-reset`, `projection-first-reconciliation`, `classic-commerce-entrypoint`, `markitect-first-*`, `standard-operating-model`, `selective-adoption-handoff`, `impact-assessment-protocol`, `integration-*`, `clean-architecture-*`), comparisons (`agents-md-vs-markitect`, `real-project-adoption-pilot`, `astra-assessment-2026-10-06`) | Dated evidence about retired source and earlier methods | Evidence about source before the V1 cutover; `vision.md` and `measurement.md` cite three comparison records and the register cites 11 as provenance (rewritten to the archive point) |
| A6 History (35) | `docs/history/**` except the release evidence (moved, §5.1) | Earlier plans, handoffs, surveys, assessments, the WinGet and source-release distribution notes, Apply design before DEC-056 | Already historical by its own map; register cites 8 |
| A7 Strategy and marketing (33) | `docs/strategy/**`, including `marketing-plan.md`, `explainer.md`, `short-posts.md` | Supplied strategy bundles and the marketing working plan of 17 October | Not needed for V1 or its qualification; register cites none (§11, decision 1) |
| A8 Research (6) | `docs/research/**` | Supplied research orientation and protocol-v1 capability records | Pre-V1; register cites `optional-research-r01-r03-20261009.md` |

### 5.4 Deleted documentation

| File | Evidence |
|---|---|
| `docs/development/session-handoff.md` | Short pointer to `docs/history/handoffs/session-handoff.md` (archived); no inbound link |
| `docs/work-items/surveys/README.md` | Short pointer to `docs/history/surveys/README.md` (archived); no inbound link |

## 6. Experiments inventory

| Group | Files | Purpose | Evidence | Class | Target |
|---|---|---|---|---|---|
| Library oracle | `library-playground/**` (9) | Independent outside oracle for Library M1/M2 (cases E01–E12 and the fourth-loan witness) and bounded process transport | CI step 17; Check Definition `tooling/gates/library-oracle`; V12 and V13 ([concept §25.2](../concepts/v1-concept.md#252-acceptance-criteria)); the qualification's oracle packets report the same twelve cases | current, moved | `tools/library-oracle/` |
| Container assessment | `runtime-baseline/assess.py`, `run_oracle.py` (2) | Builds an Agent-produced Library candidate offline in the Go runtime example image and runs the oracle in a hardened container | No test, gate or guide consumes it today; it is current because step 4 needs it: V12 and V13 require an independent oracle, the qualification record (line 50) describes a "separate offline binary build and unchanged outside oracle", and this is the only in-repository harness that builds an Agent-produced candidate offline and runs the oracle in a hardened container. The record does not name the script that built its packets; if the step-4 dispatch uses another harness, these two files are archive | current, moved | `tools/library-oracle/container/` |
| Runtime baseline rest | `runtime-baseline/README.md`, `baseline.py`, `codex_login.py` (3) | Single-Agent provider baseline on the pre-V1 runtime (Claude Haiku, Codex Luna) | No V1 consumer; Claude path is retired (D18) | archive | — |
| Apply trials | `apply-trials/**` (273) | CCF, NIGHT, D-series and plan comparison cases with answer keys | README: series closed; built on the pre-V1 Pass engine; only archived records link it | archive | — |
| Recursive Apply demo | `recursive-apply-demo/**` (23) | DEC-056 two-Apply Library demonstration and Extract runs with a Claude token | Pre-V1 runtime; only archived records link it | archive | — |
| Model assessment | `model-assessment/**` (8) | Session C method trial of the Pass-process assessment | Uses `apply.AssessCaptured` and `TestModelAssessmentHarness`, neither exists in current source | archive | — |
| Test projects | `test-projects/**` (183) | Library r2, Readinglog, Roombook, the ordinary-documentation comparison and its packet tool | Consumers are the archived trials and demo, CI step 18 (tests the comparison packet tool itself), three vendored Core copies listed in stream C, and the comparison procedure of `docs/measurement.md` (lines 56–60 name the packet tool and the Library, Readinglog and Library r2 sets); a comment in `packaging/runtime/examples/python/Dockerfile` names Roombook and Readinglog. Not named by concept, plan step 4, assignments or the qualification record; the ordinary-agent comparison is not a V1 acceptance criterion | archive | — |
| Map | `experiments/README.md` (1) | Index of the folder | Folder ceases to exist; the archive index replaces it | archive | — |

## 7. Settled questions

### 7.1 Library fixture, oracle and test data

Step 4 requalifies fresh M1, M2 with the loan limit 5 → 3, strict no-change, a seeded defect and the Verify Reject branch, judged by outside oracles ([plan](../implementation-plan.md#steps), V12 and V13). The inputs are the canonical Library model M1 with its software vocabulary, the `markitect/core` dependency and the oracle. `scripts/run-library-apply.ps1` copies exactly those three model directories into a fresh fixture repository. The [qualification record](../validation/v1-qualification-20261024.md) reports six Managers (the Packages of M1 plus Root) and twelve oracle cases (the Library playground suite).

Proposal:

- `testdata/core/` and `testdata/library/` hold the byte-stable model fixtures. A root `testdata/` fits Go conventions: the Go tool ignores directories named `testdata`, the architecture gate skips them, and these fixtures are shared by four packages, the smoke and the operator script, so they belong to no single package's `testdata/`. Package-local fixtures stay in `<package>/testdata/` as today (`internal/model`, `internal/testkit`).
- `tools/library-oracle/` holds the oracle, its expectations, its evaluation contract and the container harness. The oracle is executable, has its own test suite and must stay independent of product code, like `tools/smoke`. Keeping evaluator files apart from the model fixture makes the boundary visible: Agents may receive `testdata/library/m1` and `vocabulary`, never `tools/library-oracle/`.
- The test projects are archived. The measurement guide's comparison procedure then refers to them at the archive point. If step 4 or a scheduled comparison needs Readinglog, Roombook or the comparison packets, that dispatch restores them from the archive point into `testdata/projects/`.

### 7.2 Tooling commands, scripts and helper folders

- **`cmd/markitect-check-architecture`, `cmd/markitect-check-modules`, `cmd/markitect-release` stay in `cmd/`.** Every command built from the module's own packages lives under `cmd/`, as Go projects with several commands do; the two other `main` packages, `tools/smoke` and the consumer bootstrap, stay outside on purpose because they must import no product package and are not part of the source package. The `markitect-check-*` and `markitect-release` names already mark the three as maintenance; the architecture gate classifies them as `maintenance-command`. Moving them would change source packaging content, two Check Definitions in the self-model (`tooling/gates/architecture`, `tooling/gates/modules`), CI and release commands and the module pins without a functional gain.
- **`scripts/`** keeps maintainer scripts that operate on this repository or its CI: `check-docs.py`, `go-test-report.py`, `flake-report.py`, their tests, and `run-library-apply.ps1` (it drives the product, so it is not an independent tool).
- **`tools/`** holds independent outside checks that import no product code: `tools/smoke` (Go) and `tools/library-oracle` (Python).
- **`integration/` moves to `packaging/`.** The name suggests integration tests; it actually holds the consumer bootstrap that release bundles copy into `.markitect/bootstrap/run.go`, plus the distribution guide. `packaging/` becomes the single place for what ships besides the binary: `packaging/runtime/` (image), `packaging/bootstrap/` (consumer bootstrap) and `packaging/README.md` (distribution and release guide). The bundle's internal layout does not change.
- **`examples/` stays.** Its two configuration examples are packaged and tested.
- **`assets/` moves to `docs/assets/`.** Its only current consumer is the README wordmark.

### 7.3 Release tooling

`internal/tooling/release` (package, bundle, install) is current: CI `publish-source-artifact` runs `markitect-release package`, `release.yaml` runs `bundle`, and `tools/smoke` packages and builds the source on every CI run. `internal/tooling/publish` (verified GitHub release publication and WinGet manifest generation) has no CI or V1 consumer; only `markitect-release` without subcommand and `markitect-release distribution` call it. The [retirement record](../validation/distribution-retirement-20261017.md) kept "the source release workflow, local package/bundle/install tools and future distribution-generation markers" on purpose, and release is excluded from V1, not abandoned. Recommendation: keep both unchanged in R2; decide `publish` when the owner schedules a release.

### 7.4 Documentation shape

```text
docs/
  README.md                 documentation map
  vision.md                 product thesis
  architecture.md           implemented architecture and boundaries
  implementation-plan.md    delivery plan
  archive.md                archive index (new)
  concepts/                 concept 1.0, register, origin and rationale, reflections, owner sources
  design/                   V1 execution architecture and the canonical language v1 specification
  guides/                   using and operating Markitect: workflow, operations, providers, measurement, snapshots
  development/              contributing: working method, documentation, repository layout, modules, code map,
                            assignments, V1 audits and the pre-cutover archive record
  validation/               current qualification evidence and the retained release evidence
  markitect/project.md      generated self-model view
  assets/                   wordmark images
```

The four orientation documents stay at the `docs/` root because AGENTS.md, README.md and most inbound links point there. Target maps (R2 writes them):

- `docs/README.md`: one table "If you want to…" with rows for implementing V1 (concept, plan, V1 architecture, assignments, qualification), using Markitect (guides), the language (specification), contributing (CONTRIBUTING, development map, repository layout), decisions (register) and history (archive index). No dated status text.
- `docs/design/README.md`: three rows (V1 execution architecture, language specification, diagnostic contract).
- `docs/development/README.md`: the ten current development documents plus the repository layout.
- `docs/validation/README.md`: the qualification record and the release evidence (retirement record, its metadata, the production assessment).

### 7.5 Packages inside `internal/`

No package is moved in R2. Two responsibilities are clearly misplaced and are proposed for step 5, where the self-model Package cut already changes ownership:

- **`internal/tooling/guardedwrite` with `internal/tooling/source`.** CONTRIBUTING defines `internal/tooling/` as independent maintenance services, and the code map describes both as "release and repository tooling". In fact the product's publication and generated-view writes depend on `guardedwrite` (`internal/adapters/gitrepo`, `internal/adapters/documents`, `internal/bootstrap`), and `guardedwrite` uses `source.GitIdentity` and related helpers. Proposed target: `internal/guardedwrite` and `internal/source`, owned by a product Package. `internal/source` overlaps in purpose with `internal/snapshot` and the capture in `internal/adapters/gitrepo`; the step-5 architecture review should decide whether it merges.
- **`internal/tooling/licenses`.** The product CLI (`internal/adapters/cli`, command `markitect licenses`) embeds its notices, and the gate already gives it the `build-data` layer with `internal/buildinfo`. Proposed target: `internal/licenses`. The Rule `tooling/maintenance-boundaries` currently names licenses as a maintenance service, so the move needs that Definition changed; that is model work for step 5, not a mechanical R2 move.

## 8. Archive mechanism

### 8.1 Existing state

- Main's history starts at `447ac3b1` ("Preserve Markitect source snapshot; reset history", 23 October 2026). The remote branch `archive/v1-pre-cutover-20261023` points at that commit, with a verified local bundle ([record](v1-source-archive-20261023.md)). The branch is not protected.
- GitHub has no tags. The [retirement record](../validation/distribution-retirement-20261017.md) of 17 October still counted 261 tag refs; `retired-releases-20261017.json` keeps their tag-to-commit mapping (for example `v0.14.1` → `7dbd599c81540c8203a1b7f83afbc335174f4f1f`).
- Remaining files cite five commits from before the reset (linked from 7, 5, 3, 2 and 1 files: `5ff32069`, `1f4a1ede`, `3ebe3c6e`, `bc4b49e7`, `0c28097c`; three more files mention the last three in plain text); `7dbd599c` is the commit behind the dead `v0.14.1` links. None of the six is in a branch on GitHub. GitHub still serves them, but no ref pins them.

### 8.2 Proposal

One mechanism, reused from the pre-cutover precedent:

1. **Archive point.** The exact Main commit on which R2 starts. It contains every archived and deleted file, is in Main's ancestry and stays reachable as long as Main is not reset again.
2. **Named ref and bundle.** R2 pushes the branch `archive/main-pre-r2-YYYYMMDD` to that commit without force, reads it back, and creates a bundle with a restore check beside the existing one, exactly as `v1-source-archive-20261023.md` describes. The bundle protects against another history reset; the ref keeps the commit named.
3. **Index on Main.** `docs/archive.md` lists every archive point and every archived group from §5.3 and §6: original paths, one line on what it holds and why it still matters, the register IDs that cite it, and a full-SHA permalink `https://github.com/Glacius-Labs/Markitect/tree/<sha>/<path>`. Accepted later directions (standard library, commands, Init and skills) get their own section so they stay findable.
4. **Links.** Every link from a remaining file into an archived file becomes a full-SHA permalink with the same anchor (`blob/<sha>/<path>#anchor`, or `tree/` for directories). Links never use branch or tag names, because names can move or vanish, as the tags did.
5. **No translation.** Archived files keep their original bytes; DEC-069 applies to the documentation on Main.

Both existing points are listed in the same index: the pre-cutover branch and the pre-reset commits (with the honest note that the latter are unpinned unless the owner decides otherwise, §11).

## 9. Work list for R2

R2 runs after A1–A3 and C1 have landed (B2 and the canonical-model translation landed on 24 October) and the owner has accepted this layout. Paths below are relative to the repository root. Use `git mv` for moves so the rename history stays exact; keep pure moves, removals and edits in separate commits.

### 9.1 Preconditions

1. Record base SHA, tree, branch, worktree and clean state; confirm no open PR touches a path in §9.3 or §9.4.
2. Re-run the classification against the base: files added after `79b9baa5` need a class before R2 continues. C1 is expected to add a new core version; it belongs to `testdata/core/` if it replaces the Core dependency.

### 9.2 Archive point

1. Push `archive/main-pre-r2-YYYYMMDD` at the base commit (no force), read it back, create and verify the bundle and restore check as in the precedent.
2. Note the full SHA; every permalink in §9.7 uses it.

### 9.3 Moves (72 files, one commit, content unchanged)

| From | To |
|---|---|
| `assets/` (2) | `docs/assets/` |
| `docs/project-workflow.md`, `project-operations.md`, `provider-adapters.md`, `operations.md`, `measurement.md`, `source-snapshots.md` | `docs/guides/` (same names) |
| `docs/repository-layout.md` | `docs/development/repository-layout.md` |
| `docs/design/canonical-model/minimal-language/language-contract.md`, `diagnostic-contract.md` | `docs/design/` (same names) |
| `docs/design/canonical-model/sources/` (3) | `docs/concepts/sources/` |
| `docs/history/distribution-retirement-20261017.md`, `retired-releases-20261017.json`, `distribution/production-assessment.md` | `docs/validation/` (same names) |
| `docs/design/canonical-model/minimal-language/exchange-example/dependencies/core/` (10) | `testdata/core/` |
| `docs/design/canonical-model/library-example/README.md`, `business-rules.md`, `m1/` (24), `vocabulary/` (4) | `testdata/library/` (same relative paths) |
| `docs/design/canonical-model/library-example/evaluation-contract.md` | `tools/library-oracle/evaluation-contract.md` |
| `experiments/library-playground/` (9) | `tools/library-oracle/` |
| `experiments/runtime-baseline/assess.py`, `run_oracle.py` | `tools/library-oracle/container/` |
| `integration/README.md` | `packaging/README.md` |
| `integration/run-markitect.go`, `run-markitect_test.go` | `packaging/bootstrap/` |

### 9.4 Removals (875 files, one commit)

Directories removed completely: `docs/design/canonical-model/minimal-language/` (remaining 88 files), `docs/design/product-model-refinement/`, `docs/design/project-world/`, `docs/design/task-management/`, `docs/design/task-management-model/`, `docs/history/` (remaining 35 files), `docs/research/`, `docs/strategy/`, `docs/work-items/`, `docs/validation/assets/`, `experiments/` (remaining 491 files). Files removed: every remaining `docs/design/*.md` except `README.md` and `v1-execution-architecture.md`; every remaining file under `docs/design/canonical-model/` (35 Markdown files, `library-example/verify-model.py` and `verification.json`); the eight handoffs and contracts of A4 and `docs/development/session-handoff.md`; every file in `docs/validation/` except `README.md`, `v1-qualification-20261024.md` and the three release-evidence files moved there in §9.3 (`distribution-retirement-20261017.md`, `retired-releases-20261017.json`, `production-assessment.md`).

### 9.5 Code, CI and configuration (one commit; behavior unchanged)

| File | Change |
|---|---|
| `internal/bootstrap/main_test.go` (lines 305, 318) | `../../testdata/core/model.yaml`, `../../testdata/library/m1/model-lock.yaml` |
| `internal/project/model_test.go` (92–97) | base `../../testdata`; `library/m1`, `library/vocabulary`, `core` |
| `internal/adapters/gitrepo/fixture_test.go` (32–36), `root_dot_test.go` (78), `run_vertical_test.go` (386) | Core and Library paths under `../../../testdata/` |
| `internal/adapters/codexappserver/agents_test.go` (261), `agents_hierarchy_runtime_test.go` (419) | `../../../testdata/core` |
| `internal/model/canonical_core_vector_test.go` (2) | Source comment: archive permalink of `core.json` |
| `tools/smoke/library.go` (45–49) | `testdata/library/m1`, `testdata/library/vocabulary`, `testdata/core` |
| `tools/smoke/main.go` (226) | Read the bootstrap from `packaging/bootstrap` (without this, CI step 14 fails) |
| `scripts/run-library-apply.ps1` (110, 117, 118) | Same three paths |
| `tools/library-oracle/container/assess.py` | `DEFAULT_REPO` `parents[3]`; `git archive … tools/library-oracle`; extracted path `tools/library-oracle`; docstring |
| `tools/library-oracle/container/run_oracle.py` | Comment naming the evaluator path |
| `internal/tooling/release/bundle.go` (90, 94) and the tests `bundle_fixture_test.go`, `bundle_test.go`, `publish/publish_fixture_test.go`, `releasecli/local_distribution_test.go` | `packaging/bootstrap/run-markitect.go` and `_test.go` |
| `internal/tooling/publish/distribution_readme.go` (49, 77) | Distribution guide `packaging/README.md`; `docs/guides/project-workflow.md` |
| `internal/tooling/architecture/target.go` (55), `boundaries_test.go` (128, 328), `imports.go` (95) | `integration` → `packaging/bootstrap`; walk prefix `integration/` → `packaging/` |
| `internal/tooling/architecture/codemap.yaml` | Entry `integration` → `packaging/bootstrap`; regenerate `docs/development/code-map.md` with `go test ./internal/tooling/architecture -run TestCodeMapPage -update` |
| `.github/workflows/ci.yaml` | Step 17 `-s tools/library-oracle`; remove step 18 "Check comparison source packets" (it tests an archived tool); `publish-source-artifact` copies from `packaging/bootstrap/` |
| `.markitect/modules/pipelines.config` | New `ci.yaml` digest; the six `yamlPath` indexes stay valid (removed step 18 follows step 17) |
| `scripts/check-docs.py` `DEFAULT_DOCUMENTS` | Remove `docs/history/README.md`, `docs/research/README.md`, `docs/strategy/README.md`, `experiments/README.md`, `integration/README.md`; rename the moved guides and `repository-layout.md`; add `docs/archive.md`, `packaging/README.md`, `tools/library-oracle/README.md` |
| `.gitattributes` | Remove rules for archived paths (`docs/strategy/**`, `docs/research/ontology-projection-assurance.md`, `minimal-language` probes, exchange-example reports); replace the `minimal-language` YAML rules by `/testdata/core/** -text whitespace=blank-at-eol,blank-at-eof,space-before-tab,cr-at-eol`; keep the concept-source and audit rules |

### 9.6 Self-model and registry

1. Remove the 875 registry entries of removed files (documentation shard 383, examples 463, product/run 26, product/runtime 3).
2. Update the 72 moved paths. Shard changes: `testdata/core/**`, `testdata/library/**` and `tools/library-oracle/**` (52 files) go to the `examples` shard and realize `examples/independent-oracles`; the guides, design, concept-source, release-evidence, repository-layout and asset paths (17) stay in `documentation`; `packaging/bootstrap/**` and `packaging/README.md` stay in `tooling`.
3. Register the new `docs/archive.md` (documentation shard).
4. `.markitect/model/tooling/gates/library-oracle.yaml`: `spec.command: "python -B -m unittest discover -s tools/library-oracle"` (must equal CI step 17).
5. `.markitect/README.md`: the Core copy is byte-identical to `testdata/core/model/markitect/core/`.
6. Preview and write the generated view with the returned `previewDigest`.

### 9.7 Documentation

1. Write `docs/archive.md` (§8.2) and the four maps (§7.4); update `docs/concepts/sources/README.md` for the three moved sources.
2. Move the "Recommended target-project layout" and "Ownership and boundaries" sections of the old repository layout into `docs/guides/project-workflow.md` under "Project layout"; replace the rest of `docs/development/repository-layout.md` with the guide in §12.
3. Rewrite links by script: a link from a remaining file to a moved file gets its new relative path; a link to a removed file becomes the archive permalink with its anchor. Counted with the `check-docs.py` parser, 448 link occurrences in 56 files point to removed files (149 in the four maps, which are rewritten anyway, 129 in the register, 22 in the vision), and about 210 occurrences in 36 files need new relative paths. Links in owner sources change only their destination, never their text; no byte-preserved source (`.gitattributes -text`) links to an archived file.
4. Replace the three `tree/v0.14.1` links (CONTRIBUTING.md, docs/README.md, docs/development/documentation.md) with `tree/7dbd599c81540c8203a1b7f83afbc335174f4f1f`.
5. Remove compatibility anchors that only archived files used: the section "Historical architecture links" in `docs/architecture.md` (lines 105–152) and the anchor block at the end of `docs/implementation-plan.md` (lines 87–101), after confirming no remaining file links to them.
6. Update AGENTS.md "Where things are", README.md, CONTRIBUTING.md (commands and paths), `examples/README.md`, `tools/library-oracle/README.md` (self-check path; one section for the container harness, taken from the archived runtime-baseline README), `testdata/library/README.md`, `packaging/README.md` (one line: the 18 retired release versions `v0.1.0`–`v0.14.1` must not be reused; `docs/validation/retired-releases-20261017.json` lists them) and `docs/development/documentation.md` (owner table and maps).

### 9.8 Verification

1. `git diff -M --stat base..moves-commit` shows 72 renames at 100 % similarity.
2. `go build ./...`, `go vet ./...`, `go test ./... -count=1`, `gofmt -l`, `go run ./cmd/markitect-check-architecture --repo .`.
3. `go run ./cmd/markitect check --repo .` and `--revision HEAD`; `go run ./cmd/markitect-check-modules --repo . --hooks .markitect/modules/githooks.config --pipelines .markitect/modules/pipelines.config`; generated-view freshness as in CONTRIBUTING.
4. `python -B scripts/check-docs.py` plus every remaining Markdown file as explicit path; `python -B -m unittest discover -s scripts -p test_*.py`; `python -B -m unittest discover -s tools/library-oracle`.
5. `go run ./tools/smoke -repo . -work ABSOLUTE_EMPTY_EXTERNAL_DIRECTORY`.
6. Every permalink resolves in the archive point: `git cat-file -e <sha>:<path>` for each rewritten link.
7. `markitect check` of `testdata/library/m1` with its two dependencies still resolves the lock, so the Core digest is unchanged.
8. Independent complete-diff review; then archive this document (its content lives on in §12's guide and the index).

## 10. Risks

| Risk | Mitigation |
|---|---|
| About 660 link edits break links or anchors | Scripted rewrite; check-docs on every remaining Markdown file, not only the default corpus; permalink existence check (§9.8) |
| Byte-pinned Core fixture changes on the move (line endings, `.gitattributes`) | Pure `git mv` commit keeps the blobs; the `-text` rule follows in the same PR (§9.5) before any checkout-based check; lock resolution check |
| Collision with active writers (stream A owns the gitrepo and codexappserver tests, C1 the Core files; any further translation of held-back records) | R2 runs after them; precondition 9.1 re-checks paths |
| C1 translates files that R2 then archives | C1 should exclude the paths classified as archive here: `exchange-example/canonical/core.json`, `bootstrap/core.yaml`, `identities/core-1.0.0.yaml`, `identities/probe.json` and the three vendored Core copies under `experiments/test-projects`; C1's Core work moves to `testdata/core/` |
| A future history reset loses the archive point | Bundle per §8.2; owner-protected `archive/**` refs (§11) |
| Pre-reset commits disappear on GitHub | Owner decision §11, item 4 |
| Bundling a pre-R2 revision fails after the bootstrap path changes | Accepted: no release is current, and the plan allows no compatibility layer |
| Current guides still name archived designs as owners (§9.7 cannot fix meaning) | Content follow-ups below, owned by the coordinator |
| The self-model `examples` Package now owns `testdata/` and `tools/library-oracle/` paths | Naming only; review in the step-5 Package cut |

Content follow-ups outside R2 (meaning, not paths), for the coordinator: `docs/architecture.md` line 19 names the DEC-067 workflow as the accepted target; `docs/vision.md` lines 50, 58, 129, 143–151 (Extract), 207–209, 283, 291 and 296 route to pre-V1 designs or state that Git tags are unchanged; `docs/operations.md` line 32 and `integration/README.md` line 3 say tags are preserved; `docs/project-operations.md` line 5 and `docs/measurement.md` lines 22, 56–60, 84 and 125 cite archived records or tools as owners; `diagnostic-contract.md` lines 24–34 define its result separation by reference to the archived `exchange-example/verify.py`, `diagnostic_coordinates.py` and `diagnostic_probes.py`; the Library README links the archived `verify-model.py` and `verification.json`; `operations.md`, `project-operations.md` and `packaging/README.md` overlap; `.github/ISSUE_TEMPLATE/feature_request.yml` describes Markitect as an "authoring and structure-validation tool".

## 11. Decisions for the owner before R2

1. **Archive scope.** Remove the 873 archived files from Main, including the marketing working plan, the test projects and the accepted-but-later designs (standard library, commands, Init and skills), which the archive index keeps findable. Alternative: keep `docs/strategy/marketing-plan.md` on Main if marketing work continues.
2. **No translation of archived records.** The translation dispatch's scope list has 78 German files. PR #679 has already translated 37 of them, 32 of which this proposal archives. Of the 41 not yet translated, 40 are the held-back historical records (`product-model-refinement`, `project-world` with the shop example, the Tasks source description, the work-item comparison and the Astra assessment); all are archive and would not be translated. The 41st is the verbatim concept v0.18 source, which keeps its wording beside its new English translation.
3. **Protect the archive refs.** A ruleset that forbids deleting or force-pushing `refs/heads/archive/**` (a repository setting, so the owner's call).
4. **Pre-reset commits.** Either pin the six pre-reset commits of §8.1 with archive refs (this makes old history that the reset removed from the branch graph reachable again), or accept that those links may break and rely on the archive index.
5. **Self-model statement about tags.** The Rule `tooling/maintenance-boundaries` says "Preserve Git tag identities and required historical release evidence". The release evidence stays on Main under this proposal (§5.1); the tags do not exist on GitHub. Restore them from `retired-releases-20261017.json`, or change the Rule.
6. **Package moves in `internal/`** (§7.5) happen in step 5 with the ownership change, not in R2.

## 12. Repository guide

This is the one-page guide R2 puts at `docs/development/repository-layout.md`. It describes the target layout and replaces the "Source repository layout" part of the current `docs/repository-layout.md`; the "Recommended target-project layout" part describes projects that use Markitect and moves to the project workflow guide. After R2 there is one repository guide.

> **Repository layout**
>
> This repository holds Markitect's source, its own canonical self-model and its documentation. For the layout of a project that uses Markitect, see Project layout in `docs/guides/project-workflow.md`.
>
> | Path | Holds |
> |---|---|
> | `cmd/` | Go entrypoints. `markitect` (the CLI) and `markitect-agentd` (runs inside the runtime container) ship; `markitect-check-architecture`, `markitect-check-modules` and `markitect-release` are maintenance commands. |
> | `internal/` | Private Go packages of the product and its maintenance services. The [code map](code-map.md) lists every Go package of the module, its layer and owning guide; [modules](modules.md) states the allowed imports. |
> | `.markitect/` | Markitect's own canonical model, registry shards, ignore rules, execution declarations and module pins. Every file in this repository is accounted for here. |
> | `docs/` | Concept and decisions, design, guides, development, current evidence and the archive index `docs/archive.md`. |
> | `testdata/` | Byte-stable fixtures shared by several packages, the smoke and the qualification: the `markitect/core` dependency and the canonical Library model M1. |
> | `tools/` | Independent outside checks that import no product code: the packaged-source smoke and the Library oracle. |
> | `scripts/` | Maintainer scripts: documentation link check, CI test and flake reports, the native Library Apply operator script. |
> | `packaging/` | What ships besides the binary: the runtime image, the consumer bootstrap and the distribution guide. |
> | `examples/` | Copyable runtime-profile and run configuration examples; tests keep them valid. |
> | `.github/`, `.githooks/` | Manual CI, release and flake-report workflows; the pre-commit model check. |
>
> **Why there is no `src/`.** Markitect is one Go module. Go keeps `cmd/` and `internal/` at the module root, and the source packager refuses a `src/` layout.
>
> **Why `.markitect/` is here.** Markitect is developed model-first with its own product. `markitect check` runs in CI and before every commit and fails on any file without a registration or an explained ignore entry.
>
> **Where new things go.**
>
> | New item | Place | Also update |
> |---|---|---|
> | Go package | `internal/<responsibility>/` | Code map entry, import rules, registry entry |
> | Shipped command | `cmd/<name>/` | Source packaging roots, architecture gate |
> | Fixture used by one package | `<package>/testdata/` | Registry entry |
> | Fixture shared across packages or used by qualification | `testdata/` | Registry entry; keep bytes stable, locks pin them |
> | Outside check that must not import product code | `tools/` | Registry entry, CI step |
> | Maintainer script | `scripts/` | Registry entry, tests next to it |
> | Document | The owning folder under `docs/`, see [documentation maintenance](documentation.md) | Local map, registry entry |
> | Historical material | Not on Main: it stays at an archive point and `docs/archive.md` points to it | Index entry |
>
> **What enforces this.** The architecture gate fails on Go code outside a mapped responsibility; the source packager fails on unknown roots inside `cmd/` and `internal/` and on nested modules; `markitect check` fails on unaccounted files; `scripts/check-docs.py` fails on broken local links.
