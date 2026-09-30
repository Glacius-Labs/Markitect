package release

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"io"
	"path"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/source"
	"go.yaml.in/yaml/v3"
)

const (
	releaseManifestPath = "tools/markitect/release.yaml"
	sourceRepository    = "github.com/Glacius-Labs/Markitect"
	maxBundleBytes      = 64 << 20
	maxBundleFileBytes  = 64 << 20
	maxManifestBytes    = 64 << 10
)

var bundlePaths = []string{
	"markitect.lock.yaml",
	"scripts/markitect-bootstrap_test.go",
	"scripts/run-markitect.go",
	"tools/markitect/source.zip",
}

// BundleManifest identifies a complete, pinned Markitect distribution.
type BundleManifest struct {
	SchemaVersion    int          `yaml:"schemaVersion"`
	Version          string       `yaml:"version"`
	SourceCommit     string       `yaml:"sourceCommit"`
	SourceRepository string       `yaml:"sourceRepository"`
	Files            []BundleFile `yaml:"files"`
}

// BundleFile binds one distribution file to its SHA-256 digest.
type BundleFile struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
}

// Bundle is a validated source distribution. Files contains all five pinned
// bundle files, including release.yaml.
type Bundle struct {
	Manifest BundleManifest
	Files    map[string][]byte
	SHA256   string
}

// BuildBundle creates the deterministic five-file release bundle from one
// immutable Git snapshot. Provisional snapshots and incomplete commit IDs are
// rejected so generated metadata cannot be mistaken for a source release.
// The outer archive is limited to 64 MiB.
func BuildBundle(snapshot *source.Snapshot, version string) ([]byte, error) {
	if snapshot == nil || snapshot.Provisional || !validCommit(snapshot.Revision) {
		return nil, errors.New("release bundle requires a fixed snapshot with a full lowercase source commit")
	}
	if !validVersion(version) || strings.HasPrefix(version, "v") {
		return nil, fmt.Errorf("invalid semantic version %q", version)
	}
	if err := validateSourceVersion(snapshot, version); err != nil {
		return nil, err
	}
	moduleFiles, err := snapshotModuleFiles(snapshot)
	if err != nil {
		return nil, err
	}
	archiveFiles, err := collectModuleSnapshot(moduleFiles)
	if err != nil {
		return nil, err
	}
	sourceZip, err := makeArchive(archiveFiles)
	if err != nil {
		return nil, err
	}
	sourceDigest := digestBytes(sourceZip)
	lock := []byte("version: " + strconv.Quote(version) + "\n" +
		"source: " + strconv.Quote(sourcePath) + "\n" +
		"sha256: " + strconv.Quote(sourceDigest) + "\n")

	runner, err := snapshotText(snapshot, "integration/run-markitect.go")
	if err != nil {
		return nil, err
	}
	test, err := snapshotText(snapshot, "integration/run-markitect_test.go")
	if err != nil {
		return nil, err
	}
	files := map[string][]byte{
		"markitect.lock.yaml":                 lock,
		"scripts/run-markitect.go":            runner,
		"scripts/markitect-bootstrap_test.go": test,
		"tools/markitect/source.zip":          sourceZip,
	}
	manifest := BundleManifest{
		SchemaVersion:    1,
		Version:          version,
		SourceCommit:     snapshot.Revision,
		SourceRepository: sourceRepository,
	}
	for _, name := range bundlePaths {
		manifest.Files = append(manifest.Files, BundleFile{Path: name, SHA256: digestBytes(files[name])})
	}
	manifestData, err := yaml.Marshal(manifest)
	if err != nil {
		return nil, fmt.Errorf("encode release manifest: %w", err)
	}
	files[releaseManifestPath] = manifestData
	return makeBundleArchive(files)
}

