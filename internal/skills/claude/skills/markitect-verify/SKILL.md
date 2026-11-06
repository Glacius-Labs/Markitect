---
name: markitect-verify
description: Check whether the project matches its spec. Markitect opens a review of the last commit, and a fresh reviewer agent answers it without changing anything. Use when the user asks to verify the project against its spec.
disable-model-invocation: true
allowed-tools: Bash(markitect verify *), Bash(markitect status)
---
!`markitect verify --skill $ARGUMENTS`

Do what this printed. When the run has ended, tell the user in a line or two what it found.
