# Cockpit hub and areas — 21 October 2026

**Scope:** the owner found the [workbench](cockpit-workbench-20261021.md) too crowded ("zu umfangreich und vollgestopft", one quickly loses the overview) and asked for a modernized design with normal modern cards and a simpler structure: clear areas to navigate into, allowed to feel more like a game than a classic application. This change rebuilds the browser part of the [cockpit adapter](../../internal/adapters/cockpit/README.md#hub-and-areas) as a hub with four areas. It changes no route of the server and no write path: the owner's answer, proposals and model sessions behave as their records describe ([entry](cockpit-entry-20261020.md), [configuration](cockpit-configuration-20261020.md), [model sessions](cockpit-model-sessions-20261020.md)).

## What changed

- The desk rail with six places and the status bar are gone. The start page is a hub: the project with its check and *Observe again*, the one next step that needs the owner, and four large area tiles — Model, Runs, Decisions, Settings — each with its live figures, a footer line and a glow while it waits for the owner.
- Session moved into the Model area as its second tab beside Browse. Owner questions and proposals moved together into Decisions, waiting ones first; without a choice the first waiting item opens. The Runs area shows the runs that run now above the history; a run opens in the inspector inside the area.
- Navigation works like a game menu: the digits 1 to 4 enter an area from the hub, Escape leads one level up — from a run to Runs, from a Definition to its Package and on to all Definitions, from an area to the hub (the inspector keeps Escape while a transcript, the key help or a selection is open). The top bar shows the way back, the area with its tabs and the open item, what waits for the owner, running runs and the check.
- Presentation: rounded cards with soft depth in the Glacius Labs colors (ink ground, ice accent, quiet status colors), an accent per area, round lamps, pill chips and segmented tabs. The earlier no-curve rule and the corner ticks are dropped for this request. Motion respects reduced motion. Old addresses `#/proposals` lead to Decisions.

## Demonstration

A cockpit built from this branch served a scratch clone of the repository and two recorded runs on Windows, in headless Chrome. The hub (while observing and observed), Model Browse and a Definition, the Session tab without a run configuration, Decisions with a real settings proposal (opened automatically, with check, difference and decision) and with a recorded owner question, the Runs area and a recorded run in the inspector, and the settings form rendered without console errors. Keys were exercised: Escape from Decisions reached the hub, `2` on the hub entered Runs, and in the inspector the first Escape cleared the selection and the second led to Runs. Focus on a Decisions row and on the hub's next-step button stayed through two polls, and the Waiting/All filter kept focus when switched. At 390 px the hub, Decisions and the inspector fit the width (measured: the top bar's way back, area emblem and tabs end before the status chip), and the inspector's replay controls stay at the bottom of the screen while the page scrolls.

## Checks

`go test` of the cockpit package, `go vet`, `markitect check` and the documentation link check passed locally on Windows; the embedded-file test confirms that every module the page imports is embedded.

The independent review found no high-severity defect. Its findings are addressed: parts of the hub, the Decisions list and the top bar no longer redraw on every poll (focus and announcements survive), the filter buttons keep focus, the inspector's sticky replay controls work on phones again, the area name stays in the accessibility tree on small windows, the hub has one top heading, a question that this cockpit cannot answer says how to answer it, entry animations leave no lasting stacking context, and the Runs tile counts only readable runs.

## Not established

Owner use of the hub and areas; a live session and a live run in this build; screen-reader use beyond the existing landmarks, labels and announcements.