// ParseBundle verifies the outer digest and every structural/content binding
// before returning a release bundle. expectedSHA256 may be either 64 lowercase
// hexadecimal characters or the same value prefixed with "sha256:".
// The compressed bundle and its total expanded contents are limited to 64 MiB.
func ParseBundle(data []byte, expectedSHA256 string) (*Bundle, error) {
	expectedSHA256 = strings.TrimPrefix(expectedSHA256, "sha256:")
	if !validDigest(expectedSHA256) {
		return nil, errors.New("expected bundle SHA-256 must be 64 lowercase hexadecimal characters")
	}
	if len(data) == 0 || len(data) > maxBundleBytes {
		return nil, fmt.Errorf("release bundle must be between 1 byte and %d bytes", maxBundleBytes)
	}
	actual := digestBytes(data)
	if actual != expectedSHA256 {
		return nil, errors.New("release bundle SHA-256 does not match expected digest")
	}
	reader, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("open release bundle: %w", err)
	}
	if len(reader.File) != len(bundlePaths)+1 {
		return nil, fmt.Errorf("release bundle must contain exactly %d files", len(bundlePaths)+1)
	}
	want := make(map[string]bool, len(bundlePaths)+1)
	for _, name := range bundlePaths {
		want[name] = true
	}
	want[releaseManifestPath] = true
	files := make(map[string][]byte, len(reader.File))
	var total int64
	for _, entry := range reader.File {
		name := entry.Name
		if err := safeArchivePath(name); err != nil {
			return nil, fmt.Errorf("unsafe release bundle path %q: %w", name, err)
		}
		if !want[name] {
			return nil, fmt.Errorf("unexpected release bundle file %q", name)
		}
		if _, exists := files[name]; exists {
			return nil, fmt.Errorf("duplicate release bundle file %q", name)
		}
		want[name] = false
		mode := entry.Mode()
		if entry.FileInfo().IsDir() || !mode.IsRegular() {
			return nil, fmt.Errorf("release bundle entry %q is not a regular file", name)
		}
		if entry.UncompressedSize64 > maxBundleFileBytes {
			return nil, fmt.Errorf("release bundle file %q exceeds size limit", name)
		}
		if name == releaseManifestPath && entry.UncompressedSize64 > maxManifestBytes {
			return nil, errors.New("release manifest exceeds size limit")
		}
		if entry.UncompressedSize64 > uint64(maxBundleBytes-total) {
			return nil, errors.New("release bundle expands beyond total size limit")
		}
		file, err := entry.Open()
		if err != nil {
			return nil, fmt.Errorf("open release bundle file %q: %w", name, err)
		}
		content, readErr := io.ReadAll(io.LimitReader(file, maxBundleFileBytes+1))
		closeErr := file.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read release bundle file %q: %w", name, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close release bundle file %q: %w", name, closeErr)
		}
		if int64(len(content)) != int64(entry.UncompressedSize64) || int64(len(content)) > maxBundleFileBytes {
			return nil, fmt.Errorf("release bundle file %q has invalid expanded size", name)
		}
		total += int64(len(content))
		files[name] = content
	}
	for name, missing := range want {
		if missing {
			return nil, fmt.Errorf("release bundle is missing %q", name)
		}
	}
	manifest, err := ParseBundleManifest(files[releaseManifestPath])
	if err != nil {
		return nil, err
	}
	if err := ValidateBundleFiles(manifest, files, manifest.SourceCommit); err != nil {
		return nil, err
	}
	return &Bundle{Manifest: manifest, Files: files, SHA256: actual}, nil
}

func snapshotModuleFiles(snapshot *source.Snapshot) (map[string][]byte, error) {
	prefix := ""
	if _, ok := snapshot.Files["go.mod"]; !ok {
		prefix = "tools/markitect/"
		if _, exists := snapshot.Files[prefix+"go.mod"]; !exists {
			return nil, errors.New("fixed source snapshot is missing go.mod")
		}
	}
	module := make(map[string][]byte)
	for name, data := range snapshot.Files {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		relative := strings.TrimPrefix(name, prefix)
		if relative == "" {
			continue
		}
		if err := safeArchivePath(relative); err != nil {
			return nil, fmt.Errorf("unsafe source snapshot path %q: %w", name, err)
		}
		include := relative == "go.mod" || relative == "go.sum" || relative == "README.md"
		if strings.HasPrefix(relative, "cmd/") || strings.HasPrefix(relative, "internal/") {
			ext := strings.ToLower(path.Ext(relative))
			include = ext == ".go" || (strings.HasPrefix(relative, "internal/authoring/resources/") && (ext == ".yaml" || ext == ".yml"))
		}
		if strings.HasPrefix(relative, "schema/") {
			ext := strings.ToLower(path.Ext(relative))
			include = ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".md"
		}
		if include {
			if !snapshotRegularFile(snapshot, name) {
				return nil, fmt.Errorf("source snapshot file %q is not regular", name)
			}
			module[relative] = data
		}
	}
	return module, nil
}

