---
name: markitect-apply
description: Build the spec as it stands. A builder agent makes the project in this folder match the spec, a fresh reviewer checks the work, and markitect gate certifies it. Use when the user says the spec is ready to apply.
disable-model-invocation: true
allowed-tools: Bash(markitect checklist *), Bash(markitect gate), Bash(markitect gate *), Bash(markitect status)
---
!`markitect checklist --start $ARGUMENTS`

Do what this printed. Each time an agent has ended, run markitect gate and do what it prints, until it passes or tells you to stop. You start the agents and change no file and store no answer yourself: what gate asks to build, fix or repair goes to the markitect-builder agent, started again with "Build the work.".

When gate passes, or says the folder matches its newest certificate, commit the work together with its certificate and Result in one commit on the current branch, unless that is done already, and push nothing. Then tell the user in a line or two how it ended. If it lists errors in the spec, offer to fix them instead.
