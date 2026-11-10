package cli

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/setup"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

func (c CLI) init(_ context.Context, args []string) (int, error) {
	var o options
	var name string
	fs := c.flags("init", &o)
	fs.StringVar(&name, "name", "", "the project's title for the project anchor (default: the folder's name)")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	// Inside a project, init works on it; elsewhere it makes the folder one. It
	// makes no repository and commits nothing (DEC-097).
	ws, err := c.Folders.Project(o.repo)
	if err != nil {
		if ws, err = c.Folders.At(o.repo); err != nil {
			return Misused, err
		}
	}
	if name == "" {
		name = filepath.Base(ws.Root())
	}
	created, emptySpec, err := setup.StartSpec(ws, name)
	c.created(created)
	if err != nil {
		return Misused, err
	}
	kept, created, err := setup.WriteInstructions(ws, name)
	if kept {
		fmt.Fprintf(c.Stdout, "left %s as it is; add that this project has a spec in %s, which is the authority\n", setup.InstructionsFile, project.SpecDir)
	}
	c.created(created)
	if err != nil {
		return Misused, err
	}
	if !ws.Exists(project.LockFile) {
		if code, err := c.writeLock(ws); err != nil || code != OK {
			return code, err
		}
	}
	cfg, err := harnessConfig(ws)
	if err != nil {
		return Misused, err
	}
	installed, err := setup.InstallSkills(ws)
	if err != nil {
		return Misused, err
	}
	fmt.Fprintf(c.Stdout, "installed the skills %s into %s\n", strings.Join(installed, ", "), strings.Join(setup.SkillDirs, " and "))
	if err := setup.InstallHarness(ws, cfg); err != nil {
		return Misused, err
	}
	fmt.Fprintln(c.Stdout, "installed the harness: the build, review, cleanup, bughunt, audit, schedule and config skills, the builder, reviewer, architect and auditor agents, and the parallel workflow")
	said, err := setup.EnsureSettings(ws)
	c.say(said)
	if err != nil {
		return Misused, err
	}
	if err := setup.InstallAttributes(ws); err != nil {
		return Misused, err
	}
	if added, err := setup.EnsureIgnore(ws); err != nil {
		return Misused, err
	} else if added {
		fmt.Fprintf(c.Stdout, "added %s to %s, so that Claude Code's worktrees stay out of Git\n", project.WorktreeDir, setup.GitIgnoreFile)
	}
	fmt.Fprintln(c.Stdout, "Markitect uses no Git: commit what init wrote when you like.")
	fmt.Fprintf(c.Stdout, "\n%s\n", skills.About)
	if emptySpec {
		fmt.Fprint(c.Stdout, nextSteps)
	}
	// init never writes to the home folder, so it only says what install would add.
	if _, skill, err := personalSkillState(); err == nil && skill != current && !pluginBrings(skill) {
		fmt.Fprintf(c.Stdout, "\ntip: %s\n", personalSkillTips[skill])
	}
	c.warnPath()
	return OK, nil
}

// created says which files a command created.
func (c CLI) created(files []string) {
	for _, file := range files {
		fmt.Fprintf(c.Stdout, "created %s\n", file)
	}
}

// say prints lines setup returned, each on its own.
func (c CLI) say(lines []string) {
	for _, line := range lines {
		fmt.Fprintln(c.Stdout, line)
	}
}

// personalSkillTips say what to do about a personal skill that is not as this
// binary would write it.
var personalSkillTips = map[fileState]string{
	missing:  "markitect install tells Claude Code what Markitect is in every folder, also in one without a project yet.",
	outdated: "your personal skill comes from another markitect; markitect update refreshes it.",
}

// tidy brings the lock in line with the manifest, as go mod tidy does for go.sum:
// check only reads the lock, and says to run tidy when the two differ.
func (c CLI) tidy(_ context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("tidy", &o)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	ws, err := c.Folders.Project(o.repo)
	if err != nil {
		return Misused, err
	}
	return c.writeLock(ws)
}

// writeLock writes the lock from the manifest and the Libraries it names, and
// says what it wrote, or the manifest's errors.
func (c CLI) writeLock(ws Workspace) (int, error) {
	libs, problems, err := setup.WriteLock(ws)
	if err != nil {
		return Misused, err
	}
	if len(problems) > 0 {
		for _, d := range problems {
			fmt.Fprintln(c.Stderr, d.String())
		}
		return Failed, nil
	}
	var names []string
	for _, l := range libs {
		names = append(names, l.Name+" "+l.Version)
	}
	fmt.Fprintf(c.Stdout, "wrote %s: %s\n", project.LockFile, strings.Join(names, ", "))
	return OK, nil
}

// nextSteps ends the init of a new project. The session that ran init started
// before the skills existed, and Claude Code does not watch a skills folder that
// was not there when a session started.
const nextSteps = `next: run /markitect-architect in Claude Code here. It asks what the project is and starts the spec with you, or draws it from the project already here.
A session that was open before init runs /reload-skills first, or reads .claude/skills/markitect-architect/SKILL.md.
`
