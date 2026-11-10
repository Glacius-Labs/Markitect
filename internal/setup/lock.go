package setup

import (
	"fmt"

	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// WriteLock writes the lock from the manifest and the Libraries it names, and
// returns them. When the manifest has errors, it writes nothing and returns
// them, sorted.
func WriteLock(f Folder) ([]spec.LibrarySource, spec.Diagnostics, error) {
	data, err := f.Read(project.ManifestFile)
	if err != nil {
		return nil, nil, fmt.Errorf("read the manifest: %w; run markitect init", err)
	}
	manifest, ds := spec.ParseManifest(project.ManifestFile, data)
	// The same files the walk finds, as check reads them.
	files, err := f.Files()
	if err != nil {
		return nil, nil, err
	}
	local, err := project.LocalLibraries(f, files)
	if err != nil {
		return nil, nil, err
	}
	var libs []spec.LibrarySource
	for _, ref := range manifest.Libraries {
		lib, code, reason := project.LibrarySource(ref, local)
		if code != "" {
			ds.Add(code, project.ManifestFile, ref.Line, "%s", reason)
			continue
		}
		libs = append(libs, lib)
	}
	if len(ds) > 0 {
		return nil, ds.Sorted(), nil
	}
	if err := f.Write(project.LockFile, spec.RenderLock(libs)); err != nil {
		return nil, nil, err
	}
	return libs, nil, nil
}
