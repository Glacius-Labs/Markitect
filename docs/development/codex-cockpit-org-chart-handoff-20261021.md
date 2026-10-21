# Codex cockpit organization chart handoff

**Status: prepared owner assignment, 21 October 2026.** The owner asked for the cockpit to look like a company organization chart and for Codex to implement it ([source](../concepts/sources/cockpit-org-chart-20261021.md)). The [cockpit concept](../design/cockpit.md#organization-chart) owns the requested experience; this handoff supplies the work packages, the further cockpit work and a startup prompt. It implements nothing and starts no session or provider run.

## Base and write scope

Start from integrated Main in an isolated worktree on a `dev/cockpit-org-…` branch. Main must already contain the two Claude changes this work builds on; check that both validation records exist before writing:

- `docs/validation/cockpit-run-concepts-20261021.md`: the run inspector on the current Apply concept (Assess verdict, observation kinds, probes, lookups and context coverage, strategy, no owner channel in closed Runs);
- `docs/validation/cockpit-polish-20261021.md`: the polish of the cockpit areas.

If either is missing, ask the owner instead of building on an older cockpit.

Read [AGENTS.md](../../AGENTS.md), the [working method](working-method.md), the [delivery plan](../implementation-plan.md), [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change), the [cockpit concept](../design/cockpit.md) (especially *Organization and current work*, *Organization chart* and *Acceptance*), the [cockpit adapter guide](../../internal/adapters/cockpit/README.md) and the [accepted Apply workflow](../design/canonical-model/manager-apply-workflow.md). The write scope is the cockpit stream's: `internal/adapters/cockpit/` (Go and `web/`), the cockpit wiring in `internal/bootstrap/` and `internal/adapters/cli/`, the `product/cockpit` self-model Package and registry shard, the cockpit guides and validation records. `internal/bootstrap/main.go` and `internal/adapters/cli/command.go` are shared files: announce edits to the other sessions first. Do not change `internal/run`, `internal/apply`, the run configuration, the Runtime profile or `go.mod` without the Runtime writer's agreement.

The page is plain HTML, CSS and ES modules without a build step, embedded in the binary. The server's Content-Security-Policy forbids inline style attributes (use classes or CSSOM). Use the `h()`, `s()` and `append()` helpers in `web/js/dom.js`; `Element.append(null)` would insert the text "null". Keep the cockpit's card language, area accents, lamps, keyboard access and reduced motion.

## Work packages

Deliver them as focused PRs in this order. Each needs focused local checks, an independent complete-diff review with every finding resolved, and a dated validation record that separates checks, synthetic demonstrations and real evidence.

1. **Organization read model (Go).** Add `GET /api/organization`. Derive it from the project observation, not from a drawing: the Manager tree from the Package hierarchy (path, label, parent, children, depth), Definitions per Package, owned paths from the ownership entries, and each Manager's resolved execution settings exactly as Apply binds them (Execute and Verify profile names and their source — role default or per-Package override; the team's helper roles with profile and write permission; the helper limit; the strategy). Reuse the product's own resolution (`internal/apply` settings and `binding.go`'s `ManagerBinding`), never re-implement its rules. When the project has a Runtime profile, add each profile's provider model; without one say "not set up". Include the Orchestrator and the declared checks as staff units. A pending observation answers as pending. Test that the organization equals the `run.start` binding a run records for the same settings (use a fixture modeled on a real record).
2. **Chart component (browser).** A reusable module that lays out a tree top-down with orthogonal connectors, renders people-like cards (avatar or initial, title, name, facts, status lamp), collapses and expands departments, defaults to two expanded levels for large organizations, supports pan, zoom and fit to screen, and falls back to an indented collapsible tree on small windows. Dotted lines for the reviewer seat and the staff units. Keep the layout a pure, deterministic function of the tree; if you add a JavaScript test, use Node's built-in runner without dependencies and run it from an existing check. Accessibility: tree semantics, keyboard navigation between cards, visible focus, names for every card.
3. **Organization tab.** Model area tabs become Browse, Organization, Session (`#/organization`). The chart shows the static organization. A selected card opens its responsibilities (Definitions linked to Browse, owned paths), its team and its profiles; *Change the team* opens Settings with that Package selected, where a change becomes a proposal as today. Nothing is edited on the chart. Demonstrate on a one-Package project and on Markitect's own model.
4. **Run on the chart.** Replace the run inspector's organization stage with the chart bound to the run's view and binding: state per Manager (working, waiting, ready, done), current Pass and Turn, active team members and reviewer seat, waits as badges naming whom they wait for, observation counts, turns and cost per Manager. Exchanges animate along the reporting lines (submissions up, acceptance and release down, Orchestrator controls along the staff line) and pause with the replay. The owner card has no live lines in a closed Run; legacy records with owner exchanges still show them. Keep the rail, account, timeline, aside, record boundary, selection pinning and reduced motion. Demonstrate on real recorded runs (ask the owner for the location of current run evidence; a verified no-change and a stopped run of 21 October exist on the owner's WSL host).
5. **Finish.** A small organization thumbnail on the hub's Model tile only if it costs no extra observation; README sections, self-model Definitions only where intent changed (use the guarded `markitect docs` write), registry entries and the validation record with screenshots at 390, 860 and 1440 px.

## Further cockpit work

These are open beyond the chart, in priority order. Items that spend provider usage need the owner's grant.

1. **Report outcome (Runtime stream, not cockpit).** `report.json` never contains `outcome`: the report is written before the CLI sets it. The cockpit shows `run.status` with the publication meanwhile. Fix belongs to the Runtime writer or the runtime contract.
2. **Notifications with a person.** Permission prompt, notification click-through and title count in a headed browser at the app window.
3. **Model session on a set-up project.** One session prompt through the cockpit on a project set up through it (provider grant).
4. **A run that changes files.** Publication with per-path read-back in the inspector for a run that writes (provider grant).
5. **Windows without WSL.** Apply needs Linux containers; today the cockpit that starts runs runs in WSL and opens its window on Windows. Prepare the owner decision: support Docker Desktop on native Windows, or a Windows launcher that starts the cockpit inside WSL.
6. **Installer and updates.** With the pinned pilot build ([#503](https://github.com/Glacius-Labs/Markitect/issues/503)).
7. **Project list across processes.** Two cockpits writing `projects.json` at once can lose an update; add a lock if two cockpits per user become usual.

## Startup prompt

```text
Implement the Markitect cockpit organization chart. Read
docs/development/codex-cockpit-org-chart-handoff-20261021.md, the owner source it
links and docs/design/cockpit.md#organization-chart.

Start from fresh integrated Main in an isolated dev/cockpit-org-… worktree. First
check that docs/validation/cockpit-run-concepts-20261021.md and
docs/validation/cockpit-polish-20261021.md exist on Main; if not, stop and ask the
owner. Follow AGENTS.md, the working method, the delivery plan and CONTRIBUTING.
Record the base SHA and your path reservation; announce shared-file edits.

Deliver the five work packages in order as focused PRs: the organization read
model derived from the project observation and Apply's own settings resolution;
a reusable chart component (company-style tree, people cards, dotted reviewer and
staff lines, collapse, pan/zoom, small-window tree, keyboard, reduced motion); the
Model area's Organization tab; the run inspector's organization stage on the same
chart with live state and animated exchanges; then docs, registry and validation.
Never let the chart edit the organization or claim more than the record shows.
Each PR: focused checks, independent complete-diff review, dated validation record.
Start no provider run without the owner's grant.
```
