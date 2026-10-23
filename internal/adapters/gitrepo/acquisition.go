package gitrepo

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/snapshot"
)

type sourceBoundary struct {
	abs  string
	area snapshot.DirectoryBoundary
	disk *diskBoundary
}
type modelCapture struct {
	target   *project.ModelSource
	boundary *sourceBoundary
	files    []snapshot.Entry
	seen     map[string]snapshot.Entry
}
type acquisition struct {
	ctx                    context.Context
	repository             *Repository
	git                    *gitRepository
	head, revision, commit string
	committed              bool
	raw                    project.RawInputs
	boundaries             map[string]*sourceBoundary
	models                 []*modelCapture
	accounted              map[string]snapshot.Entry
	entries                map[string]bool
	total                  int64
	indexModes             map[string]snapshot.Mode
	census                 []snapshot.Entry
	censusMetadata         []diskEntry
}

func (r *Repository) acquire(ctx context.Context, selection project.InputSelection, revision string, committed bool) (a *acquisition, err error) {
	if r == nil {
		return nil, fmt.Errorf("%w: nil repository adapter", ErrUnsafe)
	}
	ctx, cancel := context.WithTimeout(ctx, r.limits.Timeout)
	defer cancel()
	a = &acquisition{ctx: ctx, repository: r, revision: revision, committed: committed, boundaries: make(map[string]*sourceBoundary), accounted: make(map[string]snapshot.Entry), entries: make(map[string]bool)}
	a.raw.Local.Selection = selection.Local
	a.raw.Dependencies = make([]project.ModelSource, len(selection.Dependencies))
	for i, dependency := range selection.Dependencies {
		a.raw.Dependencies[i].Selection = dependency
	}
	defer func() {
		for _, m := range a.models {
			var treeErr error
			m.target.Files, treeErr = snapshot.NewTree(m.files)
			err = errors.Join(err, treeErr)
		}
		var treeErr error
		a.raw.Census, treeErr = snapshot.NewTree(a.census)
		err = errors.Join(err, treeErr)
		if a.git != nil {
			if closeErr := a.git.finishBlobs(err == nil); err == nil {
				err = closeErr
			}
		}
		for _, b := range a.boundaries {
			if b.disk != nil {
				b.disk.close()
			}
		}
	}()
	if err = ctx.Err(); err != nil {
		return a, err
	}
	if selection.Repository == "" || selection.Local.Boundary == "" || selection.Local.ManifestPath.String() == "" {
		return a, fmt.Errorf("%w: repository and explicit local manifest required", ErrUnsafe)
	}
	a.git, err = identifyGit(ctx, selection.Repository)
	if err != nil {
		return a, err
	}
	a.head, err = a.git.commit("HEAD", false)
	if err != nil {
		return a, err
	}
	if revision != "" {
		a.commit, err = a.git.commit(revision, true)
	} else if committed {
		a.commit = a.head
	}
	if err != nil {
		return a, err
	}
	if a.commit != "" {
		if err = a.git.inventory(a.commit, r.limits.Entries); err != nil {
			return a, err
		}
	}
	if committed {
		if err = a.git.clean(); err != nil {
			return a, err
		}
	}
	if revision == "" {
		if err = a.loadIndexModes(); err != nil {
			return a, err
		}
	}
	for _, target := range append([]*project.ModelSource{&a.raw.Local}, modelPointers(a.raw.Dependencies)...) {
		if revision != "" {
			retained, found := r.retained.model(target.Selection)
			if found {
				*target = retained
				continue
			}
		}
		m := &modelCapture{target: target, seen: make(map[string]snapshot.Entry)}
		a.models = append(a.models, m)
		if err = a.manifest(m); err != nil {
			return a, err
		}
	}
	if err = a.preflightRoots(); err != nil {
		return a, err
	}
	if r.observe != nil {
		r.observe("roots-validated", a.git.path)
	}
	if err = a.preflightRoots(); err != nil {
		return a, err
	}
	for _, m := range a.models {
		if err = a.modelTree(m); err != nil {
			return a, err
		}
	}
	for _, group := range []struct {
		locations []project.SourceLocation
		target    *[]project.SourceFile
		local     bool
	}{
		{selection.Execution, &a.raw.Execution, true}, {selection.Registry, &a.raw.Registry, true}, {selection.Ignore, &a.raw.Ignore, true}, {selection.RuntimeProfiles, &a.raw.RuntimeProfiles, false}, {selection.Instructions, &a.raw.Instructions, false},
	} {
		if err = a.selectedFiles(group.locations, group.target, group.local); err != nil {
			return a, err
		}
	}
	if err = a.captureCensus(); err != nil {
		return a, err
	}
	if committed {
		if err = a.protectCommitted(); err != nil {
			return a, err
		}
		if err = a.protectTracked(); err != nil {
			return a, err
		}
	}
	if r.observe != nil {
		r.observe("before-confirm", a.git.path)
	}
	for _, b := range a.boundaries {
		if b.disk != nil {
			if err = b.disk.confirm(); err != nil {
				return a, err
			}
		}
	}
	if err = a.confirmCensus(); err != nil {
		return a, err
	}
	if revision == "" {
		before := a.indexModes
		if err = a.loadIndexModes(); err != nil {
			return a, err
		}
		if !maps.Equal(before, a.indexModes) {
			return a, ErrChanged
		}
	}
	if err = a.git.confirm(); err != nil {
		return a, err
	}
	currentHead, headErr := a.git.commit("HEAD", false)
	if headErr != nil {
		return a, headErr
	}
	if currentHead != a.head {
		return a, ErrChanged
	}
	if committed {
		if err = a.git.clean(); err != nil {
			return a, err
		}
	}
	return a, ctx.Err()
}

