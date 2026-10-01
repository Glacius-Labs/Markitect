package app

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/render"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

func WriteOutputs(root string, p *Project) ([]string, error) {
	if !p.Snapshot.Provisional {
		return nil, fmt.Errorf("writing a snapshot is forbidden; render the isolated working tree")
	}
	if len(p.Diagnostics) > 0 {
		return nil, fmt.Errorf("cannot render while project diagnostics remain")
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	branch := ""
	if hasGitMetadata(rootAbs) {
		branch, err = writeBranchName(rootAbs)
		if err != nil {
			return nil, err
		}
	}
	lockDir := filepath.Join(rootAbs, ".artifacts", "markitect")
	if _, err = safeDestination(rootAbs, ".artifacts/markitect/write.lock"); err != nil {
		return nil, err
	}
	if err = os.MkdirAll(lockDir, 0755); err != nil {
		return nil, err
	}
	lock, err := os.OpenFile(filepath.Join(lockDir, "write.lock"), os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("another renderer owns write.lock (inspect an abandoned lock before removing it): %w", err)
	}
	lock.Close()
	defer os.Remove(filepath.Join(lockDir, "write.lock"))
	if branch != "" {
		if err := ensureWriteBranch(rootAbs, branch); err != nil {
			return nil, err
		}
	}
	outputs, err := render.Generate(p.Graph)
	if err != nil {
		return nil, err
	}
	names := sortedFiles(outputs)
	fresh, err := source.Load(root, "")
	if err != nil {
		return nil, err
	}
	if fresh.Digest() != p.Snapshot.Digest() {
		return nil, fmt.Errorf("source inventory changed since capture; reload before rendering")
	}
	// Validate the entire plan and reject concurrent edits before writing anything.
	for _, name := range names {
		if branch != "" {
			if err := ensureWriteBranch(rootAbs, branch); err != nil {
				return nil, err
			}
		}
		dest, err := safeDestination(root, name)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(dest)
		original, existed := p.Snapshot.Files[name]
		if err == nil {
			if !existed || !bytes.Equal(data, original) {
				return nil, fmt.Errorf("output changed since source capture: %s", name)
			}
			if !Generated(data) && !bytes.Equal(normalize(data), normalize(outputs[name])) {
				return nil, fmt.Errorf("refusing to replace an unmanaged file: %s", name)
			}
		} else if !os.IsNotExist(err) {
			return nil, err
		} else if existed {
			return nil, fmt.Errorf("output removed since source capture: %s", name)
		}
	}
	// Recheck all YAML source bytes, including configuration, before output writes.
	for _, r := range p.Resources {
		dest, err := safeDestination(root, r.Path)
		if err != nil {
			return nil, err
		}
		current, err := os.ReadFile(dest)
		if err != nil || !bytes.Equal(current, p.Snapshot.Files[r.Path]) {
			return nil, fmt.Errorf("source changed since capture: %s", r.Path)
		}
	}
	var written []string
	for _, name := range names {
		if branch != "" {
			if err := ensureWriteBranch(rootAbs, branch); err != nil {
				return written, err
			}
		}
		dest, err := safeDestination(root, name)
		if err != nil {
			return written, err
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return written, err
		}
		// The author must hold a single-writer worktree. Revalidation reduces accidental
		// races; a filesystem cannot provide a multi-file transaction here.
		if _, err = safeDestination(root, name); err != nil {
			return written, err
		}
		current, readErr := os.ReadFile(dest)
		original, existed := p.Snapshot.Files[name]
		if existed && (readErr != nil || !bytes.Equal(current, original)) {
			return written, fmt.Errorf("output changed during render: %s", name)
		}
		if !existed && !os.IsNotExist(readErr) {
			return written, fmt.Errorf("output appeared during render: %s", name)
		}
		if branch != "" {
			if err := ensureWriteBranch(rootAbs, branch); err != nil {
				return written, err
			}
		}
		if err = atomicWrite(dest, outputs[name]); err != nil {
			return written, err
		}
		written = append(written, name)
	}
	sort.Strings(written)
	final, err := source.Load(root, "")
	if err != nil {
		return written, err
	}
	// Our own output changes are expected. Any other concurrent edit means this
	// batch is not a valid rendered candidate and must be checked again.
	for _, name := range names {
		delete(final.Files, name)
		delete(fresh.Files, name)
	}
	if final.Digest() != fresh.Digest() {
		return written, fmt.Errorf("source changed during rendering; outputs are provisional, rerun check")
	}
	if branch != "" {
		if err := ensureWriteBranch(rootAbs, branch); err != nil {
			return written, err
		}
	}
	return written, nil
}

// WriteSchemas only owns its generated schema files; path validation also
// protects standalone tool checkouts that do not yet have Git metadata.
func WriteSchemas(root string, outputs map[string][]byte) error {
	branch := ""
	var unlock func()
	if _, err := os.Lstat(filepath.Join(root, ".git")); err == nil {
		branch, err = writeBranchName(root)
		if err != nil {
			return err
		}
		unlock, err = lockWriter(root)
		if err != nil {
			return err
		}
		defer unlock()
		if err := ensureWriteBranch(root, branch); err != nil {
			return err
		}
	}
	for _, name := range sortedFiles(outputs) {
		if !strings.HasPrefix(name, "schema/") {
			return fmt.Errorf("invalid schema target %s", name)
		}
		dest, err := safeDestination(root, name)
		if err != nil {
			return err
		}
		old, err := os.ReadFile(dest)
		if err == nil && !Generated(old) {
			return fmt.Errorf("unmanaged schema output %s", name)
		}
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	for _, name := range sortedFiles(outputs) {
		if branch != "" {
			if err := ensureWriteBranch(root, branch); err != nil {
				return err
			}
		}
		dest, err := safeDestination(root, name)
		if err != nil {
			return err
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}
		if branch != "" {
			if err := ensureWriteBranch(root, branch); err != nil {
				return err
			}
		}
		if err = atomicWrite(dest, outputs[name]); err != nil {
			return err
		}
	}
	if branch != "" {
		if err := ensureWriteBranch(root, branch); err != nil {
			return err
		}
	}
	return nil
}
