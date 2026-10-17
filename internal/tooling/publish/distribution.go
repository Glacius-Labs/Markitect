package publish

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

const wingetManifestSchema = "1.12.0"

var stableVersionTag = regexp.MustCompile(`^v(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$`)

const (
	wingetID     = "GlaciusLabs.Markitect"
	installStart = "<!-- markitect-release:install:start -->"
	installEnd   = "<!-- markitect-release:install:end -->"
	tryStart     = "<!-- markitect-release:try:start -->"
	tryEnd       = "<!-- markitect-release:try:end -->"
)

type PublishedDistribution struct {
	Tag     string
	Version string
	Commit  string
	Assets  namedBytes
	Digests map[string]string
}

type DistributionOptions struct {
	Tag            string
	RepositoryRoot string
	Write          bool
	ExportWinget   string
}

type DistributionResult struct {
	Status  string   `yaml:"status" json:"status"`
	Tag     string   `yaml:"tag" json:"tag"`
	Version string   `yaml:"version" json:"version"`
	Files   []string `yaml:"files" json:"files"`
}

// LoadPublishedDistribution validates immutable release metadata, provenance,
// source tag, exact asset bytes and GitHub attestations before returning data.
func LoadPublishedDistribution(ctx context.Context, runner Runner, tag string) (*PublishedDistribution, error) {
	if runner == nil {
		return nil, errors.New("GitHub command runner is required")
	}
	if stableVersionTag.FindStringSubmatch(tag) == nil {
		return nil, fmt.Errorf("tag %q is not a stable vMAJOR.MINOR.PATCH release", tag)
	}
	version := strings.TrimPrefix(tag, "v")
	release, err := getRelease(runner, ctx, tag)
	if err != nil {
		return nil, fmt.Errorf("read published release: %w", err)
	}
	if release.ID <= 0 || release.TagName != tag || release.Draft || !release.Immutable || release.PublishedAt == "" {
		return nil, errors.New("selected release is not the published immutable release for the requested tag")
	}
	if len(release.Assets) != len(assetNames(tag)) {
		return nil, errors.New("published release does not contain exactly the four expected assets")
	}
	if err := checkImmutable(runner, ctx); err != nil {
		return nil, fmt.Errorf("verify immutable-release setting: %w", err)
	}
	if _, err := verifyAttestation(runner, ctx, "release", "verify", tag, "--repo", "github.com/"+repository); err != nil {
		return nil, fmt.Errorf("release attestation verification failed: %w", err)
	}
	dir, err := os.MkdirTemp("", "markitect-distribution-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(dir)
	if err := call(runner, ctx, "", "release", "download", tag, "--repo", "github.com/"+repository, "--dir", dir); err != nil {
		return nil, fmt.Errorf("download published release assets: %w", err)
	}
	assets, err := readAssets(dir, tag)
	if err != nil {
		return nil, fmt.Errorf("published release assets failed validation: %w", err)
	}
	if err := compareRemoteAssets(release.Assets, assets); err != nil {
		return nil, fmt.Errorf("published release asset digests do not match downloaded bytes: %w", err)
	}
	provenance, err := parseProvenance(assets[assetNames(tag)[3]])
	if err != nil {
		return nil, err
	}
	if provenance.Tag != tag {
		return nil, errors.New("release provenance tag does not match the requested tag")
	}
	if err := validateTagTarget(runner, ctx, tag, provenance.SourceCommit); err != nil {
		return nil, err
	}
	runID, err := strconv.ParseInt(provenance.WorkflowRunID, 10, 64)
	if err != nil || runID <= 0 {
		return nil, errors.New("release provenance contains an invalid workflow run ID")
	}
	attempt, err := strconv.Atoi(provenance.WorkflowRunAttempt)
	if err != nil || attempt <= 0 {
		return nil, errors.New("release provenance contains an invalid workflow run attempt")
	}
	var run runInfo
	attemptEndpoint := fmt.Sprintf("repos/%s/actions/runs/%d/attempts/%d", repository, runID, attempt)
	if err := apiJSON(runner, ctx, attemptEndpoint, &run); err != nil {
		return nil, fmt.Errorf("read provenance workflow run: %w", err)
	}
	wantURL := fmt.Sprintf("https://github.com/%s/actions/runs/%d/attempts/%d", repository, runID, attempt)
	if run.ID != runID || run.Status != "completed" || run.Conclusion != "success" || run.Event != "push" || strings.SplitN(run.Path, "@", 2)[0] != workflowPath || run.HeadBranch != tag || run.HeadSHA != provenance.SourceCommit || run.RunAttempt != attempt || provenance.WorkflowRun != wantURL {
		return nil, errors.New("release provenance does not match a successful run of the tagged release workflow")
	}
	for _, name := range assetNames(tag) {
		if _, err := verifyAttestation(runner, ctx, "release", "verify-asset", tag, filepath.Join(dir, name), "--repo", "github.com/"+repository); err != nil {
			return nil, fmt.Errorf("asset attestation verification failed for %s: %w", name, err)
		}
	}
	digests := make(map[string]string, len(assets))
	for name, data := range assets {
		digests[name] = digest(data)
	}
	return &PublishedDistribution{Tag: tag, Version: version, Commit: provenance.SourceCommit, Assets: assets, Digests: digests}, nil
}

// SyncDistribution checks, writes, or exports release-derived installation
// metadata. Export creates the normal winget-pkgs manifests/g/... layout.
func SyncDistribution(ctx context.Context, runner Runner, options DistributionOptions) (*DistributionResult, error) {
	release, err := LoadPublishedDistribution(ctx, runner, options.Tag)
	if err != nil {
		return nil, err
	}
	files, err := renderDistribution(release)
	if err != nil {
		return nil, err
	}
	result := &DistributionResult{Status: "checked", Tag: release.Tag, Version: release.Version}
	if options.ExportWinget != "" {
		if options.Write {
			return nil, errors.New("--write and --export-winget are mutually exclusive")
		}
		relatives := sortedKeys(files)
		type pendingExport struct {
			relative string
			data     []byte
		}
		exports := make([]pendingExport, 0, 3)
		relativeTargets := make([]string, 0, 3)
		for _, relative := range relatives {
			if strings.HasPrefix(filepath.ToSlash(relative), "packaging/winget/GlaciusLabs.Markitect/") {
				sourcePath := filepath.ToSlash(relative)
				data, ok := files[sourcePath]
				if !ok || len(data) == 0 {
					return nil, fmt.Errorf("generated WinGet manifest %s is missing or empty", filepath.Base(relative))
				}
				exports = append(exports, pendingExport{relative: filepath.Join("manifests", "g", "GlaciusLabs", "Markitect", release.Version, filepath.Base(relative)), data: data})
			}
		}
		for _, file := range exports {
			relativeTargets = append(relativeTargets, file.relative)
		}
		targets, err := preflightAtomicTargets(options.ExportWinget, relativeTargets)
		if err != nil {
			return nil, fmt.Errorf("preflight WinGet export: %w", err)
		}
		for i, file := range exports {
			if err := writeAtomic(targets[i], file.data); err != nil {
				return nil, fmt.Errorf("export WinGet manifest %s: %w", filepath.Base(file.relative), err)
			}
			result.Files = append(result.Files, targets[i])
		}
		result.Status = "exported"
		return result, nil
	}
	if options.RepositoryRoot == "" {
		return nil, errors.New("repository root is required for check or write")
	}
	paths := sortedKeys(files)
	type pendingFile struct {
		relative string
		data     []byte
	}
	pending := make([]pendingFile, 0, len(paths))
	for _, relative := range paths {
		expected := files[relative]
		target := filepath.Join(options.RepositoryRoot, filepath.FromSlash(relative))
		if relative == "README.md" {
			current, err := os.ReadFile(target)
			if err != nil {
				return nil, fmt.Errorf("read %s: %w", target, err)
			}
			expected, err = replaceMarkedSections(current, expected)
			if err != nil {
				return nil, fmt.Errorf("update README metadata: %w", err)
			}
		}
		if !options.Write {
			current, err := os.ReadFile(target)
			if err != nil || string(current) != string(expected) {
				return nil, fmt.Errorf("distribution file %s is stale; rerun with --write after reviewing the verified release data", target)
			}
		}
		pending = append(pending, pendingFile{relative: relative, data: expected})
		result.Files = append(result.Files, target)
	}
	if options.Write {
		relatives := make([]string, 0, len(pending))
		for _, file := range pending {
			relatives = append(relatives, file.relative)
		}
		targets, err := preflightAtomicTargets(options.RepositoryRoot, relatives)
		if err != nil {
			return nil, fmt.Errorf("preflight distribution write: %w", err)
		}
		for i, file := range pending {
			if err := writeAtomic(targets[i], file.data); err != nil {
				return nil, fmt.Errorf("write %s: %w", targets[i], err)
			}
		}
		result.Status = "written"
	}
	return result, nil
}

func preflightAtomicTargets(root string, relatives []string) ([]string, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	if info, err := os.Lstat(absoluteRoot); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			return nil, fmt.Errorf("root %s is not a real directory", absoluteRoot)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("inspect root %s: %w", absoluteRoot, err)
	}
	targets := make([]string, 0, len(relatives))
	for _, relative := range relatives {
		clean := filepath.Clean(relative)
		if !filepath.IsLocal(clean) {
			return nil, fmt.Errorf("output path %q escapes its root", relative)
		}
		parts := strings.Split(clean, string(filepath.Separator))
		current := absoluteRoot
		for i, part := range parts {
			current = filepath.Join(current, part)
			info, err := os.Lstat(current)
			if os.IsNotExist(err) {
				continue
			}
			if err != nil {
				return nil, fmt.Errorf("inspect %s: %w", current, err)
			}
			if i == len(parts)-1 {
				if !info.Mode().IsRegular() {
					return nil, fmt.Errorf("target %s exists and is not a regular file", current)
				}
			} else if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
				return nil, fmt.Errorf("parent %s is not a real directory", current)
			}
		}
		targets = append(targets, filepath.Join(absoluteRoot, clean))
	}
	return targets, nil
}

