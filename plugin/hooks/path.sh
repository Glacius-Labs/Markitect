#!/bin/sh
# Puts the plugin's bin folder on the PATH of Claude Code's Bash tool.
#
# Claude Code puts a plugin's bin folder on the PATH of its shell, but on Windows
# its Bash tool does not get it (seen with Claude Code 2.1.319): the PowerShell
# tool finds bin\markitect.cmd, and Bash finds no markitect at all. A
# session-start hook may append shell lines to the file CLAUDE_ENV_FILE names,
# and Claude Code runs that file before each command of the Bash tool. This
# script appends a line that puts the folder first on the PATH where it is
# missing, and appends it once. pwd gives the folder in the form of the shell
# that runs this script, /c/Users/... in the Git Bash of Windows, which is the
# shell that reads the line.
set -eu
[ -n "${CLAUDE_ENV_FILE:-}" ] || exit 0
bin=$(cd "$(dirname "$0")/../bin" && pwd)
quoted="'$(printf '%s' "$bin" | sed "s/'/'\\\\''/g")'"
line="case \":\$PATH:\" in *:$quoted:*) ;; *) PATH=$quoted:\$PATH; export PATH ;; esac"
grep -qxF "$line" "$CLAUDE_ENV_FILE" 2>/dev/null || printf '%s\n' "$line" >>"$CLAUDE_ENV_FILE"
