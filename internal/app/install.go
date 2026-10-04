package app

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Glacius-Labs/Markitect/internal/release"
	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
)

const releaseManifestPath = ".markitect/tool/release.yaml"

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
	exists    bool
}

type installState struct {
	files  map[string]installFileState
	head   *snapshot.Snapshot
	branch string
	kind   string
}

// Install validates a release bundle and returns a safe installation plan.
// With write=false it makes no filesystem changes. With write=true it requires
// a named non-protected branch, holds Markitect's shared writer lock, repeats
// the full preflight, then writes the complete five-file set.
func Install(root string, bundle *release.Bundle, write bool) (*InstallPlan, error) {
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
	var unlock func()
	if write {
		branch, err = installableBranch(rootAbs)
		if err != nil {
			return nil, err
		}
		unlock, err = lockWriter(rootAbs)
		if err != nil {
			return nil, err
		}
		defer unlock()
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
	if err := ensureInstallStateUnchanged(rootAbs, state); err != nil {
		return plan, err
	}
	if plan.Kind == "noop" {
		plan.Applied = true
		return plan, nil
	}

	for _, file := range plan.Files {
		if file.Action == "unchanged" {
			continue
		}
		if err := ensureInstallStateUnchanged(rootAbs, state); err != nil {
			return installWriteFailure(plan, state, err)
		}
		dest, err := safeInstallDestination(rootAbs, file.Path)
		if err != nil {
			return installWriteFailure(plan, state, err)
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return installWriteFailure(plan, state, fmt.Errorf("create parent directory for %s: %w", file.Path, err))
		}
		if _, err = safeInstallDestination(rootAbs, file.Path); err != nil {
			return installWriteFailure(plan, state, err)
		}
		current, err := readInstallFile(rootAbs, file.Path)
		if err != nil {
			return installWriteFailure(plan, state, err)
		}
		before := state.files[file.Path]
		if current.exists != before.exists || (current.exists && !bytes.Equal(current.data, before.data)) {
			return installWriteFailure(plan, state, fmt.Errorf("install target changed during write: %s", file.Path))
		}
		if err := ensureWriteBranch(rootAbs, state.branch); err != nil {
			return installWriteFailure(plan, state, err)
		}
		if err = atomicWrite(dest, bundle.Files[file.Path]); err != nil {
			return installWriteFailure(plan, state, fmt.Errorf("write %s: %w", file.Path, err))
		}
		plan.Written = append(plan.Written, file.Path)
		data := append([]byte(nil), bundle.Files[file.Path]...)
		state.files[file.Path] = installFileState{data: data, canonical: canonicalInstallReadback(file.Path, data), exists: true}
	}

	if err := verifyInstalledBundle(rootAbs, bundle); err != nil {
		return installWriteFailure(plan, state, fmt.Errorf("verify completed install: %w", err))
	}
	if err := ensureInstallStateUnchanged(rootAbs, state); err != nil {
		return installWriteFailure(plan, state, err)
	}
	plan.Applied = true
	return plan, nil
}
