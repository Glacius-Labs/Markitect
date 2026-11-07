package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/model"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/run"
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
	opened, err := c.Git.Open(ctx, o.repo)
	if err != nil {
		// A folder in no repository becomes one: a Run's work goes in and out as
		// commits. One inside a repository Git would not open, for example for its
		// owner, keeps the error rather than getting a repository of its own.
		if insideRepository(o.repo) || c.Git.Create(ctx, o.repo) != nil {
			return Misused, err
		}
		if opened, err = c.Git.Open(ctx, o.repo); err != nil {
			return Misused, err
		}
		fmt.Fprintln(c.Stdout, "created a Git repository")
	}
	rec := &recorder{Workspace: opened}
	var ws Workspace = rec
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
	if err := writeNew(project.ManifestFile, []byte(fmt.Sprintf(manifestTemplate, libraries.Version))); err != nil {
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
	if ws.Exists(instructionsFile) {
		fmt.Fprintf(c.Stdout, "left %s as it is; add that this project has a spec in %s, which is the authority\n", instructionsFile, project.ModelDir)
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
	installed, err := installSkills(ws)
	if err != nil {
		return Misused, err
	}
	fmt.Fprintf(c.Stdout, "installed the skills %s into %s\n", strings.Join(installed, ", "), strings.Join(SkillDirs, " and "))
	if err := installHarness(ws); err != nil {
		return Misused, err
	}
	fmt.Fprintln(c.Stdout, "installed the harness: the apply and verify skills, and the builder and reviewer agents")
	if err := c.ensureGuard(ws); err != nil {
		return Misused, err
	}
	switch committed, err := ws.CommitOwn(rec.files, initMessage); {
	case err != nil:
		fmt.Fprintf(c.Stdout, "could not commit what init wrote, so commit it yourself before the first apply; if Git ignores .claude/, the harness stays on disk and that is enough: %v\n", err)
	case committed:
		fmt.Fprintln(c.Stdout, "committed what init wrote")
	}
	fmt.Fprintf(c.Stdout, "\n%s\n", skills.About)
	if len(modelFiles) == 0 {
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

const initMessage = "Set up Markitect"

// recorder notes what init writes, so that it commits that and nothing else.
type recorder struct {
	Workspace
	files []string
}

func (r *recorder) Write(file string, data []byte) error {
	r.files = append(r.files, file)
	return r.Workspace.Write(file, data)
}

func (r *recorder) Remove(file string) error {
	r.files = append(r.files, file)
	return r.Workspace.Remove(file)
}

// installHarness writes the harness of a Run for Claude Code and removes every
// other file of it the binary no longer ships.
func installHarness(ws Workspace) error {
	want := map[string][]byte{}
	for _, f := range skills.Harness() {
		want[f.Path] = f.Data
	}
	for _, dir := range []string{".claude/agents", ".claude/skills"} {
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

// insideRepository says whether dir or a folder above it holds a .git entry.
func insideRepository(dir string) bool {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return false
	}
	for {
		if _, err := os.Lstat(filepath.Join(abs, ".git")); err == nil {
			return true
		}
		parent := filepath.Dir(abs)
		if parent == abs {
			return false
		}
		abs = parent
	}
}

// guardHook lets only the reviewer store answers, so that an agent cannot mark
// its own work as done, and only the owner's session begin the work. It runs only
// before the commands it guards, one handler each.
func guardHook() map[string]any {
	var handlers []any
	for _, rule := range run.GuardedCommands {
		handlers = append(handlers, map[string]any{"type": "command", "if": rule, "command": run.HookCommand})
	}
	return map[string]any{"matcher": "Bash", "hooks": handlers}
}

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

// ensureGuard adds the hook to the project's settings, in place of one an
// earlier markitect wrote, and keeps what else they hold. Settings it cannot
// read, or whose hooks have an unexpected shape, it leaves as they are and says
// so; a run then refuses to start until the hook is there.
func (c CLI) ensureGuard(ws Workspace) error {
	settings := map[string]any{}
	lacks := func(why string) error {
		fmt.Fprintf(c.Stdout, "left %s as it is, %s; add the hook that lets only the reviewer answer by hand: %q before Bash, for %s\n", run.SettingsFile, why, run.HookCommand, strings.Join(run.GuardedCommands, " and "))
		return nil
	}
	if ws.Exists(run.SettingsFile) {
		data, err := ws.Read(run.SettingsFile)
		if err != nil {
			return err
		}
		if run.HasGuard(data) {
			return nil
		}
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
	hooks := map[string]any{}
	if v, ok := settings["hooks"]; ok {
		if hooks, ok = v.(map[string]any); !ok {
			return lacks("because its hooks are not an object")
		}
	}
	var pre []any
	if v, ok := hooks["PreToolUse"]; ok {
		if pre, ok = v.([]any); !ok {
			return lacks("because its PreToolUse hooks are not a list")
		}
	}
	kept := []any{}
	for _, entry := range pre {
		if !ownGuard(entry) {
			kept = append(kept, entry)
		}
	}
	hooks["PreToolUse"] = append(kept, guardHook())
	settings["hooks"] = hooks
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
	fmt.Fprintf(c.Stdout, "added the hook that lets only the reviewer answer and only your session begin the work to %s\n", run.SettingsFile)
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
	// The same files Git sees, as check reads them.
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
const manifestTemplate = `# The Libraries this project uses, with their versions. Run markitect lock after a change.
# model, optional, names the model a run's agents use; without it they use sonnet.
libraries:
  std: %s
`

// instructionsFile is the file coding agents read when a session starts. Claude
// Code reads it through CLAUDE.md, which holds only an import of it.
const instructionsFile = "AGENTS.md"

// instructionsTemplate tells any session what kind of project this is. A Run's
// sessions read it too, so it holds no process.
const instructionsTemplate = "# %s\n\n" + skills.About + "\n" +
	"A change to what the project does starts in the spec, written with the `markitect-modeling` skill and accepted by the owner. A new project starts with the `markitect-start` skill, and a repository that already has code with `markitect-extract`.\n\n" +
	"`markitect` is a command on the path, and `markitect help` lists what it does.\n\n" +
	"Inside a Markitect run, your task says what to do, and the spec stays as it is.\n"

// nextSteps ends the init of a new project. The session that ran init started
// before the skills existed, and Claude Code does not watch a skills folder that
// was not there when a session started.
const nextSteps = `next: run /markitect-start in Claude Code here. It asks what the project is and starts the spec with you.
A session that was open before init runs /reload-skills first, or reads .claude/skills/markitect-start/SKILL.md.
If the repository already has code, use /markitect-extract instead.
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
