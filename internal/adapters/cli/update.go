package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
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

// sameText compares two texts without minding line endings: a checkout may hold a
// file with CRLF, and the binary writes LF.
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
	if err := quiet.ensureGuard(r); err != nil {
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
// writes it. It commits nothing: a change of the lock is a change of the
// Libraries the spec is checked against, so the owner reads it first.
func (c CLI) refreshProject(repo string, dry bool) (projectState, error) {
	var st projectState
	ws, err := c.Folders.Project(repo)
	if err != nil {
		st.why = "no folder here or above holds " + project.ManifestFile
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

// PluginEnv is the environment variable the Claude Code plugin's launcher sets,
// to the plugin's version, when it starts this binary. A binary the plugin
// manages is replaced by updating the plugin, not by update.
const PluginEnv = "MARKITECT_PLUGIN"

// pluginBrings says whether the personal skill is missing as a file because the
// plugin that started this binary carries it as a skill of its own.
func pluginBrings(skill fileState) bool {
	return skill == missing && os.Getenv(PluginEnv) != ""
}

// ownVersion is the version of this binary.
func (c CLI) ownVersion() string {
	if c.Version != nil {
		return c.Version()
	}
	return buildinfo.Version()
}

// executable is the file of this binary, with symbolic links resolved.
func (c CLI) executable() (string, error) {
	find := c.Executable
	if find == nil {
		find = os.Executable
	}
	exe, err := find()
	if err != nil {
		return "", err
	}
	if real, err := filepath.EvalSymlinks(exe); err == nil {
		exe = real
	}
	return exe, nil
}

// binaryStep is what update's look at the binary found.
type binaryStep struct {
	stale      bool // a newer release exists and --check was given
	handedOver bool // the new binary has done the rest; code is its exit code
	code       int
}

// upgradeBinary looks for a newer release and installs it, unless the binary is
// managed by the plugin or is a development build. It never fails because there
// is no network: update must keep working offline for everything else. After a
// successful install the new binary finishes the update, because only it knows
// the skills, agents and lock it ships.
func (c CLI) upgradeBinary(ctx context.Context, repo string, check, force bool) (binaryStep, error) {
	var step binaryStep
	cur := c.ownVersion()
	if plugin := os.Getenv(PluginEnv); plugin != "" {
		fmt.Fprintf(c.Stdout, "binary: %s comes with the Claude Code plugin (%s); update the plugin to get a newer one\n", cur, plugin)
		return step, nil
	}
	if c.Releases == nil {
		return step, nil
	}
	if exe, err := c.executable(); err == nil && !check {
		// What an earlier update on Windows moved aside: a program that runs cannot
		// delete its own file, but the next update can.
		c.Releases.Tidy(exe)
	}
	latest, err := c.Releases.Latest(ctx)
	if err != nil {
		fmt.Fprintf(c.Stdout, "binary: could not look for a newer release: %v\n", err)
		return step, nil
	}
	newer, err := buildinfo.Newer(latest, cur)
	switch {
	case err != nil:
		fmt.Fprintf(c.Stdout, "binary: %s, release %s; they cannot be compared: %v\n", cur, latest, err)
		return step, nil
	case !newer:
		fmt.Fprintf(c.Stdout, "binary: up to date (%s)\n", cur)
		return step, nil
	case buildinfo.IsDev(cur) && !force:
		fmt.Fprintf(c.Stdout, "binary: this is a development build (%s) and release %s exists; markitect update --force replaces this build by the release\n", cur, latest)
		return step, nil
	case check:
		step.stale = true
		fmt.Fprintf(c.Stdout, "binary: %s, release %s is newer; markitect update installs it\n", cur, latest)
		return step, nil
	}
	exe, err := c.executable()
	if err != nil {
		return step, err
	}
	if err := c.Releases.Install(ctx, latest, exe); err != nil {
		return step, fmt.Errorf("could not install release %s: %w\nmarkitect update --files refreshes the files of the project and the personal skill with this binary all the same", latest, err)
	}
	fmt.Fprintf(c.Stdout, "binary: updated %s to %s (%s)\n", cur, latest, exe)
	code, err := c.Releases.Run(ctx, exe, []string{"update", "--files", "--repo", repo}, c.Stdout, c.Stderr)
	if err != nil {
		return step, fmt.Errorf("the new markitect could not start (%v); run markitect update --files to finish", err)
	}
	step.handedOver, step.code = true, code
	return step, nil
}

// update installs the newest release of markitect and then brings what the
// binary wrote earlier in line with it: the personal skill, if there is one, and
// the project's skills, agents, hook and lock. It changes no model file.
func (c CLI) update(ctx context.Context, args []string) (int, error) {
	var o options
	var check, files, force bool
	fs := c.flags("update", &o)
	fs.BoolVar(&check, "check", false, "change nothing; say what is out of date and exit 1 if anything is")
	fs.BoolVar(&files, "files", false, "leave the binary alone and refresh only the skills, agents and lock")
	fs.BoolVar(&force, "force", false, "replace a development build by the newest release")
	if err := fs.Parse(args); err != nil {
		return Misused, err
	}
	if fs.NArg() != 0 {
		return Misused, fmt.Errorf("takes no arguments")
	}
	stale := false
	if !files {
		step, err := c.upgradeBinary(ctx, o.repo, check, force)
		if err != nil {
			return Misused, err
		}
		if step.handedOver {
			return step.code, nil
		}
		stale = step.stale
	}
	file, skill, err := personalSkillState()
	if err != nil {
		return Misused, err
	}
	proj, err := c.refreshProject(o.repo, check)
	if err != nil {
		return Misused, err
	}

	switch {
	case pluginBrings(skill):
		fmt.Fprintln(c.Stdout, "personal skill: comes with the plugin")
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
				fmt.Fprintln(c.Stdout, "Markitect uses no Git: commit them when you like.")
			}
		}
		switch {
		case proj.lockCode != OK:
			fmt.Fprintln(c.Stdout, "lock: not rewritten; the manifest has errors, fix them first")
		case proj.lock && check:
			stale = true
			fmt.Fprintln(c.Stdout, "lock: out of date; markitect update rewrites it")
		case proj.lock:
			fmt.Fprintf(c.Stdout, "lock: rewritten; read the change of %s before you commit it\n", project.LockFile)
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
	if plugin := os.Getenv(PluginEnv); plugin != "" {
		fmt.Fprintf(c.Stdout, "binary: comes with the Claude Code plugin (%s)\n", plugin)
	}
	if _, skill, err := personalSkillState(); err != nil {
		fmt.Fprintf(c.Stdout, "personal skill: unreadable: %v\n", err)
	} else {
		switch skill {
		case current:
			fmt.Fprintln(c.Stdout, "personal skill: up to date")
		case outdated:
			fmt.Fprintln(c.Stdout, "personal skill: out of date; run markitect update")
		case missing:
			if pluginBrings(skill) {
				fmt.Fprintln(c.Stdout, "personal skill: comes with the plugin")
				break
			}
			fmt.Fprintln(c.Stdout, "personal skill: not installed; run markitect install")
		}
	}
	proj, err := c.refreshProject(o.repo, true)
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
// from another markitect. It is best effort and changes no result.
func (c CLI) staleNotice(args []string) {
	ws, err := c.Folders.Project(repoArg(args))
	if err != nil {
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
