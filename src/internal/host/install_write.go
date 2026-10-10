package host

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/src/internal/host/guardedwrite"
	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/src/internal/tooling/release"
)

func ensureInstallStateUnchanged(root string, state *installState) error {
	branch, err := installableBranch(root)
	if err != nil && state.branch != "" {
		return fmt.Errorf("install branch is no longer writable: %w", err)
	}
	if state.branch != "" && branch != state.branch {
		return errors.New("branch changed during install")
	}
	if err := ensureHeadUnchanged(root, state.head); err != nil {
		return err
	}
	if state.head != nil {
		if err := ensureNoStagedInstallPins(root, true); err != nil {
			return fmt.Errorf("staged release pins changed during install: %w", err)
		}
	}
	for _, name := range installPaths {
		current, err := readInstallFile(root, name)
		if err != nil {
			return err
		}
		before := state.files[name]
		if current.exists != before.exists || (current.exists && !bytes.Equal(current.data, before.data)) {
			return fmt.Errorf("release pin changed during install: %s", name)
		}
	}
	return nil
}

func ensureHeadUnchanged(root string, before *snapshot.Snapshot) error {
	if before == nil {
		out, err := source.GitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return errors.New("Git HEAD appeared during a fresh install; rerun the plan")
		}
		return nil
	}
	after, err := source.GitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		return fmt.Errorf("read Git HEAD during install: %w", err)
	}
	if strings.TrimSpace(string(after)) != before.ID {
		return errors.New("Git HEAD changed during install; rerun the plan")
	}
	return nil
}

func installWriteFailure(plan *InstallPlan, state *installState, cause error) (*InstallPlan, error) {
	if len(plan.Written) > 0 {
		if state.kind == "install" {
			plan.Recovery = "The five-file installation is partial; do not use it. Remove the listed newly written files after reviewing them, or complete the exact bundle and rerun verification."
		} else {
			plan.Recovery = "The five-file installation is partial; do not use it. Restore the complete pin set from the prior committed candidate or revert the integration commit through the normal Git workflow, then verify."
		}
		return plan, fmt.Errorf("install stopped after writing %s; no complete installation is claimed: %w", strings.Join(plan.Written, ", "), cause)
	}
	return plan, cause
}

func verifyInstalledBundle(root string, bundle *release.Bundle) error {
	files := make(map[string][]byte, len(installPaths))
	for _, name := range installPaths {
		file, err := readInstallFile(root, name)
		if err != nil {
			return err
		}
		if !file.exists {
			return fmt.Errorf("installed file is missing: %s", name)
		}
		if !bytes.Equal(file.canonical, bundle.Files[name]) {
			return fmt.Errorf("installed file differs from the selected bundle: %s", name)
		}
		files[name] = file.canonical
	}
	manifest, err := release.ParseBundleManifest(files[releaseManifestPath])
	if err != nil {
		return err
	}
	return release.ValidateBundleFiles(manifest, files, bundle.Manifest.SourceCommit)
}

func safeInstallDestination(root, name string) (string, error) {
	dest, err := guardedwrite.SafeDestination(root, name)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	current := absolute
	parts := strings.Split(name, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return "", fmt.Errorf("inspect install path %s: %w", name, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink in install path %s", name)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("unmanaged file blocks install path %s at %s", name, part)
		}
	}
	return dest, nil
}
