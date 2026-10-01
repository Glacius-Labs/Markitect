package app

import (
	"bytes"
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

// InitOptions identifies one empty Markitect Project and its first area.
type InitOptions struct {
	Name      string
	Namespace string
	Path      string
}

// InitFile describes one exact UTF-8 file in an initialization plan.
type InitFile struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
	Text   string `yaml:"text"`
}

// InitPlan reports the preview or result of creating a minimal Project.
// The writes are exclusive per-file creates and are not a filesystem
// transaction; Recovery identifies files and directories left by a partial write.
type InitPlan struct {
	Kind               string     `yaml:"kind"`
	Project            string     `yaml:"project"`
	Area               core.Area  `yaml:"area"`
	Files              []InitFile `yaml:"files"`
	Applied            bool       `yaml:"applied"`
	Written            []string   `yaml:"written,omitempty"`
	CreatedDirectories []string   `yaml:"createdDirectories,omitempty"`
	Recovery           string     `yaml:"recovery,omitempty"`
}

type initState struct {
	root      string
	branch    string
	insideGit bool
	head      *source.Snapshot
	snapshot  *source.Snapshot
	plan      *InitPlan
	areaInfo  os.FileInfo
}

// Init builds and optionally applies a minimal Project+area plan. A preview is
// read-only. Writes re-run all validation, require an isolated Git branch and
// use exclusive creates; a later call never applies a prior preview object.
func Init(root string, options InitOptions, write bool) (*InitPlan, error) {
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve init root: %w", err)
	}
	if info, statErr := os.Lstat(rootAbs); statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		if statErr != nil {
			return nil, fmt.Errorf("init root must be an existing real directory: %w", statErr)
		}
		return nil, errors.New("init root must be an existing real directory")
	}
	if options.Name == "" || options.Namespace == "" || options.Path == "" {
		return nil, errors.New("init requires project name, area namespace, and area path")
	}

	var branch string
	var unlock func()
	if write {
		branch, err = writeBranchName(rootAbs)
		if err != nil {
			return nil, fmt.Errorf("init write requires a named non-protected Git branch: %w", err)
		}
		unlock, err = lockWriter(rootAbs)
		if err != nil {
			return nil, err
		}
		defer unlock()
	}

	state, err := buildInitPlan(rootAbs, options)
	if err != nil {
		return nil, err
	}
	state.branch = branch
	if !write {
		return state.plan, nil
	}
	if err := ensureInitStateUnchanged(state); err != nil {
		return state.plan, err
	}
	if err := createInitArea(state); err != nil {
		return initWriteFailure(state.plan, err)
	}

	for _, file := range state.plan.Files {
		if err := ensureInitStateUnchanged(state); err != nil {
			return initWriteFailure(state.plan, err)
		}
		dest, err := safeDestination(rootAbs, file.Path)
		if err != nil {
			return initWriteFailure(state.plan, err)
		}
		if err := ensureWriteBranch(rootAbs, branch); err != nil {
			return initWriteFailure(state.plan, err)
		}
		if _, err := safeDestination(rootAbs, file.Path); err != nil {
			return initWriteFailure(state.plan, err)
		}
		f, err := os.OpenFile(dest, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0644)
		if err != nil {
			return initWriteFailure(state.plan, fmt.Errorf("create %s exclusively: %w", file.Path, err))
		}
		state.plan.Written = append(state.plan.Written, file.Path)
		_, writeErr := f.Write([]byte(file.Text))
		syncErr := f.Sync()
		closeErr := f.Close()
		if writeErr != nil || syncErr != nil || closeErr != nil {
			return initWriteFailure(state.plan, errors.Join(writeErr, syncErr, closeErr))
		}
		state.snapshot.Files[file.Path] = []byte(file.Text)
		state.snapshot.Modes[file.Path] = "100644"
	}

	final, err := source.Load(rootAbs, "")
	if err != nil {
		return initWriteFailure(state.plan, fmt.Errorf("read back initialized Project: %w", err))
	}
	if final.Digest() != state.snapshot.Digest() {
		return initWriteFailure(state.plan, errors.New("source changed during init write; inspect the listed files and rerun check"))
	}
	project, err := Parse(final)
	if err != nil {
		return initWriteFailure(state.plan, fmt.Errorf("validate initialized Project: %w", err))
	}
	if len(project.Diagnostics) != 0 {
		return initWriteFailure(state.plan, fmt.Errorf("validate initialized Project: %s", project.Diagnostics[0].Message))
	}
	if diagnostics := CheckOutputs(project); len(diagnostics) != 0 {
		return initWriteFailure(state.plan, fmt.Errorf("check initialized Project outputs: %s", diagnostics[0].Message))
	}
	if err := ensureInitStateUnchanged(state); err != nil {
		return initWriteFailure(state.plan, err)
	}
	state.plan.Applied = true
	return state.plan, nil
}

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
	readme := []byte(fmt.Sprintf("# %s\n\nThis area owns Markitect resources in namespace `%s`.\n\nUse `markitect authoring` for core authoring guidance. Add links here as area resources are created.\n", options.Namespace, options.Namespace))
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

