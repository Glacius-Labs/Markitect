---
name: markitect-apply
description: Build the spec as it stands. Markitect commits the spec and opens a run, and a builder agent makes the project in this folder match it, with a fresh reviewer checking the work. Use when the user says the spec is ready to apply.
disable-model-invocation: true
allowed-tools: Bash(markitect apply *), Bash(markitect status)
---
!`markitect apply --skill $ARGUMENTS`

Do what this printed. When the run has ended, tell the user in a line or two how it ended. If it lists errors in the spec, offer to fix them instead.
