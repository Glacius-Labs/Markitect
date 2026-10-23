package release

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"

	"go.yaml.in/yaml/v3"
)

// ParseBundleManifest validates the strict YAML release manifest and its
// schema-level fields. File bytes and source-commit equality are checked by
// ValidateBundleFiles.
func ParseBundleManifest(data []byte) (BundleManifest, error) {
	if len(data) > maxManifestBytes || !utf8.Valid(data) {
		return BundleManifest{}, errors.New("release manifest is oversized or not UTF-8")
	}
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var node yaml.Node
	if err := decoder.Decode(&node); err != nil {
		return BundleManifest{}, fmt.Errorf("decode release manifest: %w", err)
	}
	if err := inspectYAMLNode(&node); err != nil {
		return BundleManifest{}, err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return BundleManifest{}, errors.New("release manifest must contain one YAML document")
		}
		return BundleManifest{}, fmt.Errorf("decode trailing release manifest: %w", err)
	}
	decoder = yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	var manifest BundleManifest
	if err := decoder.Decode(&manifest); err != nil {
		return BundleManifest{}, fmt.Errorf("decode release manifest fields: %w", err)
	}
	if manifest.SchemaVersion != 2 || !validVersion(manifest.Version) || strings.HasPrefix(manifest.Version, "v") || !validCommit(manifest.SourceCommit) || manifest.SourceRepository != sourceRepository {
		return BundleManifest{}, errors.New("release manifest schema, version, source commit, or source repository is invalid")
	}
	if len(manifest.Files) != len(bundlePaths) {
		return BundleManifest{}, errors.New("release manifest must bind exactly four distribution files")
	}
	for i, name := range bundlePaths {
		entry := manifest.Files[i]
		if entry.Path != name || !validDigest(entry.SHA256) {
			return BundleManifest{}, fmt.Errorf("release manifest file entry %d is invalid or not in canonical order", i)
		}
	}
	return manifest, nil
}

func inspectYAMLNode(node *yaml.Node) error {
	if node.Anchor != "" || node.Kind == yaml.AliasNode {
		return errors.New("release manifest anchors and aliases are not allowed")
	}
	if node.Tag != "" && node.Tag != "!!map" && node.Tag != "!!seq" && node.Tag != "!!str" && node.Tag != "!!int" {
		return fmt.Errorf("release manifest contains unsupported YAML tag %s", node.Tag)
	}
	if node.Kind == yaml.MappingNode {
		seen := map[string]bool{}
		for i := 0; i < len(node.Content); i += 2 {
			key := node.Content[i]
			if key.Kind != yaml.ScalarNode || key.Tag != "!!str" || key.Value == "<<" {
				return errors.New("release manifest mapping keys must be plain strings")
			}
			if seen[key.Value] {
				return fmt.Errorf("release manifest contains duplicate key %q", key.Value)
			}
			seen[key.Value] = true
		}
	}
	for _, child := range node.Content {
		if err := inspectYAMLNode(child); err != nil {
			return err
		}
	}
	return nil
}

// ToolLock is the three-field tool distribution lock.
type ToolLock struct {
	Version string
	Source  string
	SHA256  string
}

// ParseToolLock strictly parses the Markitect tool lock.
func ParseToolLock(data []byte) (ToolLock, error) {
	values, err := parseSimpleFlatYAML(data, []string{"version", "source", "sha256"}, toolLockPath)
	if err != nil {
		return ToolLock{}, err
	}
	if !validVersion(values["version"]) || values["source"] != sourcePath || !validDigest(values["sha256"]) {
		return ToolLock{}, errors.New("release lock version, source path, or SHA-256 is invalid")
	}
	return ToolLock{Version: values["version"], Source: values["source"], SHA256: values["sha256"]}, nil
}

// ValidateToolLockFiles confirms that a strict tool lock describes the source
// archive in files.
func ValidateToolLockFiles(lock ToolLock, files map[string][]byte) error {
	archive, ok := files[sourcePath]
	if !ok {
		return fmt.Errorf("installed files are missing %s", sourcePath)
	}
	if !validVersion(lock.Version) || lock.Source != sourcePath || lock.SHA256 != digestBytes(archive) {
		return errors.New("installed tool lock does not match its source archive")
	}
	return nil
}

