package app

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/release"
	"github.com/Glacius-Labs/Markitect/internal/source"
)

const releaseManifestPath = "tools/markitect/release.yaml"

var installPinPaths = []string{
	"markitect.lock.yaml",
	"scripts/markitect-bootstrap_test.go",
	"scripts/run-markitect.go",
	"tools/markitect/source.zip",
}

var installPaths = []string{
	"markitect.lock.yaml",
	"scripts/markitect-bootstrap_test.go",
	"scripts/run-markitect.go",
	"tools/markitect/release.yaml",
	"tools/markitect/source.zip",
}

// InstallPlan describes a complete release-pin change. Install performs only
// this preflight plan unless write is true. Per-file replacement is atomic;
// the five-file update is not a filesystem transaction.
type InstallPlan struct {
	Kind         string            `yaml:"kind"`
	Version      string            `yaml:"version"`
	SourceCommit string            `yaml:"sourceCommit"`
	BundleSHA256 string            `yaml:"bundleSha256"`
	Files        []InstallFilePlan `yaml:"files"`
	Written      []string          `yaml:"written,omitempty"`
	Applied      bool              `yaml:"applied"`
	Recovery     string            `yaml:"recovery,omitempty"`
}

// InstallFilePlan reports the expected digest and operation for one pin file.
type InstallFilePlan struct {
	Path   string `yaml:"path"`
	SHA256 string `yaml:"sha256"`
	Action string `yaml:"action"`
}

type installFileState struct {
	data      []byte // raw filesystem bytes
	canonical []byte // LF-normalized text bytes for pin/HEAD validation
	exists    bool
}

type installState struct {
	files  map[string]installFileState
	head   *source.Snapshot
	branch string
	kind   string
}

// Install validates a release bundle and returns a safe installation plan.
// With write=false it makes no filesystem changes. With write=true it requires
// a named non-protected branch, holds Markitect's shared writer lock, repeats
// the full preflight, then writes the complete five-file set.
func Install(root string, bundle *release.Bundle, write bool) (*InstallPlan, error) {
	if err := validateIncomingBundle(bundle); err != nil {
		return nil, err
	}
	rootAbs, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve install root: %w", err)
	}
	if info, statErr := os.Lstat(rootAbs); statErr != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		if statErr != nil {
			return nil, fmt.Errorf("install root must be an existing real directory: %w", statErr)
		}
		return nil, errors.New("install root must be an existing real directory")
	}

	var branch string
	var unlock func()
	if write {
		branch, err = installableBranch(rootAbs)
		if err != nil {
			return nil, err
		}
		unlock, err = lockWriter(rootAbs)
		if err != nil {
			return nil, err
		}
		defer unlock()
	}

	plan, state, err := buildInstallPlan(rootAbs, bundle)
	if err != nil {
		return nil, err
	}
	state.branch = branch
	if !write {
		return plan, nil
	}
	if state.head == nil {
		return plan, errors.New("install write requires a committed Git HEAD so the resulting pin set has a recovery baseline")
	}
	if current, branchErr := installableBranch(rootAbs); branchErr != nil || current != branch {
		if branchErr != nil {
			return plan, fmt.Errorf("branch changed or became protected during install preflight: %w", branchErr)
		}
		return plan, errors.New("branch changed during install preflight; rerun the plan")
	}
	if err := ensureInstallStateUnchanged(rootAbs, state); err != nil {
		return plan, err
	}
	if plan.Kind == "noop" {
		plan.Applied = true
		return plan, nil
	}

	for _, file := range plan.Files {
		if file.Action == "unchanged" {
			continue
		}
		if err := ensureInstallStateUnchanged(rootAbs, state); err != nil {
			return installWriteFailure(plan, state, err)
		}
		dest, err := safeInstallDestination(rootAbs, file.Path)
		if err != nil {
			return installWriteFailure(plan, state, err)
		}
		if err = os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return installWriteFailure(plan, state, fmt.Errorf("create parent directory for %s: %w", file.Path, err))
		}
		if _, err = safeInstallDestination(rootAbs, file.Path); err != nil {
			return installWriteFailure(plan, state, err)
		}
		current, err := readInstallFile(rootAbs, file.Path)
		if err != nil {
			return installWriteFailure(plan, state, err)
		}
		before := state.files[file.Path]
		if current.exists != before.exists || (current.exists && !bytes.Equal(current.data, before.data)) {
			return installWriteFailure(plan, state, fmt.Errorf("install target changed during write: %s", file.Path))
		}
		if err := ensureWriteBranch(rootAbs, state.branch); err != nil {
			return installWriteFailure(plan, state, err)
		}
		if err = atomicWrite(dest, bundle.Files[file.Path]); err != nil {
			return installWriteFailure(plan, state, fmt.Errorf("write %s: %w", file.Path, err))
		}
		plan.Written = append(plan.Written, file.Path)
		data := append([]byte(nil), bundle.Files[file.Path]...)
		state.files[file.Path] = installFileState{data: data, canonical: canonicalInstallReadback(file.Path, data), exists: true}
	}

	if err := verifyInstalledBundle(rootAbs, bundle); err != nil {
		return installWriteFailure(plan, state, fmt.Errorf("verify completed install: %w", err))
	}
	if err := ensureInstallStateUnchanged(rootAbs, state); err != nil {
		return installWriteFailure(plan, state, err)
	}
	plan.Applied = true
	return plan, nil
}

