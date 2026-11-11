// Package setup is what Markitect writes into a project: the first files of a
// new one, the harness with its agents rendered from the harness
// configuration, the hooks and the worktree base in Claude Code's settings, the
// record's Git attributes, the ignore line for Claude Code's worktrees and the
// lock. It works through its own Folder port and prints nothing: each function
// returns what it did, and the command line says it.
package setup

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// Folder is a project folder as setup reads and writes it.
type Folder interface {
	project.Repository
	Write(file string, data []byte) error
	Exists(file string) bool
	List(dir string) ([]string, error)
	Remove(file string) error
}

// StartSpec writes what a project lacks of its first files: the manifest with
// std, the project anchor titled name when the spec folder is empty, and Root's
// registry page. It never changes a file that is there. It returns the files it
// created, in order, also when it fails, and whether the spec folder was empty.
func StartSpec(f Folder, name string) (created []string, emptySpec bool, err error) {
	writeNew := func(file string, data []byte) error {
		if f.Exists(file) {
			return nil
		}
		if err := f.Write(file, data); err != nil {
			return err
		}
		created = append(created, file)
		return nil
	}
	if err := writeNew(project.ManifestFile, []byte(fmt.Sprintf(manifestTemplate, libraries.Version))); err != nil {
		return created, false, err
	}
	specFiles, err := f.List(project.SpecDir)
	if err != nil {
		return created, false, err
	}
	emptySpec = len(specFiles) == 0
	if emptySpec {
		if err := writeNew(project.SpecDir+"/project.md", []byte(fmt.Sprintf(anchorTemplate, name))); err != nil {
			return created, emptySpec, err
		}
	}
	if err := writeNew(project.RootPage, []byte(rootPageTemplate)); err != nil {
		return created, emptySpec, err
	}
	return created, emptySpec, nil
}

// WriteInstructions writes InstructionsFile for the project titled name, unless
// it is there, and a CLAUDE.md that imports it, unless that is there. It says
// whether it left an InstructionsFile as it was, and whether a CLAUDE.md that was
// there does not import it, so that Claude Code does not read it. It returns the
// files it created, also when it fails.
func WriteInstructions(f Folder, name string) (kept, unread bool, created []string, err error) {
	write := func(file string, data []byte) error {
		if err := f.Write(file, data); err != nil {
			return err
		}
		created = append(created, file)
		return nil
	}
	if f.Exists(InstructionsFile) {
		kept = true
	} else if err := write(InstructionsFile, []byte(fmt.Sprintf(instructionsTemplate, name))); err != nil {
		return kept, false, created, err
	}
	if !f.Exists(ClaudeFile) {
		if err := write(ClaudeFile, []byte(importLine+"\n")); err != nil {
			return kept, false, created, err
		}
		return kept, false, created, nil
	}
	data, err := f.Read(ClaudeFile)
	if err != nil {
		return kept, false, created, err
	}
	return kept, !imports(string(data)), created, nil
}

// imports says whether the text of a CLAUDE.md imports InstructionsFile: a word
// of it is the import, with or without ./ before the name. A mention in code
// marks, as in a sentence about the file, is none.
func imports(text string) bool {
	for _, word := range strings.Fields(text) {
		if word == importLine || word == "@./"+InstructionsFile {
			return true
		}
	}
	return false
}

// manifestTemplate is a new project's manifest.
const manifestTemplate = `# The Libraries this project uses, with their versions. Run markitect tidy after a change.
libraries:
  std: %s
`

// InstructionsFile is the file coding agents read when a session starts. Claude
// Code reads it through ClaudeFile, which init writes with only an import of it.
const (
	InstructionsFile = "AGENTS.md"
	ClaudeFile       = "CLAUDE.md"
	importLine       = "@" + InstructionsFile
)

// instructionsTemplate tells any session what kind of project this is. The
// agents read it too, so it holds no process.
const instructionsTemplate = "# %s\n\n" + skills.About + "\n" +
	"A change to what the project does starts in the spec, changed with the `markitect-architect` skill and accepted by the owner.\n\n" +
	"`markitect` is a command on the path, and `markitect help` lists what it does.\n\n" +
	"A builder or reviewer of Markitect work takes what to do from `markitect status`, and the spec stays as it is.\n"

const anchorTemplate = `---
kind: std/Project
---
# %s

What this project is, for whom, and what it deliberately does not do.
`

const rootPageTemplate = `# Registry: Root

Root owns every file no other Package claims. List each of them here with the
Definitions it realizes and why.

| File | Realizes | Why |
|---|---|---|
`
