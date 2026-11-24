# The plugin

The Claude Code plugin `markitect` fetches the tool: it brings the `markitect` command into a session. Beside its launcher it holds the default, which a project copies into its spec by hand ([library](library.md)). It holds no skill and no agent. [The project](../concept/the-project.md#the-plugin-and-the-version) in the concept, [DEC-135](../decisions.md#dec-135-the-tool-stays-in-go-and-is-rebuilt-for-the-language) and [DEC-159](../decisions.md#dec-159-the-default-is-a-whole-spec-to-start-from-and-lies-in-the-plugin) say why; this page says what the plugin holds, how its launcher gets the tool and how firm that is.

## What it holds

The plugin is the folder [`plugin/`](../../plugin/), named `markitect` in its manifest.

| Path | Holds |
|---|---|
| `.claude-plugin/plugin.json` | The manifest: the name `markitect`, the description and the version, which the launcher reads |
| `bin/markitect`, `bin/markitect.cmd` | The launcher, a POSIX shell script, and its shim for PowerShell and cmd on Windows |
| `hooks/hooks.json`, `hooks/path.sh` | Two hooks at the start of a session, both for the launcher |
| `default/` | The default, a whole spec that a project copies into its spec folder ([library](library.md)). It lies here because a skill of a plugin reaches only the plugin's own files; the skill that sets a project up is not built |

The plugin and the tool always have the same version: a test refuses a difference, and the [release workflow](../releasing.md#what-a-tag-publishes) checks it again.

## The launcher

While the plugin is on, Claude Code puts its `bin/` folder on the PATH of its shell, so a session runs `markitect` without anyone installing it. The first call downloads the binary of the release that has the plugin's own version, `markitect-vX.Y.Z-OS-ARCH` with `.exe` on Windows, and `markitect-vX.Y.Z-checksums.txt` from the [releases](https://github.com/Glacius-Labs/Markitect/releases), checks the binary against the checksums, keeps it in the plugin's data folder, `~/.claude/plugins/data/markitect-glacius-labs/vX.Y.Z/`, and starts it. Later calls start it at once, and a plugin update with a new version fetches the matching binary the same way. `MARKITECT_RELEASES_URL` names a mirror with the same layout; PowerShell reaches the launcher through `markitect.cmd`, which runs it with the bash of Git for Windows; that is the plugin's launcher on Windows, as Claude Code itself needs it, not the tool.

The launcher fetches the version of the plugin. A project pins no version; `markitect version` names the version and build of the tool the launcher started ([project reference](project-reference.md#version)).

A development version of the plugin, such as `0.35.0-dev`, has no release binary and fetches nothing: its launcher starts the next `markitect` on the PATH, passing over its own `bin/`, with every argument. So for development, build the tool from source, `go install github.com/Glacius-Labs/Markitect/cmd/markitect@COMMIT` with the commit of the plugin, and keep its folder on the PATH; the plugin's `bin/` may come before it.

The two hooks at the start of a session serve the launcher alone. The first puts `bin/` on the PATH of the Bash tool, which on Windows does not get it from Claude Code: it appends one line to the file `CLAUDE_ENV_FILE` names, and Claude Code waits for it, so that the first command finds `markitect`. The second fetches the binary in the background, outside the shell's sandbox, in a project that has a `.markitect/` folder; in any other project the plugin downloads nothing until `markitect` is called.

| Observation | Response |
|---|---|
| The launcher says it cannot download a file of the release | The launcher could not reach the releases page; a sandboxed shell may block it. Start a new session in the project, whose start hook fetches outside the sandbox, or run `markitect` once in a session without the sandbox |
| The launcher says a development plugin needs the tool built from source | No `markitect` follows the plugin's `bin/` on the PATH. Build the tool from source, as above, and put its folder on the PATH |
| `markitect` exists in Claude Code but not in your terminal | The plugin puts it on the PATH of Claude Code's tools only. Download the binary of the release, check it against the checksums file and put it on your PATH as `markitect` |
| `markitect: command not found` in the Bash tool on Windows | The first start hook puts it on the Bash tool's PATH. A session that started before the plugin was installed has no such hook: start a new one |
| In PowerShell, the launcher says it needs the bash of Git for Windows | `markitect.cmd` runs the launcher with the bash that belongs to the `git.exe` on the PATH, and found none. Set `CLAUDE_CODE_GIT_BASH_PATH` to the `bash.exe` of Git for Windows, as Claude Code itself reads it |

## How firm it is

The plugin brings the tool and holds the default as plain files, so it checks and steers nothing in a session.

- **The download is checked,** against the checksums of the same release, over HTTPS to GitHub: that guards against a damaged or swapped file. The binary is not signed ([releasing](../releasing.md#rules)).
- **The version is the plugin's.** The launcher starts the release of the plugin's own version, and a development plugin starts whatever `markitect` follows it on the PATH; `markitect version` names which one runs.
- **Nothing depends on the plugin.** A project that does not use it runs the same tool, built from source or downloaded from a release ([install](../../README.md#install)), and copies the default from a clone of the repository.

## The check of the plugin

The tests of the command line, `internal/cli/launcher_test.go`, serve a release under the file names the workflow builds and run the launcher against it: the fetch and the checksum, calls at the same time, the data folder of an installed plugin, the development launcher, both hooks and the shim for Windows. They also check that the plugin has the version of the source and holds no skills or agents.
