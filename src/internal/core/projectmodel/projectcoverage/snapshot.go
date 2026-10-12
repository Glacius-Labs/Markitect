package projectcoverage

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"path"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
)

// FromSnapshot builds a pure in-memory candidate universe from the exact
// files supplied by a caller. It performs no filesystem or Git I/O.
func FromSnapshot(s *snapshot.Snapshot, options Options) (*Universe, error) {
	if s == nil {
		return nil, fmt.Errorf("candidate snapshot is required")
	}
	if err := validateOptions(options); err != nil {
		return nil, err
	}
	ignoreBytes := append([]byte(nil), s.Files[IgnorePath]...)
	if _, err := DecodeIgnore(ignoreBytes); err != nil {
		return nil, err
	}
	paths := make([]PathState, 0, len(s.Files))
	for name, data := range s.Files {
		if err := validatePath(name); err != nil {
			return nil, err
		}
		mode := s.Modes[name]
		if mode != "100644" && mode != "100755" {
			return nil, fmt.Errorf("candidate path %q has unsupported source mode %q", name, mode)
		}
		paths = append(paths, PathState{Path: name, Worktree: FileState{Present: true, Mode: mode, Digest: sha256Hex(data)}})
	}
	if err := ValidateRepositoryPaths(pathNames(paths)); err != nil {
		return nil, err
	}
	result := NewUniverse(s.ID, "snapshot", paths, ignoreBytes, options, false)
	result.Snapshot = cloneSnapshot(s)
	return result, nil
}

func pathNames(states []PathState) []string {
	paths := make([]string, len(states))
	for i, state := range states {
		paths[i] = state.Path
	}
	return paths
}

func cloneSnapshot(input *snapshot.Snapshot) *snapshot.Snapshot {
	if input == nil {
		return nil
	}
	result := &snapshot.Snapshot{ID: input.ID, Provisional: input.Provisional, Files: make(map[string][]byte, len(input.Files)), Modes: make(map[string]string, len(input.Modes))}
	for name, data := range input.Files {
		result.Files[name] = append([]byte(nil), data...)
	}
	for name, mode := range input.Modes {
		result.Modes[name] = mode
	}
	return result
}

func validatePath(name string) error {
	cleanName := strings.TrimSuffix(name, "/")
	if cleanName == "" || !utf8.ValidString(cleanName) || strings.ContainsAny(cleanName, "\\:\x00") || strings.HasPrefix(cleanName, "/") || path.Clean(cleanName) != cleanName {
		return fmt.Errorf("unsafe or non-normalized repository path %q", name)
	}
	for _, part := range strings.Split(cleanName, "/") {
		if !safePathComponent(part) {
			return fmt.Errorf("repository path %q contains a reserved or traversal component", name)
		}
	}
	return nil
}

func safePathComponent(part string) bool {
	if part == "" || part == "." || part == ".." || strings.EqualFold(part, ".git") || strings.TrimRight(part, " .") != part || strings.ContainsAny(part, "<>:\\|?*\x00") {
		return false
	}
	for _, r := range part {
		if unicode.IsControl(r) {
			return false
		}
	}
	base := strings.ToUpper(strings.SplitN(part, ".", 2)[0])
	switch base {
	case "CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4", "COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4", "LPT5", "LPT6", "LPT7", "LPT8", "LPT9":
		return false
	}
	return true
}

func validatePathSet(paths map[string]*PathState) error {
	seen := map[string]string{}
	terminals := map[string]string{}
	for name := range paths {
		clean := strings.TrimSuffix(name, "/")
		terminalKey := strings.ToLower(clean)
		if previous, ok := terminals[terminalKey]; ok && previous != name {
			return fmt.Errorf("repository contains colliding file and repository-boundary paths %q and %q", previous, name)
		}
		terminals[terminalKey] = name
		parts := strings.Split(clean, "/")
		for i := range parts {
			actual := strings.Join(parts[:i+1], "/")
			key := strings.ToLower(actual)
			if previous, ok := seen[key]; ok && previous != actual {
				return fmt.Errorf("repository contains case-colliding paths %q and %q", previous, actual)
			}
			seen[key] = actual
		}
	}
	return nil
}

func validateUniversePaths(paths []PathState) error {
	set := make(map[string]*PathState, len(paths))
	for _, state := range paths {
		if strings.HasSuffix(state.Path, "/") && !state.OpaqueBoundary {
			return fmt.Errorf("repository file path %q unexpectedly ends with a slash", state.Path)
		}
		if err := validatePath(state.Path); err != nil {
			return err
		}
		if _, exists := set[state.Path]; exists {
			return fmt.Errorf("candidate contains duplicate path %q", state.Path)
		}
		set[state.Path] = &PathState{Path: state.Path}
	}
	return validatePathSet(set)
}

func sha256Hex(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}
