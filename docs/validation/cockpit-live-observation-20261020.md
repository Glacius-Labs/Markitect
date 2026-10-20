# Cockpit live observation — 20 October 2026

**Scope:** the second slice of the [cockpit](../design/cockpit.md): live observation in `markitect cockpit` through the same projections as the [recorded run inspector](cockpit-run-inspector-20261020.md), implemented in the [cockpit adapter](../../internal/adapters/cockpit/README.md#live-observation). No active Apply was observed: that needs a provider run, and no applicable grant was used. This record separates source checks, a demonstration with re-fed recorded evidence and what remains unproved.

## What was built

- A server-sent event stream per run: appended records with the last record's `seq` as event id, the run's health every two seconds, and an end event when the events file is replaced. A reconnect resumes after `Last-Event-ID`.
- Incremental reading: an append to the same file is read from just before the consumed end, checking the seam; another file, a shorter file or a broken seam is read again from its start and counted as replaced.
- Health kept apart from the record: report written (ended), this run holds the state lock (running, or finishing once its outcome is recorded), another run or nobody holds it (ended without a report), or unknown when the lock probe fails. The probe is the answer command's, at most once per two seconds and state directory.
- The page follows new records while at the latest boundary and catches up at most three playback seconds behind. Pausing, selecting, scrubbing or stepping back pins the boundary; the page counts the newer records and offers "Return to live"; End or playing on to the latest record resumes following. The link's state (connected, reconnecting, stale after six seconds without contact, stopped after a replacement) and the run state of the last contact are shown. A record that ends without outcome while nobody holds the state directory is stated as an ended run whose later effects are unknown.

## Demonstration with re-fed recorded runs

A feeder outside the repository held the state lock of a new state directory, named its run in `current-run` as an Apply does, appended the lines of a recorded run with compressed timing, copied each transcript at its turn's end and wrote the report last, then released the lock. It is not an Apply: no Orchestrator or Agent ran, and the timing is not the original. Headless Chrome observed the cockpit served from the slice's source.

| Run fed | What was observed |
| --- | --- |
| S-N 20261020T123344Z-66475687 (330 records) | Following from record 31 onward; selecting stockroom pinned record 52 while 68 newer records arrived and "Return to live (68 newer)" appeared; stopping the cockpit server showed "connection lost, reconnecting" with the last known run state; after a restart the link reconnected; returning to live followed again. The page then held 166 rows for the server's 166 records, unique and contiguous. |
| S-X 20261020T113103Z-1cbd0747, cut after 45 records | While fed: live and following. After the feeder stopped without a report and released the lock: "Run ended without a report", the notice that later effects are unknown, and the record's open owner question Q2. |
| D-N Sonnet 20261020T123344Z-49406489 (621 records, six Managers) | Following with parallel Managers while about eight records per second arrived; the view trailed the server by about twenty records, within the catch-up bound of three playback seconds; two steps back pinned record 334 with 80 newer; End resumed following. |

No browser console errors were logged.

## Checks

`go test ./internal/adapters/cockpit ./internal/adapters/cli`, `go vet`, `gofmt`, the architecture gate, `markitect check` and the guarded model view write passed locally on Windows. The stream tests append to a real file while a stream is open and check single delivery, resumption after `Last-Event-ID`, the end on replacement, health classification and the probe's rate limit; the evidence tests check an append read incrementally and a renamed-in file with another beginning read again.

## Not established

Observation of an active Apply on Linux or in WSL, where the probe and the producer's lock run natively; the lock probe across a WSL path from Windows; a long-running stream's behavior over hours; and owner use. An in-place rewrite of earlier bytes that keeps the file's identity and the last 64 consumed bytes is not detected; the producer only appends.
