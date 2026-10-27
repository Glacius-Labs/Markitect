package release

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/tooling/guardedwrite"
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
)

var installPinPaths = []string{
	".markitect/tool/lock.yaml",
	".markitect/bootstrap/run_test.go",
	".markitect/bootstrap/run.go",
	".markitect/tool/source.zip",
}

var installPaths = []string{
	".markitect/tool/lock.yaml",
	".markitect/bootstrap/run_test.go",
	".markitect/bootstrap/run.go",
	".markitect/tool/release.yaml",
	".markitect/tool/source.zip",
}

// InstallPlan describes a complete release-pin change. Install performs only
// this preflight plan unless write is true. Per-file replacement is atomic;
// the five-file update is not a filesystem transaction.
type InstallPlan struct {
	Kind         string            `yaml:"kind"`
	Version      string            `yaml:"version"`
	SourceCommit string            `yaml:"sourceCommit"`
	BundleSHA256 string            `yaml:"bundleSha256"`
	Files        []InstallFilePlan `yaml:"files"`
	Written      []string          `yaml:"written,omitempty"`
	Applied      bool              `yaml:"applied"`
	Recovery     string            `yaml:"recovery,omitempty"`
}

// InstallFilePlan reports the expected digest and operation for one pin file.
type InstallFilePlan struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
	Action string `yaml:"action"`
}

type installFileState struct {
	data      []byte // raw filesystem bytes
	canonical []byte // LF-normalized text bytes for pin/HEAD validation
	mode      fs.FileMode
	exists    bool
}

type installState struct {
	files  map[string]installFileState
	head   *source.Input
	branch string
	kind   string
}

// Install validates a release bundle and returns a safe installation plan.
// With write=false it makes no filesystem changes. With write=true it requires
// a named non-protected branch and committed HEAD, repeats the full preflight
// under Markitect's shared writer lock, then writes the complete five-file set.
func Install(root string, bundle *Bundle, write bool) (*InstallPlan, error) {
	if err := validateIncomingBundle(bundle); err != nil {
		return nil, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve install root: %w", err)
	}
	if info, statErr := os.Lstat(rootAbs); statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		if statErr != nil {
			return nil, fmt.Errorf("install root must be an existing real directory: %w", statErr)
		}
		return nil, errors.New("install root must be an existing real directory")
	}

	var branch string
	if write {
		branch, err = installableBranch(rootAbs)
		if err != nil {
			return nil, err
		}
	}

	plan, state, err := buildInstallPlan(rootAbs, bundle)
	if err != nil {
		return nil, err
	}
	state.branch = branch
	if !write {
		return plan, nil
	}
	if state.head == nil {
		return plan, errors.New("install write requires a committed Git HEAD so the resulting pin set has a recovery baseline")
	}
	if current, branchErr := installableBranch(rootAbs); branchErr != nil || current != branch {
		if branchErr != nil {
			return plan, fmt.Errorf("branch changed or became protected during install preflight: %w", branchErr)
		}
		return plan, errors.New("branch changed during install preflight; rerun the plan")
	}
	capture, err := guardedwrite.CaptureFiles(rootAbs, installPaths)
	if err != nil {
		return plan, err
	}
	changes := make([]guardedwrite.Change, 0, len(plan.Files))
	for _, file := range plan.Files {
		if file.Action == "unchanged" {
			continue
		}
		changes = append(changes, guardedwrite.Change{
			Path: file.Path, Bytes: bundle.Files[file.Path], Mode: 0644,
		})
	}
	changedPaths := make([]string, len(changes))
	for i, change := range changes {
		changedPaths[i] = change.Path
	}

	preflight := func() error {
		if err := ensureInstallStateUnchanged(rootAbs, state); err != nil {
			return err
		}
		return ensureCapturedMatchesState(capture, state)
	}
	beforeTarget := func() error {
		return ensureNoStagedInstallPins(rootAbs, true)
	}
	readback := func(result guardedwrite.Result) error {
		if !samePaths(result.CompletedPaths, changedPaths) {
			return fmt.Errorf("guarded install completed paths %v do not match planned changes %v", result.CompletedPaths, changedPaths)
		}
		if err := ensureInstallBranchAndHead(rootAbs, state); err != nil {
			return err
		}
		if err := ensureNoStagedInstallPins(rootAbs, true); err != nil {
			return err
		}
		if err := verifyInstalledBundle(rootAbs, bundle); err != nil {
			return fmt.Errorf("verify completed install: %w", err)
		}
		return nil
	}

	result, applyErr := guardedwrite.ApplyOrderedChecked(context.Background(), rootAbs, capture, changes, preflight, beforeTarget, readback)
	plan.Written = append([]string(nil), result.CompletedPaths...)
	if applyErr != nil {
		return installWriteFailure(plan, state, applyErr)
	}
	plan.Applied = true
	return plan, nil
}

func ensureCapturedMatchesState(capture *guardedwrite.Capture, state *installState) error {
	if capture == nil {
		return errors.New("installer capture is missing")
	}
	for _, name := range installPaths {
		captured, ok := capture.Files[name]
		if !ok {
			return fmt.Errorf("installer capture is missing selected path %s", name)
		}
		before := state.files[name]
		if captured.Exists != before.exists || (captured.Exists && (!bytes.Equal(captured.Bytes, before.data) || captured.Mode.Perm() != before.mode.Perm())) {
			return fmt.Errorf("install target changed between plan and guarded capture: %s", name)
		}
	}
	return nil
}

func samePaths(left, right []string) bool {
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}
