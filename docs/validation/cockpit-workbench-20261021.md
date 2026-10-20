# Cockpit workbench — 21 October 2026

**Scope:** the owner judged the cockpit's separate pages too sparse and asked for a complete, professional interface ("Professionelles Werkzeug": a dense, calm work surface with a sidebar, panels and keyboard use). This change rebuilds the [cockpit adapter's](../../internal/adapters/cockpit/README.md#the-workbench) browser part as one workbench and adds the model browser. Asked to follow the Glacius Labs brand and not to use green, the workbench takes the design language of the Glacius Labs design system (the organization's Design-System repository; the website did not resolve at the time): an ink ground, ice as the scarce accent, hairlines, no rounded edges, uppercase micro-labels, mono coordinate stamps, quiet status colors and corner registration ticks. The brand fonts DM Sans and JetBrains Mono are used where installed; the page loads nothing from outside the cockpit. It changes no write path: the owner's answer, proposals and model sessions behave as their records describe ([entry](cockpit-entry-20261020.md), [configuration](cockpit-configuration-20261020.md), [model sessions](cockpit-model-sessions-20261020.md)).

## What was built

- A desk rail with the places Overview, Model, Session, Proposals, Runs and Settings, each with the lamp of its live state, and a status bar with the place and the project's check, source head, session and running runs. One poll every four seconds while the page is visible serves the rail, the status bar, the overview, the runs and the proposals list; the session, the model browser and the run inspector read on their own.
- Overview as a dashboard; the run inspector inside the workbench; the proposals inbox with the chosen proposal beside the list; the session as a split view with the latest proposal and its decision on the right; settings in the same frame.
- The model browser over `GET /api/model`: the checked model and the registry from the project observation, Packages, Definitions with a filter, a Definition's body cut into segments at the links the model check located (so no other bracketed text becomes a link), its fields, the files that realize it and the Definitions that link to it. A model or registry that did not pass is said so.

## Demonstration

A cockpit built from this branch served a scratch clone of the repository and recorded runs on Windows, in headless Chrome. The overview, the model browser (40 Definitions, 1,409 registered files, `product/cockpit/accepted-proposals` with 13 realizing files), the runs page and a recorded run in the inspector, the proposals inbox and a real settings proposal with its decision rendered without console errors. The session page ran against a scripted session interface injected into the page (no Agent, no provider), with the real settings proposal shown in its right pane. At 390 px the layout held without horizontal scroll and the menu button opened the rail as a drawer that a chosen place, a click beside it or Escape closes; the closed drawer leaves the tab order.

## Checks

`go test` of the cockpit, bootstrap and tooling packages, `go vet`, `gofmt`, `markitect check` and the documentation link check passed locally on Windows. The tests cover the model route (pending while the observation runs, served from it, refused without a repository, kept out of the project summary), body segments that follow only located links (a `[[...]]` inside a Markdown link label stays text) and fall back to text without locations, and, on the bootstrap fixture, a project observation that carries the checked model with kind, body and source and the registry.

The independent review required located body links instead of a pattern in the page, a visible menu button on small windows and no stray text while a session prepares; these and its smaller findings on style leaks into the run inspector, landmarks, the skip link, the closed drawer, unknown counts and the registry status are addressed.

## Not established

Owner use of the workbench; the session page against a live session in this build (the real session ran on the previous build); keyboard operation beyond the existing inspector keys, the filter's `/` and Ctrl+Enter in the session composer.
