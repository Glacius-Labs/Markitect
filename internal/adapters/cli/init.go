package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// SkillDirs are the folders where coding agents look for project skills:
// Claude Code and the Agent Skills folder other agents such as Codex read.
var SkillDirs = []string{".claude/skills", ".agents/skills"}

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
	// A spec where an earlier markitect kept it is moved by update, which also
	// rewrites the record; init would only start a second, empty spec beside it.
	if old, err := ws.List(project.OldSpecDir); err != nil {
		return Misused, err
	} else if len(old) > 0 {
		return Misused, fmt.Errorf("this project keeps its spec in %s, where an earlier markitect kept it; run markitect update, which moves it to %s", project.OldSpecDir, project.SpecDir)
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
	if err := writeNew(project.ManifestFile, []byte(fmt.Sprintf(manifestTemplate, libraries.Version))); err != nil {
		return Misused, err
	}
	specFiles, err := ws.List(project.SpecDir)
	if err != nil {
		return Misused, err
	}
	if len(specFiles) == 0 {
		if err := writeNew(project.SpecDir+"/project.md", []byte(fmt.Sprintf(anchorTemplate, name))); err != nil {
			return Misused, err
		}
	}
	if err := writeNew(project.RootPage, []byte(rootPageTemplate)); err != nil {
		return Misused, err
	}
	if ws.Exists(instructionsFile) {
		fmt.Fprintf(c.Stdout, "left %s as it is; add that this project has a spec in %s, which is the authority\n", instructionsFile, project.SpecDir)
		c.noteRetiredSkills(ws)
	} else if err := writeNew(instructionsFile, []byte(fmt.Sprintf(instructionsTemplate, name))); err != nil {
		return Misused, err
	}
	if err := writeNew("CLAUDE.md", []byte("@"+instructionsFile+"\n")); err != nil {
		return Misused, err
	}
	if !ws.Exists(project.LockFile) {
		if code, err := c.writeLock(ws); err != nil || code != OK {
			return code, err
		}
	}
	// An earlier manifest's model for the agents moves to the harness
	// configuration first, so that the agents get it (DEC-103).
	proj, err := readProjectHarness(ws)
	if err != nil {
		return Misused, err
	}
	if _, moved, err := moveManifestModel(ws, proj, false); err != nil {
		return Misused, err
	} else if len(moved) > 0 {
		fmt.Fprintf(c.Stdout, "moved the agents' model from %s to %s\n", project.ManifestFile, project.HarnessFile)
	}
	cfg, err := harnessConfig(ws)
	if err != nil {
		return Misused, err
	}
	installed, err := installSkills(ws)
	if err != nil {
		return Misused, err
	}
	fmt.Fprintf(c.Stdout, "installed the skills %s into %s\n", strings.Join(installed, ", "), strings.Join(SkillDirs, " and "))
	if err := installHarness(ws, cfg); err != nil {
		return Misused, err
	}
	fmt.Fprintln(c.Stdout, "installed the harness: the build, review, cleanup, bughunt, audit, schedule and config skills, the builder, reviewer, architect and auditor agents, and the parallel workflow")
	if err := c.ensureSettings(ws); err != nil {
		return Misused, err
	}
	if err := installAttributes(ws); err != nil {
		return Misused, err
	}
	fmt.Fprintln(c.Stdout, "Markitect uses no Git: commit what init wrote when you like.")
	fmt.Fprintf(c.Stdout, "\n%s\n", skills.About)
	if len(specFiles) == 0 {
		fmt.Fprint(c.Stdout, nextSteps)
	}
	// init never writes to the home folder, so it only says what install would add.
	if _, skill, err := personalSkillState(); err == nil && skill != current && !pluginBrings(skill) {
		fmt.Fprintf(c.Stdout, "\ntip: %s\n", personalSkillTips[skill])
	}
	c.warnPath()
	return OK, nil
}

// personalSkillTips say what to do about a personal skill that is not as this
// binary would write it.
var personalSkillTips = map[fileState]string{
	missing:  "markitect install tells Claude Code what Markitect is in every folder, also in one without a project yet.",
	outdated: "your personal skill comes from another markitect; markitect update refreshes it.",
}

