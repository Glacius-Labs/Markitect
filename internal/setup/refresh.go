package setup

import (
	"bytes"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// SameText compares two texts without minding line endings: a checkout may hold a
// file with CRLF, and the binary writes LF.
func SameText(a, b []byte) bool {
	crlf, lf := []byte("\r\n"), []byte("\n")
	return bytes.Equal(bytes.ReplaceAll(a, crlf, lf), bytes.ReplaceAll(b, crlf, lf))
}

// refresher passes on the writes of a refresh that change a file, and notes
// them. A file that already holds the text is left alone and not noted, so the
// notes are the files that differ. With dry set it changes nothing and only
// notes, which is how version and update --check find out what is out of date
// with the same code that update uses to put it right.
type refresher struct {
	Folder
	dry   bool
	files []string
}

func (r *refresher) Write(file string, data []byte) error {
	if old, err := r.Folder.Read(file); err == nil && SameText(old, data) {
		return nil
	}
	r.files = append(r.files, file)
	if r.dry {
		return nil
	}
	return r.Folder.Write(file, data)
}

func (r *refresher) Remove(file string) error {
	r.files = append(r.files, file)
	if r.dry {
		return nil
	}
	return r.Folder.Remove(file)
}

// RefreshHarness brings the skills, the agents, the workflows, the hooks, the
// record's Git attributes and the ignore line for Claude Code's worktrees of a
// project in line with this binary and the harness configuration, as init does,
// and returns the files that differed. With dry it changes nothing. It never
// touches the spec, AGENTS.md or CLAUDE.md. It also returns what EnsureSettings
// says, also when it fails.
func RefreshHarness(f Folder, cfg harness.Config, dry bool) (files, said []string, err error) {
	r := &refresher{Folder: f, dry: dry}
	if err := InstallHarness(r, cfg); err != nil {
		return nil, nil, err
	}
	if said, err = EnsureSettings(r); err != nil {
		return nil, said, err
	}
	if err := InstallAttributes(r); err != nil {
		return nil, said, err
	}
	if _, err := EnsureIgnore(r); err != nil {
		return nil, said, err
	}
	sort.Strings(r.files)
	return r.files, said, nil
}

// Refreshed is what a refresh found in a project.
type Refreshed struct {
	// Files are the skills, agents and settings that differ.
	Files []string
	// Said is what the refresh of the settings says. A dry refresh returns it
	// too, although it wrote nothing, so its caller does not print it.
	Said []string
	// Lock says the lock differs, or the manifest with it.
	Lock bool
	// Named are the standard Libraries the manifest names at the version this
	// binary ships, where it named another: "std 1.1.0, where it named 1.0.0".
	Named []string
	// LockProblems are the errors of the manifest, which keep the lock as it is:
	// it names a Library the binary cannot supply.
	LockProblems spec.Diagnostics
}

// Refresh finds what differs in a project from what this binary would write,
// with the harness configuration of the user and of the project, and unless dry
// writes it. It commits nothing: a change of the lock is a change of the
// Libraries the spec is checked against, so the owner reads it first. It returns
// what it found, also when it fails.
func Refresh(f Folder, user harness.File, dry bool) (Refreshed, error) {
	var r Refreshed
	proj, err := ReadHarness(f)
	if err != nil {
		return r, err
	}
	if r.Files, r.Said, err = RefreshHarness(f, harness.Resolve(user, proj), dry); err != nil {
		return r, err
	}
	l := &refresher{Folder: f, dry: dry}
	if _, r.Named, r.LockProblems, err = WriteLock(l); err != nil {
		return r, err
	}
	r.Lock = len(l.files) > 0
	return r, nil
}
