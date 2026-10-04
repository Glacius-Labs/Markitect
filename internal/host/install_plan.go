package host

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"reflect"
	"sort"
	"strings"

	"github.com/Glacius-Labs/Markitect/internal/infrastructure/source"
	"github.com/Glacius-Labs/Markitect/internal/tooling/release"
)

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
		oldFiles := pinFileMap(state.files, installPaths)
		if err := release.ValidateBundleFiles(manifest, oldFiles, manifest.SourceCommit); err != nil {
			return nil, nil, fmt.Errorf("validate installed release: %w", err)
		}
		state.kind = "upgrade"
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
	if state.kind != "install" && state.kind != "upgrade" {
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
	if name == ".markitect/tool/source.zip" {
		return append([]byte(nil), data...)
	}
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
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
