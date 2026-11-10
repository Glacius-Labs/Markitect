package setup

import (
	"fmt"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/project"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// WriteLock writes the lock from the manifest and the Libraries it names, and
// returns them. A standard Library the manifest names at another version than
// this binary ships, it first names at the shipped one, in the manifest: the
// binary supplies no other, so there is nothing for the owner to choose. It
// returns what it named so. When the manifest has errors, it writes nothing and
// returns them, sorted.
func WriteLock(f Folder) (libs []spec.LibrarySource, named []string, problems spec.Diagnostics, err error) {
	data, err := f.Read(project.ManifestFile)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("read the manifest: %w; run markitect init", err)
	}
	manifest, ds := spec.ParseManifest(project.ManifestFile, data)
	shipped, named := shipStandard(data, manifest.Libraries)
	if named != nil {
		manifest, ds = spec.ParseManifest(project.ManifestFile, shipped)
	}
	// The same files the walk finds, as check reads them.
	files, err := f.Files()
	if err != nil {
		return nil, nil, nil, err
	}
	local, err := project.LocalLibraries(f, files)
	if err != nil {
		return nil, nil, nil, err
	}
	for _, ref := range manifest.Libraries {
		lib, code, reason := project.LibrarySource(ref, local)
		if code != "" {
			ds.Add(code, project.ManifestFile, ref.Line, "%s", reason)
			continue
		}
		libs = append(libs, lib)
	}
	if len(ds) > 0 {
		return nil, nil, ds.Sorted(), nil
	}
	if named != nil {
		if err := f.Write(project.ManifestFile, shipped); err != nil {
			return nil, nil, nil, err
		}
	}
	if err := f.Write(project.LockFile, spec.RenderLock(libs)); err != nil {
		return nil, nil, nil, err
	}
	return libs, named, nil, nil
}

// shipStandard returns the manifest with each standard Library at the version
// this binary ships, and what it changed, each as "std 1.1.0, where it named
// 1.0.0". It changes the version on the Library's line and keeps every other
// byte. A manifest that needs no change it returns as it is, with nothing.
func shipStandard(manifest []byte, refs []spec.LibraryRef) ([]byte, []string) {
	lines := strings.Split(string(manifest), "\n")
	var named []string
	for _, ref := range refs {
		if _, standard := libraries.Files(ref.Name); !standard || ref.Version == libraries.Version || ref.Line < 1 || ref.Line > len(lines) {
			continue
		}
		line := lines[ref.Line-1]
		colon := strings.Index(line, ":")
		at := strings.Index(line[colon+1:], ref.Version)
		if colon < 0 || at < 0 {
			continue
		}
		at += colon + 1
		lines[ref.Line-1] = line[:at] + libraries.Version + line[at+len(ref.Version):]
		named = append(named, fmt.Sprintf("%s %s, where it named %s", ref.Name, libraries.Version, ref.Version))
	}
	if named == nil {
		return manifest, nil
	}
	return []byte(strings.Join(lines, "\n")), named
}
