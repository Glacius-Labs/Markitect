package cli

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"go.yaml.in/yaml/v3"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// userDir is the folder of the user's own Markitect files in the home folder.
const userDir = ".markitect"

// userHarnessFile is the user's harness configuration, which holds what applies
// to all of the user's projects and is committed with none (DEC-103).
func userHarnessFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, userDir, harness.FileName), nil
}

// Comments a new configuration file starts with.
const (
	userHarnessHead = `# Markitect's harness configuration for all of your projects (DEC-103). A
# project's .markitect/harness.yaml beats it, key by key, and it beats the
# built-in values. markitect config --user changes it; markitect config --list
# shows each value in force and where it comes from.`
	projectHarnessHead = `# The harness configuration of this project (DEC-103), committed with it. It
# beats each user's ~/.markitect/harness.yaml, key by key. markitect config
# changes it and writes the agents' files; markitect config --list shows each
# value in force and where it comes from.`
)

// readUserHarness reads the user's configuration; a missing file is an empty one.
func readUserHarness() (harness.File, string, error) {
	file, err := userHarnessFile()
	if err != nil {
		return harness.File{}, "", err
	}
	data, err := os.ReadFile(file)
	if errors.Is(err, os.ErrNotExist) {
		return harness.File{}, file, nil
	}
	if err != nil {
		return harness.File{}, file, err
	}
	f, err := harness.Parse(data)
	if err != nil {
		return harness.File{}, file, fmt.Errorf("%s: %w", file, err)
	}
	return f, file, nil
}

// readProjectHarness reads a project's configuration; a missing file is an empty
// one.
func readProjectHarness(ws Workspace) (harness.File, error) {
	if ws == nil || !ws.Exists(project.HarnessFile) {
		return harness.File{}, nil
	}
	data, err := ws.Read(project.HarnessFile)
	if err != nil {
		return harness.File{}, err
	}
	f, err := harness.Parse(data)
	if err != nil {
		return harness.File{}, fmt.Errorf("%s: %w", project.HarnessFile, err)
	}
	return f, nil
}

// harnessConfig is the configuration in force for a project, or for none when ws
// is nil.
func harnessConfig(ws Workspace) (harness.Config, error) {
	user, _, err := readUserHarness()
	if err != nil {
		return harness.Config{}, err
	}
	proj, err := readProjectHarness(ws)
	if err != nil {
		return harness.Config{}, err
	}
	return harness.Resolve(user, proj), nil
}

// renderHarness returns the harness as a project gets it: its agents with the
// model and effort the configuration gives their role.
func renderHarness(cfg harness.Config) []skills.File {
	files := skills.Harness()
	for i, f := range files {
		if role, ok := harness.RoleOf(f.Path); ok {
			files[i].Data = cfg.RenderAgent(role, f.Data)
		}
	}
	return files
}

// writeAgents writes the agents' files of a project from the configuration and
// returns the files that changed.
func writeAgents(ws Workspace, cfg harness.Config) ([]string, error) {
	r := &refresher{Workspace: ws}
	for _, f := range renderHarness(cfg) {
		if _, ok := harness.RoleOf(f.Path); !ok {
			continue
		}
		if err := r.Write(f.Path, f.Data); err != nil {
			return nil, err
		}
	}
	return r.files, nil
}

const configUsage = `usage:
  markitect config [--repo DIR] [--user] KEY VALUE   set a value and refresh the agents' files
  markitect config [--repo DIR] [--user] --unset KEY remove a value
  markitect config [--repo DIR] KEY                  print the value in force
  markitect config [--repo DIR] --list [--json]      every value in force and where it comes from
keys: agents.ROLE.model, agents.ROLE.effort (ROLE: builder, reviewer, architect, auditor),
      strategy.default, strategy.SKILL (SKILL: build, review, cleanup, bughunt, audit),
      workflows.NAME (the path of a custom workflow's script)`