// ValidateBundleFiles checks all four distribution bytes against a parsed
// manifest, confirms the expected installed source commit, and validates the
// tool lock against the source archive. The files map must contain the
// exact five consumer files, including the release manifest.
func ValidateBundleFiles(manifest BundleManifest, files map[string][]byte, expectedSourceCommit string) error {
	if manifest.SchemaVersion != 2 || !validVersion(manifest.Version) || strings.HasPrefix(manifest.Version, "v") || !validCommit(manifest.SourceCommit) || manifest.SourceRepository != sourceRepository {
		return errors.New("release manifest schema, version, source commit, or source repository is invalid")
	}
	if expectedSourceCommit != "" && manifest.SourceCommit != expectedSourceCommit {
		return errors.New("installed release manifest does not match the expected source commit")
	}
	if len(manifest.Files) != len(bundlePaths) {
		return errors.New("release manifest must bind exactly four distribution files")
	}
	if len(files) != len(bundlePaths)+1 {
		return errors.New("installed release must contain exactly five consumer files")
	}
	manifestBytes, ok := files[releaseManifestPath]
	if !ok {
		return errors.New("installed release is missing its release manifest")
	}
	parsed, err := ParseBundleManifest(manifestBytes)
	if err != nil || !sameManifest(manifest, parsed) {
		return errors.New("installed release manifest bytes do not match the parsed manifest")
	}
	for i, name := range bundlePaths {
		entry := manifest.Files[i]
		data, exists := files[name]
		if !exists || entry.Path != name || !validDigest(entry.SHA256) {
			return fmt.Errorf("installed release file entry %d is missing or invalid", i)
		}
		if digestBytes(data) != entry.SHA256 {
			return fmt.Errorf("installed release file %q does not match manifest SHA-256", name)
		}
	}
	lock, err := ParseToolLock(files[toolLockPath])
	if err != nil {
		return err
	}
	if lock.Version != manifest.Version {
		return errors.New("installed release lock version does not match release manifest")
	}
	return ValidateToolLockFiles(lock, files)
}

func sameManifest(left, right BundleManifest) bool {
	if left.SchemaVersion != right.SchemaVersion || left.Version != right.Version || left.SourceCommit != right.SourceCommit || left.SourceRepository != right.SourceRepository || len(left.Files) != len(right.Files) {
		return false
	}
	for i := range left.Files {
		if left.Files[i] != right.Files[i] {
			return false
		}
	}
	return true
}

func parseSimpleFlatYAML(data []byte, fields []string, name string) (map[string]string, error) {
	if !utf8.Valid(data) {
		return nil, fmt.Errorf("%s must be UTF-8", name)
	}
	text := strings.ReplaceAll(string(data), "\r\n", "\n")
	if strings.ContainsRune(text, '\r') {
		return nil, fmt.Errorf("%s must not contain lone CR line endings", name)
	}
	text = strings.TrimSuffix(text, "\n")
	lines := strings.Split(text, "\n")
	if len(lines) != len(fields) {
		return nil, fmt.Errorf("%s must contain exactly %d fields", name, len(fields))
	}
	values := make(map[string]string, len(fields))
	for i, line := range lines {
		key, scalar, ok := strings.Cut(line, ": ")
		if !ok || key != fields[i] || len(scalar) < 2 || scalar[0] != '"' || scalar[len(scalar)-1] != '"' {
			return nil, fmt.Errorf("%s must contain canonical fields in order", name)
		}
		if _, ok := values[key]; ok {
			return nil, fmt.Errorf("%s contains duplicate field %q", name, key)
		}
		var value string
		if err := yaml.Unmarshal([]byte(scalar), &value); err != nil {
			return nil, fmt.Errorf("%s field %q is invalid: %w", name, key, err)
		}
		if value == "" {
			return nil, fmt.Errorf("%s field %q must not be empty", name, key)
		}
		values[key] = value
	}
	return values, nil
}

func validCommit(commit string) bool {
	if len(commit) != 40 {
		return false
	}
	_, err := hex.DecodeString(commit)
	return err == nil && strings.ToLower(commit) == commit
}

func validDigest(digest string) bool {
	if len(digest) != 64 {
		return false
	}
	_, err := hex.DecodeString(digest)
	return err == nil && strings.ToLower(digest) == digest
}

func digestBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
