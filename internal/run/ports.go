package run

import "github.com/Glacius-Labs/Markitect/internal/project"

// Folder reads a project folder without Git (DEC-097).
type Folder interface {
	// Root is the project folder that holds dir: the nearest one with Markitect's
	// manifest in it.
	Root(dir string) (string, error)
	// Open walks the project folder at root: its files, without what the
	// project's .gitignore files exclude.
	Open(root string) (project.Repository, error)
}
