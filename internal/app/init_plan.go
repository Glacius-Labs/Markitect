package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func buildInitPlan(root string, options InitOptions) (*initState, error) {
	area := core.Area{Name: options.Namespace, Path: options.Path}
	resource := core.Resource{
		APIVersion: core.APIVersion,
		Kind:       "Project",
		Metadata:   core.Metadata{Name: options.Name},
		Spec:       core.Spec{Areas: []core.Area{area}},
	}
	projectYAML, err := format.Encode(resource)
	if err != nil {
		return nil, fmt.Errorf("encode Project: %w", err)
	}
	readme := []byte(fmt.Sprintf("# %s\n\nThis area contains canonical, typed Markitect resources owned by namespace `%s`. Keep ordinary human-readable project documentation under `docs/`; this directory is for Markitect resources and their local navigation.\n\nUse `markitect authoring` for core authoring guidance. Add links here as area resources are created. Navigation links do not declare resource dependencies.\n", options.Namespace, options.Namespace))
	files := []InitFile{
		{Path: "markitect.yaml", SHA256: initSHA256(projectYAML), Text: string(projectYAML)},
		{Path: strings.TrimSuffix(options.Path, "/") + "/README.md", SHA256: initSHA256(readme), Text: string(readme)},
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Path < files[j].Path })
	state := &initState{root: root, plan: &InitPlan{Kind: "init", Project: options.Name, Area: area, Files: files, Written: []string{}}}
	state.snapshot, err = source.Load(root, "")
	if err != nil {
		return nil, fmt.Errorf("capture init source: %w", err)
	}
	if err := preflightInit(state, options); err != nil {
		return nil, err
	}
	projected := &source.Snapshot{Revision: "init-preview", Provisional: true, Files: cloneByteMap(state.snapshot.Files), Modes: cloneStringMap(state.snapshot.Modes)}
	for _, file := range files {
		projected.Files[file.Path] = []byte(file.Text)
		projected.Modes[file.Path] = "100644"
	}
	project, err := Parse(projected)
	if err != nil {
		return nil, fmt.Errorf("validate init preview: %w", err)
	}
	if len(project.Diagnostics) != 0 {
		return nil, fmt.Errorf("validate init preview: %s", project.Diagnostics[0].Message)
	}
	if diagnostics := CheckOutputs(project); len(diagnostics) != 0 {
		return nil, fmt.Errorf("check init preview outputs: %s", diagnostics[0].Message)
	}
	return state, nil
}

func preflightInit(state *initState, options InitOptions) error {
	paths := make([]string, 0, len(state.snapshot.Files)+len(state.plan.Files))
	for p := range state.snapshot.Files {
		paths = append(paths, p)
	}
	for _, file := range state.plan.Files {
		paths = append(paths, file.Path)
	}
	if err := source.ValidateIncludedPaths(paths); err != nil {
		return fmt.Errorf("validate init paths: %w", err)
	}
	if err := checkInitDiskAliases(state.root, options.Path, state.plan.Files); err != nil {
		return err
	}
	if state.snapshot.Files["markitect.yaml"] != nil {
		return errors.New("refusing to replace existing markitect.yaml")
	}
	if err := checkInitGitState(state); err != nil {
		return err
	}
	if ignored, err := initIgnoredPaths(state.root, pathsForPlan(state.plan)); err != nil {
		return err
	} else if len(ignored) > 0 {
		return fmt.Errorf("init target is ignored by Git: %s", strings.Join(ignored, ", "))
	}
	return nil
}

func checkInitDiskAliases(root, areaPath string, files []InitFile) error {
	for _, file := range files {
		if _, err := safeDestination(root, file.Path); err != nil {
			return fmt.Errorf("unsafe init target %s: %w", file.Path, err)
		}
		parts := strings.Split(file.Path, "/")
		current := root
		for i, part := range parts {
			entries, err := os.ReadDir(current)
			if os.IsNotExist(err) {
				break
			}
			if err != nil {
				return fmt.Errorf("inspect init path %s: %w", file.Path, err)
			}
			found := ""
			for _, entry := range entries {
				if strings.EqualFold(entry.Name(), part) {
					found = entry.Name()
					break
				}
			}
			if found == "" {
				break
			}
			if found != part {
				return fmt.Errorf("case-insensitive init path collision between %q and %q", strings.Join(parts[:i+1], "/"), strings.Join(append(append([]string(nil), parts[:i]...), found), "/"))
			}
			current = filepath.Join(current, found)
			if i == len(parts)-1 {
				return fmt.Errorf("init target already exists: %s", file.Path)
			}
			if filepath.ToSlash(strings.Join(parts[:i+1], "/")) == areaPath {
				return fmt.Errorf("area directory already exists: %s", areaPath)
			}
			info, err := os.Lstat(current)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("unsafe or non-directory init ancestor %s", strings.Join(parts[:i+1], "/"))
			}
		}
	}
	return nil
}

