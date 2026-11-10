package setup

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// harnessDirs are the folders where a file of Markitect's for agents may lie.
// The harness is for Claude Code, in .claude/; a copy of a skill in the folder
// other coding agents read is removed, since Markitect supports none of them.
var harnessDirs = []string{".claude/agents", ".claude/skills", ".claude/workflows", ".agents/skills"}

// InstallHarness writes the harness for Claude Code, its skills, its agents as
// the configuration sets them and its workflow, and removes every other file of
// Markitect's in its folders: what the binary no longer ships.
func InstallHarness(f Folder, cfg harness.Config) error {
	want := map[string][]byte{}
	for _, file := range renderHarness(cfg) {
		want[file.Path] = file.Data
	}
	for _, dir := range harnessDirs {
		existing, err := f.List(dir)
		if err != nil {
			return err
		}
		for _, file := range existing {
			_, keep := want[file]
			if !keep && strings.HasPrefix(strings.TrimPrefix(file, dir+"/"), skills.HarnessPrefix) {
				if err := f.Remove(file); err != nil {
					return err
				}
			}
		}
	}
	for file, data := range want {
		if err := f.Write(file, data); err != nil {
			return err
		}
	}
	return nil
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

// WriteAgents writes the agents' files of a project from the configuration and
// returns the files that changed.
func WriteAgents(f Folder, cfg harness.Config) ([]string, error) {
	r := &refresher{Folder: f}
	for _, file := range renderHarness(cfg) {
		if _, ok := harness.RoleOf(file.Path); !ok {
			continue
		}
		if err := r.Write(file.Path, file.Data); err != nil {
			return nil, err
		}
	}
	return r.files, nil
}

// ReadHarness reads a project's harness configuration; a missing file, or no
// project, is an empty one.
func ReadHarness(f Folder) (harness.File, error) {
	if f == nil || !f.Exists(project.HarnessFile) {
		return harness.File{}, nil
	}
	data, err := f.Read(project.HarnessFile)
	if err != nil {
		return harness.File{}, err
	}
	file, err := harness.Parse(data)
	if err != nil {
		return harness.File{}, fmt.Errorf("%s: %w", project.HarnessFile, err)
	}
	return file, nil
}
