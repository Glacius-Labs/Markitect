package projectcli

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"

	hostwrite "github.com/Glacius-Labs/Markitect/internal/host"
	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
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
	capture, err := hostwrite.CaptureGuardedWrite(root, []string{rel})
	if err != nil {
		return "", fmt.Errorf("capture Markitect record target %s: %w", rel, err)
	}
	if current, ok := capture.Files[rel]; !ok || current.Exists {
		return "", fmt.Errorf("refusing to overwrite existing Markitect record %s", rel)
	}
	_, err = hostwrite.ApplyGuardedWrite(capture.Root, capture, []hostwrite.GuardedWriteChange{{Path: rel, Bytes: data, Mode: 0644}})
	if err != nil {
		return "", fmt.Errorf("write Markitect record %s: %w", rel, err)
	}
	digest := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(digest[:]), nil
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
