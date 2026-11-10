---
name: getting-started
description: Use when someone mentions Markitect, wants to set it up, start a project with it or upgrade it, or asks about the markitect command or a .markitect folder. It says what Markitect is and how to begin.
---

# Markitect

A Markitect project has one spec: the Markdown files in .markitect/spec/. The
spec is the authority over the code, tests and documentation: where they differ,
the spec is right. People architect the project with the markitect-architect
skill, and the /markitect-build skill has a builder agent build it and a fresh
reviewer verify each Definition; markitect status says what is left to verify.

`markitect` is a command on the path, so there is no need to look for it. `markitect help` lists what it does. If the shell says the command is not found, the binary is missing or its folder is not on the PATH. The Claude Code plugin brings it (`claude plugin marketplace add Glacius-Labs/Claude-Plugins`, then `claude plugin install markitect@glacius-labs`); or download the binary for the system from https://github.com/Glacius-Labs/Markitect/releases and put it on the PATH.

## How to begin

- **An empty folder, a new project or a repository that already has code:** run `markitect init`, then use the `markitect-architect` skill. It starts the spec with the owner, or draws it from the project as it is.
- **A folder that already has `.markitect/`:** it is set up, and `markitect check` says how it stands; the `markitect-architect` skill changes the spec.

`init` installs that skill into the project, so a session that was open before it does not have it yet. Read `.claude/skills/markitect-architect/SKILL.md` and follow it, or have the user run `/reload-skills`.

## After an upgrade

`markitect version` says whether this skill and the project's skills and agents match the binary. `markitect update` installs the newest release of the binary, then brings them in line; a binary the plugin brings is updated with the plugin (`claude plugin update markitect@glacius-labs`). It changes no spec file.