func validateIncomingBundle(bundle *release.Bundle) error {
	if bundle == nil {
		return errors.New("release bundle is nil; parse and verify the bundle before installation")
	}
	if !validInstallDigest(bundle.SHA256) {
		return errors.New("release bundle has no valid outer SHA-256; parse and verify the bundle before installation")
	}
	if len(bundle.Files) != len(installPaths) {
		return fmt.Errorf("release bundle must contain exactly %d pinned files including the manifest", len(installPaths))
	}
	for _, name := range installPaths {
		if _, ok := bundle.Files[name]; !ok {
			return fmt.Errorf("release bundle is missing pinned file %s", name)
		}
	}
	manifestBytes := bundle.Files[releaseManifestPath]
	manifest, err := release.ParseBundleManifest(manifestBytes)
	if err != nil {
		return fmt.Errorf("validate release manifest: %w", err)
	}
	if !reflect.DeepEqual(manifest, bundle.Manifest) {
		return errors.New("release bundle manifest bytes do not match the parsed manifest")
	}
	if err := release.ValidateBundleFiles(manifest, bundle.Files, manifest.SourceCommit); err != nil {
		return fmt.Errorf("validate release bundle files: %w", err)
	}
	return nil
}

func buildInstallPlan(root string, bundle *release.Bundle) (*InstallPlan, *installState, error) {
	state := &installState{files: make(map[string]installFileState, len(installPaths))}
	for _, name := range installPaths {
		file, err := readInstallFile(root, name)
		if err != nil {
			return nil, nil, err
		}
		state.files[name] = file
	}
	present := 0
	for _, name := range installPaths {
		if state.files[name].exists {
			present++
		}
	}

	var err error
	state.head, err = source.Load(root, "HEAD")
	if err != nil {
		// A fresh plan is useful before a Git repository has an initial commit.
		// Any existing pin set, and every write, still requires committed evidence.
		state.head = nil
	}
	if state.head != nil {
		if err := ensureNoStagedInstallPins(root, true); err != nil {
			return nil, nil, err
		}
	} else {
		// A plan outside Git is useful for a completely empty destination. In a
		// Git worktree without a first commit, staged target paths are still a
		// partial installation and must not be adopted.
		if inside, insideErr := source.GitOutput(root, "rev-parse", "--is-inside-work-tree"); insideErr == nil && strings.TrimSpace(string(inside)) == "true" {
			if err := ensureNoStagedInstallPins(root, false); err != nil {
				return nil, nil, err
			}
		}
	}

	switch {
	case present == 0:
		if state.head != nil {
			for _, name := range installPaths {
				if _, tracked := state.head.Files[name]; tracked {
					return nil, nil, fmt.Errorf("all install pins are missing from the working tree but %s remains in committed HEAD", name)
				}
			}
		}
		state.kind = "install"
	case state.files[releaseManifestPath].exists:
		if present != len(installPaths) {
			return nil, nil, errors.New("refusing partial release pin set: manifest and all four pinned files must be present")
		}
		if err := validateCommittedPins(root, state, installPaths); err != nil {
			return nil, nil, fmt.Errorf("existing release pins are not an unchanged committed installation: %w", err)
		}
		manifest, err := release.ParseBundleManifest(state.files[releaseManifestPath].canonical)
		if err != nil {
			return nil, nil, fmt.Errorf("read installed release manifest: %w", err)
		}
		oldFiles := pinFileMap(state.files, installPinPaths)
		if err := release.ValidateBundleFiles(manifest, oldFiles, manifest.SourceCommit); err != nil {
			return nil, nil, fmt.Errorf("validate installed release: %w", err)
		}
		state.kind = "upgrade"
	case present == len(installPinPaths):
		if state.head == nil {
			return nil, nil, errors.New("legacy pins require committed Git evidence before upgrade")
		}
		if _, manifestTracked := state.head.Files[releaseManifestPath]; manifestTracked {
			return nil, nil, errors.New("release manifest is deleted from the working tree; refusing to treat it as a legacy install")
		}
		if err := validateCommittedPins(root, state, installPinPaths); err != nil {
			return nil, nil, fmt.Errorf("legacy pins are not a complete unchanged committed installation: %w", err)
		}
		legacyFiles := pinFileMap(state.files, installPinPaths)
		lock, err := release.ParseToolLock(legacyFiles["markitect.lock.yaml"])
		if err != nil {
			return nil, nil, fmt.Errorf("read legacy tool lock: %w", err)
		}
		if lock.Version != "0.1.0-rc.3" {
			return nil, nil, fmt.Errorf("legacy upgrade only accepts the RC3 flat lock, found %q", lock.Version)
		}
		if err := release.ValidateToolLockFiles(lock, legacyFiles); err != nil {
			return nil, nil, fmt.Errorf("validate legacy tool lock: %w", err)
		}
		state.kind = "legacy-upgrade"
	default:
		return nil, nil, fmt.Errorf("refusing partial or unknown release pins: found %d of %d expected files", present, len(installPaths))
	}

	plan := &InstallPlan{
		Kind:         state.kind,
		Version:      bundle.Manifest.Version,
		SourceCommit: bundle.Manifest.SourceCommit,
		BundleSHA256: bundle.SHA256,
		Files:        make([]InstallFilePlan, 0, len(installPaths)),
	}
	for _, name := range orderedInstallPaths() {
		current := state.files[name]
		action := "create"
		if current.exists {
			if bytes.Equal(current.canonical, bundle.Files[name]) {
				action = "unchanged"
			} else {
				action = "replace"
			}
		}
		plan.Files = append(plan.Files, InstallFilePlan{Path: name, SHA256: digestInstallBytes(bundle.Files[name]), Action: action})
	}
	if state.kind != "install" && state.kind != "upgrade" && state.kind != "legacy-upgrade" {
		return nil, nil, fmt.Errorf("unsupported install state %q", state.kind)
	}
	if allInstallBytesMatch(state.files, bundle.Files) {
		plan.Kind = "noop"
	}
	return plan, state, nil
}

