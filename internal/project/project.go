// Package project loads a project's Markitect folder from a set of files, the
// project folder, and checks it as a whole: manifest, lock,
// Libraries, spec and registry.
// Every command that reads the spec starts from a loaded Project.
//
// The layout is described in docs/guides/spec-language.md.
package project

import (
	"bytes"
	"fmt"
	"path"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/harness"
	"github.com/Glacius-Labs/Markitect/internal/libraries"
	"github.com/Glacius-Labs/Markitect/internal/realization"
	"github.com/Glacius-Labs/Markitect/internal/spec"
)

// The fixed layout of Markitect's own folder.
const (
	Dir = ".markitect"
	// SpecDir holds the spec, the project's Definitions; it is the Root Package.
	SpecDir      = Dir + "/spec"
	ManifestFile = Dir + "/manifest.yaml"
	LockFile     = Dir + "/lock.yaml"
	IgnoreFile   = Dir + "/ignore"
	RootPage     = Dir + "/registry.md"
	RegistryDir  = Dir + "/registry"
	LibraryDir   = Dir + "/libraries"
	// ReviewFile is the record of what was verified (DEC-099), and
	// AttributesFile has Git merge it by keeping the lines of both sides.
	ReviewFile     = Dir + "/review.sum"
	AttributesFile = Dir + "/.gitattributes"
	// HarnessFile is the project's harness configuration (DEC-103): the agents'
	// model and effort, the strategies and the custom workflows. It is the
	// owner's, and no agent changes it.
	HarnessFile = Dir + "/" + harness.FileName
)

// Attributes is the content of AttributesFile.
const Attributes = "review.sum merge=union\n"

// harnessPrefixes are what the files of the harness start with, in lower case.
var harnessPrefixes = []string{".claude/agents/markitect-", ".claude/skills/markitect-", ".agents/skills/markitect-", ".claude/workflows/markitect-"}

// instructionFiles are what a coding agent reads by itself when a session starts,
// and the project's settings, which hold the hook.
var instructionFiles = []string{"AGENTS.md", "CLAUDE.md", ".claude/settings.json"}

// IsHarness says whether a file steers the agents of the work: the tool accounts for
// it, as it does for its own folder. Without regard to case: a variant of a name
// is the same file in a Windows or macOS checkout.
func IsHarness(file string) bool {
	lower := strings.ToLower(file)
	for _, prefix := range harnessPrefixes {
		if strings.HasPrefix(lower, prefix) {
			return true
		}
	}
	for _, name := range instructionFiles {
		if strings.EqualFold(file, name) {
			return true
		}
	}
	return false
}

// WorktreeDir is where Claude Code keeps the worktrees it makes of a project, as
// for a builder that works in a worktree of its own (DEC-100). Each is another
// checkout, not a file of the project.
const WorktreeDir = ".claude/worktrees/"

// isOwn says whether the tool accounts for a file itself, as it does for its own
// folder: what init installs, Git's ignore file, and Claude Code's worktrees.
func isOwn(file string) bool {
	return IsHarness(file) || file == ".gitignore" || strings.HasPrefix(strings.ToLower(file), WorktreeDir)
}

// Repository is one set of a project's files, such as the project folder.
type Repository interface {
	// Files returns every file of the project, folder-relative with forward
	// slashes: for a project folder what a walk finds without what its .gitignore
	// files exclude.
	Files() ([]string, error)
	Read(file string) ([]byte, error)
	// GitIgnored reports whether the project's .gitignore files exclude a file.
	GitIgnored(file string) bool
}

// Project is one checked state of a project.
type Project struct {
	Files     []string  // every visible repository file
	Lock      spec.Lock // empty when the lock is missing or unreadable
	Libraries []spec.LibrarySource
	Spec      *spec.Spec
	Registry  *realization.Registry
	// Errors are the compile errors, of the spec and of the realization; a
	// Project with errors is still loaded as far as its files allow.
	Errors spec.Diagnostics
}

// Codes of the errors that say the realization does not match the spec yet:
// the implementer settles them in the work. Every other error is the spec's,
// and the owner settles it in the spec.
const (
	CodeUnregistered = "file.unregistered"
	CodeUnused       = "definition.unused"
)

// IsRealization says whether an error is one of the realization's.
func IsRealization(code string) bool {
	return code == CodeUnregistered || code == CodeUnused || strings.HasPrefix(code, "stale.")
}

// SpecErrors are the errors in the spec itself: what stops a Command before it
// starts an agent.
func (p *Project) SpecErrors() spec.Diagnostics {
	var out spec.Diagnostics
	for _, d := range p.Errors {
		if !IsRealization(d.Code) {
			out = append(out, d)
		}
	}
	return out
}

// RealizationErrors are the errors that say the work does not match the spec yet.
func (p *Project) RealizationErrors() spec.Diagnostics {
	var out spec.Diagnostics
	for _, d := range p.Errors {
		if IsRealization(d.Code) {
			out = append(out, d)
		}
	}
	return out
}

