package release

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Glacius-Labs/Markitect/internal/core/snapshot"
	"go.yaml.in/yaml/v3"
)

const (
	toolLockPath        = ".markitect/tool/lock.yaml"
	releaseManifestPath = ".markitect/tool/release.yaml"
	bootstrapPath       = ".markitect/bootstrap/run.go"
	bootstrapTestPath   = ".markitect/bootstrap/run_test.go"
	sourceRepository    = "github.com/Glacius-Labs/Markitect"
	maxBundleBytes      = 64 << 20
	maxBundleFileBytes  = 64 << 20
	maxManifestBytes    = 64 << 10
)

var bundlePaths = []string{
	bootstrapPath,
	bootstrapTestPath,
	toolLockPath,
	".markitect/tool/source.zip",
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
func BuildBundle(snapshot *snapshot.Snapshot, version string) ([]byte, error) {
	if snapshot == nil || snapshot.Provisional || !validCommit(snapshot.ID) {
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
		toolLockPath:                lock,
		bootstrapPath:               runner,
		bootstrapTestPath:           test,
		".markitect/tool/source.zip": sourceZip,
	}
	manifest := BundleManifest{
		SchemaVersion:    2,
		Version:          version,
		SourceCommit:     snapshot.ID,
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