func modelPointers(sources []project.ModelSource) []*project.ModelSource {
	result := make([]*project.ModelSource, len(sources))
	for i := range sources {
		result[i] = &sources[i]
	}
	return result
}

func (a *acquisition) boundary(location string) (*sourceBoundary, error) {
	if location == "" {
		return nil, fmt.Errorf("%w: empty source boundary", ErrUnsafe)
	}
	abs := location
	if !filepath.IsAbs(abs) {
		abs = filepath.Join(a.git.path, abs)
	}
	abs, err := filepath.Abs(abs)
	if err != nil {
		return nil, err
	}
	// identifyGit confirmed the explicitly selected root's physical identity.
	// Preserve coordinates under that root even when Git reports another native
	// spelling (for example Windows 8.3). Do not resolve other aliases or alter
	// descendant spelling; the ordinary exact-path checks still apply below.
	if rel, err := filepath.Rel(a.git.selectedPath, abs); err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		abs = filepath.Join(a.git.path, rel)
	}
	if b, ok := a.boundaries[abs]; ok {
		return b, nil
	}
	b := &sourceBoundary{abs: abs}
	a.boundaries[abs] = b
	rel, err := filepath.Rel(a.git.path, abs)
	if err != nil {
		return nil, err
	}
	if rel == "." {
		b.area = snapshot.RootDirectory()
	} else if rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		p, err := snapshot.NewPath(filepath.ToSlash(rel))
		if err != nil {
			return nil, err
		}
		b.area, err = snapshot.DirectoryAt(p)
		if err != nil {
			return nil, err
		}
	}
	if a.revision != "" {
		if b.area.String() == "" {
			return b, fmt.Errorf("%w: fixed preview has no revision binding for external boundary %q", ErrUnsafe, location)
		}
		if err = a.virtualDirectory(b.area.String(), true); err != nil {
			return b, err
		}
		return b, nil
	}
	b.disk, err = openDisk(a.ctx, abs, a.repository.limits, a.repository.observe)
	if err != nil {
		return b, err
	}
	b.disk.repoArea = b.area
	if b.area.String() != "" {
		repoDisk, err := a.repositoryDisk()
		if err != nil {
			return b, err
		}
		exact, err := repoDisk.directory(b.area.String(), true)
		if err != nil {
			return b, err
		}
		info, err := exact.Stat(".")
		if err != nil || !os.SameFile(info, b.disk.dirs["."].info) {
			return b, ErrChanged
		}
	}
	return b, nil
}

func (a *acquisition) repositoryDisk() (*diskBoundary, error) {
	if b, ok := a.boundaries[a.git.path]; ok && b.disk != nil {
		return b.disk, nil
	}
	d, err := openDisk(a.ctx, a.git.path, a.repository.limits, a.repository.observe)
	if err != nil {
		return nil, err
	}
	if !os.SameFile(a.git.rootInfo, d.dirs["."].info) {
		d.close()
		return nil, ErrChanged
	}
	d.repoArea = snapshot.RootDirectory()
	a.boundaries[a.git.path] = &sourceBoundary{abs: a.git.path, area: snapshot.RootDirectory(), disk: d}
	return d, nil
}

func within(area snapshot.DirectoryBoundary, p string) string {
	if area.IsRoot() {
		return p
	}
	return area.String() + "/" + p
}

func (a *acquisition) read(b *sourceBoundary, p string, collision bool) (snapshot.Entry, bool, error) {
	if err := a.ctx.Err(); err != nil {
		return snapshot.Entry{}, false, err
	}
	path, err := snapshot.NewPath(p)
	if err != nil {
		return snapshot.Entry{}, false, err
	}
	if technical(within(b.area, p)) {
		return snapshot.Entry{}, false, fmt.Errorf("%w: selected technical state", ErrUnsafe)
	}
	var data []byte
	var mode snapshot.Mode
	var found bool
	if a.commit != "" && b.area.String() != "" {
		repoPath := within(b.area, p)
		if collision {
			if err := a.virtualFile(repoPath); err != nil {
				return snapshot.Entry{}, false, err
			}
		}
		meta, ok := a.git.tree[repoPath]
		if !ok {
			return snapshot.Entry{}, false, nil
		}
		if a.repository.observe != nil {
			a.repository.observe("before-git-blob", repoPath)
		}
		data, err = a.git.blob(meta, a.repository.limits.FileBytes)
		if err != nil {
			return snapshot.Entry{}, false, err
		}
		mode = snapshot.Regular
		if meta.mode == "100755" {
			mode = snapshot.Executable
		}
		found = true
	} else {
		if b.disk == nil {
			return snapshot.Entry{}, false, fmt.Errorf("%w: no revision source for external boundary", ErrUnsafe)
		}
		data, mode, found, err = b.disk.read(p, collision)
		if err != nil || !found {
			return snapshot.Entry{}, found, err
		}
		if b.area.String() != "" {
			if indexed, ok := a.indexModes[within(b.area, p)]; ok {
				mode = indexed
			}
		}
	}
	entry, err := snapshot.NewEntry(path, mode, data)
	if err != nil {
		return snapshot.Entry{}, false, err
	}
	if err = a.account(a.observationKey(b, p), entry); err != nil {
		return snapshot.Entry{}, false, err
	}
	return entry, found, nil
}

