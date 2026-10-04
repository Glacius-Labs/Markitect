package host

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/render"
)

func WriteOutputs(root string, p *Project) ([]string, error) {
	return writeOutputs(root, p, nil)
}

// writeOutputPaths applies an explicitly selected subset of the generated
// outputs. An empty slice is a read-only freshness check and performs no
// filesystem writes.
func writeOutputPaths(root string, p *Project, selected []string) ([]string, error) {
	return writeOutputs(root, p, selected)
}

func writeOutputs(root string, p *Project, selected []string) ([]string, error) {
	if p == nil || p.Snapshot == nil || p.Graph == nil || p.Graph.Project == nil {
		return nil, fmt.Errorf("loaded project snapshot is required")
	}
	if !p.Snapshot.Provisional {
		return nil, fmt.Errorf("writing a snapshot is forbidden; render the isolated working tree")
	}
	if len(p.Diagnostics) > 0 {
		return nil, fmt.Errorf("cannot render while project diagnostics remain")
	}
	if findings := checkProviderAdapterInputs(p); len(findings) > 0 {
		return nil, fmt.Errorf("cannot render with invalid provider adapter inputs: %s: %s", findings[0].Path, findings[0].Message)
	}
	if selected != nil && len(selected) == 0 {
		return verifyNoopWrite(root, p)
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
	outputs, err := render.Generate(p.Graph, p.Snapshot.Files)
	if err != nil {
		return nil, err
	}
	names := sortedFiles(outputs)
	if selected != nil {
		selectedSet := make(map[string]bool, len(selected))
		for _, name := range selected {
			if selectedSet[name] {
				return nil, fmt.Errorf("duplicate selected output path %q", name)
			}
			if _, ok := outputs[name]; !ok {
				return nil, fmt.Errorf("selected path is not a generated output: %s", name)
			}
			selectedSet[name] = true
		}
		names = names[:0]
		for name := range selectedSet {
			names = append(names, name)
		}
		sort.Strings(names)
		if len(names) == 0 {
			return verifyNoopWrite(root, p)
		}
	}
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
	// Imported resources and Domain definitions are immutable archive members,
	// not paths in the consumer checkout. Recheck their physical pinned archives
	// together with local canonical sources before writing any output.
	sourcePaths := map[string]bool{}
	for _, r := range p.Resources {
		if r.Package == "" {
			sourcePaths[r.Path] = true
		}
	}
	for _, domain := range p.DomainInputs {
		if domain.Package == "" {
			sourcePaths[domain.Path] = true
		}
	}
	for _, pin := range p.Graph.Project.Spec.Packages {
		sourcePaths[pin.Archive] = true
	}
	physicalSources := make([]string, 0, len(sourcePaths))
	for name := range sourcePaths {
		physicalSources = append(physicalSources, name)
	}
	sort.Strings(physicalSources)
	for _, name := range physicalSources {
		dest, err := safeDestination(root, name)
		if err != nil {
			return nil, err
		}
		current, err := os.ReadFile(dest)
		if err != nil || !bytes.Equal(current, p.Snapshot.Files[name]) {
			return nil, fmt.Errorf("source changed since capture: %s", name)
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

func verifyNoopWrite(root string, p *Project) ([]string, error) {
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
	fresh, err := source.Load(root, "")
	if err != nil {
		return nil, err
	}
	if fresh.Digest() != p.Snapshot.Digest() {
		return nil, fmt.Errorf("source inventory changed since capture; reload before rendering")
	}
	if branch != "" {
		if err := ensureWriteBranch(rootAbs, branch); err != nil {
			return nil, err
		}
	}
	return []string{}, nil
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
