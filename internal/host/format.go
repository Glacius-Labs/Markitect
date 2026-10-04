package host

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"github.com/Glacius-Labs/Markitect/internal/host/authoring"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
)

// Format canonicalizes YAML and line endings, without changing its model.
func Format(root string, p *Project, write bool) ([]string, error) {
	if len(p.Diagnostics) > 0 {
		return nil, fmt.Errorf("resolve project diagnostics before formatting")
	}
	changed := map[string][]byte{}
	for _, resource := range p.Resources {
		if resource.Package != "" || resource.Kind == "Package" {
			continue
		}
		copy := *resource
		copy.Spec.Text = strings.ReplaceAll(copy.Spec.Text, "\r\n", "\n")
		data, err := authoring.Encode(copy)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(normalize(data), normalize(p.Snapshot.Files[resource.Path])) {
			changed[resource.Path] = data
		}
	}
	for _, input := range p.DomainInputs {
		if input.Package != "" {
			continue
		}
		definition, err := authoring.ParseDomain(input.Path, p.Snapshot.Files[input.Path])
		if err != nil {
			return nil, err
		}
		data, err := authoring.EncodeDomain(definition)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(normalize(data), normalize(p.Snapshot.Files[input.Path])) {
			changed[input.Path] = data
		}
	}
	names := sortedFiles(changed)
	if !write {
		return names, nil
	}
	if !p.Snapshot.Provisional {
		return nil, fmt.Errorf("format writes require an isolated working tree")
	}
	branch, err := writeBranchName(root)
	if err != nil {
		return nil, err
	}
	unlock, err := lockWriter(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
	if err := ensureWriteBranch(root, branch); err != nil {
		return nil, err
	}
	current, err := source.Load(root, "")
	if err != nil {
		return nil, err
	}
	if current.Digest() != p.Snapshot.Digest() {
		return nil, fmt.Errorf("source changed since capture")
	}
	// Validate the full plan before any source is replaced.
	for _, name := range names {
		dest, err := safeDestination(root, name)
		if err != nil {
			return nil, err
		}
		data, err := os.ReadFile(dest)
		if err != nil || !bytes.Equal(data, p.Snapshot.Files[name]) {
			return nil, fmt.Errorf("source changed since capture: %s", name)
		}
	}
	written := make([]string, 0, len(names))
	for _, name := range names {
		if err := ensureWriteBranch(root, branch); err != nil {
			return written, err
		}
		dest, err := safeDestination(root, name)
		if err != nil {
			return written, err
		}
		data, err := os.ReadFile(dest)
		if err != nil || !bytes.Equal(data, p.Snapshot.Files[name]) {
			return written, fmt.Errorf("source changed during formatting: %s", name)
		}
		if err := ensureWriteBranch(root, branch); err != nil {
			return written, err
		}
		if err = atomicWrite(dest, changed[name]); err != nil {
			return written, err
		}
		written = append(written, name)
		mode := os.FileMode(0644)
		if p.Snapshot.Modes[name] == "100755" {
			mode = 0755
		}
		if err = os.Chmod(dest, mode); err != nil {
			return written, err
		}
	}
	final, err := source.Load(root, "")
	if err != nil {
		return written, err
	}
	expected := &snapshot.Snapshot{Files: make(map[string][]byte, len(p.Snapshot.Files)), Modes: make(map[string]string, len(p.Snapshot.Modes))}
	for name, data := range p.Snapshot.Files {
		expected.Files[name] = data
		expected.Modes[name] = p.Snapshot.Modes[name]
	}
	for name, data := range changed {
		expected.Files[name] = data
	}
	if final.Digest() != expected.Digest() {
		return written, fmt.Errorf("source changed during formatting; reload and format the complete candidate")
	}
	if err := ensureWriteBranch(root, branch); err != nil {
		return written, err
	}
	return written, nil
}
