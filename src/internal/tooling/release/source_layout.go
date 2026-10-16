package release

import (
	"fmt"
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
	}
}

type sourceLayoutPresence struct {
	srcCommand   bool
	srcInternal  bool
	rootCommand  bool
	rootInternal bool
}

// classifySourceLayout chooses the active product tree. Root foundations are
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

// isTransitionalFoundationPath accepts Go source only below one of the four
// exact root package prefixes. A sibling such as internal/modeling is not
// admitted by prefix coincidence.
func isTransitionalFoundationPath(name string) bool {
	for _, root := range transitionalFoundationRoots() {
		if strings.HasPrefix(name, root+"/") {
			return true
		}
	}
	return false
}

// validateTransitionalFoundationFiles rejects any root internal source that
// is outside the optional foundation roots. It is shared by live and fixed
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
	}
	return nil
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
		return fmt.Errorf("inspect optional root foundations: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("optional root foundations must be a real directory")
	}
	allowed := make(map[string]bool, len(transitionalFoundationRoots()))
	for _, path := range transitionalFoundationRoots() {
		allowed[strings.TrimPrefix(path, "internal/")] = true
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return fmt.Errorf("read optional root foundations: %w", err)
	}
	for _, entry := range entries {
		if !allowed[entry.Name()] {
			return fmt.Errorf("unsupported root internal path during src layout: internal/%s", entry.Name())
		}
		if entry.Type()&os.ModeSymlink != 0 || !entry.IsDir() {
			return fmt.Errorf("root foundation must be a real directory: internal/%s", entry.Name())
		}
		if err := requireRealDirectory(filepath.Join(root, entry.Name())); err != nil {
			return fmt.Errorf("root foundation internal/%s: %w", entry.Name(), err)
		}
	}
	return nil
}
