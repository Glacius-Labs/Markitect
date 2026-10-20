# Cockpit adapter

`markitect cockpit` serves a local web view of the runs recorded in one or more state directories. It implements the [cockpit concept](../../../docs/design/cockpit.md) in three slices: the inspector of a recorded Apply Run with organization, causal waits, exchanges, timeline, step details, candidates, evidence and replay; live observation of a running Apply through the same projections; and the cockpit entry with the project, running runs, owner questions, run history and the owner's answer. The [runtime contract](../../../docs/design/canonical-model/manager-apply-runtime.md#evidence-and-state) owns the evidence it reads.

```text
markitect cockpit --state DIR [--state DIR ...] [--repo .] [--listen 127.0.0.1:7341]
markitect cockpit --run-config PATH [--repo .]
```

## Read boundary

The cockpit reads `<state>/runs/<runID>/events.jsonl`, `report.json` and `transcripts/<turn>.jsonl`, the owner channel's `questions/` and `answers/` and `current-run`. Its one write is an owner's answer (see below). Every other route accepts only `GET`; no request starts, stops, repairs or publishes anything.

- **Consistent prefix.** Only complete lines are records. A trailing partial line is reported as a tail. A line that does not decode keeps its line number (its `seq`) and shows as unreadable, so later sequence numbers never shift. An append is read incrementally when the file keeps its identity and length and continues its last 64 consumed bytes; otherwise the file is read again from its start and, if it no longer begins with the consumed history, counted as replaced. An in-place rewrite of earlier bytes that keeps the identity and the last consumed bytes is not detected; the producer only appends.
- **Boundary.** A view is the projection of the first `seq` records (`/api/runs/{run}/view?at=N`). Nothing after record N reaches it: a later answer, verdict, outcome or report stays hidden. The report appears only at the final boundary. Details (`/detail?ref=...&at=N`) and transcripts (`/transcript?turn=...&at=N`, cut at the boundary's recorded time) obey the same boundary. `/entries` lists the whole record for navigation; the page shows its entries only up to the boundary.
- **Local only.** The server listens on a loopback address and admits only requests addressed to it, so another site cannot read evidence through a rebound host name. Pages load no external resources.

## Live observation

`/api/runs/{run}/stream` sends server-sent events over the same projections: `entries` with the records appended after the client's last one (the event id is that record's `seq` and its mark, a digest prefix of its line), `health` every two seconds, and `replaced` when the events file no longer holds the client's history, after which the stream ends. A reconnecting browser sends `Last-Event-ID` and receives exactly the records after it; a different record at that position, also from a replacement while the browser was away, ends the stream as replaced. The page applies deliveries one at a time, trims an overlap and refills a gap through `/entries`, so it always holds one complete prefix.

Health is the current observation, kept apart from the recorded boundary. A written report means the run ended. Otherwise bootstrap's state lock probe, the one `markitect answer` uses, and `current-run` decide: this run holds the directory (running, or finishing once its outcome is recorded), another run holds it, or nobody does (ended without a report). A failed probe is unknown. The probe takes the lock for an instant when nobody holds it, which the brief retries of a starting Apply and of `markitect setup` absorb, and runs at most once per two seconds and state directory. Health can be briefly behind: within the probe interval after a run starts, or while a crashed run's `current-run` remains and another process holds the lock. A written report is announced by health, not by a record, so the page reads its final view again when the run state changes; the stream closes once the report is written.

The page follows new records while the boundary is at the latest record. Pausing, selecting, scrubbing or stepping back pins the boundary; the page then counts newer records and offers an explicit return to live. Playing from a pinned boundary replays the record at the chosen speed and follows the run once it reaches the latest record. A lost link, its reconnection and a stale link (no contact for longer than the health interval) are shown with the run state of the last contact; changes of the link and of the run state are announced to assistive technology, the running seconds are not. Nothing here starts, stops or answers a run.

## Cockpit entry and the owner's answer

The entry page shows the project as structural preparation of the repository (`--repo`, default `.`) observes it in the background, bounded to two minutes: model, canonical model digest, source head, Packages and the check status. Observing again is a POST with the cockpit's own request header. The run history shows the provider steps and cost a report states; without a report, usage is unknown. Below it: the runs that hold their state directory now with their current account, the owner questions, the run history with outcome, reason, publication and source head, and which actions the cockpit offers.

An owner question is Root's question to the user in the record. It is answerable while its run is running, the record shows it open, the owner channel publishes its question file and no answer file exists. Answering takes two steps, review and confirmation; drafts survive redraws and new records. The page posts to `/api/runs/{run}/questions/{id}/answer` with a JSON body and the cockpit's own request header, which a cross-site form cannot send; a foreign `Origin` or host is refused. The server checks the question again and calls bootstrap's `writeAnswer`, the function behind `markitect answer`, which accepts only an open question of the run that holds the state directory, once. The page reports the answer file it wrote; the question shows as answered only when the run records the answer. An answer clarifies within the model and cannot change it.

Starting and stopping runs are not offered. An Apply is started with `markitect apply`; starting it from the cockpit would start provider work and needs its own decision. No stop command exists: interrupting the `markitect apply` process ends the run incomplete with its evidence.

## Projection

`Project` folds records into the view the browser shows: Passes with gates, candidates, checks and seals; Managers in the organization the run's Package paths define, with state, active turn, helpers, submissions and open findings; turns, state spans, proposals, findings, questions and problems; a short factual account; and the limitations of the record (`gaps`). `Entries` labels every record for the timeline and marks exchanges with their direction and channel: Agent communication, a submission to the decider, an Orchestrator control or acceptance event, or the owner channel.

Waits keep the Orchestrator's wording. The known forms (a decision on a proposal, the results of children or helpers, the parent's basis, a declared dependency, an answer, a contact) name the participants that must act next and the records involved; the cause follows those participants through the same view. Before a Manager's productive work starts, the Orchestrator does not record a changed wait that leaves the Manager Waiting (after its release, or after one of several declared dependencies is met). A recorded parent-basis or dependency wait whose condition the record shows fulfilled is therefore marked outdated, and the view names the still unaccepted dependencies of the accepted plan instead of guessing a reason.

What the record does not establish stays visible: delivery and reading of exchanges, the candidate's file differences, the publication's candidate digest, a missing outcome or report, unknown record kinds and unreadable lines.

## Browser

The page is plain HTML, CSS and JavaScript modules without a build step, embedded in the binary and packaged with the source. Replay moves the boundary through the record with play, pause, speed, record and exchange steps; recorded gaps are compressed and bookkeeping records take no playback time. Selecting anything pauses the presentation; pausing never affects a run. Arrows illustrate recorded sends along the tree, pause with the replay and open their record; with reduced motion they are drawn still. Keyboard: Space, arrows, Shift with arrows, Home, End, `[` `]`, Esc and `?`. Narrow windows show the organization as a list.
