package cli

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing/fstest"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/edit"
	"github.com/Glacius-Labs/Markitect/internal/library"
	"github.com/Glacius-Labs/Markitect/internal/state"
)

// initTemplates are the templates init applies, each into a folder of the
// spec of its own name.
var initTemplates = []string{"project", "product", "architecture", "factory"}

// skillFolders are the folders below the project folder where the skills of
// the standard lie, for Claude Code and for Codex alike.
var skillFolders = []string{".claude/skills", ".agents/skills"}

// agentsFile is the file of the project that names the spec and the skills
// for every agent.
const agentsFile = "AGENTS.md"

// initResult is the JSON document of init: the version it pins, the spec
// folder, and the files and folders it created and the files it appended an
// entry to, each below the project folder.
type initResult struct {
	header
	Version  string   `json:"version"`
	Spec     string   `json:"spec"`
	Created  []string `json:"created"`
	Appended []string `json:"appended"`
}

// initProject sets up the folder -C names, or the working directory: it lays
// out .markitect/, the spec with the four templates applied, the skills of
// the standard and the entry in AGENTS.md, as one transaction. It refuses,
// writing nothing, when .markitect/ exists, when a folder it would create in
// the spec or a skill it would copy exists, or when AGENTS.md is no file; a
// failed init leaves nothing behind.
func initProject(o options, stdout io.Writer) (int, error) {
	dir, err := workingFolder(o)
	if err != nil {
		return ExitFailed, err
	}
	spec := state.DefaultSpec
	if o.specSet {
		if spec, err = state.SpecPath(o.spec); err != nil {
			return ExitFailed, err
		}
	}
	target := filepath.Join(dir, state.Dir)
	if _, err := os.Lstat(target); err == nil {
		return ExitFailed, fmt.Errorf("%s/ exists already in %s", state.Dir, dir)
	}
	if err := specInside(dir, spec); err != nil {
		return ExitFailed, err
	}
	info, statErr := os.Stat(filepath.Join(dir, filepath.FromSlash(spec)))
	if statErr == nil && !info.IsDir() {
		return ExitFailed, fmt.Errorf("the spec folder %s is a file", spec)
	}
	plan, err := planInit(dir, spec, statErr == nil)
	if err != nil {
		return ExitFailed, err
	}
	if err := applyChanges(dir, plan.changes); err != nil {
		return ExitFailed, err
	}
	text := ""
	for _, c := range plan.created {
		text += "created " + c + "\n"
	}
	for _, a := range plan.appended {
		text += "appended an entry to " + a + "\n"
	}
	return output(o, stdout, initResult{newHeader("init"), buildinfo.Version(), spec, plan.created, nonNil(plan.appended)}, text)
}

// initPlan is what init does: the files to write, below the project folder,
// and what to tell about them.
type initPlan struct {
	changes           edit.Changes
	created, appended []string
}

