# Cockpit run control — 21 October 2026

**Scope:** the owner adopted starting and stopping Apply from the cockpit ([DEC-066](../concepts/register.md#dec-066-the-cockpit-is-markitects-desktop-app-and-starts-and-stops-apply), [design](../design/cockpit.md#the-cockpit-as-desktop-app)). This change implements it for a cockpit that has `--run-config`, `--runtime-profile` and `--repo`, as described in the [cockpit adapter](../../internal/adapters/cockpit/README.md#starting-and-stopping-a-run). It changes neither Apply, the run engine, the run configuration, the Runtime profile nor the owner channel. The project list, setup through the cockpit, the window and notifications of the desktop app are not part of it.

## What was built

- A run controller in the cockpit adapter starts the execution function of `markitect apply` in a goroutine with a context derived from the cockpit's, keeps at most one such run, cancels the context on stop, keeps the handle until the function returned and the result (outcome, error, report path) until the next start, and cancels and waits when the cockpit ends.
- Bootstrap composes it from the existing `executionService.Execute` with the cockpit's repository, model and dependency selection, the Runtime profile, the run configuration and the product build identity, refuses a start while the state lock or the proposal publication fence holds the state directory, and reads the confirmation facts from the same captured settings and profile decoders Apply uses (execution budget, run configuration deadline, Root's Execute Agent profile and its model).
- Routes `GET /api/run-control` (with `?facts=1` for the confirmation), `POST /api/run-control/start` and `/stop`, the writes under the cockpit's own request header, origin and host rules.
- The Runs area has a card with a two-step start (the facts and a provider usage warning), the run started here with its state, a link to the inspector and a two-step stop; the inspector's head offers the same stop for that run only. Without the flags the card says how to enable starting.

## Checks

- Cockpit adapter tests with an injected execution function: start, running once `current-run` names the run, stop cancels the context and the result with its report path is kept; refusal while the state is held and while a run is active; the execution's own refusal shows as the ended run's error without a link; a cockpit that stops cancels its run and waits until the function returned and then starts nothing; a panicking function ends the handle; the facts are read only when asked for; every write needs its purpose header and own origin and host, GET reaches no write route, and a cockpit without a controller answers 501.
- Bootstrap tests on a repository with a Runtime prepared by setup and a fake execution function: the facts show the budget, the profile `markitect-agent` with its model, and with uncommitted changes the refusal instead; a start passes Apply's inputs (run configuration, selection, Runtime profile, build) and a stop ends its context and shows the report path; a publication fence and a held state lock refuse a start before the function is called; a stopping cockpit ends its run.
- The cockpit command passes the product build identity (`internal/adapters/cli/cockpit_test.go`).
- `gofmt`, `go vet` and `go test` of the cockpit, bootstrap and CLI packages and of the architecture tests, `node --check` of every page module, `markitect check` on this repository and the documentation link check.

## Review follow-up

An independent review of the first version found no blocker; its points are addressed:

- **Stale confirmation:** the start names the head of its confirmation and is refused when the committed HEAD differs. A tiny window between that check and `Execute`'s own capture remains.
- **Shutdown:** `Wait` closes the controller, so no start follows it. A hung `Execute` keeps the cockpit from exiting after the first interrupt; a second interrupt ends the process without the run's report. This was not exercised with a real hang.
- **Stop in preparation:** a stop before evidence exists ends incomplete with an explicit statement, not as a refusal.
- **Attribution:** a run is named from `current-run` only after `Execute`'s lock wait has passed with `Execute` still running; an ended run is named only from its evidence. A foreign Apply that won the lock between the probe and `Execute` is therefore not shown as the cockpit's run. The remaining assumption is that `Execute` obtains the lock or refuses within the wait.
- **Facts:** Root's Execute profile is named with its model, or with the statement that the Runtime profile states none, and distinct Execute profiles of other Managers (a per-Package override) are listed. Concurrent reads share one capture. After a start the focus moves to the card's heading.

New tests cover a stale head (also on a repository with a later commit, where the facts head equals the repository HEAD), a closed controller, a stop in preparation, the attribution delay, a refused start never taking a foreign run, shared fact reads and the distinct profiles of a per-Package override.

## Page demonstration

A cockpit built from this branch served a synthetic controller (a fake execution function that names a recorded run after two seconds and ends two seconds after its stop) and a recorded run, in headless Chrome driven over the debugging protocol. Observed: the card without a run; the confirmation with its facts after a short reading state and focus on *Start the run*; the starting state, then running with the run's link once it named itself; the stop confirmation with the sentence that the run ends incomplete, keeps evidence and report and restores nothing; the inspector of that run showing the same stop in its head and hiding it elsewhere; the stopping state; and the ended result with *Start a run* again. The console stayed empty. A redraw caused by a state change keeps the focused button. The first run of this demonstration found that a `null` child was drawn as the text "null"; the widgets now add children through the page's helper.

## Not established

- A real run started and stopped from the cockpit with a provider, container engine and credentials: the execution function was always a fake here, so the report written after a cancel, the evidence of an interrupted run and the cost of the stop are as `markitect apply` already shows them, not observed through the cockpit.
- The state probe against a real second process on Linux, and a start racing a publication that begins right after the probe (the execution's own lock then refuses and its error shows).
- That the cockpit warns before it ends with a run active: the page says that ending the cockpit stops the run, but the cockpit process cannot ask when it is interrupted. The run identity relies on `current-run`; before the run names itself the page shows it as starting.
- Owner use, and the model named in the confirmation being the one a provider actually serves.