func validateCommittedPins(root string, state *installState, names []string) error {
	if state.head == nil {
		return errors.New("Git HEAD is unavailable")
	}
	for _, name := range names {
		committed, ok := state.head.Files[name]
		if !ok {
			return fmt.Errorf("%s is not tracked in committed HEAD", name)
		}
		mode := state.head.Modes[name]
		if mode != "100644" && mode != "100755" {
			return fmt.Errorf("%s is not a committed regular file", name)
		}
		current := state.files[name]
		if !current.exists || !bytes.Equal(current.canonical, canonicalInstallReadback(name, committed)) {
			return fmt.Errorf("%s differs from committed HEAD", name)
		}
	}
	return nil
}

func ensureNoStagedInstallPins(root string, hasHead bool) error {
	args := []string{"diff", "--cached", "--name-only", "HEAD", "--"}
	if !hasHead {
		args = []string{"ls-files", "--stage", "-z", "--"}
	}
	args = append(args, installPaths...)
	out, err := source.GitOutput(root, args...)
	if err != nil {
		return fmt.Errorf("inspect staged release pins: %w", err)
	}
	if strings.TrimSpace(string(out)) != "" {
		return fmt.Errorf("release pin paths have staged changes; commit or unstage them before install: %s", strings.TrimSpace(string(out)))
	}
	return nil
}

func ensureInstallStateUnchanged(root string, state *installState) error {
	branch, err := installableBranch(root)
	if err != nil && state.branch != "" {
		return fmt.Errorf("install branch is no longer writable: %w", err)
	}
	if state.branch != "" && branch != state.branch {
		return errors.New("branch changed during install")
	}
	if err := ensureHeadUnchanged(root, state.head); err != nil {
		return err
	}
	if state.head != nil {
		if err := ensureNoStagedInstallPins(root, true); err != nil {
			return fmt.Errorf("staged release pins changed during install: %w", err)
		}
	}
	for _, name := range installPaths {
		current, err := readInstallFile(root, name)
		if err != nil {
			return err
		}
		before := state.files[name]
		if current.exists != before.exists || (current.exists && !bytes.Equal(current.data, before.data)) {
			return fmt.Errorf("release pin changed during install: %s", name)
		}
	}
	return nil
}

