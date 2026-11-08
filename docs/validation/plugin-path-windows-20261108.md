# The plugin's markitect on Windows — 8 November 2026

The smoke session of 0.25.0 found that on Windows the plugin's `markitect` was missing from the `PATH` of Claude Code's Bash tool, so the skills and agents got "command not found". The PowerShell tool had the plugin's `bin` folder on its `PATH`, but could not run the launcher there, a shell script. Hooks worked, because they call the launcher by its path. This record is the probe of the fix in 0.25.1: a session-start hook that puts `bin/` on the Bash tool's `PATH` through the file `CLAUDE_ENV_FILE` names, and `bin/markitect.cmd`, which runs the launcher from PowerShell or cmd with the bash of Git for Windows.

**What the probe shows:**

- **`markitect` resolves by name in all four places.** The main session's Bash tool, its PowerShell tool, a general-purpose subagent and a subagent isolated in a worktree each found the plugin's `markitect` and ran `markitect version`, which reported `0.25.0+3d88b648`. Bash and both subagents found `bin/markitect`; PowerShell found `bin\markitect.cmd`.
- **Without the hook and the shim it resolves nowhere.** A control run of the same four steps, with the same Claude Code and the same plugin copy minus `hooks/path.sh`, its hook and `markitect.cmd`, got "command not found" in Bash and in both subagents. In PowerShell, `Get-Command` found `bin\markitect`, which PowerShell would not run ("Cannot run a document in the middle of a pipeline"). This is the failure the smoke session saw with the installed 0.25.0.
- **The line reaches subagents and worktree agents.** Claude Code runs the file before every command of the Bash tool, in a subagent too, and in an agent whose folder is a worktree of the project.
- **One PowerShell exit code is PowerShell's.** The probe's PowerShell call reported exit 255 although both lines printed. `Select-Object -First 1` stops the native program early; it gives -1 to a plain `cmd` call as well, and `markitect version` alone through the shim exits 0.

## Source and method

**Source.** The change on branch `fix/plugin-path-windows` from Main `32ebe716`. A copy of its `plugin/` folder, with the version set to 0.25.0, so that the launcher started the 0.25.0 binary already in the plugin's data folder and downloaded nothing.

**Run.** One headless `claude -p` run of Claude Code 2.1.319 on Windows 11, model Sonnet 5.5, permission mode `auto`, at most 20 turns, a 10-minute timeout as the abort. The copy was loaded with `--plugin-dir`; the installed `markitect@glacius-labs` 0.25.0 was switched off for that run only with `--settings`. The project was a scratch Git repository with one commit. The prompt asked for four steps and nothing else: `command -v markitect; markitect version` in the Bash tool, the same in PowerShell, and the same in a subagent and in a subagent with worktree isolation. The run took 20 seconds and five turns and exited 0. The control run was the same, with the plugin copy minus the hook, its script and the shim; it took 19 seconds and five turns and exited 0. The scratch projects and their worktrees were deleted afterwards. Started by the coordinating session's assignment of the fix; nothing failed.

**Tests.** `internal/bootstrap/plugin_test.go` runs the hook against a plugin folder whose name has a space and a quote, runs the file it wrote twice, and starts `markitect` by name through it; on Windows it runs `markitect.cmd` by name from cmd, with and without a `CLAUDE_CODE_GIT_BASH_PATH` that names no file, and checks the message without Git on the `PATH`. `internal/adapters/cli/plugin_test.go` checks that the hook is the first at the start of a session and that Claude Code waits for it, and that `markitect.cmd` is checked out with CRLF. CI runs on Linux only, so the shim's own test runs only on a developer's Windows machine.

## Limits

- One run and one control on one Windows machine with Git for Windows in its default place. A `CLAUDE_CODE_GIT_BASH_PATH` was tested by hand and in the tests, not in the run. A Git whose `git.exe` is not beside its `bin\bash.exe` folder, as with Scoop's shims, needs `CLAUDE_CODE_GIT_BASH_PATH` for the shim.
- Linux and macOS were not probed in Claude Code. There the hook's line finds `bin/` on the `PATH` already and changes nothing; the tests run on Linux in CI.
- The probe used a copy of the plugin loaded with `--plugin-dir`, not the plugin installed from the marketplace. The release check and the smoke session cover the installed path.
- A session that was already running when the plugin was installed or updated does not run the new hook; it needs a new session.
