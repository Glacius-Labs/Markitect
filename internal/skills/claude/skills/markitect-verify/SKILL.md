---
name: markitect-verify
description: Check that the project matches the spec, with a reviewer in a background session. It changes nothing and ends with Findings. Use when the user asks to verify the project against the spec.
disable-model-invocation: true
allowed-tools: Bash(markitect verify *)
---
!`markitect verify --skill $ARGUMENTS`

Tell the user in a line or two what this printed, and do nothing else. The check goes on without you.
