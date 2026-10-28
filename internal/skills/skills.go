// Package skills holds the agent skills that ship inside the markitect
// binary: the modeling skill and the Extract skill built on it. markitect
// init installs them where coding agents look for skills.
package skills

import (
	"embed"
	"io/fs"
	"sort"
)

//go:embed markitect-modeling markitect-extract
var files embed.FS

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
