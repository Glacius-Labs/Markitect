package app

import (
	"bytes"
	"fmt"
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
	current, err := source.Load(root, "")
	if err != nil {
		return nil, err
	}
	if current.Digest() != p.Snapshot.Digest() {
		return nil, fmt.Errorf("source changed since capture")
	}
	for _, name := range names {
		dest, err := safeDestination(root, name)
		if err != nil {
			return nil, err
		}
		if err = atomicWrite(dest, changed[name]); err != nil {
			return nil, err
		}
	}
	return names, nil
}
