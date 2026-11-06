package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"slices"
	"sort"

	"github.com/Glacius-Labs/Markitect/internal/buildinfo"
	"github.com/Glacius-Labs/Markitect/internal/project"
)

// fileState says how something this binary writes compares with what is in place.
type fileState int

const (
	current fileState = iota
	outdated
	missing
)

const updateMessage = "Update Markitect"

// sameText compares two texts without minding line endings: Git may check a file
// out with CRLF, and the binary writes LF.
func sameText(a, b []byte) bool {
	crlf, lf := []byte("\r\n"), []byte("\n")
	return bytes.Equal(bytes.ReplaceAll(a, crlf, lf), bytes.ReplaceAll(b, crlf, lf))
}

// refresher passes on the writes of a refresh that change a file, and notes
// them. A file that already holds the text is left alone and not noted, so the
// notes are the files that differ. With dry set it changes nothing and only
// notes, which is how version and update --check find out what is out of date
// with the same code that update uses to put it right.
type refresher struct {
	Workspace
	dry   bool
	files []string
}

func (r *refresher) Write(file string, data []byte) error {
	if old, err := r.Workspace.Read(file); err == nil && sameText(old, data) {
		return nil
	}
	r.files = append(r.files, file)
	if r.dry {
		return nil
	}
	return r.Workspace.Write(file, data)
}

func (r *refresher) Remove(file string) error {
	r.files = append(r.files, file)
	if r.dry {
		return nil
	}
	return r.Workspace.Remove(file)
}

// refreshHarness brings the skills, the agents and the hook of a project in line
// with this binary, as init does, and returns the files that differed. It never
// touches the spec, AGENTS.md or CLAUDE.md.
func (c CLI) refreshHarness(ws Workspace, dry bool) ([]string, error) {
	r := &refresher{Workspace: ws, dry: dry}
	quiet := c
	if dry {
		quiet.Stdout = io.Discard
	}
	if _, err := installSkills(r); err != nil {
		return nil, err
	}
	if err := installHarness(r); err != nil {
		return nil, err
	}
	if err := quiet.ensureAnswerHook(r); err != nil {
		return nil, err
	}
	// A retired file is removed by both of the writers above in a dry run, which
	// removes nothing in between.
	sort.Strings(r.files)
	return slices.Compact(r.files), nil
}

// projectState is how a project stands against this binary.
type projectState struct {
	ws       Workspace // nil when the folder is no Markitect project
	why      string    // then, why not
	files    []string  // the skills, agents and settings that differ
	lock     bool      // the lock differs
	lockCode int       // Failed when the manifest names a Library the binary cannot supply
}

func (p projectState) stale() bool { return len(p.files) > 0 || p.lock }

// refreshProject finds what differs in the project at repo and, unless dry,
// writes it. It leaves the lock uncommitted: a change of the lock is a change of
// the Libraries the spec is checked against, so the owner reads it first.
func (c CLI) refreshProject(ctx context.Context, repo string, dry bool) (projectState, error) {
	var st projectState
	ws, err := c.Git.Open(ctx, repo)
	if err != nil {
		st.why = "this folder is in no Git repository"
		return st, nil
	}
	if !ws.Exists(project.ManifestFile) {
		st.why = "it has no " + project.ManifestFile
		return st, nil
	}
	st.ws = ws
	if st.files, err = c.refreshHarness(ws, dry); err != nil {
		return st, err
	}
	l := &refresher{Workspace: ws, dry: dry}
	quiet := c
	quiet.Stdout = io.Discard
	if st.lockCode, err = quiet.writeLock(l); err != nil {
		return st, err
	}
	st.lock = len(l.files) > 0
	return st, nil
}

