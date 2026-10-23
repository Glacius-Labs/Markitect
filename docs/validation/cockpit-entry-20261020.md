# Cockpit entry and owner answer — 20 October 2026

**Scope:** the third slice of the [cockpit](../design/cockpit.md): the entry page with project identity, running runs, owner questions, run history and outcomes, and the owner's answer as the cockpit's one action, implemented in the [cockpit adapter](https://github.com/Glacius-Labs/Markitect/blob/447ac3b152de79b52de80513e14dc94349a060bd/internal/adapters/cockpit/README.md#cockpit-entry-and-the-owners-answer). It builds on the [inspector](cockpit-run-inspector-20261020.md) and [live observation](cockpit-live-observation-20261020.md). No active Apply and no provider run were involved.

## What was built

- `markitect cockpit --repo PATH` (default `.`) observes the repository by structural preparation, as `markitect check` does, in the background: model, canonical model digest, source head, Packages and check status. A request never waits for it.
- The entry lists runs that hold their state directory now with their current account, owner questions with what the owner channel shows (published, answer file written, run state) and why each is or is not answerable, the run history with outcome, reason, publication and source head, and which actions the cockpit offers.
- The answer action: a review and a confirmation step, a JSON POST with the cockpit's own request header, refusal of foreign hosts and origins, a server-side check that the question is open, published, unanswered and its run running, then bootstrap's `writeAnswer`, the function behind `markitect answer`. The page reports the answer file it wrote and says the run has not recorded it yet; the question shows as answered only when the record shows it.
- Start is not offered: it would start provider work and needs its own decision. Stop is not offered: no stop command exists; the page says how a run is interrupted and that pausing a view never stops one.

## Demonstration

The cockpit served the S-N and Haiku D-N state directories and a new state directory into which a feeder outside the repository re-fed the recorded S-X run 20261020T113103Z-1cbd0747. The feeder held the state lock, named the run in `current-run`, published Root's question Q2 as a question file in the owner channel's format when the record asked it, and then waited for an answer file. It is not an Apply: no Orchestrator or Agent ran, and the records after the wait are the original record, including its original answer text.

- The entry showed the running feed with its account, Q2 as waiting for an answer and the Haiku D-N run's Q10 as not answerable because its report is written.
- Entering a test answer, reviewing and confirming wrote `answers/Q2.txt` through `writeAnswer`; the page reported the written file without claiming the run had recorded it. The feeder read the answer file and continued; afterwards no question waited.
- The project panel showed the observation in progress and then the model, digest, head and 24 Packages. The check status was failed at that time because the slice's new files were not yet registered; the record does not claim a passing check for that working tree. Observed again at the committed head 9e1897ab, the check passed.

No browser console errors were logged.

## Checks

`go test` of the cockpit, CLI and bootstrap packages, `go vet`, `gofmt`, the architecture gate, `markitect check` and the guarded model view write passed locally on Windows. The tests cover answers reaching the injected owner channel only for a waiting question and only with the cockpit's request, refusals for a foreign host or origin, a text/plain body, blank, oversized or malformed text, an unknown, lapsed, withdrawn or already answered question and a run that does not run, the owner question list, the background project observation behind its own request, its failures and panics, and the CLI's repository default.

After the independent review: a draft answer survived two entry polls, the move to the run's inspector and new records arriving there; the owner question list projects each run's record once per change instead of on every poll.

## Not established

An answer to a real Apply's question and the Orchestrator recording it; owner use of the entry page; start and stop from the cockpit, which are not implemented.
