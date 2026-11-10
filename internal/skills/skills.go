// Package skills holds what ships inside the markitect binary for agents: the
// architect skill, which markitect init installs where coding agents look for
// skills, and the harness of the work for Claude Code: its skills, its four agents
// and the prompts of its routines (DEC-100), and the workflow of the parallel
// strategy (DEC-103).
package skills

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed markitect-architect
var files embed.FS

//go:embed all:claude
var claudeFiles embed.FS

// File is one file of a skill, relative to the skills folder, such as
// "markitect-architect/SKILL.md".
type File struct {
	Path string
	Data []byte
}

// About says what Markitect is. It is the one place for this text: help, init, the
// instruction file of a project and the personal skill all print it.
const About = `A Markitect project has one spec: the Markdown files in .markitect/spec/. The
spec is the authority over the code, tests and documentation: where they differ,
the spec is right. People architect the project with the markitect-architect
skill, and the /markitect-build skill has a builder agent build it and a fresh
reviewer verify each Definition; markitect status says what is left to verify.
`

// PersonalName is the personal skill's folder name below a user's skills folder.
const PersonalName = "markitect"

// personalHead and personalTail surround About in the personal skill. It allows no
// command in advance.
const personalHead = `---
name: markitect
description: Use when someone mentions Markitect, wants to set it up, start a project with it or upgrade it, or asks about the markitect command or a .markitect folder. It says what Markitect is and how to begin.
---

# Markitect

`

const personalTail = "\n`markitect` is a command on the path, so there is no need to look for it. `markitect help` lists what it does. If the shell says the command is not found, the binary is missing or its folder is not on the PATH. The Claude Code plugin brings it (`claude plugin marketplace add Glacius-Labs/Claude-Plugins`, then `claude plugin install markitect@glacius-labs`); or download the binary for the system from https://github.com/Glacius-Labs/Markitect/releases and put it on the PATH.\n\n" +
	"## How to begin\n\n" +
	"- **An empty folder, a new project or a project that already exists:** run `markitect init`, then use the `markitect-architect` skill. It starts the spec with the owner, or draws it from the project as it is.\n" +
	"- **A folder that already has `.markitect/`:** it is set up, and `markitect check` says how it stands; the `markitect-architect` skill changes the spec.\n\n" +
	"`init` installs that skill into the project, so a session that was open before it does not have it yet. Read `.claude/skills/markitect-architect/SKILL.md` and follow it, or have the user run `/reload-skills`.\n\n" +
	"## After an upgrade\n\n" +
	"`markitect version` says whether this skill and the project's skills and agents match the binary. `markitect update` installs the newest release of the binary, then brings them in line; a binary the plugin brings is updated with the plugin (`claude plugin update markitect@glacius-labs`). It changes no spec file.\n"

// Personal returns the text of the personal skill, which reaches a session in a
// folder that has no project yet.
func Personal() []byte { return []byte(personalHead + About + personalTail) }

// Names returns the skill names, sorted.
func Names() []string { return []string{"markitect-architect"} }

// Files returns every file of every skill, sorted by path.
func Files() []File {
	var out []File
	_ = fs.WalkDir(files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := files.ReadFile(p)
		if err != nil {
			return err
		}
		out = append(out, File{Path: p, Data: data})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// HarnessPrefix is what every file of the harness starts with below its folder.
const HarnessPrefix = "markitect-"

// Harness returns the files of the harness for Claude Code, with the paths they
// have in a project, sorted by path.
func Harness() []File {
	var out []File
	_ = fs.WalkDir(claudeFiles, "claude", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := claudeFiles.ReadFile(p)
		if err != nil {
			return err
		}
		out = append(out, File{Path: "." + p, Data: data})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}
