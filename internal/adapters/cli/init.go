package cli

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// SkillDirs are the folders where coding agents look for project skills:
// Claude Code and the Agent Skills folder other agents such as Codex read.
var SkillDirs = []string{".claude/skills", ".agents/skills"}

func (c CLI) init(ctx context.Context, args []string) (int, error) {
	var o options
	var name string
	fs := c.flags("init", &o)
	fs.StringVar(&name, "name", "", "the project's title for the project anchor (default: the repository folder's name)")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	ws, err := c.Git.Open(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	if name == "" {
		name = filepath.Base(ws.Root())
	}
	created := func(file string) { fmt.Fprintf(c.Stdout, "created %s\n", file) }
	writeNew := func(file string, data []byte) error {
		if ws.Exists(file) {
			return nil
		}
		if err := ws.Write(file, data); err != nil {
			return err
		}
		created(file)
		return nil
	}
	if err := writeNew(project.ManifestFile, []byte(fmt.Sprintf("# The Libraries this project uses, with their versions. Run markitect lock after a change.\nlibraries:\n  std: %s\n", libraries.Version))); err != nil {
		return Misused, err
	}
	if err := writeNew(project.ChecksFile, []byte(checksTemplate)); err != nil {
		return Misused, err
	}
	modelFiles, err := ws.List(project.ModelDir)
	if err != nil {
		return Misused, err
	}
	if len(modelFiles) == 0 {
		if err := writeNew(project.ModelDir+"/project.md", []byte(fmt.Sprintf(anchorTemplate, name))); err != nil {
			return Misused, err
		}
	}
	if err := writeNew(project.RootPage, []byte(rootPageTemplate)); err != nil {
		return Misused, err
	}
	if err := c.ensureIgnored(ws, created); err != nil {
		return Misused, err
	}
	if !ws.Exists(project.LockFile) {
		if code, err := c.writeLock(ws); err != nil || code != OK {
			return code, err
		}
	}
	installed, err := installSkills(ws)
	if err != nil {
		return Misused, err
	}
	fmt.Fprintf(c.Stdout, "installed the skills %s into %s\n", strings.Join(installed, ", "), strings.Join(SkillDirs, " and "))
	fmt.Fprintln(c.Stdout, "next: register the repository's files on the registry pages, or model an existing project with the markitect-extract skill; then run markitect check")
	return OK, nil
}

// installSkills writes every skill file into each skill folder and removes
// files of the same skills that the binary no longer ships.
func installSkills(ws Workspace) ([]string, error) {
	want := map[string][]byte{}
	for _, f := range skills.Files() {
		for _, dir := range SkillDirs {
			want[dir+"/"+f.Path] = f.Data
		}
	}
	for _, dir := range SkillDirs {
		for _, name := range skills.Names() {
			existing, err := ws.List(dir + "/" + name)
			if err != nil {
				return nil, err
			}
			for _, f := range existing {
				if _, keep := want[f]; !keep {
					if err := ws.Remove(f); err != nil {
						return nil, err
					}
				}
			}
		}
	}
	for file, data := range want {
		if err := ws.Write(file, data); err != nil {
			return nil, err
		}
	}
	return skills.Names(), nil
}

// ensureIgnored adds the installed skill folders to the ignore page, so that
// they need no registry entry.
func (c CLI) ensureIgnored(ws Workspace, created func(string)) error {
	var current []byte
	if ws.Exists(project.IgnoreFile) {
		data, err := ws.Read(project.IgnoreFile)
		if err != nil {
			return err
		}
		current = data
	}
	var rows []string
	for _, dir := range SkillDirs {
		pattern := dir + "/markitect-*/**"
		if !bytes.Contains(current, []byte(pattern)) {
			rows = append(rows, fmt.Sprintf("| `%s` | Skills installed by markitect init |\n", pattern))
		}
	}
	if len(rows) == 0 {
		return nil
	}
	var b bytes.Buffer
	if current == nil {
		b.WriteString(ignoreHeader)
	} else {
		b.Write(current)
		if !bytes.HasSuffix(current, []byte("\n")) {
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}
	b.WriteString("| Path | Why |\n|---|---|\n")
	b.WriteString(strings.Join(rows, ""))
	if err := ws.Write(project.IgnoreFile, b.Bytes()); err != nil {
		return err
	}
	if current == nil {
		created(project.IgnoreFile)
	} else {
		fmt.Fprintf(c.Stdout, "added the skill folders to %s\n", project.IgnoreFile)
	}
	return nil
}

func (c CLI) lock(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("lock", &o)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	ws, err := c.Git.Open(ctx, o.repo)
	if err != nil {
		return Misused, err
	}
	return c.writeLock(ws)
}

// writeLock writes the lock from the manifest and the Libraries it names.
func (c CLI) writeLock(ws Workspace) (int, error) {
	data, err := ws.Read(project.ManifestFile)
	if err != nil {
		return Misused, fmt.Errorf("read the manifest: %w; run markitect init", err)
	}
	manifest, ds := model.ParseManifest(project.ManifestFile, data)
	files, err := ws.List(project.LibraryDir)
	if err != nil {
		return Misused, err
	}
	local, err := project.LocalLibraries(ws, files)
	if err != nil {
		return Misused, err
	}
	var libs []model.LibrarySource
	for _, ref := range manifest.Libraries {
		lib, code, reason := project.LibrarySource(ref, local)
		if code != "" {
			ds.Add(code, project.ManifestFile, ref.Line, "%s", reason)
			continue
		}
		libs = append(libs, lib)
	}
	if len(ds) > 0 {
		for _, d := range ds.Sorted() {
			fmt.Fprintln(c.Stderr, d.String())
		}
		return Failed, nil
	}
	if err := ws.Write(project.LockFile, model.RenderLock(libs)); err != nil {
		return Misused, err
	}
	var names []string
	for _, l := range libs {
		names = append(names, l.Name+" "+l.Version)
	}
	fmt.Fprintf(c.Stdout, "wrote %s: %s\n", project.LockFile, strings.Join(names, ", "))
	return OK, nil
}

const checksTemplate = `# The required checks: the one proof in a Run that does not come from an agent.
# Each check runs its command with sh -c in the named directory, inside the
# environment (a container image), within the timeout. For example:
#
# checks:
#   - name: test
#     command: go test ./...
#     directory: .
#     timeout: 10m
#     environment: golang:1.27
checks: []
`

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

const ignoreHeader = `# Ignored files

Files that deliberately belong to no Package.

`
