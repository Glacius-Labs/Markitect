package setup

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/skills"
)

// SkillDirs are the folders where coding agents look for project skills:
// Claude Code and the Agent Skills folder other agents such as Codex read.
var SkillDirs = []string{".claude/skills", ".agents/skills"}

// InstallSkills writes every skill file into each skill folder and removes
// every other file of a markitect-* skill: files and skills the binary no
// longer ships. It returns the names of the skills.
func InstallSkills(f Folder) ([]string, error) {
	want := map[string][]byte{}
	for _, file := range skills.Files() {
		for _, dir := range SkillDirs {
			want[dir+"/"+file.Path] = file.Data
		}
	}
	for _, dir := range SkillDirs {
		existing, err := f.List(dir)
		if err != nil {
			return nil, err
		}
		for _, file := range existing {
			if _, keep := want[file]; !keep && strings.HasPrefix(strings.TrimPrefix(file, dir+"/"), "markitect-") && !harnessFile(file) {
				if err := f.Remove(file); err != nil {
					return nil, err
				}
			}
		}
	}
	for file, data := range want {
		if err := f.Write(file, data); err != nil {
			return nil, err
		}
	}
	return skills.Names(), nil
}

// harnessFile says a file belongs to the harness InstallHarness looks after.
func harnessFile(file string) bool {
	for _, f := range skills.Harness() {
		if f.Path == file {
			return true
		}
	}
	return false
}

// InstallHarness writes the harness of the work for Claude Code, its agents as
// the configuration sets them, and removes every other file of it the binary no
// longer ships.
func InstallHarness(f Folder, cfg harness.Config) error {
	want := map[string][]byte{}
	for _, file := range renderHarness(cfg) {
		want[file.Path] = file.Data
	}
	for _, dir := range []string{".claude/agents", ".claude/skills", ".claude/workflows"} {
		existing, err := f.List(dir)
		if err != nil {
			return err
		}
		for _, file := range existing {
			_, keep := want[file]
			owned := strings.HasPrefix(strings.TrimPrefix(file, dir+"/"), skills.HarnessPrefix)
			if !keep && owned && !shippedSkill(file) {
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

// shippedSkill says a file belongs to a skill InstallSkills looks after.
func shippedSkill(file string) bool {
	for _, f := range skills.Files() {
		if ".claude/skills/"+f.Path == file {
			return true
		}
	}
	return false
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