// config reads and changes the harness configuration (DEC-103): the project's
// .markitect/harness.yaml, or with --user the user's ~/.markitect/harness.yaml.
// A change rewrites the agents' files of the project here, since their front
// matter comes from it.
func (c CLI) config(_ context.Context, args []string) (int, error) {
	var o options
	var user, list, unset bool
	fs := c.flags("config", &o)
	fs.BoolVar(&user, "user", false, "change the user's ~/.markitect/harness.yaml, for all of your projects, instead of the project's")
	fs.BoolVar(&list, "list", false, "print every value in force and where it comes from")
	fs.BoolVar(&unset, "unset", false, "remove the key")
	fs.Usage = func() { fmt.Fprintln(c.Stderr, configUsage) }
	positional, err := parse(fs, args)
	if err != nil {
		return Misused, err
	}
	// The project is optional: --user and --list work in any folder.
	var ws Workspace
	if found, err := c.Folders.Project(o.repo); err == nil {
		ws = found
	}
	switch {
	case list && !unset && len(positional) == 0:
		return c.configList(ws, o.json)
	case list || o.json:
		return Misused, fmt.Errorf("--list takes no key, and --json goes with --list\n%s", configUsage)
	case unset && len(positional) == 1:
		return c.configChange(ws, user, positional[0], "", true)
	case !unset && len(positional) == 2:
		return c.configChange(ws, user, positional[0], positional[1], false)
	case !unset && !user && len(positional) == 1:
		cfg, err := harnessConfig(ws)
		if err != nil {
			return Misused, err
		}
		if err := harness.CheckKey(positional[0]); err != nil {
			return Misused, err
		}
		v, ok := cfg.Get(positional[0])
		if !ok {
			return Failed, nil
		}
		fmt.Fprintln(c.Stdout, v.Value)
		return OK, nil
	}
	return Misused, fmt.Errorf("%s", configUsage)
}

// configList prints every value in force with where it comes from.
func (c CLI) configList(ws Workspace, asJSON bool) (int, error) {
	cfg, err := harnessConfig(ws)
	if err != nil {
		return Misused, err
	}
	_, userFile, err := readUserHarness()
	if err != nil {
		return Misused, err
	}
	where := func(v harness.Value) string {
		switch v.Origin {
		case harness.User:
			return userFile
		case harness.Project:
			return project.HarnessFile
		}
		return ""
	}
	problems := cfg.Problems()
	if asJSON {
		type value struct {
			harness.Value
			Origin string `json:"origin"`
			File   string `json:"file,omitempty"`
		}
		values := []value{}
		for _, v := range cfg.Values() {
			values = append(values, value{Value: v, Origin: v.Origin.String(), File: where(v)})
		}
		if problems == nil {
			problems = []string{}
		}
		if err := c.printJSON(struct {
			envelope
			Values   []value  `json:"values"`
			Problems []string `json:"problems"`
		}{newEnvelope("config"), values, problems}); err != nil {
			return Misused, err
		}
	} else {
		w := tabwriter.NewWriter(c.Stdout, 0, 0, 2, ' ', 0)
		for _, v := range cfg.Values() {
			origin := v.Origin.String()
			if f := where(v); f != "" {
				origin += " (" + f + ")"
			}
			if v.Via != "" {
				origin += ", via " + v.Via
			}
			fmt.Fprintf(w, "%s\t%s\t%s\n", v.Key, v.Value, origin)
		}
		if err := w.Flush(); err != nil {
			return Misused, err
		}
		for _, p := range problems {
			fmt.Fprintf(c.Stdout, "problem: %s\n", p)
		}
	}
	if len(problems) > 0 {
		return Failed, nil
	}
	return OK, nil
}

// configChange sets or removes a key at one level, writes that file, and
// rewrites the agents' files of the project here.
func (c CLI) configChange(ws Workspace, user bool, key, value string, unset bool) (int, error) {
	if !unset {
		if err := harness.Check(key, value); err != nil {
			return Misused, err
		}
	}
	userFile, userPath, err := readUserHarness()
	if err != nil {
		return Misused, err
	}
	if !user && ws == nil {
		return Misused, fmt.Errorf("no folder here or above holds %s: run it in a project, or with --user for all of your projects", project.ManifestFile)
	}
	projFile, err := readProjectHarness(ws)
	if err != nil {
		return Misused, err
	}
	target, name, head := &projFile, project.HarnessFile, projectHarnessHead
	if user {
		target, name, head = &userFile, userPath, userHarnessHead
	}
	if data, _ := target.Bytes(); len(data) == 0 && !unset {
		*target = harness.New(head)
	}
	if unset {
		if !target.Unset(key) {
			fmt.Fprintf(c.Stdout, "%s sets no %s\n", name, key)
			return Failed, nil
		}
	} else if err := target.Set(key, value); err != nil {
		return Misused, err
	}
	cfg := harness.Resolve(userFile, projFile)
	for _, p := range cfg.Problems() {
		if strings.HasPrefix(p, key+" ") {
			return Misused, fmt.Errorf("%s", p)
		}
	}
	data, err := target.Bytes()
	if err != nil {
		return Misused, err
	}
	if user {
		if err := os.MkdirAll(filepath.Dir(userPath), 0o755); err != nil {
			return Misused, err
		}
		if err := os.WriteFile(userPath, data, 0o644); err != nil {
			return Misused, err
		}
	} else if err := ws.Write(project.HarnessFile, data); err != nil {
		return Misused, err
	}
	if unset {
		fmt.Fprintf(c.Stdout, "removed %s from %s\n", key, name)
	} else {
		fmt.Fprintf(c.Stdout, "set %s to %s in %s\n", key, value, name)
	}
	if v, _ := cfg.Get(key); !unset && user && v.Origin == harness.Project {
		fmt.Fprintf(c.Stdout, "note: this project's %s sets %s to %s, which beats yours here\n", project.HarnessFile, key, v.Value)
	}
	if name, ok := strings.CutPrefix(key, "workflows."); ok && !unset {
		c.noteWorkflow(ws, user, name, value)
	}
	for _, p := range cfg.Problems() {
		fmt.Fprintf(c.Stdout, "problem: %s\n", p)
	}
	if ws == nil {
		return OK, nil
	}
	changed, err := writeAgents(ws, cfg)
	if err != nil {
		return Misused, err
	}
	for _, f := range changed {
		fmt.Fprintf(c.Stdout, "updated %s\n", f)
	}
	if !user || len(changed) > 0 {
		fmt.Fprintln(c.Stdout, "Markitect uses no Git: commit the change when you like.")
	}
	return OK, nil
}