// installHarness writes the harness of the work for Claude Code, its agents as
// the configuration sets them, and removes every other file of it the binary no
// longer ships.
func installHarness(ws Workspace, cfg harness.Config) error {
	want := map[string][]byte{}
	for _, f := range renderHarness(cfg) {
		want[f.Path] = f.Data
	}
	for _, dir := range []string{".claude/agents", ".claude/skills", ".claude/workflows"} {
		existing, err := ws.List(dir)
		if err != nil {
			return err
		}
		for _, f := range existing {
			_, keep := want[f]
			owned := strings.HasPrefix(strings.TrimPrefix(f, dir+"/"), skills.HarnessPrefix)
			if !keep && owned && !shippedSkill(f) {
				if err := ws.Remove(f); err != nil {
					return err
				}
			}
		}
	}
	for file, data := range want {
		if err := ws.Write(file, data); err != nil {
			return err
		}
	}
	return nil
}

// shippedSkill says a file belongs to a skill installSkills looks after.
func shippedSkill(file string) bool {
	for _, f := range skills.Files() {
		if ".claude/skills/"+f.Path == file {
			return true
		}
	}
	return false
}

// installAttributes has Git merge the record by keeping the lines of both sides
// (DEC-099): status takes the line that matches the folder.
func installAttributes(ws Workspace) error {
	return ws.Write(project.AttributesFile, []byte(project.Attributes))
}

// guardHooks hold the rights of the four roles (DEC-100): only the reviewer
// records verdicts, so that the agent that built does not verify its own work; a
// subagent writes only what its role may; and the builder starts only the
// reviewer. The first runs only before a shell command that records, the second
// before every tool that writes a file, the third before every tool that starts
// an agent.
func guardHooks() []any {
	return []any{
		map[string]any{"matcher": "Bash", "hooks": []any{map[string]any{"type": "command", "if": run.MarkRule, "command": run.HookCommand}}},
		map[string]any{"matcher": run.EditTools, "hooks": []any{map[string]any{"type": "command", "command": run.HookCommand}}},
		map[string]any{"matcher": run.AgentTools, "hooks": []any{map[string]any{"type": "command", "command": run.HookCommand}}},
	}
}

// guardedHooks returns the settings' hooks with the guard in place of one an
// earlier markitect wrote, or why it cannot add it without losing what is there.
func guardedHooks(settings map[string]any) (map[string]any, string) {
	hooks := map[string]any{}
	if v, ok := settings["hooks"]; ok {
		if hooks, ok = v.(map[string]any); !ok {
			return nil, "they are not an object"
		}
	}
	var pre []any
	if v, ok := hooks["PreToolUse"]; ok {
		if pre, ok = v.([]any); !ok {
			return nil, "its PreToolUse hooks are not a list"
		}
	}
	kept := []any{}
	for _, entry := range pre {
		if !ownGuard(entry) {
			kept = append(kept, entry)
		}
	}
	hooks["PreToolUse"] = append(kept, guardHooks()...)
	return hooks, ""
}

// worktreeBase is the base of the worktrees Claude Code makes, which init sets so
// that a builder in a worktree of its own starts from the owner's branch and not
// from the repository's default branch.
const worktreeBase = "head"

// ownGuard says whether a PreToolUse entry is one an earlier markitect wrote: a
// group whose every hook runs HookCommand. The current one replaces it.
func ownGuard(entry any) bool {
	group, ok := entry.(map[string]any)
	if !ok {
		return false
	}
	hooks, ok := group["hooks"].([]any)
	if !ok || len(hooks) == 0 {
		return false
	}
	for _, h := range hooks {
		hook, ok := h.(map[string]any)
		if !ok || hook["command"] != run.HookCommand {
			return false
		}
	}
	return true
}