func (a *acquisition) observationKey(b *sourceBoundary, p string) string {
	if a.commit != "" && b.area.String() != "" {
		return "git:" + a.commit + ":" + within(b.area, p)
	}
	return "disk:" + filepath.Join(b.abs, filepath.FromSlash(p))
}

func (a *acquisition) admit(key string) error {
	if a.entries[key] {
		return nil
	}
	if len(a.entries) >= a.repository.limits.Entries {
		return ErrLimit
	}
	a.entries[key] = true
	return nil
}

func (a *acquisition) account(key string, entry snapshot.Entry) error {
	if previous, ok := a.accounted[key]; ok {
		if previous.Mode() != entry.Mode() || previous.ContentID() != entry.ContentID() {
			return ErrChanged
		}
		return nil
	}
	if int64(len(entry.Bytes())) > a.repository.limits.TotalBytes-a.total {
		return ErrLimit
	}
	if err := a.admit(key); err != nil {
		return err
	}
	a.total += int64(len(entry.Bytes()))
	a.accounted[key] = entry
	return nil
}

func (a *acquisition) selectedFiles(locations []project.SourceLocation, target *[]project.SourceFile, local bool) error {
	seen := make(map[string]bool)
	for _, location := range locations {
		if a.revision != "" && !local {
			if file, found := a.repository.retained.instruction(location); found {
				key := location.Boundary + "/" + location.Path.String()
				if seen[key] {
					return fmt.Errorf("%w: duplicate selected input", ErrUnsafe)
				}
				seen[key] = true
				*target = append(*target, file)
				continue
			}
		}
		*target = append(*target, project.SourceFile{Location: location})
		observed := &(*target)[len(*target)-1]
		b, err := a.boundary(location.Boundary)
		if err != nil {
			return err
		}
		if local && b.area.String() == "" {
			return fmt.Errorf("%w: local input outside repository", ErrUnsafe)
		}
		key := b.abs + "/" + location.Path.String()
		if seen[key] {
			return fmt.Errorf("%w: duplicate selected input", ErrUnsafe)
		}
		seen[key] = true
		if b.area.String() != "" {
			observed.RepositoryPath, err = snapshot.NewPath(within(b.area, location.Path.String()))
			if err != nil {
				return err
			}
		}
		file, found, err := a.read(b, location.Path.String(), true)
		if err != nil {
			return err
		}
		if !found {
			return fmt.Errorf("selected input absent: %s", location.Path)
		}
		observed.File = file
	}
	return nil
}

func (a *acquisition) loadIndexModes() error {
	a.indexModes = nil
	out, err := a.git.run("config", "--bool", "--get", "core.filemode")
	if err != nil {
		// Exit 1 with no value means Git's default true. Any other error is
		// retained; avoid assuming Windows policy after an acquisition failure.
		var exit *exec.ExitError
		if errors.As(err, &exit) && exit.ExitCode() == 1 {
			return nil
		}
		return err
	}
	if strings.TrimSpace(string(out)) == "true" {
		return nil
	}
	if strings.TrimSpace(string(out)) != "false" {
		return fmt.Errorf("%w: invalid Git filemode policy", ErrUnsafe)
	}
	out, err = a.git.run("ls-files", "--stage", "-z")
	if err != nil {
		return err
	}
	a.indexModes = make(map[string]snapshot.Mode)
	for _, record := range strings.Split(string(out), "\x00") {
		if record == "" {
			continue
		}
		header, p, ok := strings.Cut(record, "\t")
		fields := strings.Fields(header)
		if !ok || len(fields) != 3 || fields[2] != "0" {
			return fmt.Errorf("%w: unmerged or malformed index", ErrUnsafe)
		}
		if technical(p) {
			continue
		}
		mode := snapshot.Regular
		if fields[0] == "100755" {
			mode = snapshot.Executable
		} else if fields[0] != "100644" {
			return fmt.Errorf("%w: nonregular index entry", ErrUnsafe)
		}
		a.indexModes[p] = mode
		if len(a.indexModes) > a.repository.limits.Entries {
			return ErrLimit
		}
	}
	return nil
}
