---
name: markitect-audit
description: Audit the spec itself. A fresh markitect-auditor agent reads the spec as a newcomer and reports contradictions, gaps, unclear and duplicated statements to the owner; a large spec is audited in parallel, one auditor per Package and one across them. It changes nothing and records nothing. Use before a spec is built, after a large change of it, or when the user asks to check the spec rather than the code.
allowed-tools: Bash(markitect status --json), Bash(markitect config strategy.audit)
---
Run `markitect config strategy.audit` for how to audit: `sequential`, `parallel`, `auto` or the name of a custom workflow. With `auto`, choose: parallel when the spec has more than about 30 Definitions in at least two Packages (`markitect status --json` counts them per Package), otherwise sequential. Change no file yourself.

- **Sequential:** start the markitect-auditor agent with the message "Audit the spec.", and add the Packages or Definitions the user asked to focus on, if any.
- **Parallel, or a custom workflow:** start the workflow with the Workflow tool, `markitect-parallel` for parallel or the custom workflow by its name, with the args `{"mode": "audit", "focus": "<what the user asked to focus on, or empty>"}`. It starts one auditor per Package and one that looks only for contradictions between Packages, and returns their reports.

When the audit has ended, give the user its findings, the most serious first, in plain words, each with its place and its question; merge findings that several auditors report. They are for the owner to settle in the spec, with the markitect-design skill or by hand: nothing is recorded and nothing is committed.
