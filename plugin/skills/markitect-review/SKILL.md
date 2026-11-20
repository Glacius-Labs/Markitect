---
name: markitect-review
description: The review job of a Markitect project - a spot check in which a fresh reviewer confirms a few confirmed documents again at random, or all of them on request, so that a verdict that contradicts the old one shows. Use when the user or a runner asks for a spot check or a full review.
---
Run the review job in the frame of a job, `${CLAUDE_PLUGIN_ROOT}/jobs.md`. The job is `review`. The review changes no artifact and no spec; it binds verdicts and writes reports.

1. **Choose.** `markitect table` lists every document with its verdict. Take a few documents whose verdict is `ok`, at random, or every document with a row when the user asks for a full review.
2. **Fingerprints.** For each, `markitect show DOCUMENT` prints `DOCUMENT@FINGERPRINT` on its first line, and its artifacts and counter-checks.
3. **Confirm.** Start `markitect:reviewer` with the entries as `DOCUMENT@FINGERPRINT`, each with its artifacts and counter-checks as `show` names them. It judges each document afresh and binds its verdict.
4. **Report.** Write the reports of its answer; those on the spec go through triage on this branch, as the frame says. A verdict that differs from the old one on a document that did not change is a contradiction, which `markitect figures` counts; say which.

Then land as the frame says. A new `fail` keeps the branch from landing: the build job repairs it, so run the `markitect:markitect-build` skill on this branch and say so.
