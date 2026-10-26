# Owner directions: development pace and Linux

Date: 26 October 2026. Source: direct owner messages in the Claude Code session "Markitect evaluation and canonical model design", written in English. The coordinator read them in that session's transcript. The owner statements are verbatim; the session's answers are summarized and labeled. The recorded meaning is the recorder's and adds no decision.

## 1. Speed things up

> Is there anything that is slowing us down in development?

Context (summary by the recorder): the session named five slowdowns:
- the shared machine, where native runs blocked local Go for hours, so code was written blind and debugged through hosted CI rounds of 30 to 60 minutes;
- the bookkeeping of Markitect's own self-model;
- coordination overhead: a verbatim record, register entry and PR per decision, single writers for shared files, and parallel PRs colliding;
- building before deciding;
- too many parallel threads.
It proposed retiring the self-model soon, separating experiments from development, batching the day's decisions into one record PR, fewer parallel threads, and settling designs before code.

> Then lets speed things up here

## 2. The machine

Context (summary by the recorder): the session asked whether experiments should run only at night or on a separate machine.

> We only have this machine, so we should do it here and ask if we really need a quiet machine for those. Or only do them on special milestones or something

## 3. CI

> Why cant we make the CI just faster? Or disable it. Waiting for 1h for each PR is not acceptable.

Context (summary by the recorder): the session measured one full run at 3 minutes for Linux quality, 4.5 minutes for the race detector and 38.5 minutes for the Windows job. It proposed Linux-only CI for pull requests and Windows at milestones.

## 4. Linux, Docker and a nightly run

> Now lets talk about wether Markitect would be better in Linux than on Windows. If so cant we use Docker for that? For CI we should definately make it a nightly check instead of wating for hours each time. Also for test runs linux would be much faster as i understand you. Also i might imagine, that Markitect Apply might be something that can be run in a Container or in a CI Agent which is Linux instead of running it locally. Or we use Docker for that. For manipulating the model however it should also work on Windows. For Context, one potential Workflow on developing with Markitect might be only modifying the Model and creating a PR. Then the PR gets checked via a Model Assessment run and apply produces the Artifacts in the CI or something like that.

Context (summary by the recorder): the session proposed a nightly Windows run on Main, Apply's host part inside a Linux image with the model commands staying on Windows, and Apply in CI only after V1's local Apply is proven. It noted the open login question for CI machines.

> Yes: add a nightly scheduled CI run on main with Windows, keep PR runs Linux-only, and coordinate the ci.yaml change with the Coordinator.

## 5. Windows tests removed

Context (summary by the recorder): the session reported that a pull request waited only for its Windows CI job, which had run for about 37 minutes.

> I dont care, the windows tests as we discussed should be removed anyways. I have lost patience in waiting for windows. I think we should move to the Linux Docker way. I already wait for days for all the runs

## 6. Repository and login in the Apply container

Context (summary by the recorder): the session put two open questions of the Linux Docker way to the owner, with the coordinator's recommendation: how the Apply container reaches the owner's repository, and how it gets the provider login.

> I accept both: (a) the runner clones from a read-only mount of my checkout and hands the verified result back as a branch pushed into my repository; (b) my provider CLI login is mounted read-only, CLI login only, no API keys.

## Recorded meaning

Recorder's interpretation, as the coordinator put it into effect the same evening:

- **Retire the self-model now** (slice R3a of [DEC-076](../register.md#dec-076-markitect-is-developed-as-a-regular-software-project)), **record the day's owner decisions in one batched PR**, and **keep few parallel writers**.
- **A quiet machine only for counted study runs,** where time and cost are measured endpoints, run as batches at milestones. Native controls, smokes and probes run alongside development and record the host load as a limit. During a native run nobody runs the full test suite, the Windows Git suite or Docker image builds; targeted package tests and builds are fine.
- **CI runs on Linux only:** Linux quality and the race detector. The nightly Windows run of statement 4 was superseded the same evening by statement 5, which removes the Windows job.
- **Markitect moves to the Linux Docker way:** its execution, meaning the Apply host, tests and CI, runs on Linux. Windows stays the owner's host for model commands and for starting Apply through Docker Desktop. The Apply runner clones from a read-only mount of the owner's checkout and hands the verified result back as a branch pushed into that repository; the provider CLI login is mounted read-only, with no API keys. Whether Apply later runs in CI is still open.
