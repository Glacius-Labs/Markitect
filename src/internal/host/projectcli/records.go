package projectcli

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/source"
)

func readRecord(root, rawPath string) ([]byte, error) {
	rel, err := recordPath(rawPath)
	if err != nil {
		return nil, err
	}
	selected, err := source.ObserveSelectedWorking(root, []string{rel})
	if err != nil {
		return nil, fmt.Errorf("read Markitect record %s: %w", rel, err)
	}
	if len(selected.MissingPaths) != 0 {
		return nil, fmt.Errorf("Markitect record %s does not exist", rel)
	}
	data, ok := selected.Snapshot.Files[rel]
	if !ok {
		return nil, fmt.Errorf("Markitect record %s was not returned by the exact-path read", rel)
	}
	return append([]byte(nil), data...), nil
}

func recordPath(raw string) (string, error) {
	if raw == "" || strings.ContainsAny(raw, "\\:\x00") || strings.HasPrefix(raw, "/") || path.Clean(raw) != raw {
		return "", errors.New("record path must be a normalized repository-relative slash path under .markitect/drafts/ or .markitect/runs/")
	}
	if !strings.HasSuffix(raw, ".json") {
		return "", errors.New("Markitect transport records must use the .json extension")
	}
	if !strings.HasPrefix(raw, ".markitect/drafts/") && !strings.HasPrefix(raw, ".markitect/runs/") {
		return "", errors.New("Markitect records may be read or written only under .markitect/drafts/ or .markitect/runs/")
	}
	for _, component := range strings.Split(raw, "/") {
		if component == "" || component == "." || component == ".." || strings.EqualFold(component, ".git") {
			return "", errors.New("record path contains an empty, traversal, or reserved component")
		}
	}
	return raw, nil
}
