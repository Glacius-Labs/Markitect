# Survey: code architecture

AI-generated, read-only, 10 October 2026, against main `02c7e529`. Input for ARCH-01 to ARCH-10, BUG-01 and the zone layout. Register DEC-014 allows removing compatibility code.

## Summary

- About 44.9k of 95.5k production lines exist only for compatibility (about 47%), plus about 37k of the 71.5k test lines.
- The current product (core, application and runtime, about 44k lines) touches the legacy layer at one point only: guarded writes, about 1.3k lines.
- The import gate in `tooling/architecture` treats all of `src/internal/host/**` as one layer, so nothing is enforced inside host.
- No import cycles; `go vet ./src/...` is clean.

## Packages (production / test lines)

**Deterministic core (5.8k / 2.7k):**
- `core` 1,173/480
- `core/snapshot` 106/46
- `modules/projectmodel` 1,297/483
- `host/projectcoverage` 1,609/517
- `host/projectbriefing` 1,630/1,209
- shared: `infrastructure/source` 2,408/1,511

**Product application (14.0k / 8.9k):**
- `projectwork` 2,221/908
- `projectapp` 1,316/653 (the facade for CLI and MCP)
- `projectcli` 2,029/2,813
- `mcp` 817/423
- `projectadoption` 5,642/2,670
- `projectexplore` 1,284/448
- `projectonboarding` 648/979

**Execution runtime (24.1k / 18.9k):**
- `projectrun` 16,576/12,787
- `codexappserver` 3,005/2,518
- `agentexec` 2,516/1,288
- `projectworkspace` 1,441/1,459
- `projectsetup` 568/805
- the Python runners in `tooling/codexrunner` and `tooling/clauderunner`

**Legacy and compatibility (46.2k / 31.6k):**
- the host root package (17,794): `canonical_*` 8,878, v0.13 Project/Domain about 7.6k, guarded-write files 1,297;
- `host/cli` 2,978;
- `compat/v0_13` (kernel plus 11 consumers) about 9.8k;
- authoring and content packages 4.2k;
- `host/canonical` 1.9k, of which only `codec.go` is used by the product;
- `records`, `recordstore` and `assurance` 3.3k;
- six canonical-alpha modules 3.2k;
- the adapter and check binaries.

**Dependency direction:**
- `cmd` → `host/cli` → `projectcli` → `projectapp` → `projectrun`, `projectadoption`, `projectwork`.
- `projectrun` → `agentexec`, `codexappserver`, `projectworkspace`, `projectwork`, `projectbriefing`, `projectcoverage`, `projectexplore`.
- `projectwork` → `core`, `projectmodel`, `host/canonical`.

## Layering violations

1. **Product → legacy.** `projectwork`, `projectrun/apply.go`, `projectcli/records.go`, `projectexplore`, `projectonboarding` and `projectadoption` import the host root package, only for `GuardedWrite*` and `ToolBuildDigest`. Those functions live in `project_guarded_write.go`, `write_safety.go`, `path_spelling_*` and `tool_build_digest.go`.
2. **Legacy → runtime.** Thirteen canonical controller, inference and goal files import `agentexec`.
3. **Runtime config inside `projectrun`.** `projectsetup` and `mcp` import `projectrun` only for its configuration types.
4. **CLI does composition.** `projectcli` calls `projectrun.Run`, `Resume`, `Repair`, `Verify`, `FullVerify`, `NewTransportInvoker` and `NewLocalWorkspaceService` directly instead of going through `projectapp`. `runAction` is 484 lines.
5. **The architecture gate binary links the whole legacy root** through `host/architecture.go`.

## Legacy weight

- **Canonical projection alpha:** about 18.1k production and 12k test lines.
  - Its own dogfooding does not use it.
  - CI uses it in two places.
  - It is the only legacy importer of `agentexec`.
- **v0.13/v0.14.1 Project/Domain:** about 26.8k production and 25k test lines. It covers 18 legacy verbs, three adapter binaries, `check-artifacts` and `check-modules`.
- **Not compatibility, but trapped in the legacy dispatcher:** `package`, `bundle`, `install`, `version`, `licenses`.
- **Dogfooding still depends on legacy:**
  - `AGENTS.md` (`check`, `context --kind Skill`);
  - CI (43 legacy calls, none to `project`);
  - the pre-commit hook (`check-modules`);
  - `markitect.yaml` and `.markitect/areas`, which render `.agents/`, `.claude/` and `docs/markitect`;
  - 24 of 25 example directories.

## Hotspots

- **Largest files:**
  - `projectrun/run.go`: 2,694 lines; `runOrResume` alone is 1,103 lines.
  - `projectrun/full_verify.go` 1,276
  - `projectbriefing/store.go` 1,267
  - `agentexec/runner.go` 1,167
  - `projectrun/helper.go` 1,108
  - `projectrun/review.go` 1,006
- **Churn since 1 October:** `projectrun` 32.6k changed lines, host root about 24k, `projectadoption` 9.1k, `projectcli` 7.1k, `codexappserver` 6.1k.
- **Duplicated concepts:**
  - byte-identical link rewriters in two compat consumers;
  - two packages named `core`;
  - four plan, apply and verify lifecycles;
  - two Manager-run stacks (`projectrun` and `projectadoption/manager_run.go`);
  - four record stores;
  - copied helpers: `sameStrings` five times, `containsString` four times.
- **Dead code:** 18 unexported functions are never referenced, for example `resolveObligations`, `resolvesConflicts`, `estimateCost`, `addCost` and `ownedPath` in `projectrun/run.go`. About 12 more are referenced only by tests.

## Documents against code

- **Holds:** `clean-architecture-consolidation.md` (cmd → Host → Core, Modules, Infra, Tooling) at the top level.
- **Missing coverage:** the ownership tables in `architecture.md` and `modules.md` omit the roughly 38k lines of product and runtime packages.
- **Stale:**
  - `repository-layout.md` says Project/Domain are retained (contradicts DEC-014).
  - `engineering-constitution.md` is still scoped to v0.13.

## Proposed order

1. **Now:**
   - code map (ARCH-01);
   - guarded-write extraction (ARCH-02);
   - layer rules (ARCH-03);
   - canonical alpha removal (ARCH-04).
2. **After pull request #92, coordinated in the runtime zone:**
   - dead code (ARCH-05);
   - runtime config package and the `projectrun` split (ARCH-06).
3. **With the verb redesign:** one composition root (CLI-03).
4. **Last, in sequence:**
   - dogfood migration (ARCH-07);
   - distribution commands (ARCH-08);
   - Project/Domain removal (ARCH-09).
