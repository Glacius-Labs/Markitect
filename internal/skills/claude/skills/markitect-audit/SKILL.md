---
name: markitect-audit
description: Audit the spec itself. A fresh markitect-auditor agent reads the spec as a newcomer and reports contradictions, gaps, unclear and duplicated statements to the owner; it changes nothing and records nothing. Use before a spec is built, after a large change of it, or when the user asks to check the spec rather than the code.
---
Start the markitect-auditor agent with the message "Audit the spec.", and add the Packages or Definitions the user asked to focus on, if any. Change no file yourself.

When it has ended, give the user its findings, the most serious first, in plain words, each with its place and its question. They are for the owner to settle in the spec, with the markitect-design skill or by hand: nothing is recorded and nothing is committed.
