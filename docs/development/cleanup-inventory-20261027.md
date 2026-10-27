# Cleanup inventory and target (R3b)

Slice R3b of [regular development](../implementation-plan.md#regular-development-of-markitect-owner-decision-26-october), written on 27 October 2026 against Main `541e6c364e7109e3d1e088db814e495af23892d9`. The owner decided that Markitect is developed like a regular software project and asked for the code and documentation to be cleaned up and consolidated ([DEC-076](../concepts/register.md#dec-076-markitect-is-developed-as-a-regular-software-project), [source](../concepts/sources/regular-development-20261026.md)); about 40 Go packages "sounds kind of bad" ([source](../concepts/sources/self-model-pilot-deferred-20261026.md)). This page classifies every Go package, command and maintained document, names one canonical page per topic and gives R3c an ordered work list. It moves, deletes and rewrites nothing. The owner sees it before R3c starts. [R1](repository-structure.md) is the model for the format.

## 1. Result

**Code.** The module has 38 Go packages today (R3a already removed the five module-check packages from the 43 the source counted). 23 are the product, 15 are tooling and support. R3b proposes 30:

| Class | Packages | Meaning |
|---|---:|---|
| Keep | 29 | Needed as it is (one of them, the architecture gate, is slimmed) |
| Keep, move | 1 | Product code that sits under `internal/tooling/` |
| Merge | 5 | Folded into a named package; behavior unchanged |
| Delete | 3 | Duplicate or unused |

Tooling and support shrink from 15 packages to 7. The product stays at 23: `workflow` merges into `apply`, and `guardedwrite` joins from tooling. About 2,000 lines of Go go away with the deletions; the merges move code without changing it.

**Documentation.** Outside `docs/concepts/sources/` (93 verbatim owner sources and their README), `docs/` has 59 Markdown pages. Keep 40, merge 9 into a named page, archive 10, delete none outright. Each topic in §4.1 ends with one canonical page.

The five largest consolidations:

1. **One architecture description.** Packages, layers and allowed imports are written down in twelve places today (§4.3). Target: one rule table in Go that drives the gate and the generated code map, and `docs/architecture.md` as the one human page.
2. **Drop the architecture command.** `go test ./...` already runs the gate (`TestRepositoryArchitecture`), so the CI step, `cmd/markitect-check-architecture` and `internal/tooling/architecturecli` duplicate it.
3. **One release package.** `release`, `release/install`, `releasecli` and the loader part of `source` become `internal/tooling/release` plus the `cmd/markitect-release` entrypoint; the unused GitHub-release and WinGet publisher (`publish`) is deleted (owner decision 1).
4. **Product code out of tooling.** `guardedwrite` and `licenses` are product code (Apply publication, `docs --write`, `markitect licenses`) under `internal/tooling/`, and the product imports `internal/tooling/source` through `guardedwrite`. Target: `internal/guardedwrite` with the Git helpers it needs, notices in `internal/buildinfo`.
5. **Fewer, non-overlapping pages.** Three operations guides become one user guide; `modules.md`, `source-snapshots.md` and the V1 execution architecture fold into `architecture.md`; four owner tables become one; closed records leave Main for the archive.

## 2. Method

- **Code:** `go list -json ./...` for every package, its imports, test imports, files and lines; grep for every package, command and document path in Go, Python, YAML and workflow files; `.github/workflows/ci.yaml`, `release.yaml` and `flake-report.yaml`; `internal/tooling/architecture` (`codemap.yaml`, `target.go`, `codemap.go`, tests) and `internal/tooling/release/source_layout.go`.
- **Documentation:** inbound Markdown links from every tracked page, path mentions in code, the `scripts/check-docs.py` corpus, and a reading of each page's purpose and overlap.
- **Rules.** *Keep* when the product, CI, the release workflow, a test or a current guide needs it as it is. *Merge* when another package or page already owns the same responsibility or topic. *Archive* when a document records a finished slice, a superseded design or retired scope: it leaves Main, stays at a protected archive point and gets an entry in the [archive index](../archive.md), as R1 established. *Delete* when nothing needs it or it duplicates something that stays.
- **Users** in §3 means who needs the package: the shipped product, CI, the release workflow, or tests only.

## 3. Code inventory

### 3.1 Go packages

| Package | Users | Class | Reason |
|---|---|---|---|
| `internal/model` | product | keep | Canonical language check; protected foundation |
| `internal/snapshot` | product | keep | Pure tree values used by ten packages |
| `internal/organization` | product | keep | Manager topology and context |
| `internal/realization` | product | keep | Registry, ownership and accounting |
| `internal/project` | product | keep | Checked preparation for every command; fix its doc comment, which says "Package apply" |
| `internal/run` | product | keep | The Run executor |
| `internal/runtimeconfig` | product | keep | Static profile metadata |
| `internal/harness` | product | keep | Provider-neutral instructions; a later Claude adapter (D18) would be its second consumer |
| `internal/apply` | product | keep | Command preflight and publication; absorbs `workflow` |
| `internal/workflow` | product | merge into `internal/apply` | 79 lines, one consumer (`apply`), no tests of its own; `apply` imports no adapter either, so the concept's rule that the workflow knows no containers or working copies still holds (owner decision 2) |
| `internal/modeldocs` | product | keep | The `docs` view for target projects |
| `internal/adapters/checks` | product | keep | Declared checks in containers |
| `internal/adapters/cli` | product | keep | Command parsing and reports |
| `internal/adapters/codexappserver` | product | keep | Codex Sessions; widen its doc comment, which names only metadata |
| `internal/adapters/container` | product | keep | Container engine client |
| `internal/adapters/documents` | product | keep | Guarded write of the view; keeps I/O out of `modeldocs` |
| `internal/adapters/gitrepo` | product | keep | Capture, Run copies, candidates, publication |
| `internal/adapters/runstore` | product | keep | Run records and Notes |
| `internal/bootstrap` | product | keep | Composition |
| `cmd/markitect` | product | keep | Product CLI |
| `internal/agentd` | product | keep | Session helper in the agent container |
| `internal/agentd/frame` | product | keep | Wire format shared by `agentd`, `container` and `codexappserver` |
| `cmd/markitect-agentd` | product | keep | Built into the runtime image |
| `internal/buildinfo` | product, release | keep | Version; absorbs `licenses` |
| `internal/tooling/licenses` | product, release | merge into `internal/buildinfo` | 9 lines that embed `notices.md`; the code map already gives both packages the same purpose |
| `internal/tooling/guardedwrite` | product, release | keep, move to `internal/guardedwrite` | Apply publication, `docs --write` and bootstrap path checks depend on it; it is not tooling |
| `internal/tooling/source` | product (via `guardedwrite`), release | merge: Git identity, Git output and stored-name helpers into `guardedwrite`, the source loader into `internal/tooling/release` | Removes the product's dependency on a tooling package; the loader serves only packaging and install |
| `internal/tooling/release` | CI, release, product distribution | keep | Source package and bundle, the consumer's `.markitect/tool`; absorbs `install` and the `source` loader |
| `internal/tooling/release/install` | release, smoke | merge into `internal/tooling/release` | Same responsibility (bundle install into a consumer); 594 lines |
| `internal/tooling/releasecli` | CI, release, smoke | merge into `cmd/markitect-release` | 343-line flag handling behind a 9-line entrypoint; a `main` package can hold it and its tests |
| `internal/tooling/publish` | none (only the manual default mode of `markitect-release`, `--tag --run --assets [--publish]`, and its `distribution` subcommand) | delete | GitHub-release publication and WinGet manifests; no workflow or test outside itself calls it, releases were retired on 17 October and V1 excludes release (owner decision 1) |
| `cmd/markitect-release` | CI, release, smoke | keep | Package and bundle commands; absorbs `releasecli` |
| `internal/tooling/architecture` | CI (through `go test`) | keep, slimmed | The import gate; one rule table replaces five lists (§5.2, step 2) |
| `internal/tooling/architecturecli` | CI step, smoke build | delete | 38-line wrapper of `CheckRepository`, which `TestRepositoryArchitecture` already runs |
| `cmd/markitect-check-architecture` | CI step, smoke build | delete | Duplicates the test above; CI runs the same check twice |
| `internal/testkit` | tests | keep | Hermetic fixtures, test-only |
| `packaging/bootstrap` | product distribution, CI, release | keep | Consumer bootstrap; standalone |
| `tools/smoke` | CI, release | keep | Packaged-source smoke; standalone |

### 3.2 Folders under `tools/` and `scripts/`

| Entry | Users | Class | Note |
|---|---|---|---|
| `tools/smoke` | CI, release | keep | Listed above |
| `tools/library-oracle` | CI | keep | Library oracle and bounded transport |
| `tools/comparison` | study, W6 | keep, frozen `phase-*` | Comparison inputs and runner; its unit tests run in no CI step and need Git |
| `tools/makerspace-oracle` | Phase C assessment | keep unchanged | Hidden assessor material that validation records cite; its 30 tests pass locally but run in no CI step |
| `tools/makerspace-reference` | Phase C | keep unchanged | Ten nested reference modules outside the root module |
| `tools/model-analysis` | W5 | keep | Model analysis tool; its 23 tests pass locally but run in no CI step |
| `scripts/check-docs.py`, `go-test-report.py`, `flake-report.py` and tests | CI, flake report | keep | `check-docs.py` needs its corpus updated with each page merge |
| `scripts/run-library-apply.ps1` | manual Windows Apply | L2 | Starts native Apply from Windows; L1's runner replaces it |

### 3.3 Left to L2, not R3c

L2 removes Windows-only Apply host code after L1 works ([Linux Docker way](../implementation-plan.md#linux-docker-way-owner-direction-26-october)). R3c does not touch these:

| Code | Why it is Apply-host only |
|---|---|
| `internal/bootstrap/runtime_credentials_windows.go` and its test | Credential files of the Apply host |
| `internal/bootstrap/lock_windows.go` | The state-directory lock, used only by `apply` |
| Windows branches in `internal/adapters/gitrepo/local_run.go`, `check_copy.go`, `publication.go` | Run copies, check copies and publication on a Windows host |
| `scripts/run-library-apply.ps1` | Windows operator script for native Apply |
| `windows-latest` in the release `bundle-smoke` job | Named by L2 |

Model commands keep working on Windows, so these stay: `guardedwrite`'s Windows path spelling (`docs --write`), the stored-name checks it takes from `source`, `gitrepo/redirect_windows.go` (capture for `check`), the Windows branch of `container/image_observation.go` (which `check` and `context` reach with `--metadata --runtime-profile`, unless L2 moves metadata to the runner), the Windows branches of `packaging/bootstrap`, and the Windows release binary. The Windows-only tests `gitrepo/repository_alias_windows_test.go`, `bootstrap/documents_alias_windows_test.go` and `guardedwrite/*_windows_test.go` no longer run in CI; L2 decides whether the milestone smoke replaces them.

## 4. Documentation inventory

### 4.1 Topics and their canonical page

| Topic | Canonical page | Other current pages on the topic |
|---|---|---|
| Product thesis | `docs/vision.md` | `concepts/origin-and-rationale.md`, `concepts/reflections/*` |
| Concept and decisions | `concepts/v1-concept.md` (target), `concepts/register.md` (decisions) | `concepts/README.md` is the record's map |
| Architecture | `docs/architecture.md` | `development/modules.md`, `guides/source-snapshots.md`, `design/v1-execution-architecture.md`, `development/code-map.md` (generated), package READMEs for detail |
| Using Markitect on a project | `guides/project-workflow.md` | `guides/project-operations.md`, `guides/operations.md` |
| Provider runtime | `guides/provider-adapters.md` | `packaging/runtime/README.md` for the image |
| Evaluation method | `guides/measurement.md` | Dated protocols in `validation/` are records |
| Development process (checks, PRs, parallel work) | `CONTRIBUTING.md` | `development/parallel-work.md`, `AGENTS.md` steps, plan "Integration" |
| Documentation rules | `development/documentation.md` | `docs/README.md` owns the owner table |
| Repository layout | `development/repository-layout.md` | — |
| Delivery plan | `docs/implementation-plan.md` | — |
| Validation records | `validation/README.md` (map) | The dated records stay as they are |
| Distribution and release | `packaging/README.md` | Release section of `guides/operations.md` |
| Language and model track | `design/language-contract.md`, `design/model-track-charter.md` | The track's pages; the track session owns them |
| Archive | `docs/archive.md` | `development/v1-source-archive-20261023.md` |

### 4.2 Every maintained page

| Page | Class | Target or reason |
|---|---|---|
| `docs/README.md` | keep | The one map and the one owner table; absorbs `development/README.md` |
| `docs/vision.md` | keep | Trim the sections marked historical (starting situation 3, "One organization, explicit scope") or link them to the archive |
| `docs/architecture.md` | keep | Absorbs the four architecture pages below |
| `docs/implementation-plan.md` | keep | Coordinator's page; stream B still names `internal/adapters/claudecode`, which does not exist |
| `docs/archive.md` | keep | Gains the R3c archive point and entries; existing entries that link removed pages get permalinks |
| `docs/assets/*` (2) | keep | README wordmarks |
| `concepts/README.md` | keep | Replace its owner table with a link to `docs/README.md` |
| `concepts/v1-concept.md`, `concepts/register.md` | keep | Concept record |
| `concepts/origin-and-rationale.md` | merge into `vision.md` | The 26 October founding intent in `vision.md` already restates it; move anything not yet there, then remove |
| `concepts/reflections/README.md` and four reflections of 16 October | archive | Dated discussions of the pre-V1 Apply design, which concept 1.0 superseded; the sources they quote stay (owner decision 4) |
| `design/README.md` | keep | Design map |
| `design/language-contract.md`, `design/diagnostic-contract.md` | keep | Normative language specification |
| `design/v1-execution-architecture.md` | merge into `architecture.md` | Target and current source now describe the same packages; still-normative contracts (Run lifecycle, candidates and state, checks, harness and authentication) move, the rest is a record at the archive point |
| `design/kind-interfaces.md`, `manager-context.md`, `model-quality-criteria.md`, `modeling-guide.md`, `standard-libraries.md`, `model-track-charter.md` | keep | Owned by the model track (W2–W6); it consolidates them when it closes. `manager-context.md` describes implemented behavior and then belongs in the harness README |
| `development/README.md` | merge into `docs/README.md` | Four process pages remain; a second map is not needed |
| `development/code-map.md` | keep | Generated; its source becomes the one rule table |
| `development/modules.md` | merge into `architecture.md` | Dependency direction and package invariants are architecture; its "Apply target dependencies" heading exists only for a historical link |
| `development/documentation.md` | keep | Remove its owner table (a copy of the one in `docs/README.md`) |
| `development/parallel-work.md` | merge into `CONTRIBUTING.md` | Overlaps CONTRIBUTING and the plan; stale lines (Extract stream, project-model coverage, a link to a pre-V1 design) |
| `development/repository-layout.md` | keep | Fix the `tools/` row (it names two of six folders) and the "new package" row after step 2 |
| `development/working-method.md` | keep | Owner-adopted method (DEC-038) |
| `development/repository-structure.md` | archive | R1 record; it says itself that it is archived once R2 is done |
| `development/v1-workstream-assignments.md` | archive | Writer scopes of steps 2 and 3, both done |
| `development/concept-conformance-audit-20261024.md` | archive | Closed by concept 1.0 |
| `development/v1-architecture-audit-20261023.md` | archive | v0.17 counter-review; superseded by concept 1.0 |
| `development/v1-source-archive-20261023.md` | merge into `archive.md` | The record of an archive point belongs in the archive index; the branch itself stays |
| `development/cleanup-inventory-20261027.md` (this page) | archive after R3c | Like R1 |
| `guides/project-workflow.md` | keep | The one user guide; absorbs the two pages below |
| `guides/project-operations.md` | merge into `project-workflow.md` | Command reference for the same commands |
| `guides/operations.md` | merge into `project-workflow.md` (troubleshooting) and `packaging/README.md` (release operations) | Also stale: it names module gates and Windows/Linux release gates |
| `guides/provider-adapters.md` | keep | L3 updates it |
| `guides/measurement.md` | keep | Evaluation method |
| `guides/source-snapshots.md` | merge into `architecture.md` | Its owner table is another package-responsibility list; one inbound link |
| `validation/README.md` | keep | Records map |
| 14 dated V1 records (`a1-native-contract-proof`, `v1-comparison-*`, `v1-final-review`, `v1-fixplan-native-controls`, `v1-ks-kr-probe`, `v1-phase-b-arm-m`, `v1-phase-c-arm-m`, `v1-qualification`, `v1-requalification`, `v1-technical-acceptance`) | keep | Frozen records; content unchanged |
| `validation/distribution-retirement-20261017.md` and `retired-releases-20261017.json` | keep | Release evidence the owner ordered kept |
| `validation/production-assessment.md` | archive | 387 lines of evidence for releases that no longer exist; its role as owner of release asset claims has nothing left to own (owner decision 4) |

Outside `docs/`: `README.md`, `AGENTS.md`, `CLAUDE.md`, `SECURITY.md`, `SUPPORT.md`, `examples/README.md`, `packaging/README.md`, `packaging/runtime/README.md`, the package READMEs and the `tools/*` READMEs stay. `CONTRIBUTING.md` absorbs `parallel-work.md` and drops its "Historical Project/Domain contracts" section (the archive index covers it). `internal/adapters/gitrepo/PUBLISHER.md` and `WORKSPACES.md` merge into one `gitrepo/README.md`, so the package has one page. `.github/ISSUE_TEMPLATE/feature_request.yml` still calls Markitect an "authoring and structure-validation tool". The verbatim sources under `docs/concepts/sources/` stay untouched.

### 4.3 Descriptions that exist more than once

| Subject | Where it is written today | Target |
|---|---|---|
| Package list, layers, allowed imports | `codemap.yaml`; `target.go` (`targetRoots`, `responsibilityImports`, `responsibilityTestImports`); `codemap.go` (`Layers`); the examples in `boundaries_test.go`; `source_layout.go` (`sourceRoots`); `code-map.md`; `modules.md`; `architecture.md` (layers and adapters tables); `v1-execution-architecture.md` (package direction); `source-snapshots.md`; `repository-layout.md` and CONTRIBUTING's layout paragraph | One Go rule table and `architecture.md` (step 2) |
| Who owns which document | `docs/README.md`, `development/documentation.md`, `concepts/README.md`, the opening of `vision.md`, `development/README.md` | `docs/README.md` |
| PR and integration rules | `CONTRIBUTING.md`, `parallel-work.md`, `AGENTS.md` step 5, plan "Integration" | `CONTRIBUTING.md`; the plan keeps only current streams |
| What `check`, `context`, `docs` and `apply` do | Openings of `project-workflow.md`, `project-operations.md`, `operations.md`, `provider-adapters.md`, `CONTRIBUTING.md` and `architecture.md` | `project-workflow.md`; the others link |
| Release operations | `operations.md`, `packaging/README.md`, CONTRIBUTING "Release work" | `packaging/README.md` |
| The list of CI checks | `ci.yaml`, CONTRIBUTING's command list and its prose list | CONTRIBUTING's command list only |

## 5. Target

### 5.1 Layout

```text
cmd/
  markitect/              product CLI
  markitect-agentd/       session helper in the agent container
  markitect-release/      release maintenance (absorbs releasecli)
internal/
  model/ snapshot/ organization/ realization/ project/
  run/ apply/ harness/ runtimeconfig/ modeldocs/      (apply absorbs workflow)
  adapters/{checks,cli,codexappserver,container,documents,gitrepo,runstore}/
  agentd/ agentd/frame/
  bootstrap/
  buildinfo/              version and license notices (absorbs tooling/licenses)
  guardedwrite/           guarded working-tree writes, with its Git helpers (from tooling)
  testkit/                test-only
  tooling/
    architecture/         import gate as a test; one rule table; generates the code map
    release/              source package, bundle and install (absorbs install and the source loader)
packaging/bootstrap/      consumer bootstrap (standalone)
tools/smoke/              packaged-source smoke (standalone)

docs/
  README.md  vision.md  architecture.md  implementation-plan.md  archive.md
  concepts/     README.md  v1-concept.md  register.md  sources/ (unchanged)
  design/       README.md  language-contract.md  diagnostic-contract.md  model-track pages
  guides/       project-workflow.md  provider-adapters.md  measurement.md
  development/  working-method.md  documentation.md  repository-layout.md  code-map.md (generated)
  validation/   README.md and the dated records (unchanged)
```

That is 30 Go packages; without owner decisions 1 and 2 it is 32. `docs/` keeps 40 maintained pages.

### 5.2 Work list for R3c

Each step is one small PR that is green on Linux CI by itself. The code steps share the gate tables, the code map, `source_layout.go` and the smoke, so they run one after another with one writer. Documentation steps can run in parallel with them when their files are disjoint. Before each removal, check open PRs for new callers of what is removed. CI no longer compiles for Windows, so every code step also runs `GOOS=windows go vet ./...`; steps 3, 6 and 7 move files that build only on Windows.

**Code**

1. **Remove the architecture command.** Delete `cmd/markitect-check-architecture` and `internal/tooling/architecturecli` and the CI step "Enforce actual responsibility import boundaries"; take them out of the smoke's build list, `sourceRoots`, the gate tables, the code map and the pages that name the command; CONTRIBUTING names `go test ./internal/tooling/architecture` instead.
2. **One rule table.** Replace `targetRoots`, `responsibilityImports`, `responsibilityTestImports`, `Layers` and `codemap.yaml` with one table: each package's layer, which layers a layer may import, and the named exceptions. The container client gets its own layer below `checks`, `codexappserver` and `gitrepo`, which import it in production. The other rules stay: the test-only edges, `testkit` only in tests, standalone packages import nothing, and the document writer uses only the reviewed guarded-write API. Generate `code-map.md` from that table and the package doc comments, with owning-document links that point at `architecture.md` or the package README instead of `modules.md` (the generated page's link to `modules.md#mechanical-dependency-gate` included). Derive `sourceRoots` from the same table, so the packager still refuses unknown paths. Drop the `src/` and `benchmark/` prefixes from `imports.go` and fix the two doc comments (§3.1). Every later step then edits one table (owner decision 3).
3. **Notices into `buildinfo`.** Move `licenses.go` and `notices.md`; update `cli`, the release CLI, `embeddedNoticesPath` in `release/package.go`, the notices test and the links in README and CONTRIBUTING.
4. **Delete the publisher** (owner decision 1). Remove `internal/tooling/publish`, the default publish mode of `markitect-release` (`--tag --run --assets [--publish]`) with `ghRunner`, `runGH` and their tests, the `distribution` subcommand, the usage text, the README release markers and the WinGet paragraph in `packaging/README.md`. Git history keeps it for a future release. This comes before step 5, so the publish wiring is not moved first.
5. **One release package.** Merge `release/install` into `release` and `releasecli` into `cmd/markitect-release`.
6. **`guardedwrite` becomes product code.** Move it to `internal/guardedwrite` together with the Git identity, Git output, stored-name and path helpers it uses from `source`; `release` then takes them from `guardedwrite`. Update the guarded-API rule that names the package (`GuardedWritePackage` today).
7. **Rest of `source` into `release`.** The loader and its tests; the `source` package disappears.
8. **`workflow` into `apply`** (owner decision 2), with `internal/apply/README.md` updated.
9. **Tool tests in CI** (independent, any time). One step for the unit tests of `tools/comparison`, `tools/makerspace-oracle` and `tools/model-analysis`, after a green run on Linux.

**Documentation**

10. **Archive point and closed records.** Create `archive/main-pre-r3c-<date>` at the Main commit before the removals (the ruleset protects `archive/**`); remove the pages classed archive in §4.2 (owner decision 4), add their index entries, merge `v1-source-archive-20261023.md` into `archive.md`, and rewrite remaining links to full-SHA permalinks.
11. **Architecture.** Merge `modules.md`, `source-snapshots.md` and the current contracts of `v1-execution-architecture.md` into `architecture.md`; merge the two `gitrepo` pages into its README; update `AGENTS.md`, `docs/README.md`, `design/README.md` and `repository-layout.md`. After code step 2, which moves the code map's owning-document links off `modules.md` and lets this step remove it. The coordinator owns `architecture.md` and assigns it.
12. **User guide.** Merge `project-operations.md` and `operations.md` into `project-workflow.md`, release operations into `packaging/README.md`. Same writer as L3, or after it, because L3 edits these guides.
13. **Process and maps.** Merge `parallel-work.md` into `CONTRIBUTING.md`, `development/README.md` into `docs/README.md`, remove the duplicate owner tables (§4.3), merge `origin-and-rationale.md` into `vision.md`, and update the `check-docs.py` corpus.

### 5.3 What must not change

- **Product behavior:** commands, flags, reports, exit codes, Apply semantics, the consumer bootstrap and the product paths `.markitect/tool` and `.markitect/bootstrap`. The source archive changes only by the moved and removed packages.
- **Frozen study material:** `tools/comparison/phase-*`, and the Phase C assessor material in `tools/makerspace-oracle` and `tools/makerspace-reference` that validation records cite.
- **Records:** verbatim owner sources under `docs/concepts/sources/` and the content of the dated validation records. The only allowed edit is a link to a page R3c removes, rewritten to its full-SHA permalink, as R2 did.
- **The archive:** the existing `archive/**` refs and what the index entries record. R3c adds entries and rewrites links in existing entries that point to removed pages.
- **Byte-stable fixtures** under `testdata/` and the Libraries under `libraries/`, which the model track owns.
- **L2's scope** (§3.3).

## 6. Decisions for the owner before R3c

1. **Delete the release publisher** (`internal/tooling/publish`, about 1,900 lines with tests): recommended. Releases were retired, V1 excludes release, and nothing calls it; a future release restores it from history or writes it fresh. Package, bundle and install stay.
2. **Merge `workflow` into `apply`:** recommended. It removes a 79-line package with one consumer.
3. **Import rules by layer instead of an exact import list per package:** recommended. Today every new import edits the gate; the rules that matter are the directions (foundations never import adapters, adapters share only the container client, `testkit` only in tests).
4. **Archive the closed records**, including the four reflections and the production assessment: recommended. They stay at a protected archive point and in the index.

## 7. Risks

- Code steps 1–8 edit the same shared files; one writer, one PR at a time, read Main back after each.
- L1, L2 and L3 touch `bootstrap`, `gitrepo`, CONTRIBUTING and the guides at the same time; R3c leaves Windows code to L2 and coordinates the guide merge with L3.
- The model-track session owns the design pages and `libraries/`; R3c only updates links there.
- R3b checked classifications against source and workflows, not by building: the merges need the normal local checks, `GOOS=windows go vet ./...` and a Linux CI run per PR.
- An independent review checked every classification and count against the code before this page was published; its corrections are included.
