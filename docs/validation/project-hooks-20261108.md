# The project's hooks with the plugin alone — 8 November 2026

Up to 0.25.2, the hooks `init` wrote into a project's `.claude/settings.json` ran `markitect hook pretool` by name. A hook does not see the plugin's `bin/`, so in a session with the plugin alone they failed with exit 127 before nearly every edit, agent start and `markitect mark`, in the session and in every agent. The smoke session on 0.25.1 reported it, and the parallel run on 0.25.2 counted 13 such failures, which blocked nothing, since the plugin's own hooks ran ([the parallel record](parallel-strategy-20261108.md)). 0.25.3 changes the hook ([the hook](../../internal/run/README.md#the-hook)). This record is one probe of the released 0.25.3, with the plugin alone.

**What the probe shows:**

- **No hook error.** Before the Edit of a code file, two hooks ran, the project's and the plugin's, and both exited 0 with no output. The same held before the Agent call. All 28 hooks of the run exited 0, and none failed with 127. Claude Code labels a hook's refusal a "hook error" too; the only one in the run is the refusal below.
- **The guard still holds, and runs once.** A general-purpose subagent tried to write `.markitect/review.sum`. One of the two hooks before that Write exited 0 with no output. The other refused it: "…\.markitect\review.sum is written only by markitect mark, which the markitect-reviewer agent runs." The file was not written, and the subagent stopped. The silent hook is the project's: its shell found no `markitect`. The refusal came from the plugin's hook, through its launcher.
- **`update` brings an old project in line.** In the test project, `markitect update`, run through the plugin's launcher, replaced the three bare hooks with the new command and `"shell": "bash"`. It also rewrote nine skill, agent and workflow files that differed from 0.25.3's, and left the binary and the personal skill to the plugin.

## Source and method

**Source.** The release v0.25.3, merge `196f3cac`, pinned by commit `414c55a` of Claude-Plugins. The installed plugin was updated with `claude plugin marketplace update glacius-labs` and `claude plugin update markitect@glacius-labs`, from 0.25.2 to 0.25.3, at user scope.

**Project.** `markitect-test` on the owner's machine: a project `init` set up with 0.25.1, with the three bare hooks in its settings, no Git repository and no code. The probe script wrote one file, `probe.go`, before the run, and it was deleted after the run.

**Run.** One headless `claude -p` run of Claude Code 2.1.319 on Windows 11, from 15:13:41 to 15:13:59 UTC, exit 0, 4 turns:
- model Haiku 5.5;
- permission mode `acceptEdits`, so only a hook could refuse the write;
- the user's own settings, with the plugin enabled;
- no `markitect` on the `PATH` of the run, and no variable of the calling session;
- `--max-turns 15` and `timeout 600` as the bounds;
- `--include-hook-events`, so every hook's exit code and output are in the stream.

It was authorized by the coordinating session as part of the 0.25.3 task: one bounded probe of about ten minutes. The stream and the settings before and after are kept in the coordinating session's notes, outside the repository.

**Tests.** `internal/bootstrap` covers:
- the hook command in `sh`, with and without a `markitect` on the `PATH`;
- when a binary leaves the guard to the plugin, in eleven cases;
- `update` replacing the bare hooks.

## Limits

- One run, on Windows, with the plugin alone. A binary on the `PATH`, alone or beside the plugin, was covered by tests only, not by a run.
- The hook events do not name which hook is the plugin's. The attribution above follows from the commands: the project's hook prints nothing when its shell finds no `markitect`.
- Haiku carried out two plain steps. That says nothing about how it builds or reviews.