func checkInitGitState(state *initState) error {
	inside, err := source.GitOutput(state.root, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(inside)) != "true" {
		state.insideGit = false
		return nil
	}
	state.insideGit = true
	top, err := source.GitOutput(state.root, "rev-parse", "--show-toplevel")
	if err != nil {
		return fmt.Errorf("resolve init Git root: %w", err)
	}
	topAbs, err := filepath.Abs(strings.TrimSpace(string(top)))
	if err != nil {
		return fmt.Errorf("resolve init Git root: %w", err)
	}
	topCanonical, err := canonicalPathSpelling(topAbs)
	if err != nil {
		return fmt.Errorf("canonicalize init Git root spelling: %w", err)
	}
	rootCanonical, err := canonicalPathSpelling(state.root)
	if err != nil {
		return fmt.Errorf("canonicalize init root spelling: %w", err)
	}
	if !strings.EqualFold(filepath.Clean(topCanonical), filepath.Clean(rootCanonical)) {
		return errors.New("init root must be the Git repository root")
	}
	state.head, err = source.Load(state.root, "HEAD")
	if err != nil {
		if _, headErr := source.GitOutput(state.root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}"); headErr == nil {
			return fmt.Errorf("load committed init baseline: %w", err)
		}
		state.head = nil // An unborn branch has no committed paths yet.
	}

	targetArea := strings.TrimSuffix(state.plan.Area.Path, "/") + "/"
	if state.head != nil {
		for p := range state.head.Files {
			if strings.EqualFold(p, "markitect.yaml") || hasCaseFoldedPrefix(p, targetArea) {
				return fmt.Errorf("refusing to initialize over Git-owned target path %s", p)
			}
		}
	}
	if err := checkNoStagedInitTargets(state.root, targetArea); err != nil {
		return err
	}
	return nil
}

func checkNoStagedInitTargets(root, targetArea string) error {
	index, err := source.GitOutput(root, "ls-files", "--cached", "--full-name", "-z")
	if err != nil {
		return fmt.Errorf("inspect staged init targets: %w", err)
	}
	for _, p := range strings.Split(string(index), "\x00") {
		if p != "" && (strings.EqualFold(p, "markitect.yaml") || hasCaseFoldedPrefix(p, targetArea)) {
			return fmt.Errorf("refusing to initialize over staged Git path %s", p)
		}
	}
	return nil
}

func hasCaseFoldedPrefix(value, prefix string) bool {
	valueRunes := []rune(value)
	prefixRunes := []rune(prefix)
	return len(valueRunes) >= len(prefixRunes) && strings.EqualFold(string(valueRunes[:len(prefixRunes)]), prefix)
}

func initIgnoredPaths(root string, paths []string) ([]string, error) {
	inside, err := source.GitOutput(root, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(string(inside)) != "true" {
		return nil, nil
	}
	ignored := []string{}
	for _, p := range paths {
		_, err := source.GitOutput(root, "check-ignore", "-q", "--no-index", "--", p)
		if err == nil {
			ignored = append(ignored, p)
			continue
		}
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			continue
		}
		return nil, fmt.Errorf("check Git ignore rules for %s: %w", p, err)
	}
	return ignored, nil
}

func checkDiskComponentAlias(parent, component, repoPath string) error {
	entries, err := os.ReadDir(parent)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect init path %s: %w", repoPath, err)
	}
	for _, entry := range entries {
		if strings.EqualFold(entry.Name(), component) && entry.Name() != component {
			return fmt.Errorf("case-insensitive init path collision at %q with existing component %q", repoPath, entry.Name())
		}
	}
	return nil
}

func initSHA256(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}

func cloneByteMap(input map[string][]byte) map[string][]byte {
	result := make(map[string][]byte, len(input))
	for key, value := range input {
		result[key] = append([]byte(nil), value...)
	}
	return result
}

func cloneStringMap(input map[string]string) map[string]string {
	result := make(map[string]string, len(input))
	for key, value := range input {
		result[key] = value
	}
	return result
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func pathsForPlan(plan *InitPlan) []string {
	paths := make([]string, 0, len(plan.Files)*2)
	for _, file := range plan.Files {
		paths = append(paths, file.Path)
		if file.Path != "markitect.yaml" {
			paths = append(paths, filepath.ToSlash(filepath.Dir(filepath.FromSlash(file.Path))))
		}
	}
	return paths
}