// noteWorkflow says when the script a custom workflow names is not there, or not
// where Claude Code looks for saved workflows, which start by name.
func (c CLI) noteWorkflow(ws Workspace, user bool, name, path string) {
	full := path
	switch {
	case strings.HasPrefix(path, "~/") || strings.HasPrefix(path, `~\`):
		if home, err := os.UserHomeDir(); err == nil {
			full = filepath.Join(home, path[2:])
		}
	case !filepath.IsAbs(path) && ws != nil:
		full = filepath.Join(ws.Root(), filepath.FromSlash(path))
	}
	if _, err := os.Stat(full); err != nil {
		fmt.Fprintf(c.Stdout, "note: %s does not exist yet\n", path)
	}
	slashed := filepath.ToSlash(path)
	if !strings.Contains(slashed, ".claude/workflows/") {
		fmt.Fprintf(c.Stdout, "note: Claude Code starts a saved workflow by name only from .claude/workflows/ or ~/.claude/workflows/; put %s there, as %s.js with meta.name %q\n", path, name, name)
	}
}

// Comment lines an earlier manifest template wrote about the agents' model.
var oldManifestModelLines = []string{
	"# model, optional, names the model Markitect's agents use; without it they use sonnet.",
	"# markitect update writes it into their agent files.",
}

// moveManifestModel moves the agents' model that an earlier manifest names into
// the project's harness configuration (DEC-103), as the model of every role the
// configuration does not set yet, and removes it from the manifest. It returns
// the configuration as it is then, and the files it changed, or would change
// when dry. A model that was never a valid name is dropped, as it was ignored.
func moveManifestModel(ws Workspace, proj harness.File, dry bool) (harness.File, []string, error) {
	data, err := ws.Read(project.ManifestFile)
	if err != nil {
		return proj, nil, nil
	}
	var doc yaml.Node
	if yaml.Unmarshal(data, &doc) != nil || len(doc.Content) == 0 || doc.Content[0].Kind != yaml.MappingNode {
		return proj, nil, nil
	}
	top := doc.Content[0]
	var key, value *yaml.Node
	for i := 0; i+1 < len(top.Content); i += 2 {
		if top.Content[i].Value == "model" {
			key, value = top.Content[i], top.Content[i+1]
		}
	}
	if key == nil || value.Kind != yaml.ScalarNode || value.Line != key.Line {
		return proj, nil, nil
	}
	lines := strings.SplitAfter(string(data), "\n")
	var kept []string
	for i, line := range lines {
		bare := strings.TrimRight(line, "\r\n")
		if i == key.Line-1 {
			continue
		}
		dropped := false
		for _, old := range oldManifestModelLines {
			if bare == old {
				dropped = true
			}
		}
		if !dropped {
			kept = append(kept, line)
		}
	}
	changed := []string{project.ManifestFile}
	model := strings.TrimSpace(value.Value)
	if harness.Check("agents.builder.model", model) == nil {
		set := proj.Values()
		if len(set) == 0 {
			if data, _ := proj.Bytes(); len(data) == 0 {
				proj = harness.New(projectHarnessHead)
			}
		}
		moved := false
		for _, role := range harness.Roles {
			k := "agents." + role + ".model"
			if _, ok := set[k]; !ok {
				if err := proj.Set(k, model); err != nil {
					return proj, nil, err
				}
				moved = true
			}
		}
		if moved {
			changed = append(changed, project.HarnessFile)
		}
	}
	if dry {
		return proj, changed, nil
	}
	if err := ws.Write(project.ManifestFile, []byte(strings.Join(kept, ""))); err != nil {
		return proj, nil, err
	}
	if len(changed) > 1 {
		out, err := proj.Bytes()
		if err != nil {
			return proj, nil, err
		}
		if err := ws.Write(project.HarnessFile, out); err != nil {
			return proj, nil, err
		}
	}
	return proj, changed, nil
}
