package contentpackage

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"fmt"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/src/internal/host/authoring"
)

// Build selects the closed package file set from a source snapshot and writes
// a reproducible ZIP. Unrelated snapshot files are omitted.
func Build(files map[string][]byte) ([]byte, error) {
	manifestData, ok := files[ManifestName]
	if !ok {
		return nil, fmt.Errorf("source files are missing %s", ManifestName)
	}
	manifest, err := authoring.Parse(ManifestName, manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse package manifest: %w", err)
	}
	if manifest.Kind != "Package" {
		return nil, fmt.Errorf("%s must have kind Package", ManifestName)
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	registry, _, err := parseDomainDefinitions(manifest, files)
	if err != nil {
		return nil, err
	}
	selected := map[string][]byte{ManifestName: manifestData}
	var areaPaths []string
	for _, area := range manifest.Spec.Areas {
		areaPaths = append(areaPaths, area.Path)
	}
	for _, domainPath := range manifest.Spec.Domains {
		data, exists := files[domainPath]
		if !exists {
			return nil, fmt.Errorf("declared package domain %q is missing", domainPath)
		}
		selected[domainPath] = data
	}
	candidates := make(map[string][]byte)
	resources := make([]*authoring.Resource, 0)
	parseErrors := make(map[string]error)
	for _, filePath := range sortedFilePaths(files) {
		data := files[filePath]
		if filePath == ManifestName {
			continue
		}
		if _, declaredDomain := selected[filePath]; declaredDomain {
			continue
		}
		if isYAML(filePath) && inAnyArea(filePath, areaPaths) {
			if resource, err := authoring.ParseWithRegistry(filePath, data, registry); err == nil {
				selected[filePath] = data
				resources = append(resources, resource)
			} else {
				candidates[filePath] = data
				parseErrors[filePath] = err
			}
		}
	}
	ordinary := declaredInputs(resources)
	for _, filePath := range sortedFilePaths(candidates) {
		data := candidates[filePath]
		if ordinary[filePath] && isValidTextInput(data) && !authoring.IsResourceEnvelopeWithRegistry(data, registry) {
			selected[filePath] = data
		} else {
			return nil, fmt.Errorf("parse package resource %q: %w", filePath, parseErrors[filePath])
		}
	}
	for _, filePath := range sortedBoolKeys(ordinary) {
		data, ok := files[filePath]
		if !ok {
			return nil, fmt.Errorf("declared package input %q is missing", filePath)
		}
		if !isValidTextInput(data) {
			return nil, fmt.Errorf("declared package input %q is not UTF-8", filePath)
		}
		selected[filePath] = data
	}
	if _, err := parseFiles(manifest.Metadata.Name, manifest.Spec.Version, selected); err != nil {
		return nil, err
	}
	if err := validateMembers(selected); err != nil {
		return nil, err
	}

	paths := make([]string, 0, len(selected))
	for filePath := range selected {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	var out bytes.Buffer
	w := zip.NewWriter(&out)
	w.RegisterCompressor(zip.Deflate, func(out io.Writer) (io.WriteCloser, error) {
		return flate.NewWriter(out, flate.DefaultCompression)
	})
	for _, filePath := range paths {
		header := &zip.FileHeader{Name: filePath, Method: zip.Deflate}
		header.SetModTime(fixedZipTime)
		header.SetMode(0o644)
		member, err := w.CreateHeader(header)
		if err != nil {
			return nil, err
		}
		if _, err := member.Write(selected[filePath]); err != nil {
			return nil, err
		}
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	if out.Len() > maxCompressedBytes {
		return nil, fmt.Errorf("built package archive exceeds the 64 MiB compressed limit")
	}
	return out.Bytes(), nil
}

func validateMembers(files map[string][]byte) error {
	if len(files) == 0 || len(files) > maxArchiveFiles {
		return fmt.Errorf("package must contain between 1 and %d files", maxArchiveFiles)
	}
	seen := make(map[string]string, len(files))
	var total uint64
	for _, filePath := range sortedFilePaths(files) {
		data := files[filePath]
		if err := validatePath(filePath); err != nil {
			return fmt.Errorf("unsafe package path %q: %w", filePath, err)
		}
		folded := strings.ToLower(filePath)
		if previous, ok := seen[folded]; ok {
			return fmt.Errorf("package paths %q and %q collide by case", previous, filePath)
		}
		seen[folded] = filePath
		if len(data) > maxMemberBytes {
			return fmt.Errorf("package member %q exceeds the 8 MiB limit", filePath)
		}
		if total > maxExpandedBytes-uint64(len(data)) {
			return fmt.Errorf("package exceeds the 128 MiB expanded limit")
		}
		total += uint64(len(data))
	}
	return nil
}

func sortedFilePaths[V any](files map[string]V) []string {
	paths := make([]string, 0, len(files))
	for filePath := range files {
		paths = append(paths, filePath)
	}
	sort.Strings(paths)
	return paths
}

func sortedBoolKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key, value := range values {
		if value {
			keys = append(keys, key)
		}
	}
	sort.Strings(keys)
	return keys
}

func isValidTextInput(data []byte) bool {
	return utf8.Valid(data) && bytes.IndexByte(data, 0) < 0
}
