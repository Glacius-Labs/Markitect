package projectcli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"sort"
	"strings"
	"time"

	hostwrite "github.com/Glacius-Labs/Markitect/src/internal/host"
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

func writeRecord(root, rawPath string, data []byte) (string, error) {
	rel, err := recordPath(rawPath)
	if err != nil {
		return "", err
	}
	for attempt := 0; attempt < 100; attempt++ {
		capture, err := hostwrite.CaptureGuardedWrite(root, []string{rel})
		if err != nil {
			if isGuardedWriterBusy(err) && waitForGuardedWriter(attempt) {
				continue
			}
			return "", fmt.Errorf("capture Markitect record target %s: %w", rel, err)
		}
		if current, ok := capture.Files[rel]; !ok || current.Exists {
			return "", fmt.Errorf("refusing to overwrite existing Markitect record %s", rel)
		}
		_, err = hostwrite.ApplyGuardedWrite(capture.Root, capture, []hostwrite.GuardedWriteChange{{Path: rel, Bytes: data, Mode: 0644}})
		if err != nil {
			if isGuardedWriterBusy(err) && waitForGuardedWriter(attempt) {
				continue
			}
			return "", fmt.Errorf("write Markitect record %s: %w", rel, err)
		}
		digest := sha256.Sum256(data)
		return "sha256:" + hex.EncodeToString(digest[:]), nil
	}
	return "", fmt.Errorf("write Markitect record %s: guarded writer remained busy after bounded retries", rel)
}

func isGuardedWriterBusy(err error) bool {
	return err != nil && errors.Is(err, fs.ErrExist) && strings.HasPrefix(err.Error(), "renderer lock is already present:")
}

func waitForGuardedWriter(attempt int) bool {
	if attempt >= 99 {
		return false
	}
	time.Sleep(10 * time.Millisecond)
	return true
}

// preflightRecordDestinations confirms that all explicit output paths are
// valid, absent Markitect records before an agent invocation can incur cost.
func preflightRecordDestinations(root string, rawPaths ...string) error {
	paths := make([]string, 0, len(rawPaths))
	seen := map[string]bool{}
	for _, raw := range rawPaths {
		rel, err := recordPath(raw)
		if err != nil {
			return err
		}
		if seen[rel] {
			return fmt.Errorf("Markitect record destination %s was specified more than once", rel)
		}
		seen[rel] = true
		paths = append(paths, rel)
	}
	if len(paths) == 0 {
		return errors.New("at least one Markitect record destination is required")
	}
	sort.Strings(paths)
	capture, err := hostwrite.CaptureGuardedWrite(root, paths)
	if err != nil {
		return fmt.Errorf("capture Markitect record targets: %w", err)
	}
	for _, path := range paths {
		current, ok := capture.Files[path]
		if !ok {
			return fmt.Errorf("Markitect record destination %s was not captured", path)
		}
		if current.Exists {
			return fmt.Errorf("refusing to overwrite existing Markitect record %s", path)
		}
	}
	return nil
}

// writeRecords persists a related group of transport records under one
// guarded capture, refusing any overwrite before applying the group.
func writeRecords(root string, records map[string][]byte) (map[string]string, error) {
	paths := make([]string, 0, len(records))
	for raw := range records {
		rel, err := recordPath(raw)
		if err != nil {
			return nil, err
		}
		paths = append(paths, rel)
	}
	sort.Strings(paths)
	if len(paths) == 0 {
		return nil, errors.New("at least one Markitect record is required")
	}
	capture, err := hostwrite.CaptureGuardedWrite(root, paths)
	if err != nil {
		return nil, fmt.Errorf("capture Markitect record targets: %w", err)
	}
	changes := make([]hostwrite.GuardedWriteChange, 0, len(paths))
	digests := make(map[string]string, len(paths))
	for _, path := range paths {
		current, ok := capture.Files[path]
		if !ok || current.Exists {
			return nil, fmt.Errorf("refusing to overwrite existing Markitect record %s", path)
		}
		data := records[path]
		changes = append(changes, hostwrite.GuardedWriteChange{Path: path, Bytes: data, Mode: 0644})
		sum := sha256.Sum256(data)
		digests[path] = "sha256:" + hex.EncodeToString(sum[:])
	}
	if _, err := hostwrite.ApplyGuardedWrite(capture.Root, capture, changes); err != nil {
		return nil, fmt.Errorf("write Markitect records: %w", err)
	}
	return digests, nil
}

func emitRecord(root, output string, data []byte, stdout io.Writer) error {
	if output == "" {
		_, err := stdout.Write(data)
		return err
	}
	digest, err := writeRecord(root, output, data)
	if err != nil {
		return err
	}
	return writeJSON(stdout, struct {
		Path   string `json:"path"`
		Digest string `json:"digest"`
	}{output, digest})
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
