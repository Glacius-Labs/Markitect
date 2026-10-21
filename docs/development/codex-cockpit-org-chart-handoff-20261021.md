# Codex cockpit organization chart handoff

**Status: prepared owner assignment, 21 October 2026.** The owner asked for the cockpit to look like a company organization chart and for Codex to implement it ([source](../concepts/sources/cockpit-org-chart-20261021.md)). The [cockpit concept](../design/cockpit.md#organization-chart) owns the requested experience; this handoff supplies the work packages, the further cockpit work and a startup prompt. It implements nothing and starts no session or provider run.

## Base and write scope

Start from integrated Main in an isolated worktree on a `dev/cockpit-org-…` branch. Main must already contain the two Claude changes this work builds on; check that both validation records exist before writing:

- `docs/validation/cockpit-run-concepts-20261021.md`: the run inspector on the current Apply concept (Assess verdict, observation kinds, probes, lookups and context coverage, strategy, no owner channel in closed Runs);
- `docs/validation/cockpit-polish-20261021.md`: the polish of the cockpit areas.

If either is missing, ask the owner instead of building on an older cockpit. While Codex works on this assignment, it is the single writer of the cockpit scope; Claude's cockpit session does not write there meanwhile.

Read [AGENTS.md](../../AGENTS.md), the [working method](working-method.md), the [delivery plan](../implementation-plan.md), [CONTRIBUTING](../../CONTRIBUTING.md#verify-a-change), the [cockpit concept](../design/cockpit.md) (especially *Organization and current work*, *Organization chart* and *Acceptance*), the [cockpit adapter guide](../../internal/adapters/cockpit/README.md) and the [accepted Apply workflow](../design/canonical-model/manager-apply-workflow.md). The write scope is the cockpit stream's: `internal/adapters/cockpit/` (Go and `web/`), the cockpit wiring in `internal/bootstrap/` and `internal/adapters/cli/`, the `product/cockpit` self-model Package and registry shard, the cockpit guides and validation records. `internal/bootstrap/main.go` and `internal/adapters/cli/command.go` are shared files: announce edits to the other sessions first. Do not change `internal/run`, `internal/apply`, the run configuration, the Runtime profile or `go.mod` without the Runtime writer's agreement.

The page is plain HTML, CSS and ES modules without a build step, embedded in the binary. The server's Content-Security-Policy forbids inline style attributes (use classes or CSSOM). Use the `h()`, `s()` and `append()` helpers in `web/js/dom.js`; `Element.append(null)` would insert the text "null". Keep the cockpit's card language, area accents, lamps, keyboard access and reduced motion.

## Work packages

Deliver them as focused PRs in this order. Each PR registers its own new files in the exact registry shard (`markitect check` fails otherwise), runs focused local checks, gets an independent complete-diff review with every finding resolved, and adds a dated validation record that separates checks, synthetic demonstrations and real evidence.

1. **Organization read model.** Add `GET /api/organization`. The cockpit adapter may import only `internal/run` and `internal/authoring` (see `internal/tooling/architecture/target.go`), so compute the organization in `internal/bootstrap` behind a port, as `ProjectObserver` and the run facts do, and serve it from the adapter.
   - *Declared, always:* from the project observation (structural preview: model, registry, authored `.markitect/execution.yaml` document): the Manager tree from the Package hierarchy (path, label, parent, children, depth), Definitions per Package, the declared Execute/Verify defaults and per-Package overrides, the declared team (helper roles, profile, writes, `maxHelpers` = helpers active at once), the strategy (run-wide) and the declared checks. Owned areas need the product's ownership layout (most specific area wins) — reuse it, do not re-implement it.
   - *Resolved, on request:* only with a Runtime profile, through the same capture as the run start's confirmation facts (`internal/bootstrap/cockpit_run.go`, `runFacts`): product defaults applied, profiles bound and their provider models (`executeProfileModel`). It needs a clean committed source and can take up to two minutes; the page asks for it and shows its progress. Label every value as declared or resolved.
   - Extend or add a `product/cockpit` Definition for the static organization view in this PR (canonical intent, written through the guarded `markitect docs` preview/write).
   - Test that the resolved profiles, helpers and helper limit equal the `run.start` binding a run records for the same settings (`internal/apply/binding.go` `ManagerBinding`; it carries no default/override source and also carries the brief context, which is not compared).
2. **Chart component (browser).** A reusable module that lays out a tree top-down with orthogonal connectors, renders people-like cards (avatar or initial, title, name, facts, status lamp), collapses and expands departments, defaults to two expanded levels for large organizations, supports pan, zoom and fit to screen, and falls back to an indented collapsible tree on small windows. The owner on top, the Orchestrator between the owner and Root, dotted lines for the reviewer seat. Keep the layout a pure, deterministic function of the tree. CI has no Node: a JavaScript test may only be a local, dependency-free `node --test` script documented in the validation record. Accessibility: tree semantics, keyboard navigation between cards, visible focus, a name for every card.
3. **Organization tab.** Model area tabs become Browse, Organization, Session (`#/organization`; route and tab are free in `web/js/main.js` and `shell.js`). The chart shows the declared organization at once and the resolved one on request. A selected card opens its responsibilities (Definitions linked to Browse, owned areas) and profiles. Add `#/settings?pkg=<Package>` so *Change profiles* opens Settings with that Package preselected under "Profiles per Package"; *Change the team* opens the project-wide team and says it applies to every Manager. Nothing is edited on the chart. Demonstrate on a one-Package project and on Markitect's own model.
4. **Run on the chart.** The cockpit's run projection does not yet read the `run.start` binding (`RunHead` in `view.go`); extend it and the contract test, with a fallback for records without a binding. Then replace the run inspector's organization stage with the chart: state per Manager (working, waiting, ready, done), current Pass and Turn, active team members and reviewer seat, waits as badges naming whom they wait for, observation counts, and turns, models and cost per Manager from `TurnView.Usage`. Submissions animate up the reporting lines, acceptance and release down; Orchestrator controls go straight to the addressed Manager; all of it pauses with the replay. The owner card has no live lines in a closed Run; legacy records with owner exchanges still show them. Keep the rail, account, timeline, aside, record boundary, selection pinning and reduced motion. Demonstrate on real recorded runs (ask the owner for current run evidence; a verified no-change and a stopped run of 21 October exist on the owner's WSL host).
5. **Finish.** A small organization thumbnail on the hub's Model tile only if it costs no extra observation; navigation entries, README sections and a closing validation record with screenshots at 390, 860 and 1440 px.

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
model computed in internal/bootstrap behind a port (declared values from the
project observation always; resolved profiles, team and provider models on
request through the run-start facts capture); a reusable chart component
(company-style tree with the owner on top, the Orchestrator between owner and
Root, people cards, a dotted reviewer seat, collapse, pan/zoom, small-window
tree, keyboard, reduced motion); the Model area's Organization tab with
#/settings?pkg= for profile changes and the project-wide team; the run
inspector's organization stage on the same chart, reading the run.start binding;
then the finishing docs and validation. Never let the chart edit the
organization or claim more than the record shows; label declared and resolved.
Each PR: its own registry entries, focused checks, independent complete-diff
review, dated validation record. Start no provider run without the owner's grant.
```