func ensureHeadUnchanged(root string, before *source.Snapshot) error {
	if before == nil {
		out, err := source.GitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return errors.New("Git HEAD appeared during a fresh install; rerun the plan")
		}
		return nil
	}
	after, err := source.GitOutput(root, "rev-parse", "--verify", "--end-of-options", "HEAD^{commit}")
	if err != nil {
		return fmt.Errorf("read Git HEAD during install: %w", err)
	}
	if strings.TrimSpace(string(after)) != before.Revision {
		return errors.New("Git HEAD changed during install; rerun the plan")
	}
	return nil
}

func installWriteFailure(plan *InstallPlan, state *installState, cause error) (*InstallPlan, error) {
	if len(plan.Written) > 0 {
		if state.kind == "install" {
			plan.Recovery = "The five-file installation is partial; do not use it. Remove the listed newly written files after reviewing them, or complete the exact bundle and rerun verification."
		} else {
			plan.Recovery = "The five-file installation is partial; do not use it. Restore the complete pin set from the prior committed candidate or revert the integration commit through the normal Git workflow, then verify."
		}
		return plan, fmt.Errorf("install stopped after writing %s; no complete installation is claimed: %w", strings.Join(plan.Written, ", "), cause)
	}
	return plan, cause
}

func verifyInstalledBundle(root string, bundle *release.Bundle) error {
	files := make(map[string][]byte, len(installPaths))
	for _, name := range installPaths {
		file, err := readInstallFile(root, name)
		if err != nil {
			return err
		}
		if !file.exists {
			return fmt.Errorf("installed file is missing: %s", name)
		}
		if !bytes.Equal(file.canonical, bundle.Files[name]) {
			return fmt.Errorf("installed file differs from the selected bundle: %s", name)
		}
		files[name] = file.canonical
	}
	manifest, err := release.ParseBundleManifest(files[releaseManifestPath])
	if err != nil {
		return err
	}
	return release.ValidateBundleFiles(manifest, files, bundle.Manifest.SourceCommit)
}

func readInstallFile(root, name string) (installFileState, error) {
	dest, err := safeInstallDestination(root, name)
	if err != nil {
		return installFileState{}, err
	}
	info, err := os.Lstat(dest)
	if os.IsNotExist(err) {
		return installFileState{}, nil
	}
	if err != nil {
		return installFileState{}, fmt.Errorf("inspect install target %s: %w", name, err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return installFileState{}, fmt.Errorf("install target %s is not a regular file", name)
	}
	data, err := os.ReadFile(dest)
	if err != nil {
		return installFileState{}, fmt.Errorf("read install target %s: %w", name, err)
	}
	return installFileState{data: data, canonical: canonicalInstallReadback(name, data), exists: true}, nil
}

// canonicalInstallReadback normalizes the four text pins (flat lock, bootstrap
// sources, and manifest) from CRLF checkout bytes to canonical LF bytes. The
// source archive remains byte-exact; installFileState.data retains raw bytes
// independently for concurrent-edit checks.
func canonicalInstallReadback(name string, data []byte) []byte {
	if name == "tools/markitect/source.zip" {
		return append([]byte(nil), data...)
	}
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}

func safeInstallDestination(root, name string) (string, error) {
	dest, err := safeDestination(root, name)
	if err != nil {
		return "", err
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	current := absolute
	parts := strings.Split(name, "/")
	for i, part := range parts {
		current = filepath.Join(current, part)
		info, statErr := os.Lstat(current)
		if os.IsNotExist(statErr) {
			continue
		}
		if statErr != nil {
			return "", fmt.Errorf("inspect install path %s: %w", name, statErr)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			return "", fmt.Errorf("symlink in install path %s", name)
		}
		if i < len(parts)-1 && !info.IsDir() {
			return "", fmt.Errorf("unmanaged file blocks install path %s at %s", name, part)
		}
	}
	return dest, nil
}

func orderedInstallPaths() []string {
	paths := append([]string(nil), installPinPaths...)
	sort.Strings(paths)
	return append(paths, releaseManifestPath)
}

func pinFileMap(states map[string]installFileState, names []string) map[string][]byte {
	files := make(map[string][]byte, len(names))
	for _, name := range names {
		files[name] = states[name].canonical
	}
	return files
}

func allInstallBytesMatch(current map[string]installFileState, target map[string][]byte) bool {
	for _, name := range installPaths {
		state := current[name]
		if !state.exists || !bytes.Equal(state.canonical, target[name]) {
			return false
		}
	}
	return true
}

func installableBranch(root string) (string, error) {
	branch, err := writeBranchName(root)
	if err != nil {
		return "", fmt.Errorf("install requires a Git worktree on a named non-protected branch: %w", err)
	}
	return branch, nil
}

func validInstallDigest(value string) bool {
	if len(value) != sha256.Size*2 || strings.ToLower(value) != value {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func digestInstallBytes(data []byte) string {
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