// NothingBuilt says that all that stands between the spec and the work is that the
// work has not begun: the registry lists no file, and the only errors are
// Definitions nothing realizes. Only the check command reads this; status does
// not, so the work still has to realize every Definition.
func (p *Project) NothingBuilt() bool {
	if len(p.Errors) == 0 || p.Registry == nil || len(p.Registry.Entries) > 0 {
		return false
	}
	for _, d := range p.Errors {
		if d.Code != CodeUnused {
			return false
		}
	}
	return true
}

// Load reads and checks the project in a set of files, such as a project folder.
func Load(repo Repository) (*Project, error) {
	files, err := repo.Files()
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	p := &Project{Files: files}
	var ds spec.Diagnostics
	read := func(file string) ([]byte, bool) {
		data, err := repo.Read(file)
		if err != nil {
			ds.Add("project.read", file, 0, "cannot read: %v", err)
			return nil, false
		}
		return Normalize(data), true
	}
	present := map[string]bool{}
	var specFiles []spec.File
	var pages []realization.PageFile
	var libraryFiles, others []string
	for _, f := range files {
		present[f] = true
		rel, own := strings.CutPrefix(f, Dir+"/")
		if !own {
			others = append(others, f)
			continue
		}
		switch first, rest, _ := strings.Cut(rel, "/"); {
		case f == ManifestFile, f == LockFile, f == IgnoreFile, f == ReviewFile, f == AttributesFile:
		case f == HarnessFile:
			if data, ok := read(f); ok {
				if _, err := harness.Parse(data); err != nil {
					ds.Add("harness.syntax", f, 0, "%v; fix it by hand, since markitect config cannot read it either", err)
				}
			}
		case f == RootPage:
			if data, ok := read(f); ok {
				pages = append(pages, realization.PageFile{Package: spec.RootPackage, File: f, Data: data})
			}
		case first == "registry" && rest != "":
			pkg, ok := strings.CutSuffix(rest, ".md")
			if !ok {
				ds.Add("registry.page", f, 0, "the registry folder holds only Markdown pages, one per Package")
				continue
			}
			if data, ok := read(f); ok {
				pages = append(pages, realization.PageFile{Package: pkg, File: f, Data: data})
			}
		case first == "spec" && rest != "":
			if data, ok := read(f); ok {
				specFiles = append(specFiles, spec.File{Path: rest, Data: data})
			}
		case first == "libraries" && strings.Contains(rest, "/"):
			libraryFiles = append(libraryFiles, f)
		default:
			ds.Add("project.file", f, 0, "%s holds only the manifest, lock, ignore file, the harness configuration %s, the registry, the spec, Libraries and the record %s with its %s", Dir, HarnessFile, ReviewFile, AttributesFile)
		}
	}
	local := map[string][]spec.File{}
	for _, f := range libraryFiles {
		name, file, _ := strings.Cut(strings.TrimPrefix(f, LibraryDir+"/"), "/")
		if data, ok := read(f); ok {
			local[name] = append(local[name], spec.File{Path: file, Data: data})
		}
	}
	p.loadManifest(read, present, local, &ds)
	sp, errs := spec.Compile(spec.Source{Dir: SpecDir, Files: specFiles, Libraries: p.Libraries})
	p.Spec = sp
	ds = append(ds, errs...)
	in := realization.Input{Spec: sp, Pages: pages, Files: others, GitIgnored: repo.GitIgnored, OwnDir: Dir, Own: isOwn}
	if present[IgnoreFile] {
		if data, ok := read(IgnoreFile); ok {
			in.Ignore = &spec.File{Path: IgnoreFile, Data: data}
		}
	}
	reg, errs := realization.Account(in)
	p.Registry = reg
	ds = append(ds, errs...)
	addRealizationErrors(&ds, sp, reg)
	p.Errors = ds.Sorted()
	return p, nil
}

// addRealizationErrors makes stale registry entries and unused Definitions
// errors: spec and code do not match, whichever side changed.
func addRealizationErrors(ds *spec.Diagnostics, sp *spec.Spec, reg *realization.Registry) {
	if reg != nil {
		for _, s := range reg.Stale {
			switch s.Code {
			case "stale.definition":
				ds.Add(s.Code, s.PageFile, s.Line, "%s links [[%s]], which no longer exists", s.File, s.Target)
			case "stale.file":
				ds.Add(s.Code, s.PageFile, s.Line, "%s no longer exists", s.File)
			case "stale.page":
				if s.File == "" {
					ds.Add(s.Code, s.PageFile, 0, "the Package of this page no longer exists")
				} else {
					ds.Add(s.Code, s.PageFile, s.Line, "%s is listed only on the page of a removed Package", s.File)
				}
			}
		}
	}
	if sp == nil {
		return
	}
	incoming := map[string]bool{}
	for _, d := range sp.Definitions {
		for _, l := range d.Links {
			if l.Target != d.ID {
				incoming[l.Target] = true
			}
		}
	}
	realized := map[string]bool{}
	if reg != nil {
		for _, e := range reg.Entries {
			for _, id := range e.Realizes {
				realized[id] = true
			}
		}
	}
	for _, id := range sp.OwnDefinitions() {
		if !incoming[id] && !realized[id] {
			ds.Add(CodeUnused, sp.Definitions[id].File, 0, "nothing links to %s and no file realizes it", id)
		}
	}
}