// ensureSettings adds the hooks to the project's settings, in place of ones an
// earlier markitect wrote, sets the base of Claude Code's worktrees unless the
// settings name one, and keeps what else they hold. Settings it cannot read it
// leaves as they are; a part of them with an unexpected shape it leaves as it is
// and writes the other. Either way it says what to add by hand.
func (c CLI) ensureSettings(ws Workspace) error {
	settings := map[string]any{}
	addHooks := fmt.Sprintf("add the hooks by hand: %q before Bash for %s, before %s and before %s", run.HookCommand, run.MarkRule, run.EditTools, run.AgentTools)
	setBase := fmt.Sprintf("set worktree.baseRef to %q by hand", worktreeBase)
	lacks := func(why string) error {
		fmt.Fprintf(c.Stdout, "left %s as it is, %s; %s, and %s\n", run.SettingsFile, why, addHooks, setBase)
		return nil
	}
	guarded := false
	if ws.Exists(run.SettingsFile) {
		data, err := ws.Read(run.SettingsFile)
		if err != nil {
			return err
		}
		guarded = run.HasGuard(data)
		dec := json.NewDecoder(bytes.NewReader(data))
		dec.UseNumber()
		if err := dec.Decode(&settings); err != nil {
			return lacks(fmt.Sprintf("because it is not valid JSON (%v)", err))
		}
		if _, err := dec.Token(); err != io.EOF {
			return lacks("because it holds more than one JSON value")
		}
		if settings == nil {
			return lacks("because it is not a JSON object")
		}
	}
	var said, left []string
	if !guarded {
		if hooks, why := guardedHooks(settings); why != "" {
			left = append(left, fmt.Sprintf("left the hooks in %s as they are, because %s; %s", run.SettingsFile, why, addHooks))
		} else {
			settings["hooks"] = hooks
			said = append(said, fmt.Sprintf("added the hooks that hold the agents' rights to %s", run.SettingsFile))
		}
	}
	setBaseRef := fmt.Sprintf("set worktree.baseRef to %q in %s, so that a builder's worktree starts from your branch", worktreeBase, run.SettingsFile)
	switch w := settings["worktree"].(type) {
	case nil: // absent, or null
		settings["worktree"] = map[string]any{"baseRef": worktreeBase}
		said = append(said, setBaseRef)
	case map[string]any:
		if w["baseRef"] == nil {
			w["baseRef"] = worktreeBase
			said = append(said, setBaseRef)
		}
	default:
		left = append(left, fmt.Sprintf("left worktree in %s as it is, because it is not an object; %s", run.SettingsFile, setBase))
	}
	for _, line := range left {
		fmt.Fprintln(c.Stdout, line)
	}
	if len(said) == 0 {
		return nil
	}
	var b bytes.Buffer
	enc := json.NewEncoder(&b)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(settings); err != nil {
		return err
	}
	if err := ws.Write(run.SettingsFile, b.Bytes()); err != nil {
		return err
	}
	for _, line := range said {
		fmt.Fprintln(c.Stdout, line)
	}
	return nil
}

// installSkills writes every skill file into each skill folder and removes
// every other file of a markitect-* skill: files and skills the binary no
// longer ships.
func installSkills(ws Workspace) ([]string, error) {
	want := map[string][]byte{}
	for _, f := range skills.Files() {
		for _, dir := range SkillDirs {
			want[dir+"/"+f.Path] = f.Data
		}
	}
	for _, dir := range SkillDirs {
		existing, err := ws.List(dir)
		if err != nil {
			return nil, err
		}
		for _, f := range existing {
			if _, keep := want[f]; !keep && strings.HasPrefix(strings.TrimPrefix(f, dir+"/"), "markitect-") && !harnessFile(f) {
				if err := ws.Remove(f); err != nil {
					return nil, err
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

// harnessFile says a file belongs to the harness installHarness looks after.
func harnessFile(file string) bool {
	for _, f := range skills.Harness() {
		if f.Path == file {
			return true
		}
	}
	return false
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

// writeLock writes the lock from the manifest and the Libraries it names.
func (c CLI) writeLock(ws Workspace) (int, error) {
	data, err := ws.Read(project.ManifestFile)
	if err != nil {
		return Misused, fmt.Errorf("read the manifest: %w; run markitect init", err)
	}
	manifest, parsed := model.ParseManifest(project.ManifestFile, data)
	// An earlier manifest's model is no matter of the lock: update moves it.
	var ds model.Diagnostics
	for _, d := range parsed {
		if d.Code != model.CodeManifestModel {
			ds = append(ds, d)
		}
	}
	// The same files the walk finds, as check reads them.
	files, err := ws.Files()
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

// manifestTemplate is a new project's manifest.
const manifestTemplate = `# The Libraries this project uses, with their versions. Run markitect tidy after a change.
libraries:
  std: %s
`

// instructionsFile is the file coding agents read when a session starts. Claude
// Code reads it through CLAUDE.md, which holds only an import of it.
const instructionsFile = "AGENTS.md"

// instructionsTemplate tells any session what kind of project this is. The
// agents read it too, so it holds no process.
const instructionsTemplate = "# %s\n\n" + skills.About + "\n" +
	"A change to what the project does starts in the spec, designed with the `markitect-design` skill and accepted by the owner.\n\n" +
	"`markitect` is a command on the path, and `markitect help` lists what it does.\n\n" +
	"A builder or reviewer of Markitect work takes what to do from `markitect status`, and the spec stays as it is.\n"

// nextSteps ends the init of a new project. The session that ran init started
// before the skills existed, and Claude Code does not watch a skills folder that
// was not there when a session started.
const nextSteps = `next: run /markitect-design in Claude Code here. It asks what the project is and starts the spec with you, or draws it from the code already here.
A session that was open before init runs /reload-skills first, or reads .claude/skills/markitect-design/SKILL.md.
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
