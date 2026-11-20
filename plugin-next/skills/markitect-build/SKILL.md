---
name: markitect-build
description: The build job of a Markitect project. On a branch of its own, a counter-checker derives counter-checks for the changed documents, a builder realizes what markitect-next status lists and a fresh reviewer confirms each entry, round after round until the list is settled; the branch lands when the landing check and the tests pass. Use when the user or a runner asks to build, or another job hands a branch over to it.
---
Run the build job in the frame of a job, `${CLAUDE_PLUGIN_ROOT}/jobs.md`, which says how the branch, the steps, the reports and the landing go. The job is `build`. On a branch the user names, such as the branch of a spec change, build there; otherwise on a branch of its own.

**Rounds.** Run `markitect-next status --json` in the job folder. While it exits 1, run a round of three steps and write the reports of their answers:

1. `markitect-next:counter-checker`, when the list has documents in `status.changed` and `markitect-next settings --json` names a folder in `settings.counter_checks`: the message lists those documents, without their artifacts.
2. `markitect-next:builder`: the message lists the entries of `status.changed` and the files of `status.unexplained` as `status` prints them.
3. `markitect-next:reviewer`: run `markitect-next status --json` again; the message lists every entry of `status.changed` and `status.suspect` as `DOCUMENT@FINGERPRINT`, with what the builder answered about the files that realize each. Suspect documents go to the reviewer alone.

When a round wrote reports on the spec, triage runs on this branch before the next round and closes each report this branch answers, as the frame says. Documents waiting on a report are not built; those waiting on the owner leave `status` at exit 0 and land unbuilt. While `check` finds errors, `status` gives no list: report them and stop.

**Slices.** When the list is too long for one builder, cut it into slices by the top folder of the spec, in no order, and run the rounds for one slice after another on the job's branch, each message listing only the slice's entries. A later slice may make a document of an earlier one suspect; the rounds bring it back.

**The end of the rounds.** After each round, count the entries of `status.changed`, `status.suspect` and `status.unexplained` and those of `status.waiting` whose `status.waiting.counts` is true. Stop when three rounds in a row leave the count no shorter than the shortest it has been. Write one report for each document or file left, with Source `job build reviewer, DAY`, its last `fail` reason or what the agents said as the finding, and the violated property they name, or `none`.

**Prevention.** When the build answers a report, write its Prevention section: the counter-check the counter-checker added for it, or why none fits.

Then land as the frame says, with `markitect-next status --target TARGETDIR` and the command `markitect-next settings --json` names in `settings.tests`.
