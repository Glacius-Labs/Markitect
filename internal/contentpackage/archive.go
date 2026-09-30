// Package contentpackage reads and builds deterministic offline Markitect
// content archives. It has no network or filesystem access.
package contentpackage

import (
	"archive/zip"
	"bytes"
	"compress/flate"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"path"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/Glacius-Labs/Markitect/internal/core"
	"github.com/Glacius-Labs/Markitect/internal/format"
)

const (
	ManifestName       = "markitect-package.yaml"
	maxCompressedBytes = 64 << 20
	maxExpandedBytes   = 128 << 20
	maxMemberBytes     = 8 << 20
	maxArchiveFiles    = 10_000
)

var fixedZipTime = time.Date(1980, time.January, 1, 0, 0, 0, 0, time.UTC)

// Archive is a verified, in-memory package. Files contains every member,
// indexed by its unchanged archive-relative POSIX path. Resources excludes
// Manifest and has runtime package origin set to the pinned package name.
type Archive struct {
	Manifest  *core.Resource
	Files     map[string][]byte
	Resources []*core.Resource
}

// Read validates the digest and ZIP structure before parsing package content.
// All data remains in memory; member paths are never extracted to disk.
func Read(pin core.PackagePin, archive []byte) (*Archive, error) {
	if len(archive) > maxCompressedBytes {
		return nil, fmt.Errorf("package archive exceeds the 64 MiB compressed limit")
	}
	if len(pin.SHA256) != 64 || strings.ToLower(pin.SHA256) != pin.SHA256 {
		return nil, fmt.Errorf("package pin sha256 must be 64 lowercase hexadecimal characters")
	}
	want, err := hex.DecodeString(pin.SHA256)
	if err != nil || len(want) != sha256.Size {
		return nil, fmt.Errorf("package pin sha256 is invalid")
	}
	digest := sha256.Sum256(archive)
	if !bytes.Equal(digest[:], want) {
		return nil, fmt.Errorf("package archive sha256 does not match pin")
	}
	r, err := zip.NewReader(bytes.NewReader(archive), int64(len(archive)))
	if err != nil {
		return nil, fmt.Errorf("invalid package ZIP: %w", err)
	}
	if len(r.File) == 0 || len(r.File) > maxArchiveFiles {
		return nil, fmt.Errorf("package archive must contain between 1 and %d files", maxArchiveFiles)
	}

	files := make(map[string][]byte, len(r.File))
	folded := make(map[string]string, len(r.File))
	var declaredTotal uint64
	var actualTotal uint64
	for _, entry := range r.File {
		name := entry.Name
		if err := validatePath(name); err != nil {
			return nil, fmt.Errorf("unsafe package member %q: %w", name, err)
		}
		if entry.FileInfo().IsDir() || !entry.Mode().IsRegular() {
			return nil, fmt.Errorf("package member %q is not a regular file", name)
		}
		if _, ok := files[name]; ok {
			return nil, fmt.Errorf("duplicate package member %q", name)
		}
		key := strings.ToLower(name)
		if previous, ok := folded[key]; ok {
			return nil, fmt.Errorf("package members %q and %q collide by case", previous, name)
		}
		folded[key] = name
		if entry.UncompressedSize64 > maxMemberBytes {
			return nil, fmt.Errorf("package member %q exceeds the 8 MiB limit", name)
		}
		if entry.CompressedSize64 > maxCompressedBytes {
			return nil, fmt.Errorf("package member %q exceeds the compressed size limit", name)
		}
		if declaredTotal > maxExpandedBytes-entry.UncompressedSize64 {
			return nil, fmt.Errorf("package exceeds the 128 MiB expanded limit")
		}
		declaredTotal += entry.UncompressedSize64

		member, err := readMember(entry)
		if err != nil {
			return nil, fmt.Errorf("read package member %q: %w", name, err)
		}
		if actualTotal > maxExpandedBytes-uint64(len(member)) {
			return nil, fmt.Errorf("package exceeds the 128 MiB expanded limit")
		}
		actualTotal += uint64(len(member))
		files[name] = member
	}
	return parseFiles(pin.Name, pin.Version, files)
}

func readMember(entry *zip.File) ([]byte, error) {
	r, err := entry.Open()
	if err != nil {
		return nil, err
	}
	defer r.Close()
	data, err := io.ReadAll(io.LimitReader(r, maxMemberBytes+1))
	if err != nil {
		return nil, err
	}
	if len(data) > maxMemberBytes {
		return nil, fmt.Errorf("actual member size exceeds the 8 MiB limit")
	}
	return data, nil
}

