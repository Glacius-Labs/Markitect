package host

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

func ensureInitStateUnchanged(state *initState) error {
	if state.writer != nil {
		if err := state.writer.checkIdentity(); err != nil {
			return err
		}
	}
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
		var info os.FileInfo
		var err error
		if state.writer != nil {
			info, err = state.writer.Lstat(state.plan.Area.Path)
		} else {
			areaPath := filepath.Join(state.root, filepath.FromSlash(state.plan.Area.Path))
			info, err = os.Lstat(areaPath)
		}
		if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.IsDir() || !os.SameFile(state.areaInfo, info) {
			return errors.New("initialized area directory changed during init")
		}
	}
	for _, file := range state.plan.Files {
		var data []byte
		var err error
		if state.writer != nil {
			data, err = state.writer.ReadFile(file.Path)
		} else {
			data, err = os.ReadFile(filepath.Join(state.root, filepath.FromSlash(file.Path)))
		}
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
	if state.writer == nil {
		return errors.New("init write root is not anchored")
	}
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
			if err := state.writer.Mkdir(rel, 0755); err != nil {
				return fmt.Errorf("create area directory exclusively %s: %w", rel, err)
			}
			state.plan.CreatedDirectories = append(state.plan.CreatedDirectories, rel)
			info, err := state.writer.Lstat(rel)
			if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
				return fmt.Errorf("area directory became unsafe after creation: %s", rel)
			}
			state.areaInfo = info
			continue
		}
		if err := state.writer.Mkdir(rel, 0755); err == nil {
			state.plan.CreatedDirectories = append(state.plan.CreatedDirectories, rel)
		} else if !os.IsExist(err) {
			return fmt.Errorf("create parent directory %s: %w", rel, err)
		}
		info, err := state.writer.Lstat(rel)
		if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("unsafe init parent directory %s", rel)
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
	if err != nil || strings.TrimSpace(string(current)) != state.head.ID {
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
