package app

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/format"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

// Format canonicalizes YAML and line endings, without changing its model.
func Format(root string, p *Project, write bool) ([]string, error) {
	if len(p.Diagnostics) > 0 {
		return nil, fmt.Errorf("resolve project diagnostics before formatting")
	}
	changed := map[string][]byte{}
	for _, resource := range p.Resources {
		copy := *resource
		copy.Spec.Text = strings.ReplaceAll(copy.Spec.Text, "\r\n", "\n")
		data, err := format.Encode(copy)
		if err != nil {
			return nil, err
		}
		if !bytes.Equal(normalize(data), normalize(p.Snapshot.Files[resource.Path])) {
			changed[resource.Path] = data
		}
	}
	names := sortedFiles(changed)
	if !write {
		return names, nil
	}
	if !p.Snapshot.Provisional {
		return nil, fmt.Errorf("format writes require an isolated working tree")
	}
	if err := writableBranch(root); err != nil {
		return nil, err
	}
	unlock, err := lockWriter(root)
	if err != nil {
		return nil, err
	}
	defer unlock()
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
		dest, err := safeDestination(root, name)
		if err != nil {
			return written, err
		}
		data, err := os.ReadFile(dest)
		if err != nil || !bytes.Equal(data, p.Snapshot.Files[name]) {
			return written, fmt.Errorf("source changed during formatting: %s", name)
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
	expected := &source.Snapshot{Files: make(map[string][]byte, len(p.Snapshot.Files)), Modes: make(map[string]string, len(p.Snapshot.Modes))}
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
	return written, nil
}