func parseFiles(name, version string, files map[string][]byte) (*Archive, error) {
	manifestData, ok := files[ManifestName]
	if !ok {
		return nil, fmt.Errorf("package archive is missing %s", ManifestName)
	}
	manifest, err := format.Parse(ManifestName, manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse package manifest: %w", err)
	}
	if manifest.Kind != "Package" {
		return nil, fmt.Errorf("%s must have kind Package", ManifestName)
	}
	if manifest.Metadata.Name != name {
		return nil, fmt.Errorf("package manifest name %q does not match pin %q", manifest.Metadata.Name, name)
	}
	manifest.Package = name
	if manifest.Spec.Version != version {
		return nil, fmt.Errorf("package manifest version %q does not match pin %q", manifest.Spec.Version, version)
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}

	areaPaths := make([]string, 0, len(manifest.Spec.Areas))
	for _, area := range manifest.Spec.Areas {
		areaPaths = append(areaPaths, area.Path)
	}
	resources := make([]*core.Resource, 0)
	parseErrors := make(map[string]error)
	identities := make(map[string]string)
	accepted := map[string]bool{ManifestName: true}
	for _, filePath := range sortedFilePaths(files) {
		data := files[filePath]
		if filePath == ManifestName {
			continue
		}
		if !isYAML(filePath) || !inAnyArea(filePath, areaPaths) {
			continue
		}
		resource, parseErr := format.Parse(filePath, data)
		if parseErr != nil {
			parseErrors[filePath] = parseErr
			continue
		}
		if resource.Kind == "Project" || resource.Kind == "Package" {
			return nil, fmt.Errorf("nested %s resource %q is not allowed in a package", resource.Kind, filePath)
		}
		if hasPackageRefs(resource) {
			return nil, fmt.Errorf("package resource %q contains a transitive package reference", filePath)
		}
		resource.Package = name
		key := resource.GraphKey()
		if prior, exists := identities[key]; exists {
			return nil, fmt.Errorf("package resources %q and %q have duplicate identity %s", prior, filePath, key)
		}
		identities[key] = filePath
		resources = append(resources, resource)
		accepted[filePath] = true
	}
	ordinary := declaredInputs(resources)
	for _, filePath := range sortedFilePaths(parseErrors) {
		parseErr := parseErrors[filePath]
		if ordinary[filePath] && isValidTextInput(files[filePath]) && !format.IsResourceEnvelope(files[filePath]) {
			accepted[filePath] = true
			continue
		}
		return nil, fmt.Errorf("parse package resource %q: %w", filePath, parseErr)
	}
	for _, filePath := range sortedBoolKeys(ordinary) {
		data, exists := files[filePath]
		if !exists {
			return nil, fmt.Errorf("declared package input %q is missing", filePath)
		}
		if !isValidTextInput(data) {
			return nil, fmt.Errorf("declared package input %q is not UTF-8", filePath)
		}
		accepted[filePath] = true
	}
	if len(resources) == 0 {
		return nil, fmt.Errorf("package must contain at least one content resource")
	}
	resourceKeys := make(map[string]bool, len(resources))
	for _, resource := range resources {
		resourceKeys[resource.Key()] = true
	}
	seenExports := make(map[string]bool, len(manifest.Spec.Exports))
	for _, export := range manifest.Spec.Exports {
		if export.Package != "" || export.Kind == "" || export.Namespace == "" || export.Name == "" {
			return nil, fmt.Errorf("package exports must be fully qualified local refs")
		}
		key := export.Key("", "")
		if !resourceKeys[key] {
			return nil, fmt.Errorf("package export %s does not identify a package resource", key)
		}
		if seenExports[key] {
			return nil, fmt.Errorf("package export %s is duplicated", key)
		}
		seenExports[key] = true
	}
	for _, filePath := range sortedFilePaths(files) {
		if !accepted[filePath] {
			return nil, fmt.Errorf("package member %q is not a manifest, area resource, or declared input", filePath)
		}
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Path < resources[j].Path })
	return &Archive{Manifest: manifest, Files: files, Resources: resources}, nil
}

