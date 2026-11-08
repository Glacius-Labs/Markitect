---
name: markitect-audit
description: Audit the spec itself. A fresh markitect-auditor agent reads the spec as a newcomer and reports contradictions, gaps, unclear and duplicated statements and missing links to the owner, triaging the unlinked mentions markitect vet finds; it changes nothing and records nothing. Use before a spec is built, after a large change of it, or when the user asks to check the spec rather than the code.
---
Start the markitect-auditor agent with the message "Audit the spec, and triage the unlinked mentions markitect vet --strict lists.", and add the Packages or Definitions the user asked to focus on, if any. Change no file yourself.

When it has ended, give the user its findings, the most serious first, in plain words, each with its place and its question. The missing links it found come as one list, each with the Definition to link: the owner adds them, or has the markitect-architect agent add them. The findings are for the owner to settle in the spec, with the markitect-design skill or by hand: nothing is recorded and nothing is committed.