func collectModuleSnapshot(files map[string][]byte) ([]sourceFile, error) {
	selected := make([]sourceFile, 0, len(files))
	for _, name := range []string{"go.mod", "go.sum"} {
		data, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("fixed source snapshot is missing %s", name)
		}
		normalized, err := normalizeTextSource(name, data)
		if err != nil {
			return nil, fmt.Errorf("invalid source snapshot text %s: %w", name, err)
		}
		selected = append(selected, sourceFile{name: name, data: normalized})
	}
	for name, data := range files {
		if name == "go.mod" || name == "go.sum" {
			continue
		}
		ext := strings.ToLower(path.Ext(name))
		include := name == "README.md"
		if strings.HasPrefix(name, "cmd/") || strings.HasPrefix(name, "internal/") {
			include = ext == ".go" || (strings.HasPrefix(name, "internal/authoring/resources/") && (ext == ".yaml" || ext == ".yml"))
		}
		if strings.HasPrefix(name, "schema/") {
			include = ext == ".json" || ext == ".yaml" || ext == ".yml" || ext == ".md"
		}
		if !include {
			continue
		}
		normalized, err := normalizeTextSource(name, data)
		if err != nil {
			return nil, fmt.Errorf("invalid source snapshot text %s: %w", name, err)
		}
		selected = append(selected, sourceFile{name: name, data: normalized})
	}
	sort.Slice(selected, func(i, j int) bool { return selected[i].name < selected[j].name })
	if len(selected) < 4 {
		return nil, errors.New("fixed source snapshot Markitect module is incomplete")
	}
	if err := validateArchivePaths(selected); err != nil {
		return nil, err
	}
	if err := validateSnapshotModuleLayout(selected); err != nil {
		return nil, err
	}
	return selected, nil
}

func validateSnapshotModuleLayout(files []sourceFile) error {
	present := make(map[string]bool, len(files))
	for _, file := range files {
		present[file.name] = true
	}
	for _, name := range []string{"go.mod", "go.sum", "cmd/markitect/main.go"} {
		if !present[name] {
			return fmt.Errorf("fixed source snapshot is missing required module file %s", name)
		}
	}
	internalFound := false
	for name := range present {
		if strings.HasPrefix(name, "internal/") {
			internalFound = true
			break
		}
	}
	if !internalFound {
		return errors.New("fixed source snapshot is missing the internal source tree")
	}
	var goMod []byte
	for _, file := range files {
		if file.name == "go.mod" {
			goMod = file.data
			break
		}
	}
	for _, line := range strings.Split(string(goMod), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if !strings.HasPrefix(line, "module ") {
			break
		}
		module := strings.TrimSpace(strings.TrimPrefix(line, "module "))
		if unquoted, err := strconv.Unquote(module); err == nil {
			module = unquoted
		}
		if module == sourceRepository {
			return nil
		}
		break
	}
	return fmt.Errorf("fixed source snapshot go.mod must declare module %s", sourceRepository)
}

func snapshotText(snapshot *source.Snapshot, name string) ([]byte, error) {
	data, ok := snapshot.Files[name]
	if !ok {
		return nil, fmt.Errorf("fixed source snapshot is missing %s", name)
	}
	if !snapshotRegularFile(snapshot, name) {
		return nil, fmt.Errorf("source snapshot file %q is not regular", name)
	}
	return normalizeTextSource(name, data)
}

func snapshotRegularFile(snapshot *source.Snapshot, name string) bool {
	if snapshot == nil || snapshot.Modes == nil {
		return false
	}
	mode := snapshot.Modes[name]
	return mode == "100644" || mode == "100755"
}

