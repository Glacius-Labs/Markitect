package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
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
	head      *snapshot.Snapshot
	snapshot  *snapshot.Snapshot
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
	if options.Name == "" || options.Namespace == "" {
		return nil, errors.New("init requires project name and area namespace")
	}
	if options.Path == "" {
		options.Path = ".markitect/areas/" + options.Namespace
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