// update brings what this binary wrote earlier in line with it, after the
// binary was upgraded: the personal skill, if there is one, and the project's
// skills, agents, hook and lock. It changes no model file and does not upgrade
// the binary.
func (c CLI) update(ctx context.Context, args []string) (int, error) {
	var o options
	var check bool
	fs := c.flags("update", &o)
	fs.BoolVar(&check, "check", false, "change nothing; say what is out of date and exit 1 if anything is")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	file, skill, err := personalSkillState()
	if err != nil {
		return Misused, err
	}
	proj, err := c.refreshProject(ctx, o.repo, check)
	if err != nil {
		return Misused, err
	}
	stale := false

	switch {
	case skill == missing:
		fmt.Fprintf(c.Stdout, "personal skill: not installed; markitect install writes %s\n", file)
	case skill == current:
		fmt.Fprintln(c.Stdout, "personal skill: up to date")
	case check:
		stale = true
		fmt.Fprintf(c.Stdout, "personal skill: out of date (%s)\n", file)
	default:
		if _, err := writePersonalSkill(); err != nil {
			return Misused, err
		}
		fmt.Fprintf(c.Stdout, "personal skill: refreshed %s\n", file)
	}

	switch {
	case proj.ws == nil:
		fmt.Fprintf(c.Stdout, "project: none here (%s); markitect init sets one up\n", proj.why)
	case !proj.stale() && proj.lockCode == OK:
		fmt.Fprintln(c.Stdout, "project: up to date")
	default:
		if len(proj.files) > 0 {
			if check {
				stale = true
				fmt.Fprintf(c.Stdout, "project: %d files out of date:\n", len(proj.files))
			} else {
				fmt.Fprintf(c.Stdout, "project: updated %d files:\n", len(proj.files))
			}
			for _, f := range proj.files {
				fmt.Fprintf(c.Stdout, "  %s\n", f)
			}
			if !check {
				switch committed, err := proj.ws.CommitOwn(proj.files, updateMessage); {
				case err != nil:
					fmt.Fprintf(c.Stdout, "could not commit them, so commit them yourself before the next apply; if Git ignores .claude/, the files stay on disk and that is enough: %v\n", err)
				case committed:
					fmt.Fprintf(c.Stdout, "committed them as %q\n", updateMessage)
				}
			}
		}
		switch {
		case proj.lockCode != OK:
			fmt.Fprintln(c.Stdout, "lock: not rewritten; the manifest has errors, fix them first")
		case proj.lock && check:
			stale = true
			fmt.Fprintln(c.Stdout, "lock: out of date; markitect update rewrites it")
		case proj.lock:
			fmt.Fprintf(c.Stdout, "lock: rewritten, not committed; review it with git diff %s and commit it\n", project.LockFile)
		}
	}
	c.warnPath()
	if proj.lockCode != OK || (check && stale) {
		return Failed, nil
	}
	return OK, nil
}

// version prints which markitect this is and whether what it wrote earlier still
// matches it.
func (c CLI) version(ctx context.Context, args []string) (int, error) {
	var o options
	fs := c.flags("version", &o)
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	fmt.Fprintln(c.Stdout, "markitect", buildinfo.Build())
	if _, skill, err := personalSkillState(); err != nil {
		fmt.Fprintf(c.Stdout, "personal skill: unreadable: %v\n", err)
	} else {
		switch skill {
		case current:
			fmt.Fprintln(c.Stdout, "personal skill: up to date")
		case outdated:
			fmt.Fprintln(c.Stdout, "personal skill: out of date; run markitect update")
		case missing:
			fmt.Fprintln(c.Stdout, "personal skill: not installed; run markitect install")
		}
	}
	proj, err := c.refreshProject(ctx, o.repo, true)
	switch {
	case err != nil:
		fmt.Fprintf(c.Stdout, "project: unreadable: %v\n", err)
	case proj.ws == nil:
		fmt.Fprintf(c.Stdout, "project: none here (%s)\n", proj.why)
	case proj.stale():
		fmt.Fprintf(c.Stdout, "project: out of date (%d files%s); run markitect update\n", len(proj.files), lockNote(proj.lock))
	default:
		fmt.Fprintln(c.Stdout, "project: up to date")
	}
	c.warnPath()
	return OK, nil
}

func lockNote(lock bool) string {
	if lock {
		return " and the lock"
	}
	return ""
}

// staleNotice prints the one line that says a project's skills and agents come
// from another markitect. Like notice it is best effort and changes no result.
func (c CLI) staleNotice(ctx context.Context, args []string) {
	ws, err := c.Git.Open(ctx, repoArg(args))
	if err != nil || !ws.Exists(project.ManifestFile) {
		return
	}
	files, err := c.refreshHarness(ws, true)
	if err != nil || len(files) == 0 {
		return
	}
	fmt.Fprintf(c.Stderr, "markitect: the skills and agents of this project come from another markitect (%d files differ); run markitect update\n", len(files))
}

// warnPath tells the user when markitect is not what the PATH finds under that
// name, since the apply and verify skills and the agents call it by name.
func (c CLI) warnPath() {
	if c.PathWarning == nil {
		return
	}
	if w := c.PathWarning(); w != "" {
		fmt.Fprintln(c.Stderr, w)
	}
}
