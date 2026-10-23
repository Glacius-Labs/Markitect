# Cockpit polish — 21 October 2026

**Scope:** the owner asked for the hub-and-areas cockpit to be refined: less explanatory prose, a top bar that fits on a phone, a hub that fits a laptop screen, and one consistent look for the project cards, the session start, decisions and the model browser. This change is presentation only. It touches the browser part of the [cockpit adapter](../../internal/adapters/cockpit/README.md) (`web/index.html` is unchanged) and no route, Go code or behaviour: what writes, what spends provider usage and the two-step confirmations are as the earlier records describe ([areas](cockpit-areas-20261021.md), [projects](cockpit-projects-20261021.md), [model sessions](cockpit-model-sessions-20261020.md)). The run inspector and the run list belong to a parallel change and are untouched.

## What changed

- **One page header.** Every page of its own has a title, one short sentence and, where more must stay reachable, a collapsed *How this works* (`pagehead`, `about` and `empty` in `dom.js`). The long leads of Projects, Settings and the session start moved into these disclosures or into tooltips; the statements of what writes (nothing is committed, no Apply starts, a removal forgets the entry only) and what spends provider usage (every prompt is one Agent turn) stay reachable, the provider-usage note stays visible at the session's start action, and the full warning still appears at its second step.
- **Top bar.** The notification control is a bell icon with a status dot; its accessible name is "Notifications off", "Notifications on" or "Notifications blocked" and the tooltip says what a click does. Below 560 px the status chip drops its leading word ("failed" instead of "Check failed"), so project name, way back, area emblem, tabs, one chip and the bell fit at 390 px.
- **Hub.** Tighter rhythm: at 1440×900 the hero, the four tiles and the key hint fit without scrolling, also with a waiting proposal. Tile texts are one line; a tile without figures names what its area holds (Browse and Session, Budget, Profiles and Team) so no middle is empty, and the Decisions tile shows waiting and held proposals. From 960 px down the tiles stay two columns until 760 px.
- **Projects.** Cards in the tiles' language: an initial emblem, the name, a lamp saying "Set up" or "Not set up", the open mark, repository and runtime, then the actions with *Remove…* at the end. *Keep in the project list*, *Add a project* and the setup form are cards with the same header; the two-step removal and the two-step setup are unchanged.
- **Session.** Without a session the page is one column: one sentence, a four-step stepper, a collapsed *How it works*, the composer and the provider-usage note by the start button. With a session it is the split view as before.
- **Decisions and proposal.** Block headers are one small uppercase style, blocks are separated by hairlines, empty states are a title and one line.
- **Model browser.** Denser tree and list; the overview's figures are small cards and the kinds are one card; section headers of a Definition share the block-header style, with the registry table's path column wrapping instead of squeezing the reason.
- **General.** Inline style objects of the changed pages became classes; focus rings inside scrolling lists are drawn inside their box; settings keep their action bar visible while scrolling.

## Checks

`node --check` of every changed module, `go test ./internal/adapters/cockpit/...` (the embedded files), `markitect check --repo .` and `git diff --check` passed locally on Windows.

Driven in headless Chrome over the DevTools protocol against cockpits built before and after the change, in project mode with a temporary `%AppData%`, in flag mode with run evidence (`--state`), and against a clean clone of the repository for a passing check, a loadable settings form and a real settings proposal. Measured at 390, 760, 860, 1100 and 1440 px on the hub, Model, Session, Decisions, a proposal, Runs, Settings and Projects: no element of the top bar crosses the right edge, none overlaps another and the page does not scroll sideways. Over a poll interval (nine seconds) the hub, the decisions list and the project cards did not move and kept focus on the control that had it. The removal confirmation, its cancellation and *Keep in the project list* (flag-opened state) worked as before.

## Screenshots taken

Before and after pairs at 1440 px (hub with a failed and a passing check, hub with a waiting proposal, projects with the flag-opened offer, with two cards and with the setup form, session start, settings, decisions with a proposal, model browser and a Definition, runs) and at 390, 760, 860 and 1100 px for the hub and the top bar of the model area. They were saved outside the repository.

## Not established

Owner use of the new look; the session page against a live session in this build (the empty start was shown with the sessions interface answered by a script in the page, no provider turn ran); a project that is really set up (the card was shown with a list entry that names a runtime directory); screen-reader and keyboard operation beyond focus staying on its control; browsers other than Chrome. Polish left to the run inspector's change: the Runs page lead is still a long paragraph.
