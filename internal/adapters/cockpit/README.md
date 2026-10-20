# Cockpit adapter

`markitect cockpit` serves a local web view of the runs recorded in one or more state directories. It implements the first slice of the [cockpit concept](../../../docs/design/cockpit.md): the inspector of a recorded Apply Run with organization, causal waits, exchanges, timeline, step details, candidates, evidence and replay. The [runtime contract](../../../docs/design/canonical-model/manager-apply-runtime.md#evidence-and-state) owns the evidence it reads.

```text
markitect cockpit --state DIR [--state DIR ...] [--listen 127.0.0.1:7341]
markitect cockpit --run-config PATH
```

## Read boundary

The cockpit reads `<state>/runs/<runID>/events.jsonl`, `report.json` and `transcripts/<turn>.jsonl`, nothing else, and writes nothing. A request never starts, answers, stops, repairs or publishes anything; the HTTP interface accepts only `GET`.

- **Consistent prefix.** Only complete lines are records. A trailing partial line is reported as a tail. A line that does not decode keeps its line number (its `seq`) and shows as unreadable, so later sequence numbers never shift. A file that no longer starts with the bytes read before counts as replaced and is read again from its start, never spliced.
- **Boundary.** A view is the projection of the first `seq` records (`/api/runs/{run}/view?at=N`). Nothing after record N reaches it: a later answer, verdict, outcome or report stays hidden. The report appears only at the final boundary. Details (`/detail?ref=...&at=N`) and transcripts (`/transcript?turn=...&at=N`, cut at the boundary's recorded time) obey the same boundary. `/entries` lists the whole record for navigation; the page shows its entries only up to the boundary.
- **Local only.** The server listens on a loopback address and admits only requests addressed to it, so another site cannot read evidence through a rebound host name. Pages load no external resources.

## Projection

`Project` folds records into the view the browser shows: Passes with gates, candidates, checks and seals; Managers in the organization the run's Package paths define, with state, active turn, helpers, submissions and open findings; turns, state spans, proposals, findings, questions and problems; a short factual account; and the limitations of the record (`gaps`). `Entries` labels every record for the timeline and marks exchanges with their direction and channel: Agent communication, a submission to the decider, an Orchestrator control or acceptance event, or the owner channel.

Waits keep the Orchestrator's wording. The known forms (a decision on a proposal, the results of children or helpers, the parent's basis, a declared dependency, an answer, a contact) name the participants that must act next and the records involved; the cause follows those participants through the same view. Before a Manager's productive work starts, the Orchestrator does not record a changed wait that leaves the Manager Waiting (after its release, or after one of several declared dependencies is met). A recorded parent-basis or dependency wait whose condition the record shows fulfilled is therefore marked outdated, and the view names the still unaccepted dependencies of the accepted plan instead of guessing a reason.

What the record does not establish stays visible: delivery and reading of exchanges, the candidate's file differences, the publication's candidate digest, a missing outcome or report, unknown record kinds and unreadable lines.

## Browser

The page is plain HTML, CSS and JavaScript modules without a build step, embedded in the binary and packaged with the source. Replay moves the boundary through the record with play, pause, speed, record and exchange steps; recorded gaps are compressed and bookkeeping records take no playback time. Selecting anything pauses the presentation; pausing never affects a run. Arrows illustrate recorded sends along the tree, pause with the replay and open their record; with reduced motion they are drawn still. Keyboard: Space, arrows, Shift with arrows, Home, End, `[` `]`, Esc and `?`. Narrow windows show the organization as a list.