// Build selects the closed package file set from a source snapshot and writes
// a reproducible ZIP. Unrelated snapshot files are omitted.
func Build(files map[string][]byte) ([]byte, error) {
	manifestData, ok := files[ManifestName]
	if !ok {
		return nil, fmt.Errorf("source files are missing %s", ManifestName)
	}
	manifest, err := format.Parse(ManifestName, manifestData)
	if err != nil {
		return nil, fmt.Errorf("parse package manifest: %w", err)
	}
	if manifest.Kind != "Package" {
		return nil, fmt.Errorf("%s must have kind Package", ManifestName)
	}
	if err := validateManifest(manifest); err != nil {
		return nil, err
	}
	selected := map[string][]byte{ManifestName: manifestData}
	var areaPaths []string
	for _, area := range manifest.Spec.Areas {
		areaPaths = append(areaPaths, area.Path)
	}
	candidates := make(map[string][]byte)
	resources := make([]*core.Resource, 0)
	parseErrors := make(map[string]error)
	for _, filePath := range sortedFilePaths(files) {
		data := files[filePath]
		if filePath == ManifestName {
			continue
		}
		if isYAML(filePath) && inAnyArea(filePath, areaPaths) {
			if resource, err := format.Parse(filePath, data); err == nil {
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
		if ordinary[filePath] && isValidTextInput(data) && !format.IsResourceEnvelope(data) {
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

func validateManifest(manifest *core.Resource) error {
	if manifest.Spec.Version == "" {
		return fmt.Errorf("package manifest requires spec.version")
	}
	if hasPackageRefs(manifest) {
		return fmt.Errorf("package manifest cannot contain transitive package references")
	}
	if len(manifest.Spec.Checks) != 0 || len(manifest.Spec.Targets) != 0 || len(manifest.Spec.RuleAdapters) != 0 || len(manifest.Spec.Packages) != 0 {
		return fmt.Errorf("package manifest must not declare checks, targets, ruleAdapters, or nested packages")
	}
	areas := map[string]bool{}
	for _, area := range manifest.Spec.Areas {
		areas[area.Name] = true
	}
	for _, area := range manifest.Spec.Areas {
		for _, imported := range area.Imports {
			if !areas[imported] {
				return fmt.Errorf("package area %q imports unknown package-local area %q", area.Name, imported)
			}
		}
	}
	for _, binding := range manifest.Spec.Bindings {
		if binding.Contract.Package != "" || binding.Implementation.Package != "" {
			return fmt.Errorf("package bindings cannot reference another package")
		}
	}
	for _, export := range manifest.Spec.Exports {
		if export.Package != "" {
			return fmt.Errorf("package exports cannot reference another package")
		}
	}
	return nil
}

func declaredInputs(resources []*core.Resource) map[string]bool {
	inputs := make(map[string]bool)
	for _, resource := range resources {
		for _, filePath := range resource.Spec.Files {
			inputs[filePath] = true
		}
	}
	return inputs
}

func hasPackageRefs(resource *core.Resource) bool {
	check := func(ref core.Ref) bool { return ref.Package != "" }
	for _, refs := range [][]core.Ref{resource.Spec.Rules, resource.Spec.Uses, resource.Spec.Needs, resource.Spec.Implements} {
		for _, ref := range refs {
			if check(ref) {
				return true
			}
		}
	}
	for _, area := range resource.Spec.Areas {
		for _, ref := range area.Rules {
			if check(ref) {
				return true
			}
		}
	}
	for _, binding := range resource.Spec.Bindings {
		if check(binding.Contract) || check(binding.Implementation) {
			return true
		}
	}
	for _, refs := range resource.Spec.RuleAdapters {
		for _, ref := range refs {
			if check(ref) {
				return true
			}
		}
	}
	return false
}

func inAnyArea(filePath string, areas []string) bool {
	for _, root := range areas {
		if filePath == root || strings.HasPrefix(filePath, root+"/") {
			return true
		}
	}
	return false
}

func isYAML(filePath string) bool {
	extension := strings.ToLower(path.Ext(filePath))
	return extension == ".yaml" || extension == ".yml"
}

func validatePath(filePath string) error {
	if filePath == "" || !utf8.ValidString(filePath) || strings.Contains(filePath, "\\") || strings.HasPrefix(filePath, "/") || path.Clean(filePath) != filePath {
		return fmt.Errorf("path must be a normalized relative POSIX path")
	}
	for _, segment := range strings.Split(filePath, "/") {
		if segment == "" || segment == "." || segment == ".." || strings.HasSuffix(segment, ".") || strings.HasSuffix(segment, " ") {
			return fmt.Errorf("path has an unsafe segment")
		}
		for _, ch := range segment {
			if ch < 0x20 || strings.ContainsRune(`<>:"|?*`, ch) {
				return fmt.Errorf("path contains a non-portable character")
			}
		}
		base := strings.ToUpper(strings.SplitN(segment, ".", 2)[0])
		if isWindowsDeviceName(base) {
			return fmt.Errorf("path contains a reserved Windows device name")
		}
	}
	return nil
}

func isWindowsDeviceName(base string) bool {
	if base == "CON" || base == "PRN" || base == "AUX" || base == "NUL" {
		return true
	}
	if len(base) == 4 && (strings.HasPrefix(base, "COM") || strings.HasPrefix(base, "LPT")) && base[3] >= '1' && base[3] <= '9' {
		return true
	}
	return false
}