func renderDistribution(r *PublishedDistribution) (map[string][]byte, error) {
	if r == nil || stableVersionTag.FindStringSubmatch(r.Tag) == nil || r.Version != strings.TrimPrefix(r.Tag, "v") {
		return nil, errors.New("invalid verified release data")
	}
	winName := "markitect-" + r.Tag + "-windows-amd64.exe"
	linuxName := "markitect-" + r.Tag + "-linux-amd64"
	winHash, linuxHash := r.Digests[winName], r.Digests[linuxName]
	if !digestPattern.MatchString(winHash) || !digestPattern.MatchString(linuxHash) {
		return nil, errors.New("verified release data is missing a valid Windows or Linux digest")
	}
	readme, err := renderReadme(r.Version, r.Tag, winHash, linuxHash)
	if err != nil {
		return nil, err
	}
	base := "packaging/winget/GlaciusLabs.Markitect/" + r.Version + "/"
	url := "https://github.com/" + repository + "/releases/download/" + r.Tag + "/" + winName
	return map[string][]byte{
		"README.md":                                      []byte(readme),
		base + "GlaciusLabs.Markitect.yaml":              []byte(fmt.Sprintf("# yaml-language-server: $schema=https://aka.ms/winget-manifest.version.%s.schema.json\nPackageIdentifier: %s\nPackageVersion: %s\nDefaultLocale: en-US\nManifestType: version\nManifestVersion: %s\n", wingetManifestSchema, wingetID, r.Version, wingetManifestSchema)),
		base + "GlaciusLabs.Markitect.locale.en-US.yaml": []byte(fmt.Sprintf("# yaml-language-server: $schema=https://aka.ms/winget-manifest.defaultLocale.%s.schema.json\nPackageIdentifier: %s\nPackageVersion: %s\nPackageLocale: en-US\nPublisher: Glacius Labs\nPublisherUrl: https://github.com/Glacius-Labs\nPublisherSupportUrl: https://github.com/Glacius-Labs/Markitect/issues\nPackageName: Markitect\nPackageUrl: https://github.com/Glacius-Labs/Markitect\nLicense: Apache-2.0\nLicenseUrl: https://github.com/Glacius-Labs/Markitect/blob/%s/LICENSE\nShortDescription: Validate and compile AI-facing engineering resources.\nDescription: Markitect validates typed resources and explicit dependencies, compiles selected context, and measures change impact across Git revisions.\nTags:\n- cli\n- documentation\n- validation\nReleaseNotesUrl: https://github.com/Glacius-Labs/Markitect/releases/tag/%s\nManifestType: defaultLocale\nManifestVersion: %s\n", wingetManifestSchema, wingetID, r.Version, r.Tag, r.Tag, wingetManifestSchema)),
		base + "GlaciusLabs.Markitect.installer.yaml":    []byte(fmt.Sprintf("# yaml-language-server: $schema=https://aka.ms/winget-manifest.installer.%s.schema.json\nPackageIdentifier: %s\nPackageVersion: %s\nInstallerType: portable\nCommands:\n- markitect\nInstallers:\n- Architecture: x64\n  InstallerUrl: %s\n  InstallerSha256: %s\nManifestType: installer\nManifestVersion: %s\n", wingetManifestSchema, wingetID, r.Version, url, strings.ToUpper(winHash), wingetManifestSchema)),
	}, nil
}

func sortedKeys(files map[string][]byte) []string {
	keys := make([]string, 0, len(files))
	for key := range files {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func replaceMarkedSections(original, generated []byte) ([]byte, error) {
	text, generatedText := string(original), string(generated)
	for _, marker := range [][2]string{{installStart, installEnd}, {tryStart, tryEnd}} {
		if strings.Count(text, marker[0]) != 1 || strings.Count(text, marker[1]) != 1 || strings.Count(generatedText, marker[0]) != 1 || strings.Count(generatedText, marker[1]) != 1 {
			return nil, fmt.Errorf("README must contain each generated marker exactly once: %s and %s", marker[0], marker[1])
		}
		start, end := strings.Index(text, marker[0]), strings.Index(text, marker[1])+len(marker[1])
		newStart, newEnd := strings.Index(generatedText, marker[0]), strings.Index(generatedText, marker[1])+len(marker[1])
		if start >= end {
			return nil, fmt.Errorf("README generated markers are out of order: %s", marker[0])
		}
		text = text[:start] + generatedText[newStart:newEnd] + text[end:]
	}
	return []byte(text), nil
}

func writeAtomic(path string, data []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".markitect-distribution-*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	defer os.Remove(name)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(name, path)
}
