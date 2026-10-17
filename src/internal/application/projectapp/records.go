package projectapp

import (
	"errors"
	"fmt"
	"path"
	"regexp"
	"strings"

	"github.com/Glacius-Labs/Markitect/src/internal/infrastructure/recordstore"
	"github.com/Glacius-Labs/Markitect/internal/tooling/source"
)

func (o Operations) ReadRecord(root, rawPath string) ([]byte, error) {
	rel, err := RecordPath(rawPath)
	if err != nil {
		return nil, err
	}
	if strings.HasPrefix(rel, recordstore.RootPath+"/") {
		name, err := recordstore.Parse(rel)
		if err != nil {
			return nil, err
		}
		store, err := recordstore.Open(root)
		if err != nil {
			return nil, err
		}
		defer store.Close()
		file, err := store.Read(name, 0)
		if err != nil {
			return nil, err
		}
		if !file.Exists {
			return nil, fmt.Errorf("Markitect record %s does not exist", rel)
		}
		return append([]byte(nil), file.Bytes...), nil
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

func RecordPath(raw string) (string, error) {
	if raw == "" || strings.ContainsAny(raw, "\\:\x00") || strings.HasPrefix(raw, "/") || path.Clean(raw) != raw {
		return "", errors.New("record path must be a normalized repository-relative slash path under .markitect/drafts/ or .markitect/runs/")
	}
	if !strings.HasSuffix(raw, ".json") {
		return "", errors.New("Markitect transport records must use the .json extension")
	}
	if strings.HasPrefix(raw, recordstore.RootPath+"/") {
		name, err := recordstore.Parse(raw)
		if err != nil {
			return "", err
		}
		if name.Kind != recordstore.Run && !(name.Kind == recordstore.Brownfield && name.Role == "record.json") {
			return "", errors.New("record transport supports only Run evidence and Brownfield Session records")
		}
		return raw, nil
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

var fullCommitID = regexp.MustCompile(`^(?:[0-9a-f]{40}|[0-9a-f]{64})$`)

// ResolveRevision binds a caller's ref to one full commit ID before invoking a use case.
func (o Operations) ResolveRevision(root, revision string) (string, error) {
	out, err := source.GitOutput(root, "rev-parse", "--verify", "--quiet", "--end-of-options", revision+"^{commit}")
	full := strings.ToLower(strings.TrimSpace(string(out)))
	if err == nil && fullCommitID.MatchString(full) {
		return full, nil
	}
	return "", fmt.Errorf("%q does not name a commit in this repository", revision)
}
