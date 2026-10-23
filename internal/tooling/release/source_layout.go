package release

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var commandConfigurationExamples = []string{
	"examples/runtime-profile.example.yaml",
	"examples/run.example.json",
}

func commandConfigurationExample(name string) bool {
	for _, path := range commandConfigurationExamples {
		if name == path {
			return true
		}
	}
	return false
}

// Source archives contain one root module with these explicit responsibility
// roots. Parent containers grant no privilege to unknown sibling packages.
var sourceRoots = []string{
	"cmd/markitect",
	"cmd/markitect-release",
	"cmd/markitect-check-architecture",
	"cmd/markitect-check-modules",
	"cmd/markitect-agentd",
	"internal/model",
	"internal/snapshot",
	"internal/organization",
	"internal/realization",
	"internal/run",
	"internal/apply",
	"internal/modeldocs",
	"internal/buildinfo",
	"internal/agentd",
	"internal/adapters/cli",
	"internal/adapters/documents",
	"internal/adapters/gitrepo",
	"internal/adapters/codexappserver",
	"internal/adapters/claudecode",
	"internal/adapters/checks",
	"internal/adapters/container",
	"internal/bootstrap",
	"internal/testkit",
	"internal/tooling/architecture",
	"internal/tooling/architecturecli",
	"internal/tooling/modulechecks",
	"internal/tooling/modulecli",
	"internal/tooling/modules/githooks",
	"internal/tooling/modules/pipelines",
	"internal/tooling/licenses",
	"internal/tooling/release",
	"internal/tooling/releasecli",
	"internal/tooling/publish",
	"internal/tooling/source",
	"internal/tooling/guardedwrite",
}

func sourceRoot(name string) bool {
	for _, root := range sourceRoots {
		if strings.HasPrefix(name, root+"/") {
			return true
		}
	}
	return false
}
func sourceContainer(name string) bool {
	for _, root := range sourceRoots {
		if root == name || strings.HasPrefix(root, name+"/") {
			return true
		}
	}
	return false
}
func sourceTestData(name string) bool {
	return sourceRoot(name) && strings.Contains(name, "/testdata/")
}

// Validate all raw paths and modes before filtering extensions. In particular,
// an unsupported package containing no Go file cannot disappear silently.
func validateSnapshotInputs(files map[string][]byte, modes map[string]string, prefix string) error {
	for name := range files {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rel := strings.TrimPrefix(name, prefix)
		if rel == "src" || strings.HasPrefix(rel, "src/") {
			return fmt.Errorf("legacy src layout is unsupported: %s", rel)
		}
		// Files has no directory entries. A literal schema entry therefore
		// cannot represent the real optional directory accepted by live capture.
		if rel == "schema" {
			return fmt.Errorf("optional schema path must be a real directory")
		}
		if strings.HasPrefix(rel, "schema/") && !hasExcludedSourceDirectory(rel) {
			if modes[name] != "100644" && modes[name] != "100755" {
				return fmt.Errorf("source snapshot schema path %q is not regular", name)
			}
		}
		if !sourceTreePath(rel) {
			continue
		}
		if !sourceRoot(rel) {
			return fmt.Errorf("unsupported source responsibility path: %s", rel)
		}
		if hasExcludedSourceDirectory(rel) {
			continue
		}
		if isNestedGoModuleManifest(rel) {
			return fmt.Errorf("nested Go module is unsupported: %s", rel)
		}
		if modes[name] != "100644" && modes[name] != "100755" {
			return fmt.Errorf("source snapshot path %q is not regular", name)
		}
	}
	return nil
}

func validateSourceDirectories(moduleDir string) error {
	for _, tree := range []string{"cmd", "internal"} {
		if err := filepath.WalkDir(filepath.Join(moduleDir, tree), func(full string, e fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			rel, err := filepath.Rel(moduleDir, full)
			if err != nil {
				return err
			}
			rel = filepath.ToSlash(rel)
			if e.IsDir() {
				if full != filepath.Join(moduleDir, tree) && excludedDirs[strings.ToLower(e.Name())] {
					return filepath.SkipDir
				}
				if !sourceContainer(rel) && !sourceRoot(rel+"/") {
					return fmt.Errorf("unsupported source responsibility path: %s", rel)
				}
				return requireRealDirectory(full)
			}
			if !sourceRoot(rel) {
				return fmt.Errorf("unsupported source responsibility path: %s", rel)
			}
			if e.Type()&os.ModeSymlink != 0 {
				return fmt.Errorf("refusing symlink in source tree: %s", rel)
			}
			if isNestedGoModuleManifest(rel) {
				return fmt.Errorf("nested Go module is unsupported: %s", rel)
			}
			return nil
		}); err != nil {
			return err
		}
	}
	return nil
}

func hasExcludedSourceDirectory(name string) bool {
	parts := strings.Split(name, "/")
	for _, p := range parts[:len(parts)-1] {
		if excludedDirs[strings.ToLower(p)] {
			return true
		}
	}
	return false
}
func isNestedGoModuleManifest(name string) bool { return strings.HasSuffix(name, "/go.mod") }

func validateCollectedSourceLayout(files []sourceFile) error {
	present := make(map[string]bool, len(files))
	for _, f := range files {
		present[f.name] = true
	}
	entry, internal, err := moduleLayoutFiles(present)
	if err != nil {
		return err
	}
	if !present[entry] {
		return fmt.Errorf("source tree is missing required command entrypoint %s", entry)
	}
	for name := range present {
		if strings.HasPrefix(name, internal) && strings.EqualFold(filepath.Ext(name), ".go") {
			return nil
		}
	}
	return fmt.Errorf("source tree is missing Go source under %s", internal)
}
