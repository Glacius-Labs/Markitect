---
name: markitect-apply
description: Apply the spec as it stands. Markitect commits the spec and starts a background session in its own worktree, which makes the project match the spec and lands the work by itself. Use when the user says the spec is ready to apply.
disable-model-invocation: true
allowed-tools: Bash(markitect apply *)
---
!`markitect apply --skill $ARGUMENTS`

Tell the user in a line or two what this printed, and do nothing else. If it lists errors in the spec, offer to fix them. The run goes on without you, and the user can keep modeling.