func ensureInitStateUnchanged(state *initState) error {
	if err := ensureInitGitState(state); err != nil {
		return err
	}
	if state.insideGit {
		if err := checkNoStagedInitTargets(state.root, strings.TrimSuffix(state.plan.Area.Path, "/")+"/"); err != nil {
			return err
		}
	}
	current, err := source.Load(state.root, "")
	if err != nil {
		return fmt.Errorf("recheck init source: %w", err)
	}
	if current.Digest() != state.snapshot.Digest() {
		return errors.New("source changed during init; rerun the plan")
	}
	if state.areaInfo != nil {
		areaPath := filepath.Join(state.root, filepath.FromSlash(state.plan.Area.Path))
		info, err := os.Lstat(areaPath)
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !os.SameFile(state.areaInfo, info) {
			return errors.New("initialized area directory changed during init")
		}
	}
	for _, file := range state.plan.Files {
		data, err := os.ReadFile(filepath.Join(state.root, filepath.FromSlash(file.Path)))
		written := containsString(state.plan.Written, file.Path)
		if written {
			if err != nil || !bytes.Equal(data, []byte(file.Text)) {
				return fmt.Errorf("written init file changed during init: %s", file.Path)
			}
		} else if !os.IsNotExist(err) {
			return fmt.Errorf("init target appeared during write: %s", file.Path)
		}
	}
	return ensureWriteBranch(state.root, state.branch)
}

// createInitArea creates parent directories as needed, but never adopts the
// requested area directory: its final mkdir is exclusive. Every directory
// created before a later failure is retained in the plan for recovery.
func createInitArea(state *initState) error {
	parts := strings.Split(state.plan.Area.Path, "/")
	current := state.root
	for i, part := range parts {
		if err := ensureWriteBranch(state.root, state.branch); err != nil {
			return err
		}
		current = filepath.Join(current, part)
		rel := strings.Join(parts[:i+1], "/")
		if err := checkDiskComponentAlias(filepath.Dir(current), part, rel); err != nil {
			return err
		}
		if i == len(parts)-1 {
			if err := ensureWriteBranch(state.root, state.branch); err != nil {
				return err
			}
			if err := os.Mkdir(current, 0755); err != nil {
				return fmt.Errorf("create area directory exclusively %s: %w", rel, err)
			}
			state.plan.CreatedDirectories = append(state.plan.CreatedDirectories, rel)
			info, err := os.Lstat(current)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("area directory became unsafe after creation: %s", rel)
			}
			state.areaInfo = info
			continue
		}
		if err := os.Mkdir(current, 0755); err == nil {
			state.plan.CreatedDirectories = append(state.plan.CreatedDirectories, rel)
		} else if !os.IsExist(err) {
			return fmt.Errorf("create parent directory %s: %w", rel, err)
		}
		info, err := os.Lstat(current)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe init parent directory %s", rel)
		}
	}
	return nil
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

func ensureInitGitState(state *initState) error {
	if !state.insideGit {
		return nil
	}
	if err := ensureWriteBranch(state.root, state.branch); err != nil {
		return err
	}
	current, err := source.GitOutput(state.root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if state.head == nil {
		if err == nil {
			return errors.New("Git HEAD appeared during init; rerun the plan")
		}
		return nil
	}
	if err != nil || strings.TrimSpace(string(current)) != state.head.Revision {
		return errors.New("Git HEAD changed during init; rerun the plan")
	}
	return nil
}

func initWriteFailure(plan *InitPlan, cause error) (*InitPlan, error) {
	if len(plan.Written) > 0 || len(plan.CreatedDirectories) > 0 {
		created := append(append([]string(nil), plan.Written...), plan.CreatedDirectories...)
		plan.Recovery = "Initialization is partial; inspect the listed files and directories plus Git diff, complete the Project deliberately, then run markitect check. No files or directories were removed automatically."
		return plan, fmt.Errorf("init stopped after creating %s: %w", strings.Join(created, ", "), cause)
	}
	return plan, cause
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