func (p *Project) loadManifest(read func(string) ([]byte, bool), present map[string]bool, local map[string][]spec.File, ds *spec.Diagnostics) {
	if !present[ManifestFile] {
		ds.Add("manifest.missing", ManifestFile, 0, "the project has no manifest; run markitect init")
		return
	}
	data, ok := read(ManifestFile)
	if !ok {
		return
	}
	manifest, errs := spec.ParseManifest(ManifestFile, data)
	*ds = append(*ds, errs...)
	named := map[string]bool{}
	otherVersion := false
	for _, ref := range manifest.Libraries {
		named[ref.Name] = true
		lib, code, reason := LibrarySource(ref, local)
		if code == "library.version" {
			// The Library is the shipped one all the same, so that its Kinds
			// resolve and the one error stands alone; the lock is not compared
			// until the manifest names the version.
			ds.Add(code, ManifestFile, ref.Line, "%s", reason)
			otherVersion = true
			lib, code, _ = LibrarySource(spec.LibraryRef{Name: ref.Name, Version: libraries.Version}, local)
		}
		if code != "" {
			ds.Add(code, ManifestFile, ref.Line, "%s", reason)
			continue
		}
		p.Libraries = append(p.Libraries, lib)
	}
	if !named["std"] {
		ds.Add("library.missing", ManifestFile, 0, "every project uses the standard Library std; add std: %s to libraries", libraries.Version)
	}
	for name := range local {
		if !named[name] {
			ds.Add("library.undeclared", path.Join(LibraryDir, name), 0, "the manifest does not name Library %s; add it with its version, or remove the folder", name)
		}
	}
	if !present[LockFile] {
		ds.Add("lock.missing", LockFile, 0, "the project has no lock; run markitect tidy")
		return
	}
	if data, ok := read(LockFile); ok {
		lock, errs := spec.ParseLock(LockFile, data)
		*ds = append(*ds, errs...)
		if len(errs) == 0 {
			p.Lock = lock
			if !otherVersion {
				*ds = append(*ds, spec.VerifyLock(LockFile, lock, p.Libraries)...)
			}
		}
	}
}

// LibrarySource finds the files of a Library the manifest names: a standard
// Library from the binary, any other from the project's Library folder. It
// returns a diagnostic code and reason when the Library is not available.
func LibrarySource(ref spec.LibraryRef, local map[string][]spec.File) (spec.LibrarySource, string, string) {
	if files, ok := libraries.Files(ref.Name); ok {
		if _, shadowed := local[ref.Name]; shadowed {
			return spec.LibrarySource{}, "library.reserved", fmt.Sprintf("%s is a standard Library that ships with markitect; remove %s/%s", ref.Name, LibraryDir, ref.Name)
		}
		if ref.Version != libraries.Version {
			return spec.LibrarySource{}, "library.version", fmt.Sprintf("this markitect ships %s %s, not %s; markitect tidy names %s in the manifest and writes the lock, or name it there by hand", ref.Name, libraries.Version, ref.Version, libraries.Version)
		}
		lib := spec.LibrarySource{Name: ref.Name, Version: ref.Version, Dir: "markitect:" + ref.Name}
		for _, f := range files {
			lib.Files = append(lib.Files, spec.File{Path: f.Path, Data: f.Data})
		}
		return lib, "", ""
	}
	files, ok := local[ref.Name]
	if !ok {
		return spec.LibrarySource{}, "library.missing", fmt.Sprintf("Library %s is neither a standard Library nor committed under %s/%s", ref.Name, LibraryDir, ref.Name)
	}
	return spec.LibrarySource{Name: ref.Name, Version: ref.Version, Dir: path.Join(LibraryDir, ref.Name), Files: files}, "", ""
}

// LocalLibraries returns the files below the project's Library folder,
// grouped by Library name.
func LocalLibraries(repo Repository, files []string) (map[string][]spec.File, error) {
	local := map[string][]spec.File{}
	for _, f := range files {
		rest, ok := strings.CutPrefix(f, LibraryDir+"/")
		if !ok || !strings.Contains(rest, "/") {
			continue
		}
		data, err := repo.Read(f)
		if err != nil {
			return nil, err
		}
		name, file, _ := strings.Cut(rest, "/")
		local[name] = append(local[name], spec.File{Path: file, Data: Normalize(data)})
	}
	return local, nil
}

// Normalize turns CRLF line ends into LF, so that a checkout with Windows
// line ends compiles, locks and compares like the commit it came from.
func Normalize(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}