// planInit computes everything init writes and refuses, with every reason
// found, when anything is in the way; nothing is written here. The spec folder
// may not exist yet.
func planInit(dir, spec string, specExists bool) (initPlan, error) {
	var plan initPlan
	var problems []string
	var writes []edit.Write
	project := func(p string) string { return filepath.Join(dir, filepath.FromSlash(p)) }

	plan.created = []string{state.Dir + "/"}
	var version edit.Write
	for _, f := range state.InitFiles(buildinfo.Version(), spec) {
		w := edit.Write{Path: f.Path, Data: f.Data, Create: true}
		if f.Path == state.VersionFile {
			// The version is written last: it makes the folder a project.
			version = w
			continue
		}
		writes = append(writes, w)
	}

	// The spec: each template into a folder of its own name, with the
	// packages it needs, as apply would. Each apply sees what the ones before
	// it planned, so that a shared package is copied once.
	if !specExists {
		plan.created = append(plan.created, spec+"/")
	}
	specFS := layered{base: os.DirFS(project(spec)), planned: fstest.MapFS{}}
	for _, tpl := range initTemplates {
		if _, err := fs.Stat(specFS, tpl); err == nil {
			problems = append(problems, path.Join(spec, tpl)+": already exists")
			continue
		}
		a, err := edit.Apply(specFS, library.FS(), tpl, tpl)
		if err != nil {
			var r *edit.Refusal
			if errors.As(err, &r) {
				for _, p := range r.Problems {
					problems = append(problems, path.Join(spec, tpl)+": "+p)
				}
				continue
			}
			return plan, err
		}
		plan.created = append(plan.created, path.Join(spec, tpl)+"/")
		for _, p := range a.Packages {
			plan.created = append(plan.created, path.Join(spec, "packages", p)+"/")
		}
		for _, w := range a.Changes.Writes {
			specFS.planned[w.Path] = &fstest.MapFile{Data: w.Data}
			writes = append(writes, edit.Write{Path: path.Join(spec, w.Path), Data: w.Data, Create: true})
		}
	}

	// The skills of the standard, whole and byte for byte, in both places.
	names, err := library.SkillNames()
	if err != nil {
		return plan, err
	}
	for _, name := range names {
		var files []string
		err := fs.WalkDir(library.Skills(), name, func(p string, e fs.DirEntry, err error) error {
			if err == nil && !e.IsDir() {
				files = append(files, p)
			}
			return err
		})
		if err != nil {
			return plan, err
		}
		for _, folder := range skillFolders {
			at := path.Join(folder, name)
			if _, err := os.Lstat(project(at)); err == nil {
				problems = append(problems, at+": already exists")
				continue
			}
			plan.created = append(plan.created, at+"/")
			for _, f := range files {
				data, err := fs.ReadFile(library.Skills(), f)
				if err != nil {
					return plan, err
				}
				writes = append(writes, edit.Write{Path: path.Join(folder, f), Data: data, Create: true})
			}
		}
	}

	// The entry in AGENTS.md: the file is created when it is missing, and
	// the entry is appended as a section of its own when it exists, so that
	// what stands there is never changed.
	switch info, err := os.Lstat(project(agentsFile)); {
	case errors.Is(err, fs.ErrNotExist):
		writes = append(writes, edit.Write{Path: agentsFile, Data: []byte("# Agent instructions\n\n" + agentsEntry(spec)), Create: true})
		plan.created = append(plan.created, agentsFile)
	case err != nil:
		return plan, err
	case !info.Mode().IsRegular():
		problems = append(problems, agentsFile+": is a link or a folder, not a file; init appends its entry to a file")
	default:
		existing, err := os.ReadFile(project(agentsFile))
		if err != nil {
			return plan, err
		}
		writes = append(writes, edit.Write{Path: agentsFile, Data: appendEntry(existing, agentsEntry(spec))})
		plan.appended = append(plan.appended, agentsFile)
	}
	if len(problems) > 0 {
		return plan, &edit.Refusal{Problems: problems}
	}
	plan.changes.Writes = append(writes, version)
	return plan, nil
}

// agentsEntry is the section init writes into AGENTS.md: where the spec is,
// where the way of working stands, where the skills lie and where a person
// starts.
func agentsEntry(spec string) string {
	return "## Markitect\n\n" +
		"The spec of this project lies in `" + spec + "/`; `.markitect/spec` names the folder. The way of working stands in its `factory` part, `" + spec + "/factory/`.\n" +
		"The skills of the standard lie in `.claude/skills/` and `.agents/skills/`. A person starts their work with the skill `norman`.\n"
}

// appendEntry returns existing with the entry as a section of its own at the
// end, in the line ends the file uses; what stands in existing stays as it is.
func appendEntry(existing []byte, entry string) []byte {
	eol := "\n"
	if bytes.Contains(existing, []byte("\r\n")) {
		eol = "\r\n"
		entry = strings.ReplaceAll(entry, "\n", eol)
	}
	out := append([]byte(nil), existing...)
	if len(out) > 0 && out[len(out)-1] != '\n' {
		out = append(out, eol...)
	}
	if len(out) > 0 && !bytes.HasSuffix(out, []byte(eol+eol)) {
		out = append(out, eol...)
	}
	return append(out, entry...)
}

// layered is a folder with the files that are planned but not written yet: a
// file system in which the second template of init sees the packages the
// first one copies.
type layered struct {
	base    fs.FS
	planned fstest.MapFS
}

func (l layered) Open(name string) (fs.File, error) {
	if f, err := l.planned.Open(name); err == nil {
		return f, nil
	}
	return l.base.Open(name)
}
