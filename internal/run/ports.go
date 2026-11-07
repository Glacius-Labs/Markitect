package run

import "github.com/Glacius-Labs/Markitect/internal/project"

// Folder reads a project folder without Git (DEC-097).
type Folder interface {
	// Root is the project folder that holds dir: the nearest one with Markitect's
	// manifest in it.
	Root(dir string) (string, error)
	// Open walks the project folder at root: its files, without what the
	// project's .gitignore files exclude, and the symbolic links among them with
	// where each points.
	Open(root string) (files project.Repository, links map[string]string, err error)
}