func validateSourceVersion(snapshot *source.Snapshot, version string) error {
	data, err := snapshotText(snapshot, "cmd/markitect/main.go")
	if err != nil {
		return err
	}
	file, err := parser.ParseFile(token.NewFileSet(), "cmd/markitect/main.go", data, parser.AllErrors)
	if err != nil {
		return fmt.Errorf("parse fixed source version declaration: %w", err)
	}
	found := false
	for _, declaration := range file.Decls {
		group, ok := declaration.(*ast.GenDecl)
		if !ok || group.Tok != token.VAR {
			continue
		}
		for _, spec := range group.Specs {
			valueSpec, ok := spec.(*ast.ValueSpec)
			if !ok {
				continue
			}
			for index, name := range valueSpec.Names {
				if name.Name != "version" {
					continue
				}
				if found || len(valueSpec.Names) != 1 || len(valueSpec.Values) != 1 || index != 0 {
					return errors.New("source version must be declared once as a string literal var version")
				}
				literal, ok := valueSpec.Values[0].(*ast.BasicLit)
				if !ok || literal.Kind != token.STRING {
					return errors.New("source version must be a string literal var version")
				}
				declared, err := strconv.Unquote(literal.Value)
				if err != nil || declared != version {
					return fmt.Errorf("bundle version %q does not match source version declaration", version)
				}
				found = true
			}
		}
	}
	if !found {
		return errors.New("fixed source snapshot is missing its var version declaration")
	}
	return nil
}

func makeBundleArchive(files map[string][]byte) ([]byte, error) {
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := []string{releaseManifestPath}
	names = append(names, bundlePaths...)
	sort.Strings(names)
	fixedTime := time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)
	for _, name := range names {
		data, ok := files[name]
		if !ok {
			return nil, fmt.Errorf("internal release bundle is missing %s", name)
		}
		header := &zip.FileHeader{Name: name, Method: zip.Store}
		header.SetModTime(fixedTime)
		header.SetMode(0644)
		entry, err := writer.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := entry.Write(data); err != nil {
			return nil, err
		}
	}
	if err := writer.Close(); err != nil {
		return nil, err
	}
	data := buffer.Bytes()
	if len(data) > maxBundleBytes {
		return nil, fmt.Errorf("release bundle exceeds %d bytes", maxBundleBytes)
	}
	return data, nil
}

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
	if manifest.SchemaVersion != 1 || !validVersion(manifest.Version) || strings.HasPrefix(manifest.Version, "v") || !validCommit(manifest.SourceCommit) || manifest.SourceRepository != sourceRepository {
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

// ToolLock is the backwards-compatible three-field tool distribution lock.
type ToolLock struct {
	Version string
	Source  string
	SHA256  string
}

// ParseToolLock strictly parses the existing flat Markitect tool lock.
func ParseToolLock(data []byte) (ToolLock, error) {
	values, err := parseSimpleFlatYAML(data, []string{"version", "source", "sha256"}, "markitect.lock.yaml")
	if err != nil {
		return ToolLock{}, err
	}
	if !validVersion(values["version"]) || values["source"] != sourcePath || !validDigest(values["sha256"]) {
		return ToolLock{}, errors.New("release lock version, source path, or SHA-256 is invalid")
	}
	return ToolLock{Version: values["version"], Source: values["source"], SHA256: values["sha256"]}, nil
}

// ValidateToolLockFiles confirms that a strict flat tool lock describes the
// source archive in files. It is useful when reading legacy installations that
// predate release.yaml.
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
// legacy flat lock against the source archive. The files map must contain the
// exact four bundle paths.
func ValidateBundleFiles(manifest BundleManifest, files map[string][]byte, expectedSourceCommit string) error {
	if manifest.SchemaVersion != 1 || !validVersion(manifest.Version) || strings.HasPrefix(manifest.Version, "v") || !validCommit(manifest.SourceCommit) || manifest.SourceRepository != sourceRepository {
		return errors.New("release manifest schema, version, source commit, or source repository is invalid")
	}
	if expectedSourceCommit != "" && manifest.SourceCommit != expectedSourceCommit {
		return errors.New("installed release manifest does not match the expected source commit")
	}
	if len(manifest.Files) != len(bundlePaths) {
		return errors.New("installed release must contain exactly four distribution files")
	}
	if len(files) == len(bundlePaths)+1 {
		manifestBytes, ok := files[releaseManifestPath]
		if !ok {
			return errors.New("installed release has an unexpected extra file")
		}
		parsed, err := ParseBundleManifest(manifestBytes)
		if err != nil || !sameManifest(manifest, parsed) {
			return errors.New("installed release manifest bytes do not match the parsed manifest")
		}
	} else if len(files) != len(bundlePaths) {
		return errors.New("installed release must contain exactly four distribution files and optionally release.yaml")
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
	lock, err := ParseToolLock(files["markitect.lock.yaml"])
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
