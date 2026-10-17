package release

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// transitionalFoundationRoots are the only root-level packages packaged while
// the product entrypoint and existing product engine remain under src/.
func transitionalFoundationRoots() []string {
	return []string{
		"internal/model",
		"internal/snapshot",
		"internal/organization",
		"internal/realization",
		"internal/apply",
		"internal/adapters/checks",
		"internal/adapters/gitrepo",
		"internal/testkit",
	}
}

type sourceLayoutPresence struct {
	srcCommand   bool
	srcInternal  bool
	rootCommand  bool
	rootInternal bool
}

// classifySourceLayout chooses the active product tree. Optional root packages are
// permitted alongside the src/ tree, but never make a product layout complete.
func classifySourceLayout(presence sourceLayoutPresence) (string, error) {
	if presence.srcCommand || presence.srcInternal {
		if !presence.srcCommand || !presence.srcInternal || presence.rootCommand {
			return "", fmt.Errorf("Markitect source must use exactly one complete cmd/internal layout")
		}
		return "src/", nil
	}
	if presence.rootCommand && presence.rootInternal {
		// Historical flat-layout source packages remain readable.
		return "", nil
	}
	return "", fmt.Errorf("Markitect source is missing a complete cmd/internal layout")
}

// isTransitionalFoundationPath accepts source only below one of the exact
// transitional root package prefixes. A sibling such as internal/modeling is
// not admitted by prefix coincidence.
func isTransitionalFoundationPath(name string) bool {
	for _, root := range transitionalFoundationRoots() {
		if strings.HasPrefix(name, root+"/") {
			return true
		}
	}
	return false
}

// isTransitionalTestkitDataPath includes non-Go files under Go's conventional
// testdata directory for the one transitional shared test-support package.
func isTransitionalTestkitDataPath(name string) bool {
	return strings.HasPrefix(name, "internal/testkit/testdata/")
}

// validateTransitionalFoundationFiles rejects any root internal source that
// is outside the optional transitional package roots. It is shared by live and fixed
// source packaging so neither path can silently omit an unknown package.
func validateTransitionalFoundationFiles(names []string) error {
	ordered := append([]string(nil), names...)
	sort.Strings(ordered)
	for _, name := range ordered {
		if !strings.HasPrefix(name, "internal/") {
			continue
		}
		if !isTransitionalFoundationPath(name) {
			return fmt.Errorf("unsupported root source path during src layout: %s", name)
		}
		if isNestedGoModuleManifest(name) && !hasExcludedSourceDirectory(name) {
			return fmt.Errorf("nested Go module is not supported during src layout: %s", name)
		}
	}
	return nil
}

// validateTransitionalSnapshotInputs checks the raw module-relative paths and
// modes before extension filtering. Only the transitional src layout receives
// this additional check; historical flat-layout snapshot behavior is unchanged.
func validateTransitionalSnapshotInputs(files map[string][]byte, modes map[string]string, modulePrefix string) error {
	presence := sourceLayoutPresence{}
	var rootInternalFiles []string
	for name := range files {
		if !strings.HasPrefix(name, modulePrefix) {
			continue
		}
		relative := strings.TrimPrefix(name, modulePrefix)
		switch {
		case strings.HasPrefix(relative, "src/cmd/"):
			presence.srcCommand = true
		case strings.HasPrefix(relative, "src/internal/"):
			presence.srcInternal = true
		case strings.HasPrefix(relative, "cmd/"):
			presence.rootCommand = true
		case strings.HasPrefix(relative, "internal/"):
			presence.rootInternal = true
			rootInternalFiles = append(rootInternalFiles, relative)
		}
	}
	if !presence.srcCommand && !presence.srcInternal {
		return nil
	}
	prefix, err := classifySourceLayout(presence)
	if err != nil {
		return fmt.Errorf("fixed source snapshot: %w", err)
	}
	if prefix != "src/" {
		return nil
	}
	if err := validateTransitionalFoundationFiles(rootInternalFiles); err != nil {
		return err
	}
	var sourcePaths []string
	for name := range files {
		if !strings.HasPrefix(name, modulePrefix) {
			continue
		}
		relative := strings.TrimPrefix(name, modulePrefix)
		if strings.HasPrefix(relative, "src/cmd/") || strings.HasPrefix(relative, "src/internal/") || strings.HasPrefix(relative, "internal/") {
			sourcePaths = append(sourcePaths, relative)
		}
	}
	ordered := append([]string(nil), sourcePaths...)
	sort.Strings(ordered)
	for _, name := range ordered {
		if hasExcludedSourceDirectory(name) {
			continue
		}
		mode := modes[modulePrefix+name]
		if mode != "100644" && mode != "100755" {
			return fmt.Errorf("fixed source snapshot source path %q is not regular", modulePrefix+name)
		}
	}
	return nil
}

