---
name: markitect-verify
description: Check whether the project matches its spec. A fresh reviewer agent answers the checklist of what changed since the newest certificate, or of every Definition with --full, without changing anything, and markitect gate --findings writes what it found. Use when the user asks to verify the project against its spec.
disable-model-invocation: true
allowed-tools: Bash(markitect checklist *), Bash(markitect gate *), Bash(markitect status)
---
!`markitect checklist --review --start $ARGUMENTS`

Do what this printed. When the reviewer has ended, run markitect gate --findings and do what it prints; while it asks for answers, start the reviewer again. You change no file and store no answer yourself.

When it has written the Result, commit the pages it wrote in one commit on the current branch and push nothing. Then tell the user in a line or two what it found.
