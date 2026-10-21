# Run organization preparation, 21 October 2026

Package 4 of the [handoff](../development/codex-cockpit-org-chart-handoff-20261021.md),
prepared after the separate package-3 commit
`22677d028532115aa32ae425aefe0d2a82294030`. Packages 2–4 are pending integration
and gated checks. A's active provider series has exclusive heavy/browser host
use; this work started no provider, container, browser, build or Go test.

The Run projection retains the recorded `run.start` binding, including per-Manager
Execute, Verify and helper profiles, identities and limits, and required checks.
It preserves absence for a historical missing binding and helper limit, and
exposes no binding before its record boundary. The producer contract test now
also inspects Apply's binding fields; it changes no producer or Run semantics.
New Go boundary/absence tests are written but **not yet run**.

The inspector uses the shared organization chart. Owner is static context above
the Orchestrator with no closed-Run connection or interactive external channel;
recorded historical owner exchanges remain inspectable. The chart reads current
Manager state, Pass, Turn, reviewer activity, active helpers, named waits,
observation kinds and adverse probes from the boundary view. Models, turns,
provider-turn counts and cost come from reported Turn Usage; missing or partial
usage remains explicit. The detail aside adds recorded profiles/team and usage.

Submissions route upward, acceptance downward and Orchestrator controls directly
to the addressed Manager, using only the recorded entry links. Animated progress
uses the inspector's clock, stays fixed on pause, becomes still with reduced
motion, and exposes a keyboard-accessible exchange list. Explicit collapse
choices survive value updates; layout changes reroute the frozen exchanges.
The old track-diagram code and its CSS/registration are removed, with its shared
activity wording moved to the new presentation module. The rail, account,
timeline, record boundaries and selection pinning retain their existing paths.

Local dependency-free tests executed:

```text
node --test internal/adapters/cockpit/org-layout.test.mjs internal/adapters/cockpit/organization-facts.test.mjs internal/adapters/cockpit/run-organization.test.mjs
```

All 12 pass. They are synthetic source-level cases: bindings vs reported usage,
reviewer/helper work, suspicions, suggestions, adverse probes, absent historical
facts, clipped exchange prefixes, closed/historical owner channels, frozen clock
progress and orthogonal upward/downward/direct routes. Module syntax and diff
checks pass. They do not show an actual rendered replay, DOM focus/collapse,
browser pause/reconnect, Go producer-contract execution or a real Run.

Independent complete-diff source review found no open issues. A final small
delta preserves exchange selection emphasis and the legend's separate Agent
message color; its syntax/whitespace checks also pass. All runtime and browser
claims remain pending the checks below.

Pending after host release: focused cockpit Go/contract tests, structural and
fixed-head accounting, actual browser source/recorded-run checks, small/low
windows and 390/860/1440 px screenshots. Existing real records may supply later
read-only UI evidence; no new provider Run is authorized by this preparation.

## Delivery after host release

Composed on actual Main `e26568f9` (#587), with the shared `.js` modules and
only this package's registrations. The final candidate passes full cockpit Go
tests (including binding producer contracts and prefix/legacy tests), 12 local
Node tests, architecture CLI, working/fixed structural accounting, product
context, documentation and diff checks. Full bootstrap and architecture package
tests and bootstrap/cockpit vet also passed on the prepared combined source;
they are separate evidence from this focused candidate's checks. Hosted CI was
not dispatched.

The actual embedded application opened read-only copies of WSL evidence:

| Run | Observed UI case |
| --- | --- |
| `20261021T121802Z-377fea2c` | 207 records, verified/no change, 18 turns, 3 Passes; failed Probe R1 at prefix 142, six final receipts and their dispositions |
| `20261021T123238Z-ee56e08d` | 126 records, incomplete after uncertain stand capture |
| `20261021T123851Z-2f742a15` | 72 records, confirmed model problems, nothing published |

Their original `events.jsonl` and `report.json` bytes were hashed before copying;
the finishing record retains those digests and screenshots. Only those two
files were copied, so unavailable receipt files/transcripts remain unavailable.
Current Markitect profiles never entered these earlier Runs. Prefix 1 hid later
usage/report facts; Manager keyboard selection, explicit collapse through a
boundary change, paused exchange rendering, reduced motion and 390/860/1440 px
were exercised without a page script error. Rail, account, timeline and aside
remain the inspector's existing surfaces.

Separate synthetic UI cases covered absent historical binding, a suspected
observation and wait, plus append/reconnect. The latter exposed an existing
pinning bug: selection at the old record end left following enabled. Selection
now clears following, including a click on the newest event; an explicit initial
boundary is pinned. The repeated Manager and newest-event cases received 207
records after offline/reconnect while retaining boundary 60, selection and the
Manager's collapse choice. Expected offline network errors are not CSP failures.
The independently reviewed correction keeps automated playback selections free
to continue. No real suspected-observation case was present in these three
records; that case remains synthetic. No provider, Apply or model session was
started, and none of these UI checks establishes owner/provider acceptance.

A non-empty-arrow replay check also found that pressing Play at a selected
send waited for the next record before starting the exchange animation loop.
Play now starts that loop immediately; the repeated browser check compares
moving arrow coordinates and then verifies that Pause freezes them.