// hasExcludedSourceDirectory checks directory components only. A filename
// that happens to match an excluded directory name is still a file; an entry
// whose own name is vendor remains visible so a symlink at that path is refused.
func hasExcludedSourceDirectory(name string) bool {
	parts := strings.Split(name, "/")
	for _, component := range parts[:len(parts)-1] {
		if excludedDirs[strings.ToLower(component)] {
			return true
		}
	}
	return false
}

func isNestedGoModuleManifest(name string) bool {
	parts := strings.Split(name, "/")
	return len(parts) > 2 && parts[len(parts)-1] == "go.mod"
}

// validateCollectedSourceLayout requires the same active command and internal
// Go source files from both live collection and fixed-snapshot selection.
func validateCollectedSourceLayout(files []sourceFile, expectedPrefix string) error {
	present := make(map[string]bool, len(files))
	for _, file := range files {
		present[file.name] = true
	}
	commandEntry, internalPrefix, err := moduleLayoutFiles(present)
	if err != nil {
		return err
	}
	actualPrefix := ""
	if strings.HasPrefix(commandEntry, "src/") {
		actualPrefix = "src/"
	}
	if actualPrefix != expectedPrefix {
		return fmt.Errorf("Markitect source layout changed during collection")
	}
	if !present[commandEntry] {
		return fmt.Errorf("source tree is missing required command entrypoint %s", commandEntry)
	}
	for name := range present {
		if strings.HasPrefix(name, internalPrefix) && strings.EqualFold(filepath.Ext(name), ".go") {
			return nil
		}
	}
	return fmt.Errorf("source tree is missing Go source under %s", internalPrefix)
}

// validateTransitionalFoundationDirectories applies the same bounded-root
// policy to the live filesystem, including empty and non-Go directories that
// cannot appear in a fixed file snapshot.
func validateTransitionalFoundationDirectories(moduleDir string) error {
	root := filepath.Join(moduleDir, "internal")
	info, err := os.Lstat(root)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect optional root packages: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("optional root packages must be a real directory")
	}
	allowed := make(map[string]*transitionalPackageDirectory)
	for _, path := range transitionalFoundationRoots() {
		current := allowed
		parts := strings.Split(strings.TrimPrefix(path, "internal/"), "/")
		for index, part := range parts {
			node := current[part]
			if node == nil {
				node = &transitionalPackageDirectory{children: make(map[string]*transitionalPackageDirectory)}
				current[part] = node
			}
			if index == len(parts)-1 {
				node.packageRoot = true
			}
			current = node.children
		}
	}
	return validateTransitionalPackageDirectory(root, "internal", &transitionalPackageDirectory{children: allowed})
}

type transitionalPackageDirectory struct {
	children    map[string]*transitionalPackageDirectory
	packageRoot bool
}

func validateTransitionalPackageDirectory(dir, relative string, node *transitionalPackageDirectory) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read transitional package path %s: %w", relative, err)
	}
	if !node.packageRoot && len(entries) == 0 {
		return fmt.Errorf("transitional package container is incomplete: %s", relative)
	}
	for _, entry := range entries {
		child := node.children[entry.Name()]
		childPath := filepath.Join(dir, entry.Name())
		childRelative := relative + "/" + entry.Name()
		if child == nil {
			return fmt.Errorf("unsupported root internal path during src layout: %s", childRelative)
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			return fmt.Errorf("transitional package path must be a real directory: %s", childRelative)
		}
		if err := requireRealDirectory(childPath); err != nil {
			return fmt.Errorf("transitional package path %s: %w", childRelative, err)
		}
		if child.packageRoot {
			if err := rejectNestedGoModules(childPath, childRelative); err != nil {
				return err
			}
		}
		if len(child.children) > 0 {
			if err := validateTransitionalPackageDirectory(childPath, childRelative, child); err != nil {
				return err
			}
		}
	}
	return nil
}

func rejectNestedGoModules(root, relative string) error {
	return filepath.WalkDir(root, func(current string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return fmt.Errorf("walk transitional package %s: %w", relative, walkErr)
		}
		if entry.IsDir() {
			if current != root && excludedDirs[strings.ToLower(entry.Name())] {
				return filepath.SkipDir
			}
			return nil
		}
		if entry.Name() == "go.mod" {
			path, err := filepath.Rel(root, current)
			if err != nil {
				return fmt.Errorf("resolve nested module path under %s: %w", relative, err)
			}
			return fmt.Errorf("nested Go module is not supported during src layout: %s/%s", relative, filepath.ToSlash(path))
		}
		return nil
	})
}
