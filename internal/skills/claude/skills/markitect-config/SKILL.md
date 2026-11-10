---
name: markitect-config
description: Change how Markitect's agents and skills run - the model and effort of the builder, reviewer, architect and auditor agents, whether a skill works sequentially, in parallel or with a custom workflow, and the custom workflows - for this project or for all of the user's projects. Use when the user asks to change a Markitect agent's model or effort, to run Markitect builds or reviews in parallel or one at a time, or to use a workflow of their own.
allowed-tools: Bash(markitect config --list), Bash(markitect config --list *)
---
Markitect keeps this in `harness.yaml` at two levels: the project's `.markitect/harness.yaml`, committed with it, and the user's `~/.markitect/harness.yaml`, for all of their projects. Key by key, the project's value beats the user's, and the user's beats the built-in one. `markitect config --list` shows each value in force and where it comes from.

1. **What to change.** Turn the user's words into keys and values:
   - `agents.ROLE.model` and `agents.ROLE.effort`, ROLE being `builder` (writes the code), `reviewer` (verifies and records), `architect` (writes large parts of the spec) or `auditor` (reads the spec). A model is `sonnet`, `opus`, `haiku`, a full model ID or `inherit`; an effort is `low`, `medium`, `high`, `xhigh` or `max`. "All agents" means the four roles. A role keeps its effort when its model changes, so say which effort it keeps, or set both.
   - `strategy.SKILL`, SKILL being `build`, `review`, `cleanup`, `bughunt` or `audit`, and `strategy.default` for each skill that sets none: `auto` (the default: sequential for now, since the parallel strategy has passed only a small real run so far), `sequential` (one builder or reviewer at a time), `parallel` (the shipped workflow: per Package, in waves) or the name of a custom workflow.
   - `workflows.NAME`: the path of a custom workflow's script. Claude Code starts a saved workflow by name from `.claude/workflows/` or `~/.claude/workflows/`, so the file is `NAME.js` there, with `meta.name` NAME. NAME does not start with `markitect-`, which `init` and `update` keep for their own files. Set it before a strategy names it.

   If a request is unclear, ask; do not guess a model or an effort.
2. **Where.** Unless the user said so, ask whether the change is for this project, and so for everyone who works on it, or for all of the user's own projects. A user's value does not apply where the project sets the same key: say so when that is the case.
3. **Change it** with `markitect config KEY VALUE` for this project, `markitect config --user KEY VALUE` for all of the user's projects, and `markitect config [--user] --unset KEY` to remove a value. It writes the file and refreshes the agents' files of this project. Change no file by hand.
4. **Report** in a line or two what changed and at which level; a change of the project is the user's to commit.

A custom workflow is a strategy of its own: a Claude Code workflow script that keeps Markitect's contract. It takes the args `{"mode": ..., "into": ..., "focus": ..., "all": ...}`, the mode being build, cleanup, bughunt, review or audit and `into` the branch the work comes back to; it reads what to do from `markitect status --json`; it starts only the four roles' agents for the work; and it stops when status lists nothing a builder can settle, or when three rounds in a row leave the list no shorter. `.claude/workflows/markitect-parallel.js` is the one to start from: copy it under a name without `markitect-`, since `markitect update` restores its own files and removes others with that prefix. A workflow of the project's own is a file of the project like any other: list it on Root's registry page or in the ignore file.

The user's own skills are files in `.claude/skills/` that Markitect leaves alone. They may start the four roles' agents, and the hook holds those agents' rights there too.
