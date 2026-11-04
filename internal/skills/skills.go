// Package skills holds what ships inside the markitect binary for agents: the
// modeling skill and the Extract skill built on it, which markitect init installs
// where coding agents look for skills, and the harness of a Run for Claude Code.
package skills

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed markitect-modeling markitect-extract
var files embed.FS

//go:embed all:claude
var claudeFiles embed.FS

// File is one file of a skill, relative to the skills folder, such as
// "markitect-modeling/SKILL.md".
type File struct {
	Path string
	Data []byte
}

// Names returns the skill names, sorted.
func Names() []string { return []string{"markitect-extract", "markitect-modeling"} }

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
