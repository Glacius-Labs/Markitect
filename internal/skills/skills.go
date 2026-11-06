// Package skills holds what ships inside the markitect binary for agents: the
// modeling skill, the start and Extract skills built on it, which markitect init
// installs where coding agents look for skills, and the harness of a Run for
// Claude Code.
package skills

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed markitect-modeling markitect-extract markitect-start
var files embed.FS

//go:embed all:claude
var claudeFiles embed.FS

// File is one file of a skill, relative to the skills folder, such as
// "markitect-modeling/SKILL.md".
type File struct {
	Path string
	Data []byte
}

// About says what Markitect is. It is the one place for this text: help, init, the
// instruction file of a project and the personal skill all print it.
const About = `A Markitect project has one spec: the Markdown files in .markitect/model/. The
spec is the authority over the code, tests and documentation: where they differ,
the spec is right. People model the spec, and markitect apply has an agent build
it in the project's folder, a fresh reviewer check it, and the work land.
`

// PersonalName is the personal skill's folder name below a user's skills folder.
const PersonalName = "markitect"

// personalHead and personalTail surround About in the personal skill. It allows no
// command in advance, since apply starts sessions.
const personalHead = `---
name: markitect
description: Use when someone mentions Markitect, wants to set it up, start a project with it or upgrade it, or asks about the markitect command or a .markitect folder. It says what Markitect is and how to begin.
---

# Markitect

`

const personalTail = "\n`markitect` is a command on the path, so there is no need to look for it. `markitect help` lists what it does. If the shell says the command is not found, the binary is missing or its folder is not on the PATH: `go install github.com/Glacius-Labs/Markitect/cmd/markitect@main` installs it.\n\n" +
	"## How to begin\n\n" +
	"- **An empty folder or a new project:** run `markitect init`, then use the `markitect-start` skill.\n" +
	"- **A repository that already has code:** run `markitect init`, then use the `markitect-extract` skill.\n" +
	"- **A folder that already has `.markitect/`:** it is set up, and `markitect check` says how it stands.\n\n" +
	"`init` installs those skills into the project, so a session that was open before it does not have them yet. Read `.claude/skills/markitect-start/SKILL.md` or `.claude/skills/markitect-extract/SKILL.md` and follow it, or have the user run `/reload-skills`.\n\n" +
	"## After an upgrade\n\n" +
	"`markitect version` says whether this skill and the project's skills and agents match the binary, and `markitect update` brings them in line. It changes no spec file.\n"

// Personal returns the text of the personal skill, which reaches a session in a
// folder that has no project yet.
func Personal() []byte { return []byte(personalHead + About + personalTail) }

// Names returns the skill names, sorted.
func Names() []string { return []string{"markitect-extract", "markitect-modeling", "markitect-start"} }

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

// Harness returns the files of the default strategy for Claude Code, with the
// paths they have in a project, sorted by path.
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
